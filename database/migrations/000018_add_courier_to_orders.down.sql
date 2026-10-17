DROP INDEX idx_orders_payment_expires_at;
ALTER TABLE orders DROP COLUMN payment_expires_at;
ALTER TABLE orders DROP COLUMN courier_service_name;
ALTER TABLE orders DROP COLUMN courier_name;
ALTER TABLE orders DROP COLUMN courier_service_code;
ALTER TABLE orders DROP COLUMN courier_code;
