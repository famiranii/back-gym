-- internal/db/query/review.sql


-- =========================================================
-- User Reviews
-- =========================================================

-- name: UpsertReview :one
INSERT INTO reviews (
    product_id,
    user_id,
    rating,
    body,
    updated_at
)
VALUES (
    @product_id,
    @user_id,
    @rating,
    @body,
    CURRENT_TIMESTAMP
)
ON CONFLICT (product_id, user_id) DO UPDATE
SET
    rating = EXCLUDED.rating,
    body = EXCLUDED.body,
    status = 'pending',
    updated_at = CURRENT_TIMESTAMP
RETURNING
    id,
    product_id,
    user_id,
    rating,
    body,
    status,
    created_at,
    updated_at;


-- name: GetReviewsByProductID :many
SELECT
    r.id,
    r.product_id,
    r.user_id,
    u.full_name AS user_name,
    r.rating,
    r.body,
    r.created_at,
    r.updated_at
FROM reviews r
JOIN users u
    ON u.id = r.user_id
WHERE r.product_id = $1
  AND r.status = 'approved'
ORDER BY r.created_at DESC;


-- name: GetAverageRating :one
SELECT
    COALESCE(
        AVG(rating::FLOAT)
        FILTER (WHERE status = 'approved'),
        0
    )::FLOAT AS average_rating
FROM reviews
WHERE product_id = @product_id;


-- name: DeleteReview :exec
DELETE FROM reviews
WHERE product_id = @product_id
  AND user_id = @user_id;


-- =========================================================
-- Admin Reviews
-- =========================================================

-- name: GetPendingReviews :many
SELECT
    r.id,
    r.product_id,
    r.user_id,
    u.full_name AS user_name,
    p.name AS product_name,
    r.rating,
    r.body,
    r.status,
    r.created_at,
    r.updated_at
FROM reviews r
JOIN users u
    ON u.id = r.user_id
JOIN products p
    ON p.id = r.product_id
WHERE r.status = 'pending'
ORDER BY r.created_at ASC;


-- name: ApproveReview :one
UPDATE reviews
SET
    status = 'approved',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND status = 'pending'
RETURNING
    id,
    product_id,
    user_id,
    rating,
    body,
    status,
    created_at,
    updated_at;


-- name: RejectReview :one
UPDATE reviews
SET
    status = 'rejected',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND status = 'pending'
RETURNING
    id,
    product_id,
    user_id,
    rating,
    body,
    status,
    created_at,
    updated_at;