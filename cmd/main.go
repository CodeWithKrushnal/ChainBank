package main

import (
	"context"
	"log"
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/internal/app"
	"github.com/CodeWithKrushnal/ChainBank/internal/config"
	"github.com/CodeWithKrushnal/ChainBank/utils"
	"golang.org/x/exp/slog"
)

// main initializes the application and starts the HTTP server.
func main() {
	// Config Setup
	ctx := context.Background()
	postgresDB, ethClient, err := config.InitConfig(ctx)
	if err != nil {
		slog.Error(utils.ErrServiceInit.Error(), "error", err)
		return
	}
	defer func() {
		if err := config.ReleaseConfig(ctx, postgresDB); err != nil {
			slog.Error("Error releasing configuration", "error", err)
		}
	}()

	deps, err := app.NewDependencies(ctx, postgresDB, ethClient)
	if err != nil {
		slog.Error(utils.ErrServiceInit.Error(), "error", err)
		return
	}

	router := app.SetupRoutes(ctx, deps)
	slog.Info(utils.ServerStartLog)
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

// Add Necessary comments,  logs - use const strings use slog use standerd error and message strings and define them, in case of errors propogate error by returning do not log errors. Remove unnecessary, redundent logs


// Add Necessary comments,  logs - use const strings use slog use standerd error and message strings and define them, in case of errors log the received errors from the called functions. Remove unnecessary, redundent logs


