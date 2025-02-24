package repo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/CodeWithKrushnal/ChainBank/utils"
)

//structs

type User struct {
	ID        string
	Username  string
	Email     string
	Password  string
	CreatedAt time.Time
}

type RequestLog struct {
	RequestID      string
	UserID         string
	Endpoint       string
	HTTPMethod     string
	RequestPayload []byte
	ResponseStatus int
	ResponseTimeMs int
	IPAddress      string
	CreatedAt      time.Time
}

type KYCRecord struct {
	KYCID              string
	UserID             string
	DocumentType       string
	DocumentNumber     string
	VerificationStatus string
	SubmittedAt        time.Time
	VerifiedAt         time.Time
	VerifiedBy         string
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
	getUserByIDQuery                 = `SELECT user_id, username, email, password_hash, created_at FROM users WHERE user_id=$1`
	getKYCDetailedInfoQuery          = `SELECT * FROM kyc_verifications WHERE 1=1`
	createRequestLogQuery            = `INSERT INTO api_requests_log (request_id, user_id, endpoint, http_method, request_payload, ip_address) VALUES ($1, $2, $3, $4, $5, $6) RETURNING request_id`
	updateRequestLogQuery            = `UPDATE api_requests_log SET response_status = $1, response_time_ms = $2 WHERE request_id = $3`
)

type userRepo struct {
	DB *sql.DB
}

type UserStorer interface {
	CreateUser(ctx context.Context, username, email, passwordHash, fullName, dob, walletAddress string, role int) error
	GetUserByEmail(ctx context.Context, email string) (User, error)
	UpdateLastLogin(ctx context.Context, userID string) error
	UserExists(ctx context.Context, userName, email string) (usernameAlreadyInExistance, emailAlreadyInExistance bool, err error)
	GetUserHighestRole(ctx context.Context, userID string) (int, error)
	InsertKYCVerification(ctx context.Context, userID, documentType, documentNumber, verificationStatus string) (string, error)
	GetAllKYCVerifications(ctx context.Context) ([]KYCRecord, error)
	UpdateKYCVerificationStatus(ctx context.Context, user_id, verificationStatus, verifiedBy string) error
	GetKYCDetailedInfo(ctx context.Context, kycID, userID string) ([]KYCRecord, error)
	GetuserByID(ctx context.Context, userID string) (User, error)
	CreateRequestLog(ctx context.Context, requestID, userID, endpoint, httpMethod string, requestPayload interface{}, ipAddress string) (string, error)
	UpdateRequestLog(ctx context.Context, requestID string, responseStatus, responseTimeMs int) error
}

// Constructor function
func NewUserRepo(db *sql.DB) UserStorer {
	return &userRepo{DB: db}
}

// Creates a new user in DB
func (rd *userRepo) CreateUser(ctx context.Context, username, email, passwordHash, fullName, dob, walletAddress string, role int) error {

	// Attempt to insert a new user into the database
	_, err := rd.DB.Exec(userRegisterQuery, username, email, passwordHash, fullName, dob)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrInsertUser, err)
	}

	// Retrieve the user object by email
	user, err := rd.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("%s %v: %w", utils.ErrFindUserByEmail, email, err)
	}

	// Assign role to the user
	_, err = rd.DB.Exec(roleAssignmentQuery, user.ID, role)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrAssignRole, err)
	}

	// Update wallet_id in wallets table
	_, err = rd.DB.Exec(updateWalletIDQuery, walletAddress, user.ID)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrInsertWallet, err)
	}

	return nil
}

// Returnes a user object by passing email
func (rd *userRepo) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var user User

	// Attempt to retrieve the user by email
	err := rd.DB.QueryRow(getUserByEmailQuery, email).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return a specific error if no user is found
			return user, fmt.Errorf("%s: %w", utils.ErrUserNotFound, err)
		}
		// Propagate any other errors
		return user, fmt.Errorf("%s: %w", utils.ErrFindUserByEmail, err)
	}

	return user, nil
}

// UpdateLastLogin updates the last login timestamp for a user.
func (rd *userRepo) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := rd.DB.Exec(updateLastLoginQuery, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrUpdatingLastLogin, err)
	}
	return nil
}

// UserExists checks if a user already exists based on username and email.
func (rd *userRepo) UserExists(ctx context.Context, userName, email string) (usernameAlreadyExists, emailAlreadyExists bool, err error) {

	// Check if username already exists
	err = rd.DB.QueryRow(usernameAlreadyInExistanceQuery, userName).Scan(&usernameAlreadyExists)
	if err != nil {
		return usernameAlreadyExists, emailAlreadyExists, fmt.Errorf("%s: %w", utils.ErrCheckingUsername, err)
	}

	// Check if email already exists
	err = rd.DB.QueryRow(emailAlreadyInExistanceQuery, email).Scan(&emailAlreadyExists)
	if err != nil {
		return usernameAlreadyExists, emailAlreadyExists, fmt.Errorf("%s: %w", utils.ErrCheckingEmail, err)
	}

	return usernameAlreadyExists, emailAlreadyExists, nil
}

// GetUserHighestRole fetches the highest role assigned to a user based on user_id.
func (rd *userRepo) GetUserHighestRole(ctx context.Context, userID string) (int, error) {

	var highestRoleLevel int

	// Query role assigned to the user.
	err := rd.DB.QueryRow(getUserRolesQuery, userID).Scan(&highestRoleLevel)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", utils.ErrFetchingRoles, err)
	}

	// Check if we found any roles, else return an error.
	if highestRoleLevel == 0 {
		return 0, fmt.Errorf("%s %s: %w", utils.ErrNoRolesFound, userID, sql.ErrNoRows)
	}

	// Return the highest role ID.
	return highestRoleLevel, nil
}

// InsertKYCVerification inserts a new KYC verification record.
func (rd *userRepo) InsertKYCVerification(ctx context.Context, userID, documentType, documentNumber, verificationStatus string) (string, error) {
	var kycID string

	err := rd.DB.QueryRowContext(ctx, insertKYCVerificationQuery, userID, documentType, documentNumber, verificationStatus).Scan(&kycID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrInsertKYCVerification, err)
	}
	return kycID, nil
}

// GetAllKYCVerifications retrieves all KYC verification records.
func (rd *userRepo) GetAllKYCVerifications(ctx context.Context) ([]KYCRecord, error) {
	rows, err := rd.DB.QueryContext(ctx, getAllKYCVerificationsQuery)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrFetchKYCVerifications, err)
	}
	defer rows.Close()

	var records []KYCRecord
	for rows.Next() {
		var kycID, userID, documentType, documentNumber, verificationStatus, verifiedBy sql.NullString
		var submittedAt, verifiedAt sql.NullTime

		if err := rows.Scan(&kycID, &userID, &documentType, &documentNumber, &verificationStatus, &submittedAt, &verifiedAt, &verifiedBy); err != nil {
			return nil, fmt.Errorf("%s: %w", utils.ErrScanRow, err)
		}

		record := KYCRecord{
			KYCID:              kycID.String,
			UserID:             userID.String,
			DocumentType:       documentType.String,
			DocumentNumber:     documentNumber.String,
			VerificationStatus: verificationStatus.String,
			SubmittedAt:        submittedAt.Time,
			VerifiedAt:         verifiedAt.Time,
			VerifiedBy:         verifiedBy.String,
		}
		records = append(records, record)
	}
	return records, nil
}

// UpdateKYCVerificationStatus updates verification_status, verified_at, and verified_by.
func (rd *userRepo) UpdateKYCVerificationStatus(ctx context.Context, kycID, verificationStatus, verifiedBy string) error {

	_, err := rd.DB.ExecContext(ctx, updateKYCVerificationStatusQuery, verificationStatus, time.Now(), verifiedBy, kycID)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrUpdateKYCVerification, err)
	}
	return nil
}

// GetKYCDetailedInfo retrieves detailed KYC information based on kycID or userID.
func (rd *userRepo) GetKYCDetailedInfo(ctx context.Context, kycID, userID string) ([]KYCRecord, error) {
	var query string
	var args []interface{}

	// Build the query based on provided kycID or userID
	if kycID != "" {
		query = getKYCDetailedInfoQuery + ` AND kyc_id = $1`
		args = append(args, kycID)
	} else {
		query = getKYCDetailedInfoQuery + ` AND user_id = $1`
		args = append(args, userID)
	}

	// Execute the query
	rows, err := rd.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrFetchKYCDetailedInfo, err)
	}
	defer rows.Close()

	var records []KYCRecord
	for rows.Next() {
		var kycID, userID, documentType, documentNumber, verificationStatus, verifiedBy sql.NullString
		var submittedAt, verifiedAt sql.NullTime

		// Scan the row into variables
		if err := rows.Scan(&kycID, &userID, &documentType, &documentNumber, &verificationStatus, &submittedAt, &verifiedAt, &verifiedBy); err != nil {
			return nil, fmt.Errorf("%s: %w", utils.ErrScanRow, err)
		}

		// Create a KYCRecord for the KYC details
		record := KYCRecord{
			KYCID:              kycID.String,
			UserID:             userID.String,
			DocumentType:       documentType.String,
			DocumentNumber:     documentNumber.String,
			VerificationStatus: verificationStatus.String,
			SubmittedAt:        submittedAt.Time,
			VerifiedAt:         verifiedAt.Time,
			VerifiedBy:         verifiedBy.String,
		}
		records = append(records, record)
	}
	return records, nil
}

// GetuserByID retrieves user information based on userID.
func (rd *userRepo) GetuserByID(ctx context.Context, userID string) (User, error) {
	var user User
	err := rd.DB.QueryRow(getUserByIDQuery, userID).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt)
	return user, err
}

// CreateRequestLog creates a new request log entry.
func (rd *userRepo) CreateRequestLog(ctx context.Context, requestID, userID, endpoint, httpMethod string, requestPayload interface{}, ipAddress string) (string, error) {
	// Check if userID is empty and set to default value
	if userID == "" {
		userID = "00000000-0000-0000-0000-000000000000"
		slog.Warn(utils.ErrEmptyUserID.Error())
	}

	// Convert requestPayload to a string for better readability
	requestPayloadJSON := string(requestPayload.([]byte))

	if requestPayloadJSON == "" {
		requestPayloadJSON = "{}"
	}

	// Execute the query
	var logRequestID string
	err := rd.DB.QueryRowContext(ctx, createRequestLogQuery, requestID, userID, endpoint, httpMethod, requestPayloadJSON, ipAddress).Scan(&logRequestID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrInsertRequestLog, err)
	}

	return logRequestID, nil
}

// UpdateRequestLog updates the request log entry with the given response status and response time.
func (rd *userRepo) UpdateRequestLog(ctx context.Context, requestID string, responseStatus, responseTimeMs int) error {
	// Execute the query to update the request log
	_, err := rd.DB.ExecContext(ctx, updateRequestLogQuery, responseStatus, responseTimeMs, requestID)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrUpdateRequestLog, err)
	}

	return nil
}