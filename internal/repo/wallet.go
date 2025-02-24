package repo

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/CodeWithKrushnal/ChainBank/utils"
	"github.com/google/uuid"
)

const (
	getWalletIDFromUserIDQuery          = `SELECT wallet_id FROM wallets WHERE user_id = $1`
	getWalletIDFromEmailQuery           = `SELECT w.wallet_id FROM wallets w INNER JOIN users u on w.user_id = u.user_id WHERE u.email = $1`
	updateWalletBalanceQuery            = `UPDATE wallets SET balance =$1 WHERE user_id= $2`
	retrievePrivateKeyFromUserIDQuery   = `SELECT private_key FROM wallet_private_keys WHERE user_id = $1`
	retrievePrivateKeyFromWalletIDQuery = `SELECT private_key FROM wallet_private_keys WHERE wallet_id = $1`
	getTransactionByIDQuery             = `SELECT transaction_id, sender_wallet_id, receiver_wallet_id, amount, transaction_type, status, transaction_hash, fee, created_at FROM transactions WHERE transaction_id = $1`
	addTransactionQuery                 = `INSERT INTO transactions (transaction_id, sender_wallet_id, receiver_wallet_id, amount, transaction_type, status, transaction_hash, fee) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	updateBalanceQuery                  = `UPDATE wallets SET balance = $1, last_updated = $2 WHERE wallet_id = $3;`
	InsertPrivateKeyQuery               = `INSERT INTO wallet_private_keys (user_id, wallet_id, private_key) VALUES ($1, $2, $3)`
	GetTransactionsQuery                = `SELECT transaction_id, sender_wallet_id, receiver_wallet_id, amount, transaction_type, status, transaction_hash, fee, created_at FROM transactions WHERE 1=1`
)

const (
	encryptionKey = "your-32-bytelen-secret-key-here!"
)

// Transaction represents a row in the transactions table
type Transaction struct {
	TransactionID    uuid.UUID `json:"transaction_id"`
	SenderWalletID   string    `json:"sender_wallet_id"`
	ReceiverWalletID string    `json:"receiver_wallet_id"`
	Amount           float64   `json:"amount"`
	TransactionType  string    `json:"transaction_type"`
	Status           string    `json:"status"`
	TransactionHash  string    `json:"transaction_hash"`
	Fee              float64   `json:"fee"`
	CreatedAt        time.Time `json:"created_at"`
}

type WalletRepo struct {
	DB *sql.DB
}

type WalletStorer interface {
	GetWalletID(ctx context.Context, email, userID string) (string, error)
	UpdateWalletBalance(ctx context.Context, userID string, balance *big.Float) error
	InsertPrivateKey(ctx context.Context, userID, walletID, privateKey string) error
	RetrievePrivateKey(ctx context.Context, userID, walletID string) (string, error)
	AddTransaction(ctx context.Context, transactionID uuid.UUID, senderWalletID, receiverWalletID string, amount *big.Float, transactionType, status, transactionHash string, fee *big.Float) (Transaction, error)
	GetTransactionByID(ctx context.Context, transactionID uuid.UUID) (Transaction, error)
	UpdateBalance(ctx context.Context, walletID string, balance *big.Float) error
	GetTransactions(ctx context.Context, transactionID uuid.UUID, senderWalletID string, receiverWalletID string, commonWalletID string, fromTime time.Time, toTime time.Time, page int, limit int) ([]Transaction, error)
}

// Constructor function
func NewWalletRepo(db *sql.DB) WalletStorer {
	return &WalletRepo{DB: db}
}

// GetWalletID retrieves the wallet ID based on the provided email or userID. It prioritizes userID if both are provided.
func (repoDep *WalletRepo) GetWalletID(ctx context.Context, email, userID string) (string, error) {
	var walletID string

	// Check if both parameters are empty
	if email == "" && userID == "" {
		return "", utils.ErrBothParamsEmpty
	}

	// If userID is provided (non-empty), prioritize that
	if userID != "" {
		err := repoDep.DB.QueryRow(getWalletIDFromUserIDQuery, userID).Scan(&walletID)
		if err != nil {
			return "", fmt.Errorf("%s: %w", utils.ErrRetrievingWalletIDFromUserID, err)
		}
	} else if email != "" {
		// If userID is not provided, fall back to email
		err := repoDep.DB.QueryRow(getWalletIDFromEmailQuery, email).Scan(&walletID)
		if err != nil {
			return "", fmt.Errorf("%s: %w", utils.ErrRetrievingWalletIDFromEmail, err)
		}
	}

	return walletID, nil
}

// UpdateWalletBalance updates the balance of a wallet in the database. It takes the userID and the new balance as parameters. Returns an error if the update fails or if no user is found.
func (repoDep *WalletRepo) UpdateWalletBalance(ctx context.Context, userID string, balance *big.Float) error {
	balanceFloat64, _ := balance.Float64()

	// Execute the update query
	result, err := repoDep.DB.Exec(updateWalletBalanceQuery, balanceFloat64, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrUpdateWalletBalance, err)
	}

	// Check if any row was affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrCheckAffectedRows, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%w: %s", utils.ErrNoUserFound, userID)
	}

	slog.Info("Wallet balance updated successfully", "userID", userID)

	return nil
}

// ensureValidKey validates the encryption key size and adjusts it to 32 bytes if necessary. It returns the valid key as a byte slice or an error if the key size is invalid.
func ensureValidKey(key string) ([]byte, error) {
	keyLength := len(key)
	if keyLength != 16 && keyLength != 24 && keyLength != 32 {
		if keyLength > 32 {
			slog.Warn(utils.ErrInvalidEncryptionKeySize.Error(), "keyLength", keyLength, "message", utils.ErrKeyTooLong)
			key = key[:32] // Truncate to 32 bytes if the key is too long
		} else {
			slog.Warn(utils.ErrInvalidEncryptionKeySize.Error(), "keyLength", keyLength, "message", utils.ErrKeyTooShort)
			// Pad the key with 0s if it's too short
			paddedKey := make([]byte, 32)
			copy(paddedKey, key)
			key = string(paddedKey)
		}
	}
	return []byte(key), nil
}

// encryptPrivateKey encrypts the private key using AES-256-CFB encryption. It returns the encrypted private key as a base64 encoded string or an error if the encryption fails.
func encryptPrivateKey(privateKey string) (string, error) {
	// Ensure the encryption key is valid
	validKey, err := ensureValidKey(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrInvalidEncryptionKeySize, err)
	}

	// Check if the private key is empty
	if privateKey == "" {
		return "", fmt.Errorf("%s: %w", utils.ErrEmptyPrivateKey, err)
	}

	block, err := aes.NewCipher(validKey)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrCipherCreationError, err)
	}

	// Generate random IV (Initialization Vector)
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrIVGenerationError, err)
	}

	// Pad the private key to a multiple of AES block size
	paddedPrivateKey, err := pad([]byte(privateKey))
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrPaddingFailed, err)
	}

	// Encrypt the private key
	cipherText := make([]byte, len(paddedPrivateKey))
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText, paddedPrivateKey)

	// Combine the IV and cipherText (IV comes first for later decryption)
	result := append(iv, cipherText...)

	// Return the result as a base64 encoded string
	encodedResult := base64.StdEncoding.EncodeToString(result)

	return encodedResult, nil
}

// decryptPrivateKey decrypts the encrypted private key using AES-256-CFB decryption. It returns the decrypted private key as a string or an error if the decryption fails.
func decryptPrivateKey(encryptedKey string) (string, error) {
	// Ensure the encryption key is valid
	validKey, err := ensureValidKey(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrInvalidEncryptionKey, err)
	}

	// Check if the encrypted key is empty
	if encryptedKey == "" {
		return "", fmt.Errorf("%s: %w", utils.ErrEmptyEncryptedKey, err)
	}

	// Decode the base64 string
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrDecodingBase64String, err)
	}

	// Ensure the encrypted data has the proper length (at least BlockSize + 1 byte for cipherText)
	if len(encryptedData) < aes.BlockSize {
		return "", fmt.Errorf("%s: %w", utils.ErrEncryptedDataTooShort, err)
	}

	// Extract the IV and cipherText from the encrypted data
	iv := encryptedData[:aes.BlockSize]
	cipherText := encryptedData[aes.BlockSize:]

	block, err := aes.NewCipher(validKey)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrDecryptingPrivateKey.Error(), err)
	}

	// Decrypt the private key
	stream := cipher.NewCFBDecrypter(block, iv)
	decrypted := make([]byte, len(cipherText))
	stream.XORKeyStream(decrypted, cipherText)

	// Remove padding from the decrypted data
	decrypted = unpad(decrypted)

	return string(decrypted), nil
}

// pad adds padding to the data to make its length a multiple of the AES block size.
func pad(data []byte) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrPaddingFailed, utils.ErrNilData)
	}

	padding := aes.BlockSize - len(data)%aes.BlockSize
	padText := make([]byte, padding)
	for i := 0; i < padding; i++ {
		padText[i] = byte(padding)
	}
	return append(data, padText...), nil
}

// Unpadding function to remove padding from the decrypted private key
func unpad(data []byte) []byte {
	padding := int(data[len(data)-1])

	if padding > len(data) {
		return nil
	}

	return data[:len(data)-padding]
}

// InsertPrivateKey inserts the user_id, wallet_id, and encrypted private key into the database.
func (repoDep *WalletRepo) InsertPrivateKey(ctx context.Context, userID, walletID, privateKey string) error {
	// Encrypt the private key
	encryptedKey, err := encryptPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrEncryptingPrivateKey, err)
	}

	// Execute the insert query
	_, err = repoDep.DB.Exec(InsertPrivateKeyQuery, userID, walletID, encryptedKey)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrExecutingInsertQuery, err)
	}

	return nil
}

// RetrievePrivateKey retrieves the encrypted private key from the database using either userID or walletID.
func (repoDep *WalletRepo) RetrievePrivateKey(ctx context.Context, userID, walletID string) (string, error) {
	var encryptedKey string

	// Prepare the SQL query based on the available parameter (userID or walletID)
	var query string
	var args []interface{}

	if userID != "" {
		query = retrievePrivateKeyFromUserIDQuery
		args = append(args, userID)
	} else if walletID != "" {
		query = retrievePrivateKeyFromWalletIDQuery
		args = append(args, walletID)
	} else {
		return "", fmt.Errorf("%s: %w", utils.ErrMissingParameters, utils.ErrInvalidInput)
	}

	// Execute the query
	if err := repoDep.DB.QueryRow(query, args...).Scan(&encryptedKey); err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrRetrievePrivateKey, err)
	}

	// Decrypt the private key
	privateKey, err := decryptPrivateKey(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrDecryptingPrivateKey, err)
	}

	return privateKey, nil
}

// GetTransactionByID retrieves a transaction by its unique transaction ID.
func (repoDep *WalletRepo) GetTransactionByID(ctx context.Context, transactionID uuid.UUID) (Transaction, error) {
	var transaction Transaction

	// Execute the query to fetch the transaction details
	err := repoDep.DB.QueryRow(getTransactionByIDQuery, transactionID).Scan(
		&transaction.TransactionID,
		&transaction.SenderWalletID,
		&transaction.ReceiverWalletID,
		&transaction.Amount,
		&transaction.TransactionType,
		&transaction.Status,
		&transaction.TransactionHash,
		&transaction.Fee,
		&transaction.CreatedAt,
	)
	if err != nil {
		return Transaction{}, fmt.Errorf("%s: %w", utils.ErrRetrieveTransaction, err)
	}

	return transaction, nil
}

// AddTransaction inserts a new transaction into the transactions table and returns the inserted data.
func (repoDep *WalletRepo) AddTransaction(ctx context.Context, transactionID uuid.UUID, senderWalletID, receiverWalletID string, amount *big.Float, transactionType, status, transactionHash string, fee *big.Float) (Transaction, error) {
	// Log the start of the transaction insertion
	slog.Info(utils.LogTransactionInsertion)

	// Convert big.Float to float64 for database insertion
	amountFloat64, _ := amount.Float64()
	feeFloat64, _ := fee.Float64()

	// Execute the insert query
	_, err := repoDep.DB.Exec(addTransactionQuery, transactionID, senderWalletID, receiverWalletID, amountFloat64, transactionType, status, transactionHash, feeFloat64)
	if err != nil {
		return Transaction{}, fmt.Errorf("%s: %w", utils.ErrInsertingTransaction, err)
	}

	// Fetch the inserted transaction data
	insertedTransaction, err := repoDep.GetTransactionByID(ctx, transactionID)
	if err != nil {
		return Transaction{}, fmt.Errorf("%s: %w", utils.ErrFetchingInsertedTransaction, err)
	}

	// Log the successful insertion of the transaction
	slog.Info(utils.LogTransactionInsertionSuccess)
	return insertedTransaction, nil
}

// UpdateBalance updates the balance of a wallet in the database.
func (repoDep *WalletRepo) UpdateBalance(ctx context.Context, walletID string, balance *big.Float) error {
	slog.Info(utils.LogUpdatingWalletBalance)

	// Convert big.Float to string to maintain precision
	balanceStr := balance.Text('f', 20)

	// Execute the update query
	_, err := repoDep.DB.Exec(updateBalanceQuery, balanceStr, time.Now(), walletID)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrUpdatingWalletBalance, err)
	}

	slog.Info(utils.LogWalletBalanceUpdatedSuccessfully)
	return nil
}

// GetTransactions retrieves a list of transactions based on various filters.
func (repo *WalletRepo) GetTransactions(ctx context.Context, transactionID uuid.UUID, senderWalletID string, receiverWalletID string, commonWalletID string, fromTime time.Time, toTime time.Time, page int, limit int) ([]Transaction, error) {
	const defaultLimit = 100

	// Set default limit and page if not provided
	if limit <= 0 {
		limit = defaultLimit
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	query := GetTransactionsQuery
	var args []interface{}
	argIndex := 1

	// Build the query with provided filters
	if transactionID != uuid.Nil {
		query += fmt.Sprintf(" AND transaction_id = $%d", argIndex)
		args = append(args, transactionID)
		argIndex++
	}
	if senderWalletID != "" {
		query += fmt.Sprintf(" AND sender_wallet_id = $%d", argIndex)
		args = append(args, senderWalletID)
		argIndex++
	}
	if receiverWalletID != "" {
		query += fmt.Sprintf(" AND receiver_wallet_id = $%d", argIndex)
		args = append(args, receiverWalletID)
		argIndex++
	}
	if commonWalletID != "" {
		query += fmt.Sprintf(" AND (sender_wallet_id = $%d OR receiver_wallet_id = $%d)", argIndex, argIndex+1)
		args = append(args, commonWalletID, commonWalletID)
		argIndex += 2
	}
	if !fromTime.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, fromTime)
		argIndex++
	}
	if !toTime.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, toTime)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute the query
	rows, err := repo.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrFetchingTransactions, err)
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var tx Transaction
		var transactionHash sql.NullString
		if err := rows.Scan(
			&tx.TransactionID, &tx.SenderWalletID, &tx.ReceiverWalletID, &tx.Amount,
			&tx.TransactionType, &tx.Status, &transactionHash, &tx.Fee, &tx.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", utils.ErrScanningTransactionRow, err)
		}
		if transactionHash.Valid {
			tx.TransactionHash = transactionHash.String
		} else {
			tx.TransactionHash = ""
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}
