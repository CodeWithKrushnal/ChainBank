package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/utils"
)

// SignupRequest represents the signup request body
type SignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	DOB      string `json:"dob"`
	Role     string `json:"role"`
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
		slog.Error(utils.ErrRetrievingUserDetails.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidDuration.Error(), http.StatusBadRequest)
		return
	}

	// Create a new user account
	walletAddress, err := hd.Service.CreateUserAccount(ctx, req)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare the response
	resp := SignupResponse{
		Message:       utils.SuccessMessage,
		WalletAddress: walletAddress,
	}
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
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
	slog.Info(utils.LogRetrievingUserDetails, "originIP", originIP)

	// Decode the request body into Credentials struct
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		slog.Error(utils.ErrInvalidDuration.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidDuration.Error(), http.StatusBadRequest)
		return
	}

	// Authenticate the user
	response, err := hd.Service.AuthenticateUser(ctx, struct {
		Email    string
		Password string
	}(credentials), originIP)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusUnauthorized)
		return
	}

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
