-- =========================================
-- UP Migration (Refactor Final) — NO timestamp triggers
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

  user_quran_records_session_id UUID
    REFERENCES class_attendance_sessions(class_attendance_sessions_id)
    ON DELETE SET NULL,

  -- masih refer ke users (legacy, kalau dipakai)
  user_quran_records_teacher_user_id UUID
    REFERENCES users(id) ON DELETE SET NULL,

  user_quran_records_source_kind VARCHAR(24),

  -- bidang yang bisa di-search
  user_quran_records_scope TEXT,

  -- catatan
  user_quran_records_user_note    TEXT,
  user_quran_records_teacher_note TEXT,

  -- nilai (0..100, boleh desimal)
  user_quran_records_score NUMERIC(5,2)
    CHECK (user_quran_records_score >= 0 AND user_quran_records_score <= 100),

  -- pengganti "next": boolean (nullable)
  user_quran_records_is_next BOOLEAN,

  -- timestamps (dikelola aplikasi/GORM)
  user_quran_records_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quran_records_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quran_records_deleted_at TIMESTAMPTZ
);

-- Tambah kolom (idempotent)
ALTER TABLE user_quran_records
  ADD COLUMN IF NOT EXISTS user_quran_records_is_next BOOLEAN;

ALTER TABLE user_quran_records
  ADD COLUMN IF NOT EXISTS user_quran_records_score NUMERIC(5,2)
    CHECK (user_quran_records_score >= 0 AND user_quran_records_score <= 100);

-- Migrasi next (VARCHAR) -> is_next (BOOLEAN) jika kolom lama ada
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='user_quran_records' AND column_name='user_quran_records_next'
  ) THEN
    UPDATE user_quran_records
    SET user_quran_records_is_next = CASE
      WHEN user_quran_records_next IS NULL THEN NULL
      WHEN length(btrim(user_quran_records_next)) = 0 THEN NULL
      WHEN lower(btrim(user_quran_records_next)) IN ('1','true','yes','y','next','lanjut') THEN TRUE
      WHEN lower(btrim(user_quran_records_next)) IN ('0','false','no','n') THEN FALSE
      ELSE NULL
    END;

    IF EXISTS (SELECT 1 FROM pg_class WHERE relname='idx_uqr_status_next') THEN
      DROP INDEX idx_uqr_status_next;
    END IF;

    ALTER TABLE user_quran_records
      DROP COLUMN IF EXISTS user_quran_records_next;
  END IF;
END$$;

-- Bersihkan trigger timestamp lama kalau ada (biar GORM yang kelola)
DROP TRIGGER IF EXISTS set_ts_user_quran_records ON user_quran_records;
DROP FUNCTION IF EXISTS trg_set_ts_user_quran_records();

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_quran_records_session
  ON user_quran_records(user_quran_records_session_id);

CREATE INDEX IF NOT EXISTS idx_uqr_masjid_created_at
  ON user_quran_records(user_quran_records_masjid_id, user_quran_records_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_uqr_user_created_at
  ON user_quran_records(user_quran_records_user_id, user_quran_records_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_uqr_source_kind
  ON user_quran_records(user_quran_records_source_kind);

CREATE INDEX IF NOT EXISTS idx_uqr_is_next
  ON user_quran_records(user_quran_records_is_next);

CREATE INDEX IF NOT EXISTS idx_uqr_teacher
  ON user_quran_records(user_quran_records_teacher_user_id);

CREATE INDEX IF NOT EXISTS gin_uqr_scope_trgm
  ON user_quran_records USING GIN (user_quran_records_scope gin_trgm_ops)
  WHERE user_quran_records_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_uqr_created_at
  ON user_quran_records USING BRIN (user_quran_records_created_at);


-- =========================================
-- B) USER QURAN URLS (child)
-- =========================================
CREATE TABLE IF NOT EXISTS user_quran_urls (
  user_quran_urls_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_quran_urls_record_id UUID NOT NULL
    REFERENCES user_quran_records(user_quran_records_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  user_quran_urls_label VARCHAR(120),

  -- file utama
  user_quran_urls_href TEXT NOT NULL,

  -- housekeeping (opsional)
  user_quran_urls_trash_url            TEXT,
  user_quran_urls_delete_pending_until TIMESTAMPTZ,

  -- uploader (salah satu boleh NULL)
  user_quran_urls_uploader_teacher_id UUID,
  user_quran_urls_uploader_user_id    UUID
    REFERENCES users(id) ON DELETE SET NULL,

  -- timestamps (dikelola aplikasi/GORM)
  user_quran_urls_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quran_urls_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_quran_urls_deleted_at TIMESTAMPTZ
);

-- Tambahkan FK ke masjid_teachers(masjid_teacher_id) bila tabel & kolomnya ada
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema='public' AND table_name='masjid_teachers'
  ) AND EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='masjid_teachers' AND column_name='masjid_teacher_id'
  ) THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_uqri_uploader_teacher') THEN
      ALTER TABLE user_quran_urls
        ADD CONSTRAINT fk_uqri_uploader_teacher
        FOREIGN KEY (user_quran_urls_uploader_teacher_id)
        REFERENCES masjid_teachers(masjid_teacher_id)
        ON DELETE SET NULL;
    END IF;
  ELSE
    RAISE NOTICE 'masjid_teachers / masjid_teacher_id belum ada; lewati FK user_quran_urls → masjid_teachers.';
  END IF;
END$$;

-- Bersihkan trigger timestamp lama kalau ada (biar GORM yang kelola)
DROP TRIGGER IF EXISTS set_ts_user_quran_urls ON user_quran_urls;
DROP FUNCTION IF EXISTS trg_set_ts_user_quran_urls();

-- Indexes dasar
CREATE INDEX IF NOT EXISTS idx_user_quran_urls_record
  ON user_quran_urls(user_quran_urls_record_id);

CREATE INDEX IF NOT EXISTS idx_uqri_created_at
  ON user_quran_urls(user_quran_urls_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_uqri_uploader_teacher
  ON user_quran_urls(user_quran_urls_uploader_teacher_id);

CREATE INDEX IF NOT EXISTS idx_uqri_uploader_user
  ON user_quran_urls(user_quran_urls_uploader_user_id);

-- Optimizations
CREATE INDEX IF NOT EXISTS idx_uqri_record_alive
  ON user_quran_urls(user_quran_urls_record_id, user_quran_urls_created_at DESC)
  WHERE user_quran_urls_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_uqri_created_at
  ON user_quran_urls USING BRIN (user_quran_urls_created_at);

CREATE UNIQUE INDEX IF NOT EXISTS uq_uqri_record_href
  ON user_quran_urls(user_quran_urls_record_id, user_quran_urls_href)
  WHERE user_quran_urls_deleted_at IS NULL;


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

  -- status kehadiran
  user_attendance_status VARCHAR(16) NOT NULL DEFAULT 'present'
    CHECK (user_attendance_status IN ('present','absent','excused','late')),

  -- catatan
  user_attendance_user_note    TEXT,
  user_attendance_teacher_note TEXT,

  -- timestamps (dikelola aplikasi/GORM)
  user_attendance_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_deleted_at TIMESTAMPTZ
);

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

DROP TRIGGER IF EXISTS trg_user_attendance_tenant_guard ON user_attendance;
CREATE TRIGGER trg_user_attendance_tenant_guard
  BEFORE INSERT OR UPDATE ON user_attendance
  FOR EACH ROW EXECUTE FUNCTION fn_user_attendance_tenant_guard();

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
  ON user_attendance USING BRIN (user_attendance_created_at);



-- =========================================
-- D) USER ATTENDANCE URLS (child)
-- =========================================
CREATE TABLE IF NOT EXISTS user_attendance_urls (
  user_attendance_urls_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant scope langsung di child (cepat buat filter)
  user_attendance_urls_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Relasi ke parent attendance
  user_attendance_urls_attendance_id UUID NOT NULL
    REFERENCES user_attendance(user_attendance_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Metadata
  user_attendance_urls_label VARCHAR(120),

  -- URL aktif (wajib)
  user_attendance_urls_href TEXT NOT NULL,

  -- housekeeping (opsional)
  user_attendance_urls_trash_url            TEXT,
  user_attendance_urls_delete_pending_until TIMESTAMPTZ,

  -- uploader (bisa teacher/user; salah satu/null keduanya ok)
  user_attendance_urls_uploader_teacher_id UUID,
  user_attendance_urls_uploader_user_id    UUID
    REFERENCES users(id) ON DELETE SET NULL,

  -- timestamps (dikelola aplikasi/GORM)
  user_attendance_urls_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_urls_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_urls_deleted_at TIMESTAMPTZ
);

-- Tambahkan FK ke masjid_teachers(masjid_teacher_id) bila tabel & kolomnya ada
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema='public' AND table_name='masjid_teachers'
  ) AND EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='masjid_teachers' AND column_name='masjid_teacher_id'
  ) THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_uau_uploader_teacher') THEN
      ALTER TABLE user_attendance_urls
        ADD CONSTRAINT fk_uau_uploader_teacher
        FOREIGN KEY (user_attendance_urls_uploader_teacher_id)
        REFERENCES masjid_teachers(masjid_teacher_id)
        ON DELETE SET NULL;
    END IF;
  ELSE
    RAISE NOTICE 'masjid_teachers / masjid_teacher_id belum ada; lewati FK user_attendance_urls → masjid_teachers.';
  END IF;
END$$;

-- Tenant guard: masjid child harus sama dengan masjid parent attendance
CREATE OR REPLACE FUNCTION fn_user_attendance_urls_tenant_guard()
RETURNS TRIGGER AS $$
DECLARE v_mid UUID;
BEGIN
  SELECT user_attendance_masjid_id
    INTO v_mid
  FROM user_attendance
  WHERE user_attendance_id = NEW.user_attendance_urls_attendance_id
    AND user_attendance_deleted_at IS NULL;

  IF v_mid IS NULL THEN
    RAISE EXCEPTION 'Kehadiran tidak valid/terhapus';
  END IF;

  IF NEW.user_attendance_urls_masjid_id IS DISTINCT FROM v_mid THEN
    RAISE EXCEPTION 'Masjid mismatch: url(%) vs attendance(%)',
      NEW.user_attendance_urls_masjid_id, v_mid;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_user_attendance_urls_tenant_guard ON user_attendance_urls;
CREATE TRIGGER trg_user_attendance_urls_tenant_guard
  BEFORE INSERT OR UPDATE ON user_attendance_urls
  FOR EACH ROW EXECUTE FUNCTION fn_user_attendance_urls_tenant_guard();

-- Indexes dasar
CREATE INDEX IF NOT EXISTS idx_user_attendance_urls_attendance
  ON user_attendance_urls(user_attendance_urls_attendance_id);

CREATE INDEX IF NOT EXISTS idx_user_attendance_urls_masjid_created_at
  ON user_attendance_urls(user_attendance_urls_masjid_id, user_attendance_urls_created_at DESC);

CREATE INDEX IF NOT EXISTS idx_uau_uploader_teacher
  ON user_attendance_urls(user_attendance_urls_uploader_teacher_id);

CREATE INDEX IF NOT EXISTS idx_uau_uploader_user
  ON user_attendance_urls(user_attendance_urls_uploader_user_id);

-- Optimizations
CREATE INDEX IF NOT EXISTS idx_uau_attendance_alive
  ON user_attendance_urls(user_attendance_urls_attendance_id, user_attendance_urls_created_at DESC)
  WHERE user_attendance_urls_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS brin_uau_created_at
  ON user_attendance_urls USING BRIN (user_attendance_urls_created_at);

CREATE UNIQUE INDEX IF NOT EXISTS uq_uau_attendance_href
  ON user_attendance_urls(user_attendance_urls_attendance_id, user_attendance_urls_href)
  WHERE user_attendance_urls_deleted_at IS NULL;

COMMIT;
