-- +migrate Down
-- =========================================
-- DOWN Migration — User Class Session Attendances (renamed, explicit)
-- =========================================
BEGIN;

-- =========================================
-- C) USER_CLASS_SESSION_ATTENDANCE_URLS — DROP INDEXES & TABLE
-- =========================================

-- Indexes
DROP INDEX IF EXISTS ix_ucsaurl_by_owner_live;
DROP INDEX IF EXISTS ix_ucsaurl_by_masjid_live;
DROP INDEX IF EXISTS uq_ucsaurl_primary_per_kind_alive;
DROP INDEX IF EXISTS uq_ucsaurl_attendance_href_alive;
DROP INDEX IF EXISTS ix_ucsaurl_purge_due;
DROP INDEX IF EXISTS ix_ucsaurl_uploader_teacher_live;
DROP INDEX IF EXISTS ix_ucsaurl_uploader_student_live;
DROP INDEX IF EXISTS brin_ucsaurl_created_at;
DROP INDEX IF EXISTS gin_ucsaurl_label_trgm_live;

-- Table
DROP TABLE IF EXISTS user_class_session_attendance_urls CASCADE;


-- =========================================
-- B) USER_CLASS_SESSION_ATTENDANCES — DROP INDEXES & TABLE
-- =========================================

-- Indexes
DROP INDEX IF EXISTS uq_ucsa_alive;
DROP INDEX IF EXISTS idx_ucsa_session;
DROP INDEX IF EXISTS idx_ucsa_student;
DROP INDEX IF EXISTS idx_ucsa_status;
DROP INDEX IF EXISTS idx_ucsa_type_id;
DROP INDEX IF EXISTS idx_ucsa_session_status;
DROP INDEX IF EXISTS brin_ucsa_created_at;
DROP INDEX IF EXISTS brin_ucsa_marked_at;
DROP INDEX IF EXISTS gin_ucsa_desc_trgm;

-- Table
DROP TABLE IF EXISTS user_class_session_attendances CASCADE;


-- =========================================
-- A) USER_CLASS_SESSION_ATTENDANCE_TYPES — DROP INDEXES & TABLE
-- =========================================

-- Indexes
DROP INDEX IF EXISTS uq_ucsat_code_per_masjid_alive;
DROP INDEX IF EXISTS gin_ucsat_label_trgm;
DROP INDEX IF EXISTS idx_ucsat_masjid_active;
DROP INDEX IF EXISTS idx_ucsat_masjid_created_desc;
DROP INDEX IF EXISTS brin_ucsat_created_at;

-- Table
DROP TABLE IF EXISTS user_class_session_attendance_types CASCADE;

COMMIT;
