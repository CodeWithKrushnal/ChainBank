package config

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"crypto/ecdsa"
	"encoding/hex"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/CodeWithKrushnal/ChainBank/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
)

type Dependencies struct {
	PostgresDB *sql.DB
	EthClient  *ethclient.Client
}

// Inintialize all Configurations for the Server
func InitConfig(ctx context.Context) (utils.ConfigStruct, *sql.DB, *ethclient.Client, error) {
	// Load configuration from environment variables or file
	configDetails, err := LoadConfig("")
	if err != nil {
		return utils.ConfigStruct{}, nil, nil, fmt.Errorf("%w: %v", utils.ErrConfigInit, err)
	}

	// Check for missing required configuration values
	if len(configDetails.DatabaseURL) == 0 || len(configDetails.DatabasePassword) == 0 ||
		len(configDetails.DatabaseUsername) == 0 || len(configDetails.EthereumRPC) == 0 ||
		len(configDetails.JWTSecretKey) == 0 || len(configDetails.JWTResetSecretKey) == 0 ||
		len(configDetails.SuperUserEmail) == 0 || len(configDetails.SuperUserPassword) == 0 ||
		len(configDetails.SMTPHost) == 0 || len(configDetails.SMTPPort) == 0 ||
		len(configDetails.SenderEmail) == 0 || len(configDetails.SenderPassword) == 0 || len(configDetails.EtherscanAPI) == 0 || len(configDetails.EtherscanAPIKey) == 0 {
		return utils.ConfigStruct{}, nil, nil, fmt.Errorf("%w: missing environment variable or file", utils.ErrConfigInit)
	}

	// Start DB Connection
	configDetails.DatabaseURL = strings.Replace(configDetails.DatabaseURL, "user", configDetails.DatabaseUsername, 1)
	configDetails.DatabaseURL = strings.Replace(configDetails.DatabaseURL, "password", configDetails.DatabasePassword, 1)

	postgresDB, err := repo.InitDB(configDetails.DatabaseURL)
	if err != nil {
		return utils.ConfigStruct{}, nil, nil, fmt.Errorf("%w: failed to connect to database", utils.ErrConfigInit)
	}

	// Initialize Ethereum Client
	ethClient, err := ethereum.InitEthereumClient(configDetails.EthereumRPC)
	if err != nil {
		return utils.ConfigStruct{}, nil, nil, fmt.Errorf("%w: error connecting to Ethereum RPC server", utils.ErrConfigInit)
	}

	return configDetails, postgresDB, ethClient, nil
}

func ReleaseConfig(ctx context.Context, db *sql.DB) error {
	return repo.CloseDB(db)
}

func PrivateKeyToHex(privateKey *ecdsa.PrivateKey) string {
	privateKeyBytes := crypto.FromECDSA(privateKey) // Convert to byte slice
	return hex.EncodeToString(privateKeyBytes)      // Convert to hex string
}

func LoadConfig(path string) (config utils.ConfigStruct, err error) {
	viper.AddConfigPath("./")
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
