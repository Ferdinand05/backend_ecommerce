CREATE TABLE order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    product_variant_id UUID NOT NULL,

    product_name VARCHAR(150) NOT NULL,
    variant_name VARCHAR(150) NOT NULL,
    sku VARCHAR(100) NOT NULL,

    price NUMERIC(19,4) NOT NULL,
    quantity INT NOT NULL,
    subtotal NUMERIC(19,4) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_order_items_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_order_items_product_variant
        FOREIGN KEY (product_variant_id)
        REFERENCES product_variants(id)
        ON DELETE RESTRICT,

    CONSTRAINT chk_order_items_price
        CHECK (price >= 0),

    CONSTRAINT chk_order_items_quantity
        CHECK (quantity > 0),

    CONSTRAINT chk_order_items_subtotal
        CHECK (subtotal >= 0)
);

CREATE INDEX idx_order_items_order_id
    ON order_items(order_id);

CREATE INDEX idx_order_items_product_variant_id
    ON order_items(product_variant_id);