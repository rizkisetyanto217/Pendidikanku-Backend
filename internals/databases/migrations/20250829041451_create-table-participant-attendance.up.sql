-- +migrate Up
-- =========================================
-- UP Migration — Class Attendance Session Participants (student + teacher)
-- =========================================
BEGIN;

-- =========================================
-- EXTENSIONS
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE, search)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- kombinasi index opsional

-- =========================================
-- ENUMS (idempotent)
-- =========================================
DO $$
BEGIN
  -- jenis peserta: student / teacher / assistant / guest
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'participant_kind_enum') THEN
    CREATE TYPE participant_kind_enum AS ENUM ('student','teacher','assistant','guest');
  END IF;

  -- peran guru di sesi
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'teacher_role_enum') THEN
    CREATE TYPE teacher_role_enum AS ENUM ('primary','co','substitute','observer','assistant');
  END IF;

  -- status kehadiran umum
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'attendance_state_enum') THEN
    CREATE TYPE attendance_state_enum AS ENUM ('present','absent','late','excused','sick','leave', 'unmarked');
  END IF;

  -- =========================================================
-- ENUM baru: attendance_window_mode_enum
-- =========================================================
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'attendance_window_mode_enum') THEN
    CREATE TYPE attendance_window_mode_enum AS ENUM (
      'anytime',         -- bebas kapan saja
      'same_day',        -- hanya di hari H (00:00–23:59 waktu lokal)
      'three_days',      -- H-1, H, H+1
      'session_time',    -- hanya saat sesi berlangsung
      'relative_window'  -- pakai offset menit (open/close) relatif dari jam sesi
    );
  END IF;
END$$;


CREATE TABLE IF NOT EXISTS class_attendance_session_participant_types (
  class_attendance_session_participant_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant guard
  class_attendance_session_participant_type_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- data utama
  class_attendance_session_participant_type_code  VARCHAR(32)  NOT NULL,  -- ex: SETORAN, MURAJAAH, TILAWAH
  class_attendance_session_participant_type_label VARCHAR(80),
  class_attendance_session_participant_type_slug  VARCHAR(120),
  class_attendance_session_participant_type_color VARCHAR(20),
  class_attendance_session_participant_type_desc  TEXT,

  -- konfigurasi per participant-type
  class_attendance_session_participant_type_allow_student_self_attendance  BOOLEAN NOT NULL DEFAULT TRUE,
  class_attendance_session_participant_type_allow_teacher_mark_attendance  BOOLEAN NOT NULL DEFAULT TRUE,
  class_attendance_session_participant_type_require_teacher_attendance     BOOLEAN NOT NULL DEFAULT TRUE,
  -- daftar state yang WAJIB punya alasan (reason) kalau dipilih
  class_attendance_session_participant_type_require_attendance_reason
    attendance_state_enum[] NOT NULL DEFAULT ARRAY['unmarked']::attendance_state_enum[],
  class_attendance_session_participant_type_meta                           JSONB,
  class_attendance_session_type_attendance_window_mode          attendance_window_mode_enum NOT NULL DEFAULT 'same_day',
  class_attendance_session_type_attendance_open_offset_minutes  INT,
  class_attendance_session_type_attendance_close_offset_minutes INT
  -- status
  class_attendance_session_participant_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  class_attendance_session_participant_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_participant_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_participant_type_deleted_at TIMESTAMPTZ
);

-- Unik per school + code (case-insensitive, soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_caspt_code_per_school_alive
  ON class_attendance_session_participant_types (
    class_attendance_session_participant_type_school_id,
    UPPER(class_attendance_session_participant_type_code)
  )
  WHERE class_attendance_session_participant_type_deleted_at IS NULL;

-- Pencarian label cepat (trigram) untuk data aktif
CREATE INDEX IF NOT EXISTS gin_caspt_label_trgm
  ON class_attendance_session_participant_types USING GIN (
    class_attendance_session_participant_type_label gin_trgm_ops
  )
  WHERE class_attendance_session_participant_type_deleted_at IS NULL;

-- Filter umum per school (aktif saja)
CREATE INDEX IF NOT EXISTS idx_caspt_school_active
  ON class_attendance_session_participant_types (
    class_attendance_session_participant_type_school_id,
    class_attendance_session_participant_type_is_active
  )
  WHERE class_attendance_session_participant_type_deleted_at IS NULL;

-- Listing terbaru per school
CREATE INDEX IF NOT EXISTS idx_caspt_school_created_desc
  ON class_attendance_session_participant_types (
    class_attendance_session_participant_type_school_id,
    class_attendance_session_participant_type_created_at DESC
  )
  WHERE class_attendance_session_participant_type_deleted_at IS NULL;

-- BRIN untuk time-series
CREATE INDEX IF NOT EXISTS brin_caspt_created_at
  ON class_attendance_session_participant_types USING BRIN (
    class_attendance_session_participant_type_created_at
  );



BEGIN;

-- =========================================
-- B) CLASS_ATTENDANCE_SESSION_PARTICIPANTS
--    (per peserta per sesi: student/teacher/assistant/guest)
-- =========================================
CREATE TABLE IF NOT EXISTS class_attendance_session_participants (
  class_attendance_session_participant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & relasi utama
  class_attendance_session_participant_school_id  UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  class_attendance_session_participant_session_id UUID NOT NULL
    REFERENCES class_attendance_sessions(class_attendance_session_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- jenis peserta
  class_attendance_session_participant_kind participant_kind_enum NOT NULL,  -- 'student','teacher','assistant','guest'

  -- relasi detail (opsional, tergantung kind)
  class_attendance_session_participant_school_student_id UUID
    REFERENCES school_students(school_student_id) ON DELETE CASCADE,

  class_attendance_session_participant_school_teacher_id UUID
    REFERENCES school_teachers(school_teacher_id) ON DELETE SET NULL,

  class_attendance_session_participant_teacher_role teacher_role_enum,

  -- status kehadiran (enum global)
  class_attendance_session_participant_state attendance_state_enum NOT NULL DEFAULT 'unmarked',

  -- jam checkin/checkout (bisa dipakai untuk jam guru absen juga)
  class_attendance_session_participant_checkin_at  TIMESTAMPTZ,
  class_attendance_session_participant_checkout_at TIMESTAMPTZ,

  -- jenis kegiatan/tipe absensi (opsional)
  class_attendance_session_participant_type_id UUID
    REFERENCES class_attendance_session_participant_types(class_attendance_session_participant_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- kualitas/penilaian harian (opsional)
  class_attendance_session_participant_desc      TEXT,
  class_attendance_session_participant_score     NUMERIC(5,2)
    CHECK (
      class_attendance_session_participant_score IS NULL
      OR class_attendance_session_participant_score BETWEEN 0 AND 100
    ),
  class_attendance_session_participant_is_passed BOOLEAN,

  -- meta penandaan
  class_attendance_session_participant_marked_at TIMESTAMPTZ,
  class_attendance_session_participant_marked_by_teacher_id UUID
    REFERENCES school_teachers(school_teacher_id) ON DELETE SET NULL,

  -- metode absen (flow/source, bukan device enum)
  class_attendance_session_participant_method VARCHAR(16),
  CONSTRAINT chk_cas_participant_method
    CHECK (
      class_attendance_session_participant_method IS NULL
      OR class_attendance_session_participant_method IN ('manual','qr','geo','import','api','self')
    ),

  -- geolocation (opsional; utk self/geo)
  class_attendance_session_participant_lat        DOUBLE PRECISION,
  class_attendance_session_participant_lng        DOUBLE PRECISION,
  class_attendance_session_participant_distance_m INTEGER
    CHECK (
      class_attendance_session_participant_distance_m IS NULL
      OR class_attendance_session_participant_distance_m >= 0
    ),

  -- keterlambatan (detik)
  class_attendance_session_participant_late_seconds INT
    CHECK (
      class_attendance_session_participant_late_seconds IS NULL
      OR class_attendance_session_participant_late_seconds >= 0
    ),

  -- Snapshot users_profile (per siswa saat sesi dibuat)
  class_attendance_session_participant_user_profile_name_snapshot                VARCHAR(80),
  class_attendance_session_participant_user_profile_avatar_url_snapshot          VARCHAR(255),
  class_attendance_session_participant_user_profile_whatsapp_url_snapshot        VARCHAR(50),
  class_attendance_session_participant_user_profile_parent_name_snapshot         VARCHAR(80),
  class_attendance_session_participant_user_profile_parent_whatsapp_url_snapshot VARCHAR(50),
  class_attendance_session_participant_user_profile_gender_snapshot              VARCHAR(20),

  -- catatan tambahan
  class_attendance_session_participant_user_note    TEXT,
  class_attendance_session_participant_teacher_note TEXT,

  -- locking (opsional)
  class_attendance_session_participant_locked_at TIMESTAMPTZ,

  -- audit
  class_attendance_session_participant_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_participant_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_participant_deleted_at TIMESTAMPTZ
);

-- =========================================
-- INDEXES CLASS_ATTENDANCE_SESSION_PARTICIPANTS
-- =========================================

-- unik aktif per (school, session, student)
CREATE UNIQUE INDEX IF NOT EXISTS uq_casp_student_alive
  ON class_attendance_session_participants (
    class_attendance_session_participant_school_id,
    class_attendance_session_participant_session_id,
    class_attendance_session_participant_school_student_id
  )
  WHERE class_attendance_session_participant_deleted_at IS NULL
    AND class_attendance_session_participant_school_student_id IS NOT NULL;

-- unik aktif per (school, session, teacher)
CREATE UNIQUE INDEX IF NOT EXISTS uq_casp_teacher_alive
  ON class_attendance_session_participants (
    class_attendance_session_participant_school_id,
    class_attendance_session_participant_session_id,
    class_attendance_session_participant_school_teacher_id
  )
  WHERE class_attendance_session_participant_deleted_at IS NULL
    AND class_attendance_session_participant_school_teacher_id IS NOT NULL;

-- jalur query umum
CREATE INDEX IF NOT EXISTS idx_casp_session
  ON class_attendance_session_participants (class_attendance_session_participant_session_id)
  WHERE class_attendance_session_participant_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_casp_student
  ON class_attendance_session_participants (class_attendance_session_participant_school_student_id)
  WHERE class_attendance_session_participant_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_casp_teacher
  ON class_attendance_session_participants (class_attendance_session_participant_school_teacher_id)
  WHERE class_attendance_session_participant_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_casp_state
  ON class_attendance_session_participants (class_attendance_session_participant_state)
  WHERE class_attendance_session_participant_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_casp_type_id
  ON class_attendance_session_participants (class_attendance_session_participant_type_id)
  WHERE class_attendance_session_participant_deleted_at IS NULL;

-- rekap cepat per sesi+state
CREATE INDEX IF NOT EXISTS idx_casp_session_state
  ON class_attendance_session_participants (
    class_attendance_session_participant_session_id,
    class_attendance_session_participant_state
  )
  WHERE class_attendance_session_participant_deleted_at IS NULL;

-- time-series
CREATE INDEX IF NOT EXISTS brin_casp_created_at
  ON class_attendance_session_participants USING BRIN (
    class_attendance_session_participant_created_at
  );

CREATE INDEX IF NOT EXISTS brin_casp_marked_at
  ON class_attendance_session_participants USING BRIN (
    class_attendance_session_participant_marked_at
  );

-- teks
CREATE INDEX IF NOT EXISTS gin_casp_desc_trgm
  ON class_attendance_session_participants USING GIN (
    class_attendance_session_participant_desc gin_trgm_ops
  )
  WHERE class_attendance_session_participant_deleted_at IS NULL;

COMMIT;


-- =========================================
-- C) CLASS_ATTENDANCE_SESSION_PARTICIPANT_URLS (lampiran/url per participant)
-- =========================================
CREATE TABLE IF NOT EXISTS class_attendance_session_participant_urls (
  class_attendance_session_participant_url_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  class_attendance_session_participant_url_school_id   UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  class_attendance_session_participant_url_participant_id UUID NOT NULL
    REFERENCES class_attendance_session_participants(class_attendance_session_participant_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- (opsional) tipe media eksternal (lookup table)
  class_attendance_session_participant_type_id UUID
    REFERENCES class_attendance_session_participant_types(class_attendance_session_participant_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Jenis/peran aset (mis. 'image','video','attachment','link','audio', dll.)
  class_attendance_session_participant_url_kind VARCHAR(24) NOT NULL,

  -- Lokasi file/link (skema dua-slot + retensi)
  class_attendance_session_participant_url                  TEXT,
  class_attendance_session_participant_url_object_key       TEXT,
  class_attendance_session_participant_url_old              TEXT,
  class_attendance_session_participant_url_object_key_old   TEXT,
  class_attendance_session_participant_url_delete_pending_until TIMESTAMPTZ,

  -- Metadata tampilan
  class_attendance_session_participant_url_label      VARCHAR(160),
  class_attendance_session_participant_url_order      INT NOT NULL DEFAULT 0,
  class_attendance_session_participant_url_is_primary BOOLEAN NOT NULL DEFAULT FALSE,

  -- Uploader (opsional)
  class_attendance_session_participant_url_uploader_teacher_id  UUID
    REFERENCES school_teachers(school_teacher_id),
  class_attendance_session_participant_url_uploader_student_id  UUID
    REFERENCES school_students(school_student_id) ON DELETE SET NULL,

  -- Audit
  class_attendance_session_participant_url_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_participant_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_participant_url_deleted_at TIMESTAMPTZ
);

-- Lookup per participant (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_caspurl_by_owner_live
  ON class_attendance_session_participant_urls (
    class_attendance_session_participant_url_participant_id,
    class_attendance_session_participant_url_kind,
    class_attendance_session_participant_url_is_primary DESC,
    class_attendance_session_participant_url_order,
    class_attendance_session_participant_url_created_at
  )
  WHERE class_attendance_session_participant_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_caspurl_by_school_live
  ON class_attendance_session_participant_urls (class_attendance_session_participant_url_school_id)
  WHERE class_attendance_session_participant_url_deleted_at IS NULL;

-- Satu primary per (participant, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_caspurl_primary_per_kind_alive
  ON class_attendance_session_participant_urls (
    class_attendance_session_participant_url_participant_id,
    class_attendance_session_participant_url_kind
  )
  WHERE class_attendance_session_participant_url_deleted_at IS NULL
    AND class_attendance_session_participant_url_is_primary = TRUE;

-- 🔧 Anti-duplikat **URL** per participant (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_caspurl_participant_url_alive
  ON class_attendance_session_participant_urls (
    class_attendance_session_participant_url_participant_id,
    LOWER(class_attendance_session_participant_url)
  )
  WHERE class_attendance_session_participant_url_deleted_at IS NULL
    AND class_attendance_session_participant_url IS NOT NULL;

-- Kandidat purge:
CREATE INDEX IF NOT EXISTS ix_caspurl_purge_due
  ON class_attendance_session_participant_urls (
    class_attendance_session_participant_url_delete_pending_until
  )
  WHERE class_attendance_session_participant_url_delete_pending_until IS NOT NULL
    AND (
      (class_attendance_session_participant_url_deleted_at IS NULL  AND class_attendance_session_participant_url_object_key_old IS NOT NULL) OR
      (class_attendance_session_participant_url_deleted_at IS NOT NULL AND class_attendance_session_participant_url_object_key     IS NOT NULL)
    );

-- Uploader lookups (live only)
CREATE INDEX IF NOT EXISTS ix_caspurl_uploader_teacher_live
  ON class_attendance_session_participant_urls (class_attendance_session_participant_url_uploader_teacher_id)
  WHERE class_attendance_session_participant_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_caspurl_uploader_student_live
  ON class_attendance_session_participant_urls (class_attendance_session_participant_url_uploader_student_id)
  WHERE class_attendance_session_participant_url_deleted_at IS NULL;

-- Time-scan (arsip/waktu)
CREATE INDEX IF NOT EXISTS brin_caspurl_created_at
  ON class_attendance_session_participant_urls USING BRIN (
    class_attendance_session_participant_url_created_at
  );

-- (opsional) pencarian label cepat (live only)
-- CREATE INDEX IF NOT EXISTS gin_caspurl_label_trgm_live
--   ON class_attendance_session_participant_urls USING GIN (
--     class_attendance_session_participant_url_label gin_trgm_ops
--   )
--   WHERE class_attendance_session_participant_url_deleted_at IS NULL;

COMMIT;
