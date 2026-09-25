-- name: CreatePayment :one
INSERT INTO payments (
    order_id,
    amount,
    authority,
    expires_at
)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetPaymentByAuthority :one
SELECT *
FROM payments
WHERE authority = $1
LIMIT 1;


-- name: GetPaymentByOrderID :one
SELECT *
FROM payments
WHERE order_id = $1
ORDER BY id DESC
LIMIT 1;


-- name: MarkPaymentPaid :one
UPDATE payments
SET
    status = 'paid',
    ref_id = $2,
    card_pan = $3,
    card_hash = $4,
    fee_type = $5,
    fee = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;


-- name: MarkPaymentFailed :one
UPDATE payments
SET
    status = 'failed',
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetPaymentByAuthorityForUpdate :one
SELECT *
FROM payments
WHERE authority = $1
FOR UPDATE;

-- name: MarkPaymentExpired :one
UPDATE payments
SET status = 'expired'
WHERE id = $1
  AND status = 'pending'
RETURNING *;