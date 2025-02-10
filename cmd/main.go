package main

import (
	"context"
	"log"
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/internal/app"
	"github.com/CodeWithKrushnal/ChainBank/internal/config"
)

func main() {
	// Config Setup
	ctx := context.Background()
	postgresDB, ethClient := config.InitConfig(ctx)

	defer config.ReleaseConfig(ctx, postgresDB)

	deps := app.NewDependencies(ctx, postgresDB, ethClient)

	router := app.SetupRoutes(ctx, deps)
	log.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// Creates a Superuser along with Server Initialization
// func CreateSuperUser() {
// 	//Checking if the superuser already exists
// 	user, _ := repo.GetUserByEmail(config.ConfigDetails.SuperUserEmail)

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
