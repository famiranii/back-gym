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
    p.id          AS product_id,
    p.name        AS product_name,
    p.price       AS product_price,
    p.discount    AS product_discount,
    p.is_active   AS product_is_active,
    (SELECT url FROM product_images
     WHERE product_id = p.id AND is_primary = true
     LIMIT 1)     AS primary_image
FROM wishlists w
JOIN products p ON p.id = w.product_id
WHERE w.user_id = $1
ORDER BY w.created_at DESC;

-- name: IsInWishlist :one
SELECT EXISTS (
    SELECT 1 FROM wishlists
    WHERE user_id = $1 AND product_id = $2
) AS exists;

