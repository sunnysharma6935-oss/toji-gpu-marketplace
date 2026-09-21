-- Phase 1 schema. Only what Day 1-3 through Week 4 actually needs.
-- No speculative columns for features we haven't built yet.

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    credit_balance_paise BIGINT NOT NULL DEFAULT 0, -- store money in paise (int), never float
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A host is a machine owner. One user can own multiple hosts (multiple PCs).
CREATE TABLE hosts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    machine_label   TEXT NOT NULL,            -- e.g. "PC1" - host-chosen friendly name
    agent_token     TEXT UNIQUE NOT NULL,     -- long random token the agent uses to auth itself
    status          TEXT NOT NULL DEFAULT 'pending', -- pending | online | offline
    last_heartbeat_at TIMESTAMPTZ,
    reliability_score NUMERIC(4,3) NOT NULL DEFAULT 1.000, -- 0.000 to 1.000, updated later
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A GPU belongs to a host. One host (PC) can have multiple GPUs (your PCs have 2 each).
CREATE TABLE gpus (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id         UUID NOT NULL REFERENCES hosts(id),
    gpu_index       INT NOT NULL,             -- index as reported by nvidia-smi (0, 1, ...)
    model           TEXT NOT NULL,            -- e.g. "RTX 5060"
    vram_mb         INT NOT NULL,
    driver_version  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(host_id, gpu_index)
);

-- Built later (Week 2+) but declared now so migrations don't need reshuffling:
CREATE TABLE listings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gpu_id          UUID NOT NULL REFERENCES gpus(id),
    price_paise_per_hour BIGINT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rentals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id      UUID NOT NULL REFERENCES listings(id),
    customer_id     UUID NOT NULL REFERENCES users(id),
    status          TEXT NOT NULL DEFAULT 'provisioning', -- provisioning | running | stopped | failed
    ssh_port        INT,
    jupyter_token   TEXT,
    started_at      TIMESTAMPTZ,
    stopped_at      TIMESTAMPTZ,
    total_paise_billed BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE usage_heartbeats (
    id              BIGSERIAL PRIMARY KEY,
    rental_id       UUID NOT NULL REFERENCES rentals(id),
    reported_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    gpu_util_pct    INT
);

CREATE TABLE transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    type            TEXT NOT NULL, -- topup | charge | payout
    amount_paise    BIGINT NOT NULL,
    gateway_ref     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_hosts_status ON hosts(status);
CREATE INDEX idx_listings_active ON listings(is_active);
CREATE INDEX idx_rentals_status ON rentals(status);
