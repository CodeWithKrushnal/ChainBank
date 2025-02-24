package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// RequestLoggingMiddleware generates a unique request ID for each request and logs the request details after processing.
func (md Handler) RequestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate a new unique request ID for the request
		requestID := uuid.New().String()

		// Add the request ID and an empty UserID field to the context
		ctx := context.WithValue(r.Context(), "RequestID", requestID)

		// Set the request ID in the response headers for logging/tracing purposes
		w.Header().Set("X-Request-ID", requestID)

		// Capture the start time
		startTime := time.Now()

		// Initialize with default OK status
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		if r.RequestURI == "/signup" || r.RequestURI == "/signin" {
			// Get the IP address
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

			// Create the request log
			receivedRequestID, err := md.service.CreateRequestLog(ctx, requestID, "", r.RequestURI, r.Method, []byte{}, ipAddress)
			if err != nil || receivedRequestID != requestID || receivedRequestID == "" {
				log.Printf("Error Creating the Request Log: %v", err)
				return
			}
		}

		// Execute the handler (the actual business logic)
		next.ServeHTTP(rw, r.WithContext(ctx))

		// Capture the end time
		endTime := time.Now()

		// Calculate duration
		duration := endTime.Sub(startTime)

		err := md.service.UpdateRequestLog(ctx, requestID, rw.status, int(duration.Milliseconds()))

		if err != nil {
			log.Printf("Error updating operation log: %v", err)
		}
	})
}
