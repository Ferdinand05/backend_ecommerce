CREATE TABLE password_resets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,

    token_hash TEXT NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_password_resets_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_password_resets_user_id
    ON password_resets(user_id);

CREATE INDEX idx_password_resets_expires_at
    ON password_resets(expires_at);