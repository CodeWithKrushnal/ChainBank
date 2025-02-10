package repo

import (
	"context"
	"database/sql"
	"fmt"
	"log"

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
	CreateLoanOffer(ctx context.Context, lenderID string, amount float64, interestRate float64, termMonths int) (string, error)
	GetLoanOffer(ctx context.Context, offerID string) (string, float64, error)
	AcceptLoanOffer(ctx context.Context, offerID, borrowerID string, amount float64) (string, error)
	GetLoanDetails(ctx context.Context, loanID, borrowerID string) (string, float64, error)
	UpdateLoanRepayment(ctx context.Context, loanID string, newRemaining float64) error
}

const (
	createLoanOfferQuery = `INSERT INTO lending_offers (offer_id, lender_id, amount, interest_rate, loan_term_months, status, created_at) VALUES ($1, $2, $3, $4, $5, 'pending', NOW())`
)

// CreateLoanOffer inserts a new loan offer in the DB.
func (rd *loanRepo) CreateLoanOffer(ctx context.Context, lenderID string, amount float64, interestRate float64, termMonths int) (string, error) {
	offerID := uuid.New().String()
	_, err := rd.DB.ExecContext(ctx, createLoanOfferQuery, offerID, lenderID, amount, interestRate, termMonths)
	if err != nil {
		return "", fmt.Errorf("failed to create loan offer: %w", err)
	}
	return offerID, nil
}

// GetLoanOffer fetches a loan offer by ID.
func (rd *loanRepo) GetLoanOffer(ctx context.Context, offerID string) (string, float64, error) {
	var lenderID string
	var amount float64
	query := `SELECT lender_id, amount FROM lending_offers WHERE offer_id = $1 AND status = 'pending'`
	err := rd.DB.QueryRowContext(ctx, query, offerID).Scan(&lenderID, &amount)
	if err != nil {
		return "", 0, fmt.Errorf("loan offer not found or already taken")
	}
	return lenderID, amount, nil
}

// AcceptLoanOffer updates loan offer status and creates a new loan.
func (rd *loanRepo) AcceptLoanOffer(ctx context.Context, offerID, borrowerID string, amount float64) (string, error) {
	tx, err := rd.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}

	// Create loan record
	loanID := uuid.New().String()
	query := `INSERT INTO loans (loan_id, offer_id, borrower_id, total_amount, remaining_amount, status, start_date, next_payment_date) 
             VALUES ($1, $2, $3, $4, $4, 'active', NOW(), NOW() + INTERVAL '30 days')`
	_, err = tx.ExecContext(ctx, query, loanID, offerID, borrowerID, amount)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to create loan record: %w", err)
	}

	// Update offer status
	query = `UPDATE lending_offers SET status = 'funded' WHERE offer_id = $1`
	_, err = tx.ExecContext(ctx, query, offerID)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to update offer status: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return "", fmt.Errorf("transaction commit failed: %w", err)
	}
	return loanID, nil
}

// GetLoanDetails fetches loan details.
func (r *loanRepo) GetLoanDetails(ctx context.Context, loanID, borrowerID string) (string, float64, error) {
	var lenderID string
	var remainingAmount float64

	log.Println("loan id: ", loanID, "borrower: ", borrowerID)

	query := `SELECT o.lender_id, l.remaining_amount FROM loans l 
              JOIN lending_offers o ON l.offer_id = o.offer_id 
              WHERE l.loan_id = $1 AND l.borrower_id = $2`
	err := r.DB.QueryRowContext(ctx, query, loanID, borrowerID).Scan(&lenderID, &remainingAmount)
	if err != nil {
		log.Println("error is:", err.Error())
		return "", 0, fmt.Errorf("loan not found")
	}

	return lenderID, remainingAmount, nil
}

// UpdateLoanRepayment updates loan status after repayment.
func (r *loanRepo) UpdateLoanRepayment(ctx context.Context, loanID string, newRemaining float64) error {
	status := "active"
	if newRemaining <= 0 {
		status = "closed"
	}

	query := `UPDATE loans SET remaining_amount = $1, status = $2, next_payment_date = NOW() + INTERVAL '30 days' WHERE loan_id = $3`
	_, err := r.DB.ExecContext(ctx, query, newRemaining, status, loanID)
	return err
}
