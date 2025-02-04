package config

import (
	"database/sql"
	"log"
	"strings"

	"crypto/ecdsa"
	"encoding/hex"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/caarlos0/env/v11"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ConfigStruct struct {
	DatabaseURL       string `env:"DATABASE_URL"`
	DatabaseUsername  string `env:"DB_USERNAME"`
	DatabasePassword  string `env:"DB_PASSWORD"`
	EthereumRPC       string `env:"ETHEREUM_RPC"`
	JWTSecretKey      string `env:"JWT_SECRET"`
	JWTResetSecretKey string `env:"JWT_RESET_SECRET"`
	SuperUserEmail    string `env:"SUPER_USER_EMAIL"`
	SuperUserPassword string `env:"SUPER_USER_PASSWORD"`
}

var ConfigDetails ConfigStruct

// Creates a Superuser along with Server Initialization
// func CreateSuperUser() {
// 	//Checking if the superuser already exists
// 	user, _ := repo.GetUserByEmail(ConfigDetails.SuperUserEmail)

// 	if user.Username != "" {
// 		log.Println("The Superuser Already exists Therefore No Need To Initialize a new Superuser")
// 		return
// 	}

// 	// Create an Ethereum wallet
// 	walletAddress, privateKey, err := ethereum.CreateWallet(ConfigDetails.SuperUserPassword)
// 	if err != nil {
// 		log.Println("Error creating Ethereum wallet")
// 		return
// 	}

// 	//Convert private key to hex format
// 	privateKeyHex := PrivateKeyToHex(privateKey)

// 	// Preload tokens to the wallet
// 	testnetAmount := big.NewInt(5e18) // 1 ETH in wei
// 	if err := ethereum.PreloadTokens(walletAddress, testnetAmount); err != nil {
// 		log.Println("Error preloading tokens to wallet")
// 		return
// 	}

// 	// Hash the password
// 	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(ConfigDetails.SuperUserPassword), bcrypt.DefaultCost)
// 	repo.CreateUser("SuperUser", ConfigDetails.SuperUserEmail, string(hashedPassword), "SuperUser", "01/01/2001", walletAddress, 3)

// 	savedUser, err := repo.GetUserByEmail(ConfigDetails.SuperUserEmail)
// 	if err != nil {
// 		log.Println("Error Retriving User ID in SuperUser Config : ", err.Error())
// 	}

// 	log.Println("privateKeyHex", privateKeyHex)

// 	repo.InsertPrivateKey(savedUser.ID, walletAddress, privateKeyHex)
// }

type Dependencies struct {
	PostgresDB *sql.DB
	EthClient  *ethclient.Client
}

// Inintialize all Configurations for the Server
func InitConfig() (*sql.DB, *ethclient.Client) {

	//Parse & Load Environment Variables
	errenv := env.Parse(&ConfigDetails)
	if errenv != nil {
		log.Fatal("Error Parsing the Environment Variables", errenv)
		return nil, nil
	}

	if len(ConfigDetails.DatabaseURL) == 0 || len(ConfigDetails.DatabasePassword) == 0 || len(ConfigDetails.DatabaseUsername) == 0 || len(ConfigDetails.EthereumRPC) == 0 || len(ConfigDetails.JWTSecretKey) == 0 || len(ConfigDetails.JWTResetSecretKey) == 0 || len(ConfigDetails.SuperUserEmail) == 0 || len(ConfigDetails.SuperUserPassword) == 0 {
		log.Fatal("Missing Environment variable or file")
	}

	log.Println("Environment Variables Loaded Successfully")

	//Start DB Connection
	ConfigDetails.DatabaseURL = strings.Replace(ConfigDetails.DatabaseURL, "user", ConfigDetails.DatabaseUsername, 1)
	ConfigDetails.DatabaseURL = strings.Replace(ConfigDetails.DatabaseURL, "password", ConfigDetails.DatabasePassword, 1)

	postgresDB, err := repo.InitDB(ConfigDetails.DatabaseURL)

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	//Initialize Ethereum Client
	ethClient, err := ethereum.InitEthereumClient(ConfigDetails.EthereumRPC)
	if err != nil {
		log.Fatalf("Error Connecting to Ethereum RPC Sever : %v", err.Error())
	}

	//Creating Superuser
	// CreateSuperUser()
	return postgresDB, ethClient
}

func ReleaseConfig(db *sql.DB) {
	repo.CloseDB(db)
}

func PrivateKeyToHex(privateKey *ecdsa.PrivateKey) string {
	privateKeyBytes := crypto.FromECDSA(privateKey) // Convert to byte slice
	return hex.EncodeToString(privateKeyBytes)      // Convert to hex string
}
