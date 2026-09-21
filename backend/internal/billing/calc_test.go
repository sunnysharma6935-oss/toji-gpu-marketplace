package billing

import "testing"

// Proves: over many ticks of a long-running rental, the sum of charges converges
// to the mathematically exact total (rate * total_seconds / 3600), to within the
// one bounded final-rounding step — never drifting further no matter how many
// ticks occur in between.
func TestComputeTickCharge_ConvergesToExactTotal(t *testing.T) {
	const ratePaisePerHour = int64(2900) // ₹29.00/hr, e.g. an RTX 4090 offer
	const tickSeconds = int64(30)        // 30-second billing tick, per the approved decision
	const numTicks = 500                 // ~4h10m rental

	var remainder int64 = 0
	var totalCharged int64 = 0

	for i := 0; i < numTicks; i++ {
		charge, newRemainder := ComputeTickCharge(ratePaisePerHour, tickSeconds, remainder)
		totalCharged += charge
		remainder = newRemainder
	}

	// Apply the final rounding rule as rental close would.
	finalCharge := FinalizeRemainder(0, remainder)
	totalCharged += finalCharge

	totalSeconds := tickSeconds * int64(numTicks)
	exactTotal := (ratePaisePerHour * totalSeconds) / 3600 // for comparison only — production code never does this in one shot
	// Since totalSeconds*rate is exactly divisible here (30*500*2900 / 3600), the two should match exactly.
	if totalCharged != exactTotal {
		t.Errorf("accumulated per-tick charges = %d paise, exact total = %d paise — drifted by %d",
			totalCharged, exactTotal, totalCharged-exactTotal)
	}
}

// Proves the same convergence property holds even when the math doesn't divide evenly —
// this is the actual case remainder-carry exists to handle correctly.
func TestComputeTickCharge_ConvergesWithUnevenRate(t *testing.T) {
	const ratePaisePerHour = int64(1733) // an intentionally "ugly" rate
	const tickSeconds = int64(30)
	const numTicks = 733

	var remainder int64 = 0
	var totalCharged int64 = 0
	for i := 0; i < numTicks; i++ {
		charge, newRemainder := ComputeTickCharge(ratePaisePerHour, tickSeconds, remainder)
		totalCharged += charge
		if newRemainder < 0 || newRemainder > 3599 {
			t.Fatalf("remainder escaped [0,3599] bound: got %d at tick %d", newRemainder, i)
		}
		remainder = newRemainder
	}
	finalCharge := FinalizeRemainder(0, remainder)
	totalCharged += finalCharge

	totalSeconds := tickSeconds * int64(numTicks)
	// Exact rational total, computed independently via a different method (big-number-free since
	// these values are small enough for int64): rate*seconds is the exact numerator over 3600.
	exactNumerator := ratePaisePerHour * totalSeconds
	exactWhole := exactNumerator / 3600
	exactFrac := exactNumerator % 3600 // in [0,3599]

	expectedTotal := exactWhole
	if exactFrac >= 1800 {
		expectedTotal++
	}

	if totalCharged != expectedTotal {
		t.Errorf("accumulated = %d paise, expected (exact + final half-up rounding) = %d paise",
			totalCharged, expectedTotal)
	}
}

// Proves the reconciliation invariant holds for every split, including edge cases
// (zero gross, commission rounding to zero, 100% commission).
func TestSplitGrossAmount_AlwaysReconciles(t *testing.T) {
	cases := []struct {
		gross int64
		bps   int64
	}{
		{gross: 0, bps: 1500},
		{gross: 1, bps: 1500},   // 1 paise gross — smallest possible nonzero charge
		{gross: 241, bps: 1500}, // ₹0.29/hr * 30s tick, roughly
		{gross: 100000, bps: 1500},
		{gross: 999, bps: 1},     // near-zero commission
		{gross: 999, bps: 9999},  // near-100% commission
		{gross: 999, bps: 10000}, // exactly 100% commission
		{gross: 999, bps: 0},     // exactly 0% commission
	}

	for _, c := range cases {
		platform, host := SplitGrossAmount(c.gross, c.bps)
		if platform+host != c.gross {
			t.Errorf("gross=%d bps=%d: platform(%d) + host(%d) = %d, want %d",
				c.gross, c.bps, platform, host, platform+host, c.gross)
		}
		if platform < 0 || host < 0 {
			t.Errorf("gross=%d bps=%d: negative split — platform=%d host=%d", c.gross, c.bps, platform, host)
		}
	}
}

// Proves ComputeTickCharge panics rather than silently corrupting state if called
// with a remainder outside its documented invariant — this is a programmer-error
// guard, not a runtime condition that should ever occur via the tick processor.
func TestComputeTickCharge_PanicsOnInvalidRemainder(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for out-of-range carriedRemainder, got none")
		}
	}()
	ComputeTickCharge(1000, 30, 3600) // 3600 is out of the documented [0,3599] range
}
