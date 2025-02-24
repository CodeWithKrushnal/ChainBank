package loan

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Handler struct {
	Service Service
}

// Constructor function
func NewHandler(service Service) *Handler {
	return &Handler{Service: service}
}

// Structs

type LoanApplicationPayload struct {
	Amount       float64 `json:"amount"`
	InterestRate float64 `json:"interestRate"`
	TermMonths   int     `json:"termMonths"`
}

type LoanOfferPayload struct {
	Amount       float64 `json:"amount"`
	InterestRate float64 `json:"interest_rate"`
	Duration     int     `json:"duration"`
}

// Handlers

// Create Loan New Application Handler
func (hd Handler) CreateLoanApplicationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error(), utils.ErrorTag, utils.ErrUnauthorized)
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Decode application body
	var payload LoanApplicationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Validate request data
	if payload.Amount <= 0 {
		slog.Error(utils.ErrInvalidAmount.Error(), utils.ErrorTag, utils.ErrInvalidAmount)
		http.Error(w, utils.ErrInvalidAmount.Error(), http.StatusBadRequest)
		return
	}
	if payload.InterestRate <= 0 {
		slog.Error(utils.ErrInvalidInterestRate.Error(), utils.ErrorTag, utils.ErrInvalidInterestRate)
		http.Error(w, utils.ErrInvalidInterestRate.Error(), http.StatusBadRequest)
		return
	}
	if payload.TermMonths <= 0 {
		slog.Error(utils.ErrInvalidTermMonths.Error(), utils.ErrorTag, utils.ErrInvalidTermMonths)
		http.Error(w, utils.ErrInvalidTermMonths.Error(), http.StatusBadRequest)
		return
	}

	// Call the service to create loan application
	loanapplication, err := hd.Service.CreateLoanapplication(ctx, UserID, payload.Amount, payload.InterestRate, payload.TermMonths)
	if err != nil {
		slog.Error(utils.ErrCreateLoanApplication.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrCreateLoanApplication.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(loanapplication); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// GetLoanApplicationByIDHandler retrieves a loan application by its application ID.
func (hd Handler) GetLoanApplicationByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	applicationIDStr, exists := vars[utils.ApplicationID]
	if !exists || applicationIDStr == "" {
		http.Error(w, utils.ErrInvalidApplicationID.Error(), http.StatusBadRequest)
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(applicationIDStr); err != nil {
		http.Error(w, utils.ErrInvalidApplicationID.Error(), http.StatusBadRequest)
		return
	}

	// Fetch loan application details
	loanApplications, err := hd.Service.GetLoanapplications(ctx, applicationIDStr, "", "")
	if err != nil {
		slog.Error(utils.ErrRetrievingApplication.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingApplication.Error(), http.StatusInternalServerError)
		return
	}

	// Check if no loan applications found
	if len(loanApplications) == 0 {
		http.Error(w, utils.ErrNoLoanApplicationFound.Error(), http.StatusNotFound)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanApplications); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// GetLoanApplicationsHandler retrieves loan applications based on borrowerID and status.
func (hd Handler) GetLoanAppliactionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingLoanApplications)

	// Extract user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Fetch user information from the database
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUser.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUser.Error(), http.StatusInternalServerError)
		return
	}

	// Extract query parameters
	query := r.URL.Query()
	borrowerID := query.Get(utils.RequestUserID)
	status := query.Get(utils.Status)

	// Authorization check using helper function
	if err := hd.checkUserAuthorization(userInfo, borrowerID, status); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Fetch loan applications based on query parameters
	loanApplications, err := hd.Service.GetLoanapplications(ctx, "", borrowerID, status)
	if err != nil {
		slog.Error(utils.ErrFailedToFetchLoanApplications.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToFetchLoanApplications.Error(), http.StatusInternalServerError)
		return
	}

	// Check if no loan applications found
	if len(loanApplications) == 0 {
		http.Error(w, utils.ErrNoLoanApplicationFound.Error(), http.StatusNotFound)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanApplications); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// Create Loan Offer Handler
func (hd Handler) CreateLoanOfferHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogAcceptingLoanOffer) // Log the incoming request

	// Extract UserID from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok || UserID == "" {
		slog.Error(utils.ErrUnauthorized.Error()) // Log unauthorized access
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	applicationID, err := uuid.Parse(vars[utils.ApplicationID])
	if err != nil {
		slog.Error(utils.ErrInvalidApplicationID.Error(), utils.ErrorTag, err) // Log the error
		http.Error(w, utils.ErrInvalidApplicationID.Error(), http.StatusBadRequest)
		return
	}

	var payload LoanOfferPayload
	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error(utils.ErrInvalidRequestPayload.Error(), utils.ErrorTag, err) // Log the error
		http.Error(w, utils.ErrInvalidRequestPayload.Error(), http.StatusBadRequest)
		return
	}

	// Validate request fields
	if payload.Amount <= 0 {
		slog.Error(utils.ErrInvalidAmount.Error(), utils.ErrorTag, utils.ErrInvalidAmount) // Log the error
		http.Error(w, utils.ErrInvalidAmount.Error(), http.StatusBadRequest)
		return
	}
	if payload.InterestRate <= 0 {
		slog.Error(utils.ErrInvalidInterestRate.Error(), utils.ErrorTag, utils.ErrInvalidInterestRate) // Log the error
		http.Error(w, utils.ErrInvalidInterestRate.Error(), http.StatusBadRequest)
		return
	}
	if payload.Duration <= 0 {
		slog.Error(utils.ErrInvalidDuration.Error(), utils.ErrorTag, utils.ErrInvalidDuration) // Log the error
		http.Error(w, utils.ErrInvalidDuration.Error(), http.StatusBadRequest)
		return
	}

	// Call service layer to create the loan offer
	loanOffer, err := hd.Service.CreateLoanOffer(ctx, UserID, payload.Amount, payload.InterestRate, payload.Duration, applicationID.String())
	if err != nil {
		slog.Error(utils.ErrCreateLoanOffer.Error(), utils.ErrorTag, err) // Log the error
		http.Error(w, utils.ErrCreateLoanOffer.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the created loan offer
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(loanOffer); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err) // Log the error
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// Get Loan Offers with Offer ID
func (hd Handler) GetLoanOfferByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	offerIDStr, exists := vars[utils.OfferID]
	if !exists || offerIDStr == "" {
		http.Error(w, utils.ErrInvalidOfferID.Error(), http.StatusBadRequest)
		return
	}

	// Validate UUID format
	_, err := uuid.Parse(offerIDStr)
	if err != nil {
		http.Error(w, utils.ErrInvalidOfferID.Error(), http.StatusBadRequest)
		return
	}

	// Fetch loan application details
	loanApplications, err := hd.Service.GetLoanOffers(ctx, offerIDStr, "", "", "")
	if err != nil {
		slog.Error(utils.ErrFailedToFetchLoanOffers.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToFetchLoanOffers.Error(), http.StatusInternalServerError)
		return
	}

	// Check if no loan applications found
	if len(loanApplications) == 0 {
		http.Error(w, utils.ErrNoLoanOfferFound.Error(), http.StatusNotFound)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanApplications); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// GetOffersByApplicationIDHandler retrieves loan offers based on the provided application ID.
func (hd Handler) GetOffersByApplicationIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingOffersFromApplicationID)

	// Extract UserID from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok || UserID == "" {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	applicationID, err := uuid.Parse(vars[utils.ApplicationID])
	if err != nil {
		slog.Error(utils.ErrInvalidApplicationID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidApplicationID.Error(), http.StatusBadRequest)
		return
	}

	// Fetch loan offers based on application ID
	offer, err := hd.Service.GetLoanOffers(ctx, "", applicationID.String(), "", "")
	if err != nil {
		slog.Error(utils.ErrRetrievingOffersFromApplicationID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingOffersFromApplicationID.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the retrieved loan offers
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusOK) // Changed to StatusOK for successful retrieval
	if err := json.NewEncoder(w).Encode(offer); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// GetLoanOffersHandler retrieves loan offers based on query parameters.
func (hd Handler) GetLoanOffersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingOffersFromOfferID)

	// Extract UserID from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Retrieve user information
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUser.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUser.Error(), http.StatusInternalServerError)
		return
	}

	// Extract query parameters for lenderID and status
	query := r.URL.Query()
	lenderID := query.Get(utils.RequestUserID)
	status := query.Get(utils.Status)

	// Authorization check using helper function
	if err := hd.checkUserAuthorization(userInfo, lenderID, status); err != nil {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Fetch loan offers based on query parameters
	loanOffers, err := hd.Service.GetLoanOffers(ctx, "", "", lenderID, status)
	if err != nil {
		slog.Error(utils.ErrFailedToFetchLoanOffers.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToFetchLoanOffers.Error(), http.StatusInternalServerError)
		return
	}

	// Check if no loan offers found
	if len(loanOffers) == 0 {
		slog.Warn(utils.ErrNoLoanOfferFound.Error())
		http.Error(w, utils.ErrNoLoanOfferFound.Error(), http.StatusNotFound)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanOffers); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// AcceptOfferHandler handles the acceptance of a loan offer.
func (hd Handler) AcceptOfferHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogAcceptingLoanOffer)

	// Extract user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Extract offerID from path parameters
	vars := mux.Vars(r)
	offerID, err := uuid.Parse(vars[utils.OfferID])
	if err != nil {
		slog.Error(utils.ErrInvalidOfferID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidOfferID.Error(), http.StatusBadRequest)
		return
	}

	// Attempt to accept the loan offer
	loan, err := hd.Service.AcceptOffer(ctx, offerID.String(), UserID)
	if err != nil {
		slog.Error(utils.ErrAcceptingOffer.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrAcceptingOffer.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(loan); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// DisburseLoanHandler handles the disbursement of a loan.
func (hd Handler) DisburseLoanHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogDisbursingLoan)

	// Extract user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Retrieve user information from the database
	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusInternalServerError)
		return
	}

	// Extract offerID from path parameters
	vars := mux.Vars(r)
	offerID, err := uuid.Parse(vars[utils.OfferID])
	if err != nil {
		slog.Error(utils.ErrInvalidOfferID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrInvalidOfferID.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve loan offers based on the offerID
	offer, err := hd.Service.GetLoanOffers(ctx, offerID.String(), "", "", "")
	if err != nil {
		slog.Error(utils.ErrRetrievingOffersFromOfferID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingOffersFromOfferID.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the offer status is accepted
	if len(offer) == 0 || offer[0].Status != utils.StatusApproved {
		slog.Error(utils.ErrOfferNotAccepted.Error())
		http.Error(w, utils.ErrOfferNotAccepted.Error(), http.StatusBadRequest)
		return
	}

	// Verify that the user is the lender of the offer
	if offer[0].LenderID.String() != userInfo.UserID {
		slog.Error(utils.ErrNotLender.Error())
		http.Error(w, utils.ErrNotLender.Error(), http.StatusBadRequest)
		return
	}

	// Attempt to disburse the loan
	loan, err := hd.Service.DisburseLoan(ctx, userInfo.UserID, offerID.String())
	if err != nil {
		slog.Error(utils.ErrDisbursingLoan.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrDisbursingLoan.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(loan); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// GetLoanDetailsHandler handles the retrieval of loan details by loan ID.
func (hd *Handler) GetLoanDetailsByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingLoanDetailsByID)

	// Extract user info from context
	UserID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	userInfo, err := hd.Service.GetUserByID(ctx, UserID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUser.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUser.Error(), http.StatusInternalServerError)
		return
	}

	// Extract loan_id from path parameters
	vars := mux.Vars(r)
	loanID := vars[utils.LoanID]

	// Validate loan ID
	if loanID == "" {
		slog.Error(utils.ErrInvalidLoanID.Error())
		http.Error(w, utils.ErrInvalidLoanID.Error(), http.StatusBadRequest)
		return
	}

	// Fetch loan details using the service
	loanDetails, err := hd.Service.GetLoanDetails(ctx, loanID, "", "", "", "", "")
	if err != nil {
		slog.Error(utils.ErrRetrievingLoanDetails.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingLoanDetails.Error(), http.StatusInternalServerError)
		return
	}

	// Check if loan details were found
	if len(loanDetails) == 0 {
		slog.Error(utils.ErrNoLoanFound.Error())
		http.Error(w, utils.ErrNoLoanFound.Error(), http.StatusNotFound)
		return
	}

	slog.Info(utils.LogLoanDetailsByIDRetrievedSuccessfully)
	if loanDetails[0].BorrowerID != userInfo.UserID && loanDetails[0].LenderID != userInfo.UserID && userInfo.UserRole != 3 {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanDetails[0]); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// GetLoanDetailsHandler handles the retrieval of loan details based on various parameters.
func (hd *Handler) GetLoanDetailsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.Info(utils.LogRetrievingLoanDetails) // Log the incoming request

	// Extract user info from context
	userID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrRetrievingUser.Error())
		http.Error(w, utils.ErrRetrievingUser.Error(), http.StatusUnauthorized)
		return
	}

	// Fetch user information from the service
	userInfo, err := hd.Service.GetUserByID(ctx, userID)
	if err != nil {
		slog.Error(utils.ErrRetrievingUserByID.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingUserByID.Error(), http.StatusInternalServerError)
		return
	}

	// Extract parameters from query
	query := r.URL.Query()
	offerID := query.Get(utils.LoanOfferID)
	applicationID := query.Get(utils.LoanApplicationID)
	borrowerID := query.Get(utils.BorrowerID)
	lenderID := query.Get(utils.LenderID)
	status := query.Get(utils.Status)

	// Validate that at least one parameter is provided
	if offerID == "" && applicationID == "" && borrowerID == "" && lenderID == "" && status == "" {
		slog.Error(utils.ErrMissingParameters.Error()) // Use standard error message
		http.Error(w, utils.ErrMissingParameters.Error(), http.StatusBadRequest)
		return
	}

	// Fetch loan details based on provided parameters
	loanDetails, err := hd.Service.GetLoanDetails(ctx, "", offerID, borrowerID, lenderID, status, applicationID)
	if err != nil {
		slog.Error(utils.ErrRetrievingLoanDetails.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrRetrievingLoanDetails.Error(), http.StatusInternalServerError)
		return
	}

	// Check if loan details were found
	if len(loanDetails) == 0 {
		slog.Error(utils.ErrNoLoanFound.Error())
		http.Error(w, utils.ErrNoLoanFound.Error(), http.StatusNotFound)
		return
	}

	// Check authorization based on userID and roles
	if loanDetails[0].BorrowerID != userID && loanDetails[0].LenderID != userID && userInfo.UserRole != 3 {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanDetails[0]); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// CalculatePayableHandler handles the request to calculate the total payable amount for a loan.
func (hd *Handler) CalculatePayableHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract userID from context (assuming it's set during authentication)
	userID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Extract loan_id from URL path parameters
	vars := mux.Vars(r)
	loanID := vars[utils.LoanID]

	// Validate loanID
	if loanID == "" {
		slog.Error(utils.ErrInvalidLoanID.Error())
		http.Error(w, utils.ErrInvalidLoanID.Error(), http.StatusBadRequest)
		return
	}

	// Calculate total payable amount
	payableBreakdown, err := hd.Service.CalculateTotalPayable(ctx, loanID, userID)
	if err != nil {
		slog.Error(utils.ErrCalculatingTotalPayable.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrCalculatingTotalPayable.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(payableBreakdown); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// HandleSettleLoan handles the request to settle a loan.
func (hd *Handler) SettleLoanHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract userID from context (assuming it's set during authentication)
	userID, ok := ctx.Value(utils.CtxUserID).(string)
	if !ok {
		slog.Error(utils.ErrUnauthorized.Error())
		http.Error(w, utils.ErrUnauthorized.Error(), http.StatusUnauthorized)
		return
	}

	// Extract loan_id from URL path parameters
	vars := mux.Vars(r)
	loanID := vars[utils.LoanID]

	// Validate loanID
	if loanID == "" {
		slog.Error(utils.ErrInvalidLoanID.Error())
		http.Error(w, utils.ErrInvalidLoanID.Error(), http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(loanID); err != nil {
		slog.Error(utils.ErrInvalidLoanIDFormat.Error())
		http.Error(w, utils.ErrInvalidLoanIDFormat.Error(), http.StatusBadRequest)
		return
	}

	// Call the SettleLoan service function
	settledLoan, err := hd.Service.SettleLoan(ctx, userID, loanID)
	if err != nil {
		slog.Error(utils.ErrSettlingLoan.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrSettlingLoan.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON data
	w.Header().Set(utils.ContentTypeHeader, utils.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(settledLoan); err != nil {
		slog.Error(utils.ErrFailedToEncodeResponse.Error(), utils.ErrorTag, err)
		http.Error(w, utils.ErrFailedToEncodeResponse.Error(), http.StatusInternalServerError)
	}
}

// Helper function for authorization checks
func (hd Handler) checkUserAuthorization(userInfo utils.User, requestUserID, status string) error {
	if requestUserID != "" && userInfo.UserRole != 3 && requestUserID != userInfo.UserID {
		return utils.ErrUnauthorized
	} else if requestUserID == "" && status != utils.StatusOpen && userInfo.UserRole != 3 {
		return utils.ErrUnauthorized
	}
	return nil
}