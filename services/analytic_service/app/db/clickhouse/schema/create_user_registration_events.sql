CREATE TABLE IF NOT EXISTS user_registration_events (
    user_id String,
    name String,
    email String,
    phone String,
    role LowCardinality(String),
    registered_at DateTime64(3),
    ingested_at DateTime64(3) DEFAULT now64(3)
) ENGINE = ReplacingMergeTree(ingested_at)
ORDER BY (user_id);
