package postgres

import (
	"database/sql"
	"log"
	_ "github.com/lib/pq" // Import PostgreSQL driver
)

var DB *sql.DB // Global database connection

// InitDB initializes the database connection
func InitDB(connString string) error {

	var err error
	DB, err = sql.Open("postgres", connString)
	if err != nil {
		log.Printf("Error initializing database: %v", err)
		return err
	}
	if err = DB.Ping(); err != nil {
		log.Printf("Error connecting to database: %v", err)
		return err
	}
	log.Println("Database connection established")
	return nil
}

// CloseDB closes the database connection
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
