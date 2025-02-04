package user

import (
	"encoding/json"
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
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	walletAddress, err := hd.Service.CreateUserAccount(req)
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
	var credentials Credentials

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	response, err := hd.Service.AuthenticateUser(struct {
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
