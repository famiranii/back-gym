-- name: GetValidDiscountCode :one
SELECT
    id,
    code,
    discount_type,
    discount_value,
    min_order_amount,
    max_discount_amount,
    usage_limit,
    used_count,
    starts_at,
    expires_at,
    is_active,
    created_at,
    updated_at
FROM discount_codes
WHERE code = $1
  AND is_active = TRUE
  AND (starts_at IS NULL OR starts_at <= CURRENT_TIMESTAMP)
  AND (expires_at IS NULL OR expires_at >= CURRENT_TIMESTAMP)
  AND (usage_limit IS NULL OR used_count < usage_limit)
LIMIT 1;

-- name: IncrementDiscountCodeUsage :exec
UPDATE discount_codes
SET
    used_count = used_count + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND is_active = TRUE
  AND (usage_limit IS NULL OR used_count < usage_limit);

-- name: ListDiscountCodes :many
SELECT
    id,
    code,
    discount_type,
    discount_value,
    min_order_amount,
    max_discount_amount,
    usage_limit,
    used_count,
    starts_at,
    expires_at,
    is_active,
    created_at,
    updated_at
FROM discount_codes
ORDER BY created_at DESC;


-- name: DeleteDiscountCode :exec
DELETE FROM discount_codes
WHERE id = $1
  AND used_count = 0;

-- name: CreateDiscountCode :one
INSERT INTO discount_codes (
    code,
    discount_type,
    discount_value,
    min_order_amount,
    max_discount_amount,
    usage_limit,
    starts_at,
    expires_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING
    id,
    code,
    discount_type,
    discount_value,
    min_order_amount,
    max_discount_amount,
    usage_limit,
    used_count,
    starts_at,
    expires_at,
    is_active,
    created_at,
    updated_at;