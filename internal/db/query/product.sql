-- name: CreateProduct :one
INSERT INTO products (name, description, price, discount, category_id, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetProductByID :one
SELECT
    p.*,
    p.price - (p.price * p.discount / 100) AS final_price
FROM products p
WHERE p.id = $1;


-- name: GetAllProducts :many
SELECT p.*, c.name as category_name,
  (SELECT url FROM product_images 
   WHERE product_id = p.id AND is_primary = true 
   LIMIT 1) as primary_image,
   p.price - (p.price * p.discount / 100) AS final_price
FROM products p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.is_active = true
ORDER BY p.created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateProduct :one
UPDATE products
SET name = $2, description = $3, price = $4, discount = $5, category_id = $6, is_active = $7, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = $1;
