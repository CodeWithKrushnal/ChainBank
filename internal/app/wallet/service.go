package wallet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"golang.org/x/crypto/bcrypt"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
)

type service struct {
	userRepo   repo.UserStorer
	walletRepo repo.WalletStorer
	ethRepo    ethereum.EthRepo
}

type Service interface {
	GetWalletIDForUser(userInfo struct {
		UserID    string
		UserEmail string
		UserRole  int
	}, queryEmail, queryUserID string) (string, error)
	GetBalanceByWalletID(walletID string) (*big.Float, error)
	TransferFunds(userInfo struct {
		UserID    string
		UserEmail string
		UserRole  int
	}, req TransferRequest) (string, error)
	ValidateSenderAddress(senderWalletID string, privateKey *ecdsa.PrivateKey) error
	ValidateUserPassword(email, password string) error
}

// Constructor function
func NewService(userRepo repo.UserStorer, walletRepo repo.WalletStorer, ethRepo ethereum.EthRepo) Service {
	return service{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		ethRepo:    ethRepo,
	}
}

// GetWalletIDForUser retrieves the wallet ID based on user role and query params.
func (sd service) GetWalletIDForUser(userInfo struct {
	UserID    string
	UserEmail string
	UserRole  int
}, queryEmail, queryUserID string) (string, error) {
	if userInfo.UserRole == 3 && (queryUserID != "" || queryEmail != "") {
		return sd.walletRepo.GetWalletID(queryEmail, queryUserID)
	}
	return sd.walletRepo.GetWalletID(userInfo.UserEmail, userInfo.UserID)
}

// GetBalanceByWalletID retrieves the wallet balance from the blockchain.
func (sd service) GetBalanceByWalletID(walletID string) (*big.Float, error) {
	if !common.IsHexAddress(walletID) {
		return nil, fmt.Errorf("invalid wallet address")
	}

	balance, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(walletID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch balance: %w", err)
	}

	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))
	return ethBalance, nil
}

// TransferFunds handles the fund transfer logic.
func (sd service) TransferFunds(userInfo struct {
	UserID    string
	UserEmail string
	UserRole  int
}, req TransferRequest) (string, error) {
	// Get sender and recipient wallet IDs
	senderWalletID, err := sd.walletRepo.GetWalletID(userInfo.UserEmail, userInfo.UserID)
	if err != nil {
		return "", fmt.Errorf("sender wallet not found")
	}

	recipientWalletID, err := sd.walletRepo.GetWalletID("", req.RecipientUserID)
	if err != nil {
		return "", fmt.Errorf("recipient wallet not found")
	}

	// Validate user password
	err = sd.ValidateUserPassword(userInfo.UserEmail, req.Password)
	if err != nil {
		return "", err
	}

	// Retrieve sender's private key
	privateKeyHex, err := sd.walletRepo.RetrievePrivateKey(userInfo.UserID, "")
	if err != nil {
		return "", fmt.Errorf("error retrieving private key: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key")
	}

	// Validate sender address
	if err := sd.ValidateSenderAddress(senderWalletID, privateKey); err != nil {
		return "", err
	}

	// Convert amount
	amount, success := new(big.Int).SetString(req.AmountETH, 10)
	if !success {
		return "", fmt.Errorf("invalid amount format")
	}

	// Set gas details and chain ID
	gasPrice := big.NewInt(20000000000) // 20 Gwei
	gasLimit := uint64(21000)
	chainID := big.NewInt(1337) // Ganache

	privateKeyHexStr := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))

	// Transfer funds
	signedTx, err := sd.ethRepo.TransferFunds(privateKeyHexStr, senderWalletID, recipientWalletID, amount, gasPrice, gasLimit, chainID)
	if err != nil {
		return "", fmt.Errorf("transaction failed: %w", err)
	}

	// Send transaction
	err = ethereum.EthereumClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	return signedTx.Hash().Hex(), nil
}

// ValidateSenderAddress ensures the sender's wallet matches the derived address.
func (sd service) ValidateSenderAddress(senderWalletID string, privateKey *ecdsa.PrivateKey) error {
	senderAddress := common.HexToAddress(senderWalletID)
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	derivedAddress := crypto.PubkeyToAddress(*publicKey)

	if senderAddress != derivedAddress {
		return fmt.Errorf("unauthorized: sender wallet mismatch")
	}

	return nil
}

// ValidateUserPassword verifies the user's password.
func (sd service) ValidateUserPassword(email, password string) error {
	user, err := sd.userRepo.GetUserByEmail(email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return fmt.Errorf("invalid password")
	}

	return nil
}
