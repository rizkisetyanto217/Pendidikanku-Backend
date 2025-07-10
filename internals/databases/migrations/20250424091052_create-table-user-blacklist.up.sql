CREATE TABLE IF NOT EXISTS token_blacklist (
    id SERIAL PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    expired_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

-- Index gabungan agar query cleanup lebih efisien
CREATE INDEX IF NOT EXISTS idx_token_blacklist_cleanup 
ON token_blacklist (expired_at, created_at);

CREATE INDEX IF NOT EXISTS idx_token_blacklist_token_not_deleted 
ON token_blacklist(token)
WHERE deleted_at IS NULL;