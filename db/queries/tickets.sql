-- name: CreateTicket :one
INSERT INTO tickets (
    event_id, name, price, total_quantity
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, event_id, name, price, total_quantity, created_at, updated_at;

-- name: GetTicketByID :one
SELECT id, event_id, name, price, total_quantity, created_at, updated_at
FROM tickets
WHERE id = $1;

-- name: ListTicketsByEvent :many
SELECT id, event_id, name, price, total_quantity, created_at, updated_at
FROM tickets
WHERE event_id = $1
ORDER BY price ASC;

-- name: UpdateTicket :one
UPDATE tickets
SET 
    name = $2,
    price = $3,
    total_quantity = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, event_id, name, price, total_quantity, created_at, updated_at;

-- name: UpdateTicketStock :exec
UPDATE tickets
SET total_quantity = total_quantity - $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteTicket :exec
DELETE FROM tickets
WHERE id = $1;
