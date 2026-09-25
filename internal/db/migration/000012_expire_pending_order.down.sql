-- 000XXX_add_expires_at_to_payments.down.sql

DROP INDEX IF EXISTS idx_payments_expires_at;

ALTER TABLE payments
DROP COLUMN IF EXISTS expires_at;