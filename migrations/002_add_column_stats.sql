-- +goose Up

ALTER TABLE urls
ADD COLUMN access_count BIGINT NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE urls
DROP COLUMN access_count;