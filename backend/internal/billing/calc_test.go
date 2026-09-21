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

// TestSplitGrossAmount_ZeroSplitScenarios is a regression test for the incident that
// caused rental b91e32af-b9b8-4a16-8041-995822bc6d55's stop to return HTTP 500: a
// short final-settlement tick whose platform share rounds down to exactly 0 paise.
// ledger_entries has CHECK (amount_paise != 0), so tick.go's ProcessBillingTickTx must
// skip writing the platform_revenue (or host_payable) row entirely when its split is
// zero, rather than attempting to insert a zero-amount row. This test proves the split
// math that decision depends on, using the exact rate from the incident (900 paise/hr,
// 15% commission) — the actual insert-skipping logic lives in tick.go and isn't
// directly unit-testable here without a live database; this is the pure-math half of
// the fix's correctness.
func TestSplitGrossAmount_ZeroSplitScenarios(t *testing.T) {
	const ratePaisePerHour = int64(900) // the exact rate from the production incident

	cases := []struct {
		name             string
		elapsedSeconds   int64
		wantPlatformZero bool
		wantHostZero     bool
	}{
		// Elapsed=10s: gross=2 paise (900*10/3600, integer). This is the exact shape
		// of tick that caused the incident — a stop landing shortly after a heartbeat.
		{name: "10s final tick — platform rounds to zero, host does not", elapsedSeconds: 10, wantPlatformZero: true, wantHostZero: false},
		// Elapsed=27s: gross=6 paise — still below the threshold where 15% of the
		// charge reaches 1 paise. Confirms the zero-platform window is not a single
		// instant but a real, commonly-hit range (0 up to just under 28s elapsed).
		{name: "27s final tick — still zero platform", elapsedSeconds: 27, wantPlatformZero: true, wantHostZero: false},
		// Elapsed=28s: gross=7 paise exactly. 15% of 7 is 1.05, truncating to 1 —
		// the first elapsed-time value at this rate where platform is nonzero again.
		{name: "28s final tick — platform becomes nonzero", elapsedSeconds: 28, wantPlatformZero: false, wantHostZero: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gross, _ := ComputeTickCharge(ratePaisePerHour, c.elapsedSeconds, 0)
			platform, host := SplitGrossAmount(gross, PlatformCommissionBps)

			if platform+host != gross {
				t.Fatalf("reconciliation broken: platform(%d) + host(%d) != gross(%d)", platform, host, gross)
			}
			if (platform == 0) != c.wantPlatformZero {
				t.Errorf("elapsed=%ds gross=%d: platform=%d, wantZero=%v", c.elapsedSeconds, gross, platform, c.wantPlatformZero)
			}
			if (host == 0) != c.wantHostZero {
				t.Errorf("elapsed=%ds gross=%d: host=%d, wantZero=%v", c.elapsedSeconds, gross, host, c.wantHostZero)
			}
		})
	}
	// The symmetric case — host rounding to zero — is already covered by
	// TestSplitGrossAmount_AlwaysReconciles's {gross: 999, bps: 10000} case (100%
	// commission). It isn't reachable in production at the current fixed 15%
	// (PlatformCommissionBps), since host = gross - platform is always >= 85% of a
	// nonzero gross, but the fix in tick.go guards both sides symmetrically regardless.
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
