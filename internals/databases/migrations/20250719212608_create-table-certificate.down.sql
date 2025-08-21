
BEGIN;
-- =====================================================================
-- ==============================  DOWN  ================================
-- =====================================================================

-- Triggers
DROP TRIGGER IF EXISTS trg_user_certificates_touch ON user_certificates;
DROP TRIGGER IF EXISTS trg_certificates_touch ON certificates;

-- Indexes (user_certificates)
DROP INDEX IF EXISTS idx_user_cert_rank;
DROP INDEX IF EXISTS idx_user_cert_active_by_cert;
DROP INDEX IF EXISTS idx_user_cert_active_by_user;
DROP INDEX IF EXISTS idx_user_cert_cert_issued;
DROP INDEX IF EXISTS idx_user_cert_user_created;

-- Indexes (certificates)
DROP INDEX IF EXISTS uq_certificates_lecture_title;
DROP INDEX IF EXISTS idx_certificates_desc_trgm;
DROP INDEX IF EXISTS idx_certificates_title_trgm;
DROP INDEX IF EXISTS idx_certificates_lecture_created;

-- Tables
DROP TABLE IF EXISTS user_certificates;
DROP TABLE IF EXISTS certificates;

-- Helper
-- (biarkan fn_touch_updated_at_generic untuk dipakai tabel lain; hapus jika perlu)
-- DROP FUNCTION IF EXISTS fn_touch_updated_at_generic;

COMMIT;