-- name: GetShippingCost :one
SELECT cost FROM shipping_settings WHERE id = 1;

-- name: UpdateShippingCost :one
UPDATE shipping_settings
SET cost = $1
WHERE id = 1
RETURNING *;