package repo

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type loanRepo struct {
	DB *sql.DB
}

// Constructor function
func NewLoanRepo(db *sql.DB) LoanStorer {
	return &loanRepo{DB: db}
}

type LoanStorer interface {
	CreateLoanOffer(ctx context.Context, lenderID string, amount, interestRate float64, termMonths int, applicationID string) (LoanOffer, error)
	AcceptLoanOffer(ctx context.Context, offerID, borrowerID string) (LoanOffer, error)
	GetLoanDetails(ctx context.Context, loanID, offerID, borrowerID, lenderID, status, applicationID string) ([]Loan, error)
	UpdateLoanRepayment(ctx context.Context, loanID string, newRemaining float64) error
	CreateLoanapplication(ctx context.Context, borrowerID string, amount, interestRate float64, termMonths int) (Loanapplication, error)
	GetLoanapplications(ctx context.Context, applicationID string, borrowerID string, status string) ([]Loanapplication, error)
	GetLoanOffers(ctx context.Context, offerID string, applicationID string, lenderID string, status string) ([]LoanOffer, error)
	IsKYCVerified(ctx context.Context, userID string) (bool, error)
	DisburseLoan(ctx context.Context, offerID, borrowerID, lenderID, applicationID string, totalPrinciple, interestRate float64, nextPaymentDate time.Time, DisbursementTransactionID string) (Loan, error)
	SettleLoan(ctx context.Context, loanID string, settledAmount, accruedInterest float64, settlementTransactionID string) (Loan, error)
}

// All Loan Queries
const (
	createLoanOfferQuery               = `INSERT INTO loan_offers (offer_id, lender_id, amount, interest_rate, loan_term_months, status, application_id, created_at) VALUES ($1, $2, $3, $4, $5, 'Open', $6, NOW()) RETURNING offer_id, lender_id, amount, interest_rate, loan_term_months, status, created_at, application_id`
	DisburseLoanQuery                  = `INSERT INTO loans (loan_id, offer_id, borrower_id, total_principle, remaining_principle, interest_rate, lender_id, application_id, status, start_date, next_payment_date, disbursement_transaction_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', NOW(), $9, $10) RETURNING loan_id, offer_id, borrower_id, lender_id, total_principle, remaining_principle, status, start_date, next_payment_date, application_id, interest_rate, disbursement_transaction_id`
	acceptLoanOfferStatusUpdationQuery = `UPDATE loan_offers SET status = 'Accepted' WHERE offer_id = $1 RETURNING offer_id, lender_id, amount, interest_rate, loan_term_months, status, created_at, application_id`
	getLoanDetailsQuery                = `SELECT loan_id, offer_id, borrower_id, lender_id, total_principle, remaining_principle, status, start_date, next_payment_date, application_id, interest_rate, settled_amount, settlement_date, accrued_interest FROM loans WHERE 1=1`
	updateLoanRepaymentQuery           = `UPDATE loans SET remaining_principle = $1, status = $2, WHERE loan_id = $3`
	createLoanapplicationQuery         = `INSERT INTO loan_applications (application_id, borrower_id, amount, interest_rate, term_months, status) VALUES ($1, $2, $3, $4, $5, 'open') RETURNING application_id, borrower_id, amount, interest_rate, term_months, status, created_at, updated_at`
	getLoanOffersQuery                 = `SELECT offer_id, lender_id, amount, interest_rate, loan_term_months, status, created_at, application_id FROM loan_offers WHERE 1=1`
	getLoanapplicationsQuery           = `SELECT application_id, borrower_id, amount, interest_rate, term_months, status, created_at, updated_at FROM loan_applications WHERE 1=1`
	settleLoanQuery                    = `UPDATE loans SET settled_amount = $1, accrued_interest = $2, settlement_date = NOW(), remaining_principle = 0, status = 'closed' WHERE loan_id = $3 RETURNING loan_id, offer_id, borrower_id, lender_id, total_principle, remaining_principle, status, start_date, next_payment_date, application_id, interest_rate, settled_amount, settlement_date, accrued_interest`
)

// Structs

// Loan offers Struct
type LoanOffer struct {
	OfferID        uuid.UUID `db:"offer_id"`
	LenderID       uuid.UUID `db:"lender_id"`
	Amount         float64   `db:"amount"`
	InterestRate   float64   `db:"interest_rate"`
	LoanTermMonths int       `db:"loan_term_months"`
	Status         string    `db:"status"`
	CreatedAt      time.Time `db:"created_at"`
	ApplicationID  uuid.UUID `db:"application_id"`
}

// Loan Struct
type LoanDetails struct {
	LoanID          uuid.UUID `db:"loan_id"`
	OfferID         uuid.UUID `db:"offer_id"`
	BorrowerID      uuid.UUID `db:"borrower_id"`
	LenderID        uuid.UUID `db:"lender_id"`
	TotalAmount     float64   `db:"total_principle"`
	RemainingAmount float64   `db:"remaining_principle"`
	Status          string    `db:"status"`
	StartDate       time.Time `db:"start_date"`
	NextPaymentDate time.Time `db:"next_payment_date"`
	ApplicationID   uuid.UUID `db:"application_id"`
}

// Loan applications Struct
type Loanapplication struct {
	ApplicationID uuid.UUID `json:"application_id"`
	BorrowerID    uuid.UUID `json:"borrower_id"`
	Amount        float64   `json:"amount"`
	InterestRate  float64   `json:"interest_rate"`
	TermMonths    int       `json:"term_months"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Loan struct {
	LoanID                    string  `json:"loan_id"`
	OfferID                   string  `json:"offer_id"`
	BorrowerID                string  `json:"borrower_id"`
	LenderID                  string  `json:"lender_id"`
	TotalPrinciple            float64 `json:"total_principle"`
	RemainingPrinciple        float64 `json:"remaining_principle"`
	Status                    string  `json:"status"`
	StartDate                 string  `json:"start_date"`
	NextPaymentDate           string  `json:"next_payment_date"`
	ApplicationID             string  `json:"application_id"`
	InterestRate              float64 `json:"interest_rate"`
	SettledAmount             float64 `json:"settled_amount"`
	SettlementDate            string  `json:"settlement_date"`
	AccruedInterest           float64 `json:"accrued_interest"`
	DisbursementTransactionID string  `json:"disbursement_transaction_id"`
	SettlementTransactionID   string  `json:"settlement_transaction_id"`
}

// CreateLoanOffer creates a new loan offer and returns the created LoanOffer
func (rd *loanRepo) CreateLoanOffer(ctx context.Context, lenderID string, amount, interestRate float64, termMonths int, applicationID string) (LoanOffer, error) {
	offerID := uuid.New()

	// Execute the query and scan the result into a LoanOffer struct
	var loanOffer LoanOffer
	err := rd.DB.QueryRowContext(ctx, createLoanOfferQuery, offerID, lenderID, amount, interestRate, termMonths, applicationID).Scan(
		&loanOffer.OfferID,
		&loanOffer.LenderID,
		&loanOffer.Amount,
		&loanOffer.InterestRate,
		&loanOffer.LoanTermMonths,
		&loanOffer.Status,
		&loanOffer.CreatedAt,
		&loanOffer.ApplicationID,
	)
	if err != nil {
		return loanOffer, fmt.Errorf("failed to create loan offer: %w", err)
	}

	return loanOffer, nil
}

// AcceptLoanOffer updates loan offer status and returns the updated loan offer.
func (rd *loanRepo) AcceptLoanOffer(ctx context.Context, offerID, borrowerID string) (LoanOffer, error) {
	// Declare a variable to store the result of the updated loan offer
	var updatedLoanOffer LoanOffer

	// Execute the update query and return the updated row
	err := rd.DB.QueryRowContext(ctx, acceptLoanOfferStatusUpdationQuery, offerID).Scan(
		&updatedLoanOffer.OfferID,
		&updatedLoanOffer.LenderID,
		&updatedLoanOffer.Amount,
		&updatedLoanOffer.InterestRate,
		&updatedLoanOffer.LoanTermMonths,
		&updatedLoanOffer.Status,
		&updatedLoanOffer.CreatedAt,
		&updatedLoanOffer.ApplicationID,
	)

	log.Println("Updated Loan Offer", updatedLoanOffer)

	// Handle any errors
	if err != nil {
		return LoanOffer{}, fmt.Errorf("failed to update offer status or retrieve updated loan offer: %w", err)
	}

	return updatedLoanOffer, nil
}

// GetLoanDetails fetches loan details.
func (rd *loanRepo) GetLoanDetails(ctx context.Context, loanID, offerID, borrowerID, lenderID, status, applicationID string) ([]Loan, error) {
	var loans []Loan
	var query string = getLoanDetailsQuery
	var args []interface{}

	// Start building the query
	query = getLoanDetailsQuery

	// Adding filters based on the parameters passed
	if loanID != "" {
		query += " AND loan_id = $1"
		args = append(args, loanID)
	} else if offerID != "" {
		query += " AND offer_id = $1"
		args = append(args, offerID)
	} else if applicationID != "" {
		query += " AND application_id = $1"
		args = append(args, applicationID)
	} else {
		argcount := 0
		if borrowerID != "" {
			query += " AND borrower_id = $" + fmt.Sprint(argcount+1)
			args = append(args, borrowerID)
		}
		if lenderID != "" {
			query += " AND lender_id = $" + fmt.Sprint(argcount+1)
			args = append(args, lenderID)
		}
		if status != "" {
			query += " AND status = $" + fmt.Sprint(argcount+1)
			args = append(args, status)
		}
	}

	// Execute the query
	rows, err := rd.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Scan the result into the Loan struct
	for rows.Next() {
		var loan Loan
		err := rows.Scan(
			&loan.LoanID, &loan.OfferID, &loan.BorrowerID, &loan.LenderID, &loan.TotalPrinciple, &loan.RemainingPrinciple,
			&loan.Status, &loan.StartDate, &loan.NextPaymentDate, &loan.ApplicationID, &loan.InterestRate, &loan.SettledAmount,
			&loan.SettlementDate, &loan.AccruedInterest,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		loans = append(loans, loan)
	}

	// Check for errors after scanning
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return loans, nil
}

// UpdateLoanRepayment updates loan status after repayment.
func (rd *loanRepo) UpdateLoanRepayment(ctx context.Context, loanID string, newRemaining float64) error {
	status := "active"
	if newRemaining <= 0 {
		status = "closed"
	}
	_, err := rd.DB.ExecContext(ctx, updateLoanRepaymentQuery, newRemaining, status, loanID)
	return err
}

// CreateLoanapplication creates a new loan application and returns the created Loanapplication
func (rd *loanRepo) CreateLoanapplication(ctx context.Context, borrowerID string, amount, interestRate float64, termMonths int) (Loanapplication, error) {
	// Generate a new application ID
	applicationID := uuid.New().String()

	// Execute the query and scan the result into a Loanapplication struct
	var loanapplication Loanapplication
	err := rd.DB.QueryRowContext(ctx, createLoanapplicationQuery, applicationID, borrowerID, amount, interestRate, termMonths).Scan(
		&loanapplication.ApplicationID,
		&loanapplication.BorrowerID,
		&loanapplication.Amount,
		&loanapplication.InterestRate,
		&loanapplication.TermMonths,
		&loanapplication.Status,
		&loanapplication.CreatedAt,
		&loanapplication.UpdatedAt,
	)
	if err != nil {
		log.Printf("failed to create loan application: %w", err)
		return loanapplication, fmt.Errorf("failed to create loan application: %w", err)
	}

	return loanapplication, nil
}

// GetLoanapplications fetches Loan applications based on either application_id or borrower_id or status, clubbing borrower_id and status is allowed
func (rd *loanRepo) GetLoanapplications(ctx context.Context, applicationID string, borrowerID string, status string) ([]Loanapplication, error) {
	var loanapplications []Loanapplication
	var query string = getLoanapplicationsQuery
	var args []interface{}

	// Adding filters based on the parameters passed
	if applicationID != "" {
		query += " AND application_id = $1"
		args = append(args, applicationID)
	} else {
		// Apply filters for borrower_id and status if application_id is not passed
		if borrowerID != "" && status != "" {
			query += " AND borrower_id = $1 AND status = $2"
			args = append(args, borrowerID)
			args = append(args, status)
		} else if status != "" {
			query += " AND status = $1"
			args = append(args, status)
		} else if borrowerID != "" {
			query += " AND borrower_id = $1"
			args = append(args, borrowerID)
		}
	}

	// Execute the query
	rows, err := rd.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch loan applications: %w", err)
	}
	defer rows.Close()

	// Scan the results into a slice of Loanapplication structs
	for rows.Next() {
		var loanapplication Loanapplication
		if err := rows.Scan(&loanapplication.ApplicationID, &loanapplication.BorrowerID, &loanapplication.Amount, &loanapplication.InterestRate, &loanapplication.TermMonths, &loanapplication.Status, &loanapplication.CreatedAt, &loanapplication.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan loan application: %w", err)
		}
		loanapplications = append(loanapplications, loanapplication)
	}

	// Check for any row iteration errors
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return loanapplications, nil
}

// GetLoanOffers fetches Loan Offers based on either offerID or applicationID or lenderID or status, clubbing lenderID and status is allowed
func (rd *loanRepo) GetLoanOffers(ctx context.Context, offerID string, applicationID string, lenderID string, status string) ([]LoanOffer, error) {
	var loanOffers []LoanOffer
	var query string = getLoanOffersQuery
	var args []interface{}

	// Adding filters based on the parameters passed
	if offerID != "" {
		query += " AND offer_id = $1"
		args = append(args, offerID)
	} else if applicationID != "" {
		query += " AND application_id = $1"
		args = append(args, applicationID)
	} else {
		// Apply filters for lender_id and status if offer_id and application_id are not passed
		if lenderID != "" && status != "" {
			query += " AND lender_id = $1 AND status = $2"
			args = append(args, lenderID)
			args = append(args, status)
		} else if status != "" {
			query += " AND status = $1"
			args = append(args, status)
		} else if lenderID != "" {
			query += " AND lender_id = $1"
			args = append(args, lenderID)
		}
	}

	// Execute the query
	rows, err := rd.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch loan offers: %w", err)
	}
	defer rows.Close()

	// Scan the results into a slice of LoanOffer structs
	for rows.Next() {
		var loanOffer LoanOffer
		if err := rows.Scan(&loanOffer.OfferID, &loanOffer.LenderID, &loanOffer.Amount, &loanOffer.InterestRate, &loanOffer.LoanTermMonths, &loanOffer.Status, &loanOffer.CreatedAt, &loanOffer.ApplicationID); err != nil {
			return nil, fmt.Errorf("failed to scan loan offer: %w", err)
		}
		loanOffers = append(loanOffers, loanOffer)
	}

	// Check for any row iteration errors
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return loanOffers, nil
}

// IsKYCVerified checks if the user's KYC is verified.
func (rd *loanRepo) IsKYCVerified(ctx context.Context, userID string) (bool, error) {
	var isVerified bool
	query := `SELECT EXISTS (SELECT 1 FROM kyc_verifications WHERE user_id = $1 AND verification_status = 'Verified')`

	err := rd.DB.QueryRowContext(ctx, query, userID).Scan(&isVerified)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // No KYC record found
		}
		return false, fmt.Errorf("failed to check KYC verification: %w", err)
	}

	return isVerified, nil
}

func (rd *loanRepo) DisburseLoan(ctx context.Context, offerID, borrowerID, lenderID, applicationID string, totalPrinciple, interestRate float64, nextPaymentDate time.Time, DisbursementTransactionID string) (Loan, error) {

	// Begin a transaction
	tx, err := rd.DB.BeginTx(ctx, nil)
	if err != nil {
		return Loan{}, err
	}

	// Generate a unique loan ID
	loanID := uuid.New().String()

	// Insert the loan record and use RETURNING to fetch the inserted row
	var loan Loan
	err = tx.QueryRowContext(ctx, DisburseLoanQuery, loanID, offerID, borrowerID, totalPrinciple, totalPrinciple, interestRate, lenderID, applicationID, nextPaymentDate, DisbursementTransactionID).Scan(
		&loan.LoanID, &loan.OfferID, &loan.BorrowerID, &loan.LenderID, &loan.TotalPrinciple, &loan.RemainingPrinciple,
		&loan.Status, &loan.StartDate, &loan.NextPaymentDate, &loan.ApplicationID, &loan.InterestRate, &loan.DisbursementTransactionID)

	if err != nil {
		tx.Rollback()
		return Loan{}, fmt.Errorf("failed to create loan record: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE loan_offers SET status = 'Funded' WHERE offer_id = $1", loan.OfferID)
	if err != nil {
		tx.Rollback()
		return Loan{}, fmt.Errorf("failed to update loan offer status to Funded: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE loan_applications SET status = 'Funded' WHERE application_id = $1", loan.ApplicationID)
	if err != nil {
		tx.Rollback()
		return Loan{}, fmt.Errorf("failed to update loan application status to Funded: %w", err)
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return Loan{}, fmt.Errorf("transaction commit failed: %w", err)
	}

	// Return the inserted loan record
	return loan, nil
}

func (rd *loanRepo) SettleLoan(ctx context.Context, loanID string, settledAmount, accruedInterest float64, settlementTransactionID string) (Loan, error) {
	// Execute the update query and scan the result into a Loan struct
	var loan Loan
	tx, err := rd.DB.BeginTx(ctx, nil)
	if err != nil {
		return Loan{}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	err = tx.QueryRowContext(ctx, settleLoanQuery, settledAmount, accruedInterest, loanID).Scan(&loan.LoanID, &loan.OfferID, &loan.BorrowerID, &loan.LenderID, &loan.TotalPrinciple, &loan.RemainingPrinciple, &loan.Status, &loan.StartDate, &loan.NextPaymentDate, &loan.ApplicationID, &loan.InterestRate, &loan.SettledAmount, &loan.SettlementDate, &loan.AccruedInterest)

	if err != nil {
		tx.Rollback()
		log.Printf("failed to settle loan: %w", err.Error())
		return Loan{}, fmt.Errorf("failed to settle loan: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE loan_applications SET status = 'Closed' WHERE application_id = $1", loan.ApplicationID)
	if err != nil {
		tx.Rollback()
		return Loan{}, fmt.Errorf("failed to update loan application status: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE loan_offers SET status = 'Closed' WHERE offer_id = $1", loan.OfferID)
	if err != nil {
		tx.Rollback()
		return Loan{}, fmt.Errorf("failed to update loan offer status: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return Loan{}, fmt.Errorf("transaction commit failed: %w", err)
	}

	// Return the updated loan record
	return loan, nil
}
