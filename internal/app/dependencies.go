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
	"github.com/CodeWithKrushnal/ChainBank/utils"
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
// NewDependencies initializes all the necessary services and repositories for the application.
func NewDependencies(ctx context.Context, db *sql.DB, ethClient *ethclient.Client, configDetails utils.ConfigStruct) (*Dependencies, error) {
	// Initialize repositories
	userRepo := repo.NewUserRepo(db, configDetails)
	walletRepo := repo.NewWalletRepo(db, configDetails)
	loanRepo := repo.NewLoanRepo(db, configDetails)
	ethRepo := ethereum.NewEthRepo(ethClient, configDetails)

	// Initialize services
	userService := user.NewService(ctx, userRepo, walletRepo, ethRepo, configDetails)
	walletService := wallet.NewService(ctx, userRepo, walletRepo, ethRepo, configDetails)
	loanService := loan.NewService(ctx, userRepo, walletRepo, loanRepo, ethRepo, configDetails)
	middlewareService := middleware.NewService(ctx, userRepo, walletRepo, configDetails)

	// Check if services are initialized correctly
	if userService == nil || walletService == nil || loanService == nil || middlewareService == nil {
		return nil, utils.ErrServiceInit // Propagate error if any service fails to initialize
	}

	// Return initialized dependencies
	return &Dependencies{
		UserService:       userService,
		WalletService:     walletService,
		LoanService:       loanService,
		MiddlewareService: middlewareService,
	}, nil
}
