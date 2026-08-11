-- name: CreatePaste :exec
INSERT INTO paste (id, filename, detected_language, content, size_bytes, expires_at, created_at)
VALUES (@id, @filename, @detected_language, @content, @size_bytes, @expires_at, @created_at);

-- name: GetPaste :one
SELECT *
FROM paste
WHERE id = @id;

-- name: IncrementViewCount :exec
UPDATE paste
SET view_count = view_count + 1
WHERE id = @id;

-- name: DeleteExpiredPastes :exec
DELETE
FROM paste
WHERE expires_at < @timestamp;