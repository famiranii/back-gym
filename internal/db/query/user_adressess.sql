-- internal/db/query/user_address.sql

-- name: GetUserAddresses :many
SELECT * FROM user_addresses
WHERE user_id = @user_id
ORDER BY is_default DESC, created_at DESC;

-- name: CreateUserAddress :one
INSERT INTO user_addresses (user_id, title, province, city, address, postal_code, lat, lng, is_default)
VALUES (@user_id, @title, @province, @city, @address, @postal_code, @lat, @lng, @is_default)
RETURNING *;

-- name: UpdateUserAddress :one
UPDATE user_addresses
SET
    title       = @title,
    province    = @province,
    city        = @city,
    address     = @address,
    postal_code = @postal_code,
    lat         = @lat,
    lng         = @lng,
    is_default  = @is_default,
    updated_at  = CURRENT_TIMESTAMP
WHERE id = @id AND user_id = @user_id
RETURNING *;

-- name: DeleteUserAddress :exec
DELETE FROM user_addresses
WHERE id = @id AND user_id = @user_id;

-- name: SetDefaultAddress :exec
UPDATE user_addresses
SET
    is_default = (id = @id),
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = @user_id;

-- name: GetUserAddress :one
SELECT *
FROM user_addresses
WHERE id = @id
  AND user_id = @user_id
LIMIT 1;