package wallet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"

	"golang.org/x/crypto/bcrypt"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/CodeWithKrushnal/ChainBank/utils"
)

type service struct {
	userRepo   repo.UserStorer
	walletRepo repo.WalletStorer
	ethRepo    ethereum.EthRepo
}

type Service interface {
	GetWalletIDForUser(ctx context.Context, userInfo utils.User, queryEmail, queryUserID string) (string, error)
	GetBalanceByWalletID(ctx context.Context, walletID string) (*big.Float, error)
	TransferFunds(ctx context.Context, userInfo utils.User, req TransferRequest) (repo.Transaction, *big.Int, error)
	ValidateSenderAddress(ctx context.Context, senderWalletID string, privateKey *ecdsa.PrivateKey) error
	ValidateUserPassword(ctx context.Context, email, password string) error
	AddTransaction(ctx context.Context, transactionID uuid.UUID, senderWalletID, receiverWalletID string, amount *big.Float, transactionType, status, transactionHash string, fee *big.Float) (repo.Transaction, error)
	FetchTransactions(ctx context.Context, transactionID uuid.UUID, senderEmail string, receiverEmail string, commonEmail string, fromTime time.Time, toTime time.Time, page int, limit int) ([]repo.Transaction, error)
	GetUserByID(ctx context.Context, userID string) (utils.User, error)
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
func (sd service) GetWalletIDForUser(ctx context.Context, userInfo utils.User, queryEmail, queryUserID string) (string, error) {
	if userInfo.UserRole == 3 && (queryUserID != "" || queryEmail != "") {
		walletID, err := sd.walletRepo.GetWalletID(ctx, queryEmail, queryUserID)
		if err != nil {
			return "", fmt.Errorf("%s: %w", utils.ErrFetchingWalletID, err)
		}
		return walletID, nil
	}

	walletID, err := sd.walletRepo.GetWalletID(ctx, userInfo.UserEmail, userInfo.UserID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", utils.ErrFetchingWalletID, err)
	}
	return walletID, nil
}

// GetBalanceByWalletID retrieves the wallet balance from the blockchain. It returns the balance in ETH as a big.Float.
func (sd service) GetBalanceByWalletID(ctx context.Context, walletID string) (*big.Float, error) {
	// Validate the wallet address format
	if !common.IsHexAddress(walletID) {
		return nil, fmt.Errorf("%s: %w", utils.ErrInvalidWalletAddress, utils.ErrInvalidInput)
	}

	// Fetch the balance from the Ethereum client
	balance, err := ethereum.EthereumClient.BalanceAt(ctx, common.HexToAddress(walletID), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrFetchBalance, err)
	}

	// Convert the balance from wei to ETH
	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))

	// Update the wallet balance in the repository
	if err := sd.walletRepo.UpdateBalance(ctx, walletID, ethBalance); err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrUpdatingBalance, err)
	}

	return ethBalance, nil
}

// ValidateSenderAddress ensures the sender's wallet matches the derived address.
func (sd service) ValidateSenderAddress(ctx context.Context, senderWalletID string, privateKey *ecdsa.PrivateKey) error {

	// Convert the sender wallet ID to an Ethereum address
	senderAddress := common.HexToAddress(senderWalletID)

	// Derive the address from the public key
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	derivedAddress := crypto.PubkeyToAddress(*publicKey)

	// Check if the derived address matches the sender's address
	if senderAddress != derivedAddress {
		return fmt.Errorf("%s: %w", utils.ErrUnauthorizedSenderAddress, utils.ErrInvalidSenderAddress)
	}

	return nil
}

// ValidateUserPassword verifies the user's password against the stored hash.
func (sd service) ValidateUserPassword(ctx context.Context, email, password string) error {
	// Retrieve the user by email
	user, err := sd.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("%s: %w", utils.ErrUserNotFound, err)
	}

	// Compare the provided password with the stored password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return fmt.Errorf("%s: %w", utils.ErrInvalidPassword, err)
	}

	return nil
}

// GetTransactionByID retrieves a transaction by its ID from the repository.
func (sd service) GetTransactionByID(ctx context.Context, transactionID uuid.UUID) (repo.Transaction, error) {
	// Fetch the transaction from the repository
	transaction, err := sd.walletRepo.GetTransactionByID(ctx, transactionID)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrFetchingTransaction, err)
	}

	return transaction, nil
}

// AddTransaction inserts a new transaction into the database and returns the inserted data.
func (sd service) AddTransaction(ctx context.Context, transactionID uuid.UUID, senderWalletID, receiverWalletID string, amount *big.Float, transactionType, status, transactionHash string, fee *big.Float) (repo.Transaction, error) {

	// Attempt to add the transaction using the repository method
	insertedTransaction, err := sd.walletRepo.AddTransaction(ctx, transactionID, senderWalletID, receiverWalletID, amount, transactionType, status, transactionHash, fee)
	if err != nil {
		// Propagate the error with a standard error message
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrAddingTransaction, err)
	}

	// Return the successfully inserted transaction
	return insertedTransaction, nil
}

// TransferFunds handles the transfer of funds between two wallets.
func (sd service) TransferFunds(ctx context.Context, userInfo utils.User, req TransferRequest) (repo.Transaction, *big.Int, error) {
	// Get sender wallet ID
	senderWalletID, err := sd.walletRepo.GetWalletID(ctx, userInfo.UserEmail, userInfo.UserID)
	if err != nil {
		return repo.Transaction{}, nil, fmt.Errorf("%s: %w", utils.ErrSenderWalletNotFound, err)
	}

	// Get recipient wallet ID
	recipientWalletID, err := sd.walletRepo.GetWalletID(ctx, req.RecipientEmail, "")
	if err != nil {
		return repo.Transaction{}, nil, fmt.Errorf("%s: %w", utils.ErrRecipientWalletNotFound, err)
	}

	// Validate user password
	if err := sd.ValidateUserPassword(ctx, userInfo.UserEmail, req.Password); err != nil {
		return repo.Transaction{}, nil, err
	}

	// Retrieve sender's private key
	privateKeyHex, err := sd.walletRepo.RetrievePrivateKey(ctx, userInfo.UserID, "")
	if err != nil {
		return repo.Transaction{}, nil, fmt.Errorf("%s: %w", utils.ErrRetrievingPrivateKey, err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return repo.Transaction{}, nil, fmt.Errorf("%s: %w", utils.ErrInvalidPrivateKey, err)
	}

	// Validate sender address
	if err := sd.ValidateSenderAddress(ctx, senderWalletID, privateKey); err != nil {
		return repo.Transaction{}, nil, err
	}

	// Convert amount from string to big.Int
	amount, success := new(big.Int).SetString(req.AmountETH, 10)
	if !success {
		return repo.Transaction{}, nil, fmt.Errorf("%s: %w", utils.ErrInvalidAmountFormat, err)
	}

	// Set gas details and chain ID
	gasPrice := big.NewInt(20000000000) // 20 Gwei
	gasLimit := uint64(21000)
	chainID := big.NewInt(1337) // Ganache

	privateKeyHexStr := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))

	// Transfer funds
	signedTx, err := sd.ethRepo.TransferFunds(privateKeyHexStr, senderWalletID, recipientWalletID, amount, gasPrice, gasLimit, chainID)
	if err != nil {
		return repo.Transaction{}, nil, fmt.Errorf("%s: %w", utils.ErrTransactionFailed, err)
	}

	// Send transaction
	if err := ethereum.EthereumClient.SendTransaction(context.Background(), signedTx); err != nil {
		return repo.Transaction{}, nil, fmt.Errorf("%s: %w", utils.ErrFailedToBroadcastTransaction, err)
	}

	// Get transaction receipt to fetch actual gas used
	txHash := signedTx.Hash().Hex()
	receipt, err := ethereum.EthereumClient.TransactionReceipt(ctx, signedTx.Hash())
	if err != nil {
		return repo.Transaction{}, nil, fmt.Errorf("%s: %w", utils.ErrFailedToGetTransactionReceipt, err)
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

	transaction, err := sd.walletRepo.AddTransaction(
		ctx,
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
		return repo.Transaction{}, exactFee, fmt.Errorf("%s: %w", utils.ErrFailedToAddTransactionToDB, err)
	}

	// Update sender's balance
	balance1, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(senderWalletID), nil)
	if err != nil {
		return repo.Transaction{}, exactFee, fmt.Errorf("%s: %w", utils.ErrFailedToFetchBalance, err)
	}
	ethBalance1 := new(big.Float).Quo(new(big.Float).SetInt(balance1), big.NewFloat(1e18))
	if err := sd.walletRepo.UpdateBalance(ctx, senderWalletID, ethBalance1); err != nil {
		return repo.Transaction{}, exactFee, fmt.Errorf("%s: %w", utils.ErrFailedToUpdateWalletBalance, err)
	}

	// Update recipient's balance
	balance2, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(recipientWalletID), nil)
	if err != nil {
		return repo.Transaction{}, exactFee, fmt.Errorf("%s: %w", utils.ErrFailedToFetchBalance, err)
	}
	ethBalance2 := new(big.Float).Quo(new(big.Float).SetInt(balance2), big.NewFloat(1e18))
	if err := sd.walletRepo.UpdateBalance(ctx, recipientWalletID, ethBalance2); err != nil {
		return repo.Transaction{}, exactFee, fmt.Errorf("%s: %w", utils.ErrFailedToUpdateWalletBalance, err)
	}

	return transaction, exactFee, nil
}

// FetchTransactions retrieves a list of transactions based on the provided filters.
func (sd service) FetchTransactions(ctx context.Context, transactionID uuid.UUID, senderEmail string, receiverEmail string, commonEmail string, fromTime time.Time, toTime time.Time, page int, limit int) ([]repo.Transaction, error) {
	var senderWalletID, receiverWalletID, commonWalletID string

	// Retrieve sender wallet ID if provided
	if senderEmail != "" {
		id, err := sd.walletRepo.GetWalletID(ctx, senderEmail, "")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", utils.ErrFetchingWalletID, err)
		}
		senderWalletID = id
	}

	// Retrieve receiver wallet ID if provided
	if receiverEmail != "" {
		id, err := sd.walletRepo.GetWalletID(ctx, receiverEmail, "")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", utils.ErrFetchingWalletID, err)
		}
		receiverWalletID = id
	}

	// Retrieve common wallet ID if provided
	if commonEmail != "" {
		id, err := sd.walletRepo.GetWalletID(ctx, commonEmail, "")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", utils.ErrFetchingWalletID, err)
		}
		commonWalletID = id
	}

	// Fetch transactions based on the retrieved wallet IDs and other filters
	transactions, err := sd.walletRepo.GetTransactions(ctx, uuid.Nil, senderWalletID, receiverWalletID, commonWalletID, fromTime, toTime, page, limit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrFetchingTransactions, err)
	}
	return transactions, nil
}

// GetUserByID retrieves a user by their unique user ID and returns the user details.
func (sd service) GetUserByID(ctx context.Context, userID string) (utils.User, error) {
	// Fetch detailed user information from the repository
	detailedUser, err := sd.userRepo.GetuserByID(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf("%s: %w", utils.ErrNoUserFound, err)
	}

	// Fetch the highest role of the user
	role, err := sd.userRepo.GetUserHighestRole(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf("%s: %w", utils.ErrFetchingRoles, err)
	}

	// Return the user details including ID, email, and role
	return utils.User{UserID: detailedUser.ID, UserEmail: detailedUser.Email, UserRole: role}, nil
}
