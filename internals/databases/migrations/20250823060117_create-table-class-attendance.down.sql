-- =========================================================
-- DOWN MIGRATION
-- =========================================================

-- 1) Kembalikan kolom lama di class_attendance_sessions (jika belum ada)
ALTER TABLE class_attendance_sessions
  ADD COLUMN IF NOT EXISTS class_attendance_sessions_image_url TEXT,
  ADD COLUMN IF NOT EXISTS class_attendance_sessions_image_trash_url TEXT,
  ADD COLUMN IF NOT EXISTS class_attendance_sessions_image_delete_pending_until TIMESTAMPTZ;

-- 2) BACKFILL dari class_attendance_session_url -> kolom lama (1 URL per session)
--    Prioritas: label mengandung 'cover' (case-insensitive), lalu yang paling awal dibuat.
WITH ranked AS (
  SELECT
    u.class_attendance_session_url_session_id AS sid,
    u.class_attendance_session_url_href       AS href,
    u.class_attendance_session_url_trash_url  AS trash,
    u.class_attendance_session_url_delete_pending_until AS due,
    ROW_NUMBER() OVER (
      PARTITION BY u.class_attendance_session_url_session_id
      ORDER BY
        CASE WHEN COALESCE(u.class_attendance_session_url_label,'') ILIKE '%cover%' THEN 0 ELSE 1 END,
        u.class_attendance_session_url_created_at ASC,
        u.class_attendance_session_url_id ASC
    ) AS rn
  FROM class_attendance_session_url u
  WHERE u.class_attendance_session_url_deleted_at IS NULL
)
UPDATE class_attendance_sessions s
SET
  class_attendance_sessions_image_url = r.href,
  class_attendance_sessions_image_trash_url = r.trash,
  class_attendance_sessions_image_delete_pending_until = r.due
FROM ranked r
WHERE r.sid = s.class_attendance_sessions_id
  AND r.rn = 1
  AND (s.class_attendance_sessions_image_url IS NULL OR btrim(s.class_attendance_sessions_image_url) = '');

-- 3) Hapus TRIGGER/FUNCTION di class_attendance_session_url
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_casu_tenant_guard') THEN
    EXECUTE 'DROP TRIGGER trg_casu_tenant_guard ON class_attendance_session_url';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_casu_tenant_guard') THEN
    EXECUTE 'DROP FUNCTION fn_casu_tenant_guard()';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_casu_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_touch_casu_updated_at ON class_attendance_session_url';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname='fn_touch_casu_updated_at') THEN
    EXECUTE 'DROP FUNCTION fn_touch_casu_updated_at()';
  END IF;
END$$;

-- 4) Hapus INDEX di class_attendance_session_url
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='uq_casu_href_per_session_alive') THEN
    EXECUTE 'DROP INDEX uq_casu_href_per_session_alive';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_casu_session_alive') THEN
    EXECUTE 'DROP INDEX idx_casu_session_alive';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_casu_created_at') THEN
    EXECUTE 'DROP INDEX idx_casu_created_at';
  END IF;
END$$;

-- 5) DROP TABLE class_attendance_session_url
DROP TABLE IF EXISTS class_attendance_session_url;

-- 6) Hapus TRIGGER/FUNCTION yang ditambahkan di class_attendance_sessions (validasi & touch)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cas_validate_links') THEN
    EXECUTE 'DROP TRIGGER trg_cas_validate_links ON class_attendance_sessions';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'fn_cas_validate_links') THEN
    EXECUTE 'DROP FUNCTION fn_cas_validate_links()';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_cas_touch_updated_at') THEN
    EXECUTE 'DROP TRIGGER trg_cas_touch_updated_at ON class_attendance_sessions';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'fn_touch_class_attendance_sessions_updated_at') THEN
    EXECUTE 'DROP FUNCTION fn_touch_class_attendance_sessions_updated_at()';
  END IF;
END$$;

-- 7) (OPSIONAL) Hapus index yang dibuat oleh UP pada class_attendance_sessions
--    Uncomment bila ingin benar-benar mengembalikan ke skema tanpa constraint unik tsb.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='uq_cas_section_date_when_cs_null') THEN
    EXECUTE 'DROP INDEX uq_cas_section_date_when_cs_null';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='uq_cas_section_cs_date_when_cs_not_null') THEN
    EXECUTE 'DROP INDEX uq_cas_section_cs_date_when_cs_not_null';
  END IF;

  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_cas_section') THEN
    EXECUTE 'DROP INDEX idx_cas_section';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_cas_masjid') THEN
    EXECUTE 'DROP INDEX idx_cas_masjid';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_cas_date') THEN
    EXECUTE 'DROP INDEX idx_cas_date';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_cas_class_subject') THEN
    EXECUTE 'DROP INDEX idx_cas_class_subject';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_cas_csst') THEN
    EXECUTE 'DROP INDEX idx_cas_csst';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname='idx_cas_teacher_user') THEN
    EXECUTE 'DROP INDEX idx_cas_teacher_user';
  END IF;
END$$;

-- 8) (OPSIONAL) Hapus FK yang ditambahkan UP
--    WARNING: menghapus FK bisa membuka risiko inkonsistensi. Lanjutkan hanya jika memang perlu.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_section_masjid_pair') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_section_masjid_pair;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_subject') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_class_subject;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cas_class_section_subject_teacher') THEN
    ALTER TABLE class_attendance_sessions DROP CONSTRAINT fk_cas_class_section_subject_teacher;
  END IF;
END$$;

-- Selesai.
