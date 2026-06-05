-- name: InsertOutboxEvent :one
INSERT INTO outbox_events (id, household_id, event_type, payload)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: FetchPendingOutboxEvents :many
SELECT * FROM outbox_events
WHERE published_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events SET published_at = NOW() WHERE id = $1;
