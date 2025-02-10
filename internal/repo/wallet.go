package repo

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"time"

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
)

type WalletRepo struct {
	DB *sql.DB
}

type WalletStorer interface {
	GetWalletID(email, userID string) (string, error)
	UpdateWalletBalance(userID string, balance *big.Float) error
	InsertPrivateKey(userID, walletID, privateKey string) error
	RetrievePrivateKey(userID, walletID string) (string, error)
	AddTransaction(
		transactionID uuid.UUID, senderWalletID, receiverWalletID string,
		amount *big.Float,
		transactionType, status, transactionHash string,
		fee *big.Float,
	) (*Transaction, error)
	GetTransactionByID(transactionID uuid.UUID) (*Transaction, error)
	UpdateBalance(walletID string, balance *big.Float) error
	GetTransactions(
		transactionID uuid.UUID,
		senderWalletID string,
		receiverWalletID string,
		commonWalletID string,
		fromTime time.Time,
		toTime time.Time,
		page int,
		limit int,
	) ([]Transaction, error)
}

// Constructor function
func NewWalletRepo(db *sql.DB) WalletStorer {
	return &WalletRepo{DB: db}
}

// Returnes walletID from email or userID Precedance given to user_id if both parameters are passed
func (repoDep *WalletRepo) GetWalletID(email, userID string) (string, error) {
	var walletID string

	// Check if both parameters are empty
	if email == "" && userID == "" {
		return "", fmt.Errorf("both email and userID cannot be empty")
	}

	// If userID is provided (non-empty), prioritize that
	if userID != "" {
		log.Println("Using userID:", userID)
		err := repoDep.DB.QueryRow(getWalletIDFromUserIDQuery, userID).Scan(&walletID)
		if err != nil {
			log.Println("Error Retrieving wallet_id from user_id", err.Error())
			return "", fmt.Errorf("Error Retrieving wallet_id from user_id : %v", err.Error())
		}
	} else if email != "" {
		// If userID is not provided, fall back to email
		log.Println("Using email:", email)
		err := repoDep.DB.QueryRow(getWalletIDFromEmailQuery, email).Scan(&walletID)
		if err != nil {
			log.Println("Error Retrieving wallet_id from email",email, err.Error())
			return "", fmt.Errorf("Error Retrieving wallet_id from email : %v", err.Error())
		}
	}

	return walletID, nil
}

func (repoDep *WalletRepo) UpdateWalletBalance(userID string, balance *big.Float) error {
	balanceFloat64, _ := balance.Float64()

	result, err := repoDep.DB.Exec(updateWalletBalanceQuery, balanceFloat64, userID)
	if err != nil {
		log.Printf("Error executing Update Balance query: %v", err)
		return fmt.Errorf("error updating balance: %v", err)
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

	log.Print("Updated last balance successfully")
	return nil
}

const (
	encryptionKey = "your-32-bytelen-secret-key-here!" // 32 bytes for AES-256
)

// Function to ensure the encryption key is valid (16, 24, or 32 bytes)
func ensureValidKey(key string) ([]byte, error) {
	keyLength := len(key)
	if keyLength != 16 && keyLength != 24 && keyLength != 32 {
		log.Printf("Error: Invalid encryption key size: %d bytes\n", keyLength)
		if keyLength > 32 {
			key = key[:32] // Truncate to 32 bytes if the key is too long
		} else {
			// Pad the key with 0s if it's too short
			paddedKey := make([]byte, 32)
			copy(paddedKey, key)
			key = string(paddedKey)
		}
	}
	return []byte(key), nil
}

// Function to encrypt the private key
func encryptPrivateKey(privateKey string) (string, error) {
	log.Println("Encrypting private key...")

	// Ensure the encryption key is valid
	validKey, err := ensureValidKey(encryptionKey)
	if err != nil {
		log.Printf("Error: Invalid encryption key: %v\n", err)
		return "", err
	}

	// Check if the private key is empty
	if privateKey == "" {
		log.Println("Error: Provided private key is empty.")
		return "", fmt.Errorf("private key is empty")
	}

	block, err := aes.NewCipher(validKey)
	if err != nil {
		log.Printf("Error: Failed to create cipher: %v\n", err)
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	// Generate random IV (Initialization Vector)
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		log.Printf("Error: Failed to generate IV: %v\n", err)
		return "", fmt.Errorf("failed to generate IV: %v", err)
	}

	// Pad the private key to a multiple of AES block size
	paddedPrivateKey := pad([]byte(privateKey))

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

// Function to decrypt the private key
func decryptPrivateKey(encryptedKey string) (string, error) {
	log.Println("Decrypting private key...")

	// Ensure the encryption key is valid
	validKey, err := ensureValidKey(encryptionKey)
	if err != nil {
		log.Printf("Error: Invalid encryption key: %v\n", err)
		return "", err
	}

	// Check if the encrypted key is empty
	if encryptedKey == "" {
		log.Println("Error: Provided encrypted key is empty.")
		return "", fmt.Errorf("encrypted key is empty")
	}

	// Decode the base64 string
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		log.Printf("Error: Failed to decode base64 string: %v\n", err)
		return "", fmt.Errorf("failed to decode base64 string: %v", err)
	}

	// Ensure the encrypted data has the proper length (at least BlockSize + 1 byte for cipherText)
	if len(encryptedData) < aes.BlockSize {
		log.Println("Error: Encrypted data is too short.")
		return "", fmt.Errorf("encrypted data is too short")
	}

	// Extract the IV and cipherText from the encrypted data
	iv := encryptedData[:aes.BlockSize]
	cipherText := encryptedData[aes.BlockSize:]

	log.Printf("IV: %x\n", iv)
	log.Printf("CipherText: %x\n", cipherText)

	block, err := aes.NewCipher(validKey)
	if err != nil {
		log.Printf("Error: Failed to create cipher: %v\n", err)
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	// Decrypt the private key
	stream := cipher.NewCFBDecrypter(block, iv)
	decrypted := make([]byte, len(cipherText))
	stream.XORKeyStream(decrypted, cipherText)

	// Remove padding from the decrypted data
	decrypted = unpad(decrypted)
	log.Printf("Decrypted private key (after unpadding): %s\n", decrypted)

	return string(decrypted), nil
}

// Padding function to pad the private key to AES block size
func pad(data []byte) []byte {
	padding := aes.BlockSize - len(data)%aes.BlockSize
	padText := make([]byte, padding)
	for i := 0; i < padding; i++ {
		padText[i] = byte(padding)
	}
	return append(data, padText...)
}

// Unpadding function to remove padding from the decrypted private key
func unpad(data []byte) []byte {
	padding := int(data[len(data)-1])
	log.Printf("Unpadding data, padding byte: %d\n", padding)

	if padding > len(data) {
		log.Println("Error: Padding is larger than data length.")
		return nil
	}

	return data[:len(data)-padding]
}

// Function to insert the user_id, wallet_id, and encrypted private key into the database
func (repoDep *WalletRepo) InsertPrivateKey(userID, walletID, privateKey string) error {

	log.Println("Started Private key insertion")
	encryptedKey, err := encryptPrivateKey(privateKey)

	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %v", err)
	}

	// Prepare the SQL query to insert the data
	query := `INSERT INTO wallet_private_keys (user_id, wallet_id, private_key)
              VALUES ($1, $2, $3)`

	// Execute the insert query
	_, err = repoDep.DB.Exec(query, userID, walletID, encryptedKey)
	if err != nil {
		return fmt.Errorf("failed to execute insert query: %v", err)
	}

	return nil
}

// Function to retrieve the encrypted private key from the database using either userID or walletID
func (repoDep *WalletRepo) RetrievePrivateKey(userID, walletID string) (string, error) {
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
		return "", fmt.Errorf("either userID or walletID must be provided")
	}

	// Execute the query
	err := repoDep.DB.QueryRow(query, args...).Scan(&encryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve private key: %v", err)
	}

	// Decrypt the private key
	privateKey, err := decryptPrivateKey(encryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt private key: %v", err)
	}

	return privateKey, nil
}

// getTransactionByID fetches a transaction by its ID
func (repoDep *WalletRepo) GetTransactionByID(transactionID uuid.UUID) (*Transaction, error) {

	var transaction Transaction
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
		return nil, fmt.Errorf("failed to fetch transaction: %v", err)
	}

	return &transaction, nil
}

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

// AddTransaction inserts a new transaction into the transactions table and returns the inserted data.
func (repoDep *WalletRepo) AddTransaction(
	transactionID uuid.UUID, senderWalletID, receiverWalletID string,
	amount *big.Float,
	transactionType, status, transactionHash string,
	fee *big.Float,
) (*Transaction, error) {
	log.Println("Inserting new transaction into the database...")

	// Convert big.Float to float64 for database insertion
	amountFloat64, _ := amount.Float64()
	feeFloat64, _ := fee.Float64()

	// Execute the insert query
	_, err := repoDep.DB.Exec(
		addTransactionQuery,
		transactionID,
		senderWalletID,
		receiverWalletID,
		amountFloat64,
		transactionType,
		status,
		transactionHash,
		feeFloat64,
	)
	if err != nil {
		log.Printf("Error inserting transaction: %v", err)
		return nil, fmt.Errorf("failed to insert transaction: %v", err)
	}

	// Fetch the inserted transaction data
	insertedTransaction, err := repoDep.GetTransactionByID(transactionID)
	if err != nil {
		log.Printf("Error fetching inserted transaction: %v", err)
		return nil, fmt.Errorf("failed to fetch inserted transaction: %v", err)
	}

	log.Println("Transaction inserted successfully")
	return insertedTransaction, nil
}

// UpdateBalance updates the balance of a wallet in the database.
func (repoDep *WalletRepo) UpdateBalance(walletID string, balance *big.Float) error {
	log.Println("Updating wallet balance...")

	// Convert big.Float to string to maintain precision
	balanceStr := balance.Text('f', 20)

	// Prepare the update query
	updateBalanceQuery := `
		UPDATE wallets 
		SET balance = $1, last_updated = $2 
		WHERE wallet_id = $3;
	`

	// Execute the update query
	_, err := repoDep.DB.Exec(updateBalanceQuery, balanceStr, time.Now(), walletID)
	if err != nil {
		log.Printf("Error updating wallet balance: %v", err)
		return fmt.Errorf("failed to update wallet balance: %v", err)
	}

	log.Println("Wallet balance updated successfully")
	return nil
}

func (repo *WalletRepo) GetTransactions(
	transactionID uuid.UUID,
	senderWalletID string,
	receiverWalletID string,
	commonWalletID string,
	fromTime time.Time,
	toTime time.Time,
	page int,
	limit int,
) ([]Transaction, error) {

	// Default limit if not provided
	if limit <= 0 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	query := "SELECT transaction_id, sender_wallet_id, receiver_wallet_id, amount, transaction_type, status, transaction_hash, fee, created_at FROM transactions WHERE 1=1"
	var args []interface{}
	argIndex := 1

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
	// if fromTime != time.Now() {
	// 	query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
	// 	args = append(args, fromTime)
	// 	argIndex++
	// }
	// if toTime != time.Now() {
	// 	query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
	// 	args = append(args, toTime)
	// 	argIndex++
	// }

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	log.Println("Query:",query)
	rows, err := repo.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching transactions: %v", err)
		return nil, err
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
			log.Printf("Error scanning transaction row: %v", err)
			return nil, err
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
