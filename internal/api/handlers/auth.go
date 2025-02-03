package handlers

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"math/big"
	"net/http"

	"crypto/ecdsa"
	"encoding/hex"

	"github.com/CodeWithKrushnal/ChainBank/internal/api/config"
	"github.com/CodeWithKrushnal/ChainBank/internal/blockchain/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/repository/postgres"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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

// SignupHandler handles user signup
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	//Check for valid user role input
	digitRole, err := strconv.Atoi(req.Role)
	log.Print("role", digitRole)
	if err != nil && digitRole != 1 && digitRole != 2 {
		http.Error(w, "Invalid Role Entered", http.StatusBadRequest)
		return
	}

	log.Printf("Cheking Username and Email Availability : username: %v, email: %v", req.Username, req.Password)
	usernameAlreadyInExistance, emailAlreadyInExistance, err := postgres.UserExists(req.Username, req.Email)

	if err != nil {
		http.Error(w, "Error Checking the user Existance status", http.StatusInternalServerError)
		return
	}

	if usernameAlreadyInExistance && emailAlreadyInExistance {
		http.Error(w, "Username and email aleady Taken Kindly use a different username and email", http.StatusBadRequest)
		return
	} else if usernameAlreadyInExistance {
		http.Error(w, "Username aleady Taken Kindly use a different username", http.StatusBadRequest)
		return
	} else if emailAlreadyInExistance {
		http.Error(w, "Email aleady Taken Kindly use a different email", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	// Create an Ethereum wallet
	walletAddress, privateKey, err := ethereum.CreateWallet(req.Password)
	if err != nil {
		http.Error(w, "Error creating Ethereum wallet", http.StatusInternalServerError)
		return
	}

	//Convert private key to hex format
	privateKeyHex := PrivateKeyToHex(privateKey)

	log.Println("privateKeyHex", privateKeyHex)

	// Preload tokens to the wallet
	testnetAmount := big.NewInt(1e18) // 1 ETH in wei
	if err := ethereum.PreloadTokens(walletAddress, testnetAmount); err != nil {
		http.Error(w, "Error preloading tokens to wallet", http.StatusInternalServerError)
		return
	}

	// Save user to the database
	if err := postgres.CreateUser(req.Username, req.Email, string(hashedPassword), req.FullName, req.DOB, walletAddress, digitRole); err != nil {
		http.Error(w, "Error saving user to database", http.StatusInternalServerError)
		return
	}
	log.Printf("Saved User %v to database", req.Email)

	user, err := postgres.GetUserByEmail(req.Email)
	if err != nil {
		log.Println("Error Retriving User ID in Signup Handler : ", err.Error())
	}

	postgres.InsertPrivateKey(user.ID, walletAddress, privateKeyHex)

	// Respond to the client
	resp := SignupResponse{
		Message:       "User registered successfully",
		WalletAddress: walletAddress,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func SignInHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Fetch user from DB (Replace with actual DB query)
	user, err := postgres.GetUserByEmail(credentials.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWTs
	loginToken, resetToken, err := GenerateTokens(user.Email)
	log.Print(err)
	if err != nil {
		http.Error(w, "Error generating tokens", http.StatusInternalServerError)
		return
	}

	// Return tokens
	response := map[string]string{
		"login_token": loginToken,
		"reset_token": resetToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GenerateTokens(email string) (string, string, error) {

	JWT_SECRET := []byte(config.ConfigDetails.JWTSecretKey)
	JWT_RESET_SECRET := []byte(config.ConfigDetails.JWTResetSecretKey)

	// Define expiration times
	loginExpiration := time.Now().Add(time.Hour * 24) // 24 hours
	resetExpiration := time.Now().Add(time.Hour * 1)  // 1 hour

	// Create Login Token
	loginClaims := jwt.MapClaims{
		"email": email,
		"exp":   loginExpiration.Unix(),
		"iat":   time.Now().Unix(),
	}
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	loginTokenString, err := loginToken.SignedString(JWT_SECRET)
	if err != nil {
		return "", "", err
	}

	// Create Reset Token
	resetClaims := jwt.MapClaims{
		"email": email,
		"exp":   resetExpiration.Unix(),
		"iat":   time.Now().Unix(),
		"reset": true,
	}
	resetToken := jwt.NewWithClaims(jwt.SigningMethodHS256, resetClaims)
	resetTokenString, err := resetToken.SignedString(JWT_RESET_SECRET)
	if err != nil {
		return "", "", err
	}

	return loginTokenString, resetTokenString, nil
}

func PrivateKeyToHex(privateKey *ecdsa.PrivateKey) string {
	privateKeyBytes := crypto.FromECDSA(privateKey) // Convert to byte slice
	return hex.EncodeToString(privateKeyBytes)      // Convert to hex string
}
