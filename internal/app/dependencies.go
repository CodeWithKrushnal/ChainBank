package app

import (
	"database/sql"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/app/user"
	"github.com/CodeWithKrushnal/ChainBank/internal/app/wallet"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/CodeWithKrushnal/ChainBank/middleware"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Dependencies struct for dependency injection
type Dependencies struct {
	UserService       user.Service
	WalletService     wallet.Service
	MiddlewareService middleware.Service
}

// NewDependencies initializes all dependencies
func NewDependencies(db *sql.DB, ethClient *ethclient.Client) *Dependencies {
	// Initialize repositories
	userRepo := repo.NewUserRepo(db)
	walletRepo := repo.NewWalletRepo(db)
	ethRepo := ethereum.NewEthRepo(ethClient)

	// Initialize services
	userService := user.NewService(userRepo, walletRepo, ethRepo)
	walletService := wallet.NewService(userRepo, walletRepo, ethRepo)
	middlewareService := middleware.NewService(userRepo, walletRepo)

	// Return initialized dependencies
	return &Dependencies{
		UserService:       userService,
		WalletService:     walletService,
		MiddlewareService: middlewareService,
	}
}
