CREATE TABLE order_addresses (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL UNIQUE,

    recipient_name VARCHAR(150) NOT NULL,
    phone VARCHAR(30) NOT NULL,

    address TEXT NOT NULL,
    city VARCHAR(100) NOT NULL,
    province VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,

    latitude NUMERIC(10,7),
    longitude NUMERIC(10,7),

    biteship_area_id VARCHAR(100),
    biteship_area_name VARCHAR(255),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_order_addresses_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_order_addresses_biteship_area_id
    ON order_addresses(biteship_area_id);