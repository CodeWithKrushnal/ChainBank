package app

import (
	"context"
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/loan"
	"github.com/CodeWithKrushnal/ChainBank/internal/app/user"
	"github.com/CodeWithKrushnal/ChainBank/internal/app/wallet"
	"github.com/CodeWithKrushnal/ChainBank/middleware"
	"github.com/gorilla/mux"
)

func SetupRoutes(ctx context.Context, deps *Dependencies) *mux.Router {
	router := mux.NewRouter()
	// Inject dependencies into handlers
	userHandler := user.NewHandler(deps.UserService)
	walletHandler := wallet.NewHandler(deps.WalletService)
	loanHandler := loan.NewHandler(deps.LoanService)
	middlewareHandler := middleware.NewHandler(deps.MiddlewareService)

	//Signup Endpoint
	router.HandleFunc("/signup", userHandler.SignupHandler).Methods(http.MethodPost)
	//SignIn Endpoint
	router.HandleFunc("/signin", userHandler.SignInHandler).Methods(http.MethodPost)

	// Protected routes (Require authentication)
	protectedRoutes := router.PathPrefix("/api").Subrouter()
	protectedRoutes.Use(middleware.AuthMiddleware(middlewareHandler))

	// Wallet Routes
	protectedRoutes.HandleFunc("/balance", walletHandler.GetBalanceHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/transfer", walletHandler.TransferFundsHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc("/transactions", walletHandler.GetTransactionsHandler).Methods(http.MethodGet)

	// KYC Routes
	protectedRoutes.HandleFunc("/requestkyc", userHandler.RequestKYCHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc("/kycrequests", userHandler.GetKYCRequestsHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/kycaction", userHandler.KYCRequestActionHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc("/kycdetails", userHandler.GetKYCDetailedInfoHandler).Methods(http.MethodGet)

	// loan Routes
	protectedRoutes.HandleFunc("/loan/create", loanHandler.CreateLoanOfferHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc("/loan/accept", loanHandler.AcceptLoanOfferHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc("/loan/repay", loanHandler.RepayLoanHandler).Methods(http.MethodPost)

	return router
}
