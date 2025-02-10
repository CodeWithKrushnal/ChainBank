package repo

import (
	"context"
	"database/sql"
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
	roleAssignmentQuery              = `INSERT INTO user_roles_assignment(user_id, role_id) VALUES ($1, $2)`
	userRegisterQuery                = `INSERT INTO users (username, email, password_hash, full_name, date_of_birth) VALUES ($1, $2, $3, $4, $5)`
	getUserByEmailQuery              = `SELECT user_id, username, email, password_hash, created_at FROM users WHERE email=$1`
	updateLastLoginQuery             = `UPDATE users SET last_login = $1 WHERE user_id = $2`
	usernameAlreadyInExistanceQuery  = `SELECT CASE WHEN username = $1 THEN TRUE ELSE FALSE END FROM users`
	emailAlreadyInExistanceQuery     = `SELECT CASE WHEN email = $1 THEN TRUE ELSE FALSE END FROM users`
	getUserRolesQuery                = `SELECT MAX(role_id) FROM user_roles_assignment WHERE user_id = $1`
	updateWalletIDQuery              = `INSERT INTO wallets (wallet_id,user_id) VALUES ($1,$2)`
	updateKYCVerificationStatusQuery = `UPDATE kyc_verifications SET verification_status = $1, verified_at = $2, verified_by = $3 WHERE kyc_id = $4`
	getAllKYCVerificationsQuery      = `SELECT * FROM kyc_verifications WHERE verification_status='Pending'`
	insertKYCVerificationQuery       = `INSERT INTO kyc_verifications (user_id, document_type, document_number, verification_status) VALUES ($1, $2, $3, $4) RETURNING kyc_id`
)

type userRepo struct {
	DB *sql.DB
}

type UserStorer interface {
	CreateUser(ctx context.Context, username, email, passwordHash, fullName, dob, walletAddress string, role int) error
	GetUserByEmail(ctx context.Context, email string) (User, error)
	UpdateLastLogin(ctx context.Context) error
	UserExists(ctx context.Context, userName, email string) (usernameAlreadyInExistance, emailAlreadyInExistance bool, err error)
	GetUserHighestRole(ctx context.Context, userID string) (int, error)
	InsertKYCVerification(ctx context.Context, userID, documentType, documentNumber, verificationStatus string) (string, error)
	GetAllKYCVerifications(ctx context.Context) ([]map[string]interface{}, error)
	UpdateKYCVerificationStatus(ctx context.Context, user_id, verificationStatus, verifiedBy string) error
	GetKYCDetailedInfo(ctx context.Context, kycID, userID string) ([]map[string]interface{}, error)
}

// Constructor function
func NewUserRepo(db *sql.DB) UserStorer {
	return &userRepo{DB: db}
}

// Creates a new user in DB
func (repoDep *userRepo) CreateUser(ctx context.Context, username, email, passwordHash, fullName, dob, walletAddress string, role int) error {
	_, err := repoDep.DB.Exec(userRegisterQuery, username, email, passwordHash, fullName, dob)
	if err != nil {
		log.Printf("Error inserting user into database: %v", err.Error())
		return err
	}

	//Retrieveing the user object from email
	user, err := repoDep.GetUserByEmail(ctx, email)

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
func (repoDep *userRepo) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := repoDep.DB.QueryRow(getUserByEmailQuery, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	return user, err
}

// Updates the last login field in users table to current time
func (repoDep *userRepo) UpdateLastLogin(ctx context.Context) error {
	userInfo, ok := ctx.Value("userInfo").(struct {
		UserID    string
		UserEmail string
		UserRole  int
	})
	if !ok {
		return fmt.Errorf("Unauthorized: user info not found in context")
	}
	log.Print("Received the Request to update login time")

	_, err := repoDep.DB.Exec(updateLastLoginQuery, time.Now(), userInfo.UserID)

	if err != nil {
		log.Printf("Error executing query: %v", err)
		return fmt.Errorf("error updating last_login: %v", err)
	}

	log.Print("Updated last login successfully")
	return nil
}

// Returnes if User Already exists on the basis of email & username
func (repoDep *userRepo) UserExists(ctx context.Context, userName, email string) (usernameAlreadyInExistance, emailAlreadyInExistance bool, err error) {

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
func (repoDep *userRepo) GetUserHighestRole(ctx context.Context, userID string) (int, error) {

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

// InsertKYCVerification inserts a new KYC verification record.
func (repo *userRepo) InsertKYCVerification(ctx context.Context, userID, documentType, documentNumber, verificationStatus string) (string, error) {
	var kycID string

	log.Println("received info", userID, documentType, documentNumber, verificationStatus)
	err := repo.DB.QueryRowContext(ctx, insertKYCVerificationQuery, userID, documentType, documentNumber, verificationStatus).Scan(&kycID)
	if err != nil {
		log.Printf("Error inserting KYC verification: %v", err)
		return "", fmt.Errorf("failed to insert KYC verification: %v", err)
	}
	return kycID, nil
}

// GetAllKYCVerifications retrieves all KYC verification records.
func (repo *userRepo) GetAllKYCVerifications(ctx context.Context) ([]map[string]interface{}, error) {

	rows, err := repo.DB.QueryContext(ctx, getAllKYCVerificationsQuery)
	if err != nil {
		log.Printf("Error fetching KYC verifications: %v", err)
		return nil, fmt.Errorf("failed to fetch KYC verifications: %v", err)
	}
	defer rows.Close()

	var records []map[string]interface{}
	for rows.Next() {
		var kycID, userID, documentType, documentNumber, verificationStatus, verifiedBy sql.NullString
		var submittedAt, verifiedAt sql.NullTime

		err := rows.Scan(&kycID, &userID, &documentType, &documentNumber, &verificationStatus, &submittedAt, &verifiedAt, &verifiedBy)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		record := map[string]interface{}{
			"kyc_id":              kycID.String,
			"user_id":             userID.String,
			"document_type":       documentType.String,
			"document_number":     documentNumber.String,
			"verification_status": verificationStatus.String,
			"submitted_at":        submittedAt.Time,
			"verified_at":         verifiedAt.Time,
			"verified_by":         verifiedBy.String,
		}
		records = append(records, record)
	}
	return records, nil
}

// UpdateKYCVerificationStatus updates verification_status, verified_at, and verified_by.
func (repo *userRepo) UpdateKYCVerificationStatus(ctx context.Context, kyc_id, verificationStatus, verifiedBy string) error {
	_, err := repo.DB.ExecContext(ctx, updateKYCVerificationStatusQuery, verificationStatus, time.Now(), verifiedBy, kyc_id)
	if err != nil {
		log.Printf("Error updating KYC verification status: %v", err)
		return fmt.Errorf("failed to update KYC verification status: %v", err)
	}
	return nil
}

func (r *userRepo) GetKYCDetailedInfo(ctx context.Context, kycID, userID string) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}

	if kycID != "" {
		query = `SELECT * FROM kyc_verifications WHERE kyc_id = $1`
		args = append(args, kycID)
	} else {
		query = `SELECT * FROM kyc_verifications WHERE user_id = $1`
		args = append(args, userID)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("Error fetching KYC details: %v", err)
		return nil, fmt.Errorf("failed to fetch KYC details: %v", err)
	}
	defer rows.Close()

	var records []map[string]interface{}
	for rows.Next() {
		var kycID, userID, documentType, documentNumber, verificationStatus, verifiedBy sql.NullString
		var submittedAt, verifiedAt sql.NullTime

		err := rows.Scan(&kycID, &userID, &documentType, &documentNumber, &verificationStatus, &submittedAt, &verifiedAt, &verifiedBy)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		record := map[string]interface{}{
			"kyc_id":              kycID.String,
			"user_id":             userID.String,
			"document_type":       documentType.String,
			"document_number":     documentNumber.String,
			"verification_status": verificationStatus.String,
			"submitted_at":        submittedAt.Time,
			"verified_at":         verifiedAt.Time,
			"verified_by":         verifiedBy.String,
		}
		records = append(records, record)
	}
	return records, nil
}
