-- name: AddToWishlist :one

INSERT INTO wishlists (user_id, product_id)
VALUES ($1, $2)
ON CONFLICT (user_id, product_id) DO NOTHING
RETURNING user_id, product_id, created_at;


-- name: RemoveFromWishlist :exec

DELETE FROM wishlists
WHERE user_id = $1
  AND product_id = $2;


-- name: GetWishlistByUserID :many

SELECT
    p.id AS id,
    w.created_at,
    p.name,
    p.price,
    p.discount,
    p.is_active,
    c.name AS category_name,
    (
        SELECT pi.url
        FROM product_images pi
        WHERE pi.product_id = p.id
          AND pi.is_primary = TRUE
        ORDER BY pi.created_at ASC
        LIMIT 1
    ) AS primary_image,
    (
        p.price - (p.price * p.discount / 100)
    )::bigint AS final_price
FROM wishlists w
JOIN products p
    ON p.id = w.product_id
LEFT JOIN categories c
    ON c.id = p.category_id
WHERE w.user_id = $1
ORDER BY w.created_at DESC;


-- name: IsInWishlist :one

SELECT EXISTS (
    SELECT 1
    FROM wishlists
    WHERE user_id = $1
      AND product_id = $2
) AS exists;