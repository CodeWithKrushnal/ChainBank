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
	service Service
}





// Constructor function
func NewHandler(service Service) Handler {
	return Handler{service: service}
}





// GetBalanceHandler handles the balance retrieval request.
func (hd Handler) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming Request On GetBalance Handler")

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

	// Get Wallet ID
	walletID, err := hd.service.GetWalletIDForUser(ctx, userInfo, queryEmail, queryUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get Balance
	balance, err := hd.service.GetBalanceByWalletID(ctx, walletID)
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
	userInfo, ok := r.Context().Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Process fund transfer
	txHash, fee, err := hd.service.TransferFunds(ctx, userInfo, req)
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

	userInfo, ok := ctx.Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})

	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	var commonEmail string
	if userInfo.UserRole != 3 {
		commonEmail = userInfo.UserEmail
	}

	senderEmail := r.URL.Query().Get("senderEmail")
	receiverEmail := r.URL.Query().Get("receiverEmail")
	log.Print(senderEmail, receiverEmail)

	transactions, err := hd.service.FetchTransactions(ctx, uuid.Nil, senderEmail, "", commonEmail, time.Now(), time.Now(), 1, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}
