-- =========================================
-- UP Migration: user_quran_records (+ images) & user_attendance
-- dengan user/teacher notes
-- =========================================
BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================
-- A) USER QURAN RECORDS (parent)
-- =========================================
CREATE TABLE IF NOT EXISTS user_quran_records (
  user_quran_records_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_quran_records_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_quran_records_user_id   UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- relasi opsional ke sesi absensi
  user_quran_records_session_id UUID
    REFERENCES class_attendance_sessions(class_attendance_sessions_id)
    ON DELETE SET NULL,

  -- metadata/penugasan
  user_quran_records_teacher_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,

  -- pencatatan & status
  user_quran_records_source_kind VARCHAR(24),
  user_quran_records_status      VARCHAR(24),
  user_quran_records_next        VARCHAR(24),

  -- bidang yang bisa di-search
  user_quran_records_scope TEXT,

  -- catatan umum (legacy)
  user_quran_records_note  TEXT,

  -- ✅ catatan terpisah (baru)
  user_quran_records_user_note    TEXT,
  user_quran_records_teacher_note TEXT,

  user_quran_records_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quran_records_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quran_records_deleted_at TIMESTAMPTZ
);

-- Jika tabel sudah ada, tambah kolom notes baru secara aman
ALTER TABLE user_quran_records
  ADD COLUMN IF NOT EXISTS user_quran_records_user_note    TEXT,
  ADD COLUMN IF NOT EXISTS user_quran_records_teacher_note TEXT;

-- Touch updated_at trigger
CREATE OR REPLACE FUNCTION trg_set_ts_user_quran_records()
RETURNS TRIGGER AS $$
BEGIN
  NEW.user_quran_records_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='set_ts_user_quran_records') THEN
    DROP TRIGGER set_ts_user_quran_records ON user_quran_records;
  END IF;

  CREATE TRIGGER set_ts_user_quran_records
  BEFORE UPDATE ON user_quran_records
  FOR EACH ROW
  EXECUTE FUNCTION trg_set_ts_user_quran_records();
END$$;

-- Indexes (nama selaras dengan DOWN)
CREATE INDEX IF NOT EXISTS idx_user_quran_records_session
  ON user_quran_records(user_quran_records_session_id);

CREATE INDEX IF NOT EXISTS idx_uqr_masjid_created_at
  ON user_quran_records(user_quran_records_masjid_id, user_quran_records_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_uqr_user_created_at
  ON user_quran_records(user_quran_records_user_id, user_quran_records_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_uqr_source_kind
  ON user_quran_records(user_quran_records_source_kind);

CREATE INDEX IF NOT EXISTS idx_uqr_status_next
  ON user_quran_records(user_quran_records_status, user_quran_records_next);

CREATE INDEX IF NOT EXISTS idx_uqr_teacher
  ON user_quran_records(user_quran_records_teacher_user_id);

-- GIN trigram pada scope (data hidup saja)
CREATE INDEX IF NOT EXISTS gin_uqr_scope_trgm
  ON user_quran_records USING gin (user_quran_records_scope gin_trgm_ops)
  WHERE user_quran_records_deleted_at IS NULL;

-- BRIN pada created_at
CREATE INDEX IF NOT EXISTS brin_uqr_created_at
  ON user_quran_records USING brin (user_quran_records_created_at);

-- (Opsional) dedup unik — aktifkan bila perlu
-- CREATE UNIQUE INDEX IF NOT EXISTS uidx_uqr_dedup
--   ON user_quran_records(
--     user_quran_records_masjid_id,
--     user_quran_records_user_id,
--     COALESCE(user_quran_records_session_id, '00000000-0000-0000-0000-000000000000'::uuid),
--     lower(btrim(COALESCE(user_quran_records_scope,''))),
--     date(user_quran_records_created_at)
--   )
--   WHERE user_quran_records_deleted_at IS NULL;


-- =========================================
-- B) USER QURAN RECORD IMAGES (child)
-- =========================================
CREATE TABLE IF NOT EXISTS user_quran_record_images (
  user_quran_record_images_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_quran_record_images_record_id UUID NOT NULL
    REFERENCES user_quran_records(user_quran_records_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  user_quran_record_images_label VARCHAR(120),
  user_quran_record_images_href  TEXT NOT NULL,
  user_quran_record_images_uploader_role VARCHAR(24),

  user_quran_record_images_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quran_record_images_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quran_record_images_deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_user_quran_record_images_record
  ON user_quran_record_images(user_quran_record_images_record_id);

CREATE INDEX IF NOT EXISTS idx_uqri_created_at
  ON user_quran_record_images(user_quran_record_images_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_uqri_uploader_role
  ON user_quran_record_images(user_quran_record_images_uploader_role);


-- =========================================
-- C) USER ATTENDANCE (per siswa per sesi)
-- =========================================
CREATE TABLE IF NOT EXISTS user_attendance (
  user_attendance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_attendance_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_attendance_session_id UUID NOT NULL
    REFERENCES class_attendance_sessions(class_attendance_sessions_id)
    ON UPDATE CASCADE ON DELETE CASCADE,
  user_attendance_user_id    UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  -- status kehadiran: bebas, atau batasi pakai CHECK
  user_attendance_status VARCHAR(16) NOT NULL DEFAULT 'present'
    CHECK (user_attendance_status IN ('present','absent','excused','late')),

  -- catatan umum (legacy)
  user_attendance_note TEXT,

  -- ✅ catatan terpisah (baru)
  user_attendance_user_note    TEXT,
  user_attendance_teacher_note TEXT,

  -- siapa yang menandai
  user_attendance_marked_by_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,
  user_attendance_marked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  user_attendance_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_deleted_at TIMESTAMPTZ
);

-- Jika tabel already exists, tambahkan kolom notes baru secara aman
ALTER TABLE user_attendance
  ADD COLUMN IF NOT EXISTS user_attendance_user_note    TEXT,
  ADD COLUMN IF NOT EXISTS user_attendance_teacher_note TEXT;

-- Tenant guard: masjid attendance harus sama dengan masjid session
CREATE OR REPLACE FUNCTION fn_user_attendance_tenant_guard()
RETURNS TRIGGER AS $$
DECLARE v_mid UUID;
BEGIN
  SELECT class_attendance_sessions_masjid_id
    INTO v_mid
  FROM class_attendance_sessions
  WHERE class_attendance_sessions_id = NEW.user_attendance_session_id
    AND class_attendance_sessions_deleted_at IS NULL;

  IF v_mid IS NULL THEN
    RAISE EXCEPTION 'Session tidak valid/terhapus';
  END IF;

  IF NEW.user_attendance_masjid_id IS DISTINCT FROM v_mid THEN
    RAISE EXCEPTION 'Masjid mismatch: attendance(%) vs session(%)',
      NEW.user_attendance_masjid_id, v_mid;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_user_attendance_tenant_guard') THEN
    DROP TRIGGER trg_user_attendance_tenant_guard ON user_attendance;
  END IF;

  CREATE TRIGGER trg_user_attendance_tenant_guard
  BEFORE INSERT OR UPDATE ON user_attendance
  FOR EACH ROW EXECUTE FUNCTION fn_user_attendance_tenant_guard();
END$$;

-- Unique: 1 baris aktif per (masjid, session, user)
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_attendance_alive
  ON user_attendance(user_attendance_masjid_id, user_attendance_session_id, user_attendance_user_id)
  WHERE user_attendance_deleted_at IS NULL;

-- Indexes bantu
CREATE INDEX IF NOT EXISTS idx_user_attendance_session
  ON user_attendance(user_attendance_session_id);

CREATE INDEX IF NOT EXISTS idx_user_attendance_user
  ON user_attendance(user_attendance_user_id);

CREATE INDEX IF NOT EXISTS idx_user_attendance_status
  ON user_attendance(user_attendance_status);

CREATE INDEX IF NOT EXISTS brin_user_attendance_created_at
  ON user_attendance USING brin (user_attendance_created_at);

-- Touch updated_at
CREATE OR REPLACE FUNCTION fn_touch_user_attendance_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.user_attendance_updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_user_attendance_updated_at') THEN
    DROP TRIGGER trg_touch_user_attendance_updated_at ON user_attendance;
  END IF;

  CREATE TRIGGER trg_touch_user_attendance_updated_at
  BEFORE UPDATE ON user_attendance
  FOR EACH ROW EXECUTE FUNCTION fn_touch_user_attendance_updated_at();
END$$;

COMMIT;
