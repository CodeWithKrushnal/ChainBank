package user

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

type Handler struct {
	Service Service
}

// Constructor function
func NewHandler(service Service) *Handler {
	return &Handler{Service: service}
}

// Handlers
func (hd *Handler) SignupHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	walletAddress, err := hd.Service.CreateUserAccount(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := SignupResponse{
		Message:       "User registered successfully",
		WalletAddress: walletAddress,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (hd *Handler) SignInHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var credentials Credentials

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	response, err := hd.Service.AuthenticateUser(ctx, struct {
		Email    string
		Password string
	}(credentials))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (hd Handler) RequestKYCHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming Request On RequestKYCHandler")

	userInfo, ok := ctx.Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	var req struct {
		DocumentType   string `json:"document_type"`
		DocumentNumber string `json:"document_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	log.Println("received in handler", req)
	kycID, err := hd.Service.InsertKYCVerificationService(ctx, userInfo.UserEmail, req.DocumentType, req.DocumentNumber, "Pending")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"kyc_id": kycID})
}

// GetKYCRequestsHandler retrieves all KYC verification records.
func (hd Handler) GetKYCRequestsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming Request On GetKYCRequestsHandler")

	userInfo, ok := ctx.Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	if userInfo.UserRole != 3 {
		http.Error(w, "Unauthorized: Non Admin Role Cannot Access", http.StatusUnauthorized)
		return
	}

	kycRecords, err := hd.Service.GetAllKYCVerificationsService(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if kycRecords == nil {
		w.Write([]byte("No New KYC Requests"))
	} else {
		json.NewEncoder(w).Encode(kycRecords)
	}
}

// KYCRequestActionHandler updates KYC verification status.
func (hd Handler) KYCRequestActionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Println("Incoming Request On KYCRequestActionHandler")

	userInfo, ok := ctx.Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	if userInfo.UserRole != 3 {
		http.Error(w, "Unauthorized: Non Admin Role Cannot Access", http.StatusUnauthorized)
		return
	}

	var req struct {
		KYCID              string `json:"kyc_id"`
		VerificationStatus string `json:"verification_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("error in decoding KYC status Update request:", err.Error())
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.KYCID == "" || req.VerificationStatus == "" {
		http.Error(w, "Plese Supply the KYCID and VerificationStatus in request body properly", http.StatusBadRequest)
	}

	var verificationStatus string
	if req.VerificationStatus == "1" {
		verificationStatus = "Verified"
	} else if req.VerificationStatus == "2" {
		verificationStatus = "Unverified"
	} else {
		http.Error(w, "Invalid verification_status please use 1 for Verified and 2 for Unverified ", http.StatusBadRequest)
	}

	err := hd.Service.UpdateKYCVerificationStatusService(ctx, req.KYCID, verificationStatus, userInfo.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "KYC status updated successfully"})
}

// GetKYCDetailedInfoHandler handles requests to retrieve KYC details based on kyc_id or user_id.
func (hd *Handler) GetKYCDetailedInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	kycID := r.URL.Query().Get("kyc_id")
	userEmail := r.URL.Query().Get("user_email")

	userInfo, ok := ctx.Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		http.Error(w, "Unauthorized: user info not found in context", http.StatusUnauthorized)
		return
	}

	if userInfo.UserRole != 3 {
		kycID = ""
		userEmail = userInfo.UserEmail
	}

	log.Println("userid", userEmail)
	if (kycID == "" && userEmail == "") || (kycID != "" && userEmail != "") {
		http.Error(w, "Exactly one of kyc_id or user_id must be provided", http.StatusBadRequest)
		return
	}
	log.Println("using info", kycID, userEmail)
	kycDetails, err := hd.Service.GetKYCDetailedInfo(ctx, kycID, userEmail)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching KYC details: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kycDetails)
}
