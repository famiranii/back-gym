-- 000XXX_add_expires_at_to_payments.up.sql

ALTER TABLE payments
ADD COLUMN expires_at TIMESTAMPTZ;

UPDATE payments
SET expires_at = created_at + INTERVAL '15 minutes'
WHERE expires_at IS NULL;

ALTER TABLE payments
ALTER COLUMN expires_at SET NOT NULL;

CREATE INDEX idx_payments_expires_at
ON payments (expires_at);