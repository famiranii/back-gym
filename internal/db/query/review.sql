-- internal/db/query/review.sql

-- name: UpsertReview :one
INSERT INTO reviews (product_id, user_id, rating, body, updated_at)
VALUES (@product_id, @user_id, @rating, @body, CURRENT_TIMESTAMP)
ON CONFLICT (product_id, user_id) DO UPDATE
SET
    rating     = EXCLUDED.rating,
    body       = EXCLUDED.body,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetReviewsByProductID :many
SELECT
    r.id,
    r.product_id,
    r.user_id,
    u.full_name,
    r.rating,
    r.body,
    r.created_at,
    r.updated_at
FROM reviews r
JOIN users u ON u.id = r.user_id
WHERE r.product_id = @product_id
ORDER BY r.created_at DESC;

-- name: GetAverageRating :one
SELECT COALESCE(AVG(rating::FLOAT), 0)::FLOAT AS average_rating
FROM reviews
WHERE product_id = @product_id;

-- name: DeleteReview :exec
DELETE FROM reviews
WHERE product_id = @product_id AND user_id = @user_id;