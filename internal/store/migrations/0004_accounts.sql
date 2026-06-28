-- Application accounts for (opt-in) dashboard authentication. Distinct from the
-- OpenTelemetry "end users" (user.id) aggregated from spans.

CREATE TABLE IF NOT EXISTS accounts (
    id            VARCHAR PRIMARY KEY,
    email         VARCHAR NOT NULL UNIQUE,
    password_hash VARCHAR NOT NULL,
    role          VARCHAR NOT NULL DEFAULT 'member', -- admin | member
    created_at    TIMESTAMP DEFAULT now()
);
