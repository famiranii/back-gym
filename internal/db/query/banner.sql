-- name: GetActiveBanners :many
SELECT * FROM banners
WHERE is_active = true
ORDER BY sort_order ASC;

-- name: CreateBanner :one
INSERT INTO banners (title, subtitle, button_text, button_url, image_url, is_active, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateBanner :one
UPDATE banners
SET title = $2, subtitle = $3, button_text = $4, button_url = $5, image_url = $6, is_active = $7, sort_order = $8, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteBanner :exec
DELETE FROM banners
WHERE id = $1;