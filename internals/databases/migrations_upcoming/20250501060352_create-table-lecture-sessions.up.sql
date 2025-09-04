-- Extensions yang dipakai index/UUID
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- trigram index untuk ILIKE/fuzzy

-- =========================================================
-- LECTURE SESSIONS
-- =========================================================
CREATE TABLE IF NOT EXISTS lecture_sessions (
  lecture_session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Info
  lecture_session_title        VARCHAR(255) NOT NULL,
  lecture_session_description  TEXT,

  -- Slug untuk URL (unik case-insensitive via index; JANGAN pakai UNIQUE di kolom)
  lecture_session_slug         VARCHAR(255) NOT NULL,

  -- Pengajar
  lecture_session_teacher_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  lecture_session_teacher_name VARCHAR(255),

  -- Jadwal (TIMESTAMPTZ)
  lecture_session_start_time   TIMESTAMPTZ NOT NULL,
  lecture_session_end_time     TIMESTAMPTZ NOT NULL,

  -- Lokasi & Gambar
  lecture_session_place        TEXT,
  lecture_session_image_url    TEXT,

  -- Relasi & cache masjid
  lecture_session_lecture_id   UUID REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  lecture_session_masjid_id    UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Validasi
  lecture_session_approved_by_admin_id   UUID REFERENCES users(id),
  lecture_session_approved_by_admin_at   TIMESTAMPTZ,
  lecture_session_approved_by_author_id  UUID REFERENCES users(id),
  lecture_session_approved_by_author_at  TIMESTAMPTZ,
  lecture_session_approved_by_teacher_id UUID REFERENCES users(id),
  lecture_session_approved_by_teacher_at TIMESTAMPTZ,
  lecture_session_approved_by_dkm_at     TIMESTAMPTZ,

  -- Publikasi
  lecture_session_is_active    BOOLEAN NOT NULL DEFAULT FALSE,

  -- Metadata (TIMESTAMPTZ tanpa TZ)
  lecture_session_created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_session_updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lecture_session_deleted_at   TIMESTAMPTZ NULL,

  -- Validasi waktu
  CONSTRAINT ck_ls_time_order CHECK (lecture_session_start_time <= lecture_session_end_time),

  -- Kolom FTS (judul + deskripsi)
  lecture_session_search tsvector GENERATED ALWAYS AS (
    to_tsvector('simple',
      coalesce(lecture_session_title,'') || ' ' || coalesce(lecture_session_description,'')
    )
  ) STORED
);

-- Slug unik case-insensitive, abaikan yang soft-deleted
CREATE UNIQUE INDEX IF NOT EXISTS uq_ls_slug_ci
  ON lecture_sessions (LOWER(lecture_session_slug))
  WHERE lecture_session_deleted_at IS NULL;

-- Listing upcoming per masjid (aktif)
CREATE INDEX IF NOT EXISTS idx_ls_masjid_active_start
  ON lecture_sessions (lecture_session_masjid_id, lecture_session_is_active, lecture_session_start_time)
  WHERE lecture_session_deleted_at IS NULL;

-- Jadwal per teacher
CREATE INDEX IF NOT EXISTS idx_ls_teacher_start
  ON lecture_sessions (lecture_session_teacher_id, lecture_session_start_time)
  WHERE lecture_session_deleted_at IS NULL;

-- Per lecture utama
CREATE INDEX IF NOT EXISTS idx_ls_lecture_start
  ON lecture_sessions (lecture_session_lecture_id, lecture_session_start_time)
  WHERE lecture_session_deleted_at IS NULL;

-- Sesi aktif terbaru per masjid (opsional)
CREATE INDEX IF NOT EXISTS idx_ls_masjid_active_created_desc
  ON lecture_sessions (lecture_session_masjid_id, lecture_session_is_active, lecture_session_created_at DESC)
  WHERE lecture_session_deleted_at IS NULL;

-- Full-Text Search
CREATE INDEX IF NOT EXISTS idx_ls_search_fts
  ON lecture_sessions USING GIN (lecture_session_search);

-- Trigram untuk ILIKE/fuzzy pada title & slug
CREATE INDEX IF NOT EXISTS idx_ls_title_trgm
  ON lecture_sessions USING GIN (lecture_session_title gin_trgm_ops)
  WHERE lecture_session_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ls_slug_trgm
  ON lecture_sessions USING GIN (lecture_session_slug gin_trgm_ops)
  WHERE lecture_session_deleted_at IS NULL;

-- (Opsional) akses cepat ke rentang waktu
CREATE INDEX IF NOT EXISTS idx_ls_start_time
  ON lecture_sessions (lecture_session_start_time)
  WHERE lecture_session_deleted_at IS NULL;

-- =========================================================
-- USER LECTURE SESSIONS
-- =========================================================
CREATE TABLE IF NOT EXISTS user_lecture_sessions (
  user_lecture_session_id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Nilai/hasil
  user_lecture_session_grade_result       NUMERIC(5,2),

  -- Relasi
  user_lecture_session_lecture_session_id UUID NOT NULL REFERENCES lecture_sessions(lecture_session_id) ON DELETE CASCADE,
  user_lecture_session_user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_lecture_session_lecture_id         UUID NOT NULL REFERENCES lectures(lecture_id) ON DELETE CASCADE,

  -- Masjid cache
  user_lecture_session_masjid_id          UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Waktu (TIMESTAMPTZ)
  user_lecture_session_created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_lecture_session_updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_lecture_session_deleted_at         TIMESTAMPTZ,

  -- Anti duplikasi: 1 user / 1 session
  CONSTRAINT uq_uls_user_per_session UNIQUE (user_lecture_session_user_id, user_lecture_session_lecture_session_id),

  -- Validasi skor (opsional 0..100)
  CONSTRAINT ck_uls_grade_range CHECK (
    user_lecture_session_grade_result IS NULL
    OR (user_lecture_session_grade_result >= 0 AND user_lecture_session_grade_result <= 100)
  )
);

-- Indexes
-- Peserta per session (alive only)
CREATE INDEX IF NOT EXISTS idx_uls_by_session_alive
  ON user_lecture_sessions (user_lecture_session_lecture_session_id)
  WHERE user_lecture_session_deleted_at IS NULL;

-- Partisipasi user per masjid terbaru (alive only)
CREATE INDEX IF NOT EXISTS idx_uls_user_masjid_created_desc_alive
  ON user_lecture_sessions (user_lecture_session_user_id, user_lecture_session_masjid_id, user_lecture_session_created_at DESC)
  WHERE user_lecture_session_deleted_at IS NULL;

-- Analitik nilai per lecture (alive only)
CREATE INDEX IF NOT EXISTS idx_uls_lecture_for_grade_alive
  ON user_lecture_sessions (user_lecture_session_lecture_id)
  WHERE user_lecture_session_deleted_at IS NULL;
