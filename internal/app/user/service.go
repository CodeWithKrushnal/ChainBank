package user

import (
	"crypto/ecdsa"
	"encoding/hex"
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
func NewService(userRepo repo.UserStorer, walletRepo repo.WalletStorer, ethRepo ethereum.EthRepo) Service {
	return service{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		ethRepo:    ethRepo,
	}
}

// Add necesary method signature to be made accesible by service layer
type Service interface {
	CreateUserAccount(req SignupRequest) (string, error)
	AuthenticateUser(credentials struct{ Email, Password string }) (map[string]string, error)
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
func (sd service) CreateUserAccount(req SignupRequest) (string, error) {
	digitRole, err := strconv.Atoi(req.Role)
	if err != nil || (digitRole != 1 && digitRole != 2) {
		return "", err
	}

	usernameExists, emailExists, err := sd.userRepo.UserExists(req.Username, req.Email)
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

	if err := sd.userRepo.CreateUser(req.Username, req.Email, string(hashedPassword), req.FullName, req.DOB, walletAddress, digitRole); err != nil {
		return "", err
	}

	user, err := sd.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		log.Println("Error Retrieving User ID: ", err.Error())
	}

	sd.walletRepo.InsertPrivateKey(user.ID, walletAddress, privateKeyHex)

	return walletAddress, nil
}

func (sd service) AuthenticateUser(credentials struct{ Email, Password string }) (map[string]string, error) {
	user, err := sd.userRepo.GetUserByEmail(credentials.Email)
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
