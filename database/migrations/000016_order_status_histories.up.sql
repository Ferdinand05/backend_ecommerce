CREATE TABLE order_status_histories (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,

    status VARCHAR(30) NOT NULL,
    note TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_order_status_histories_order
        FOREIGN KEY (order_id)
        REFERENCES orders(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_order_status_histories_order_id
    ON order_status_histories(order_id);

CREATE INDEX idx_order_status_histories_order_created_at
    ON order_status_histories(order_id, created_at);