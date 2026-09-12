-- name: CreateUser :one
INSERT INTO users (full_name, phone, password)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByPhone :one
SELECT * FROM users
WHERE phone = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetAllUsers :many
SELECT *
FROM users
ORDER BY created_at DESC;

-- name: UpdateUser :one
UPDATE users
SET
    full_name = $2,
    phone = $3,
    password = $4
WHERE id = $1
RETURNING *;

-- name: SearchUsers :many
SELECT id, full_name, phone, created_at
FROM users
WHERE
    full_name ILIKE '%' || sqlc.arg(query)::text || '%'
    OR phone ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY created_at DESC
LIMIT 20;
