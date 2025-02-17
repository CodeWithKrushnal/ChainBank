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
	protectedRoutes.HandleFunc("/loans/apply", loanHandler.CreateLoanApplicationHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc("/loans/applications/{application_id}", loanHandler.GetLoanApplicationByIDHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/loans/applications", loanHandler.GetLoanAppliactionsHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/loans/applications/{application_id}/offers", loanHandler.CreateLoanOfferHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc("/loans/offers/{offer_id}", loanHandler.GetLoanOfferByIDHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/loans/applications/{application_id}/offers", loanHandler.GetOffersByApplicationIDHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/loans/offers", loanHandler.GetLoanOffersHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/loans/offers/{offer_id}/accept", loanHandler.AcceptOfferHandler).Methods(http.MethodPut)
	protectedRoutes.HandleFunc("/loans/disburse/{offer_id}", loanHandler.DisburseLoanHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc("/loans/{loan_id}", loanHandler.GetLoanDetailsByIDHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/loans", loanHandler.GetLoanDetailsHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/loans/{loan_id}/settle", loanHandler.CalculatePayableHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/loans/{loan_id}/settle", loanHandler.SettleLoanHandler).Methods(http.MethodPost)

	return router
}
