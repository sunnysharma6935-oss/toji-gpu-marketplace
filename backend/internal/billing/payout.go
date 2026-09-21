package billing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RequestPayout implements the exact payout lifecycle from the approved spec (E):
// create a PaymentIntent, and — only on confirmed success — write the payout
// LedgerEntry that actually decreases host_payable. Manual/mock provider only,
// per the approved decision; no automated triggering exists yet.
func RequestPayout(ctx context.Context, pool *pgxpool.Pool, hostID, hostAccountID uuid.UUID, amountPaise int64, method string) (paymentIntentID uuid.UUID, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	available, err := Balance(ctx, tx, hostAccountID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("check available balance: %w", err)
	}
	if amountPaise > available {
		return uuid.Nil, fmt.Errorf("requested payout %d paise exceeds available balance %d paise", amountPaise, available)
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO payment_intents (direction, host_id, amount_paise, method, provider, status)
		 VALUES ('payout', $1, $2, $3, 'mock', 'initiated') RETURNING id`,
		hostID, amountPaise, method,
	).Scan(&paymentIntentID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create payment intent: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("commit: %w", err)
	}
	return paymentIntentID, nil
}

// ConfirmPayoutSuccess is called once the (mock, for now) provider reports success —
// this is the step that actually creates the negative host_payable LedgerEntry, per
// your correction: a payout only counts once this entry exists, not when the
// PaymentIntent is merely created.
func ConfirmPayoutSuccess(ctx context.Context, pool *pgxpool.Pool, paymentIntentID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var hostID uuid.UUID
	var amountPaise int64
	var status string
	err = tx.QueryRow(ctx,
		`SELECT host_id, amount_paise, status FROM payment_intents WHERE id = $1 AND direction = 'payout' FOR UPDATE`,
		paymentIntentID,
	).Scan(&hostID, &amountPaise, &status)
	if err != nil {
		return fmt.Errorf("load payment intent: %w", err)
	}
	if status == "succeeded" {
		return nil // already confirmed — idempotent, no duplicate entry
	}

	hostAccountID, err := GetOrCreateAccount(ctx, tx, "host_payable", "host", &hostID)
	if err != nil {
		return fmt.Errorf("resolve host account: %w", err)
	}

	if _, err = InsertLedgerEntry(ctx, tx, EntryInput{
		AccountID: hostAccountID, AmountPaise: -amountPaise,
		EntryType: "payout", Status: "completed", PaymentIntentID: &paymentIntentID,
	}); err != nil {
		return fmt.Errorf("insert payout entry: %w", err)
	}

	if _, err = tx.Exec(ctx,
		`UPDATE payment_intents SET status = 'succeeded', updated_at = now() WHERE id = $1`,
		paymentIntentID,
	); err != nil {
		return fmt.Errorf("update payment intent status: %w", err)
	}

	return tx.Commit(ctx)
}

// ConfirmPayoutFailure marks the intent failed. No LedgerEntry is created — per the
// spec, nothing was ever recorded as paid, so there is nothing to reverse.
func ConfirmPayoutFailure(ctx context.Context, pool *pgxpool.Pool, paymentIntentID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`UPDATE payment_intents SET status = 'failed', updated_at = now() WHERE id = $1`,
		paymentIntentID,
	)
	return err
}
