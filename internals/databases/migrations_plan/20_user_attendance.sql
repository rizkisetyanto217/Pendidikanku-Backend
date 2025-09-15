BEGIN;

-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- trigram ops (label search)

-- =========================================================
-- A) USER_ATTENDANCE_TYPE (master jenis attendance per masjid)
--    FINAL CREATE (semua kolom sudah terintegrasi)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_attendance_type (
  user_attendance_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_attendance_type_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Identitas dasar
  user_attendance_type_code  VARCHAR(32) NOT NULL,  -- ex: SETORAN, MURAJAAH, TILAWAH
  user_attendance_type_label VARCHAR(80),
  user_attendance_type_desc  TEXT,

  -- Status
  user_attendance_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- Presentasi & urutan
  user_attendance_type_order_index INT,
  user_attendance_type_color       VARCHAR(20),
  user_attendance_type_icon        VARCHAR(64),
  user_attendance_type_slug        VARCHAR(120),

  -- Kebijakan input & visibilitas
  user_attendance_type_require_desc        BOOLEAN DEFAULT FALSE,
  user_attendance_type_require_evidence    BOOLEAN DEFAULT FALSE,
  user_attendance_type_visible_to_guardian BOOLEAN DEFAULT TRUE,
  user_attendance_type_allowed_channels    TEXT[],   -- ['web','mobile','kiosk','api']

  -- Rentang aktif
  user_attendance_type_active_from DATE,
  user_attendance_type_active_to   DATE,

  -- Skoring & pelaporan
  user_attendance_type_default_score  NUMERIC(5,2),
  user_attendance_type_min_score      NUMERIC(5,2),
  user_attendance_type_max_score      NUMERIC(5,2),
  user_attendance_type_passing_score  NUMERIC(5,2),
  user_attendance_type_weight_percent NUMERIC(6,3),

  -- Batas & kuota
  user_attendance_type_max_per_day_per_student     SMALLINT,
  user_attendance_type_max_per_session_per_student SMALLINT,

  -- Kategori & workflow
  user_attendance_type_category        TEXT,     -- 'quran','activity','behavior',...
  user_attendance_type_requires_review BOOLEAN DEFAULT FALSE,
  user_attendance_type_is_summative    BOOLEAN DEFAULT FALSE,

  -- Form dinamis, rubric & mapping ekspor
  user_attendance_type_form_schema JSONB,  -- JSON Schema form
  user_attendance_type_rubric      JSONB,  -- rubrik skoring
  user_attendance_type_export_mapping JSONB,

  -- Audit
  user_attendance_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_type_deleted_at TIMESTAMPTZ
);

-- Unik per masjid + code (soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_uatt_code_per_masjid_alive
  ON user_attendance_type (
    user_attendance_type_masjid_id,
    UPPER(user_attendance_type_code)
  )
  WHERE user_attendance_type_deleted_at IS NULL;

-- Pencarian label cepat (trigram) untuk data aktif
CREATE INDEX IF NOT EXISTS gin_uatt_label_trgm
  ON user_attendance_type USING GIN (user_attendance_type_label gin_trgm_ops)
  WHERE user_attendance_type_deleted_at IS NULL;

-- Filter umum per masjid (aktif saja)
CREATE INDEX IF NOT EXISTS idx_uatt_masjid_active
  ON user_attendance_type (user_attendance_type_masjid_id, user_attendance_type_is_active)
  WHERE user_attendance_type_deleted_at IS NULL;

-- Listing terbaru per masjid
CREATE INDEX IF NOT EXISTS idx_uatt_masjid_created_desc
  ON user_attendance_type (user_attendance_type_masjid_id, user_attendance_type_created_at DESC)
  WHERE user_attendance_type_deleted_at IS NULL;

-- BRIN untuk time-series besar
CREATE INDEX IF NOT EXISTS brin_uatt_created_at
  ON user_attendance_type USING BRIN (user_attendance_type_created_at);

-- Guard slug tidak kosong (kalau diisi)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_uatt_slug_not_blank') THEN
    ALTER TABLE user_attendance_type
      ADD CONSTRAINT chk_uatt_slug_not_blank
      CHECK (user_attendance_type_slug IS NULL OR length(btrim(user_attendance_type_slug)) > 0);
  END IF;
END$$;


-- =========================================================
-- B) USER_ATTENDANCE (per siswa per sesi) — FINAL CREATE
--    (semua kolom & FTS sudah terintegrasi)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_attendance (
  user_attendance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & relasi
  user_attendance_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_attendance_session_id UUID NOT NULL
    REFERENCES class_attendance_sessions(class_attendance_sessions_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  user_attendance_masjid_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE CASCADE,

  -- status kehadiran
  user_attendance_status VARCHAR(16) NOT NULL DEFAULT 'present'
    CHECK (user_attendance_status IN ('present','absent','excused','late')),

  -- jenis kegiatan (nullable)
  user_attendance_type_id UUID
    REFERENCES public.user_attendance_type(user_attendance_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- konten & hasil
  user_attendance_desc       TEXT,
  user_attendance_score      NUMERIC(5,2) CHECK (user_attendance_score IS NULL OR user_attendance_score BETWEEN 0 AND 100),
  user_attendance_is_passed  BOOLEAN,

  -- catatan tambahan
  user_attendance_user_note    TEXT,
  user_attendance_teacher_note TEXT,

  -- bukti, audit & idempotensi
  user_attendance_verified          BOOLEAN DEFAULT FALSE,
  user_attendance_verified_by_user_id UUID,
  user_attendance_verified_at       TIMESTAMPTZ,

  -- alasan/justifikasi
  user_attendance_excuse_type TEXT,  -- 'sick'|'leave'|'traffic'|...
  user_attendance_excuse_note TEXT,

  user_attendance_approved        BOOLEAN,
  user_attendance_approved_at     TIMESTAMPTZ,
  user_attendance_approved_by_user_id UUID,
  user_attendance_approval_note   TEXT,

  -- hubungan remedial/makeup
  user_attendance_makeup_of_id UUID,  -- refer ke user_attendance_id lain

  -- review/QA
  user_attendance_review_status   TEXT,        -- 'pending'|'approved'|'rejected'
  user_attendance_reviewed_by_user_id UUID,
  user_attendance_reviewed_at     TIMESTAMPTZ,
  user_attendance_review_note     TEXT,

  -- FTS (desc & notes)
  user_attendance_search tsvector
    GENERATED ALWAYS AS (
      setweight(to_tsvector('simple', coalesce(user_attendance_desc,'')), 'A') ||
      setweight(to_tsvector('simple', coalesce(user_attendance_user_note,'')), 'B') ||
      setweight(to_tsvector('simple', coalesce(user_attendance_teacher_note,'')), 'B')
    ) STORED,

  -- timestamps
  user_attendance_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_deleted_at TIMESTAMPTZ
);

-- Anti-duplikat: satu baris per (masjid, session, student) untuk data hidup
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_attendance_per_session_student_alive
  ON user_attendance (user_attendance_masjid_id, user_attendance_session_id, user_attendance_masjid_student_id)
  WHERE user_attendance_deleted_at IS NULL;

-- Idempotensi import/API: cegah duplikat ketika dedup_key terisi
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_attendance_dedup_alive
  ON user_attendance (user_attendance_masjid_id, user_attendance_session_id, user_attendance_masjid_student_id, user_attendance_dedup_key)
  WHERE user_attendance_deleted_at IS NULL
    AND user_attendance_dedup_key IS NOT NULL
    AND length(btrim(user_attendance_dedup_key)) > 0;

-- FTS index
CREATE INDEX IF NOT EXISTS gin_user_attendance_search
  ON user_attendance USING GIN (user_attendance_search);

-- Query umum
CREATE INDEX IF NOT EXISTS idx_user_attendance_session_alive
  ON user_attendance (user_attendance_session_id, user_attendance_status)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_student_alive
  ON user_attendance (user_attendance_masjid_student_id, user_attendance_status)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_type_alive
  ON user_attendance (user_attendance_type_id)
  WHERE user_attendance_deleted_at IS NULL;

-- Pipelines & laporan
CREATE INDEX IF NOT EXISTS idx_user_attendance_status_time_alive
  ON user_attendance (user_attendance_status, user_attendance_marked_at)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_review_pipeline
  ON user_attendance (user_attendance_review_status, user_attendance_reviewed_at)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_guardian_ack
  ON user_attendance (user_attendance_guardian_user_id, user_attendance_guardian_acknowledged_at)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_export
  ON user_attendance (user_attendance_export_batch_id, user_attendance_exported_at)
  WHERE user_attendance_deleted_at IS NULL;

-- Listing terbaru per masjid
CREATE INDEX IF NOT EXISTS idx_user_attendance_masjid_created_desc
  ON user_attendance (user_attendance_masjid_id, user_attendance_created_at DESC)
  WHERE user_attendance_deleted_at IS NULL;

-- BRIN time-series
CREATE INDEX IF NOT EXISTS brin_user_attendance_created_at
  ON user_attendance USING BRIN (user_attendance_created_at);

COMMIT;