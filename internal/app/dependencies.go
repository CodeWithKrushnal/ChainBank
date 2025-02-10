package app

import (
	"context"
	"database/sql"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/app/loan"
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
	LoanService       loan.Service
	MiddlewareService middleware.Service
}

// NewDependencies initializes all dependencies
func NewDependencies(ctx context.Context, db *sql.DB, ethClient *ethclient.Client) *Dependencies {
	// Initialize repositories
	userRepo := repo.NewUserRepo(db)
	walletRepo := repo.NewWalletRepo(db)
	loanRepo := repo.NewLoanRepo(db)
	ethRepo := ethereum.NewEthRepo(ethClient)

	// Initialize services
	userService := user.NewService(ctx, userRepo, walletRepo, ethRepo)
	walletService := wallet.NewService(ctx, userRepo, walletRepo, ethRepo)
	loanService := loan.NewService(ctx, userRepo, walletRepo, loanRepo, ethRepo)
	middlewareService := middleware.NewService(ctx, userRepo, walletRepo)

	// Return initialized dependencies
	return &Dependencies{
		UserService:       userService,
		WalletService:     walletService,
		LoanService:       loanService,
		MiddlewareService: middlewareService,
	}
}
