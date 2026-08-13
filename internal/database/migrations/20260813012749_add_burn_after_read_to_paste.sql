-- +goose Up
ALTER TABLE paste
    ADD COLUMN burn_after_read INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE paste
    DROP COLUMN burn_after_read;