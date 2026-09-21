-- Rental lifecycle additions for billing integration. Additive only — no existing
-- column touched, no existing table altered beyond this one new nullable column.

ALTER TABLE rentals ADD COLUMN status_reason TEXT;
-- Used for BOTH 'stopped' and 'failed' terminal states — one field, not two, since
-- both are "why did this rental end" and duplicating the column would be unnecessary
-- schema growth for the same underlying question.
-- Expected values (convention, not enforced by CHECK — keeps this flexible while the
-- exact set of reasons is still settling):
--   customer_requested | insufficient_balance | cancelled_before_start | provisioning_failed

-- Records money that was owed but could not be charged because it would have driven
-- the customer's balance negative. This is NOT a ledger_entries row — a shortfall is
-- money that did NOT move, and every ledger_entries row must represent money that did.
-- No recovery/collection logic exists yet, per the approved decision — this table only
-- records the fact so it is never silently lost.
CREATE TABLE rental_shortfalls (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rental_id       UUID NOT NULL REFERENCES rentals(id),
    amount_paise    BIGINT NOT NULL CHECK (amount_paise > 0),
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
