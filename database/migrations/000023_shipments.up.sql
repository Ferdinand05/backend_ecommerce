-- migrations/202610220001_create_shipments.up.sql

CREATE TABLE shipments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Satu order maksimal memiliki satu shipment.
    order_id UUID NOT NULL UNIQUE,

    -- Provider pengiriman, misalnya BITESHIP.
    provider VARCHAR(30) NOT NULL DEFAULT 'BITESHIP',

    -- ID order dari provider pengiriman.
    provider_order_id VARCHAR(100) UNIQUE,

    -- Informasi kurir dan resi.
    courier_company VARCHAR(100),
    courier_type VARCHAR(100),
    tracking_number VARCHAR(100),
    tracking_url TEXT,

    -- Status internal aplikasi dan status dari provider.
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    provider_status VARCHAR(100),

    shipping_cost NUMERIC(15, 2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',

    shipped_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_shipments_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE RESTRICT,

    CONSTRAINT chk_shipments_shipping_cost
        CHECK (shipping_cost >= 0)
);

CREATE INDEX idx_shipments_status
    ON shipments(status);

CREATE INDEX idx_shipments_tracking_number
    ON shipments(tracking_number);

CREATE INDEX idx_shipments_created_at
    ON shipments(created_at DESC);