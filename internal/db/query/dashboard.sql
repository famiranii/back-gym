-- name: GetTodayOrdersCount :one
SELECT COUNT(*)
FROM orders
WHERE created_at >= CURRENT_DATE;

-- name: GetTodaySales :one
SELECT COALESCE(SUM(total_price), 0)
FROM orders
WHERE status IN ('paid', 'shipped', 'delivered')
  AND created_at >= CURRENT_DATE;

-- name: GetPendingOrdersCount :one
SELECT COUNT(*)
FROM orders
WHERE status = 'pending';


-- name: GetPaidOrdersCount :one
SELECT COUNT(*)
FROM orders
WHERE status = 'paid';

-- name: GetSalesLast7Days :many
SELECT
    d::date AS date,
    COALESCE(SUM(o.total_price), 0)::bigint AS amount
FROM generate_series(
    CURRENT_DATE - INTERVAL '6 days',
    CURRENT_DATE,
    INTERVAL '1 day'
) AS d
LEFT JOIN orders o
    ON o.created_at::date = d::date
    AND o.status IN ('paid', 'shipped', 'delivered')
GROUP BY d::date
ORDER BY d::date;

-- name: GetRecentOrders :many
SELECT
    o.id,
    o.user_id,
    u.full_name AS customer_name,
    o.total_price,
    o.status,
    o.created_at
FROM orders o
JOIN users u ON u.id = o.user_id
ORDER BY o.created_at DESC
LIMIT 10;

-- name: GetLowStockProducts :many
SELECT
    p.id,
    p.name,
    COALESCE(SUM(pv.stock), 0)::bigint AS stock
FROM products p
LEFT JOIN product_variants pv
    ON pv.product_id = p.id
WHERE p.is_active = TRUE
GROUP BY p.id, p.name
HAVING COALESCE(SUM(pv.stock), 0) <= 5
ORDER BY stock ASC, p.name ASC
LIMIT 10;

-- name: GetTopProducts :many
SELECT
    p.id,
    p.name,
    COALESCE(SUM(oi.quantity), 0)::bigint AS sold
FROM order_items oi
JOIN orders o
    ON o.id = oi.order_id
JOIN products p
    ON p.id = oi.product_id
WHERE o.status IN ('paid', 'shipped', 'delivered')
GROUP BY p.id, p.name
ORDER BY sold DESC
LIMIT 10;

