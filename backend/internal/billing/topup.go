package billing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InitiateTopUp creates a PaymentIntent for a customer top-up. Provider is always
// "mock" pre-launch — no real payment gateway is wired in.
func InitiateTopUp(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, amountPaise int64, method string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO payment_intents (direction, customer_id, amount_paise, method, provider, status)
		 VALUES ('top_up', $1, $2, $3, 'mock', 'initiated') RETURNING id`,
		customerID, amountPaise, method,
	).Scan(&id)
	return id, err
}

// ConfirmTopUpSuccess credits the customer's balance once the (mock) provider confirms.
func ConfirmTopUpSuccess(ctx context.Context, pool *pgxpool.Pool, paymentIntentID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var customerID uuid.UUID
	var amountPaise int64
	var status string
	err = tx.QueryRow(ctx,
		`SELECT customer_id, amount_paise, status FROM payment_intents WHERE id = $1 AND direction = 'top_up' FOR UPDATE`,
		paymentIntentID,
	).Scan(&customerID, &amountPaise, &status)
	if err != nil {
		return fmt.Errorf("load payment intent: %w", err)
	}
	if status == "succeeded" {
		return nil // idempotent
	}

	custAccountID, err := GetOrCreateAccount(ctx, tx, "customer_balance", "customer", &customerID)
	if err != nil {
		return fmt.Errorf("resolve customer account: %w", err)
	}

	if _, err = InsertLedgerEntry(ctx, tx, EntryInput{
		AccountID: custAccountID, AmountPaise: amountPaise,
		EntryType: "top_up", Status: "completed", PaymentIntentID: &paymentIntentID,
	}); err != nil {
		return fmt.Errorf("insert top-up entry: %w", err)
	}

	if _, err = tx.Exec(ctx,
		`UPDATE payment_intents SET status = 'succeeded', updated_at = now() WHERE id = $1`,
		paymentIntentID,
	); err != nil {
		return fmt.Errorf("update payment intent: %w", err)
	}
	return tx.Commit(ctx)
}

// IssueRefund creates an offsetting credit entry for a customer — e.g. when a rental
// is interrupted by a host going offline. Per the immutability rule, this NEVER edits
// the original rental_charge entry; it's always a new, separate row.
func IssueRefund(ctx context.Context, pool *pgxpool.Pool, customerAccountID uuid.UUID, amountPaise int64, originalEntryID uuid.UUID, rentalID *uuid.UUID) (uuid.UUID, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	refundID, err := InsertLedgerEntry(ctx, tx, EntryInput{
		AccountID: customerAccountID, AmountPaise: amountPaise, // positive — a credit back to the customer
		EntryType: "refund", Status: "completed", RentalID: rentalID,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert refund entry: %w", err)
	}

	// Link the original entry to its correction, without ever mutating the original's
	// financial fields (the immutability trigger would reject that anyway).
	if _, err = tx.Exec(ctx,
		`UPDATE ledger_entries SET reversed_by_entry_id = $1, status = 'reversed' WHERE id = $2`,
		refundID, originalEntryID,
	); err != nil {
		return uuid.Nil, fmt.Errorf("link reversal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit: %w", err)
	}
	return refundID, nil
}
