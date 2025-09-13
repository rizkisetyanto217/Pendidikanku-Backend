-- =========================================
-- DOWN Migration — Certificates
-- =========================================
BEGIN;

-- ===============================
-- DROP INDEXES: certificate_urls
-- ===============================
DROP INDEX IF EXISTS uq_certificate_urls_cert_href_alive;
DROP INDEX IF EXISTS idx_certificate_urls_publish_flags;
DROP INDEX IF EXISTS brin_certificate_urls_created_at;

-- ===========================
-- DROP INDEXES: certificates
-- ===========================
DROP INDEX IF EXISTS uq_certificates_summary_alive;
DROP INDEX IF EXISTS uq_certificates_masjid_serial_alive;
DROP INDEX IF EXISTS idx_certificates_masjid_created_at;
DROP INDEX IF EXISTS idx_certificates_student_alive;
DROP INDEX IF EXISTS idx_certificates_status_alive;
DROP INDEX IF EXISTS brin_certificates_created_at;

-- ==================================
-- DROP INDEXES: certificate_templates
-- ==================================
DROP INDEX IF EXISTS uq_certificate_templates_masjid_key_alive;
DROP INDEX IF EXISTS idx_certificate_templates_active;

-- =========================
-- DROP TABLES (child first)
-- =========================
DROP TABLE IF EXISTS certificate_urls;
DROP TABLE IF EXISTS certificates;
DROP TABLE IF EXISTS certificate_templates;

-- (Opsional) Kalau ingin bersih total:
-- DROP EXTENSION IF EXISTS pgcrypto;

COMMIT;
