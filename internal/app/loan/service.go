package loan

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/CodeWithKrushnal/ChainBank/internal/app/ethereum"
	"github.com/CodeWithKrushnal/ChainBank/utils"
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
	CreateLoanapplication(ctx context.Context, borrowerID string, amount float64, interestRate float64, termMonths int) (repo.Loanapplication, error)
	CreateLoanOffer(ctx context.Context, lenderID string, amount, interestRate float64, termMonths int, applicationID string) (repo.LoanOffer, error)
	GetLoanapplications(ctx context.Context, applicationID string, borrowerID string, status string) ([]repo.Loanapplication, error)
	GetLoanOffers(ctx context.Context, offerID string, applicationID string, lenderID string, status string) ([]repo.LoanOffer, error)
	GetLoanDetails(ctx context.Context, loanID, offerID, borrowerID, lenderID, status, applicationID string) ([]repo.Loan, error)
	GetUserByID(ctx context.Context, userID string) (utils.User, error)
	AcceptOffer(ctx context.Context, offerID, borrowerID string) (repo.LoanOffer, error)
	DisburseLoan(ctx context.Context, lenderID, offerID string) (repo.Loan, error)
	CalculateTotalPayable(ctx context.Context, loanID, userID string) (PayableBreakdown, error)
	SettleLoan(ctx context.Context, userID, loanID string) (repo.Loan, error)
}

// structs

type PayableBreakdown struct {
	LoanID       string  `json:"loan_id"`
	Principal    float64 `json:"principal"`
	Interest     float64 `json:"interest"`
	Fees         float64 `json:"fees"`
	Penalty      float64 `json:"penalty"`
	TotalPayable float64 `json:"total_payable"`
}

//Service Functions

func (sd service) TransferFunds(ctx context.Context, userID, recipientID, amountETH string) (repo.Transaction, error) {
	// Return Transaction Hash and Exact Fee

	// Get sender and recipient wallet IDs
	senderWalletID, err := sd.walletRepo.GetWalletID("", userID)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("sender wallet not found")
	}

	recipientWalletID, err := sd.walletRepo.GetWalletID("", recipientID)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("recipient wallet not found")
	}

	// Retrieve sender's private key
	privateKeyHex, err := sd.walletRepo.RetrievePrivateKey(userID, senderWalletID)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("error retrieving private key: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("invalid private key")
	}

	// Convert amount
	amount, success := new(big.Int).SetString(amountETH, 10)
	if !success {
		return repo.Transaction{}, fmt.Errorf("invalid amount format")
	}

	// Set gas details and chain ID
	gasPrice := big.NewInt(20000000000) // 20 Gwei
	gasLimit := uint64(21000)
	chainID := big.NewInt(1337) // Ganache

	privateKeyHexStr := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))

	// Transfer funds
	signedTx, err := sd.ethRepo.TransferFunds(privateKeyHexStr, senderWalletID, recipientWalletID, amount, gasPrice, gasLimit, chainID)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("transaction failed: %w", err)
	}

	// Send transaction
	err = ethereum.EthereumClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Get transaction receipt to fetch actual gas used
	txHash := signedTx.Hash().Hex()
	receipt, err := ethereum.EthereumClient.TransactionReceipt(ctx, signedTx.Hash())
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Calculate exact transaction fee
	actualGasUsed := receipt.GasUsed
	exactFee := new(big.Int).Mul(big.NewInt(int64(actualGasUsed)), gasPrice) // exactFee = Gas Used * Gas Price

	// Convert amount to big.Float for database insertion
	amountFloat := new(big.Float).SetInt(amount)
	feeFloat := new(big.Float).SetInt(exactFee)

	// Add transaction to the database
	transactionID := uuid.New()
	transactionType := "Debt"
	status := "completed" // Assuming the transaction is successful at this point

	transaction, err := sd.walletRepo.AddTransaction(transactionID, senderWalletID, recipientWalletID, amountFloat, transactionType, status, txHash, feeFloat)
	if err != nil {
		log.Printf("Failed to add transaction to database: %v", err)
		return repo.Transaction{}, fmt.Errorf("failed to add transaction to database: %w", err)
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

	return transaction, nil
}

// CreateLoanapplication creates a new loan application.
func (sd service) CreateLoanapplication(ctx context.Context, borrowerID string, amount float64, interestRate float64, termMonths int) (repo.Loanapplication, error) {

	borrowerIsVerified, err := sd.loanRepo.IsKYCVerified(ctx, borrowerID)

	if err != nil {
		log.Println("Error Checking the KYC Vefication status of borrower", err.Error())
		return repo.Loanapplication{}, fmt.Errorf("Error Checking the KYC Vefication status of borrower")
	}

	if !borrowerIsVerified {
		return repo.Loanapplication{}, fmt.Errorf("User is not KYC Veried")
	}

	if borrowerID == "" || amount <= 0 || interestRate <= 0 || termMonths <= 0 {
		return repo.Loanapplication{}, fmt.Errorf("invalid input parameters")
	}

	createdLoan, err := sd.loanRepo.CreateLoanapplication(ctx, borrowerID, amount, interestRate, termMonths)
	if err != nil {
		return repo.Loanapplication{}, err
	}

	return createdLoan, nil
}

// CreateLoanOffer creates a new loan offer.
func (sd service) CreateLoanOffer(ctx context.Context, lenderID string, amount, interestRate float64, termMonths int, applicationID string) (repo.LoanOffer, error) {
	lenderIsVerified, err := sd.loanRepo.IsKYCVerified(ctx, lenderID)

	if err != nil {
		log.Println("Error Checking the KYC Vefication status of borrower", err.Error())
		return repo.LoanOffer{}, fmt.Errorf("Error Checking the KYC Vefication status of borrower")
	}

	if !lenderIsVerified {
		return repo.LoanOffer{}, fmt.Errorf("User is not KYC Veried")
	}

	if lenderID == "" || amount <= 0 || interestRate <= 0 || termMonths <= 0 || applicationID == "" {
		return repo.LoanOffer{}, fmt.Errorf("invalid input parameters")
	}

	createdOffer, err := sd.loanRepo.CreateLoanOffer(ctx, lenderID, amount, interestRate, termMonths, applicationID)

	if err != nil {
		return repo.LoanOffer{}, err
	}

	return createdOffer, nil
}

// GetLoanapplications fetches Loan applications based on either application_id or borrower_id or status, clubbing borrower_id and status is allowed
func (sd service) GetLoanapplications(ctx context.Context, applicationID string, borrowerID string, status string) ([]repo.Loanapplication, error) {
	return sd.loanRepo.GetLoanapplications(ctx, applicationID, borrowerID, status)
}

// GetLoanOffers fetches Loan Offers based on either offerID or applicationID or lenderID or status, clubbing lenderID and status is allowed
func (sd service) GetLoanOffers(ctx context.Context, offerID string, applicationID string, lenderID string, status string) ([]repo.LoanOffer, error) {
	return sd.loanRepo.GetLoanOffers(ctx, offerID, applicationID, lenderID, status)
}

// GetLoanDetails fetches Loan Details based on either loanID or offerID or borrowerID, or lenderID or status, clubbing lenderID and status is allowed
func (sd service) GetLoanDetails(ctx context.Context, loanID, offerID, borrowerID, lenderID, status, applicationID string) ([]repo.Loan, error) {
	return sd.loanRepo.GetLoanDetails(ctx, loanID, offerID, borrowerID, lenderID, status, applicationID)
}

func (sd service) GetUserByID(ctx context.Context, userID string) (utils.User, error) {
	detailedUser, err := sd.userRepo.GetuserByID(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf("Error Fetching the User from DB", err.Error())
	}
	role, err := sd.userRepo.GetUserHighestRole(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf("Error Etching the role from DB", err.Error())
	}
	return utils.User{UserID: detailedUser.ID, UserEmail: detailedUser.Email, UserRole: role}, nil
}

func (sd service) AcceptOffer(ctx context.Context, offerID, borrowerID string) (repo.LoanOffer, error) {
	loan, err := sd.loanRepo.AcceptLoanOffer(ctx, offerID, borrowerID)
	if err != nil {
		log.Println("Error Encountered in Accepting loan Offer: ", err.Error())
		return repo.LoanOffer{}, err
	}
	return loan, nil
}

func (sd service) DisburseLoan(ctx context.Context, lenderID, offerID string) (repo.Loan, error) {
	offer, err := sd.loanRepo.GetLoanOffers(ctx, offerID, "", "", "")
	if err != nil {
		log.Println("Error Retriving the offer", err.Error())
		return repo.Loan{}, err
	}

	application, err := sd.GetLoanapplications(ctx, offer[0].ApplicationID.String(), "", "")
	if err != nil {
		log.Println("Error Retriving the application", err.Error())
		return repo.Loan{}, err
	}

	amountStr := strconv.FormatFloat(offer[0].Amount, 'f', -1, 64)

	transaction, err := sd.TransferFunds(ctx, offer[0].LenderID.String(), application[0].BorrowerID.String(), amountStr)
	if err != nil {
		log.Println("Error in Amount Transfer", err.Error())
		return repo.Loan{}, err
	}

	log.Printf("Transaction completed", transaction)

	nextPaymentDate := time.Now().AddDate(0, offer[0].LoanTermMonths, 0)

	loan, err := sd.loanRepo.DisburseLoan(ctx, offer[0].OfferID.String(), application[0].BorrowerID.String(), offer[0].LenderID.String(), application[0].ApplicationID.String(), offer[0].Amount, offer[0].InterestRate, nextPaymentDate, transaction.TransactionID.String())
	if err != nil {
		log.Println("Error in Disbursing the loan", err.Error())
		return repo.Loan{}, err
	}

	return loan, nil
}

func (sd service) CalculateTotalPayable(ctx context.Context, loanID, userID string) (PayableBreakdown, error) {
	var loan repo.Loan
	var totalPayable float64
	var penalty float64

	// Fetch loan details
	loanDetails, err := sd.loanRepo.GetLoanDetails(ctx, loanID, "", "", "", "", "")
	if err != nil {
		log.Println("Error Retrieving loan details:", err.Error())
		return PayableBreakdown{}, err
	}

	if len(loanDetails) == 0 {
		return PayableBreakdown{}, fmt.Errorf("loan not found")
	}

	loan = loanDetails[0]

	// Check if user is either borrower or lender
	if loan.BorrowerID != userID && loan.LenderID != userID {
		return PayableBreakdown{}, fmt.Errorf("user is neither borrower nor lender")
	}

	// Calculate interest till current date
	startDate, _ := time.Parse(time.RFC3339, loan.StartDate)
	timeSinceStart := time.Since(startDate)
	if timeSinceStart < 0 {
		timeSinceStart = 0
	}
	daysElapsed := float64(timeSinceStart.Hours() / 24)
	yearlyInterest := loan.TotalPrinciple * loan.InterestRate / 100 // Yearly interest
	interest := yearlyInterest * (daysElapsed / 365)                // Prorated interest for days elapsed

	// Calculate penalty if current date exceeds next payment date
	nextPaymentDate, _ := time.Parse(time.RFC3339, loan.NextPaymentDate)
	if time.Now().After(nextPaymentDate) {
		monthsOverdue := int(time.Since(nextPaymentDate).Hours() / 24 / 30)
		penalty = (loan.TotalPrinciple * loan.InterestRate / 100 / 12) * float64(monthsOverdue) * 0.10 // 10% of the monthly interest
	}

	fees := 0.0

	// Total payable calculation
	totalPayable = loan.TotalPrinciple + interest + fees + penalty

	return PayableBreakdown{
		LoanID:       loan.LoanID,
		Principal:    loan.TotalPrinciple,
		Interest:     interest,
		Fees:         fees,
		Penalty:      penalty,
		TotalPayable: totalPayable,
	}, nil
}

func (sd service) SettleLoan(ctx context.Context, userID, loanID string) (repo.Loan, error) {
	// Fetch loan details
	loanDetails, err := sd.loanRepo.GetLoanDetails(ctx, loanID, "", "", "", "", "")
	if err != nil {
		log.Println("Error Retrieving loan details:", err.Error())
		return repo.Loan{}, err
	}

	if len(loanDetails) == 0 {
		return repo.Loan{}, fmt.Errorf("loan not found")
	}

	loan := loanDetails[0]

	// Check if the user is the borrower
	if loan.BorrowerID != userID {
		return repo.Loan{}, fmt.Errorf("user is not the borrower of this loan")
	}

	// Calculate total payable amount
	payableBreakdown, err := sd.CalculateTotalPayable(ctx, loan.LoanID, userID)
	if err != nil {
		return repo.Loan{}, fmt.Errorf("failed to calculate total payable: %w", err)
	}

	// Initiate payment for TotalPayable
	transaction, err := sd.TransferFunds(ctx, userID, loan.LenderID, strconv.FormatFloat(payableBreakdown.TotalPayable, 'f', 2, 64))
	if err != nil {
		return repo.Loan{}, fmt.Errorf("failed to transfer funds: %w", err)
	}

	// Call SettleLoan function to update the database
	settledLoan, err := sd.loanRepo.SettleLoan(ctx, loan.LoanID, payableBreakdown.TotalPayable, 0, transaction.TransactionID.String())
	if err != nil {
		return repo.Loan{}, fmt.Errorf("failed to settle loan: %w", err)
	}

	return settledLoan, nil
}
