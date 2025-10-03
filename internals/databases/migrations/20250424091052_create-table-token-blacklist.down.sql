DROP INDEX IF EXISTS idx_token_blacklist_hash_alive;
DROP INDEX IF EXISTS idx_token_blacklist_cleanup;
DROP TABLE IF EXISTS token_blacklist;
-- Extension pgcrypto dibiarkan (dipakai tabel lain juga)
