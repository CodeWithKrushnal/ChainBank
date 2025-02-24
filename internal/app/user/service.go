package user

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/config"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/CodeWithKrushnal/ChainBank/utils"
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
	AuthenticateUser(ctx context.Context, credentials AuthCredentials, originIP string) (map[string]string, error)
	InsertKYCVerificationService(ctx context.Context, UserEmail, documentType, documentNumber, verificationStatus string) (string, error)
	GetAllKYCVerificationsService(ctx context.Context) ([]repo.KYCRecord, error)
	UpdateKYCVerificationStatusService(ctx context.Context, kycID, verificationStatus, verifiedBy string) error
	GetKYCDetailedInfo(ctx context.Context, kycID, userEmail string) ([]repo.KYCRecord, error)
	GetUserByID(ctx context.Context, userID string) (utils.User, error)
}

type AuthCredentials struct {
	Email    string
	Password string
}

// GenerateTokens generates JWT tokens for user authentication and password reset.
func GenerateTokens(ctx context.Context, email string, originIP string) (string, string, error) {
	const (
		loginTokenExpirationHours = 24
		resetTokenExpirationHours = 1
	)

	JWT_SECRET := []byte(config.ConfigDetails.JWTSecretKey)
	JWT_RESET_SECRET := []byte(config.ConfigDetails.JWTResetSecretKey)

	// Define expiration times
	loginExpiration := time.Now().Add(time.Hour * loginTokenExpirationHours) // 24 hours
	resetExpiration := time.Now().Add(time.Hour * resetTokenExpirationHours) // 1 hour

	// Create Login Token
	loginClaims := jwt.MapClaims{
		"email":  email,
		"exp":    loginExpiration.Unix(),
		"iat":    time.Now().Unix(),
		"origin": originIP,
	}
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	loginTokenString, err := loginToken.SignedString(JWT_SECRET)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", utils.ErrGeneratingToken, err)
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
		return "", "", fmt.Errorf("%s: %w", utils.ErrGeneratingResetToken, err)
	}

	return loginTokenString, resetTokenString, nil
}

// PrivateKeyToHex converts an ECDSA private key to its hexadecimal string representation.
func PrivateKeyToHex(privateKey *ecdsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", fmt.Errorf("%s: %w", utils.ErrInvalidPrivateKey, utils.ErrNilData)
	}

	privateKeyBytes := crypto.FromECDSA(privateKey) // Convert to byte slice
	if len(privateKeyBytes) == 0 {
		return "", fmt.Errorf("%s: %w", utils.ErrInvalidPrivateKey, utils.ErrNilData)
	}

	hexString := hex.EncodeToString(privateKeyBytes) // Convert to hex string
	return hexString, nil
}

// Service functions

// CreateUserAccount creates a new user account and returns the wallet address.
func (sd service) CreateUserAccount(ctx context.Context, req SignupRequest) (string, error) {

	// Convert role from string to integer
	digitRole, err := strconv.Atoi(req.Role)
	if err != nil || (digitRole != 1 && digitRole != 2) {
		return "", fmt.Errorf("%s: %w", utils.ErrInvalidRole, err)
	}

	// Check if the username or email already exists
	usernameExists, emailExists, err := sd.userRepo.UserExists(ctx, req.Username, req.Email)
	if err != nil {
		return "", err
	}
	if usernameExists || emailExists {
		return "", fmt.Errorf("%s: %w", utils.ErrUsernameOrEmailTaken, err)
	}

	// Hash the user's password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrPasswordHashing, err)
	}

	// Create a new wallet for the user
	walletAddress, privateKey, err := sd.ethRepo.CreateWallet(req.Password)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrWalletCreation, err)
	}

	// Convert the private key to a hexadecimal string
	privateKeyHex, err := PrivateKeyToHex(privateKey)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrInvalidPrivateKeyConversion, err)
	}

	// Preload tokens into the user's wallet
	testnetAmount := big.NewInt(1e18)
	if err := sd.ethRepo.PreloadTokens(walletAddress, testnetAmount); err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrTokenPreload, err)
	}

	// Create the user in the database
	if err := sd.userRepo.CreateUser(ctx, req.Username, req.Email, string(hashedPassword), req.FullName, req.DOB, walletAddress, digitRole); err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrUserCreation, err)
	}

	// Retrieve the user by email to get the user ID
	user, err := sd.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrRetrievingUserID, err)
	}

	// Insert the private key into the wallet repository
	if err := sd.walletRepo.InsertPrivateKey(ctx, user.ID, walletAddress, privateKeyHex); err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrInsertingPrivateKey, err)
	}

	return walletAddress, nil
}

// AuthenticateUser authenticates a user based on provided credentials and returns login and reset tokens.
func (sd service) AuthenticateUser(ctx context.Context, credentials AuthCredentials, originIP string) (map[string]string, error) {
	// Retrieve user by email
	user, err := sd.userRepo.GetUserByEmail(ctx, credentials.Email)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrUserNotFound, err)
	}

	// Compare the provided password with the stored hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrInvalidCredentials, err)
	}

	// Generate login and reset tokens
	loginToken, resetToken, err := GenerateTokens(ctx, user.Email, originIP)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrTokenGenerationFailed, err)
	}

	// Return the generated tokens
	return map[string]string{
		"login_token": loginToken,
		"reset_token": resetToken,
	}, nil
}

// InsertKYCVerificationService inserts a new KYC verification record.
func (sd service) InsertKYCVerificationService(ctx context.Context, userEmail, documentType, documentNumber, verificationStatus string) (string, error) {
	// Retrieve user by email
	user, err := sd.userRepo.GetUserByEmail(ctx, userEmail)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrUserNotFound, err)
	}

	// Insert KYC verification record
	kycID, err := sd.userRepo.InsertKYCVerification(ctx, user.ID, documentType, documentNumber, verificationStatus)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrKYCVerificationInsertion, err)
	}

	return kycID, nil
}

// GetAllKYCVerificationsService retrieves all KYC verification records.
func (sd service) GetAllKYCVerificationsService(ctx context.Context) ([]repo.KYCRecord, error) {
	const errFetchingKYCRecords = "failed to fetch KYC verification records"

	// Retrieve all KYC verification records
	kycRecords, err := sd.userRepo.GetAllKYCVerifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errFetchingKYCRecords, err)
	}

	return kycRecords, nil
}

// UpdateKYCVerificationStatusService updates the KYC verification status for a given KYC ID. It updates the verification status, verified_at timestamp, and the user who verified it.
func (sd service) UpdateKYCVerificationStatusService(ctx context.Context, kycID, verificationStatus, verifiedBy string) error {

	// Update the KYC verification status in the repository
	if err := sd.userRepo.UpdateKYCVerificationStatus(ctx, kycID, verificationStatus, verifiedBy); err != nil {
		return fmt.Errorf("%s: %w", utils.ErrUpdatingKYCVerificationStatus, err)
	}

	return nil
}

// GetKYCDetailedInfo retrieves KYC details based on either kyc_id or user_id. It ensures that exactly one of the parameters is provided.
func (sd service) GetKYCDetailedInfo(ctx context.Context, kycID, userEmail string) ([]repo.KYCRecord, error) {
	// Validate input parameters
	if (kycID == "" && userEmail == "") || (kycID != "" && userEmail != "") {
		return nil, fmt.Errorf("%s: %w", utils.ErrInvalidInput, utils.ErrInvalidInput)
	}

	var userID string
	if userEmail != "" {
		// Retrieve user by email
		user, err := sd.userRepo.GetUserByEmail(ctx, userEmail)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", utils.ErrUserRetrievalFailed, err)
		}
		userID = user.ID
	}

	// Retrieve KYC detailed information
	return sd.userRepo.GetKYCDetailedInfo(ctx, kycID, userID)
}

// GetUserByID retrieves a user by their ID, including their email and highest role.
func (sd service) GetUserByID(ctx context.Context, userID string) (utils.User, error) {
	// Fetch detailed user information from the repository
	detailedUser, err := sd.userRepo.GetuserByID(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf("%s: %w", utils.ErrFetchingUser, err)
	}

	// Fetch the highest role of the user
	role, err := sd.userRepo.GetUserHighestRole(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf("%s: %w", utils.ErrFetchingRole, err)
	}

	// Return the user details including ID, email, and role
	return utils.User{UserID: detailedUser.ID, UserEmail: detailedUser.Email, UserRole: role}, nil
}