-- name: CreateOTP :one
INSERT INTO otp_codes (phone, code, purpose, full_name, password, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetLatestOTP :one
SELECT * FROM otp_codes
WHERE phone = $1 AND purpose = $2
ORDER BY created_at DESC
LIMIT 1;

-- name: IncrementOTPAttempts :exec
UPDATE otp_codes
SET attempts = attempts + 1
WHERE id = $1;

-- name: ConsumeOTP :exec
UPDATE otp_codes
SET consumed_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteOTPByPhonePurpose :exec
DELETE FROM otp_codes
WHERE phone = $1 AND purpose = $2;

-- name: UpdateUserPassword :one
UPDATE users
SET password = $2
WHERE id = $1
RETURNING *;
