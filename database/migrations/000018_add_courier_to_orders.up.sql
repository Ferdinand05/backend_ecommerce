ALTER TABLE orders ADD COLUMN courier_code VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN courier_service_code VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN courier_name VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN courier_service_name VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE orders ADD COLUMN payment_expires_at TIMESTAMPTZ;
CREATE INDEX idx_orders_payment_expires_at ON orders(payment_expires_at);
