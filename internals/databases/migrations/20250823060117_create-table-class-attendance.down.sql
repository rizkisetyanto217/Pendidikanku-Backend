BEGIN;

-- =========================
-- Bersihkan: URL indexes
-- =========================
DROP INDEX IF EXISTS uq_casu_href_per_session_alive;
DROP INDEX IF EXISTS idx_casu_session_alive;
DROP INDEX IF EXISTS idx_casu_masjid_created_alive;
DROP INDEX IF EXISTS idx_casu_created_at;

-- (jika ada trigger/function lama yang kebetulan hidup)
DROP TRIGGER IF EXISTS trg_casu_tenant_guard ON class_attendance_session_url;
DROP FUNCTION IF EXISTS fn_casu_tenant_guard();

-- =========================
-- Drop tabel URL
-- =========================
DROP TABLE IF EXISTS class_attendance_session_url CASCADE;


-- =========================
-- Bersihkan: CAS indexes
-- =========================
DROP INDEX IF EXISTS idx_cas_masjid_teacher_date_alive;
DROP INDEX IF EXISTS idx_cas_masjid_subject_date_alive;
DROP INDEX IF EXISTS idx_cas_masjid_section_date_alive;
DROP INDEX IF EXISTS uq_cas_masjid_section_subject_date;

DROP INDEX IF EXISTS idx_cas_teacher_id;
DROP INDEX IF EXISTS idx_cas_class_room;
DROP INDEX IF EXISTS idx_cas_class_subject;
DROP INDEX IF EXISTS idx_cas_date;
DROP INDEX IF EXISTS idx_cas_masjid;
DROP INDEX IF EXISTS idx_cas_section;

-- (jika ada trigger/function lama yang kebetulan hidup)
DROP TRIGGER IF EXISTS trg_cas_validate_links ON class_attendance_sessions;
DROP FUNCTION IF EXISTS fn_cas_validate_links();

-- =========================
-- Drop tabel CAS
-- =========================
DROP TABLE IF EXISTS class_attendance_sessions CASCADE;

COMMIT;
