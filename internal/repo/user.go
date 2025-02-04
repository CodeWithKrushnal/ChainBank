package repo

import (
	"database/sql"
	_ "database/sql"
	"fmt"
	"log"
	"time"
)

// User Regular struct
type User struct {
	ID        string
	Username  string
	Email     string
	Password  string
	CreatedAt time.Time
}

// All User Queries
const (
	roleAssignmentQuery             = `INSERT INTO user_roles_assignment(user_id, role_id) VALUES ($1, $2)`
	userRegisterQuery               = `INSERT INTO users (username, email, password_hash, full_name, date_of_birth) VALUES ($1, $2, $3, $4, $5)`
	getUserByEmailQuery             = `SELECT user_id, username, email, password_hash, created_at FROM users WHERE email=$1`
	updateLastLoginQuery            = `UPDATE users SET last_login = $1 WHERE user_id = $2`
	usernameAlreadyInExistanceQuery = `SELECT CASE WHEN username = $1 THEN TRUE ELSE FALSE END FROM users`
	emailAlreadyInExistanceQuery    = `SELECT CASE WHEN email = $1 THEN TRUE ELSE FALSE END FROM users`
	getUserRolesQuery               = `SELECT MAX(role_id) FROM user_roles_assignment WHERE user_id = $1`
	updateWalletIDQuery             = `INSERT INTO wallets (wallet_id,user_id) VALUES ($1,$2)`
)

type userRepo struct {
	DB *sql.DB
}

type UserStorer interface {
	CreateUser(username, email, passwordHash, fullName, dob, walletAddress string, role int) error
	GetUserByEmail(email string) (User, error)
	UpdateLastLogin(userID string) error
	UserExists(userName, email string) (usernameAlreadyInExistance, emailAlreadyInExistance bool, err error)
	GetUserHighestRole(userID string) (int, error)
}

// Constructor function
func NewUserRepo(db *sql.DB) UserStorer {
	return &userRepo{DB: db}
}

// Creates a new user in DB
func (repoDep *userRepo) CreateUser(username, email, passwordHash, fullName, dob, walletAddress string, role int) error {
	_, err := repoDep.DB.Exec(userRegisterQuery, username, email, passwordHash, fullName, dob)
	if err != nil {
		log.Printf("Error inserting user into database: %v", err.Error())
		return err
	}

	//Retrieveing the user object from email
	user, err := repoDep.GetUserByEmail(email)

	if err != nil {
		log.Printf("Error Finding the User related to email ID %v", &email)
		return err
	}

	// Assigning Role to user
	_, err = repoDep.DB.Exec(roleAssignmentQuery, user.ID, role)

	if err != nil {
		log.Println("Error Writing the role information realted to user in user_roles_assignment table")
	}

	//Update wallet_id In wallets table
	_, err = repoDep.DB.Exec(updateWalletIDQuery, walletAddress, user.ID)
	if err != nil {
		log.Println("Error Occured While Inserting data into wallet Table")
	}

	return nil
}

// Returnes a user object by passing email
func (repoDep *userRepo) GetUserByEmail(email string) (User, error) {
	var user User
	err := repoDep.DB.QueryRow(getUserByEmailQuery, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	return user, err
}

// Updates the last login field in users table to current time
func (repoDep *userRepo) UpdateLastLogin(userID string) error {
	log.Print("Received the Request to update login time")

	result, err := repoDep.DB.Exec(updateLastLoginQuery, time.Now(), userID)

	if err != nil {
		log.Printf("Error executing query: %v", err)
		return fmt.Errorf("error updating last_login: %v", err)
	}

	// Check if any row was affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking affected rows: %v", err)
		return fmt.Errorf("error checking affected rows: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no user found with userID: %s", userID)
	}

	log.Print("Updated last login successfully")
	return nil
}

// Returnes if User Already exists on the basis of email & username
func (repoDep *userRepo) UserExists(userName, email string) (usernameAlreadyInExistance, emailAlreadyInExistance bool, err error) {

	//Check if username already Exists
	err = repoDep.DB.QueryRow(usernameAlreadyInExistanceQuery, userName).Scan(&usernameAlreadyInExistance)

	if err != nil {
		log.Printf("Error Checking the user Existance status: %v", err)
		return usernameAlreadyInExistance, emailAlreadyInExistance, err
	}

	//Checking if Email Already Exists
	err = repoDep.DB.QueryRow(emailAlreadyInExistanceQuery, email).Scan(&emailAlreadyInExistance)

	if err != nil {
		log.Printf("Error Checking the user Existance status: %v", err)
		return usernameAlreadyInExistance, emailAlreadyInExistance, err
	}
	return usernameAlreadyInExistance, emailAlreadyInExistance, err
}

// GetHighestRole fetches the highest role assigned to a user based on user_id.
func (repoDep *userRepo) GetUserHighestRole(userID string) (int, error) {

	var highestRoleLevel int

	// Query role assigned to the user.
	err := repoDep.DB.QueryRow(getUserRolesQuery, userID).Scan(&highestRoleLevel)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return 0, fmt.Errorf("error fetching user roles: %v", err)
	}

	// Check if we found any roles, else return an error.
	if highestRoleLevel == 0 {
		return 0, fmt.Errorf("no roles found for user %s", userID)
	}

	// Return the highest role ID.
	return highestRoleLevel, nil
}
