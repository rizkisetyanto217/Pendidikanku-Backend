-- +migrate Up
-- =========================================
-- UP Migration — Student Class Session Attendances (renamed, explicit)
-- =========================================
BEGIN;

-- =========================================
-- EXTENSIONS
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE, search)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- kombinasi index opsional

-- =========================================
-- A) STUDENT_CLASS_SESSION_ATTENDANCE_TYPES (master jenis attendance per masjid)
-- =========================================
CREATE TABLE IF NOT EXISTS student_class_session_attendance_types (
  student_class_session_attendance_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant guard
  student_class_session_attendance_type_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- data utama
  student_class_session_attendance_type_code  VARCHAR(32)  NOT NULL,  -- ex: SETORAN, MURAJAAH, TILAWAH
  student_class_session_attendance_type_label VARCHAR(80),
  student_class_session_attendance_type_slug  VARCHAR(120),
  student_class_session_attendance_type_color VARCHAR(20),
  student_class_session_attendance_type_desc  TEXT,

  -- status
  student_class_session_attendance_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  student_class_session_attendance_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_session_attendance_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_session_attendance_type_deleted_at TIMESTAMPTZ
);

-- Unik per masjid + code (case-insensitive, soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_scsat_code_per_masjid_alive
  ON student_class_session_attendance_types (student_class_session_attendance_type_masjid_id, UPPER(student_class_session_attendance_type_code))
  WHERE student_class_session_attendance_type_deleted_at IS NULL;

-- Pencarian label cepat (trigram) untuk data aktif
CREATE INDEX IF NOT EXISTS gin_scsat_label_trgm
  ON student_class_session_attendance_types USING GIN (student_class_session_attendance_type_label gin_trgm_ops)
  WHERE student_class_session_attendance_type_deleted_at IS NULL;

-- Filter umum per masjid (aktif saja)
CREATE INDEX IF NOT EXISTS idx_scsat_masjid_active
  ON student_class_session_attendance_types (student_class_session_attendance_type_masjid_id, student_class_session_attendance_type_is_active)
  WHERE student_class_session_attendance_type_deleted_at IS NULL;

-- Listing terbaru per masjid
CREATE INDEX IF NOT EXISTS idx_scsat_masjid_created_desc
  ON student_class_session_attendance_types (student_class_session_attendance_type_masjid_id, student_class_session_attendance_type_created_at DESC)
  WHERE student_class_session_attendance_type_deleted_at IS NULL;

-- BRIN untuk time-series
CREATE INDEX IF NOT EXISTS brin_scsat_created_at
  ON student_class_session_attendance_types USING BRIN (student_class_session_attendance_type_created_at);




-- =========================================
-- B) STUDENT_CLASS_SESSION_ATTENDANCES (per siswa per sesi)
-- =========================================
CREATE TABLE IF NOT EXISTS student_class_session_attendances (
  student_class_session_attendance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & relasi utama
  student_class_session_attendance_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  student_class_session_attendance_session_id UUID NOT NULL
    REFERENCES class_attendance_sessions(class_attendance_session_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  student_class_session_attendance_masjid_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE CASCADE,

  -- status kehadiran
  student_class_session_attendance_status VARCHAR(16) NOT NULL DEFAULT 'unmarked'
    CHECK (student_class_session_attendance_status IN ('unmarked','present','absent','excused','late')),

  -- jenis kegiatan/tipe absensi (opsional)
  student_class_session_attendance_type_id UUID
    REFERENCES student_class_session_attendance_types(student_class_session_attendance_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- kualitas/penilaian harian (opsional)
  student_class_session_attendance_desc      TEXT,
  student_class_session_attendance_score     NUMERIC(5,2)
    CHECK (student_class_session_attendance_score IS NULL OR student_class_session_attendance_score BETWEEN 0 AND 100),
  student_class_session_attendance_is_passed BOOLEAN,

  -- meta penandaan
  student_class_session_attendance_marked_at TIMESTAMPTZ,
  student_class_session_attendance_marked_by_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,

  -- metode absen
  student_class_session_attendance_method VARCHAR(16),
  CONSTRAINT chk_scsa_method
    CHECK (
      student_class_session_attendance_method IS NULL
      OR student_class_session_attendance_method IN ('manual','qr','geo','import','api','self')
    ),

  -- geolocation (opsional; utk self/geo)
  student_class_session_attendance_lat        DOUBLE PRECISION,
  student_class_session_attendance_lng        DOUBLE PRECISION,
  student_class_session_attendance_distance_m INTEGER CHECK (student_class_session_attendance_distance_m IS NULL OR student_class_session_attendance_distance_m >= 0),

  -- keterlambatan (detik)
  student_class_session_attendance_late_seconds INT
    CHECK (student_class_session_attendance_late_seconds IS NULL OR student_class_session_attendance_late_seconds >= 0),

  -- catatan tambahan
  student_class_session_attendance_user_note    TEXT,
  student_class_session_attendance_teacher_note TEXT,

  -- locking (opsional)
  student_class_session_attendance_locked_at TIMESTAMPTZ,

  -- audit
  student_class_session_attendance_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_session_attendance_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_session_attendance_deleted_at TIMESTAMPTZ
);

-- =========================================
-- INDEXES STUDENT_CLASS_SESSION_ATTENDANCES
-- =========================================

-- unik aktif per (masjid, session, student)
CREATE UNIQUE INDEX IF NOT EXISTS uq_scsa_alive
  ON student_class_session_attendances (
    student_class_session_attendance_masjid_id,
    student_class_session_attendance_session_id,
    student_class_session_attendance_masjid_student_id
  )
  WHERE student_class_session_attendance_deleted_at IS NULL;

-- jalur query umum
CREATE INDEX IF NOT EXISTS idx_scsa_session
  ON student_class_session_attendances (student_class_session_attendance_session_id)
  WHERE student_class_session_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_scsa_student
  ON student_class_session_attendances (student_class_session_attendance_masjid_student_id)
  WHERE student_class_session_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_scsa_status
  ON student_class_session_attendances (student_class_session_attendance_status)
  WHERE student_class_session_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_scsa_type_id
  ON student_class_session_attendances (student_class_session_attendance_type_id)
  WHERE student_class_session_attendance_deleted_at IS NULL;

-- rekap cepat per sesi+status
CREATE INDEX IF NOT EXISTS idx_scsa_session_status
  ON student_class_session_attendances (student_class_session_attendance_session_id, student_class_session_attendance_status)
  WHERE student_class_session_attendance_deleted_at IS NULL;

-- time-series
CREATE INDEX IF NOT EXISTS brin_scsa_created_at
  ON student_class_session_attendances USING BRIN (student_class_session_attendance_created_at);

CREATE INDEX IF NOT EXISTS brin_scsa_marked_at
  ON student_class_session_attendances USING BRIN (student_class_session_attendance_marked_at);

-- teks
CREATE INDEX IF NOT EXISTS gin_scsa_desc_trgm
  ON student_class_session_attendances USING GIN (student_class_session_attendance_desc gin_trgm_ops)
  WHERE student_class_session_attendance_deleted_at IS NULL;




-- =========================================
-- C) STUDENT_CLASS_SESSION_ATTENDANCE_URLS (lampiran/url per attendance)
-- =========================================
CREATE TABLE IF NOT EXISTS student_class_session_attendance_urls (
  student_class_session_attendance_url_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  student_class_session_attendance_url_masjid_id   UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  student_class_session_attendance_url_attendance_id UUID NOT NULL
    REFERENCES student_class_session_attendances(student_class_session_attendance_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- (opsional) tipe media eksternal (lookup table)
  student_class_session_attendance_type_id UUID
    REFERENCES student_class_session_attendance_types(student_class_session_attendance_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Jenis/peran aset (mis. 'image','video','attachment','link','audio', dll.)
  student_class_session_attendance_url_kind VARCHAR(24) NOT NULL,

  -- Lokasi file/link (skema dua-slot + retensi)
  student_class_session_attendance_url                  TEXT,
  student_class_session_attendance_url_object_key       TEXT,
  student_class_session_attendance_url_old              TEXT,
  student_class_session_attendance_url_object_key_old   TEXT,
  student_class_session_attendance_url_delete_pending_until TIMESTAMPTZ,

  -- Metadata tampilan
  student_class_session_attendance_url_label      VARCHAR(160),
  student_class_session_attendance_url_order      INT NOT NULL DEFAULT 0,
  student_class_session_attendance_url_is_primary BOOLEAN NOT NULL DEFAULT FALSE,

  -- Uploader (opsional)
  student_class_session_attendance_url_uploader_teacher_id  UUID REFERENCES masjid_teachers(masjid_teacher_id),
  student_class_session_attendance_url_uploader_student_id  UUID
    REFERENCES masjid_students(masjid_student_id) ON DELETE SET NULL,

  -- Audit
  student_class_session_attendance_url_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_session_attendance_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  student_class_session_attendance_url_deleted_at TIMESTAMPTZ
);

-- Lookup per attendance (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_scsaurl_by_owner_live
  ON student_class_session_attendance_urls (
    student_class_session_attendance_url_attendance_id,
    student_class_session_attendance_url_kind,
    student_class_session_attendance_url_is_primary DESC,
    student_class_session_attendance_url_order,
    student_class_session_attendance_url_created_at
  )
  WHERE student_class_session_attendance_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_scsaurl_by_masjid_live
  ON student_class_session_attendance_urls (student_class_session_attendance_url_masjid_id)
  WHERE student_class_session_attendance_url_deleted_at IS NULL;

-- Satu primary per (attendance, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_scsaurl_primary_per_kind_alive
  ON student_class_session_attendance_urls (
    student_class_session_attendance_url_attendance_id,
    student_class_session_attendance_url_kind
  )
  WHERE student_class_session_attendance_url_deleted_at IS NULL
    AND student_class_session_attendance_url_is_primary = TRUE;

-- 🔧 Anti-duplikat **URL** per attendance (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_scsaurl_attendance_url_alive
  ON student_class_session_attendance_urls (
    student_class_session_attendance_url_attendance_id,
    LOWER(student_class_session_attendance_url)
  )
  WHERE student_class_session_attendance_url_deleted_at IS NULL
    AND student_class_session_attendance_url IS NOT NULL;

-- Kandidat purge:
CREATE INDEX IF NOT EXISTS ix_scsaurl_purge_due
  ON student_class_session_attendance_urls (student_class_session_attendance_url_delete_pending_until)
  WHERE student_class_session_attendance_url_delete_pending_until IS NOT NULL
    AND (
      (student_class_session_attendance_url_deleted_at IS NULL  AND student_class_session_attendance_url_object_key_old IS NOT NULL) OR
      (student_class_session_attendance_url_deleted_at IS NOT NULL AND student_class_session_attendance_url_object_key     IS NOT NULL)
    );

-- Uploader lookups (live only)
CREATE INDEX IF NOT EXISTS ix_scsaurl_uploader_teacher_live
  ON student_class_session_attendance_urls (student_class_session_attendance_url_uploader_teacher_id)
  WHERE student_class_session_attendance_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_scsaurl_uploader_student_live
  ON student_class_session_attendance_urls (student_class_session_attendance_url_uploader_student_id)
  WHERE student_class_session_attendance_url_deleted_at IS NULL;

-- Time-scan (arsip/waktu)
CREATE INDEX IF NOT EXISTS brin_scsaurl_created_at
  ON student_class_session_attendance_urls USING BRIN (student_class_session_attendance_url_created_at);

-- (opsional) pencarian label cepat (live only)
-- CREATE INDEX IF NOT EXISTS gin_scsaurl_label_trgm_live
--   ON student_class_session_attendance_urls USING GIN (student_class_session_attendance_url_label gin_trgm_ops)
--   WHERE student_class_session_attendance_url_deleted_at IS NULL;

COMMIT;
