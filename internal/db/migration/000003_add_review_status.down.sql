DROP INDEX IF EXISTS idx_reviews_product_status;
DROP INDEX IF EXISTS idx_reviews_status;

ALTER TABLE reviews
DROP CONSTRAINT IF EXISTS reviews_status_check;

ALTER TABLE reviews
DROP COLUMN IF EXISTS status;