ALTER TABLE reviews
ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'pending';

ALTER TABLE reviews
ADD CONSTRAINT reviews_status_check
CHECK (status IN ('pending', 'approved', 'rejected'));

CREATE INDEX idx_reviews_status
ON reviews(status);

CREATE INDEX idx_reviews_product_status
ON reviews(product_id, status);