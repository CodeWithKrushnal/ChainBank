package wallet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"

	"golang.org/x/crypto/bcrypt"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
)

type service struct {
	userRepo   repo.UserStorer
	walletRepo repo.WalletStorer
	ethRepo    ethereum.EthRepo
}

type Service interface {
	GetWalletIDForUser(ctx context.Context, userInfo struct {
		UserID    string
		UserEmail string
		UserRole  int
	}, queryEmail, queryUserID string) (string, error)
	GetBalanceByWalletID(ctx context.Context, walletID string) (*big.Float, error)
	TransferFunds(ctx context.Context, userInfo struct {
		UserID    string
		UserEmail string
		UserRole  int
	}, req TransferRequest) (string, *big.Int, error)
	ValidateSenderAddress(ctx context.Context, senderWalletID string, privateKey *ecdsa.PrivateKey) error
	ValidateUserPassword(ctx context.Context, email, password string) error
	AddTransaction(
		ctx context.Context,
		transactionID uuid.UUID, senderWalletID, receiverWalletID string,
		amount *big.Float,
		transactionType, status, transactionHash string,
		fee *big.Float,
	) (*repo.Transaction, error)
	FetchTransactions(
		ctx context.Context,
		transactionID uuid.UUID,
		senderEmail string,
		receiverEmail string,
		commonEmail string,
		fromTime time.Time,
		toTime time.Time,
		page int,
		limit int,
	) ([]repo.Transaction, error)
}

// Constructor function
func NewService(ctx context.Context, userRepo repo.UserStorer, walletRepo repo.WalletStorer, ethRepo ethereum.EthRepo) Service {
	return service{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		ethRepo:    ethRepo,
	}
}

// GetWalletIDForUser retrieves the wallet ID based on user role and query params.
func (sd service) GetWalletIDForUser(ctx context.Context, userInfo struct {
	UserID    string
	UserEmail string
	UserRole  int
}, queryEmail, queryUserID string) (string, error) {
	if userInfo.UserRole == 3 && (queryUserID != "" || queryEmail != "") {
		return sd.walletRepo.GetWalletID(queryEmail, queryUserID)
	}
	return sd.walletRepo.GetWalletID(userInfo.UserEmail, userInfo.UserID)
}

// GetBalanceByWalletID retrieves the wallet balance from the blockchain.
func (sd service) GetBalanceByWalletID(ctx context.Context, walletID string) (*big.Float, error) {
	if !common.IsHexAddress(walletID) {
		return nil, fmt.Errorf("invalid wallet address")
	}

	balance, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(walletID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch balance: %w", err)
	}

	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))

	sd.walletRepo.UpdateBalance(walletID, ethBalance)
	return ethBalance, nil
}

// ValidateSenderAddress ensures the sender's wallet matches the derived address.
func (sd service) ValidateSenderAddress(ctx context.Context, senderWalletID string, privateKey *ecdsa.PrivateKey) error {
	senderAddress := common.HexToAddress(senderWalletID)
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	derivedAddress := crypto.PubkeyToAddress(*publicKey)

	if senderAddress != derivedAddress {
		return fmt.Errorf("unauthorized: sender wallet mismatch")
	}

	return nil
}

// ValidateUserPassword verifies the user's password.
func (sd service) ValidateUserPassword(ctx context.Context, email, password string) error {
	user, err := sd.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return fmt.Errorf("invalid password")
	}

	return nil
}

// GetTransactionByID retrieves a transaction by its ID.
func (sd service) GetTransactionByID(ctx context.Context, transactionID uuid.UUID) (*repo.Transaction, error) {
	// Call the repository method to fetch the transaction
	transaction, err := sd.GetTransactionByID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	return transaction, nil
}

// AddTransaction inserts a new transaction into the database and returns the inserted data.
func (sd service) AddTransaction(
	ctx context.Context,
	transactionID uuid.UUID, senderWalletID, receiverWalletID string,
	amount *big.Float,
	transactionType, status, transactionHash string,
	fee *big.Float,
) (*repo.Transaction, error) {
	// Call the repository method to add the transaction
	insertedTransaction, err := sd.walletRepo.AddTransaction(
		transactionID,
		senderWalletID,
		receiverWalletID,
		amount,
		transactionType,
		status,
		transactionHash,
		fee,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add transaction: %w", err)
	}

	return insertedTransaction, nil
}

func (sd service) TransferFunds(ctx context.Context, userInfo struct {
	UserID    string
	UserEmail string
	UserRole  int
}, req TransferRequest) (string, *big.Int, error) { // Return Transaction Hash and Exact Fee
	// Get sender and recipient wallet IDs
	senderWalletID, err := sd.walletRepo.GetWalletID(userInfo.UserEmail, userInfo.UserID)
	if err != nil {
		return "", nil, fmt.Errorf("sender wallet not found")
	}

	recipientWalletID, err := sd.walletRepo.GetWalletID(req.RecipientEmail,"")

	if err != nil {
		return "", nil, fmt.Errorf("recipient wallet not found")
	}

	// Validate user password
	err = sd.ValidateUserPassword(ctx, userInfo.UserEmail, req.Password)
	if err != nil {
		return "", nil, err
	}

	// Retrieve sender's private key
	privateKeyHex, err := sd.walletRepo.RetrievePrivateKey(userInfo.UserID, "")
	if err != nil {
		return "", nil, fmt.Errorf("error retrieving private key: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", nil, fmt.Errorf("invalid private key")
	}

	// Validate sender address
	if err := sd.ValidateSenderAddress(ctx, senderWalletID, privateKey); err != nil {
		return "", nil, err
	}

	// Convert amount
	amount, success := new(big.Int).SetString(req.AmountETH, 10)
	if !success {
		return "", nil, fmt.Errorf("invalid amount format")
	}

	// Set gas details and chain ID
	gasPrice := big.NewInt(20000000000) // 20 Gwei
	gasLimit := uint64(21000)
	chainID := big.NewInt(1337) // Ganache

	privateKeyHexStr := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))

	// Transfer funds
	signedTx, err := sd.ethRepo.TransferFunds(privateKeyHexStr, senderWalletID, recipientWalletID, amount, gasPrice, gasLimit, chainID)
	if err != nil {
		return "", nil, fmt.Errorf("transaction failed: %w", err)
	}

	// Send transaction
	err = ethereum.EthereumClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Get transaction receipt to fetch actual gas used
	txHash := signedTx.Hash().Hex()
	receipt, err := ethereum.EthereumClient.TransactionReceipt(ctx, signedTx.Hash())
	if err != nil {
		return txHash, nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Calculate exact transaction fee
	actualGasUsed := receipt.GasUsed
	exactFee := new(big.Int).Mul(big.NewInt(int64(actualGasUsed)), gasPrice) // exactFee = Gas Used * Gas Price

	// Convert amount to big.Float for database insertion
	amountFloat := new(big.Float).SetInt(amount)
	feeFloat := new(big.Float).SetInt(exactFee)

	// Add transaction to the database
	transactionID := uuid.New()
	transactionType := "transfer"
	status := "completed" // Assuming the transaction is successful at this point

	_, err = sd.walletRepo.AddTransaction(
		transactionID,
		senderWalletID,
		recipientWalletID,
		amountFloat,
		transactionType,
		status,
		txHash,
		feeFloat,
	)
	if err != nil {
		log.Printf("Failed to add transaction to database: %v", err)
		return txHash, exactFee, fmt.Errorf("failed to add transaction to database: %w", err)
	}

	balance1, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(senderWalletID), nil)
	if err != nil {
		log.Printf("failed to fetch balance: %w", err)
	}

	ethBalance1 := new(big.Float).Quo(new(big.Float).SetInt(balance1), big.NewFloat(1e18))
	sd.walletRepo.UpdateBalance(senderWalletID, ethBalance1)

	balance2, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(recipientWalletID), nil)
	if err != nil {
		log.Printf("failed to fetch balance: %w", err)
	}

	ethBalance2 := new(big.Float).Quo(new(big.Float).SetInt(balance2), big.NewFloat(1e18))

	sd.walletRepo.UpdateBalance(recipientWalletID, ethBalance2)

	log.Println("Transaction successfully added to the database")

	return txHash, exactFee, nil
}

func (sd service) FetchTransactions(
	ctx context.Context,
	transactionID uuid.UUID,
	senderEmail string,
	receiverEmail string,
	commonEmail string,
	fromTime time.Time,
	toTime time.Time,
	page int,
	limit int,
) ([]repo.Transaction, error) {

	var senderWalletID, receiverWalletID, commonWalletID string

	if senderEmail != "" {
		id, err := sd.walletRepo.GetWalletID(senderEmail, "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch sender wallet ID: %w", err)
		}
		senderWalletID = id
	}

	if receiverEmail != "" {
		id, err := sd.walletRepo.GetWalletID(receiverEmail, "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch receiver wallet ID: %w", err)
		}
		receiverWalletID = id
	}

	if commonEmail != "" {
		id, err := sd.walletRepo.GetWalletID(commonEmail, "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch comman wallet ID: %w", err)
		}
		commonWalletID = id
	}

	log.Println("sw: ", senderWalletID, "rw:", receiverWalletID, "com:", commonWalletID)

	transactions, err := sd.walletRepo.GetTransactions(
		uuid.Nil, senderWalletID, receiverWalletID, commonWalletID, fromTime, toTime, page, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	return transactions, nil
}
