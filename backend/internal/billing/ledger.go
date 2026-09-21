package billing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Queryable is satisfied by both *pgxpool.Pool and pgx.Tx — every read helper below
// accepts this so it can be called either standalone or inside an in-flight transaction.
type Queryable interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

const PlatformCommissionBps = 1500 // 15.00% — snapshotted onto every RentalUsageEvent, not hardcoded at read time

// GetOrCreateAccount resolves a LedgerAccount for (accountType, ownerID), creating it
// on first use. ownerID is nil for the platform-wide revenue account.
func GetOrCreateAccount(ctx context.Context, q Queryable, accountType string, ownerType string, ownerID *uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRow(ctx,
		`SELECT id FROM ledger_accounts WHERE account_type = $1 AND owner_id IS NOT DISTINCT FROM $2`,
		accountType, ownerID,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return uuid.Nil, fmt.Errorf("lookup account: %w", err)
	}
	// ON CONFLICT handles the race where two concurrent first-uses both miss the SELECT
	// above and both try to create the account — the unique (account_type, owner_id)
	// constraint would otherwise turn that into a crash instead of a safe no-op.
	err = q.QueryRow(ctx,
		`INSERT INTO ledger_accounts (account_type, owner_type, owner_id) VALUES ($1, $2, $3)
		 ON CONFLICT (account_type, owner_id) DO UPDATE SET account_type = EXCLUDED.account_type
		 RETURNING id`,
		accountType, ownerType, ownerID,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create account: %w", err)
	}
	return id, nil
}

// InsertLedgerEntry writes one immutable ledger row. This is the ONLY function in the
// codebase that should ever INSERT into ledger_entries — every balance-affecting
// operation (top-up, charge, payout, refund) must go through this, so there is exactly
// one place that could get the invariants wrong, not several.
type EntryInput struct {
	AccountID       uuid.UUID
	AmountPaise     int64 // signed; positive=credit, negative=debit. Must not be zero.
	EntryType       string
	Status          string // usually "completed"; "pending" only for not-yet-settled provider callbacks
	EventID         *uuid.UUID
	RentalID        *uuid.UUID
	PaymentIntentID *uuid.UUID
}

func InsertLedgerEntry(ctx context.Context, tx pgx.Tx, e EntryInput) (uuid.UUID, error) {
	if e.AmountPaise == 0 {
		return uuid.Nil, fmt.Errorf("refusing to insert a zero-amount ledger entry (type=%s)", e.EntryType)
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx,
		`INSERT INTO ledger_entries
			(account_id, amount_paise, entry_type, status, event_id, rental_id, payment_intent_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		e.AccountID, e.AmountPaise, e.EntryType, e.Status, e.EventID, e.RentalID, e.PaymentIntentID,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert ledger entry: %w", err)
	}
	return id, nil
}

// Balance sums all completed ledger entries for an account. This is intentionally the
// ONLY way a balance is ever computed — there is no stored balance field anywhere.
func Balance(ctx context.Context, q Queryable, accountID uuid.UUID) (int64, error) {
	var total int64
	err := q.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_paise), 0) FROM ledger_entries
		 WHERE account_id = $1 AND status = 'completed'`,
		accountID,
	).Scan(&total)
	return total, err
}

// ActiveReservationTotal sums currently-active reservation holds against an account.
func ActiveReservationTotal(ctx context.Context, q Queryable, accountID uuid.UUID) (int64, error) {
	var total int64
	err := q.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_paise), 0) FROM reservations
		 WHERE account_id = $1 AND status = 'active'`,
		accountID,
	).Scan(&total)
	return total, err
}

// SpendableBalance = completed balance - active reservations. This is the number
// checked before starting any new rental, per the approved spec.
func SpendableBalance(ctx context.Context, q Queryable, accountID uuid.UUID) (int64, error) {
	bal, err := Balance(ctx, q, accountID)
	if err != nil {
		return 0, err
	}
	reserved, err := ActiveReservationTotal(ctx, q, accountID)
	if err != nil {
		return 0, err
	}
	return bal - reserved, nil
}
