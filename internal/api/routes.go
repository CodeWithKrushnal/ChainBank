package api

import (
	"github.com/CodeWithKrushnal/ChainBank/internal/api/handlers"
	"github.com/CodeWithKrushnal/ChainBank/internal/api/middleware"
	"github.com/gorilla/mux"
)

func SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	//Signup Endpoint
	router.HandleFunc("/signup", handlers.SignupHandler).Methods("POST")
	//SignIn Endpoint
	router.HandleFunc("/signin", handlers.SignInHandler).Methods("POST")

	// Protected routes (Require authentication)
	protectedRoutes := router.PathPrefix("/api").Subrouter()
	protectedRoutes.Use(middleware.AuthMiddleware)

	protectedRoutes.HandleFunc("/trial", handlers.TrialHandler).Methods("POST")
	protectedRoutes.HandleFunc("/balance", handlers.GetBalanceHandler).Methods("GET")
	protectedRoutes.HandleFunc("/transfer",handlers.TransferFundsHandler).Methods("POST")

	return router
}
