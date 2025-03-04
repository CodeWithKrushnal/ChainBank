package middleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/CodeWithKrushnal/ChainBank/utils"
	"github.com/golang-jwt/jwt/v5"
)

type service struct {
	userRepo   repo.UserStorer
	walletRepo repo.WalletStorer
	configDetails utils.ConfigStruct
}

type Service interface {
	getUserByEmail(ctx context.Context, email string) (repo.User, error)
	getUserHighestRole(ctx context.Context, userID string) (int, error)
	updateLastLogin(ctx context.Context, userID string) error
	CreateRequestLog(ctx context.Context, requestID, userID, endpoint, httpMethod string, requestPayload interface{}, ipAddress string) (string, error)
	UpdateRequestLog(ctx context.Context, requestID string, responseStatus, responseTimeMs int) error
	ValidateJWT(tokenString string, originIP string) (string, error)
}

func NewService(ctx context.Context, userRepo repo.UserStorer, walletRepo repo.WalletStorer, configDetails utils.ConfigStruct) Service {
	return service{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		configDetails: configDetails,
	}
}

func (authServiceDep service) getUserByEmail(ctx context.Context, email string) (repo.User, error) {
	return authServiceDep.userRepo.GetUserByEmail(ctx, email)
}

func (authServiceDep service) getUserHighestRole(ctx context.Context, userID string) (int, error) {
	return authServiceDep.userRepo.GetUserHighestRole(ctx, userID)
}

func (authServiceDep service) updateLastLogin(ctx context.Context, userID string) error {
	return authServiceDep.userRepo.UpdateLastLogin(ctx, userID)
}

func (authServiceDep service) CreateRequestLog(ctx context.Context, requestID, userID, endpoint, httpMethod string, requestPayload interface{}, ipAddress string) (string, error) {
	return authServiceDep.userRepo.CreateRequestLog(ctx, requestID, userID, endpoint, httpMethod, requestPayload, ipAddress)
}

func (authServiceDep service) UpdateRequestLog(ctx context.Context, requestID string, responseStatus, responseTimeMs int) error {
	return authServiceDep.userRepo.UpdateRequestLog(ctx, requestID, responseStatus, responseTimeMs)
}

func (authServiceDep service)ValidateJWT(tokenString string, originIP string) (string, error) {

	JWT_SECRET := []byte(authServiceDep.configDetails.JWTSecretKey)

	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return JWT_SECRET, nil
	})

	if err != nil {
		return "", err
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userEmail, ok := claims["email"].(string)
		if !ok {
			return "", fmt.Errorf("invalid token claims")
		}

		if claims["origin"].(string) != originIP {
			return "", fmt.Errorf("Token is invalid : invalid Token Origin")
		}
		return userEmail, nil
	}

	return "", errors.New("invalid token")
}