-- +goose Up
ALTER TABLE orders ADD COLUMN rating INTEGER;
ALTER TABLE orders ADD COLUMN comment TEXT;
ALTER TABLE orders ADD CONSTRAINT orders_rating_range CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5));

-- +goose Down
ALTER TABLE orders DROP CONSTRAINT orders_rating_range;
ALTER TABLE orders DROP COLUMN comment;
ALTER TABLE orders DROP COLUMN rating;
