
CREATE TABLE payments (
    id UUID PRIMARY KEY,

    order_id UUID NOT NULL,

    provider VARCHAR(30) NOT NULL,

    external_id VARCHAR(100) NOT NULL,

    provider_invoice_id VARCHAR(100),

    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',

    amount NUMERIC(19,4) NOT NULL,

    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',

    checkout_url TEXT,

    expires_at TIMESTAMPTZ,

    paid_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_payments_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE RESTRICT,

    CONSTRAINT uq_payments_external_id
        UNIQUE (external_id),

    CONSTRAINT uq_payments_provider_invoice_id
        UNIQUE (provider_invoice_id),

    CONSTRAINT chk_payments_provider
        CHECK (provider IN ('XENDIT')),

    CONSTRAINT chk_payments_status
        CHECK (status IN ('PENDING', 'PAID', 'EXPIRED', 'FAILED')),

    CONSTRAINT chk_payments_amount
        CHECK (amount >= 0),

    CONSTRAINT chk_payments_currency
        CHECK (currency = 'IDR')
);

CREATE INDEX idx_payments_order_id
    ON payments(order_id);

CREATE INDEX idx_payments_status
    ON payments(status);

-- Satu order hanya boleh punya satu payment PENDING pada satu waktu;
-- banyak attempt sepanjang waktu tetap diizinkan.
CREATE UNIQUE INDEX uq_payments_one_pending_per_order
    ON payments(order_id) WHERE status = 'PENDING';
