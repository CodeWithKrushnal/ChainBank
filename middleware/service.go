package middleware

import (
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
)

type service struct {
	userRepo   repo.UserStorer
	walletRepo repo.WalletStorer
}

type Service interface {
	getUserByEmail(email string) (repo.User, error)
	getUserHighestRole(userID string) (int, error)
	updateLastLogin(userID string) error
}

func NewService(userRepo repo.UserStorer, walletRepo repo.WalletStorer) Service {
	return service{
		userRepo:   userRepo,
		walletRepo: walletRepo,
	}
}

func (authServiceDep service) getUserByEmail(email string) (repo.User, error) {
	return authServiceDep.userRepo.GetUserByEmail(email)
}

func (authServiceDep service) getUserHighestRole(userID string) (int, error) {
	return authServiceDep.userRepo.GetUserHighestRole(userID)
}

func (authServiceDep service) updateLastLogin(userID string) error {
	return authServiceDep.userRepo.UpdateLastLogin(userID)
}
