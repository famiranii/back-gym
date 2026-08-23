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

