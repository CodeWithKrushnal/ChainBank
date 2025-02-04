package repo

import (
	"database/sql"
	_ "github.com/lib/pq" // Import PostgreSQL driver
	"log"
)

// InitDB initializes the database connection
func InitDB(connString string) (*sql.DB, error) {
	var db *sql.DB

	var err error
	db, err = sql.Open("postgres", connString)
	if err != nil {
		log.Printf("Error initializing database: %v", err)
		return db, err
	}
	if err = db.Ping(); err != nil {
		log.Printf("Error connecting to database: %v", err)
		return db, err
	}
	log.Println("Database connection established")
	return db, err
}

// CloseDB closes the database connection
func CloseDB(db *sql.DB) {
	if db != nil {
		db.Close()
	}
}
