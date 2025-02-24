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

const (
	SignupEndpoint                = "/signup"
	SignInEndpoint                = "/signin"
	APIPathPrefix                 = "/api"
	BalanceEndpoint               = "/balance"
	TransferEndpoint              = "/transfer"
	TransactionsEndpoint          = "/transactions"
	RequestKYCEndpoint            = "/requestkyc"
	KYCRequestsEndpoint           = "/kycrequests"
	KYCActionEndpoint             = "/kycaction"
	KYCDetailsEndpoint            = "/kycdetails"
	LoansApplyEndpoint            = "/loans/apply"
	LoanApplicationByIDEndpoint   = "/loans/applications/{application_id}"
	LoanApplicationsEndpoint      = "/loans/applications"
	LoanOfferEndpoint             = "/loans/applications/{application_id}/offers"
	LoanOfferByIDEndpoint         = "/loans/offers/{offer_id}"
	OffersByApplicationIDEndpoint = "/loans/applications/{application_id}/offers"
	LoanOffersEndpoint            = "/loans/offers"
	AcceptOfferEndpoint           = "/loans/offers/{offer_id}/accept"
	DisburseLoanEndpoint          = "/loans/disburse/{offer_id}"
	LoanDetailsByIDEndpoint       = "/loans/{loan_id}"
	LoanDetailsEndpoint           = "/loans"
	CalculatePayableEndpoint      = "/loans/{loan_id}/settle"
	SettleLoanEndpoint            = "/loans/{loan_id}/settle"
)

func SetupRoutes(ctx context.Context, deps *Dependencies) *mux.Router {
	router := mux.NewRouter()

	// Inject dependencies into handlers
	userHandler := user.NewHandler(deps.UserService)
	walletHandler := wallet.NewHandler(deps.WalletService)
	loanHandler := loan.NewHandler(deps.LoanService)
	middlewareHandler := middleware.NewHandler(deps.MiddlewareService)

	// Use RequestIDMiddleware and PostProcessingMiddleware globally
	router.Use(middlewareHandler.RequestLoggingMiddleware)

	// Signup Endpoint
	router.HandleFunc(SignupEndpoint, userHandler.SignupHandler).Methods(http.MethodPost)
	// SignIn Endpoint
	router.HandleFunc(SignInEndpoint, userHandler.SignInHandler).Methods(http.MethodPost)

	// Protected routes (Require authentication)
	protectedRoutes := router.PathPrefix(APIPathPrefix).Subrouter()
	protectedRoutes.Use(middleware.AuthMiddleware(middlewareHandler))

	// Wallet Routes
	protectedRoutes.HandleFunc(BalanceEndpoint, walletHandler.GetBalanceHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(TransferEndpoint, walletHandler.TransferFundsHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc(TransactionsEndpoint, walletHandler.GetTransactionsHandler).Methods(http.MethodGet)

	// KYC Routes
	protectedRoutes.HandleFunc(RequestKYCEndpoint, userHandler.RequestKYCHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc(KYCRequestsEndpoint, userHandler.GetKYCRequestsHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(KYCActionEndpoint, userHandler.KYCRequestActionHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc(KYCDetailsEndpoint, userHandler.GetKYCDetailedInfoHandler).Methods(http.MethodGet)

	// Loan Routes
	protectedRoutes.HandleFunc(LoansApplyEndpoint, loanHandler.CreateLoanApplicationHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc(LoanApplicationByIDEndpoint, loanHandler.GetLoanApplicationByIDHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(LoanApplicationsEndpoint, loanHandler.GetLoanAppliactionsHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(LoanOfferEndpoint, loanHandler.CreateLoanOfferHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc(LoanOfferByIDEndpoint, loanHandler.GetLoanOfferByIDHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(OffersByApplicationIDEndpoint, loanHandler.GetOffersByApplicationIDHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(LoanOffersEndpoint, loanHandler.GetLoanOffersHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(AcceptOfferEndpoint, loanHandler.AcceptOfferHandler).Methods(http.MethodPut)
	protectedRoutes.HandleFunc(DisburseLoanEndpoint, loanHandler.DisburseLoanHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc(LoanDetailsByIDEndpoint, loanHandler.GetLoanDetailsByIDHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(LoanDetailsEndpoint, loanHandler.GetLoanDetailsHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(CalculatePayableEndpoint, loanHandler.CalculatePayableHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(SettleLoanEndpoint, loanHandler.SettleLoanHandler).Methods(http.MethodPost)

	return router
}
