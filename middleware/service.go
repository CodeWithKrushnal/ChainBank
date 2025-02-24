package middleware

import (
	"context"

	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
)

type service struct {
	userRepo   repo.UserStorer
	walletRepo repo.WalletStorer
}

type Service interface {
	getUserByEmail(ctx context.Context, email string) (repo.User, error)
	getUserHighestRole(ctx context.Context, userID string) (int, error)
	updateLastLogin(ctx context.Context, userID string) error
	CreateRequestLog(ctx context.Context, requestID, userID, endpoint, httpMethod string, requestPayload interface{}, ipAddress string) (string, error)
	UpdateRequestLog(ctx context.Context, requestID string, responseStatus, responseTimeMs int) error
}

func NewService(ctx context.Context, userRepo repo.UserStorer, walletRepo repo.WalletStorer) Service {
	return service{
		userRepo:   userRepo,
		walletRepo: walletRepo,
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
