-- +goose Up
CREATE TABLE paste
(
    id                text PRIMARY KEY NOT NULL,
    filename          text,
    detected_language text             NOT NULL,
    content           text             NOT NULL,
    size_bytes        integer          NOT NULL,
    view_count        integer          NOT NULL DEFAULT 0,
    -- Timestamps are ISO8601
    expires_at        text             NOT NULL,
    created_at        text             NOT NULL
);

CREATE INDEX paste_expires_at_idx ON paste (expires_at);

-- +goose Down
DROP TABLE IF EXISTS paste;