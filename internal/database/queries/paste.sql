-- name: CreatePaste :exec
INSERT INTO paste (id, filename, detected_language, content, size_bytes, expires_at, created_at, burn_after_read)
VALUES (@id, @filename, @detected_language, @content, @size_bytes, @expires_at, @created_at, @burn_after_read);

-- name: GetPaste :one
SELECT *
FROM paste
WHERE id = @id AND expires_at > @now;

-- name: IncrementViewCount :exec
UPDATE paste
SET view_count = view_count + 1
WHERE id = @id;

-- name: DeletePasteByID :exec
DELETE FROM paste WHERE id = @id;

-- name: DeleteExpiredPastes :exec
DELETE
FROM paste
WHERE expires_at < @timestamp;