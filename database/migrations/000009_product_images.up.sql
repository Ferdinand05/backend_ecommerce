CREATE TABLE product_images (
    id UUID PRIMARY KEY,

    product_id UUID NOT NULL,
    product_variant_id UUID,

    storage_key TEXT NOT NULL,
    alt VARCHAR(255),
    sort_order INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_product_images_product
        FOREIGN KEY (product_id)
        REFERENCES products(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_product_images_variant
        FOREIGN KEY (product_variant_id)
        REFERENCES product_variants(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_product_images_product_id
    ON product_images(product_id);

CREATE INDEX idx_product_images_variant_id
    ON product_images(product_variant_id);

CREATE INDEX idx_product_images_sort_order
    ON product_images(product_id, product_variant_id, sort_order);