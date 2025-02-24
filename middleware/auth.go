package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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

			requestID := ctx.Value("RequestID").(string)

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

			// Get IP address without port number and handle IPv6
			ipAddress := r.RemoteAddr
			ipAddress = strings.TrimPrefix(ipAddress, "[") // Remove leading bracket for IPv6
			if i := strings.LastIndex(ipAddress, ":"); i != -1 {
				ipAddress = ipAddress[:i]
			}
			ipAddress = strings.TrimSuffix(ipAddress, "]") // Remove trailing bracket for IPv6

			// Convert IPv6 localhost to IPv4 localhost if needed
			if ipAddress == "::1" {
				ipAddress = "127.0.0.1"
			}

			// Read the request body
			requestBody, err := io.ReadAll(r.Body)
			if err != nil {
				log.Println("Error reading request body", err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			// Restore the request body so it can be read again later
			r.Body = io.NopCloser(bytes.NewBuffer(requestBody))

			receivedRequestID, err := authDep.service.CreateRequestLog(ctx, requestID, user.ID, r.RequestURI, r.Method, requestBody, ipAddress)
			if err != nil || receivedRequestID != requestID || receivedRequestID == "" {
				log.Println("Error Creating the Request Log", err.Error())
				return
			}

			log.Print("User Authenticated")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
