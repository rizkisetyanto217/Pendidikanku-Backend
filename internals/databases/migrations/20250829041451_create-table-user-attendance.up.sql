-- +migrate Up
-- =========================================
-- UP Migration — User Class Session Attendances (renamed, explicit)
-- =========================================
BEGIN;

-- =========================================
-- EXTENSIONS
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE, search)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- kombinasi index opsional

-- =========================================
-- A) USER_CLASS_SESSION_ATTENDANCE_TYPES (master jenis attendance per masjid)
-- =========================================
CREATE TABLE IF NOT EXISTS user_class_session_attendance_types (
  user_class_session_attendance_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant guard
  user_class_session_attendance_type_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- data utama
  user_class_session_attendance_type_code  VARCHAR(32)  NOT NULL,  -- ex: SETORAN, MURAJAAH, TILAWAH
  user_class_session_attendance_type_label VARCHAR(80),
  user_class_session_attendance_type_slug  VARCHAR(120),
  user_class_session_attendance_type_color VARCHAR(20),
  user_class_session_attendance_type_desc  TEXT,

  -- status
  user_class_session_attendance_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  user_class_session_attendance_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_session_attendance_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_session_attendance_type_deleted_at TIMESTAMPTZ
);

-- Unik per masjid + code (case-insensitive, soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsat_code_per_masjid_alive
  ON user_class_session_attendance_types (user_class_session_attendance_type_masjid_id, UPPER(user_class_session_attendance_type_code))
  WHERE user_class_session_attendance_type_deleted_at IS NULL;

-- Pencarian label cepat (trigram) untuk data aktif
CREATE INDEX IF NOT EXISTS gin_ucsat_label_trgm
  ON user_class_session_attendance_types USING GIN (user_class_session_attendance_type_label gin_trgm_ops)
  WHERE user_class_session_attendance_type_deleted_at IS NULL;

-- Filter umum per masjid (aktif saja)
CREATE INDEX IF NOT EXISTS idx_ucsat_masjid_active
  ON user_class_session_attendance_types (user_class_session_attendance_type_masjid_id, user_class_session_attendance_type_is_active)
  WHERE user_class_session_attendance_type_deleted_at IS NULL;

-- Listing terbaru per masjid
CREATE INDEX IF NOT EXISTS idx_ucsat_masjid_created_desc
  ON user_class_session_attendance_types (user_class_session_attendance_type_masjid_id, user_class_session_attendance_type_created_at DESC)
  WHERE user_class_session_attendance_type_deleted_at IS NULL;

-- BRIN untuk time-series
CREATE INDEX IF NOT EXISTS brin_ucsat_created_at
  ON user_class_session_attendance_types USING BRIN (user_class_session_attendance_type_created_at);




-- =========================================
-- B) USER_CLASS_SESSION_ATTENDANCES (per siswa per sesi)
-- =========================================
CREATE TABLE IF NOT EXISTS user_class_session_attendances (
  user_class_session_attendance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & relasi utama
  user_class_session_attendance_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_class_session_attendance_session_id UUID NOT NULL
    REFERENCES class_attendance_sessions(class_attendance_session_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  user_class_session_attendance_masjid_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE CASCADE,

  -- status kehadiran
  user_class_session_attendance_status VARCHAR(16) NOT NULL DEFAULT 'unmarked'
    CHECK (user_class_session_attendance_status IN ('unmarked','present','absent','excused','late')),

  -- jenis kegiatan/tipe absensi (opsional)
  user_class_session_attendance_type_id UUID
    REFERENCES user_class_session_attendance_types(user_class_session_attendance_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- kualitas/penilaian harian (opsional)
  user_class_session_attendance_desc      TEXT,
  user_class_session_attendance_score     NUMERIC(5,2)
    CHECK (user_class_session_attendance_score IS NULL OR user_class_session_attendance_score BETWEEN 0 AND 100),
  user_class_session_attendance_is_passed BOOLEAN,

  -- meta penandaan
  user_class_session_attendance_marked_at TIMESTAMPTZ,
  user_class_session_attendance_marked_by_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,

  -- metode absen
  user_class_session_attendance_method VARCHAR(16),
  CONSTRAINT chk_ucsa_method
    CHECK (
      user_class_session_attendance_method IS NULL
      OR user_class_session_attendance_method IN ('manual','qr','geo','import','api','self')
    ),

  -- geolocation (opsional; utk self/geo)
  user_class_session_attendance_lat        DOUBLE PRECISION,
  user_class_session_attendance_lng        DOUBLE PRECISION,
  user_class_session_attendance_distance_m INTEGER CHECK (user_class_session_attendance_distance_m IS NULL OR user_class_session_attendance_distance_m >= 0),

  -- keterlambatan (detik)
  user_class_session_attendance_late_seconds INT
    CHECK (user_class_session_attendance_late_seconds IS NULL OR user_class_session_attendance_late_seconds >= 0),

  -- catatan tambahan
  user_class_session_attendance_user_note    TEXT,
  user_class_session_attendance_teacher_note TEXT,

  -- locking (opsional)
  user_class_session_attendance_locked_at TIMESTAMPTZ,

  -- audit
  user_class_session_attendance_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_session_attendance_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_session_attendance_deleted_at TIMESTAMPTZ
);

-- =========================================
-- INDEXES USER_CLASS_SESSION_ATTENDANCES
-- =========================================

-- unik aktif per (masjid, session, student)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsa_alive
  ON user_class_session_attendances (
    user_class_session_attendance_masjid_id,
    user_class_session_attendance_session_id,
    user_class_session_attendance_masjid_student_id
  )
  WHERE user_class_session_attendance_deleted_at IS NULL;

-- jalur query umum
CREATE INDEX IF NOT EXISTS idx_ucsa_session
  ON user_class_session_attendances (user_class_session_attendance_session_id)
  WHERE user_class_session_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsa_student
  ON user_class_session_attendances (user_class_session_attendance_masjid_student_id)
  WHERE user_class_session_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsa_status
  ON user_class_session_attendances (user_class_session_attendance_status)
  WHERE user_class_session_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ucsa_type_id
  ON user_class_session_attendances (user_class_session_attendance_type_id)
  WHERE user_class_session_attendance_deleted_at IS NULL;

-- rekap cepat per sesi+status
CREATE INDEX IF NOT EXISTS idx_ucsa_session_status
  ON user_class_session_attendances (user_class_session_attendance_session_id, user_class_session_attendance_status)
  WHERE user_class_session_attendance_deleted_at IS NULL;

-- time-series
CREATE INDEX IF NOT EXISTS brin_ucsa_created_at
  ON user_class_session_attendances USING BRIN (user_class_session_attendance_created_at);

CREATE INDEX IF NOT EXISTS brin_ucsa_marked_at
  ON user_class_session_attendances USING BRIN (user_class_session_attendance_marked_at);

-- teks
CREATE INDEX IF NOT EXISTS gin_ucsa_desc_trgm
  ON user_class_session_attendances USING GIN (user_class_session_attendance_desc gin_trgm_ops)
  WHERE user_class_session_attendance_deleted_at IS NULL;




-- =========================================
-- C) USER_CLASS_SESSION_ATTENDANCE_URLS (lampiran/url per attendance)
-- =========================================
CREATE TABLE IF NOT EXISTS user_class_session_attendance_urls (
  user_class_session_attendance_url_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  user_class_session_attendance_url_masjid_id   UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_class_session_attendance_url_attendance_id UUID NOT NULL
    REFERENCES user_class_session_attendances(user_class_session_attendance_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- (opsional) tipe media eksternal (lookup table)
  user_class_session_attendance_type_id UUID
    REFERENCES user_class_session_attendance_types(user_class_session_attendance_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Jenis/peran aset (mis. 'image','video','attachment','link','audio', dll.)
  user_class_session_attendance_url_kind VARCHAR(24) NOT NULL,

  -- Lokasi file/link
  user_class_session_attendance_url_href           TEXT,        -- URL publik
  user_class_session_attendance_url_object_key     TEXT,        -- object key aktif di storage
  user_class_session_attendance_url_object_key_old TEXT,        -- object key lama (retensi in-place replace)

  -- Metadata tampilan
  user_class_session_attendance_url_label      VARCHAR(160),
  user_class_session_attendance_url_order      INT NOT NULL DEFAULT 0,
  user_class_session_attendance_url_is_primary BOOLEAN NOT NULL DEFAULT FALSE,

  -- Housekeeping (retensi/purge)
  user_class_session_attendance_url_trash_url            TEXT,
  user_class_session_attendance_url_delete_pending_until TIMESTAMPTZ,

  -- Uploader (opsional)
  user_class_session_attendance_url_uploader_teacher_id  UUID REFERENCES masjid_teachers(masjid_teacher_id), -- FK bisa ditambah terpisah bila perlu
  user_class_session_attendance_url_uploader_student_id  UUID
    REFERENCES masjid_students(masjid_student_id) ON DELETE SET NULL,

  -- Audit
  user_class_session_attendance_url_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_session_attendance_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_class_session_attendance_url_deleted_at TIMESTAMPTZ
);

-- Lookup per attendance (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_ucsaurl_by_owner_live
  ON user_class_session_attendance_urls (
    user_class_session_attendance_url_attendance_id,
    user_class_session_attendance_url_kind,
    user_class_session_attendance_url_is_primary DESC,
    user_class_session_attendance_url_order,
    user_class_session_attendance_url_created_at
  )
  WHERE user_class_session_attendance_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_ucsaurl_by_masjid_live
  ON user_class_session_attendance_urls (user_class_session_attendance_url_masjid_id)
  WHERE user_class_session_attendance_url_deleted_at IS NULL;

-- Satu primary per (attendance, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsaurl_primary_per_kind_alive
  ON user_class_session_attendance_urls (user_class_session_attendance_url_attendance_id, user_class_session_attendance_url_kind)
  WHERE user_class_session_attendance_url_deleted_at IS NULL
    AND user_class_session_attendance_url_is_primary = TRUE;

-- Anti-duplikat href per attendance (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_ucsaurl_attendance_href_alive
  ON user_class_session_attendance_urls (user_class_session_attendance_url_attendance_id, LOWER(user_class_session_attendance_url_href))
  WHERE user_class_session_attendance_url_deleted_at IS NULL
    AND user_class_session_attendance_url_href IS NOT NULL;

-- Kandidat purge:
CREATE INDEX IF NOT EXISTS ix_ucsaurl_purge_due
  ON user_class_session_attendance_urls (user_class_session_attendance_url_delete_pending_until)
  WHERE user_class_session_attendance_url_delete_pending_until IS NOT NULL
    AND (
      (user_class_session_attendance_url_deleted_at IS NULL  AND user_class_session_attendance_url_object_key_old IS NOT NULL) OR
      (user_class_session_attendance_url_deleted_at IS NOT NULL AND user_class_session_attendance_url_object_key     IS NOT NULL)
    );

-- Uploader lookups (live only)
CREATE INDEX IF NOT EXISTS ix_ucsaurl_uploader_teacher_live
  ON user_class_session_attendance_urls (user_class_session_attendance_url_uploader_teacher_id)
  WHERE user_class_session_attendance_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ucsaurl_uploader_student_live
  ON user_class_session_attendance_urls (user_class_session_attendance_url_uploader_student_id)
  WHERE user_class_session_attendance_url_deleted_at IS NULL;

-- Time-scan (arsip/waktu)
CREATE INDEX IF NOT EXISTS brin_ucsaurl_created_at
  ON user_class_session_attendance_urls USING BRIN (user_class_session_attendance_url_created_at);

-- (opsional) pencarian label cepat (live only)
-- CREATE INDEX IF NOT EXISTS gin_ucsaurl_label_trgm_live
--   ON user_class_session_attendance_urls USING GIN (user_class_session_attendance_url_label gin_trgm_ops)
--   WHERE user_class_session_attendance_url_deleted_at IS NULL;

COMMIT;
