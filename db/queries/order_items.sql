-- name: CreateOrderItem :one
INSERT INTO order_items (
    order_id, ticket_id, quantity, unit_price
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, order_id, ticket_id, quantity, unit_price, created_at, updated_at;

-- name: GetOrderItemByID :one
SELECT id, order_id, ticket_id, quantity, unit_price, created_at, updated_at
FROM order_items
WHERE id = $1;

-- name: ListOrderItemsByOrder :many
SELECT id, order_id, ticket_id, quantity, unit_price, created_at, updated_at
FROM order_items
WHERE order_id = $1
ORDER BY created_at ASC;
