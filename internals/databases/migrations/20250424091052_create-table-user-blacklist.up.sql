-- ============================ --
-- TABLE TOKEN BLACKLIST --
-- ============================ --

CREATE TABLE IF NOT EXISTS token_blacklist (
  id SERIAL PRIMARY KEY,
  token TEXT NOT NULL UNIQUE,
  expired_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ NULL,
  -- opsional: pastikan expired_at tidak lebih kecil dari created_at
  CONSTRAINT token_blacklist_time_chk CHECK (expired_at >= created_at)
);

-- Index untuk cleanup (hanya baris yang belum soft-delete)
CREATE INDEX IF NOT EXISTS idx_token_blacklist_cleanup
  ON token_blacklist (expired_at, created_at)
  WHERE deleted_at IS NULL;

-- Index lookup cepat saat verifikasi (hanya baris yang belum soft-delete)
CREATE INDEX IF NOT EXISTS idx_token_blacklist_token_not_deleted
  ON token_blacklist (token)
  WHERE deleted_at IS NULL;
