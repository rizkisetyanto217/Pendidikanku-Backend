-- =========================
-- DOWN: refresh_tokens
-- =========================

-- 1) Indexes
DROP INDEX IF EXISTS idx_rt_expires_active;
DROP INDEX IF EXISTS idx_rt_user_active;
DROP INDEX IF EXISTS idx_rt_token_active;

-- 2) Trigger
DROP TRIGGER IF EXISTS trg_refresh_tokens_updated_at ON refresh_tokens;

-- 3) Function
DROP FUNCTION IF EXISTS set_refresh_tokens_updated_at();

-- 4) Table
DROP TABLE IF EXISTS refresh_tokens;
