package loan

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/CodeWithKrushnal/ChainBank/internal/repo"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
)

type service struct {
	userRepo   repo.UserStorer
	walletRepo repo.WalletStorer
	loanRepo   repo.LoanStorer
	ethRepo    ethereum.EthRepo
}

// Constructor function
func NewService(ctx context.Context, userRepo repo.UserStorer, walletRepo repo.WalletStorer, loanRepo repo.LoanStorer, ethRepo ethereum.EthRepo) Service {
	return service{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		loanRepo:   loanRepo,
		ethRepo:    ethRepo,
	}
}

// Add necesary method signature to be made accesible by service layer
type Service interface {
	CreateLoanOffer(ctx context.Context, lenderID string, amount float64, interestRate float64, termMonths int) (string, error)
	AcceptLoanOffer(ctx context.Context, offerID, borrowerID string) error
	RepayLoan(ctx context.Context, borrowerID, loanID string, amount float64) error
	TransferFunds(ctx context.Context, userID, recipientID, amountETH string) (string, *big.Int, error)
}

// CreateLoanOffer calls repo layer to insert a loan offer.
func (sd service) CreateLoanOffer(ctx context.Context, lenderID string, amount float64, interestRate float64, termMonths int) (string, error) {
	walletID, err := sd.walletRepo.GetWalletID("", lenderID)
	if err != nil {
		return "", err
	}
	balance, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(walletID), nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch balance: %w", err)
	}

	ethBalance, _ := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18)).Float64()

	if amount > ethBalance {
		return "", fmt.Errorf("Insufficient balance to create offer")
	}
	return sd.loanRepo.CreateLoanOffer(ctx, lenderID, amount, interestRate, termMonths)
}

// AcceptLoanOffer handles loan matching & fund transfer.
func (sd service) AcceptLoanOffer(ctx context.Context, offerID, borrowerID string) error {

	kycdetails, err := sd.userRepo.GetKYCDetailedInfo(ctx, "", borrowerID)
	if kycdetails[0]["verification_status"] != "Verified" {
		return fmt.Errorf("KYC Status is %v request could not be processed", kycdetails[0]["verification_status"])
	}
	if err != nil {
		log.Println("Error retriving kyc details")
	}

	// Fetch lender and loan amount from the repository
	lenderID, amount, err := sd.loanRepo.GetLoanOffer(ctx, offerID)
	if err != nil {
		return err
	}

	// Convert amount to string
	amountStr := strconv.FormatFloat(amount, 'f', -1, 64) // Converts float64 to string with full precision

	// Perform fund transfer from lender to borrower
	txHash, exactFee, err := sd.TransferFunds(ctx, lenderID, borrowerID, amountStr)
	if err != nil {
		return fmt.Errorf("fund transfer failed: %w", err)
	}

	// Update loan offer as accepted
	_, err = sd.loanRepo.AcceptLoanOffer(ctx, offerID, borrowerID, amount)
	if err != nil {
		return fmt.Errorf("failed to accept loan offer: %w", err)
	}

	log.Printf("Loan offer accepted: %s | Transaction: %s | Fee: %s", offerID, txHash, exactFee.String())
	return nil
}

// RepayLoan processes loan repayment.
func (sd service) RepayLoan(ctx context.Context, borrowerID, loanID string, amount float64) error {
	// Fetch lender details and remaining loan amount
	lenderID, remainingAmount, err := sd.loanRepo.GetLoanDetails(ctx, loanID, borrowerID)
	if err != nil {
		return err
	}

	// Validate repayment amount
	if amount > remainingAmount {
		return fmt.Errorf("repayment amount exceeds remaining loan")
	}

	// Convert amount to string
	amountStr := strconv.FormatFloat(amount, 'f', -1, 64) // Converts float64 to string with full precision

	// Perform fund transfer from borrower to lender
	txHash, exactFee, err := sd.TransferFunds(ctx, borrowerID, lenderID, amountStr)
	if err != nil {
		return fmt.Errorf("repayment transfer failed: %w", err)
	}

	// Update remaining loan balance
	newRemaining := remainingAmount - amount
	err = sd.loanRepo.UpdateLoanRepayment(ctx, loanID, newRemaining)
	if err != nil {
		return fmt.Errorf("failed to update loan repayment: %w", err)
	}

	log.Printf("Loan repayment successful: %s | Transaction: %s | Remaining Balance: %.2f | Fee: %s", loanID, txHash, newRemaining, exactFee.String())
	return nil
}

func (sd service) TransferFunds(ctx context.Context, userID, recipientID, amountETH string) (string, *big.Int, error) { // Return Transaction Hash and Exact Fee
	// Get sender and recipient wallet IDs
	senderWalletID, err := sd.walletRepo.GetWalletID("", userID)
	if err != nil {
		return "", nil, fmt.Errorf("sender wallet not found")
	}

	recipientWalletID, err := sd.walletRepo.GetWalletID("", recipientID)
	if err != nil {
		return "", nil, fmt.Errorf("recipient wallet not found")
	}

	// Retrieve sender's private key
	privateKeyHex, err := sd.walletRepo.RetrievePrivateKey(userID, senderWalletID)
	if err != nil {
		return "", nil, fmt.Errorf("error retrieving private key: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", nil, fmt.Errorf("invalid private key")
	}

	// Convert amount
	amount, success := new(big.Int).SetString(amountETH, 10)
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
