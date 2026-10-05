CREATE TABLE stock_movements (
    id UUID PRIMARY KEY,
    inventory_item_id UUID NOT NULL,

    type VARCHAR(30) NOT NULL,
    quantity INT NOT NULL,
    note TEXT,

    reference_type VARCHAR(50),
    reference_id UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_stock_movements_inventory_item
        FOREIGN KEY (inventory_item_id)
        REFERENCES inventory_items(id)
        ON DELETE CASCADE,

    CONSTRAINT chk_stock_movements_quantity
        CHECK (quantity <> 0)
);