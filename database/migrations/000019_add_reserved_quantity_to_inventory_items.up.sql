ALTER TABLE inventory_items ADD COLUMN reserved_quantity INT NOT NULL DEFAULT 0;
ALTER TABLE inventory_items ADD CONSTRAINT chk_inventory_items_reserved_quantity CHECK (reserved_quantity >= 0);
