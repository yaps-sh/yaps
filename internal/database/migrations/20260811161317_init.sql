-- +goose Up
create table if not exists paste (
    id text primary key,
    filename text,
    detected_language text not null,
    content text not null,
    size_bytes integer not null,
    view_count integer not null default 0,
    -- Timestamps are ISO8601
    expires_at text not null,
    created_at text not null
);

create index if not exists paste_expires_at_idx on paste (expires_at);

-- +goose Down
DROp table if exists paste;