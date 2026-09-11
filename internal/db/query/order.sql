-- name: CreateOrder :one
INSERT INTO orders (
    user_id,
    status,
    shipping_cost,
    total_price,
    address_title,
    address_province,
    address_city,
    address_detail,
    address_postal_code,
    address_lat,
    address_lng
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    order_id,
    variant_id,
    product_id,
    quantity,
    unit_price,
    total_price
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetOrdersByUser :many
SELECT * FROM orders
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetOrderByID :one
SELECT
    o.*,
    u.phone AS user_phone
FROM orders o
JOIN users u ON u.id = o.user_id
WHERE o.id = $1
LIMIT 1;

-- name: GetOrderItems :many
SELECT
    oi.id,
    oi.order_id,
    oi.variant_id,
    oi.product_id,
    oi.quantity,
    oi.unit_price,
    oi.total_price,
    p.name AS product_name,
    pv.label,
    pv.color,
    pi.url AS image_url
FROM order_items oi
JOIN products p ON p.id = oi.product_id
JOIN product_variants pv ON pv.id = oi.variant_id
LEFT JOIN LATERAL (
    SELECT url FROM product_images
    WHERE product_id = p.id
    ORDER BY created_at
    LIMIT 1
) pi ON true
WHERE oi.order_id = $1;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $1, updated_at = NOW()
WHERE id = $2
RETURNING *;


-- name: GetOrdersByUserAndStatus :many
SELECT *
FROM orders
WHERE user_id = $1
  AND status = $2
ORDER BY created_at DESC
LIMIT $3
OFFSET $4;