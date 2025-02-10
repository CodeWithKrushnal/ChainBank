package loan

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type Handler struct {
	Service Service
}

// Constructor function
func NewHandler(service Service) *Handler {
	return &Handler{Service: service}
}

// CreateLoanOfferRequest represents the loan offer request body
type CreateLoanOfferRequest struct {
	Amount       string `json:"amount"`
	InterestRate string `json:"interest_rate"`
	TermMonths   string `json:"term_months"`
}

// CreateLoanOfferResponse represents the response for a loan offer
type CreateLoanOfferResponse struct {
	OfferID string `json:"offer_id"`
	Message string `json:"message"`
}

// CreateLoanOfferHandler handles loan offer creation
func (hd *Handler) CreateLoanOfferHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CreateLoanOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("error is", err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	interestRate, err := strconv.ParseFloat(req.InterestRate, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	termMonths, err := strconv.Atoi(req.TermMonths)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userInfo, ok := ctx.Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	offerID, err := hd.Service.CreateLoanOffer(ctx, userInfo.UserID, amount, interestRate, termMonths)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := CreateLoanOfferResponse{
		OfferID: offerID,
		Message: "Loan offer created successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// AcceptLoanOfferRequest represents the request body for accepting a loan offer
type AcceptLoanOfferRequest struct {
	OfferID string `json:"offer_id"`
}

// AcceptLoanOfferHandler handles accepting a loan offer
func (hd *Handler) AcceptLoanOfferHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userInfo, ok := ctx.Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	var req AcceptLoanOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := hd.Service.AcceptLoanOffer(ctx, req.OfferID, userInfo.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Loan offer accepted successfully"})
}

// RepayLoanRequest represents the request body for loan repayment
type RepayLoanRequest struct {
	LoanID string `json:"loan_id"`
	Amount string `json:"amount"`
}

// RepayLoanHandler handles loan repayment
func (hd *Handler) RepayLoanHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req RepayLoanRequest

	userInfo, ok := ctx.Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("error:",err.Error())
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(req.Amount, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := hd.Service.RepayLoan(ctx, userInfo.UserID, req.LoanID, amount); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Loan repayment successful"})
}
