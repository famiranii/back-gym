-- name: CreateProductImage :one
INSERT INTO product_images (product_id, url, is_primary)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetProductImages :many
SELECT * FROM product_images
WHERE product_id = $1
ORDER BY is_primary DESC, created_at ASC;

-- name: DeleteProductImage :exec
DELETE FROM product_images
WHERE id = $1;