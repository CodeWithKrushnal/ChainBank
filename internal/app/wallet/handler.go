package wallet

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// BalanceResponse defines the structure of the API response.
type BalanceResponse struct {
	WalletID string `json:"wallet_id"`
	Balance  string `json:"balance"`
}

type Handler struct {
	Service Service
}

// Constructor function
func NewHandler(service Service) Handler {
	return Handler{Service: service}
}

// GetBalanceHandler handles the balance retrieval request.
func (hd Handler) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming Request On GetBalance Handler")

	// Retrieve user info from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	//get userinfo from userID
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		log.Println("Error retriving user info from user ID")
		http.Error(w, "Error retriving user info from user ID", http.StatusInternalServerError)
		return
	}

	// Extract query parameters
	queryUserID := r.URL.Query().Get("userid")
	queryEmail := r.URL.Query().Get("email")

	// Get Wallet ID
	walletID, err := hd.Service.GetWalletIDForUser(ctx, userInfo, queryEmail, queryUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get Balance
	balance, err := hd.Service.GetBalanceByWalletID(ctx, walletID)
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

// TransferRequest represents the structure of a transfer request.
type TransferRequest struct {
	RecipientEmail string `json:"recipient_email"`
	AmountETH      string `json:"amount"`
	Password       string `json:"password"`
}

// TransferFundsHandler handles fund transfer requests.
func (hd *Handler) TransferFundsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user info from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		log.Println("Error Retrieving the User information from id", err.Error())
	}

	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Process fund transfer
	txHash, fee, err := hd.Service.TransferFunds(ctx, userInfo, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Fees Incurrend :", fee)

	// Respond with transaction details
	response := map[string]string{
		"transaction_hash": txHash,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (hd Handler) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming Request On GetTransactions Handler")

	// Retrieve user info from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	//get userinfo from userID
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		log.Println("Error retriving user info from user ID")
		http.Error(w, "Error retriving user info from user ID", http.StatusInternalServerError)
		return
	}

	var commonEmail string
	if userInfo.UserRole != 3 {
		commonEmail = userInfo.UserEmail
	}

	senderEmail := r.URL.Query().Get("senderEmail")
	receiverEmail := r.URL.Query().Get("receiverEmail")
	log.Print(senderEmail, receiverEmail)

	transactions, err := hd.Service.FetchTransactions(ctx, uuid.Nil, senderEmail, "", commonEmail, time.Now(), time.Now(), 1, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}
