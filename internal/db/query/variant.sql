-- name: CreateVariant :one
INSERT INTO product_variants (product_id, label, color, stock)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetVariantsByProductID :many
SELECT * FROM product_variants
WHERE product_id = $1
ORDER BY created_at ASC;

-- name: UpdateVariantStock :one
UPDATE product_variants
SET stock = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteVariant :exec
DELETE FROM product_variants
WHERE id = $1;

-- name: GetVariantByID :one
SELECT * FROM product_variants
WHERE id = $1;

-- name: UpdateVariant :one
UPDATE product_variants
SET
    label = $2,
    color = $3,
    stock = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DecreaseVariantStock :exec
UPDATE product_variants
SET stock = stock - $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND stock >= $2;