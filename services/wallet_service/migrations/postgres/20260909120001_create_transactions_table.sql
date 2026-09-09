-- +goose Up
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    type text NOT NULL,
    amount_minor_units BIGINT NOT NULL,
    reference_id text NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (wallet_id, reference_id, type)
);

-- +goose Down
DROP TABLE IF EXISTS transactions;
