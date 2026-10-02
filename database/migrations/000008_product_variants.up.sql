CREATE TABLE product_variants (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL,

    sku VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,

    price NUMERIC(19,4) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_product_variants_product
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE CASCADE,

    CONSTRAINT chk_product_variants_price
        CHECK (price >= 0)
);