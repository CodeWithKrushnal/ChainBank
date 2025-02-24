package ethereum

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log/slog"
	"math/big"
	"os"

	"github.com/CodeWithKrushnal/ChainBank/utils"
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

// EthRepo interface
type EthRepo interface {
	CreateWallet(password string) (string, *ecdsa.PrivateKey, error)
	TransferFunds(fromPrivateKeyHex string, fromAddressHex string, toAddressHex string, amount *big.Int, gasPrice *big.Int, gasLimit uint64, chainID *big.Int) (*types.Transaction, error)
	PreloadTokens(walletAddress string, amount *big.Int) error
}

// CreateWallet creates a new Ethereum wallet and returns the wallet address and private key
func (ethdep ethRepo) CreateWallet(password string) (string, *ecdsa.PrivateKey, error) {
	// Step 1: Initialize the keystore
	ks := keystore.NewKeyStore("./wallets", keystore.StandardScryptN, keystore.StandardScryptP)
	if ks == nil {
		return "", nil, fmt.Errorf("%s", utils.ErrKeystoreInitFailed)
	}

	// Step 2: Create a new account
	account, err := ks.NewAccount(password)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", utils.ErrAccountCreationFailed, err)
	}

	// Step 3: Extract the private key from the keystore file
	keyJSON, err := os.ReadFile(account.URL.Path) // Read the keystore file
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", utils.ErrKeystoreReadFailed, err)
	}

	key, err := keystore.DecryptKey(keyJSON, password) // Decrypt the keystore file
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", utils.ErrKeyDecryptionFailed, err)
	}

	privateKey := key.PrivateKey // Extract the private key
	return account.Address.Hex(), privateKey, nil
}

// TransferFunds transfers funds between two Ethereum addresses
func (ethdep ethRepo) TransferFunds(fromPrivateKeyHex string, fromAddressHex string, toAddressHex string, amount *big.Int, gasPrice *big.Int, gasLimit uint64, chainID *big.Int) (*types.Transaction, error) {
	// Convert hex addresses to common.Address type
	fromAddress := common.HexToAddress(fromAddressHex)
	toAddress := common.HexToAddress(toAddressHex)

	// Parse the private key from hex string
	privateKey, err := crypto.HexToECDSA(fromPrivateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrTransferFunds, err) // Propagate error
	}

	// Verify the sender address derived from the private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}
	derivedAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	if derivedAddress != fromAddress {
		return nil, fmt.Errorf("derived address (%s) does not match fromAddress (%s)", derivedAddress.Hex(), fromAddress.Hex())
	}

	// Get the nonce for the sender's address
	nonce, err := ethdep.ethereumClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrTransferFunds, err) // Propagate error
	}

	// Sign the transaction using LegacyTxType for compatibility
	signedTx, err := types.SignNewTx(privateKey, types.NewEIP155Signer(chainID), &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       &toAddress,
		Value:    amount,
		Data:     nil,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrTransferFunds, err) // Propagate error
	}

	// Verify the signature of the signed transaction
	signer := types.NewEIP155Signer(chainID)
	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrTransferFunds, err) // Propagate error
	}
	if sender != fromAddress {
		return nil, fmt.Errorf("recovered sender (%s) does not match fromAddress (%s)", sender.Hex(), fromAddress.Hex())
	}

	return signedTx, nil
}

// PreloadTokens preloads tokens into a wallet
func (ethdep ethRepo) PreloadTokens(walletAddress string, amount *big.Int) error {
	// Log the start of the token preloading process
	slog.Info(utils.LogTokenPreloadingStart)

	// Check if the Ethereum client is initialized
	if ethdep.ethereumClient == nil {
		return fmt.Errorf("%s", utils.ErrEthereumClientNotInitialized)
	}

	// Define the private key and sender address
	fromPrivateKeyHex := "ea97d6b94a9086cf06acdd6504b9e78e67af38d7fefaea5d05f96e2e9532aeea"
	fromAddressHex := "0x6AA382D6b0586027CF8491a81F691DC43AE281Da"

	// Set gas price and gas limit
	gasPrice := big.NewInt(20000000000) // 20 Gwei
	gasLimit := uint64(21000)
	chainID := big.NewInt(1337) // For Ganache

	// Call TransferFunds to handle the actual fund transfer
	signedTx, err := ethdep.TransferFunds(fromPrivateKeyHex, fromAddressHex, walletAddress, amount, gasPrice, gasLimit, chainID)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrTransferFunds, err) // Propagate error
	}

	// Send the transaction
	err = ethdep.ethereumClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrWalletTransactionFailed, err) // Propagate error
	}

	slog.Info(fmt.Sprintf(utils.LogTokenPreloadingSuccess, walletAddress, signedTx.Hash().Hex()))
	return nil
}
