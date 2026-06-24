-- name: CreateEvent :one
INSERT INTO events (
    title, description, location, event_date
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, title, description, location, event_date, created_at, updated_at;

-- name: GetEventByID :one
SELECT id, title, description, location, event_date, created_at, updated_at
FROM events
WHERE id = $1;

-- name: ListEvents :many
SELECT id, title, description, location, event_date, created_at, updated_at
FROM events
ORDER BY event_date ASC
LIMIT $1 OFFSET $2;

-- name: UpdateEvent :one
UPDATE events
SET 
    title = $2,
    description = $3,
    location = $4,
    event_date = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, title, description, location, event_date, created_at, updated_at;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE id = $1;
