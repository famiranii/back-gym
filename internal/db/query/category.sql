-- name: GetAllCategories :many
SELECT id, name, parent_id, image_url, created_at, updated_at FROM categories
ORDER BY name;

-- name: GetCategoryByID :one
SELECT id, name, parent_id, image_url, created_at, updated_at FROM categories
WHERE id = $1;

-- name: CreateCategory :one
INSERT INTO categories (name, parent_id, image_url)
VALUES (@name, @parent_id, @image_url)
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = $1;