package wallet

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/CodeWithKrushnal/ChainBank/utils"
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

// TransferRequest represents the structure of a transfer request.
type TransferRequest struct {
	RecipientEmail string `json:"recipient_email"`
	AmountETH      string `json:"amount"`
	Password       string `json:"password"`
}

// GetBalanceHandler handles the balance retrieval request.
func (hd Handler) GetBalanceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingUserDetails)

	// Retrieve user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), utils.ErrorTag, utils.UserInfoNotFoundInContext)
		http.Error(w, utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), http.StatusUnauthorized)
		return
	}

	// Get user info from userID
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusInternalServerError)
		return
	}

	// Extract query parameters
	queryUserID := r.URL.Query().Get(utils.RequestUserID)
	queryEmail := r.URL.Query().Get(utils.UserEmail)

	// Get Wallet ID
	walletID, err := hd.Service.GetWalletIDForUser(ctx, userInfo, queryEmail, queryUserID)
	if err != nil {
		slog.Error(err.Error(), utils.ErrorTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get Balance
	balance, err := hd.Service.GetBalanceByWalletID(ctx, walletID)
	if err != nil {
		slog.Error(err.Error(), utils.ErrorTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response
	response := BalanceResponse{
		WalletID: walletID,
		Balance:  balance.String(),
	}

	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// TransferFundsHandler handles fund transfer requests.
func (hd *Handler) TransferFundsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), utils.ErrorTag, utils.UserInfoNotFoundInContext)
		http.Error(w, utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), http.StatusUnauthorized)
		return
	}

	// Retrieve user information by UserID
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusInternalServerError)
		return
	}

	// Decode the request body into TransferRequest
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Process fund transfer
	transaction, fee, err := hd.Service.TransferFunds(ctx, userInfo, req)
	if err != nil {
		slog.Error(err.Error(), utils.ErrorTag, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log the fees incurred during the transaction
	slog.Info("Transaction fees", "amount", fee)

	// Set response header and encode transaction to JSON
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(transaction); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// GetTransactionsHandler handles requests to retrieve transactions for a user.
func (hd Handler) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingUserDetails) // Log the start of the handler

	// Retrieve user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), utils.ErrorTag, utils.UserInfoNotFoundInContext)
		http.Error(w, utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), http.StatusUnauthorized)
		return
	}

	// Get user info from userID
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusInternalServerError)
		return
	}

	// Determine common email based on user role
	var commonEmail string
	if userInfo.UserRole != 3 {
		commonEmail = userInfo.UserEmail
	}

	// Retrieve sender and receiver email from query parameters
	senderEmail := r.URL.Query().Get(utils.SenderEmail)
	receiverEmail := r.URL.Query().Get(utils.ReceiverEmail)
	fromTimeStr := r.URL.Query().Get(utils.FromTime)
	toTimeStr := r.URL.Query().Get(utils.ToTime)

	var fromTime, toTime time.Time

	if fromTimeStr != "" {
		var err error
		fromTime, err = time.Parse(time.RFC3339, fromTimeStr)
		if err != nil {
			slog.Error(utils.ErrInvalidFromTimeFormat.Error(), utils.ErrorTag, err)
			http.Error(w, utils.ErrInvalidFromTimeFormat.Error(), http.StatusBadRequest)
			return
		}
	}

	if toTimeStr != "" {
		var err error
		toTime, err = time.Parse(time.RFC3339, toTimeStr)
		if err != nil {
			slog.Error(utils.ErrInvalidToTimeFormat.Error(), utils.ErrorTag, err)
			http.Error(w, utils.ErrInvalidToTimeFormat.Error(), http.StatusBadRequest)
			return
		}
	} else {
		toTime = time.Time{} // Set to zero value if empty
	}
	slog.Info(utils.LogRetrievingTransactions, senderEmail, receiverEmail)

	// Fetch transactions based on the provided parameters
	transactions, err := hd.Service.FetchTransactions(ctx, TransactionFilter{
		TransactionID: uuid.Nil,
		SenderEmail:   senderEmail,
		ReceiverEmail: receiverEmail,
		CommonEmail:   commonEmail,
		FromTime:      fromTime,
		ToTime:        toTime,
		Page:          1,
		Limit:         10,
	})
	if err != nil {
		slog.Error(utils.ErrRetrievingOffersFromApplicationID.Error(), utils.ErrorTag, err) // Use a relevant error message
		http.Error(w, utils.ErrRetrievingOffersFromApplicationID.Error(), http.StatusInternalServerError)
		return
	}

	// Set response header and encode transactions to JSON
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(transactions); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}
