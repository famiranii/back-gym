
-- name: AddToCart :one
INSERT INTO cart_items (user_id, variant_id, quantity)
VALUES (@user_id, @variant_id, @quantity)
ON CONFLICT (user_id, variant_id) DO UPDATE
SET
    quantity = cart_items.quantity + EXCLUDED.quantity,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;


-- name: GetCart :many
SELECT
    ci.id,
    ci.quantity,
    ci.variant_id,

    pv.label,
    pv.color,
    pv.stock,

    p.id AS product_id,
    p.name AS product_name,
    p.price,
    p.discount,

    -- قیمت نهایی بعد از تخفیف
    ROUND(p.price * (1 - COALESCE(p.discount, 0) / 100)) AS final_price,

    pi.url AS image_url

FROM cart_items ci

JOIN product_variants pv
    ON pv.id = ci.variant_id

JOIN products p
    ON p.id = pv.product_id

LEFT JOIN LATERAL (
    SELECT url
    FROM product_images
    WHERE product_id = p.id
    ORDER BY created_at
    LIMIT 1
) pi ON true

WHERE ci.user_id = @user_id

ORDER BY ci.created_at DESC;


-- name: UpdateCartItemQuantity :one
UPDATE cart_items
SET
    quantity = @quantity,
    updated_at = CURRENT_TIMESTAMP
WHERE id = @id
  AND user_id = @user_id
RETURNING *;


-- name: RemoveFromCart :exec
DELETE FROM cart_items
WHERE id = @id
  AND user_id = @user_id;


-- name: ClearCart :exec
DELETE FROM cart_items
WHERE user_id = @user_id;