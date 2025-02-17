package middleware

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/CodeWithKrushnal/ChainBank/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func ValidateJWT(tokenString string, originIP string) (string, error) {

	JWT_SECRET := []byte(config.ConfigDetails.JWTSecretKey)

	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return JWT_SECRET, nil
	})

	if err != nil {
		return "", err
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userEmail, ok := claims["email"].(string)
		if !ok {
			return "", fmt.Errorf("invalid token claims")
		}

		if claims["origin"].(string) != originIP {
			return "", fmt.Errorf("Token is invalid : invalid Token Origin")
		}
		return userEmail, nil
	}

	return "", errors.New("invalid token")
}

type Handler struct {
	service Service
}

// Constructor function
func NewHandler(service Service) Handler {
	return Handler{service: service}
}

func AuthMiddleware(authDep Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			// Allow signup and login without authentication
			if r.URL.Path == "/signup" || r.URL.Path == "/signin" {
				next.ServeHTTP(w, r)
				return
			}

			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
				return
			}

			// Check if it follows "Bearer <token>" format
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				http.Error(w, "Invalid Authorization Header Format", http.StatusUnauthorized)
				return
			}

			// Validate token
			userEmail, err := ValidateJWT(tokenParts[1], r.RemoteAddr)
			if err != nil {
				http.Error(w, "Unauthorized: Invalid Token", http.StatusUnauthorized)
				return
			}

			// Getting User Details from userRepo
			user, err := authDep.service.getUserByEmail(ctx, userEmail)
			if err != nil {
				log.Println("Error Retrieving the UserID From email in authmiddleware")
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			// Add user info to request context
			ctx = context.WithValue(r.Context(), "UserID", user.ID)

			// Update last login
			err = authDep.service.updateLastLogin(ctx, user.ID)
			if err != nil {
				log.Println("Error Updating the Login Info", err.Error())
				return
			}

			log.Print("User Authenticated")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
