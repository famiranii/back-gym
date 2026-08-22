-- name: CreateUser :one
INSERT INTO users (first_name, last_name, phone, password)
VALUES ($1, $2, $3, $4)
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

