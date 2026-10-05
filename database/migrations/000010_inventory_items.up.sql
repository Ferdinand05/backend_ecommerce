CREATE TABLE inventory_items (
    id UUID PRIMARY KEY,
    product_variant_id UUID NOT NULL UNIQUE,
    quantity INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_inventory_items_product_variant
        FOREIGN KEY (product_variant_id)
        REFERENCES product_variants(id)
        ON DELETE CASCADE,

    CONSTRAINT chk_inventory_items_quantity
        CHECK (quantity >= 0)
);