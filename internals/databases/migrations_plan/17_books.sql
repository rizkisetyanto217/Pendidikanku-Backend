-- =========================================================
-- UP MIGRATION (Books & Class Subject Books - Final)
-- =========================================================

-- Prasyarat
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =========================================================
-- BOOKS
-- =========================================================
CREATE TABLE IF NOT EXISTS books (
  books_id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  books_masjid_id          UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- Identitas & deskripsi
  books_title              TEXT NOT NULL,
  books_subtitle           TEXT,
  books_authors            TEXT[],          -- multi-penulis
  books_contributors       JSONB,           -- editor, penerjemah, dsb.
  books_desc               TEXT,
  books_language           VARCHAR(20),     -- 'id', 'en', dst.
  books_locale             VARCHAR(20),     -- 'id-ID', 'en-US'
  books_slug               VARCHAR(160),

  -- Bibliografis
  books_publisher          TEXT,
  books_publication_year   SMALLINT,
  books_publication_date   DATE,
  books_edition            VARCHAR(40),
  books_series_name        TEXT,
  books_series_index       INT,
  books_volume             VARCHAR(20),

  -- Klasifikasi & kurikulum
  books_tags               TEXT[],
  books_categories         TEXT[],
  books_subject_codes      TEXT[],
  books_grade_levels       INT[],
  books_curriculum_notes   TEXT,

  -- Fisik & format
  books_format             TEXT,            -- 'paperback'|'hardcover'|'ebook'|'audio'
  books_page_count         INT,

  -- Sirkulasi lanjutan
  books_table_of_contents  JSONB,
  books_loan_policy        JSONB,           -- {max_days, renewals, penalty_cents}
  books_max_loan_days      INT,
  books_late_fee_cents     BIGINT,

  -- Visibilitas & status
  books_status             TEXT,            -- 'active'|'archived'|'out_of_print'
  books_visibility_scope   TEXT,
  books_is_active          BOOLEAN NOT NULL DEFAULT TRUE,

  -- Audit & concurrency
  books_created_by_user_id UUID,
  books_updated_by_user_id UUID,
  books_row_version        INT DEFAULT 1,
  books_etag               TEXT,

  books_created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  books_updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  books_deleted_at         TIMESTAMPTZ
);

-- =========================================================
-- CLASS_SUBJECT_BOOKS
-- =========================================================
CREATE TABLE IF NOT EXISTS class_subject_books (
  class_subject_books_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  class_subject_books_masjid_id      UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  class_subject_books_class_subject_id UUID NOT NULL REFERENCES class_subjects(class_subjects_id) ON DELETE CASCADE,
  class_subject_books_book_id        UUID NOT NULL REFERENCES books(books_id) ON DELETE RESTRICT,

  -- Konteks akademik
  class_subject_books_term_id        UUID,
  class_subject_books_academic_year_id UUID,

  -- Penggunaan di kelas
  class_subject_books_is_active      BOOLEAN NOT NULL DEFAULT TRUE,
  class_subject_books_is_core        BOOLEAN DEFAULT FALSE,
  class_subject_books_required       BOOLEAN DEFAULT TRUE,
  class_subject_books_required_edition_min VARCHAR(40),
  class_subject_books_required_isbn  VARCHAR(20),
  class_subject_books_order_index    INT,
  class_subject_books_start_week     SMALLINT,
  class_subject_books_end_week       SMALLINT,
  class_subject_books_coverage_weeks SMALLINT,

  -- Periode
  class_subject_books_start_date     DATE,
  class_subject_books_end_date       DATE,


  -- Penilaian & bobot
  class_subject_books_assessment_weight NUMERIC(6,3),

  -- Kebutuhan & biaya estimasi
  class_subject_books_qty_per_student NUMERIC(10,2),
  class_subject_books_estimated_cost_cents BIGINT,
  class_subject_books_currency        VARCHAR(10),

  -- Kebijakan & visibilitas
  class_subject_books_usage_policy   TEXT,
  class_subject_books_visibility_scope TEXT,

  -- Info tambahan
  class_subject_books_desc           TEXT,
  class_subject_books_tags           TEXT[],
  class_subject_books_alignment      JSONB,
  class_subject_books_teacher_notes  TEXT,
  class_subject_books_external_ref   TEXT,
  class_subject_books_extra          JSONB,

  -- Audit & concurrency
  class_subject_books_created_by_user_id UUID,
  class_subject_books_updated_by_user_id UUID,
  class_subject_books_deleted_reason TEXT,
  class_subject_books_row_version    INT DEFAULT 1,
  class_subject_books_etag           TEXT,

  class_subject_books_created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_books_updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_subject_books_deleted_at     TIMESTAMPTZ
);