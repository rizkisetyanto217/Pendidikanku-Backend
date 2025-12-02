-- +migrate Up
/* =====================================================================
   MATERIALS
   - Enums:
       * material_type_enum
       * material_importance_enum
       * material_progress_status_enum
   - Table: class_materials
   - Table: school_materials
   - Table: student_class_material_progresses
   ===================================================================== */

BEGIN;

-- ---------------------------------------------------------------------
-- ENUMS (idempotent, shared)
-- ---------------------------------------------------------------------
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'material_type_enum') THEN
    CREATE TYPE material_type_enum AS ENUM (
      'article',      -- artikel / rich text internal
      'doc',          -- dokumen teks: doc/docx/txt/rtf
      'ppt',          -- slide presentasi: ppt/pptx
      'pdf',          -- file PDF
      'image',        -- gambar: jpg/png/webp, dll.
      'youtube',      -- video YouTube
      'video_file',   -- video upload sendiri
      'link',         -- link eksternal
      'embed'         -- embed iframe / lainnya
    );
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'material_importance_enum') THEN
    CREATE TYPE material_importance_enum AS ENUM (
      'important',    -- core / utama (lebih strict)
      'additional',   -- pendukung
      'optional'      -- bonus
    );
  END IF;
END$$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'material_progress_status_enum') THEN
    CREATE TYPE material_progress_status_enum AS ENUM (
      'not_started',  -- disiapkan kalau mau pre-generate row
      'in_progress',  -- sedang diakses / sebagian
      'completed',    -- sudah selesai sesuai definisi sekolah
      'skipped'       -- dilewati (opsional dipakai)
    );
  END IF;
END$$;

/* =====================================================================
   TABLE: class_materials
   Materi yang dipakai di level CSST (kelas x subject x guru)
   Inilah yang dilihat murid & dilacak progress-nya
   ===================================================================== */
CREATE TABLE IF NOT EXISTS class_materials (
  class_material_id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- scope / tenant + CSST
  class_material_school_id                  UUID NOT NULL,
  class_material_csst_id                    UUID NOT NULL, -- FK ke class_section_subject_teachers

  -- creator (opsional)
  class_material_created_by_user_id         UUID,

  -- konten utama
  class_material_title                      TEXT NOT NULL,
  class_material_description                TEXT,
  class_material_type                       material_type_enum NOT NULL,

  -- artikel (rich text)
  class_material_content_html               TEXT,

  -- file upload (pdf/doc/video/gambar dll.)
  class_material_file_url                   TEXT,
  class_material_file_name                  TEXT,
  class_material_file_mime_type             TEXT,
  class_material_file_size_bytes            BIGINT,

  -- link / embed / YouTube
  class_material_external_url               TEXT,
  class_material_youtube_id                 TEXT,
  class_material_duration_sec               INT,   -- total detik (video, opsional)

  -- tier kepentingan (English)
  class_material_importance                 material_importance_enum NOT NULL DEFAULT 'important',
  class_material_is_required_for_pass       BOOLEAN NOT NULL DEFAULT FALSE,  -- disiapkan utk gating
  class_material_affects_scoring            BOOLEAN NOT NULL DEFAULT FALSE,  -- disiapkan utk nilai engagement

  -- struktur pertemuan & sesi
  class_material_meeting_number             INT,      -- pertemuan ke-1,2,3,... (opsional)
  class_material_session_id                 UUID,     -- optional FK ke class_attendance_sessions

  -- sumber materi
  -- 'school'  = hasil generate dari school_materials
  -- 'teacher' = dibuat langsung oleh guru
  class_material_source_kind                TEXT,
  class_material_source_school_material_id  UUID,     -- referensi ke school_materials (jika dari template)

  -- urutan & status
  class_material_order                      INT,      -- urutan dalam satu CSST
  class_material_is_active                  BOOLEAN NOT NULL DEFAULT TRUE,
  class_material_is_published               BOOLEAN NOT NULL DEFAULT FALSE,
  class_material_published_at               TIMESTAMPTZ,

  -- soft delete
  class_material_deleted                    BOOLEAN NOT NULL DEFAULT FALSE,
  class_material_deleted_at                 TIMESTAMPTZ,

  -- timestamps
  class_material_created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
  class_material_updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- tenant-safe pair
  UNIQUE (class_material_id, class_material_school_id),

  -- FK KOMPOSIT TENANT-SAFE
  CONSTRAINT fk_class_material_school_same_school
    FOREIGN KEY (class_material_school_id)
    REFERENCES schools (school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_class_material_csst_same_school
    FOREIGN KEY (class_material_csst_id, class_material_school_id)
    REFERENCES class_section_subject_teachers (csst_id, csst_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  CONSTRAINT fk_class_material_session
    FOREIGN KEY (class_material_session_id)
    REFERENCES class_attendance_sessions (session_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- =====================================================================
-- INDEXES: class_materials
-- =====================================================================

-- query by tenant + csst
CREATE INDEX IF NOT EXISTS idx_class_material_school_csst
  ON class_materials (class_material_school_id, class_material_csst_id)
  WHERE NOT class_material_deleted;

-- published & active
CREATE INDEX IF NOT EXISTS idx_class_material_school_published
  ON class_materials (class_material_school_id, class_material_is_published, class_material_is_active)
  WHERE NOT class_material_deleted;

-- urutan dalam CSST
CREATE INDEX IF NOT EXISTS idx_class_material_order
  ON class_materials (class_material_csst_id, class_material_order)
  WHERE NOT class_material_deleted;

-- importance tier
CREATE INDEX IF NOT EXISTS idx_class_material_importance
  ON class_materials (class_material_school_id, class_material_importance)
  WHERE NOT class_material_deleted;

-- pertemuan ke berapa
CREATE INDEX IF NOT EXISTS idx_class_material_csst_meeting
  ON class_materials (class_material_csst_id, class_material_meeting_number)
  WHERE NOT class_material_deleted;

-- sumber materi
CREATE INDEX IF NOT EXISTS idx_class_material_source_kind
  ON class_materials (class_material_source_kind)
  WHERE NOT class_material_deleted;

CREATE INDEX IF NOT EXISTS idx_class_material_source_school_material
  ON class_materials (class_material_source_school_material_id)
  WHERE NOT class_material_deleted;


/* =====================================================================
   TABLE: school_materials
   Master materi di level sekolah (template)
   Bisa di-generate jadi class_materials untuk tiap CSST
   ===================================================================== */
CREATE TABLE IF NOT EXISTS school_materials (
  school_material_id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- scope / tenant
  school_material_school_id                UUID NOT NULL,

  -- subject / mapel di level sekolah (template anchor)
  school_material_class_subject_id         UUID,  -- FK ke class_subjects (atau school_subjects)

  -- creator (opsional)
  school_material_created_by_user_id       UUID,

  -- konten utama
  school_material_title                    TEXT NOT NULL,
  school_material_description              TEXT,
  school_material_type                     material_type_enum NOT NULL,

  -- artikel (rich text)
  school_material_content_html             TEXT,

  -- file upload (pdf/doc/video/gambar dll.)
  school_material_file_url                 TEXT,
  school_material_file_name                TEXT,
  school_material_file_mime_type           TEXT,
  school_material_file_size_bytes          BIGINT,

  -- link / embed / YouTube
  school_material_external_url             TEXT,
  school_material_youtube_id               TEXT,
  school_material_duration_sec             INT,  -- total detik (video, opsional)

  -- tier kepentingan (English, sama dengan class_materials)
  school_material_importance               material_importance_enum NOT NULL DEFAULT 'important',
  school_material_is_required_for_pass     BOOLEAN NOT NULL DEFAULT FALSE,
  school_material_affects_scoring          BOOLEAN NOT NULL DEFAULT FALSE,

  -- struktur kurikulum / pertemuan (hint untuk generate ke CSST)
  school_material_meeting_number           INT,   -- pertemuan ke-1,2,3,... (opsional)
  school_material_default_order            INT,   -- urutan default dalam satu CSST

  -- metadata scope (opsional, bisa dipakai untuk filter tambahan)
  -- misal: "grade_7", "grade_8", "tahfidz", "fiqih", dst.
  school_material_scope_tag                TEXT,

  -- status & publikasi
  school_material_is_active                BOOLEAN NOT NULL DEFAULT TRUE,
  school_material_is_published             BOOLEAN NOT NULL DEFAULT FALSE,
  school_material_published_at             TIMESTAMPTZ,

  -- soft delete
  school_material_deleted                  BOOLEAN NOT NULL DEFAULT FALSE,
  school_material_deleted_at               TIMESTAMPTZ,

  -- timestamps
  school_material_created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
  school_material_updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- tenant-safe pair
  UNIQUE (school_material_id, school_material_school_id),

  -- FK tenant-safe
  CONSTRAINT fk_school_material_school_same_school
    FOREIGN KEY (school_material_school_id)
    REFERENCES schools (school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- NOTE: sesuaikan nama tabel & kolom dengan schema-mu
  -- misal: class_subjects (class_subject_id, class_subject_school_id)
  CONSTRAINT fk_school_material_class_subject_same_school
    FOREIGN KEY (school_material_class_subject_id, school_material_school_id)
    REFERENCES class_subjects (class_subject_id, class_subject_school_id)
    ON UPDATE CASCADE ON DELETE SET NULL
);

-- =====================================================================
-- INDEXES: school_materials
-- =====================================================================

-- query by tenant + status publikasi
CREATE INDEX IF NOT EXISTS idx_school_material_school_published
  ON school_materials (school_material_school_id, school_material_is_published, school_material_is_active)
  WHERE NOT school_material_deleted;

-- importance tier
CREATE INDEX IF NOT EXISTS idx_school_material_importance
  ON school_materials (school_material_school_id, school_material_importance)
  WHERE NOT school_material_deleted;

-- meeting number & default order (buat generate urutan ke CSST)
CREATE INDEX IF NOT EXISTS idx_school_material_meeting_order
  ON school_materials (
    school_material_school_id,
    school_material_meeting_number,
    school_material_default_order
  )
  WHERE NOT school_material_deleted;

-- scope tag (filter berdasarkan kategori/jenjang)
CREATE INDEX IF NOT EXISTS idx_school_material_scope_tag
  ON school_materials (school_material_school_id, school_material_scope_tag)
  WHERE NOT school_material_deleted;

-- subject anchor (buat query: materi per subject)
CREATE INDEX IF NOT EXISTS idx_school_material_class_subject
  ON school_materials (school_material_school_id, school_material_class_subject_id)
  WHERE NOT school_material_deleted;


-- (Opsional tapi recommended): FK class_material → school_material (tenant-safe)
ALTER TABLE class_materials
  ADD CONSTRAINT fk_class_material_source_school_material_same_school
  FOREIGN KEY (class_material_source_school_material_id, class_material_school_id)
  REFERENCES school_materials (school_material_id, school_material_school_id)
  ON UPDATE CASCADE ON DELETE SET NULL;


/* =====================================================================
   TABLE: student_class_material_progresses
   Progress murid per materi di level CSST
   ===================================================================== */
CREATE TABLE IF NOT EXISTS student_class_material_progresses (
  student_class_material_progress_id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- scope / tenant
  student_class_material_progress_school_id               UUID NOT NULL,

  -- murid & enrollment (SCSST)
  student_class_material_progress_student_id              UUID NOT NULL, -- FK ke school_students
  student_class_material_progress_scsst_id                UUID NOT NULL, -- FK ke student_class_section_subject_teachers

  -- materi yang dilacak
  student_class_material_progress_class_material_id       UUID NOT NULL,

  -- status & progress (umum untuk article, video, pdf, dll.)
  student_class_material_progress_status                  material_progress_status_enum NOT NULL DEFAULT 'in_progress',
  student_class_material_progress_last_percent            INT NOT NULL DEFAULT 0,      -- 0..100
  student_class_material_progress_is_completed            BOOLEAN NOT NULL DEFAULT FALSE,

  -- timestamps aktivitas
  student_class_material_progress_first_started_at        TIMESTAMPTZ,
  student_class_material_progress_last_activity_at        TIMESTAMPTZ,
  student_class_material_progress_completed_at            TIMESTAMPTZ,

  -- metrik engagement (dipakai semua tipe: article, youtube, pdf, dll.)
  student_class_material_progress_view_duration_sec       BIGINT,                      -- total detik akses (approx)
  student_class_material_progress_open_count              INT NOT NULL DEFAULT 0,      -- berapa kali dibuka

  -- fleksibel: posisi terakhir video / scroll / halaman / dsb (per-type)
  student_class_material_progress_extra                   JSONB,

  -- timestamps row
  student_class_material_progress_created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  student_class_material_progress_updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- tenant-safe uniqueness: 1 row per (murid x scsst x materi)
  UNIQUE (
    student_class_material_progress_school_id,
    student_class_material_progress_scsst_id,
    student_class_material_progress_class_material_id
  ),

  -- ===================================================================
  -- FOREIGN KEYS (tenant-safe)
  -- ===================================================================

  -- School scope
  CONSTRAINT fk_student_class_material_progress_school_same_school
    FOREIGN KEY (student_class_material_progress_school_id)
    REFERENCES schools (school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Murid di school ini
  CONSTRAINT fk_student_class_material_progress_student_same_school
    FOREIGN KEY (student_class_material_progress_student_id, student_class_material_progress_school_id)
    REFERENCES school_students (student_id, student_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Enrollment SCSST (murid x CSST)
  CONSTRAINT fk_student_class_material_progress_scsst_same_school
    FOREIGN KEY (student_class_material_progress_scsst_id, student_class_material_progress_school_id)
    REFERENCES student_class_section_subject_teachers (scsst_id, scsst_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Materi yang diikuti murid ini
  CONSTRAINT fk_student_class_material_progress_class_material_same_school
    FOREIGN KEY (student_class_material_progress_class_material_id, student_class_material_progress_school_id)
    REFERENCES class_materials (class_material_id, class_material_school_id)
    ON UPDATE CASCADE ON DELETE CASCADE
);

-- =====================================================================
-- INDEXES: student_class_material_progresses
-- =====================================================================

-- by student (untuk dashboard murid: list materi + progress)
CREATE INDEX IF NOT EXISTS idx_student_class_material_progress_by_student
  ON student_class_material_progresses (
    student_class_material_progress_school_id,
    student_class_material_progress_student_id,
    student_class_material_progress_status
  );

-- by SCSST (untuk view kelas-mapel tertentu)
CREATE INDEX IF NOT EXISTS idx_student_class_material_progress_by_scsst
  ON student_class_material_progresses (
    student_class_material_progress_school_id,
    student_class_material_progress_scsst_id,
    student_class_material_progress_status
  );

-- by materi (untuk lihat siapa saja yang sudah / belum selesai materi ini)
CREATE INDEX IF NOT EXISTS idx_student_class_material_progress_by_class_material
  ON student_class_material_progresses (
    student_class_material_progress_school_id,
    student_class_material_progress_class_material_id,
    student_class_material_progress_status
  );

-- optimisasi query progress completion (misal gating)
CREATE INDEX IF NOT EXISTS idx_student_class_material_progress_completed
  ON student_class_material_progresses (
    student_class_material_progress_school_id,
    student_class_material_progress_class_material_id
  )
  WHERE student_class_material_progress_is_completed = TRUE;

COMMIT;
