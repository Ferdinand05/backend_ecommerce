ALTER TABLE product_variants ADD COLUMN weight_grams INT NOT NULL DEFAULT 0;
ALTER TABLE order_items ADD COLUMN weight_grams INT NOT NULL DEFAULT 0;
