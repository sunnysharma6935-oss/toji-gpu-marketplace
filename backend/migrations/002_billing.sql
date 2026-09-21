-- Billing ledger schema (v3). Additive — does not touch users/hosts/gpus/listings/rentals.
-- All amounts are integer paise. INR only for this MVP (currency column kept for future-proofing,
-- but application code should reject anything but 'INR' for now — see billing.go).

CREATE TABLE ledger_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_type    TEXT NOT NULL CHECK (account_type IN
                        ('customer_balance', 'host_payable', 'platform_revenue', 'payment_provider_clearing')),
    owner_type      TEXT NOT NULL CHECK (owner_type IN ('customer', 'host', 'platform', 'provider')),
    owner_id        UUID, -- NULL for platform-wide accounts (platform_revenue has exactly one row)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_type, owner_id)
);

-- Seed the one platform-wide revenue account. Application code looks this up by
-- (account_type='platform_revenue', owner_id IS NULL) rather than hardcoding an ID.
INSERT INTO ledger_accounts (account_type, owner_type, owner_id)
VALUES ('platform_revenue', 'platform', NULL);

CREATE TABLE payment_intents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    direction           TEXT NOT NULL CHECK (direction IN ('top_up', 'payout')),
    customer_id         UUID REFERENCES users(id),
    host_id             UUID REFERENCES hosts(id),
    amount_paise        BIGINT NOT NULL CHECK (amount_paise > 0),
    currency            TEXT NOT NULL DEFAULT 'INR' CHECK (currency = 'INR'),
    method              TEXT NOT NULL CHECK (method IN ('upi', 'card', 'netbanking', 'manual')),
    provider            TEXT NOT NULL DEFAULT 'mock' CHECK (provider IN ('mock', 'manual', 'razorpay', 'cashfree')),
    provider_reference  TEXT,
    status              TEXT NOT NULL DEFAULT 'initiated' CHECK (status IN ('initiated', 'pending', 'succeeded', 'failed')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((direction = 'top_up' AND customer_id IS NOT NULL AND host_id IS NULL)
        OR (direction = 'payout' AND host_id IS NOT NULL AND customer_id IS NULL))
);

CREATE TABLE rental_usage_events (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rental_id               UUID NOT NULL REFERENCES rentals(id), -- reference only, never duplicated data
    period_start            TIMESTAMPTZ NOT NULL,
    period_end              TIMESTAMPTZ NOT NULL,
    elapsed_seconds         INT NOT NULL CHECK (elapsed_seconds >= 0),
    rate_paise_per_hour     BIGINT NOT NULL,
    commission_rate_bps     INT NOT NULL CHECK (commission_rate_bps BETWEEN 0 AND 10000),
    gross_amount_paise      BIGINT NOT NULL CHECK (gross_amount_paise >= 0),
    platform_revenue_paise  BIGINT NOT NULL CHECK (platform_revenue_paise >= 0),
    host_payable_paise      BIGINT NOT NULL CHECK (host_payable_paise >= 0),
    is_final_settlement     BOOLEAN NOT NULL DEFAULT false,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- the reconciliation invariant, enforced by the database itself, not just application code
    CHECK (gross_amount_paise = platform_revenue_paise + host_payable_paise)
);

CREATE TABLE ledger_entries (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id              UUID NOT NULL REFERENCES ledger_accounts(id),
    amount_paise            BIGINT NOT NULL, -- signed; positive=credit, negative=debit. Never zero.
    currency                TEXT NOT NULL DEFAULT 'INR' CHECK (currency = 'INR'),
    entry_type              TEXT NOT NULL CHECK (entry_type IN
                                ('top_up', 'rental_charge', 'platform_revenue', 'host_payable',
                                 'payout', 'refund', 'credit', 'reversal')),
    status                  TEXT NOT NULL DEFAULT 'completed' CHECK (status IN ('pending', 'completed', 'failed', 'reversed')),
    event_id                UUID REFERENCES rental_usage_events(id),
    rental_id               UUID REFERENCES rentals(id), -- reference only
    payment_intent_id       UUID REFERENCES payment_intents(id),
    reversed_by_entry_id    UUID REFERENCES ledger_entries(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (amount_paise != 0)
);

-- Ledger rows are never updated or deleted by application code. This trigger makes that a hard
-- database guarantee, not just a convention — the only two exceptions are setting reversed_by_entry_id
-- (linking to a correction) and status pending->completed/failed (settling an in-flight entry).
CREATE OR REPLACE FUNCTION forbid_ledger_entry_mutation() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.amount_paise != NEW.amount_paise
        OR OLD.account_id != NEW.account_id
        OR OLD.entry_type != NEW.entry_type
        OR OLD.rental_id IS DISTINCT FROM NEW.rental_id
        OR OLD.event_id IS DISTINCT FROM NEW.event_id THEN
        RAISE EXCEPTION 'ledger_entries rows are immutable — insert a new row instead of editing this one';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_ledger_entries_immutable
    BEFORE UPDATE ON ledger_entries
    FOR EACH ROW EXECUTE FUNCTION forbid_ledger_entry_mutation();

CREATE TABLE rental_billing_state (
    rental_id           UUID PRIMARY KEY REFERENCES rentals(id),
    rate_paise_per_hour BIGINT NOT NULL,
    last_billed_at      TIMESTAMPTZ NOT NULL,
    carried_remainder   INT NOT NULL DEFAULT 0 CHECK (carried_remainder BETWEEN 0 AND 3599)
);

CREATE TABLE reservations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID NOT NULL REFERENCES ledger_accounts(id),
    rental_id       UUID NOT NULL REFERENCES rentals(id),
    amount_paise    BIGINT NOT NULL CHECK (amount_paise >= 0),
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'released', 'consumed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_entries_account ON ledger_entries(account_id) WHERE status = 'completed';
CREATE INDEX idx_ledger_entries_rental ON ledger_entries(rental_id);
CREATE INDEX idx_reservations_account_active ON reservations(account_id) WHERE status = 'active';
CREATE INDEX idx_rue_rental ON rental_usage_events(rental_id);
