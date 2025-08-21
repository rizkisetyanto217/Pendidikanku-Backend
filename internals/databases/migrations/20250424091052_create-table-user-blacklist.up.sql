-- Extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- digest() / gen_random_uuid()

-- Tabel blacklist (pakai hash token demi keamanan)
CREATE TABLE IF NOT EXISTS token_blacklist (
  token_blacklist_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  token_hash BYTEA NOT NULL UNIQUE,           -- sha256(token) → BYTEA
  expired_at TIMESTAMPTZ NOT NULL,            -- gunakan waktu dengan zona
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ NULL,

  -- token yang di-blacklist minimal punya expired_at >= created_at
  CONSTRAINT token_blacklist_time_chk CHECK (expired_at >= created_at)
);

-- Index untuk cleanup (hanya baris yang belum soft-delete)
CREATE INDEX IF NOT EXISTS idx_token_blacklist_cleanup
  ON token_blacklist (expired_at, created_at)
  WHERE deleted_at IS NULL;

-- Index lookup cepat saat verifikasi (hanya baris yang belum soft-delete)
CREATE INDEX IF NOT EXISTS idx_token_blacklist_hash_alive
  ON token_blacklist (token_hash)
  WHERE deleted_at IS NULL;
