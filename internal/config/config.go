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

type ConfigStruct struct {
	DatabaseURL       string `mapstructure:"DATABASE_URL"`
	DatabaseUsername  string `mapstructure:"DB_USERNAME"`
	DatabasePassword  string `mapstructure:"DB_PASSWORD"`
	EthereumRPC       string `mapstructure:"ETHEREUM_RPC"`
	JWTSecretKey      string `mapstructure:"JWT_SECRET"`
	JWTResetSecretKey string `mapstructure:"JWT_RESET_SECRET"`
	SuperUserEmail    string `mapstructure:"SUPER_USER_EMAIL"`
	SuperUserPassword string `mapstructure:"SUPER_USER_PASSWORD"`
	SendGridAPIKey    string `mapstructure:"SENDGRID_API_KEY"`
}

var ConfigDetails ConfigStruct

type Dependencies struct {
	PostgresDB *sql.DB
	EthClient  *ethclient.Client
}

// Inintialize all Configurations for the Server
func InitConfig(ctx context.Context) (*sql.DB, *ethclient.Client, error) {
	// Load configuration from environment variables or file
	ConfigDetails, err := LoadConfig("")
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", utils.ErrConfigInit, err)
	}

	// Check for missing required configuration values
	if len(ConfigDetails.DatabaseURL) == 0 || len(ConfigDetails.DatabasePassword) == 0 ||
		len(ConfigDetails.DatabaseUsername) == 0 || len(ConfigDetails.EthereumRPC) == 0 ||
		len(ConfigDetails.JWTSecretKey) == 0 || len(ConfigDetails.JWTResetSecretKey) == 0 ||
		len(ConfigDetails.SuperUserEmail) == 0 || len(ConfigDetails.SuperUserPassword) == 0 {
		return nil, nil, fmt.Errorf("%w: missing environment variable or file", utils.ErrConfigInit)
	}

	// Start DB Connection
	ConfigDetails.DatabaseURL = strings.Replace(ConfigDetails.DatabaseURL, "user", ConfigDetails.DatabaseUsername, 1)
	ConfigDetails.DatabaseURL = strings.Replace(ConfigDetails.DatabaseURL, "password", ConfigDetails.DatabasePassword, 1)

	postgresDB, err := repo.InitDB(ConfigDetails.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed to connect to database", utils.ErrConfigInit)
	}

	// Initialize Ethereum Client
	ethClient, err := ethereum.InitEthereumClient(ConfigDetails.EthereumRPC)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: error connecting to Ethereum RPC server", utils.ErrConfigInit)
	}

	return postgresDB, ethClient, nil
}

func ReleaseConfig(ctx context.Context, db *sql.DB) error {
	return repo.CloseDB(db)
}

func PrivateKeyToHex(privateKey *ecdsa.PrivateKey) string {
	privateKeyBytes := crypto.FromECDSA(privateKey) // Convert to byte slice
	return hex.EncodeToString(privateKeyBytes)      // Convert to hex string
}

func LoadConfig(path string) (config ConfigStruct, err error) {
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
