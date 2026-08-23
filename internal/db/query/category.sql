-- name: GetAllCategories :many
SELECT * FROM categories
ORDER BY name;

-- name: GetCategoryByID :one
SELECT * FROM categories
WHERE id = $1;

-- name: CreateCategory :one
INSERT INTO categories (name, parent_id)
VALUES ($1, $2)
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = $1;