CREATE TABLE IF NOT EXISTS driver_ratings (
    order_id String,
    driver_id String,
    rating Int32,
    comment String,
    rated_at DateTime64(3),
    ingested_at DateTime64(3) DEFAULT now64(3)
) ENGINE = ReplacingMergeTree(ingested_at)
ORDER BY (order_id);
