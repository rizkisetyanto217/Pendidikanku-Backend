-- butuh pgcrypto untuk gen_random_uuid() & digest()/hmac()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- simpan HASH token (lebih aman & lebih kecil di index)
    token_hash   BYTEA NOT NULL UNIQUE,

    -- status & masa berlaku
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,

    -- metadata opsional
    user_agent   TEXT,
    ip           INET,

    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- expired tidak boleh sebelum dibuat
    CONSTRAINT ck_rt_expiry_future CHECK (expires_at > created_at)
);

-- Trigger auto-UPDATE updated_at
CREATE OR REPLACE FUNCTION set_refresh_tokens_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_refresh_tokens_updated_at ON refresh_tokens;
CREATE TRIGGER trg_refresh_tokens_updated_at
BEFORE UPDATE ON refresh_tokens
FOR EACH ROW EXECUTE FUNCTION set_refresh_tokens_updated_at();

-- INDEXING (disetel untuk pola query umum)

-- 1) Verifikasi token: WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
--    (token_hash, expires_at) + partial revoked_at IS NULL
CREATE INDEX IF NOT EXISTS idx_rt_token_active
  ON refresh_tokens (token_hash, expires_at)
  WHERE revoked_at IS NULL;

-- 2) Listing token aktif per user: WHERE user_id = $1 AND revoked_at IS NULL
--    plus sort by terbaru/akan-expire
CREATE INDEX IF NOT EXISTS idx_rt_user_active
  ON refresh_tokens (user_id, expires_at DESC)
  WHERE revoked_at IS NULL;

-- 3) Pembersihan terjadwal: WHERE revoked_at IS NOT NULL OR expires_at <= NOW()
--    (tidak boleh pakai NOW() di predicate index; fokus ke expires_at)
CREATE INDEX IF NOT EXISTS idx_rt_expires_active
  ON refresh_tokens (expires_at)
  WHERE revoked_at IS NULL;

-- Optional (sangat besar/append-only): akselerasi sweep berdasarkan waktu
-- CREATE INDEX IF NOT EXISTS brin_rt_created_at ON refresh_tokens USING BRIN (created_at);
