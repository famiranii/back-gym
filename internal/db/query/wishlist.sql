-- name: AddToWishlist :one
INSERT INTO wishlists (user_id, product_id)
VALUES ($1, $2)
ON CONFLICT (user_id, product_id) DO NOTHING
RETURNING *;

-- name: RemoveFromWishlist :exec
DELETE FROM wishlists
WHERE user_id = $1 AND product_id = $2;

-- name: GetWishlistByUserID :many

SELECT
    w.id,
    w.created_at,
    p.id AS id,
    p.name AS name,
    p.price AS price,
    p.discount AS discount,
    p.is_active AS is_active,
    c.name AS category_name,
    (
        SELECT url
        FROM product_images
        WHERE product_id = p.id
          AND is_primary = true
        LIMIT 1
    ) AS primary_image,
    p.price - (p.price * p.discount / 100) AS final_price
FROM wishlists w
JOIN products p
    ON p.id = w.product_id
LEFT JOIN categories c
    ON c.id = p.category_id
WHERE w.user_id = $1
ORDER BY w.created_at DESC;

-- name: IsInWishlist :one
SELECT EXISTS (
    SELECT 1 FROM wishlists
    WHERE user_id = $1 AND product_id = $2
) AS exists;

