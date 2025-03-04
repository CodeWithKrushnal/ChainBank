package repo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
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

type CreateUserParams struct {
	Username      string
	Email         string
	PasswordHash  string
	FullName      string
	DOB           string
	WalletAddress string
	Role          int
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

type RequestLogFilter struct {
	RequestID      string     `json:"request_id"`
	UserID         string     `json:"user_id"`
	Endpoint       string     `json:"endpoint"`
	HTTPMethod     string     `json:"http_method"`
	ResponseStatus int        `json:"response_status"`
	ResponseTimeMs int        `json:"response_time_ms"`
	FromTime       *time.Time `json:"from_time"`
	ToTime         *time.Time `json:"to_time"`
}

type RequestLogStatsFilter struct {
	FromTime *time.Time `json:"from_time"`
	ToTime   *time.Time `json:"to_time"`
	Column   string     `json:"column"`
	Count    bool       `json:"count"`
	Avg      bool       `json:"avg"`
	Unique   bool       `json:"unique"`
	Min      bool       `json:"min"`
	Max      bool       `json:"max"`
	Sum      bool       `json:"sum"`
}

type TransactionStatsFilter struct {
	Column     string     `json:"column"`
	Count      bool       `json:"count"`
	Avg        bool       `json:"avg"`
	Unique     bool       `json:"unique"`
	Min        bool       `json:"min"`
	Max        bool       `json:"max"`
	Sum        bool       `json:"sum"`
	FromTime   *time.Time `json:"from_time"`
	ToTime     *time.Time `json:"to_time"`
	WalletType string     `json:"wallet_type"`
	WalletID   string     `json:"wallet_id"`
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
	updatePasswordHashQuery          = `UPDATE users SET password_hash = $1 WHERE user_id = $2`
	getRequestLogsQuery              = `SELECT request_id, user_id, endpoint, http_method, response_status, response_time_ms, created_at, ip_address FROM api_requests_log WHERE 1=1`
	getUserInfoQuery                 = `SELECT u.user_id, u.username, u.full_name, u.email, w.wallet_id, r.role_id from users u JOIN wallets w on u.user_id=w.user_id JOIN user_roles_assignment r ON u.user_id=r.user_id where u.user_id=$1`
)

type userRepo struct {
	DB            *sql.DB
	configDetails utils.ConfigStruct
}

type UserStorer interface {
	CreateUser(ctx context.Context, params CreateUserParams) error
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
	UpdateUserPasswordHash(ctx context.Context, userID string, newPasswordHash string) error
	GetRequestLogs(ctx context.Context, filter RequestLogFilter) ([]RequestLog, error)
	GetRequestLogStats(ctx context.Context, filter RequestLogStatsFilter) (interface{}, error)
	GetUserInfo(ctx context.Context, userID string) (utils.UserInfo, error)
	GetTransactionStats(ctx context.Context, filter TransactionStatsFilter) (interface{}, error)
	ExecuteQuery(ctx context.Context, query string, args ...interface{}) (interface{}, error)
}

// Constructor function
func NewUserRepo(db *sql.DB, configDetails utils.ConfigStruct) UserStorer {
	return &userRepo{DB: db, configDetails: configDetails}
}

// Creates a new user in DB
func (rd *userRepo) CreateUser(ctx context.Context, params CreateUserParams) error {
	// Attempt to insert a new user into the database
	_, err := rd.DB.Exec(userRegisterQuery, params.Username, params.Email, params.PasswordHash, params.FullName, params.DOB)
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrInsertUser, err)
	}

	// Retrieve the user object by email
	user, err := rd.GetUserByEmail(ctx, params.Email)
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrFindUserByEmail, err)
	}

	// Assign role to the user
	_, err = rd.DB.Exec(roleAssignmentQuery, user.ID, params.Role)
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrAssignRole, err)
	}

	// Update wallet_id in wallets table
	_, err = rd.DB.Exec(updateWalletIDQuery, params.WalletAddress, user.ID)
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrInsertWallet, err)
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
		return fmt.Errorf(utils.ErrorFormat, utils.ErrUpdatingLastLogin, err)
	}
	return nil
}

// UserExists checks if a user already exists based on username and email.
func (rd *userRepo) UserExists(ctx context.Context, userName, email string) (usernameAlreadyExists, emailAlreadyExists bool, err error) {

	// Check if username already exists
	err = rd.DB.QueryRow(usernameAlreadyInExistanceQuery, userName).Scan(&usernameAlreadyExists)
	if err != nil {
		return usernameAlreadyExists, emailAlreadyExists, fmt.Errorf(utils.ErrorFormat, utils.ErrCheckingUsername, err)
	}

	// Check if email already exists
	err = rd.DB.QueryRow(emailAlreadyInExistanceQuery, email).Scan(&emailAlreadyExists)
	if err != nil {
		return usernameAlreadyExists, emailAlreadyExists, fmt.Errorf(utils.ErrorFormat, utils.ErrCheckingEmail, err)
	}

	return usernameAlreadyExists, emailAlreadyExists, nil
}

// GetUserHighestRole fetches the highest role assigned to a user based on user_id.
func (rd *userRepo) GetUserHighestRole(ctx context.Context, userID string) (int, error) {

	var highestRoleLevel int

	// Query role assigned to the user.
	err := rd.DB.QueryRow(getUserRolesQuery, userID).Scan(&highestRoleLevel)
	if err != nil {
		return 0, fmt.Errorf(utils.ErrorFormat, utils.ErrFetchingRoles, err)
	}

	// Check if we found any roles, else return an error.
	if highestRoleLevel == 0 {
		return 0, fmt.Errorf(utils.ErrorFormat, utils.ErrNoRolesFound, err)
	}

	// Return the highest role ID.
	return highestRoleLevel, nil
}

// InsertKYCVerification inserts a new KYC verification record.
func (rd *userRepo) InsertKYCVerification(ctx context.Context, userID, documentType, documentNumber, verificationStatus string) (string, error) {
	var kycID string

	err := rd.DB.QueryRowContext(ctx, insertKYCVerificationQuery, userID, documentType, documentNumber, verificationStatus).Scan(&kycID)
	if err != nil {
		return "", fmt.Errorf(utils.ErrorFormat, utils.ErrInsertKYCVerification, err)
	}
	return kycID, nil
}

// GetAllKYCVerifications retrieves all KYC verification records.
func (rd *userRepo) GetAllKYCVerifications(ctx context.Context) ([]KYCRecord, error) {
	rows, err := rd.DB.QueryContext(ctx, getAllKYCVerificationsQuery)
	if err != nil {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrFetchKYCVerifications, err)
	}
	defer rows.Close()

	var records []KYCRecord
	for rows.Next() {
		var kycID, userID, documentType, documentNumber, verificationStatus, verifiedBy sql.NullString
		var submittedAt, verifiedAt sql.NullTime

		if err := rows.Scan(&kycID, &userID, &documentType, &documentNumber, &verificationStatus, &submittedAt, &verifiedAt, &verifiedBy); err != nil {
			return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrScanRow, err)
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
		return fmt.Errorf(utils.ErrorFormat, utils.ErrUpdateKYCVerification, err)
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
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrFetchKYCDetailedInfo, err)
	}
	defer rows.Close()

	var records []KYCRecord
	for rows.Next() {
		var kycID, userID, documentType, documentNumber, verificationStatus, verifiedBy sql.NullString
		var submittedAt, verifiedAt sql.NullTime

		// Scan the row into variables
		if err := rows.Scan(&kycID, &userID, &documentType, &documentNumber, &verificationStatus, &submittedAt, &verifiedAt, &verifiedBy); err != nil {
			return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrScanRow, err)
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
		return "", fmt.Errorf(utils.ErrorFormat, utils.ErrInsertRequestLog, err)
	}

	return logRequestID, nil
}

// UpdateRequestLog updates the request log entry with the given response status and response time.
func (rd *userRepo) UpdateRequestLog(ctx context.Context, requestID string, responseStatus, responseTimeMs int) error {
	// Execute the query to update the request log
	_, err := rd.DB.ExecContext(ctx, updateRequestLogQuery, responseStatus, responseTimeMs, requestID)
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrUpdateRequestLog, err)
	}

	return nil
}

// UpdateUserPasswordHash updates the password hash for a user based on their user ID.
func (rd *userRepo) UpdateUserPasswordHash(ctx context.Context, userID string, newPasswordHash string) error {
	// Execute the query to update the password hash
	_, err := rd.DB.ExecContext(ctx, updatePasswordHashQuery, newPasswordHash, userID)
	if err != nil {
		return fmt.Errorf(utils.ErrorFormat, utils.ErrUpdatePasswordHash, err)
	}

	return nil
}

func (rd *userRepo) GetRequestLogs(ctx context.Context, filter RequestLogFilter) ([]RequestLog, error) {
	var query string
	var args []interface{}

	// Start building the query
	query = getRequestLogsQuery

	// Add filters based on provided parameters
	argCount := 1 // Initialize the argument counter

	if filter.RequestID != "" {
		query += " AND request_id = $" + strconv.Itoa(argCount)
		args = append(args, filter.RequestID)
		argCount++
	} else {
		if filter.UserID != "" {
			query += " AND user_id = $" + strconv.Itoa(argCount)
			args = append(args, filter.UserID)
			argCount++
		}
		if filter.Endpoint != "" {
			query += " AND endpoint = $" + strconv.Itoa(argCount)
			args = append(args, filter.Endpoint)
			argCount++
		}
		if filter.HTTPMethod != "" {
			query += " AND http_method = $" + strconv.Itoa(argCount)
			args = append(args, filter.HTTPMethod)
			argCount++
		}
		if filter.ResponseStatus != 0 {
			query += " AND response_status = $" + strconv.Itoa(argCount)
			args = append(args, filter.ResponseStatus)
			argCount++
		}
		if filter.ResponseTimeMs != 0 {
			query += " AND response_time_ms = $" + strconv.Itoa(argCount)
			args = append(args, filter.ResponseTimeMs)
			argCount++
		}
		if filter.FromTime != nil {
			query += " AND created_at >= $" + strconv.Itoa(argCount)
			args = append(args, filter.FromTime)
			argCount++
		}
		if filter.ToTime != nil {
			query += " AND created_at <= $" + strconv.Itoa(argCount)
			args = append(args, filter.ToTime)
		}
	}

	query+=" ORDER BY created_at DESC LIMIT 100"

	// Execute the query
	rows, err := rd.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrRetrievingRequestLogs, err)
	}
	defer rows.Close()

	var logs []RequestLog
	for rows.Next() {
		var log RequestLog
		if err := rows.Scan(&log.RequestID, &log.UserID, &log.Endpoint, &log.HTTPMethod, &log.ResponseStatus, &log.ResponseTimeMs, &log.CreatedAt, &log.IPAddress); err != nil {
			return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrScanningRequestLog, err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrIteratingRequestLogs, err)
	}

	return logs, nil
}

func (rd *userRepo) GetRequestLogStats(ctx context.Context, filter RequestLogStatsFilter) (interface{}, error) {
	if filter.Column != "request_id" && filter.Column != "user_id" && filter.Column != "endpoint" &&
		filter.Column != "http_method" && filter.Column != "response_status" && filter.Column != "response_time_ms" {
		return nil, fmt.Errorf("invalid column name: %s", filter.Column)
	}

	query := "SELECT "
	var aggregates []string

	if filter.Count {
		aggregates = append(aggregates, "COUNT("+filter.Column+")")
	}
	if filter.Avg {
		aggregates = append(aggregates, "AVG("+filter.Column+")")
	}
	if filter.Unique {
		aggregates = append(aggregates, "COUNT(DISTINCT "+filter.Column+")")
	}
	if filter.Min {
		aggregates = append(aggregates, "MIN("+filter.Column+")")
	}
	if filter.Max {
		aggregates = append(aggregates, "MAX("+filter.Column+")")
	}
	if filter.Sum {
		aggregates = append(aggregates, "SUM("+filter.Column+")")
	}

	if len(aggregates) == 0 {
		return nil, fmt.Errorf("at least one aggregate function must be specified")
	}

	query += strings.Join(aggregates, ", ") + " FROM api_requests_log WHERE 1=1"

	if filter.FromTime != nil {
		query += " AND created_at >= $1"
	}
	if filter.ToTime != nil {
		query += " AND created_at <= $2"
	}

	args := []interface{}{}
	if filter.FromTime != nil {
		args = append(args, filter.FromTime)
	}
	if filter.ToTime != nil {
		args = append(args, filter.ToTime)
	}

	// Execute the query
	row := rd.DB.QueryRowContext(ctx, query, args...)
	var result interface{}
	if err := row.Scan(&result); err != nil {
		return nil, fmt.Errorf(utils.ErrorFormat, utils.ErrRetrievingRequestLogs, err)
	}

	return result, nil
}

func (rd *userRepo) GetUserInfo(ctx context.Context, userID string) (utils.UserInfo, error) {
	var user utils.UserInfo
	err := rd.DB.QueryRow(getUserInfoQuery, userID).Scan(&user.UserID, &user.Username, &user.FullName, &user.Email, &user.WalletID, &user.Role)
	return user, err
}

func (td *userRepo) GetTransactionStats(ctx context.Context, filter TransactionStatsFilter) (interface{}, error) {
	// Validate the column
	if filter.Column != "amount" && filter.Column != "status" && filter.Column != "fee" {
		return nil, fmt.Errorf("invalid column name: %s", filter.Column)
	}

	query := "SELECT "
	var aggregates []string

	// Handle aggregation based on the filter
	if filter.Count {
		aggregates = append(aggregates, "COUNT("+filter.Column+")")
	}
	if filter.Avg {
		aggregates = append(aggregates, "CAST(ROUND(AVG("+filter.Column+")/1e18, 2) AS FLOAT)")
	}
	if filter.Unique {
		aggregates = append(aggregates, "CAST(ROUND(COUNT(DISTINCT "+filter.Column+", 2)/1e18) AS FLOAT)")
	}
	if filter.Min {
		aggregates = append(aggregates, "CAST(ROUND(MIN("+filter.Column+")/1e18, 2) AS FLOAT)")
	}
	if filter.Max {
		aggregates = append(aggregates, "CAST(ROUND(MAX("+filter.Column+")/1e18, 2) AS FLOAT)")
	}
	if filter.Sum {
		aggregates = append(aggregates, "CAST(ROUND(SUM("+filter.Column+")/1e18, 2) AS FLOAT)")
	}

	// Ensure at least one aggregate function is specified
	if len(aggregates) == 0 {
		return nil, fmt.Errorf("at least one aggregate function must be specified")
	}

	// Base query
	query += strings.Join(aggregates, ", ") + " FROM transactions WHERE 1=1"

	argcount :=0

	// Add time range filter if provided
	if filter.FromTime != nil {
		argcount+=1
		query += " AND created_at >= " + "$" + strconv.Itoa(argcount)
	}
	if filter.ToTime != nil {
		argcount+=1
		query += " AND created_at <= " + "$" + strconv.Itoa(argcount)
	}

	// Add wallet ID filter based on sender or receiver
	if filter.WalletType == "sender" {
		query += " AND sender_wallet_id = " + "$" + strconv.Itoa(argcount+1)
	} else if filter.WalletType == "receiver" {
		query += " AND receiver_wallet_id = " + "$" + strconv.Itoa(argcount+1)
	}

	args := []interface{}{}
	// Append the time range filter arguments
	if filter.FromTime != nil {
		args = append(args, filter.FromTime)
	}
	if filter.ToTime != nil {
		args = append(args, filter.ToTime)
	}

	// Append the wallet ID filter argument
	if filter.WalletType == "sender" || filter.WalletType == "receiver" {
		args = append(args, filter.WalletID)
	}

	// Execute the query
	row := td.DB.QueryRowContext(ctx, query, args...)
	var result interface{}
	if err := row.Scan(&result); err != nil {
		return nil, fmt.Errorf("error retrieving transaction stats: %v", err)
	}

	return result, nil
}


func (td *userRepo) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
    // Execute the query
    rows, err := td.DB.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("error executing query: %v", err)
    }
    defer rows.Close()

    var results []map[string]interface{}

    cols, err := rows.Columns()
    if err != nil {
        return nil, fmt.Errorf("error getting columns: %v", err)
    }

    for rows.Next() {
        values := make([]interface{}, len(cols))
        valuePtrs := make([]interface{}, len(cols))

        for i := range values {
            valuePtrs[i] = &values[i]
        }

        if err := rows.Scan(valuePtrs...); err != nil {
            return nil, fmt.Errorf("error scanning row: %v", err)
        }

        rowMap := make(map[string]interface{})
        for i, colName := range cols {
            rowMap[colName] = values[i]
        }

        results = append(results, rowMap)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating rows: %v", err)
    }

    return results, nil
}
