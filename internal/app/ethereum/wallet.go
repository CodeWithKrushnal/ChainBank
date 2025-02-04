package ethereum

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ethRepo struct {
	ethereumClient *ethclient.Client
}

// Constructor function
func NewEthRepo(ethereumClient *ethclient.Client) EthRepo {
	return &ethRepo{ethereumClient: ethereumClient}
}

type EthRepo interface {
	CreateWallet(password string) (string, *ecdsa.PrivateKey, error)
	TransferFunds(fromPrivateKeyHex string, fromAddressHex string, toAddressHex string, amount *big.Int, gasPrice *big.Int, gasLimit uint64, chainID *big.Int) (*types.Transaction, error)
	PreloadTokens(walletAddress string, amount *big.Int) error
}

// CreateWallet generates a new Ethereum wallet
func (ethdep ethRepo) CreateWallet(password string) (string, *ecdsa.PrivateKey, error) {
	log.Println("Starting wallet creation process...")

	// Step 1: Initialize the keystore
	log.Println("Initializing the keystore...")
	ks := keystore.NewKeyStore("./wallets", keystore.StandardScryptN, keystore.StandardScryptP)
	if ks == nil {
		log.Println("Failed to initialize the keystore.")
		return "", nil, fmt.Errorf("keystore initialization failed")
	}
	log.Println("Keystore initialized successfully.")

	// Step 2: Create a new account
	log.Println("Creating a new Ethereum account...")
	account, err := ks.NewAccount(password)
	if err != nil {
		log.Printf("Error creating a new account: %v", err)
		return "", nil, err
	}
	log.Printf("New account created successfully. Address: %s", account.Address.Hex())

	// Step 3: Extract the private key from the keystore file
	log.Println("Extracting the private key from the account...")
	keyJSON, err := os.ReadFile(account.URL.Path) // Read the keystore file
	if err != nil {
		log.Printf("Error reading keystore file: %v", err)
		return "", nil, err
	}
	key, err := keystore.DecryptKey(keyJSON, password) // Decrypt the keystore file
	if err != nil {
		log.Printf("Error decrypting keystore file: %v", err)
		return "", nil, err
	}
	privateKey := key.PrivateKey // Extract the private key
	log.Println("Private key extracted successfully.")

	log.Println("Wallet creation process completed successfully.")
	return account.Address.Hex(), privateKey, nil
}

func (ethdep ethRepo) TransferFunds(fromPrivateKeyHex string, fromAddressHex string, toAddressHex string, amount *big.Int, gasPrice *big.Int, gasLimit uint64, chainID *big.Int) (*types.Transaction, error) {
	// Convert addresses
	fromAddress := common.HexToAddress(fromAddressHex)
	toAddress := common.HexToAddress(toAddressHex)

	log.Println("fromPrivateKeyHex", fromPrivateKeyHex)

	// Parse the private key
	privateKey, err := crypto.HexToECDSA(fromPrivateKeyHex)
	if err != nil {
		log.Printf("Error parsing private key: %v", err)
		return nil, err
	}

	// Verify the sender address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}
	derivedAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	if derivedAddress != fromAddress {
		return nil, fmt.Errorf("derived address (%s) does not match fromAddress (%s)", derivedAddress.Hex(), fromAddress.Hex())
	}

	// Get the nonce
	nonce, err := ethdep.ethereumClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Printf("Error fetching nonce: %v", err)
		return nil, err
	}

	// Create transaction data
	tx := types.NewTransaction(nonce, toAddress, amount, gasLimit, gasPrice, nil)

	log.Println("Transaction data", tx)

	// Sign the transaction using LegacyTxType for Ganache compatibility
	signedTx, err := types.SignNewTx(privateKey, types.NewEIP155Signer(chainID), &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &toAddress,
		Value:    amount,
		Data:     nil,
	})
	if err != nil {
		log.Printf("Error signing transaction: %v", err)
		return nil, err
	}

	// Verify the signature
	signer := types.NewEIP155Signer(chainID)
	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		log.Printf("Error recovering sender from signature: %v", err)
		return nil, err
	}
	if sender != fromAddress {
		return nil, fmt.Errorf("recovered sender (%s) does not match fromAddress (%s)", sender.Hex(), fromAddress.Hex())
	}

	return signedTx, nil
}

func (ethdep ethRepo) PreloadTokens(walletAddress string, amount *big.Int) error {
	log.Println("Starting the token preloading process...")
	if ethdep.ethereumClient == nil {
		return fmt.Errorf("Ethereum client is not initialized")
	}

	// Define the private key and sender address
	fromPrivateKeyHex := "ea97d6b94a9086cf06acdd6504b9e78e67af38d7fefaea5d05f96e2e9532aeea"
	fromAddressHex := "0x6AA382D6b0586027CF8491a81F691DC43AE281Da"

	// Log the recipient address
	toAddress := walletAddress
	log.Printf("From Address: %s, To Address: %s", fromAddressHex, toAddress)

	// Set gas price and gas limit
	gasPrice := big.NewInt(20000000000) // 20 Gwei
	gasLimit := uint64(21000)
	chainID := big.NewInt(1337) // For Ganache

	// Call TransferFunds to handle the actual fund transfer
	signedTx, err := ethdep.TransferFunds(fromPrivateKeyHex, fromAddressHex, toAddress, amount, gasPrice, gasLimit, chainID)
	if err != nil {
		log.Printf("Error during fund transfer: %v", err)
		return err
	}

	// Send the transaction
	err = ethdep.ethereumClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Printf("Error sending transaction: %v", err)
		return err
	}

	log.Printf("Tokens successfully preloaded to wallet: %s. Transaction Hash: %s",
		toAddress, signedTx.Hash().Hex())
	return nil
}
