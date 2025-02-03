package handlers

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/internal/blockchain/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/repository/postgres"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/bcrypt"
)

// BalanceResponse defines the structure of the API response.
type BalanceResponse struct {
	WalletID string `json:"wallet_id"`
	Balance  string `json:"balance"`
}

// Get Balance Handler
func GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Incoming Request On Getbalance Handler")

	// Retrieve user info from context
	userInfo, ok := r.Context().Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	// Extract query parameters
	queryUserID := r.URL.Query().Get("userid")
	queryEmail := r.URL.Query().Get("email")

	var walletID string
	var err error

	// If admin (UserRole == 3) and `userid` or `email` is provided, fetch corresponding wallet ID
	if userInfo.UserRole == 3 && (queryUserID != "" || queryEmail != "") {
		walletID, err = postgres.GetWalletID(queryEmail, queryUserID)
		if err != nil {
			http.Error(w, "Failed to retrieve wallet ID for the requested account", http.StatusInternalServerError)
			return
		}
	} else {
		// Default case: Retrieve wallet ID of the authenticated user
		walletID, err = postgres.GetWalletID(userInfo.UserEmail, userInfo.UserID)
		if err != nil {
			http.Error(w, "Failed to retrieve wallet ID", http.StatusInternalServerError)
			return
		}
	}

	// Get the balance using the new function
	balance, err := GetBalanceByWalletID(walletID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response
	response := BalanceResponse{
		WalletID: walletID,
		Balance:  balance.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetBalanceByWalletID(walletID string) (*big.Float, error) {
	// Validate wallet ID
	if !common.IsHexAddress(walletID) {
		return nil, fmt.Errorf("invalid wallet address")
	}

	// Fetch balance from blockchain
	balance, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(walletID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch balance: %w", err)
	}

	// Convert balance from Wei to ETH
	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))
	return ethBalance, nil
}

type TransferRequest struct {
	RecipientUserID string `json:"recipient_user_id"`
	AmountETH       string `json:"amount"` // ETH as string to avoid precision loss
	Password        string `json:"password"`
}

func TransferFundsHandler(w http.ResponseWriter, r *http.Request) {
	// Decode user info from request context
	userInfo, ok := r.Context().Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Get sender wallet ID
	senderWalletID, err := postgres.GetWalletID(userInfo.UserEmail, userInfo.UserID)
	if err != nil {
		http.Error(w, "Sender wallet not found", http.StatusInternalServerError)
		return
	}

	// Get recipient wallet ID
	recipientWalletID, err := postgres.GetWalletID("", req.RecipientUserID)
	if err != nil {
		http.Error(w, "Recipient wallet not found", http.StatusInternalServerError)
		return
	}

	// Fetch user from DB (Replace with actual DB query)
	user, err := postgres.GetUserByEmail(userInfo.UserEmail)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid Password", http.StatusUnauthorized)
		return
	}

	privateKey, err := postgres.RetrievePrivateKey(userInfo.UserID, "")

	if err != nil {
		log.Println("Error Retrieving Private Key", err)
		return
	}

	// Convert password to private key
	privateKeyHex, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		http.Error(w, "Invalid private key", http.StatusBadRequest)
		return
	}

	// Verify sender address matches derived address
	senderAddress := common.HexToAddress(senderWalletID)
	publicKey := privateKeyHex.Public().(*ecdsa.PublicKey)
	derivedAddress := crypto.PubkeyToAddress(*publicKey)
	if senderAddress != derivedAddress {
		http.Error(w, "Unauthorized: sender wallet mismatch", http.StatusUnauthorized)
		return
	}

	// Convert amount from string to big.Int
	amount, success := new(big.Int).SetString(req.AmountETH, 10)
	if !success {
		http.Error(w, "Invalid amount format", http.StatusBadRequest)
		return
	}

	// Set gas details and chain ID
	gasPrice := big.NewInt(20000000000) // 20 Gwei
	gasLimit := uint64(21000)
	chainID := big.NewInt(1337) // Ganache

	// Call TransferFunds
	signedTx, err := ethereum.TransferFunds(privateKey, senderWalletID, recipientWalletID, amount, gasPrice, gasLimit, chainID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Transaction failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Signed Transaction:", signedTx)
	// Send transaction
	err = ethereum.EthereumClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to broadcast transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with transaction details
	response := map[string]string{
		"transaction_hash": signedTx.Hash().Hex(),
		"sender":           senderWalletID,
		"recipient":        recipientWalletID,
		"amount":           req.AmountETH,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
