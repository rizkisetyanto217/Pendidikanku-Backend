-- +migrate Up
-- =========================================
-- UP Migration — Attendance Types & User Attendance
-- =========================================
BEGIN;

-- =========================================
-- EXTENSIONS
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE, search)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- kombinasi index opsional

-- =========================================
-- A) USER_ATTENDANCE_TYPE (master jenis attendance per masjid)
-- =========================================
CREATE TABLE IF NOT EXISTS user_attendance_type (
  user_attendance_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant guard
  user_attendance_type_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- data utama
  user_attendance_type_code  VARCHAR(32)  NOT NULL,  -- ex: SETORAN, MURAJAAH, TILAWAH
  user_attendance_type_label VARCHAR(80),
  user_attendance_type_slug  VARCHAR(120),
  user_attendance_type_color VARCHAR(20),
  user_attendance_type_desc  TEXT,

  -- status
  user_attendance_type_is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- audit
  user_attendance_type_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_type_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_type_deleted_at TIMESTAMPTZ
);

-- Unik per masjid + code (case-insensitive, soft-delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS uq_uatt_code_per_masjid_alive
  ON user_attendance_type (user_attendance_type_masjid_id, UPPER(user_attendance_type_code))
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

-- BRIN untuk time-series
CREATE INDEX IF NOT EXISTS brin_uatt_created_at
  ON user_attendance_type USING BRIN (user_attendance_type_created_at);




-- =========================================
-- B) USER_ATTENDANCE (per siswa per sesi)
-- =========================================
CREATE TABLE IF NOT EXISTS user_attendance (
  user_attendance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant & relasi utama
  user_attendance_masjid_id  UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_attendance_session_id UUID NOT NULL
    REFERENCES class_attendance_sessions(class_attendance_sessions_id)
    ON UPDATE CASCADE ON DELETE CASCADE,
  user_attendance_masjid_student_id UUID NOT NULL
    REFERENCES masjid_students(masjid_student_id) ON DELETE CASCADE,

  -- status kehadiran
  user_attendance_status VARCHAR(16) NOT NULL DEFAULT 'unmarked'
    CHECK (user_attendance_status IN ('unmarked','present','absent','excused','late')),

  -- jenis kegiatan/tipe absensi (opsional)
  user_attendance_type_id UUID
    REFERENCES user_attendance_type(user_attendance_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- kualitas/penilaian harian (opsional)
  user_attendance_desc      TEXT,
  user_attendance_score     NUMERIC(5,2)
    CHECK (user_attendance_score IS NULL OR user_attendance_score BETWEEN 0 AND 100),
  user_attendance_is_passed BOOLEAN,

  -- meta penandaan
  user_attendance_marked_at TIMESTAMPTZ,
  user_attendance_marked_by_teacher_id UUID
    REFERENCES masjid_teachers(masjid_teacher_id) ON DELETE SET NULL,

  -- metode absen
  user_attendance_method VARCHAR(16),
  CONSTRAINT chk_user_attendance_method
    CHECK (
      user_attendance_method IS NULL
      OR user_attendance_method IN ('manual','qr','geo','import','api','self')
    ),

  -- geolocation (opsional; utk self/geo)
  user_attendance_lat        DOUBLE PRECISION,
  user_attendance_lng        DOUBLE PRECISION,
  user_attendance_distance_m INTEGER CHECK (user_attendance_distance_m IS NULL OR user_attendance_distance_m >= 0),

  -- keterlambatan (detik)
  user_attendance_late_seconds INT
    CHECK (user_attendance_late_seconds IS NULL OR user_attendance_late_seconds >= 0),

  -- catatan tambahan
  user_attendance_user_note    TEXT,
  user_attendance_teacher_note TEXT,

  -- locking (opsional)
  user_attendance_locked_at TIMESTAMPTZ,

  -- audit
  user_attendance_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_deleted_at TIMESTAMPTZ
);

-- =========================================
-- INDEXES USER_ATTENDANCE
-- =========================================

-- unik aktif per (masjid, session, student)
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_attendance_alive
  ON user_attendance (user_attendance_masjid_id, user_attendance_session_id, user_attendance_masjid_student_id)
  WHERE user_attendance_deleted_at IS NULL;

-- jalur query umum
CREATE INDEX IF NOT EXISTS idx_user_attendance_session
  ON user_attendance (user_attendance_session_id)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_student
  ON user_attendance (user_attendance_masjid_student_id)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_status
  ON user_attendance (user_attendance_status)
  WHERE user_attendance_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_attendance_type_id
  ON user_attendance (user_attendance_type_id)
  WHERE user_attendance_deleted_at IS NULL;

-- rekap cepat per sesi+status
CREATE INDEX IF NOT EXISTS idx_user_attendance_session_status
  ON user_attendance (user_attendance_session_id, user_attendance_status)
  WHERE user_attendance_deleted_at IS NULL;

-- time-series
CREATE INDEX IF NOT EXISTS brin_user_attendance_created_at
  ON user_attendance USING BRIN (user_attendance_created_at);

CREATE INDEX IF NOT EXISTS brin_user_attendance_marked_at
  ON user_attendance USING BRIN (user_attendance_marked_at);

-- teks
CREATE INDEX IF NOT EXISTS gin_user_attendance_desc_trgm
  ON user_attendance USING GIN (user_attendance_desc gin_trgm_ops)
  WHERE user_attendance_deleted_at IS NULL;

COMMIT;

-- =========================================================
-- USER_ATTENDANCE_URLS — selaras dengan pola announcement_urls
--  - tambah: kind, object_key, object_key_old, mime
--  - tambah: order, is_primary
--  - pertahankan: uploader fields, tenant guard via FK
-- =========================================================
CREATE TABLE IF NOT EXISTS user_attendance_urls (
  user_attendance_url_id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  user_attendance_url_masjid_id            UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  user_attendance_url_attendance_id        UUID NOT NULL
    REFERENCES user_attendance(user_attendance_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- (opsional) tipe media eksternal (lookup table)
  user_attendance_type_id                  UUID
    REFERENCES public.user_attendance_type(user_attendance_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- Jenis/peran aset (mis. 'image','video','attachment','link','audio', dll.)
  user_attendance_url_kind                 VARCHAR(24) NOT NULL,

  -- Lokasi file/link
  user_attendance_url_href                 TEXT,        -- URL publik (boleh NULL jika pakai object storage)
  user_attendance_url_object_key           TEXT,        -- object key aktif di storage
  user_attendance_url_object_key_old       TEXT,        -- object key lama (retensi in-place replace)

  -- Metadata tampilan
  user_attendance_url_label                VARCHAR(160),
  user_attendance_url_order                INT NOT NULL DEFAULT 0,
  user_attendance_url_is_primary           BOOLEAN NOT NULL DEFAULT FALSE,

  -- Housekeeping (retensi/purge)
  user_attendance_url_trash_url            TEXT,
  user_attendance_url_delete_pending_until TIMESTAMPTZ,

  -- Uploader (opsional)
  user_attendance_url_uploader_teacher_id  UUID, -- FK ke masjid_teachers bisa ditambah di skrip terpisah jika diperlukan
  user_attendance_url_uploader_student_id  UUID
    REFERENCES masjid_students(masjid_student_id) ON DELETE SET NULL,

  -- Audit
  user_attendance_url_created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_url_updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_attendance_url_deleted_at           TIMESTAMPTZ
);

-- =========================================================
-- INDEXING / OPTIMIZATION (paritas dg announcement_urls)
-- =========================================================

-- Lookup per attendance (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_uau_by_owner_live
  ON user_attendance_urls (
    user_attendance_url_attendance_id,
    user_attendance_url_kind,
    user_attendance_url_is_primary DESC,
    user_attendance_url_order,
    user_attendance_url_created_at
  )
  WHERE user_attendance_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_uau_by_masjid_live
  ON user_attendance_urls (user_attendance_url_masjid_id)
  WHERE user_attendance_url_deleted_at IS NULL;

-- Satu primary per (attendance, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_uau_primary_per_kind_alive
  ON user_attendance_urls (user_attendance_url_attendance_id, user_attendance_url_kind)
  WHERE user_attendance_url_deleted_at IS NULL
    AND user_attendance_url_is_primary = TRUE;

-- Anti-duplikat href per attendance (live only) — opsional
CREATE UNIQUE INDEX IF NOT EXISTS uq_uau_attendance_href_alive
  ON user_attendance_urls (user_attendance_url_attendance_id, LOWER(user_attendance_url_href))
  WHERE user_attendance_url_deleted_at IS NULL
    AND user_attendance_url_href IS NOT NULL;

-- Kandidat purge:
--  - baris AKTIF dengan object_key_old (in-place replace)
--  - baris SOFT-DELETED dengan object_key (versi-per-baris)
CREATE INDEX IF NOT EXISTS ix_uau_purge_due
  ON user_attendance_urls (user_attendance_url_delete_pending_until)
  WHERE user_attendance_url_delete_pending_until IS NOT NULL
    AND (
      (user_attendance_url_deleted_at IS NULL  AND user_attendance_url_object_key_old IS NOT NULL) OR
      (user_attendance_url_deleted_at IS NOT NULL AND user_attendance_url_object_key     IS NOT NULL)
    );

-- Uploader lookups (live only)
CREATE INDEX IF NOT EXISTS ix_uau_uploader_teacher_live
  ON user_attendance_urls (user_attendance_url_uploader_teacher_id)
  WHERE user_attendance_url_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_uau_uploader_student_live
  ON user_attendance_urls (user_attendance_url_uploader_student_id)
  WHERE user_attendance_url_deleted_at IS NULL;

-- Time-scan (arsip/waktu)
CREATE INDEX IF NOT EXISTS brin_uau_created_at
  ON user_attendance_urls USING BRIN (user_attendance_url_created_at);

-- (opsional) pencarian label cepat (live only)
-- CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- CREATE INDEX IF NOT EXISTS gin_uau_label_trgm_live
--   ON user_attendance_urls USING GIN (user_attendance_url_label gin_trgm_ops)
--   WHERE user_attendance_url_deleted_at IS NULL;

COMMIT;
