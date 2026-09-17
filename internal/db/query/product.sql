-- name: CreateProduct :one
INSERT INTO products (name, description, price, discount, category_id, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetProductByID :one
SELECT
    p.*,
    p.price - (p.price * p.discount / 100) AS final_price
FROM products p
WHERE p.id = $1;

-- name: GetAllProducts :many
SELECT
    p.id,
    p.name,
    p.description,
    p.price,
    p.discount,
    p.category_id,
    p.is_active,
    p.created_at,
    p.updated_at,

    COALESCE(
        (
            SELECT pi.url
            FROM product_images pi
            WHERE pi.product_id = p.id
              AND pi.is_primary = TRUE
            ORDER BY pi.created_at ASC
            LIMIT 1
        ),
        ''
    ) AS primary_image,

    (
        p.price - (p.price * p.discount / 100)
    )::bigint AS final_price,

    COALESCE(
        (
            SELECT SUM(oi.quantity)
            FROM order_items oi
            JOIN orders o
                ON o.id = oi.order_id
            WHERE oi.product_id = p.id
              AND o.status IN ('paid', 'shipped', 'delivered')
        ),
        0
    )::bigint AS sold_count,

    COALESCE(
        (
            SELECT ROUND(AVG(r.rating), 1)
            FROM reviews r
            WHERE r.product_id = p.id
              AND r.rating IS NOT NULL
        ),
        0
    )::numeric AS rating,

    (
        SELECT COUNT(*)
        FROM reviews r
        WHERE r.product_id = p.id
          AND r.rating IS NOT NULL
    )::bigint AS rating_count

FROM products p

WHERE p.is_active = TRUE

ORDER BY
    CASE
        WHEN sqlc.arg('sort')::text = 'price_asc'
        THEN p.price - (p.price * p.discount / 100)
    END ASC NULLS LAST,

    CASE
        WHEN sqlc.arg('sort')::text = 'price_desc'
        THEN p.price - (p.price * p.discount / 100)
    END DESC NULLS LAST,

    CASE
        WHEN sqlc.arg('sort')::text = 'discount'
        THEN p.discount
    END DESC NULLS LAST,

    CASE
        WHEN sqlc.arg('sort')::text = 'best_selling'
        THEN COALESCE(
            (
                SELECT SUM(oi.quantity)
                FROM order_items oi
                JOIN orders o
                    ON o.id = oi.order_id
                WHERE oi.product_id = p.id
                  AND o.status IN ('paid', 'shipped', 'delivered')
            ),
            0
        )
    END DESC NULLS LAST,

    p.created_at DESC,
    p.id DESC

LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');


-- name: SearchProducts :many
SELECT
    p.id,
    p.name,
    p.description,
    p.price,
    p.discount,
    p.category_id,
    p.is_active,
    p.created_at,
    p.updated_at,

    COALESCE(
        (
            SELECT pi.url
            FROM product_images pi
            WHERE pi.product_id = p.id
              AND pi.is_primary = TRUE
            ORDER BY pi.created_at ASC
            LIMIT 1
        ),
        ''
    ) AS primary_image,

    (
        p.price - (p.price * p.discount / 100)
    )::bigint AS final_price,

    COALESCE(
        (
            SELECT SUM(oi.quantity)
            FROM order_items oi
            JOIN orders o
                ON o.id = oi.order_id
            WHERE oi.product_id = p.id
              AND o.status IN ('paid', 'shipped', 'delivered')
        ),
        0
    )::bigint AS sold_count

FROM products p

WHERE p.is_active = TRUE
  AND (
      p.name ILIKE '%' || sqlc.arg('query') || '%'
      OR p.description ILIKE '%' || sqlc.arg('query') || '%'
  )

ORDER BY
    CASE
        WHEN sqlc.arg('sort')::text = 'price_asc'
        THEN p.price - (p.price * p.discount / 100)
    END ASC NULLS LAST,

    CASE
        WHEN sqlc.arg('sort')::text = 'price_desc'
        THEN p.price - (p.price * p.discount / 100)
    END DESC NULLS LAST,

    CASE
        WHEN sqlc.arg('sort')::text = 'discount'
        THEN p.discount
    END DESC NULLS LAST,

    CASE
        WHEN sqlc.arg('sort')::text = 'best_selling'
        THEN COALESCE(
            (
                SELECT SUM(oi.quantity)
                FROM order_items oi
                JOIN orders o
                    ON o.id = oi.order_id
                WHERE oi.product_id = p.id
                  AND o.status IN ('paid', 'shipped', 'delivered')
            ),
            0
        )
    END DESC NULLS LAST,

    p.created_at DESC,
    p.id DESC

LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: GetProductsByCategoryName :many
SELECT
    p.id,
    p.name,
    p.description,
    p.price,
    p.discount,
    p.category_id,
    p.is_active,
    p.created_at,
    p.updated_at,
    COALESCE(
        (
            SELECT pi.url
            FROM product_images pi
            WHERE pi.product_id = p.id
              AND pi.is_primary = TRUE
            ORDER BY pi.created_at ASC
            LIMIT 1
        ),
        ''
    ) AS primary_image,
    (
        p.price - (p.price * p.discount / 100)
    )::bigint AS final_price,
    COALESCE(
        (
            SELECT SUM(oi.quantity)
            FROM order_items oi
            JOIN orders o ON o.id = oi.order_id
            WHERE oi.product_id = p.id
              AND o.status IN ('paid', 'shipped', 'delivered')
        ),
        0
    )::bigint AS sold_count
FROM products p
JOIN categories c ON c.id = p.category_id
WHERE p.is_active = TRUE
  AND c.name = sqlc.arg('category_name')::text
ORDER BY
    CASE
        WHEN sqlc.arg('sort')::text = 'price_asc'
        THEN p.price - (p.price * p.discount / 100)
    END ASC NULLS LAST,
    CASE
        WHEN sqlc.arg('sort')::text = 'price_desc'
        THEN p.price - (p.price * p.discount / 100)
    END DESC NULLS LAST,
    CASE
        WHEN sqlc.arg('sort')::text = 'discount'
        THEN p.discount
    END DESC NULLS LAST,
    CASE
        WHEN sqlc.arg('sort')::text = 'best_selling'
        THEN COALESCE(
            (
                SELECT SUM(oi.quantity)
                FROM order_items oi
                JOIN orders o ON o.id = oi.order_id
                WHERE oi.product_id = p.id
                  AND o.status IN ('paid', 'shipped', 'delivered')
            ),
            0
        )
    END DESC NULLS LAST,
    p.created_at DESC,
    p.id DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: UpdateProduct :one
UPDATE products
SET name = $2, description = $3, price = $4, discount = $5, category_id = $6, is_active = $7, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = $1;
