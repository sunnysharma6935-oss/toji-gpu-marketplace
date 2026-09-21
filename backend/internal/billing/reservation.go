package billing

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ReservationWindowHours is the OPEN CONFIG DECISION flagged in the approved spec
// (section F.1). Set to 1 hour as a starting value — revisit once real usage data
// exists to inform what a sane safety margin actually is.
const ReservationWindowHours = 1

// ReservationTopUpThresholdPercent: attempt a top-up once the held amount drops below
// this percentage of the full window. Integer percentage (25 = 25%), not a float — every
// money-derived calculation in this package stays integer-only, no exceptions.
const ReservationTopUpThresholdPercent = 25

var ErrInsufficientBalance = errors.New("insufficient spendable balance")

// ReserveForRental performs the atomic check-and-create from the approved spec (D.2).
// Must be called inside a transaction with the customer's account row locked
// (SELECT ... FOR UPDATE) to actually prevent the concurrent-overspend race —
// callers are responsible for opening that transaction; this function assumes it.
func ReserveForRental(ctx context.Context, tx pgx.Tx, customerAccountID, rentalID uuid.UUID, ratePaisePerHour int64) (uuid.UUID, error) {
	// Lock the account's ledger rows for the duration of this check. A real implementation
	// should lock a dedicated per-account row (e.g. SELECT ... FOR UPDATE on ledger_accounts
	// itself) rather than relying on read consistency alone — flagged here explicitly since
	// this is exactly the concurrency decision the approved spec (F.3) left open.
	if _, err := tx.Exec(ctx, `SELECT id FROM ledger_accounts WHERE id = $1 FOR UPDATE`, customerAccountID); err != nil {
		return uuid.Nil, fmt.Errorf("lock account: %w", err)
	}

	spendable, err := SpendableBalance(ctx, tx, customerAccountID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("check spendable balance: %w", err)
	}

	required := ratePaisePerHour * ReservationWindowHours
	if spendable < required {
		return uuid.Nil, ErrInsufficientBalance
	}

	var reservationID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO reservations (account_id, rental_id, amount_paise, status)
		 VALUES ($1, $2, $3, 'active') RETURNING id`,
		customerAccountID, rentalID, required,
	).Scan(&reservationID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create reservation: %w", err)
	}
	return reservationID, nil
}

// ConsumeFromReservation reduces the rental's active reservation by exactly the amount
// just billed (spec D.3) — this is what keeps "reserved" and "already charged" from
// double-counting the same money.
func ConsumeFromReservation(ctx context.Context, tx pgx.Tx, rentalID uuid.UUID, chargedPaise int64) error {
	_, err := tx.Exec(ctx,
		`UPDATE reservations
		 SET amount_paise = GREATEST(amount_paise - $1, 0), updated_at = now()
		 WHERE rental_id = $2 AND status = 'active'`,
		chargedPaise, rentalID,
	)
	return err
}

// MaybeTopUpReservation re-checks spendable balance and tops the reservation back up
// to the full window if it's fallen below the threshold (spec D.4). Returns whether
// the rental should continue (false means: insufficient funds, caller must stop it).
func MaybeTopUpReservation(ctx context.Context, tx pgx.Tx, customerAccountID, rentalID uuid.UUID, ratePaisePerHour int64) (shouldContinue bool, err error) {
	var current int64
	err = tx.QueryRow(ctx,
		`SELECT amount_paise FROM reservations WHERE rental_id = $1 AND status = 'active'`,
		rentalID,
	).Scan(&current)
	if err != nil {
		return false, fmt.Errorf("read reservation: %w", err)
	}

	fullWindow := ratePaisePerHour * ReservationWindowHours
	threshold := (fullWindow * ReservationTopUpThresholdPercent) / 100 // integer division, no float anywhere
	if current > threshold {
		return true, nil // no top-up needed yet
	}

	if _, err := tx.Exec(ctx, `SELECT id FROM ledger_accounts WHERE id = $1 FOR UPDATE`, customerAccountID); err != nil {
		return false, fmt.Errorf("lock account: %w", err)
	}
	// Spendable balance here already excludes this rental's own active reservation amount
	// (SpendableBalance subtracts ALL active reservations for the account, this one included) —
	// so topping up to `fullWindow` means checking that spendable can cover the *additional*
	// amount needed, not the full window again.
	spendable, err := SpendableBalance(ctx, tx, customerAccountID)
	if err != nil {
		return false, fmt.Errorf("check spendable: %w", err)
	}
	topUpNeeded := fullWindow - current
	if spendable < topUpNeeded {
		return false, nil // top-up fails -> caller must trigger shutdown
	}

	_, err = tx.Exec(ctx,
		`UPDATE reservations SET amount_paise = $1, updated_at = now() WHERE rental_id = $2 AND status = 'active'`,
		fullWindow, rentalID,
	)
	if err != nil {
		return false, fmt.Errorf("top up reservation: %w", err)
	}
	return true, nil
}

// ReleaseReservation closes out a rental's reservation at stop time (spec D.5).
// consumed=true if it was drawn all the way to zero, false if unused amount remains.
func ReleaseReservation(ctx context.Context, tx pgx.Tx, rentalID uuid.UUID) error {
	var remaining int64
	err := tx.QueryRow(ctx,
		`SELECT amount_paise FROM reservations WHERE rental_id = $1 AND status = 'active'`,
		rentalID,
	).Scan(&remaining)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // already released/consumed, nothing to do
	}
	if err != nil {
		return fmt.Errorf("read reservation: %w", err)
	}

	status := "released"
	if remaining == 0 {
		status = "consumed"
	}
	_, err = tx.Exec(ctx,
		`UPDATE reservations SET status = $1, updated_at = now() WHERE rental_id = $2 AND status = 'active'`,
		status, rentalID,
	)
	return err
}
