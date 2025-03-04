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
	SignupEndpoint                    = "/signup"
	SignInEndpoint                    = "/signin"
	APIPathPrefix                     = "/api"
	GetUserEndpoint                   = "/user"
	BalanceEndpoint                   = "/balance"
	TransferEndpoint                  = "/transfer"
	TransactionsEndpoint              = "/transactions"
	RequestKYCEndpoint                = "/requestkyc"
	KYCRequestsEndpoint               = "/kycrequests"
	KYCActionEndpoint                 = "/kycaction"
	KYCDetailsEndpoint                = "/kycdetails"
	LoansApplyEndpoint                = "/loans/apply"
	LoanApplicationByIDEndpoint       = "/loans/applications/{application_id}"
	LoanApplicationsEndpoint          = "/loans/applications"
	LoanOfferEndpoint                 = "/loans/applications/{application_id}/offers"
	LoanOfferByIDEndpoint             = "/loans/offers/{offer_id}"
	OffersByApplicationIDEndpoint     = "/loans/applications/{application_id}/offers"
	LoanOffersEndpoint                = "/loans/offers"
	AcceptOfferEndpoint               = "/loans/offers/{offer_id}/accept"
	DisburseLoanEndpoint              = "/loans/disburse/{offer_id}"
	LoanDetailsByIDEndpoint           = "/loans/{loan_id}"
	LoanDetailsEndpoint               = "/loans"
	CalculatePayableEndpoint          = "/loans/{loan_id}/settle"
	SettleLoanEndpoint                = "/loans/{loan_id}/settle"
	ResetPasswordEndpoint             = "/resetpassword"
	UpdatePasswordEndpoint            = "/updatepassword"
	GenerateEmailVerificationEndpoint = "/verifyemail"
	RequestLogsEndpoint               = "/requestlogs"
	RequestLogStatsEndpoint           = "/requestlogstats"
	TransactionStats                  = "/transactionstats"
	EthPrice                          = "/ethprice"
	AdminStats                        = "/adminstats"
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
	// Reset Password Endpoint
	router.HandleFunc(ResetPasswordEndpoint, userHandler.ResetPasswordHandler).Methods(http.MethodPost)
	// Update Password Endpoint
	router.HandleFunc(UpdatePasswordEndpoint, userHandler.UpdatePasswordHandler).Methods(http.MethodPost)
	// Generate Email Verification Endpoint
	router.HandleFunc(GenerateEmailVerificationEndpoint, userHandler.GenerateEmailVerificationHandler).Methods(http.MethodPost)

	// Protected routes (Require authentication)
	protectedRoutes := router.PathPrefix(APIPathPrefix).Subrouter()
	protectedRoutes.Use(middleware.AuthMiddleware(middlewareHandler))

	//User Routes
	protectedRoutes.HandleFunc(GetUserEndpoint, userHandler.GetUserDetailsHandler).Methods(http.MethodGet)

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
	protectedRoutes.HandleFunc(RequestLogsEndpoint, userHandler.GetRequestLogsHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc(RequestLogStatsEndpoint, userHandler.GetRequestLogStatsHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc(TransactionStats, userHandler.GetTransactionStatsHandler).Methods(http.MethodPost)
	protectedRoutes.HandleFunc(EthPrice, userHandler.GetEthPriceHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc(AdminStats, userHandler.ExecuteQueryHandler).Methods(http.MethodPost)

	return router
}
