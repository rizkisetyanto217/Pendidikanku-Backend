-- +migrate Up
-- =========================================
-- EXTENSIONS (idempotent)
-- =========================================
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- trigram (ILIKE search)
CREATE EXTENSION IF NOT EXISTS btree_gin;  -- opsional; untuk kombinasi tertentu

-- =========================================
-- ENUMS (idempotent)
-- =========================================
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'session_status_enum') THEN
    CREATE TYPE session_status_enum AS ENUM ('scheduled','ongoing','completed','canceled');
  END IF;
END$$;

-- =========================================
-- TABLE: class_attendance_session_types (master per tenant)
-- =========================================
CREATE TABLE IF NOT EXISTS class_attendance_session_types (
  class_attendance_session_type_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant
  class_attendance_session_type_school_id UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,

  -- identitas
  class_attendance_session_type_slug        VARCHAR(160) NOT NULL,
  class_attendance_session_type_name        TEXT NOT NULL,
  class_attendance_session_type_description TEXT,

  -- tampilan (opsional, utk UI)
  class_attendance_session_type_color       TEXT,  -- hex / tailwind token / bebas
  class_attendance_session_type_icon        TEXT,  -- nama icon di FE

  -- control
  class_attendance_session_type_is_active   BOOLEAN NOT NULL DEFAULT TRUE,
  class_attendance_session_type_sort_order  INT NOT NULL DEFAULT 0,

  -- audit
  class_attendance_session_type_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_type_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_type_deleted_at  TIMESTAMPTZ
);

-- Unik slug per tenant (alive only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_castype_school_slug_alive
  ON class_attendance_session_types (
    class_attendance_session_type_school_id,
    lower(class_attendance_session_type_slug)
  )
  WHERE class_attendance_session_type_deleted_at IS NULL;

-- Nama unik per tenant (opsional)
CREATE UNIQUE INDEX IF NOT EXISTS uq_castype_school_name_alive
  ON class_attendance_session_types (
    class_attendance_session_type_school_id,
    lower(class_attendance_session_type_name)
  )
  WHERE class_attendance_session_type_deleted_at IS NULL;

-- UNIQUE komposit utk FK tenant-safe (no WHERE!)
CREATE UNIQUE INDEX IF NOT EXISTS uq_castype_school_id_pair
  ON class_attendance_session_types (
    class_attendance_session_type_school_id,
    class_attendance_session_type_id
  );

-- Listing by tenant + aktif
CREATE INDEX IF NOT EXISTS idx_castype_school_sort_alive
  ON class_attendance_session_types (
    class_attendance_session_type_school_id,
    class_attendance_session_type_is_active,
    class_attendance_session_type_sort_order
  )
  WHERE class_attendance_session_type_deleted_at IS NULL;

-- BRIN utk scan besar
CREATE INDEX IF NOT EXISTS brin_castype_created_at
  ON class_attendance_session_types
  USING BRIN (class_attendance_session_type_created_at);



-- =========================================
-- TABLE: class_attendance_sessions (fresh create)
-- =========================================
CREATE TABLE IF NOT EXISTS class_attendance_sessions (
  class_attendance_session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- tenant
  class_attendance_session_school_id   UUID NOT NULL,

  -- relasi utama: schedule (header)
  class_attendance_session_schedule_id UUID,

  -- jejak rule asal occurrence (opsional)
  class_attendance_session_rule_id UUID
    REFERENCES class_schedule_rules(class_schedule_rule_id) ON DELETE SET NULL,

  -- SLUG (opsional; unik per tenant saat alive)
  class_attendance_session_slug VARCHAR(160),

  -- occurrence (tanggal + waktu aktual yang dipakai sesi)
  class_attendance_session_date      DATE NOT NULL,
  class_attendance_session_starts_at TIMESTAMPTZ,
  class_attendance_session_ends_at   TIMESTAMPTZ,

  -- lifecycle
  class_attendance_session_status            session_status_enum NOT NULL DEFAULT 'scheduled', -- scheduled|ongoing|completed|canceled
  class_attendance_session_attendance_status TEXT NOT NULL DEFAULT 'open',                     -- open|closed
  class_attendance_session_locked            BOOLEAN NOT NULL DEFAULT FALSE,

  -- overrides (ubah harian tanpa mengubah rules)
  class_attendance_session_is_override BOOLEAN NOT NULL DEFAULT FALSE,
  class_attendance_session_is_canceled BOOLEAN NOT NULL DEFAULT FALSE,
  class_attendance_session_original_start_at TIMESTAMPTZ,
  class_attendance_session_original_end_at   TIMESTAMPTZ,
  class_attendance_session_kind TEXT,
  class_attendance_session_override_reason TEXT,
  class_attendance_session_override_event_id UUID, -- utk relasi/cek/index override event

  -- override resources (referensi langsung; opsional)
  class_attendance_session_teacher_id    UUID REFERENCES school_teachers(school_teacher_id) ON DELETE SET NULL,
  class_attendance_session_class_room_id UUID REFERENCES class_rooms(class_room_id)         ON DELETE SET NULL,
  class_attendance_session_csst_id       UUID REFERENCES class_section_subject_teachers(class_section_subject_teacher_id) ON DELETE SET NULL,

  -- tipe sesi (master per tenant)
  class_attendance_session_type_id UUID,

  -- snapshot tipe sesi (jejak saat sesi dibuat)
  class_attendance_session_type_snapshot JSONB,

  -- info & rekap
  class_attendance_session_title         TEXT,
  class_attendance_session_general_info  TEXT NOT NULL DEFAULT '',
  class_attendance_session_note          TEXT,
  class_attendance_session_present_count INT,
  class_attendance_session_absent_count  INT,
  class_attendance_session_late_count    INT,
  class_attendance_session_excused_count INT,
  class_attendance_session_sick_count    INT,
  class_attendance_session_leave_count   INT,

  -- ===== Snapshot CSST (sumber data turunan) =====
  class_attendance_session_csst_snapshot JSONB,

  -- ===== Kolom turunan (GENERATED) dari snapshot CSST =====
  class_attendance_session_csst_id_snapshot    UUID   GENERATED ALWAYS AS ((class_attendance_session_csst_snapshot->>'csst_id')::uuid) STORED,
  class_attendance_session_subject_id_snapshot UUID   GENERATED ALWAYS AS ((class_attendance_session_csst_snapshot->>'subject_id')::uuid) STORED,
  class_attendance_session_section_id_snapshot UUID   GENERATED ALWAYS AS ((class_attendance_session_csst_snapshot->>'section_id')::uuid) STORED,
  class_attendance_session_teacher_id_snapshot UUID   GENERATED ALWAYS AS ((class_attendance_session_csst_snapshot->>'teacher_id')::uuid) STORED,
  class_attendance_session_room_id_snapshot    UUID   GENERATED ALWAYS AS ((class_attendance_session_csst_snapshot->>'room_id')::uuid) STORED,

  class_attendance_session_subject_code_snapshot  TEXT GENERATED ALWAYS AS (NULLIF(class_attendance_session_csst_snapshot->>'subject_code','')) STORED,
  class_attendance_session_subject_name_snapshot  TEXT GENERATED ALWAYS AS (NULLIF(class_attendance_session_csst_snapshot->>'subject_name','')) STORED,
  class_attendance_session_section_name_snapshot  TEXT GENERATED ALWAYS AS (NULLIF(class_attendance_session_csst_snapshot->>'section_name','')) STORED,
  class_attendance_session_teacher_name_snapshot  TEXT GENERATED ALWAYS AS (NULLIF(class_attendance_session_csst_snapshot->>'teacher_name','')) STORED,
  class_attendance_session_room_name_snapshot     TEXT GENERATED ALWAYS AS (NULLIF(class_attendance_session_csst_snapshot->>'room_name','')) STORED,

  class_attendance_session_display_title TEXT
    GENERATED ALWAYS AS (
      NULLIF(
        COALESCE(
          (class_attendance_session_csst_snapshot->>'subject_name'),
          (class_attendance_session_csst_snapshot->>'subject_code'),
          ''
        )
        || CASE WHEN (class_attendance_session_csst_snapshot->>'section_name') IS NOT NULL
           THEN ' — ' || (class_attendance_session_csst_snapshot->>'section_name') ELSE '' END
        || CASE WHEN (class_attendance_session_csst_snapshot->>'teacher_name') IS NOT NULL
           THEN ' (' || (class_attendance_session_csst_snapshot->>'teacher_name') || ')' ELSE '' END
      , '')
    ) STORED,

  -- ===== Snapshot RULE (jejak pola saat generate) =====
  class_attendance_session_rule_snapshot JSONB,

  -- Turunan (GENERATED) dari rule snapshot (untuk filter cepat)
  class_attendance_session_rule_day_of_week_snapshot INT
    GENERATED ALWAYS AS ((class_attendance_session_rule_snapshot->>'day_of_week')::INT) STORED,
  class_attendance_session_rule_start_time_snapshot TEXT
    GENERATED ALWAYS AS (NULLIF(class_attendance_session_rule_snapshot->>'start_time','')) STORED,
  class_attendance_session_rule_end_time_snapshot TEXT
    GENERATED ALWAYS AS (NULLIF(class_attendance_session_rule_snapshot->>'end_time','')) STORED,
  class_attendance_session_rule_week_parity_snapshot TEXT
    GENERATED ALWAYS AS (NULLIF(class_attendance_session_rule_snapshot->>'week_parity','')) STORED,

  -- audit
  class_attendance_session_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_deleted_at TIMESTAMPTZ,

  -- FK komposit tenant-safe → schedules
  CONSTRAINT fk_cas_schedule_tenant
    FOREIGN KEY (class_attendance_session_school_id, class_attendance_session_schedule_id)
    REFERENCES class_schedules (class_schedule_school_id, class_schedule_id)
    ON UPDATE CASCADE ON DELETE RESTRICT,

  -- FK komposit tenant-safe → session types
  CONSTRAINT fk_cas_type_tenant
    FOREIGN KEY (class_attendance_session_school_id, class_attendance_session_type_id)
    REFERENCES class_attendance_session_types (class_attendance_session_type_school_id, class_attendance_session_type_id)
    ON UPDATE CASCADE ON DELETE SET NULL,

  -- CHECKS
  CONSTRAINT chk_cas_time_order
    CHECK (
      class_attendance_session_starts_at IS NULL
      OR class_attendance_session_ends_at IS NULL
      OR class_attendance_session_ends_at >= class_attendance_session_starts_at
    ),
  CONSTRAINT chk_cas_time_order_original
    CHECK (
      class_attendance_session_original_start_at IS NULL
      OR class_attendance_session_original_end_at IS NULL
      OR class_attendance_session_original_end_at >= class_attendance_session_original_start_at
    ),
  CONSTRAINT chk_cas_attendance_status
    CHECK (class_attendance_session_attendance_status IN ('open','closed')),
  CONSTRAINT chk_cas_override_event_requires_flag
    CHECK (
      class_attendance_session_override_event_id IS NULL
      OR class_attendance_session_is_override = TRUE
    ),
  -- jika ada rule_id, sebaiknya snapshot rule juga ada (opsional tapi baik)
  CONSTRAINT chk_cas_rule_snapshot_when_rule
    CHECK (
      class_attendance_session_rule_id IS NULL
      OR class_attendance_session_rule_snapshot IS NOT NULL
    )
);

-- =========================================
-- INDEXES (idempotent)
-- =========================================

-- Pair unik id+tenant
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_id_tenant
  ON class_attendance_sessions (class_attendance_session_id, class_attendance_session_school_id);

-- SLUG unik per tenant (alive only, CI)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_slug_per_tenant_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    lower(class_attendance_session_slug)
  )
  WHERE class_attendance_session_deleted_at IS NULL
    AND class_attendance_session_slug IS NOT NULL;

-- pencarian slug cepat
CREATE INDEX IF NOT EXISTS gin_cas_slug_trgm_alive
  ON class_attendance_sessions USING GIN ((lower(class_attendance_session_slug)) gin_trgm_ops)
  WHERE class_attendance_session_deleted_at IS NULL
    AND class_attendance_session_slug IS NOT NULL;

-- Unik per (tenant, schedule, date) yg alive
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_school_schedule_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_schedule_id,
    class_attendance_session_date
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- Kalender per tenant
CREATE INDEX IF NOT EXISTS idx_cas_school_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- Per schedule
CREATE INDEX IF NOT EXISTS idx_cas_schedule_date_alive
  ON class_attendance_sessions (
    class_attendance_session_schedule_id,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- Lookup per guru (override direct)
CREATE INDEX IF NOT EXISTS idx_cas_teacher_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_teacher_id,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- Canceled / Override flags
CREATE INDEX IF NOT EXISTS idx_cas_canceled_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_is_canceled,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cas_override_date_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_is_override,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- Override event lookup
CREATE INDEX IF NOT EXISTS idx_cas_override_event_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_override_event_id
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- Join cepat ke rule
CREATE INDEX IF NOT EXISTS idx_cas_rule_alive
  ON class_attendance_sessions (class_attendance_session_rule_id)
  WHERE class_attendance_session_deleted_at IS NULL;

-- Unik per schedule+start (dedup occurrence by schedule)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_sched_start
  ON class_attendance_sessions (
    class_attendance_session_schedule_id,
    class_attendance_session_starts_at
  );

-- Idempotent materialize per (tenant, csst, date, start)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cas_tenant_csst_date_start_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_csst_id_snapshot,
    class_attendance_session_date,
    class_attendance_session_starts_at
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- BRIN untuk scan besar by created_at
CREATE INDEX IF NOT EXISTS brin_cas_created_at
  ON class_attendance_sessions USING BRIN (class_attendance_session_created_at);

-- Pencarian cepat judul untuk kalender
CREATE INDEX IF NOT EXISTS idx_cas_school_date_title_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_date DESC,
    class_attendance_session_display_title
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- Trigram pada display_title (case-insensitive)
CREATE INDEX IF NOT EXISTS gin_cas_display_title_trgm_alive
  ON class_attendance_sessions
  USING GIN ((lower(class_attendance_session_display_title)) gin_trgm_ops)
  WHERE class_attendance_session_deleted_at IS NULL
    AND class_attendance_session_display_title IS NOT NULL;

-- GIN untuk rule snapshot (filter by day_of_week, parity, dll)
CREATE INDEX IF NOT EXISTS gin_cas_rule_snapshot
  ON class_attendance_sessions
  USING GIN (class_attendance_session_rule_snapshot);

-- BTree untuk turunan rule (query umum: hari & jam)
CREATE INDEX IF NOT EXISTS idx_cas_rule_day_start_alive
  ON class_attendance_sessions (
    class_attendance_session_rule_day_of_week_snapshot,
    class_attendance_session_rule_start_time_snapshot
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- Index by type (per tenant)
CREATE INDEX IF NOT EXISTS idx_cas_type_alive
  ON class_attendance_sessions (
    class_attendance_session_school_id,
    class_attendance_session_type_id,
    class_attendance_session_date DESC
  )
  WHERE class_attendance_session_deleted_at IS NULL;

-- GIN utk snapshot type (query by JSONB)
CREATE INDEX IF NOT EXISTS gin_cas_type_snapshot
  ON class_attendance_sessions
  USING GIN (class_attendance_session_type_snapshot);



-- =====================================================
-- BACKFILL: isi flat keys di JSON snapshot
-- supaya kolom GENERATED *_snapshot kebaca
-- =====================================================
UPDATE class_attendance_sessions
SET class_attendance_session_csst_snapshot =
  COALESCE(class_attendance_session_csst_snapshot, '{}'::jsonb)
  ||
  jsonb_strip_nulls(
    jsonb_build_object(
      -- CSST ID (optional, kalau mau ada juga di snapshot)
      'csst_id',
        NULLIF(btrim(class_attendance_session_csst_snapshot->>'csst_id'), ''),

      -- SUBJECT (flat)
      'subject_id',
        NULLIF(btrim(class_attendance_session_csst_snapshot->'subject'->>'id'), ''),
      'subject_code',
        NULLIF(btrim(class_attendance_session_csst_snapshot->'subject'->>'code'), ''),
      'subject_name',
        NULLIF(btrim(class_attendance_session_csst_snapshot->'subject'->>'name'), ''),

      -- CLASS SECTION (flat)
      'section_id',
        NULLIF(btrim(class_attendance_session_csst_snapshot->'class_section'->>'id'), ''),
      'section_name',
        NULLIF(btrim(class_attendance_session_csst_snapshot->'class_section'->>'name'), ''),

      -- TEACHER (flat)
      'teacher_id',
        NULLIF(btrim(class_attendance_session_csst_snapshot->'school_teacher'->>'id'), ''),
      'teacher_name',
        NULLIF(btrim(class_attendance_session_csst_snapshot->'school_teacher'->>'name'), ''),

      -- ROOM (flat, opsional)
      'room_id',
        NULLIF(btrim(class_attendance_session_csst_snapshot->'class_room'->>'id'), ''),
      'room_name',
        NULLIF(btrim(class_attendance_session_csst_snapshot->'class_room'->>'name'), '')
    )
  )
WHERE class_attendance_session_csst_snapshot IS NOT NULL;



-- =========================================
-- TABLE: class_attendance_session_urls
-- =========================================
CREATE TABLE IF NOT EXISTS class_attendance_session_urls (
  class_attendance_session_url_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Tenant & owner
  class_attendance_session_url_school_id  UUID NOT NULL
    REFERENCES schools(school_id) ON DELETE CASCADE,
  class_attendance_session_url_session_id UUID NOT NULL
    REFERENCES class_attendance_sessions(class_attendance_session_id)
    ON UPDATE CASCADE ON DELETE CASCADE,

  -- Jenis/peran aset (mis. 'banner','image','video','attachment','link')
  class_attendance_session_url_kind VARCHAR(24) NOT NULL,

  -- storage (2-slot + retensi)
  class_attendance_session_url               TEXT,
  class_attendance_session_url_object_key    TEXT,
  class_attendance_session_url_old           TEXT,
  class_attendance_session_url_object_key_old TEXT,
  class_attendance_session_url_delete_pending_until TIMESTAMPTZ,

  -- Tampilan
  class_attendance_session_url_label      VARCHAR(160),
  class_attendance_session_url_order      INT NOT NULL DEFAULT 0,
  class_attendance_session_url_is_primary BOOLEAN NOT NULL DEFAULT FALSE,

  -- Audit & retensi
  class_attendance_session_url_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_url_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  class_attendance_session_url_deleted_at TIMESTAMPTZ  -- ← TANPA koma di akhir!
);

-- INDEXING / OPTIMIZATION (class_attendance_session_urls)

-- Lookup per session (live only) + urutan tampil
CREATE INDEX IF NOT EXISTS ix_casu_by_owner_live
  ON class_attendance_session_urls (
    class_attendance_session_url_session_id,
    class_attendance_session_url_kind,
    class_attendance_session_url_is_primary DESC,
    class_attendance_session_url_order,
    class_attendance_session_url_created_at
  )
  WHERE class_attendance_session_url_deleted_at IS NULL;

-- Filter per tenant (live only)
CREATE INDEX IF NOT EXISTS ix_casu_by_school_live
  ON class_attendance_session_urls (class_attendance_session_url_school_id)
  WHERE class_attendance_session_url_deleted_at IS NULL;

-- Satu primary per (session, kind) (live only)
CREATE UNIQUE INDEX IF NOT EXISTS uq_casu_primary_per_kind_alive
  ON class_attendance_session_urls (class_attendance_session_url_session_id, class_attendance_session_url_kind)
  WHERE class_attendance_session_url_deleted_at IS NULL
    AND class_attendance_session_url_is_primary = TRUE;

-- Kandidat purge (retensi object storage)
CREATE INDEX IF NOT EXISTS ix_casu_purge_due
  ON class_attendance_session_urls (class_attendance_session_url_delete_pending_until)
  WHERE class_attendance_session_url_delete_pending_until IS NOT NULL
    AND (
      (class_attendance_session_url_deleted_at IS NULL  AND class_attendance_session_url_object_key_old IS NOT NULL) OR
      (class_attendance_session_url_deleted_at IS NOT NULL AND class_attendance_session_url_object_key     IS NOT NULL)
    );

-- (opsional) pencarian label (live only)
CREATE INDEX IF NOT EXISTS gin_casu_label_trgm_live
  ON class_attendance_session_urls USING GIN ((class_attendance_session_url_label) gin_trgm_ops)
  WHERE class_attendance_session_url_deleted_at IS NULL;
