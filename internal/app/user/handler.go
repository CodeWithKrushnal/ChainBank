package user

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/CodeWithKrushnal/ChainBank/utils"
)

// SignupRequest represents the signup request body
type SignupRequest struct {
	Username               string `json:"username"`
	Email                  string `json:"email"`
	Password               string `json:"password"`
	FullName               string `json:"full_name"`
	DOB                    string `json:"dob"`
	Role                   string `json:"role"`
	EmailVerificationToken string `json:"email_verification_token"`
}

// SignupResponse represents the signup response
type SignupResponse struct {
	Message       string `json:"message"`
	WalletAddress string `json:"wallet_address"`
}

// Define a reusable struct for credentials
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Handler struct
type Handler struct {
	Service Service
}

// Constructor function
func NewHandler(service Service) *Handler {
	return &Handler{Service: service}
}

// KYCRequest represents the KYC request body
type KYCRequest struct {
	DocumentType   string `json:"document_type"`
	DocumentNumber string `json:"document_number"`
}

// KYCRequestAction represents the KYC request action body
type KYCRequestAction struct {
	KYCID              string `json:"kyc_id"`
	VerificationStatus string `json:"verification_status"`
}

// Handlers

// SignupHandler handles the signup request and creates a new user account.
func (hd Handler) SignupHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingUserDetails) // Log the incoming request

	var req SignupRequest
	// Decode the request body into SignupRequest struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Create a new user account
	user, err := hd.Service.CreateUserAccount(ctx, req)
	user.Password = ""
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// SignInHandler handles the user sign-in request and authenticates the user.
func (hd Handler) SignInHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var credentials Credentials

	// Log the origin IP address of the request
	originIP := r.RemoteAddr
	// Convert IPv6 loopback with port to IPv4 loopback
	if strings.HasPrefix(originIP, "[::1]:") {
		originIP = "127.0.0.1"
	}
	slog.Info(utils.LogRetrievingUserDetails, "originIP", originIP)

	// Decode the request body into Credentials struct
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Authenticate the user
	response, err := hd.Service.AuthenticateUser(ctx, struct {
		Email    string
		Password string
	}(credentials), originIP)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidCredentials.Error(), http.StatusUnauthorized)
		return
	}

	timestamp := time.Now().Format(time.RFC1123)
	message := fmt.Sprintf("A new login was detected from the device with IP: <strong>%s</strong> at <strong>%s</strong>.<br>If this was not you, please take appropriate action.", originIP, timestamp)
	hd.Service.SendEmail(ctx, credentials.Email, "New Login Alert", message)

	// Set the response header and encode the response
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// RequestKYCHandler handles the request for KYC verification.
func (hd Handler) RequestKYCHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingUserDetails)

	// Retrieve user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Get user info from userID
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusInternalServerError)
		return
	}

	// Define the request structure for KYC
	var KYCRequest KYCRequest
	if err := json.NewDecoder(r.Body).Decode(&KYCRequest); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	slog.Info(utils.LogReceivedKYCRequest, "request", KYCRequest)
	kycID, err := hd.Service.InsertKYCVerificationService(ctx, userInfo.UserEmail, KYCRequest.DocumentType, KYCRequest.DocumentNumber, "Pending")
	if err != nil {
		slog.Error(utils.ErrInsertingKYCVerification.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInsertingKYCVerification.Error(), http.StatusInternalServerError)
		return
	}

	// Set response header and encode the response
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(map[string]string{utils.KYCID: kycID}); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// GetKYCRequestsHandler retrieves all KYC verification records for the authenticated user.
func (hd Handler) GetKYCRequestsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingUserDetails)

	// Retrieve user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Get user info from userID
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the user has admin role
	if userInfo.UserRole != 3 {
		slog.Error(utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), "userID", UserID)
		http.Error(w, utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), http.StatusUnauthorized)
		return
	}

	// Retrieve all KYC verification records
	kycRecords, err := hd.Service.GetAllKYCVerificationsService(ctx)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserDetails.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserDetails.Error(), http.StatusInternalServerError)
		return
	}

	// Set response header for JSON content
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)

	// Check if there are no KYC records
	if kycRecords == nil {
		slog.Info(utils.LogNoNewKYCRequestsFound)
		w.Write([]byte(utils.ErrNoNewKYCRequestsFound))
	} else {
		if err := json.NewEncoder(w).Encode(kycRecords); err != nil {
			slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
			http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
		}
	}
}

// KYCRequestActionHandler updates KYC verification status.
func (hd Handler) KYCRequestActionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogAcceptingLoanOffer) // Log the incoming request

	// Retrieve user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Get user info from userID
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the user has admin role
	if userInfo.UserRole != 3 {
		slog.Error(utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), "userID", UserID)
		http.Error(w, utils.ErrUnauthorizedAccessAttemptByNonAdminUser.Error(), http.StatusUnauthorized)
		return
	}

	// Define request structure for KYC update
	var KYCRequestAction KYCRequestAction

	// Decode the request body
	if err := json.NewDecoder(r.Body).Decode(&KYCRequestAction); err != nil {
		slog.Error(utils.ErrInvalidDuration.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidDuration.Error(), http.StatusBadRequest)
		return
	}

	// Validate KYCID and VerificationStatus
	if KYCRequestAction.KYCID == "" || KYCRequestAction.VerificationStatus == "" {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Determine verification status
	var verificationStatus string
	switch KYCRequestAction.VerificationStatus {
	case "1":
		verificationStatus = utils.Verified
	case "2":
		verificationStatus = utils.Unverified
	default:
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Update KYC verification status
	err = hd.Service.UpdateKYCVerificationStatusService(ctx, KYCRequestAction.KYCID, verificationStatus, UserID)
	if err != nil {
		slog.Error(utils.ErrUpdatingKYCVerificationStatus.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrUpdatingKYCVerificationStatus.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with success message
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{utils.SuccessMessage: utils.KYCStatusUpdatedSuccessfully})
}

// GetKYCDetailedInfoHandler handles requests to retrieve KYC details based on kyc_id or user_email.
func (hd Handler) GetKYCDetailedInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Retrieve kyc_id and user_email from query parameters
	kycID := r.URL.Query().Get(utils.KYCID)
	userEmail := r.URL.Query().Get(utils.UserEmail)

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

	// If user is not an admin, restrict access to KYC ID
	if userInfo.UserRole != 3 {
		kycID = ""
		userEmail = userInfo.UserEmail
	} else {
		if userEmail == "" {
			userEmail = userInfo.UserEmail
		}
	}

	// Validate that exactly one of kyc_id or user_email is provided
	if (kycID == "" && userEmail == "") || (kycID != "" && userEmail != "") {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, utils.BothKYCIDAndUserEmailProvided)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Fetch KYC details using the provided kycID or userEmail
	kycDetails, err := hd.Service.GetKYCDetailedInfo(ctx, kycID, userEmail)
	if err != nil {
		slog.Error(utils.ErrRetrievingKYCDetails.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingKYCDetails.Error(), http.StatusInternalServerError)
		return
	}

	// Set response header and encode KYC details to JSON
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(kycDetails); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// ResetPasswordHandler handles the request to reset a user's password.
func (hd Handler) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info("Reset Password Request Received")

	var req struct {
		Email string `json:"email"`
	}

	// Decode the request body into the struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Check if the user exists
	_, err := hd.Service.GetUserByEmail(ctx, req.Email)
	if err != nil {
		slog.Error(utils.ErrRetrievingUser.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUser.Error(), http.StatusNotFound)
		return
	}

	// Generate a reset token (this should be a secure token generation)
	err = hd.Service.GenerateAndSendResetToken(ctx, req.Email) // Ensure this function exists in utils
	if err != nil {
		slog.Error(utils.ErrGeneratingResetToken.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrGeneratingResetToken.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Reset token sent to your email"))
}

// UpdatePasswordHandler handles the request to update the user's password using the reset token.
func (hd Handler) UpdatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info("Update Password Request Received")

	var req struct {
		ResetToken  string `json:"reset_token"`
		NewPassword string `json:"new_password"`
	}

	// Decode the request body into the struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Verify the reset token and get the user ID
	_, err := hd.Service.ValidateResetToken(req.ResetToken)
	if err != nil {
		slog.Error("Invalid or expired reset token", utils.ErrorTag, err)
		http.Error(w, "Invalid or expired reset token", http.StatusUnauthorized)
		return
	}

	// Update the user's password
	err = hd.Service.ValidateAndResetPassword(ctx, req.ResetToken, req.NewPassword)
	if err != nil {
		slog.Error("Failed to update password", utils.ErrorTag, err)
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Password updated successfully"))
}

func (hd Handler) GenerateEmailVerificationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogGenerateEmailVerificationRequest)

	var req struct {
		Email string `json:"email"`
	}

	// Decode the request body into the struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Extract the origin IP from the request
	originIP := r.RemoteAddr

	// Generate the email verification token
	token, err := hd.Service.GenerateEmailVerificationToken(ctx, req.Email, originIP)
	if err != nil {
		slog.Error(utils.ErrFailedToGenerateToken.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToGenerateToken.Error(), http.StatusInternalServerError)
		return
	}

	// Send the token via email
	err = hd.Service.SendEmail(ctx, req.Email, "Email Verification", "Your verification token is: "+token)
	if err != nil {
		slog.Error(utils.ErrFailedToSendVerificationEmail.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToSendVerificationEmail.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(utils.VerificationEmailSentSuccessfully))
}

func (hd Handler) GetRequestLogsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogFetchingRequestLogs)

	// Extract user ID from context (assuming it's set during authentication)
	userID := r.Context().Value(utils.CtxUserID).(string)

	// Fetch user details to verify role
	user, err := hd.Service.GetUserByID(ctx, userID)
	if err != nil {
		slog.Error(utils.ErrFetchingUser.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Check if the user role is 3
	if user.UserRole != 3 {
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Decode the filter from the request body
	var filter repo.RequestLogFilter
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Get request logs
	requestLogs, err := hd.Service.GetRequestLogs(ctx, filter)
	if err != nil {
		slog.Error(utils.ErrRetrievingRequestLogs.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingRequestLogs.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the request logs
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(requestLogs); err != nil {
		slog.Error(utils.ErrEncodingResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrEncodingResponse.Error(), http.StatusInternalServerError)
		return
	}
}

func (hd Handler) GetRequestLogStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogFetchingRequestLogStats)

	// Extract user ID from context (assuming it's set during authentication)
	userID := r.Context().Value(utils.CtxUserID).(string)

	// Fetch user details to verify role
	user, err := hd.Service.GetUserByID(ctx, userID)
	if err != nil {
		slog.Error(utils.ErrFetchingUser.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Check if the user role is 3
	if user.UserRole != 3 {
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Decode the filter from the request body
	var filter repo.RequestLogStatsFilter
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Get request log stats
	stats, err := hd.Service.GetRequestLogStats(ctx, filter)
	if err != nil {
		slog.Error(utils.ErrFetchingRequestLogStats.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFetchingRequestLogStats.Error(), http.StatusInternalServerError)
		return
	}

	res:=make(map[string]interface{})
	res["stat"]=stats

	// Respond with the stats
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error(utils.ErrEncodingResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrEncodingResponse.Error(), http.StatusInternalServerError)
		return
	}
}

func (hd Handler) GetUserDetailsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogFetchingUserDetails)

	// Extract user ID from context (assuming it's set during authentication)
	userID := r.Context().Value(utils.CtxUserID).(string)

	// Fetch user details
	user, err := hd.Service.GetUserInfo(ctx, userID)
	if err != nil {
		slog.Error(utils.ErrFetchingUser.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFetchingUser.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the user details
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		slog.Error(utils.ErrEncodingResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrEncodingResponse.Error(), http.StatusInternalServerError)
		return
	}
}

func (hd Handler) GetTransactionStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogFetchingTransactionStats)

	// Decode the filter from the request body
	var filter repo.TransactionStatsFilter
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Get transaction stats
	stats, err := hd.Service.GetTransactionStats(ctx, filter)

	resp:= make(map[string]interface{})
	resp["amount"]=stats

	if err != nil {
		slog.Error(utils.ErrFetchingTransactionStats.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFetchingTransactionStats.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the stats
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error(utils.ErrEncodingResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrEncodingResponse.Error(), http.StatusInternalServerError)
		return
	}
}

type ApiResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		EthUsd string `json:"ethusd"`
	} `json:"result"`
}

// Handler to fetch Ethereum price
func (hd Handler) GetEthPriceHandler(w http.ResponseWriter, r *http.Request) {
	// The URL with your API key (replace "your_api_key" with the actual key)
	ctx := r.Context()
	config,err := hd.Service.GetConfig(ctx)
	if err != nil {
		slog.Error(utils.ErrFetchingConfig.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFetchingConfig.Error(), http.StatusInternalServerError)
		return
	}

	apiURL:=config.EtherscanAPI+"&apikey="+config.EtherscanAPIKey
	
	// Make the HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		http.Error(w, "Failed to fetch Ethereum price", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse the JSON response
	var apiResponse ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	// Check the status of the response
	if apiResponse.Status != "1" {
		http.Error(w, "Error in response from API", http.StatusInternalServerError)
		return
	}

	// Return only the ethusd field
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ethusd": "%s"}`, apiResponse.Result.EthUsd)
}


var queryMap = map[string]string{
	"unique_senders": "SELECT COUNT(DISTINCT sender_wallet_id) FROM transactions",
	"total_users": "SELECT count(user_id) FROM users",
	"total_transactions": "SELECT count(transaction_id) FROM transactions",
	"endpoint_usage": "SELECT endpoint, count(request_id) as count FROM api_requests_log GROUP BY endpoint ORDER BY count DESC LIMIT 10",
	"transaction_types_distribution" :"SELECT transaction_type, count(transaction_id) as count from transactions GROUP BY transaction_type",
}

func (hd Handler) ExecuteQueryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info("Executing query based on request parameter")

	// Get query option from request parameters
	option := r.URL.Query().Get("option")
	if option == "" {
		http.Error(w, "missing query option", http.StatusBadRequest)
		return
	}

	// Validate the option and get the corresponding query
	query, exists := queryMap[option]
	if !exists {
		http.Error(w, "invalid query option", http.StatusBadRequest)
		return
	}

	// Execute the query
	result, err := hd.Service.ExecuteQuery(ctx, query)
	if err != nil {
		slog.Error("error executing query", utils.ErrorTag, err)
		http.Error(w, "error executing query", http.StatusInternalServerError)
		return
	}

	// Prepare response
	resp := map[string]interface{}{"stat": result}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("error encoding response", utils.ErrorTag, err)
		http.Error(w, "error encoding response", http.StatusInternalServerError)
		return
	}
}
