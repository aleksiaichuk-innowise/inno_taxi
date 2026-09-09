CREATE TABLE IF NOT EXISTS order_events (
    order_id String,
    user_id String,
    taxi_type LowCardinality(String),
    status LowCardinality(String),
    order_created_at DateTime64(3),
    ingested_at DateTime64(3) DEFAULT now64(3)
) ENGINE = ReplacingMergeTree(ingested_at)
ORDER BY (order_id);
