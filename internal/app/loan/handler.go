package loan

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
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
	log.Println("Incoming application On CreateLoanapplicationHandler")

	// Extract user info from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	// Decode application body
	var payload LoanApplicationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate request data
	if payload.Amount <= 0 {
		http.Error(w, "Amount must be greater than zero", http.StatusBadRequest)
		return
	}
	if payload.InterestRate <= 0 {
		http.Error(w, "Interest rate must be greater than zero", http.StatusBadRequest)
		return
	}
	if payload.TermMonths <= 0 {
		http.Error(w, "Term months must be greater than zero", http.StatusBadRequest)
		return
	}

	// Call the service to create loan application
	loanapplication, err := hd.Service.CreateLoanapplication(ctx, UserID, payload.Amount, payload.InterestRate, payload.TermMonths)
	if err != nil {
		http.Error(w, "Failed to create loan application", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(loanapplication)
}

// Get Loan Application with Application ID
func (hd Handler) GetLoanApplicationByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	applicationIDStr, exists := vars["application_id"]
	if !exists || applicationIDStr == "" {
		http.Error(w, "application_id is required", http.StatusBadRequest)
		return
	}

	// Validate UUID format
	_, err := uuid.Parse(applicationIDStr)
	if err != nil {
		http.Error(w, "invalid application_id format", http.StatusBadRequest)
		return
	}

	// Fetch loan application details
	loanApplications, err := hd.Service.GetLoanapplications(ctx, applicationIDStr, "", "")
	if err != nil {
		log.Printf("Error fetching loan applications: %v", err)
		http.Error(w, "failed to fetch loan application", http.StatusInternalServerError)
		return
	}

	// Check if no loan applications found
	if len(loanApplications) == 0 {
		http.Error(w, "no loan application found", http.StatusNotFound)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanApplications); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// Get Loan Appliactions with borrowerID and status
func (hd Handler) GetLoanAppliactionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming request On GetLoanAppliactionsHandler")

	// Extract user info from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	userInfo, err := hd.Service.GetUserByID(ctx, UserID)

	if err != nil {
		http.Error(w, "Error Retriving the user from DB", http.StatusInternalServerError)
		return
	}

	// Extract query parameters
	query := r.URL.Query()
	borrowerID := query.Get("user_id")
	status := query.Get("status")

	// Authorization check: Non-admin users cannot access other user's data
	if borrowerID != "" && userInfo.UserRole != 3 && borrowerID != userInfo.UserID {
		http.Error(w, "Non Admin Roles Cannot Access other user's info", http.StatusUnauthorized)
		return
	} else if borrowerID == "" && status != "open" && userInfo.UserRole != 3 {
		http.Error(w, "Non Admin roles cannot access closed applications", http.StatusUnauthorized)
		return
	}

	// Fetch loan applications based on query params
	loanApplications, err := hd.Service.GetLoanapplications(ctx, "", borrowerID, status)
	if err != nil {
		log.Printf("Error fetching loan applications: %v", err)
		http.Error(w, "failed to fetch loan application", http.StatusInternalServerError)
		return
	}

	// Check if no loan applications found
	if len(loanApplications) == 0 {
		http.Error(w, "no loan application found", http.StatusNotFound)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanApplications); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// Create Loan Offer Handler
func (hd Handler) CreateLoanOfferHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming request on CreateLoanOfferHandler")

	// Extract UserID from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok || UserID == "" {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	applicationID, err := uuid.Parse(vars["application_id"])
	if err != nil {
		http.Error(w, "Invalid application_id: must be a valid UUID", http.StatusBadRequest)
		return
	}

	var payload LoanOfferPayload
	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body: JSON decoding failed", http.StatusBadRequest)
		return
	}

	// Validate request fields
	if payload.Amount <= 0 {
		http.Error(w, "Invalid amount: must be greater than zero", http.StatusBadRequest)
		return
	}
	if payload.InterestRate <= 0 {
		http.Error(w, "Invalid interest_rate: must be greater than zero", http.StatusBadRequest)
		return
	}
	if payload.Duration <= 0 {
		http.Error(w, "Invalid duration: must be a positive integer", http.StatusBadRequest)
		return
	}

	// Call service layer to create the loan offer
	loanOffer, err := hd.Service.CreateLoanOffer(ctx, UserID, payload.Amount, payload.InterestRate, payload.Duration, applicationID.String())
	if err != nil {
		http.Error(w, "Failed to create loan offer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the created loan offer
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(loanOffer)
}

// Get Loan Offers with Offer ID
func (hd Handler) GetLoanOfferByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	offerIDStr, exists := vars["offer_id"]
	if !exists || offerIDStr == "" {
		http.Error(w, "offer_id is required", http.StatusBadRequest)
		return
	}

	// Validate UUID format
	_, err := uuid.Parse(offerIDStr)
	if err != nil {
		http.Error(w, "invalid offer_id format", http.StatusBadRequest)
		return
	}

	// Fetch loan application details
	loanApplications, err := hd.Service.GetLoanOffers(ctx, offerIDStr, "", "", "")
	if err != nil {
		log.Printf("Error fetching loan offers: %v", err)
		http.Error(w, "failed to fetch loan offer", http.StatusInternalServerError)
		return
	}

	// Check if no loan applications found
	if len(loanApplications) == 0 {
		http.Error(w, "no loan offer found", http.StatusNotFound)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanApplications); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// Get Loan Offer by application ID Handler
func (hd Handler) GetOffersByApplicationIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming request on CreateLoanOfferHandler")

	// Extract UserID from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok || UserID == "" {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	applicationID, err := uuid.Parse(vars["application_id"])
	if err != nil {
		http.Error(w, "Invalid application_id: must be a valid UUID", http.StatusBadRequest)
		return
	}

	offer, err := hd.Service.GetLoanOffers(ctx, "", applicationID.String(), "", "")

	if err != nil {
		log.Println("Error Retrieving the offers from application ID ", err.Error())
		http.Error(w, "Error Retrieving the offers from application ID", http.StatusInternalServerError)
		return
	}
	// Respond with the created loan offer
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(offer)
}

// Get Loan Appliactions with borrowerID and status
func (hd Handler) GetLoanOffersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming request On GetLoanAppliactionsHandler")

	// Extract user info from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	userInfo, err := hd.Service.GetUserByID(ctx, UserID)

	if err != nil {
		http.Error(w, "Error Retriving the user from DB", http.StatusInternalServerError)
		return
	}

	// Extract query parameters
	query := r.URL.Query()
	lenderID := query.Get("user_id")
	status := query.Get("status")

	// Authorization check: Non-admin users cannot access other user's data
	if lenderID != "" && userInfo.UserRole != 3 && lenderID != userInfo.UserID {
		http.Error(w, "Non Admin Roles Cannot Access other user's info", http.StatusUnauthorized)
		return
	} else if lenderID == "" && status != "Open" && userInfo.UserRole != 3 {
		http.Error(w, "Non Admin roles cannot access closed offers", http.StatusUnauthorized)
		return
	}

	// Fetch loan offers based on query params
	loanOffers, err := hd.Service.GetLoanOffers(ctx, "", "", lenderID, status)
	if err != nil {
		log.Printf("Error fetching loan offers: %v", err)
		http.Error(w, "failed to fetch loan offers", http.StatusInternalServerError)
		return
	}

	// Check if no loan offers found
	if len(loanOffers) == 0 {
		http.Error(w, "no loan offer found", http.StatusNotFound)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanOffers); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// AcceptOfferHandler handles the acceptance of a loan offer.
func (hd Handler) AcceptOfferHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming request On AcceptOfferHandler")

	// Extract user info from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	offerID, err := uuid.Parse(vars["offer_id"])
	if err != nil {
		http.Error(w, "Invalid offer_id: must be a valid UUID", http.StatusBadRequest)
		return
	}

	loan, err := hd.Service.AcceptOffer(ctx, offerID.String(), UserID)

	if err != nil {
		log.Println("Error Accepting Offer", err.Error())
		http.Error(w, "Error Accepting Offer", http.StatusInternalServerError)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(loan); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// DisburseLoanHandler handles the disbursement of a loan.
func (hd Handler) DisburseLoanHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming request On AcceptOfferHandler")

	// Extract user info from context
	UserID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	userInfo, err := hd.Service.GetUserByID(ctx, UserID)

	if err != nil {
		http.Error(w, "Error Retriving the user from DB", http.StatusInternalServerError)
		return
	}

	// Extract application_id from path parameters
	vars := mux.Vars(r)
	offerID, err := uuid.Parse(vars["offer_id"])
	if err != nil {
		http.Error(w, "Invalid offer_id: must be a valid UUID", http.StatusBadRequest)
		return
	}

	offer, err := hd.Service.GetLoanOffers(ctx, offerID.String(), "", "", "")

	if err != nil {
		log.Println("Error Retrieving the offer from DB", err.Error())
		http.Error(w, "Error Retrieving the offer from DB", http.StatusInternalServerError)
		return
	}

	if offer[0].Status != "Accepted" {
		http.Error(w, "Offer is not accepted", http.StatusBadRequest)
		return
	}

	if offer[0].LenderID.String() != userInfo.UserID {
		http.Error(w, "You are not the lender of this offer", http.StatusBadRequest)
		return
	}

	loan, err := hd.Service.DisburseLoan(ctx, userInfo.UserID, offerID.String())

	if err != nil {
		log.Println("Error Disbursing the loan", err.Error())
		http.Error(w, "Error Disbursing the loan", http.StatusInternalServerError)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(loan); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// GetLoanDetailsHandler handles the retrieval of loan details by loan ID.
func (hd *Handler) GetLoanDetailsByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming request On GetLoanDetailsHandler")

	// Extract user info from context
	userID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	userInfo, err := hd.Service.GetUserByID(ctx, userID)

	if err != nil {
		http.Error(w, "Error Retriving the user from DB", http.StatusInternalServerError)
		return
	}

	// Extract loan_id from path parameters
	vars := mux.Vars(r)
	loanID := vars["loan_id"]

	// Validate loan ID
	if loanID == "" {
		http.Error(w, "Loan ID is required", http.StatusBadRequest)
		return
	}

	// Fetch loan details using the service
	loanDetails, err := hd.Service.GetLoanDetails(ctx, loanID, "", "", "", "", "")
	if err != nil {
		log.Println("Error retrieving loan details from DB", err.Error())
		http.Error(w, "Error retrieving loan details from DB", http.StatusInternalServerError)
		return
	}

	// Check if loan details were found
	if len(loanDetails) == 0 {
		http.Error(w, "No loan found with the provided ID", http.StatusNotFound)
		return
	}

	log.Println("userInfo", userInfo)
	if loanDetails[0].BorrowerID != userInfo.UserID && loanDetails[0].LenderID != userInfo.UserID && userInfo.UserRole != 3 {
		http.Error(w, "You are not authorized to access this loan", http.StatusUnauthorized)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanDetails[0]); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// GetLoanDetailsHandler handles the retrieval of loan details based on various parameters.
func (hd *Handler) GetLoanDetailsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming request On GetLoanDetailsHandler")

	// Extract user info from context
	userID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	userInfo, err := hd.Service.GetUserByID(ctx, userID)
	if err != nil {
		http.Error(w, "Error Retriving the user from DB", http.StatusInternalServerError)
		return
	}

	// Extract parameters from query
	query := r.URL.Query()
	offerID := query.Get("offer_id")
	applicationID := query.Get("application_id")
	borrowerID := query.Get("borrower_id")
	lenderID := query.Get("lender_id")
	status := query.Get("status")

	// Fetch loan details based on offerID or applicationID
	var loanDetails []repo.Loan

	if offerID != "" || applicationID != "" || borrowerID != "" || lenderID != "" || status != "" {
		loanDetails, err = hd.Service.GetLoanDetails(ctx, "", offerID, borrowerID, lenderID, status, applicationID)
	} else {
		http.Error(w, "Either offerID or applicationID or borrowerID or lenderID or status is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Println("Error retrieving loan details from DB", err.Error())
		http.Error(w, "Error retrieving loan details from DB", http.StatusInternalServerError)
		return
	}

	// Check if loan details were found
	if len(loanDetails) == 0 {
		http.Error(w, "No loan found with the provided parameters", http.StatusNotFound)
		return
	}

	// Check authorization based on userID and roles
	if loanDetails[0].BorrowerID != userID && loanDetails[0].LenderID != userID && userInfo.UserRole != 3 {
		http.Error(w, "You are not authorized to access this loan", http.StatusUnauthorized)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(loanDetails[0]); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// HandleCalculatePayable handles the request to calculate the total payable amount for a loan.
func (hd *Handler) CalculatePayableHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract userID from context (assuming it's set during authentication)
	userID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract loan_id from URL path parameters
	vars := mux.Vars(r)
	loanID := vars["loan_id"]

	// Validate loanID
	if loanID == "" {
		http.Error(w, "loan_id is required", http.StatusBadRequest)
		return
	}

	// Calculate total payable amount
	payableBreakdown, err := hd.Service.CalculateTotalPayable(ctx, loanID, userID)
	if err != nil {
		log.Println("Error calculating total payable amount:", err.Error())
		http.Error(w, "Error calculating total payable amount", http.StatusInternalServerError)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(payableBreakdown); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// HandleSettleLoan handles the request to settle a loan.
func (hd *Handler) SettleLoanHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract userID from context (assuming it's set during authentication)
	userID, ok := ctx.Value("UserID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract loan_id from URL path parameters
	vars := mux.Vars(r)
	loanID := vars["loan_id"]

	// Validate loanID
	if loanID == "" {
		http.Error(w, "loan_id is required", http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(loanID); err != nil {
		http.Error(w, "Invalid loan_id format", http.StatusBadRequest)
		return
	}

	// Call the SettleLoan service function
	settledLoan, err := hd.Service.SettleLoan(ctx, userID, loanID)
	if err != nil {
		log.Println("Error settling loan:", err.Error())
		http.Error(w, "Error settling loan", http.StatusInternalServerError)
		return
	}

	// Respond with JSON data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(settledLoan); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
