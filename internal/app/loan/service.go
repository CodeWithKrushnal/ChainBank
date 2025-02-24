package loan

import (
	"context"
	"fmt"
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

// TransferFunds transfers funds between wallets
func (sd service) TransferFunds(ctx context.Context, userID, recipientID, amountETH string) (repo.Transaction, error) {
	// Get sender and recipient wallet IDs
	senderWalletID, err := sd.walletRepo.GetWalletID(ctx, "", userID)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrSenderWalletNotFound, err)
	}

	recipientWalletID, err := sd.walletRepo.GetWalletID(ctx, "", recipientID)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrRecipientWalletNotFound, err)
	}

	// Retrieve sender's private key
	privateKeyHex, err := sd.walletRepo.RetrievePrivateKey(ctx, userID, senderWalletID)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrRetrievingPrivateKey, err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrInvalidPrivateKey, err)
	}

	// Convert amount
	amount, success := new(big.Int).SetString(amountETH, 10)
	if !success {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrInvalidAmountFormat, err)
	}

	// Set gas details and chain ID
	gasPrice := big.NewInt(20000000000) // 20 Gwei
	gasLimit := uint64(21000)
	chainID := big.NewInt(1337) // Ganache

	privateKeyHexStr := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))

	// Transfer funds
	signedTx, err := sd.ethRepo.TransferFunds(privateKeyHexStr, senderWalletID, recipientWalletID, amount, gasPrice, gasLimit, chainID)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrTransactionFailed, err)
	}

	// Send transaction
	err = ethereum.EthereumClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrFailedToBroadcastTransaction, err)
	}

	// Get transaction receipt to fetch actual gas used
	txHash := signedTx.Hash().Hex()
	receipt, err := ethereum.EthereumClient.TransactionReceipt(ctx, signedTx.Hash())
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrFailedToGetTransactionReceipt, err)
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

	transaction, err := sd.walletRepo.AddTransaction(ctx, transactionID, senderWalletID, recipientWalletID, amountFloat, transactionType, status, txHash, feeFloat)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrAddingTransactionFailed, err)
	}

	// Update sender's balance
	balance1, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(senderWalletID), nil)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrFetchingBalanceFailed, err)
	}
	ethBalance1 := new(big.Float).Quo(new(big.Float).SetInt(balance1), big.NewFloat(1e18))
	sd.walletRepo.UpdateBalance(ctx, senderWalletID, ethBalance1)

	// Update recipient's balance
	balance2, err := ethereum.EthereumClient.BalanceAt(context.Background(), common.HexToAddress(recipientWalletID), nil)
	if err != nil {
		return repo.Transaction{}, fmt.Errorf("%s: %w", utils.ErrFetchingBalanceFailed, err)
	}
	ethBalance2 := new(big.Float).Quo(new(big.Float).SetInt(balance2), big.NewFloat(1e18))
	sd.walletRepo.UpdateBalance(ctx, recipientWalletID, ethBalance2)

	return transaction, nil
}

// CreateLoanapplication creates a new loan application for a verified borrower.
func (sd service) CreateLoanapplication(ctx context.Context, borrowerID string, amount float64, interestRate float64, termMonths int) (repo.Loanapplication, error) {
	// Check if the borrower is KYC verified
	borrowerIsVerified, err := sd.loanRepo.IsKYCVerified(ctx, borrowerID)
	if err != nil {
		return repo.Loanapplication{}, fmt.Errorf("%s: %w", utils.ErrKYCVerificationFailed, err)
	}

	if !borrowerIsVerified {
		return repo.Loanapplication{}, fmt.Errorf("%s", utils.ErrUserNotVerified)
	}

	// Validate input parameters
	if borrowerID == "" || amount <= 0 || interestRate <= 0 || termMonths <= 0 {
		return repo.Loanapplication{}, fmt.Errorf("%s", utils.ErrInvalidInput)
	}

	// Create the loan application
	createdLoan, err := sd.loanRepo.CreateLoanapplication(ctx, borrowerID, amount, interestRate, termMonths)
	if err != nil {
		return repo.Loanapplication{}, fmt.Errorf("%s: %w", utils.ErrCreatingLoanApplication, err)
	}

	return createdLoan, nil
}

// CreateLoanOffer creates a new loan offer. It checks if the lender is KYC verified and validates input parameters before creating the loan offer.
func (sd service) CreateLoanOffer(ctx context.Context, lenderID string, amount, interestRate float64, termMonths int, applicationID string) (repo.LoanOffer, error) {
	// Check if the lender is KYC verified
	lenderIsVerified, err := sd.loanRepo.IsKYCVerified(ctx, lenderID)
	if err != nil {
		return repo.LoanOffer{}, fmt.Errorf("%s: %w", utils.ErrKYCVerificationFailed, err)
	}

	if !lenderIsVerified {
		return repo.LoanOffer{}, fmt.Errorf("%s", utils.ErrUserNotKYCVerified)
	}

	// Validate input parameters
	if lenderID == "" || amount <= 0 || interestRate <= 0 || termMonths <= 0 || applicationID == "" {
		return repo.LoanOffer{}, fmt.Errorf("%s", utils.ErrInvalidInputParameters)
	}

	// Create the loan offer
	createdOffer, err := sd.loanRepo.CreateLoanOffer(ctx, lenderID, amount, interestRate, termMonths, applicationID)
	if err != nil {
		return repo.LoanOffer{}, fmt.Errorf("%s: %w", utils.ErrCreatingLoanOffer, err)
	}

	return createdOffer, nil
}

// GetLoanapplications fetches Loan applications based on either application_id or borrower_id or status, clubbing borrower_id and status is allowed
func (sd service) GetLoanapplications(ctx context.Context, applicationID string, borrowerID string, status string) ([]repo.Loanapplication, error) {
	// Fetch loan applications from the repository
	loanApplications, err := sd.loanRepo.GetLoanapplications(ctx, applicationID, borrowerID, status)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrFetchingLoanApplications, err)
	}
	return loanApplications, nil
}

// GetLoanOffers fetches Loan Offers based on either offerID or applicationID or lenderID or status, clubbing lenderID and status is allowed
func (sd service) GetLoanOffers(ctx context.Context, offerID string, applicationID string, lenderID string, status string) ([]repo.LoanOffer, error) {
	// Fetch loan offers from the repository
	loanOffers, err := sd.loanRepo.GetLoanOffers(ctx, offerID, applicationID, lenderID, status)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrFetchingLoanOffers, err)
	}

	return loanOffers, nil
}

// GetLoanDetails fetches Loan Details based on either loanID or offerID or borrowerID, or lenderID or status, clubbing lenderID and status is allowed
func (sd service) GetLoanDetails(ctx context.Context, loanID, offerID, borrowerID, lenderID, status, applicationID string) ([]repo.Loan, error) {
	// Fetch loan details from the repository
	loans, err := sd.loanRepo.GetLoanDetails(ctx, loanID, offerID, borrowerID, lenderID, status, applicationID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrFetchingLoanDetails, err)
	}

	return loans, nil
}

// GetUserByID retrieves a user by their ID from the repository.
func (sd service) GetUserByID(ctx context.Context, userID string) (utils.User, error) {
	// Fetch detailed user information from the repository
	detailedUser, err := sd.userRepo.GetuserByID(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf("%s: %w", utils.ErrFetchingUserFromDB, err)
	}

	// Fetch the highest role of the user
	role, err := sd.userRepo.GetUserHighestRole(ctx, userID)
	if err != nil {
		return utils.User{}, fmt.Errorf("%s: %w", utils.ErrFetchingUserRoleFromDB, err)
	}

	// Return the user details along with their role
	return utils.User{UserID: detailedUser.ID, UserEmail: detailedUser.Email, UserRole: role}, nil
}

// AcceptOffer processes the acceptance of a loan offer by the borrower.
func (sd service) AcceptOffer(ctx context.Context, offerID, borrowerID string) (repo.LoanOffer, error) {
	loan, err := sd.loanRepo.AcceptLoanOffer(ctx, offerID, borrowerID)
	if err != nil {
		return repo.LoanOffer{}, fmt.Errorf("%s: %w", utils.ErrAcceptingLoanOffer, err)
	}
	return loan, nil
}

// DisburseLoan processes the disbursement of a loan to the borrower.
func (sd service) DisburseLoan(ctx context.Context, lenderID, offerID string) (repo.Loan, error) {
	// Fetch the loan offer based on the offerID
	offer, err := sd.loanRepo.GetLoanOffers(ctx, offerID, "", "", "")
	if err != nil {
		return repo.Loan{}, fmt.Errorf("%s: %w", utils.ErrRetrievingOffer, err)
	}

	// Fetch the loan application associated with the offer
	application, err := sd.GetLoanapplications(ctx, offer[0].ApplicationID.String(), "", "")
	if err != nil {
		return repo.Loan{}, fmt.Errorf("%s: %w", utils.ErrRetrievingApplication, err)
	}

	// Prepare the amount for transfer
	amountStr := strconv.FormatFloat(offer[0].Amount, 'f', -1, 64)

	// Transfer funds from lender to borrower
	transaction, err := sd.TransferFunds(ctx, offer[0].LenderID.String(), application[0].BorrowerID.String(), amountStr)
	if err != nil {
		return repo.Loan{}, fmt.Errorf("%s: %w", utils.ErrTransferFunds, err)
	}

	// Calculate the next payment date
	nextPaymentDate := time.Now().AddDate(0, offer[0].LoanTermMonths, 0)

	// Disburse the loan to the borrower
	loan, err := sd.loanRepo.DisburseLoan(ctx, offer[0].OfferID.String(), application[0].BorrowerID.String(), offer[0].LenderID.String(), application[0].ApplicationID.String(), offer[0].Amount, offer[0].InterestRate, nextPaymentDate, transaction.TransactionID.String())
	if err != nil {
		return repo.Loan{}, fmt.Errorf("%s: %w", utils.ErrDisbursingLoan, err)
	}

	return loan, nil
}

// CalculateTotalPayable calculates the total amount payable for a loan by the user.
func (sd service) CalculateTotalPayable(ctx context.Context, loanID, userID string) (PayableBreakdown, error) {
	var loan repo.Loan
	var totalPayable float64
	var penalty float64

	// Fetch loan details
	loanDetails, err := sd.loanRepo.GetLoanDetails(ctx, loanID, "", "", "", "", "")
	if err != nil {
		return PayableBreakdown{}, fmt.Errorf("%s: %w", utils.ErrFetchingLoanDetails, err)
	}

	if len(loanDetails) == 0 {
		return PayableBreakdown{}, fmt.Errorf("%s", utils.ErrLoanDetailsNotFound)
	}

	loan = loanDetails[0]

	// Check if user is either borrower or lender
	if loan.BorrowerID != userID && loan.LenderID != userID {
		return PayableBreakdown{}, fmt.Errorf("%s", utils.ErrUserNotBorrowerOrLender)
	}

	// Calculate interest till current date
	startDate, err := time.Parse(time.RFC3339, loan.StartDate)
	if err != nil {
		return PayableBreakdown{}, fmt.Errorf("%s: %w", utils.ErrInvalidStartDateFormat, err)
	}
	timeSinceStart := time.Since(startDate)
	if timeSinceStart < 0 {
		timeSinceStart = 0
	}
	daysElapsed := float64(timeSinceStart.Hours() / 24)
	yearlyInterest := loan.TotalPrinciple * loan.InterestRate / 100 // Yearly interest
	interest := yearlyInterest * (daysElapsed / 365)                // Prorated interest for days elapsed

	// Calculate penalty if current date exceeds next payment date
	nextPaymentDate, err := time.Parse(time.RFC3339, loan.NextPaymentDate)
	if err != nil {
		return PayableBreakdown{}, fmt.Errorf("%s: %w", utils.ErrInvalidNextPaymentDateFormat, err)
	}
	if time.Now().After(nextPaymentDate) {
		monthsOverdue := int(time.Since(nextPaymentDate).Hours() / 24 / 30)
		penalty = (loan.TotalPrinciple * loan.InterestRate / 100 / 12) * float64(monthsOverdue) * 0.10 // 10% of the monthly interest
	}

	fees := 0.0 // Placeholder for any additional fees

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
		return repo.Loan{}, fmt.Errorf("%s: %w", utils.ErrFetchingLoanDetails, err)
	}

	if len(loanDetails) == 0 {
		return repo.Loan{}, fmt.Errorf("%s", utils.ErrLoanDetailsNotFound)
	}

	loan := loanDetails[0]

	// Check if the user is the borrower
	if loan.BorrowerID != userID {
		return repo.Loan{}, fmt.Errorf("%s", utils.ErrUserNotBorrower)
	}

	// Calculate total payable amount
	payableBreakdown, err := sd.CalculateTotalPayable(ctx, loan.LoanID, userID)
	if err != nil {
		return repo.Loan{}, fmt.Errorf("%s: %w", utils.ErrCalculatingTotalPayable, err)
	}

	// Initiate payment for TotalPayable
	transaction, err := sd.TransferFunds(ctx, userID, loan.LenderID, strconv.FormatFloat(payableBreakdown.TotalPayable, 'f', 2, 64))
	if err != nil {
		return repo.Loan{}, fmt.Errorf("%s: %w", utils.ErrTransferFunds, err)
	}

	// Call SettleLoan function to update the database
	settledLoan, err := sd.loanRepo.SettleLoan(ctx, loan.LoanID, payableBreakdown.TotalPayable, 0, transaction.TransactionID.String())
	if err != nil {
		return repo.Loan{}, fmt.Errorf("%s: %w", utils.ErrSettlingLoan, err)
	}

	return settledLoan, nil
}
