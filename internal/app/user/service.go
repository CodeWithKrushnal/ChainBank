package user

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/config"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type service struct {
	userRepo   repo.UserStorer
	walletRepo repo.WalletStorer
	ethRepo    ethereum.EthRepo
}

// Constructor function
func NewService(ctx context.Context, userRepo repo.UserStorer, walletRepo repo.WalletStorer, ethRepo ethereum.EthRepo) Service {
	return service{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		ethRepo:    ethRepo,
	}
}

// Add necesary method signature to be made accesible by service layer
type Service interface {
	CreateUserAccount(ctx context.Context, req SignupRequest) (string, error)
	AuthenticateUser(ctx context.Context, credentials struct{ Email, Password string }) (map[string]string, error)
	InsertKYCVerificationService(ctx context.Context, UserEmail, documentType, documentNumber, verificationStatus string) (string, error)
	GetAllKYCVerificationsService(ctx context.Context) ([]map[string]interface{}, error)
	UpdateKYCVerificationStatusService(ctx context.Context, kycID, verificationStatus, verifiedBy string) error
	GetKYCDetailedInfo(ctx context.Context, kycID, userEmail string) ([]map[string]interface{}, error)
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

// Service functions
func (sd service) CreateUserAccount(ctx context.Context, req SignupRequest) (string, error) {
	digitRole, err := strconv.Atoi(req.Role)
	if err != nil || (digitRole != 1 && digitRole != 2) {
		return "", err
	}

	usernameExists, emailExists, err := sd.userRepo.UserExists(ctx, req.Username, req.Email)
	if err != nil {
		return "", err
	}
	if usernameExists || emailExists {
		return "Username or email already taken", nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	walletAddress, privateKey, err := sd.ethRepo.CreateWallet(req.Password)
	if err != nil {
		return "", err
	}

	privateKeyHex := PrivateKeyToHex(privateKey)
	testnetAmount := big.NewInt(1e18)
	if err := sd.ethRepo.PreloadTokens(walletAddress, testnetAmount); err != nil {
		return "", err
	}

	if err := sd.userRepo.CreateUser(ctx, req.Username, req.Email, string(hashedPassword), req.FullName, req.DOB, walletAddress, digitRole); err != nil {
		return "", err
	}

	user, err := sd.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		log.Println("Error Retrieving User ID: ", err.Error())
	}

	sd.walletRepo.InsertPrivateKey(user.ID, walletAddress, privateKeyHex)

	return walletAddress, nil
}

func (sd service) AuthenticateUser(ctx context.Context, credentials struct{ Email, Password string }) (map[string]string, error) {
	user, err := sd.userRepo.GetUserByEmail(ctx, credentials.Email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		return nil, err
	}

	loginToken, resetToken, err := GenerateTokens(user.Email)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"login_token": loginToken,
		"reset_token": resetToken,
	}, nil
}

// InsertKYCVerificationService inserts a new KYC verification record.
func (sd service) InsertKYCVerificationService(ctx context.Context, UserEmail, documentType, documentNumber, verificationStatus string) (string, error) {
	user, err := sd.userRepo.GetUserByEmail(ctx, UserEmail)
	if err != nil {
		return "", err
	}
	return sd.userRepo.InsertKYCVerification(ctx, user.ID, documentType, documentNumber, verificationStatus)
}

// GetAllKYCVerificationsService retrieves all KYC verification records.
func (sd service) GetAllKYCVerificationsService(ctx context.Context) ([]map[string]interface{}, error) {
	return sd.userRepo.GetAllKYCVerifications(ctx)
}

// UpdateKYCVerificationStatusService updates verification_status, verified_at, and verified_by.
func (sd service) UpdateKYCVerificationStatusService(ctx context.Context, kycID, verificationStatus, verifiedBy string) error {
	return sd.userRepo.UpdateKYCVerificationStatus(ctx, kycID, verificationStatus, verifiedBy)
}

// GetKYCDetailedInfo retrieves KYC details based on either kyc_id or user_id.
func (sd service) GetKYCDetailedInfo(ctx context.Context, kycID, userEmail string) ([]map[string]interface{}, error) {
	if (kycID == "" && userEmail == "") || (kycID != "" && userEmail != "") {
		return nil, fmt.Errorf("exactly one of kycID or userEmail must be provided")
	}

	var userID string
	if userEmail != "" {
		user, err := sd.userRepo.GetUserByEmail(ctx, userEmail)
		if err != nil {
			log.Println("Error Retrieving user id related to provided email", err)
			return nil, err
		}
		userID = user.ID
	}

	return sd.userRepo.GetKYCDetailedInfo(ctx, kycID, userID)
}
