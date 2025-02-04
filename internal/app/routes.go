package app

import (
	"net/http"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/user"
	"github.com/CodeWithKrushnal/ChainBank/internal/app/wallet"
	"github.com/CodeWithKrushnal/ChainBank/middleware"
	"github.com/gorilla/mux"
)

func SetupRoutes(deps *Dependencies) *mux.Router {
	router := mux.NewRouter()
	// Inject dependencies into handlers
	userHandler := user.NewHandler(deps.UserService)
	walletHandler := wallet.NewHandler(deps.WalletService)
	middlewareHandler := middleware.NewHandler(deps.MiddlewareService)

	//Signup Endpoint
	router.HandleFunc("/signup", userHandler.SignupHandler).Methods(http.MethodPost)
	//SignIn Endpoint
	router.HandleFunc("/signin", userHandler.SignInHandler).Methods(http.MethodPost)

	// Protected routes (Require authentication)
	protectedRoutes := router.PathPrefix("/api").Subrouter()
	protectedRoutes.Use(middleware.AuthMiddleware(middlewareHandler))

	protectedRoutes.HandleFunc("/balance", walletHandler.GetBalanceHandler).Methods(http.MethodGet)
	protectedRoutes.HandleFunc("/transfer", walletHandler.TransferFundsHandler).Methods(http.MethodPost)

	return router
}
