package user

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"net/smtp"
	"strconv"
	"time"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/CodeWithKrushnal/ChainBank/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type service struct {
	userRepo      repo.UserStorer
	walletRepo    repo.WalletStorer
	ethRepo       ethereum.EthRepo
	configDetails utils.ConfigStruct
}

// Constructor function
func NewService(ctx context.Context, userRepo repo.UserStorer, walletRepo repo.WalletStorer, ethRepo ethereum.EthRepo, configDetails utils.ConfigStruct) Service {
	return service{
		userRepo:      userRepo,
		walletRepo:    walletRepo,
		ethRepo:       ethRepo,
		configDetails: configDetails,
	}
}

// Add necesary method signature to be made accesible by service layer
type Service interface {
	CreateUserAccount(ctx context.Context, req SignupRequest) (repo.User, error)
	AuthenticateUser(ctx context.Context, credentials utils.Credentials, originIP string) (map[string]string, error)
	InsertKYCVerificationService(ctx context.Context, UserEmail, documentType, documentNumber, verificationStatus string) (string, error)
	GetAllKYCVerificationsService(ctx context.Context) ([]repo.KYCRecord, error)
	UpdateKYCVerificationStatusService(ctx context.Context, kycID, verificationStatus, verifiedBy string) error
	GetKYCDetailedInfo(ctx context.Context, kycID, userEmail string) ([]repo.KYCRecord, error)
	GetUserByID(ctx context.Context, userID string) (utils.User, error)
	SendEmail(ctx context.Context, email, subject, message string) error
	GenerateAndSendResetToken(ctx context.Context, email string) error
	ValidateResetToken(tokenString string) (jwt.MapClaims, error)
	ValidateAndResetPassword(ctx context.Context, resetToken, newPassword string) error
	GetUserByEmail(ctx context.Context, email string) (repo.User, error)
	GetRequestLogs(ctx context.Context, filter repo.RequestLogFilter) ([]repo.RequestLog, error)
	GetRequestLogStats(ctx context.Context, filter repo.RequestLogStatsFilter) (interface{}, error)
	GenerateEmailVerificationToken(ctx context.Context, email string, originIP string) (string, error)
	GetUserInfo(ctx context.Context, userID string) (utils.UserInfo, error)
	GetTransactionStats(ctx context.Context, filter repo.TransactionStatsFilter) (interface{}, error)
	GetConfig(ctx context.Context) (utils.ConfigStruct, error)
	ExecuteQuery(ctx context.Context, query string, args ...interface{}) (interface{}, error)
	ValidateEmailVerificationToken(tokenString string) (jwt.MapClaims, error)
}

// GenerateLoginToken generates a JWT token for user authentication.
func GenerateLoginToken(ctx context.Context, email string, originIP string, configDetails utils.ConfigStruct) (string, error) {
	const loginTokenExpirationHours = 24

	if configDetails.JWTSecretKey == "" {
		return "", utils.ErrEmptySecretKey
	}

	jwtSecret := []byte(configDetails.JWTSecretKey)

	// Define expiration time
	loginExpiration := time.Now().Add(time.Hour * loginTokenExpirationHours) // 24 hours

	// Create Login Token
	loginClaims := jwt.MapClaims{
		"email":  email,
		"exp":    loginExpiration.Unix(),
		"iat":    time.Now().Unix(),
		"origin": originIP,
	}
	loginToken := jwt.NewWithClaims(jwt.SigningMethodHS256, loginClaims)
	loginTokenString, err := loginToken.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf(utils.ErrorFormat, utils.ErrGeneratingToken, err)
	}

	return loginTokenString, nil
}

// GenerateResetToken generates a JWT token for password reset.
func GenerateResetToken(ctx context.Context, email string, originIP string, configDetails utils.ConfigStruct) (string, error) {
	const resetTokenExpirationMinutes = 5

	if configDetails.JWTResetSecretKey == "" {
		return "", utils.ErrEmptySecretKey
	}

	jwtResetSecret := []byte(configDetails.JWTResetSecretKey)

	// Define expiration time
	resetExpiration := time.Now().Add(time.Minute * resetTokenExpirationMinutes) // 5 minutes

	// Create Reset Token
	resetClaims := jwt.MapClaims{
		"email":  email,
		"exp":    resetExpiration.Unix(),
		"iat":    time.Now().Unix(),
		"reset":  true,
		"origin": originIP,
	}
	resetToken := jwt.NewWithClaims(jwt.SigningMethodHS256, resetClaims)
	resetTokenString, err := resetToken.SignedString(jwtResetSecret)
	if err != nil {
		return "", fmt.Errorf(utils.ErrorFormat, utils.ErrGeneratingResetToken, err)
	}

	return resetTokenString, nil
}

// PrivateKeyToHex converts an ECDSA private key to its hexadecimal string representation.
func PrivateKeyToHex(privateKey *ecdsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", fmt.Errorf(utils.ErrorFormat, utils.ErrInvalidPrivateKey, utils.ErrNilData)
	}

	privateKeyBytes := crypto.FromECDSA(privateKey) // Convert to byte slice
	if len(privateKeyBytes) == 0 {
		return "", fmt.Errorf(utils.ErrorFormat, utils.ErrInvalidPrivateKey, utils.ErrNilData)
	}

	hexString := hex.EncodeToString(privateKeyBytes) // Convert to hex string
	return hexString, nil
}

// Service functions

// CreateUserAccount creates a new user account and returns the wallet address.
func (sd service) CreateUserAccount(ctx context.Context, req SignupRequest) (repo.User, error) {

	// Convert role from string to integer
	digitRole, err := strconv.Atoi(req.Role)
	if err != nil || (digitRole != 1 && digitRole != 2) {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrInvalidRole, err)
	}

	// Check if the username or email already exists
	usernameExists, emailExists, err := sd.userRepo.UserExists(ctx, req.Username, req.Email)
	if err != nil {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrUsernameOrEmailTaken, err)
	}
	if usernameExists || emailExists {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrUsernameOrEmailTaken, err)
	}

	claims, err := sd.ValidateEmailVerificationToken(req.EmailVerificationToken)
	if err != nil {
		slog.Error(err.Error())
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrInvalidEmailVerificationToken, err)
	}
	if email, ok := claims["email"].(string); !ok || email != req.Email {
		return repo.User{}, utils.ErrInvalidEmailVerificationToken
	}

	// Hash the user's password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrPasswordHashing, err)
	}

	// Create a new wallet for the user
	walletAddress, privateKey, err := sd.ethRepo.CreateWallet(req.Password)
	if err != nil {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrWalletCreation, err)
	}

	// Convert the private key to a hexadecimal string
	privateKeyHex, err := PrivateKeyToHex(privateKey)
	if err != nil {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrInvalidPrivateKeyConversion, err)
	}

	// Preload tokens into the user's wallet
	testnetAmount := big.NewInt(1e18)
	if err := sd.ethRepo.PreloadTokens(walletAddress, testnetAmount); err != nil {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrTokenPreload, err)
	}

	// Create the user in the database
	if err := sd.userRepo.CreateUser(ctx, repo.CreateUserParams{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		FullName:      req.FullName,
		DOB:           req.DOB,
		WalletAddress: walletAddress,
		Role:          digitRole,
	}); err != nil {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrUserCreation, err)
	}

	// Retrieve the user by email to get the user ID
	user, err := sd.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrRetrievingUserID, err)
	}

	// Insert the private key into the wallet repository
	if err := sd.walletRepo.InsertPrivateKey(ctx, user.ID, walletAddress, privateKeyHex); err != nil {
		return repo.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrInsertingPrivateKey, err)
	}

	return user, nil
}

// AuthenticateUser authenticates a user based on provided credentials and returns login and reset tokens.
func (sd service) AuthenticateUser(ctx context.Context, credentials utils.Credentials, originIP string) (map[string]string, error) {
	// Retrieve user by email
	user, err := sd.userRepo.GetUserByEmail(ctx, credentials.Email)
	if err != nil {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrUserNotFound, err)
	}

	// Compare the provided password with the stored hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrInvalidCredentials, err)
	}

	// Generate login and reset tokens
	loginToken, err := GenerateLoginToken(ctx, user.Email, originIP, sd.configDetails)
	if err != nil {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrTokenGenerationFailed, err)
	}

	// Return the generated tokens
	return map[string]string{"login_token": loginToken}, nil
}

// InsertKYCVerificationService inserts a new KYC verification record.
func (sd service) InsertKYCVerificationService(ctx context.Context, userEmail, documentType, documentNumber, verificationStatus string) (string, error) {
	// Retrieve user by email
	user, err := sd.userRepo.GetUserByEmail(ctx, userEmail)
	if err != nil {
		return "", fmt.Errorf(utils.ErrorFormat, utils.ErrUserNotFound, err)
	}

	// Insert KYC verification record
	kycID, err := sd.userRepo.InsertKYCVerification(ctx, user.ID, documentType, documentNumber, verificationStatus)
	if err != nil {
		return "", fmt.Errorf(utils.ErrorFormat, utils.ErrKYCVerificationInsertion, err)
	}

	return kycID, nil
}

// GetAllKYCVerificationsService retrieves all KYC verification records.
func (sd service) GetAllKYCVerificationsService(ctx context.Context) ([]repo.KYCRecord, error) {

	// Retrieve all KYC verification records
	kycRecords, err := sd.userRepo.GetAllKYCVerifications(ctx)
	if err != nil {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrFetchKYCDetailedInfo, err)
	}

	return kycRecords, nil
}

// UpdateKYCVerificationStatusService updates the KYC verification status for a given KYC ID. It updates the verification status, verified_at timestamp, and the user who verified it.
func (sd service) UpdateKYCVerificationStatusService(ctx context.Context, kycID, verificationStatus, verifiedBy string) error {

	// Update the KYC verification status in the repository
	if err := sd.userRepo.UpdateKYCVerificationStatus(ctx, kycID, verificationStatus, verifiedBy); err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrUpdatingKYCVerificationStatus, err)
	}

	return nil
}

// GetKYCDetailedInfo retrieves KYC details based on either kyc_id or user_id. It ensures that exactly one of the parameters is provided.
func (sd service) GetKYCDetailedInfo(ctx context.Context, kycID, userEmail string) ([]repo.KYCRecord, error) {
	// Validate input parameters
	if (kycID == "" && userEmail == "") || (kycID != "" && userEmail != "") {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrInvalidInput, utils.ErrInvalidInput)
	}

	var userID string
	if userEmail != "" {
		// Retrieve user by email
		user, err := sd.userRepo.GetUserByEmail(ctx, userEmail)
		if err != nil {
			return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrUserRetrievalFailed, err)
		}
		userID = user.ID
	}

	// Retrieve KYC detailed information
	kycRecords, err := sd.userRepo.GetKYCDetailedInfo(ctx, kycID, userID)
	if err != nil {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrFetchKYCDetailedInfo, err)
	}

	return kycRecords, nil
}

// GetUserByID retrieves a user by their ID, including their email and highest role.
func (sd service) GetUserByID(ctx context.Context, userID string) (utils.User, error) {
	// Fetch detailed user information from the repository
	detailedUser, err := sd.userRepo.GetuserByID(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrFetchingUser, err)
	}

	// Fetch the highest role of the user
	role, err := sd.userRepo.GetUserHighestRole(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf(utils.ErrorFormat, utils.ErrFetchingRole, err)
	}

	// Return the user details including ID, email, and role
	return utils.User{UserID: detailedUser.ID, UserEmail: detailedUser.Email, UserRole: role}, nil
}

func (sd service) SendEmail(ctx context.Context, email, subject, message string) error {
	credentials := sd.configDetails
	slog.Info(credentials.SMTPHost)
	// Define the SMTP server configuration
	smtpHost := sd.configDetails.SMTPHost
	smtpPort := sd.configDetails.SMTPPort
	senderEmail := sd.configDetails.SenderEmail
	senderPassword := sd.configDetails.SenderPassword

	// Set up authentication
	auth := smtp.PlainAuth("", senderEmail, senderPassword, smtpHost)

	// Create the email message with HTML content
	body := message
	msg := []byte("To: " + email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=\"utf-8\"\r\n" + // Set content type to HTML
		"\r\n" +
		body + "\r\n")

	// Send the email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, []string{email}, msg)
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrSendingEmail, err)
	}

	return nil
}

func (sd service) GenerateAndSendResetToken(ctx context.Context, email string) error {
	// Generate the reset token
	resetToken, err := GenerateResetToken(ctx, email, "originIP", sd.configDetails) // Replace "originIP" with actual IP if needed
	if err != nil {
		slog.Error(utils.ErrGeneratingResetToken.Error(), "error", err)
		return fmt.Errorf(utils.ErrorFormat, utils.ErrGeneratingResetToken, err)
	}

	// Prepare the email message
	subject := "Password Reset Request"
	message := fmt.Sprintf("To reset your password, please use the following token: <strong>%s</strong>", resetToken)

	// Send the email
	if err := sd.SendEmail(ctx, email, subject, message); err != nil {
		slog.Error(utils.ErrSendingEmail.Error(), "error", err)
		return fmt.Errorf("%s: %w", utils.ErrSendingEmail, err)
	}

	return nil
}

// ValidateResetToken validates a JWT reset token and returns its claims.
func (sd service) ValidateResetToken(tokenString string) (jwt.MapClaims, error) {
	jwtResetSecret := []byte(sd.configDetails.JWTResetSecretKey)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtResetSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if reset, ok := claims["reset"].(bool); !ok || !reset {
			return nil, fmt.Errorf("invalid reset token")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// ValidateAndResetPassword validates the reset token and updates the user's password.
func (sd service) ValidateAndResetPassword(ctx context.Context, resetToken, newPassword string) error {
	// Validate the reset token
	claims, err := sd.ValidateResetToken(resetToken) // Implement this function to validate the token
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrInvalidResetToken, err)
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrHashingPassword, err)
	}

	user, err := sd.GetUserByEmail(ctx, claims["email"].(string))
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrRetrievingUser, err)
	}

	// Update the user's password in the repository
	if err := sd.userRepo.UpdateUserPasswordHash(ctx, user.ID, string(hashedPassword)); err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrUpdatingPassword, err)
	}

	return nil
}

// GetUserByEmail retrieves a user by their email.
func (sd service) GetUserByEmail(ctx context.Context, email string) (repo.User, error) {
	return sd.userRepo.GetUserByEmail(ctx, email)
}

// GenerateEmailVerificationToken generates a JWT token for email verification.
func (sd service) GenerateEmailVerificationToken(ctx context.Context, email string, originIP string) (string, error) {
	const emailVerificationTokenExpirationMinutes = 5

	if sd.configDetails.JWTSecretKey == "" {
		return "", utils.ErrEmptySecretKey
	}

	jwtSecret := []byte(sd.configDetails.JWTSecretKey)


	// Define expiration time
	emailVerificationExpiration := time.Now().Add(time.Minute * emailVerificationTokenExpirationMinutes) // 5 Minutes

	// Create Email Verification Token
	emailVerificationClaims := jwt.MapClaims{
		"email":  email,
		"exp":    emailVerificationExpiration.Unix(),
		"iat":    time.Now().Unix(),
		"origin": originIP,
		"verify": true,
	}
	emailVerificationToken := jwt.NewWithClaims(jwt.SigningMethodHS256, emailVerificationClaims)
	emailVerificationTokenString, err := emailVerificationToken.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf(utils.ErrorFormat, utils.ErrGeneratingToken, err)
	}

	return emailVerificationTokenString, nil
}

// ValidateEmailVerificationToken validates the email verification token.
func (sd service) ValidateEmailVerificationToken(tokenString string) (jwt.MapClaims, error) {
	jwtSecret := []byte(sd.configDetails.JWTSecretKey)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if verify, ok := claims["verify"].(bool); !ok || !verify {
			return nil, fmt.Errorf("invalid verification token")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func (sd service) GetRequestLogs(ctx context.Context, filter repo.RequestLogFilter) ([]repo.RequestLog, error) {
	return sd.userRepo.GetRequestLogs(ctx, filter)
}

func (sd service) GetRequestLogStats(ctx context.Context, filter repo.RequestLogStatsFilter) (interface{}, error) {
	return sd.userRepo.GetRequestLogStats(ctx, filter)
}

func (sd service) GetUserInfo(ctx context.Context, userID string) (utils.UserInfo, error) {
	return sd.userRepo.GetUserInfo(ctx, userID)
}

func (sd service) GetTransactionStats(ctx context.Context, filter repo.TransactionStatsFilter) (interface{}, error) {
	return sd.userRepo.GetTransactionStats(ctx, filter)
}

func (sd service) GetConfig(ctx context.Context) (utils.ConfigStruct, error) {
	return sd.configDetails, nil
}

func (sd service) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	return sd.userRepo.ExecuteQuery(ctx, query, args...)
}