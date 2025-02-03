package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/CodeWithKrushnal/ChainBank/internal/api/config"
	"github.com/CodeWithKrushnal/ChainBank/internal/repository/postgres"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		userEmail, err := ValidateJWT(tokenParts[1])
		if err != nil {
			http.Error(w, "Unauthorized: Invalid Token", http.StatusUnauthorized)
			return
		}

		//Getting User Details
		user, err := postgres.GetUserByEmail(userEmail)
		if err != nil {
			log.Println("Error Retrieving the UserID From email in authmiddleware")
		}

		//Getting User Role
		userRole, err := postgres.GetUserHighestRole(user.ID)
		if err != nil {
			log.Printf("Error Retrieing the role for user")
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), "userInfo", struct {
			UserID    string
			UserEmail string
			UserRole  int
		}{
			UserID:    user.ID, // replace with actual user ID field
			UserEmail: userEmail,
			UserRole:  userRole,
		})

		err = postgres.UpdateLastLogin(user.ID)
		if err != nil {
			log.Println("Error Updating the Login Info")
			return
		}
		log.Print("User Authenticated")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ValidateJWT(tokenString string) (string, error) {

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
			return "", errors.New("invalid token claims")
		}
		return userEmail, nil
	}

	return "", errors.New("invalid token")
}
