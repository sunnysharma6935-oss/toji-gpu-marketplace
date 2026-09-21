package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TickResult tells the caller what happened, so it can act.
type TickResult struct {
	ChargedPaise   int64
	ShortfallPaise int64 // >0 only possible when IsFinalSettle is true — see the balance-cap rule below
	MustStopRental bool  // true if this (non-final) tick could not be fully funded
	IsFinalSettle  bool
}

// ProcessBillingTickTx is the core billing-tick logic, operating on a transaction the
// CALLER owns — it never begins, commits, or rolls back anything itself. This exists
// specifically so a caller that needs a wider atomic operation (finalizeRental, see
// internal/api/rental_lifecycle.go) can hold one lock across the entire stop flow:
// the rentals row lock, the billing tick, AND the status flip, all in one transaction,
// which is what makes two simultaneous stop triggers for the same rental serialize
// correctly instead of racing. A normal usage-heartbeat tick doesn't need this — it
// uses the ProcessBillingTick wrapper below, which manages its own short transaction.
func ProcessBillingTickTx(ctx context.Context, tx pgx.Tx, rentalID uuid.UUID, customerAccountID uuid.UUID, hostAccountID uuid.UUID, now time.Time, isStop bool) (TickResult, error) {
	var rate int64
	var lastBilledAt time.Time
	var remainder int64
	err := tx.QueryRow(ctx,
		`SELECT rate_paise_per_hour, last_billed_at, carried_remainder
		 FROM rental_billing_state WHERE rental_id = $1 FOR UPDATE`,
		rentalID,
	).Scan(&rate, &lastBilledAt, &remainder)
	if err != nil {
		return TickResult{}, fmt.Errorf("load billing state (has RentalReady run yet?): %w", err)
	}

	elapsedSeconds := int64(now.Sub(lastBilledAt).Seconds())
	if elapsedSeconds <= 0 {
		// Clock skew or a duplicate tick arrived — nothing to bill, not an error.
		return TickResult{ChargedPaise: 0, IsFinalSettle: isStop}, nil
	}

	chargePaise, newRemainder := ComputeTickCharge(rate, elapsedSeconds, remainder)

	if isStop {
		chargePaise = FinalizeRemainder(chargePaise, newRemainder)
		newRemainder = 0
	}

	// BALANCE-CAP RULE (final settlement only, unchanged from the approved decision):
	// TOJI must never create a negative customer balance. If the intended final charge
	// exceeds available balance, only the available amount is charged; the remainder
	// is recorded in rental_shortfalls, never silently dropped, never charged negative.
	var shortfallPaise int64
	if isStop && chargePaise > 0 {
		availableBalance, err := Balance(ctx, tx, customerAccountID)
		if err != nil {
			return TickResult{}, fmt.Errorf("check available balance for final settlement: %w", err)
		}
		if chargePaise > availableBalance {
			cappedCharge := availableBalance
			if cappedCharge < 0 {
				cappedCharge = 0
			}
			shortfallPaise = chargePaise - cappedCharge
			chargePaise = cappedCharge
		}
	}

	if chargePaise > 0 {
		platformPaise, hostPaise := SplitGrossAmount(chargePaise, PlatformCommissionBps)

		var eventID uuid.UUID
		err = tx.QueryRow(ctx,
			`INSERT INTO rental_usage_events
				(rental_id, period_start, period_end, elapsed_seconds, rate_paise_per_hour,
				 commission_rate_bps, gross_amount_paise, platform_revenue_paise, host_payable_paise,
				 is_final_settlement)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
			rentalID, lastBilledAt, now, elapsedSeconds, rate,
			PlatformCommissionBps, chargePaise, platformPaise, hostPaise, isStop,
		).Scan(&eventID)
		if err != nil {
			return TickResult{}, fmt.Errorf("insert usage event: %w", err)
		}

		if _, err = InsertLedgerEntry(ctx, tx, EntryInput{
			AccountID: customerAccountID, AmountPaise: -chargePaise,
			EntryType: "rental_charge", Status: "completed", EventID: &eventID, RentalID: &rentalID,
		}); err != nil {
			return TickResult{}, fmt.Errorf("insert customer charge entry: %w", err)
		}

		// platform_revenue and host_payable are inserted only when their split is
		// actually nonzero. ledger_entries has CHECK (amount_paise != 0) — a zero-value
		// split (e.g. a short final settlement where commission rounds down to 0 paise)
		// is not a real money movement, so no row is written for that side. This is NOT
		// a relaxation of the ledger constraint; the constraint stays exactly as-is.
		// gross = platform + host remains provable from rental_usage_events alone,
		// which always stores all three numbers including any zero, regardless of how
		// many ledger_entries rows follow. A usage event therefore produces 2 or 3
		// ledger_entries rows (customer charge always present, plus whichever of
		// platform/host are individually nonzero — at least one always is, since they
		// sum to a positive gross and can't both be zero).
		if platformPaise > 0 {
			platformAccountID, err := GetOrCreateAccount(ctx, tx, "platform_revenue", "platform", nil)
			if err != nil {
				return TickResult{}, fmt.Errorf("resolve platform account: %w", err)
			}
			if _, err = InsertLedgerEntry(ctx, tx, EntryInput{
				AccountID: platformAccountID, AmountPaise: platformPaise,
				EntryType: "platform_revenue", Status: "completed", EventID: &eventID, RentalID: &rentalID,
			}); err != nil {
				return TickResult{}, fmt.Errorf("insert platform revenue entry: %w", err)
			}
		}
		if hostPaise > 0 {
			if _, err = InsertLedgerEntry(ctx, tx, EntryInput{
				AccountID: hostAccountID, AmountPaise: hostPaise,
				EntryType: "host_payable", Status: "completed", EventID: &eventID, RentalID: &rentalID,
			}); err != nil {
				return TickResult{}, fmt.Errorf("insert host payable entry: %w", err)
			}
		}

		if err = ConsumeFromReservation(ctx, tx, rentalID, chargePaise); err != nil {
			return TickResult{}, fmt.Errorf("consume reservation: %w", err)
		}
	}

	if shortfallPaise > 0 {
		if _, err = tx.Exec(ctx,
			`INSERT INTO rental_shortfalls (rental_id, amount_paise) VALUES ($1, $2)`,
			rentalID, shortfallPaise,
		); err != nil {
			return TickResult{}, fmt.Errorf("record shortfall: %w", err)
		}
	}

	if _, err = tx.Exec(ctx,
		`UPDATE rental_billing_state SET last_billed_at = $1, carried_remainder = $2 WHERE rental_id = $3`,
		now, newRemainder, rentalID,
	); err != nil {
		return TickResult{}, fmt.Errorf("update billing state: %w", err)
	}

	result := TickResult{ChargedPaise: chargePaise, ShortfallPaise: shortfallPaise, IsFinalSettle: isStop}

	if isStop {
		if err = ReleaseReservation(ctx, tx, rentalID); err != nil {
			return TickResult{}, fmt.Errorf("release reservation: %w", err)
		}
	} else {
		shouldContinue, err := MaybeTopUpReservation(ctx, tx, customerAccountID, rentalID, rate)
		if err != nil {
			return TickResult{}, fmt.Errorf("check reservation top-up: %w", err)
		}
		if !shouldContinue {
			result.MustStopRental = true
		}
	}

	return result, nil
}

// ProcessBillingTick is the standalone entry point used by a plain usage-heartbeat
// (which doesn't need to coordinate with anything else) — it owns its own short
// transaction around ProcessBillingTickTx. For the stop flow, see finalizeRental in
// internal/api/rental_lifecycle.go, which calls ProcessBillingTickTx directly inside
// its own wider transaction instead of using this wrapper.
func ProcessBillingTick(ctx context.Context, pool *pgxpool.Pool, rentalID uuid.UUID, customerAccountID uuid.UUID, hostAccountID uuid.UUID, now time.Time, isStop bool) (TickResult, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return TickResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op if committed

	result, err := ProcessBillingTickTx(ctx, tx, rentalID, customerAccountID, hostAccountID, now, isStop)
	if err != nil {
		return TickResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TickResult{}, fmt.Errorf("commit tick: %w", err)
	}
	return result, nil
}

// ReserveAtRentalCreation is called from CreateRental, inside ITS transaction, before
// the rentals row is committed.
func ReserveAtRentalCreation(ctx context.Context, tx pgx.Tx, customerAccountID, rentalID uuid.UUID, ratePaisePerHour int64) error {
	_, err := ReserveForRental(ctx, tx, customerAccountID, rentalID, ratePaisePerHour)
	return err
}

// InitBillingState is called from RentalReady, inside ITS transaction, once the rental
// is confirmed running.
func InitBillingState(ctx context.Context, tx pgx.Tx, rentalID uuid.UUID, ratePaisePerHour int64, startedAt time.Time) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO rental_billing_state (rental_id, rate_paise_per_hour, last_billed_at, carried_remainder)
		 VALUES ($1, $2, $3, 0)`,
		rentalID, ratePaisePerHour, startedAt,
	)
	return err
}
