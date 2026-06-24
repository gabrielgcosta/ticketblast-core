-- name: CreateOrder :one
INSERT INTO orders (
    user_id, status, total_amount
) VALUES (
    $1, $2, $3
)
RETURNING id, user_id, status, total_amount, created_at, updated_at;

-- name: GetOrderByID :one
SELECT id, user_id, status, total_amount, created_at, updated_at
FROM orders
WHERE id = $1;

-- name: ListOrdersByUser :many
SELECT id, user_id, status, total_amount, created_at, updated_at
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateOrderStatus :one
UPDATE orders
SET 
    status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, user_id, status, total_amount, created_at, updated_at;

-- name: DeleteOrder :exec
DELETE FROM orders
WHERE id = $1;
