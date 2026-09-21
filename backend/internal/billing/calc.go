// Package billing implements the TOJI AI financial ledger: deterministic per-second
// billing math, and the read/write operations over LedgerAccount / LedgerEntry /
// RentalUsageEvent / Reservation / PaymentIntent defined in migrations/002_billing.sql.
//
// Hard rule enforced throughout this package: every money amount is an int64 of paise.
// No float64 appears anywhere in this file or any file that touches an amount.
package billing

// ComputeTickCharge implements the exact remainder-carry rounding formula:
//
//	total_minor_numerator = (rate_paise_per_hour * elapsed_seconds) + carried_remainder
//	charge_paise          = total_minor_numerator / 3600           (integer division)
//	new_carried_remainder = total_minor_numerator % 3600
//
// carried_remainder is always in [0, 3599] on input and output — it's a mod-3600
// result by construction, so it can never grow unbounded across a long rental.
func ComputeTickCharge(ratePaisePerHour int64, elapsedSeconds int64, carriedRemainder int64) (chargePaise int64, newRemainder int64) {
	if carriedRemainder < 0 || carriedRemainder > 3599 {
		panic("carriedRemainder must be in [0, 3599] — caller violated the invariant")
	}
	totalNumerator := ratePaisePerHour*elapsedSeconds + carriedRemainder
	chargePaise = totalNumerator / 3600
	newRemainder = totalNumerator % 3600
	return chargePaise, newRemainder
}

// FinalizeRemainder applies the one-time, bounded final rounding rule at rental close:
// round half up if the leftover remainder is >= half of 3600, otherwise drop it.
// This is the only place in a rental's entire billing history where any fraction
// is discarded, and it's bounded to under half a paise, exactly once.
func FinalizeRemainder(finalTickChargePaise int64, remainder int64) int64 {
	if remainder >= 1800 {
		return finalTickChargePaise + 1
	}
	return finalTickChargePaise
}

// SplitGrossAmount computes the platform/host split from one tick's gross charge.
// Platform's share is computed first (basis points of gross, rounded down); the host
// gets the remainder. This construction — not rounding both independently — is what
// guarantees gross == platform + host by definition, not by coincidence.
func SplitGrossAmount(grossPaise int64, commissionRateBps int64) (platformPaise, hostPaise int64) {
	if commissionRateBps < 0 || commissionRateBps > 10000 {
		panic("commissionRateBps must be in [0, 10000]")
	}
	platformPaise = (grossPaise * commissionRateBps) / 10000 // integer division, rounds down
	hostPaise = grossPaise - platformPaise                   // remainder — guarantees the invariant by construction
	return platformPaise, hostPaise
}
