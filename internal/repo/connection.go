package repo

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/CodeWithKrushnal/ChainBank/utils"
	_ "github.com/lib/pq" // Import PostgreSQL driver
)

// InitDB initializes the database connection using the provided connection string.
// It returns a pointer to the sql.DB instance and an error if any occurs.
func InitDB(connString string) (*sql.DB, error) {
	var db *sql.DB

	// Attempt to open a connection to the database
	var err error
	db, err = sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrConfigInit, err)
	}

	// Check if the database is reachable
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrServiceInit, err)
	}

	// Log the successful database connection establishment
	slog.Info("Database connection established")
	return db, nil
}

// CloseDB closes the database connection if it is not nil.
// It returns an error if the closing operation fails.
func CloseDB(db *sql.DB) error {
	if db != nil {
		if err := db.Close(); err != nil {
			// Propagate the error without logging
			return fmt.Errorf("%s: %w", utils.ErrConfigInit, err)
		}
	}
	return nil
}
