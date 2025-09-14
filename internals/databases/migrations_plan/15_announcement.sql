BEGIN;

-- Extensions yang dibutuhkan
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- trigram index untuk pencarian

-- =========================================================
-- TABLE: announcements (final)
-- =========================================================
DROP TABLE IF EXISTS announcements CASCADE;
CREATE TABLE IF NOT EXISTS announcements (
  announcement_id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_masjid_id          UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- authoring
  announcement_created_by_teacher_id UUID,
  announcement_created_by_user_id    UUID,

  -- audience (scope & targets)
  announcement_is_global         BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_class_section_id  UUID,        -- simple single-target (legacy)
  announcement_section_ids       UUID[],      -- multi section
  announcement_class_ids         UUID[],      -- multi class
  announcement_grade_levels      INT[],       -- target tingkat
  announcement_audience_tags     TEXT[],      -- target by tag
  announcement_audience_query    JSONB,       -- rule dynamic segmentasi (opsional)

  -- theme (opsional, putuskan ON DELETE sesuai kebutuhan)
  announcement_theme_id          UUID,

  -- content
  announcement_title             VARCHAR(200) NOT NULL,
  announcement_subtitle          VARCHAR(200),
  announcement_summary           TEXT,
  announcement_content           TEXT NOT NULL,

  -- media & attachments
  announcement_image_url         TEXT,
  announcement_banner_url        TEXT,
  announcement_image_urls        TEXT[],      -- multi image
  announcement_video_url         TEXT,
  announcement_attachments       JSONB,       -- daftar file/link terstruktur
  announcement_attachments_count INT,

  -- call-to-action
  announcement_link_url          TEXT,
  announcement_link_label        VARCHAR(80),

  -- scheduling & lifecycle
  announcement_status            TEXT,        -- 'draft'|'scheduled'|'published'|'archived'
  announcement_date              DATE,        -- tanggal tampil (opsional)
  announcement_publish_at        TIMESTAMPTZ, -- jadwal tayang
  announcement_expire_at         TIMESTAMPTZ, -- kadaluarsa konten
  announcement_embargo_until     TIMESTAMPTZ, -- hold sampai waktu tertentu
  announcement_recurrence_rule   TEXT,        -- RRULE (opsional)
  announcement_resend_policy     TEXT,        -- 'none'|'once'|'daily'... (opsional)
  announcement_is_active         BOOLEAN NOT NULL DEFAULT TRUE,
  announcement_is_pinned         BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_pin_until         TIMESTAMPTZ, -- auto unpin
  announcement_priority          SMALLINT,    -- 1 = paling tinggi
  announcement_is_silent         BOOLEAN DEFAULT FALSE, -- kirim tanpa notif

  -- delivery channels
  announcement_via_web           BOOLEAN NOT NULL DEFAULT TRUE,
  announcement_via_push          BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_via_email         BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_via_sms           BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_via_whatsapp      BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_sent_via          JSONB,       -- ringkasan channel terpakai

  -- delivery & engagement metrics
  announcement_total_recipients  INT,
  announcement_sent_count        INT,
  announcement_read_count        INT,
  announcement_ack_count         INT,
  announcement_last_sent_at      TIMESTAMPTZ,
  announcement_last_read_at      TIMESTAMPTZ,
  announcement_failure_reason    TEXT,

  -- interaction
  announcement_allow_comments    BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_require_ack       BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_allow_reactions   BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_reaction_counts   JSONB,   -- { "like": 12, "love": 3, ... }
  announcement_poll              JSONB,   -- { "question":"...", "options":[...]}
  announcement_poll_ends_at      TIMESTAMPTZ,

  -- categorization & integration
  announcement_tags              TEXT[],
  announcement_campaign_code     VARCHAR(80),
  announcement_utm               JSONB,   -- {source, medium, ...}
  announcement_source            TEXT,    -- 'manual'|'import'|'api'
  announcement_template_id       UUID,
  announcement_content_version   INT DEFAULT 1,
  announcement_draft_note        TEXT,
  announcement_external_ref      TEXT,

  -- compliance / access / i18n
  announcement_slug              VARCHAR(160),
  announcement_locale            VARCHAR(20),
  announcement_timezone          TEXT,
  announcement_moderation_status TEXT,    -- 'pending'|'approved'|'rejected'
  announcement_approved_by_user_id UUID,
  announcement_approved_at       TIMESTAMPTZ,
  announcement_rejection_reason  TEXT,
  announcement_visibility_scope  TEXT,    -- 'tenant'|'campus'|'class'... (sesuaikan)
  announcement_exclude_section_ids UUID[],
  announcement_min_app_version   TEXT,
  announcement_is_sensitive      BOOLEAN DEFAULT FALSE,
  announcement_requires_login    BOOLEAN DEFAULT TRUE,
  announcement_data_retention_days INT,

  -- audit
  announcement_created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_deleted_at        TIMESTAMPTZ,

  -- Full-text search (title + content)
  announcement_search tsvector
    GENERATED ALWAYS AS (
      setweight(to_tsvector('simple', coalesce(announcement_title,   '')), 'A') ||
      setweight(to_tsvector('simple', coalesce(announcement_content, '')), 'B')
    ) STORED,

  -- CHECK ringan untuk enum/aturan penting (opsional tapi membantu kualitas data)
  CONSTRAINT ck_ann_status CHECK (
    announcement_status IS NULL OR announcement_status IN ('draft','scheduled','published','archived')
  ),
  CONSTRAINT ck_ann_moderation CHECK (
    announcement_moderation_status IS NULL OR announcement_moderation_status IN ('pending','approved','rejected')
  ),
  CONSTRAINT ck_ann_publish_vs_expire CHECK (
    announcement_publish_at IS NULL OR announcement_expire_at IS NULL OR announcement_expire_at >= announcement_publish_at
  )
);

-- =========================================================
-- INDEXES (soft-delete aware di mana relevan)
-- =========================================================

-- Slug unik per tenant (live rows)
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcements_slug_per_tenant_live
  ON announcements (announcement_masjid_id, lower(announcement_slug))
  WHERE announcement_slug IS NOT NULL AND announcement_deleted_at IS NULL;

-- Status & active
CREATE INDEX IF NOT EXISTS ix_announcements_status_live
  ON announcements (announcement_masjid_id, announcement_status)
  WHERE announcement_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_announcements_active_live
  ON announcements (announcement_masjid_id, announcement_is_active)
  WHERE announcement_deleted_at IS NULL;

-- Publish & expire window
CREATE INDEX IF NOT EXISTS ix_announcements_publish_at_live
  ON announcements (announcement_masjid_id, announcement_publish_at)
  WHERE announcement_deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS ix_announcements_expire_at_live
  ON announcements (announcement_masjid_id, announcement_expire_at)
  WHERE announcement_deleted_at IS NULL;

-- Pin & priority
CREATE INDEX IF NOT EXISTS ix_announcements_pinned_live
  ON announcements (announcement_masjid_id, announcement_is_pinned, announcement_priority NULLS LAST, announcement_pin_until)
  WHERE announcement_deleted_at IS NULL;

-- Multichannel flags (untuk filter cepat)
CREATE INDEX IF NOT EXISTS ix_announcements_channels_live
  ON announcements (announcement_masjid_id, announcement_via_web, announcement_via_push, announcement_via_email, announcement_via_sms, announcement_via_whatsapp)
  WHERE announcement_deleted_at IS NULL;

-- Moderation status
CREATE INDEX IF NOT EXISTS ix_announcements_moderation_live
  ON announcements (announcement_masjid_id, announcement_moderation_status)
  WHERE announcement_deleted_at IS NULL;

-- FTS
CREATE INDEX IF NOT EXISTS gin_announcements_search
  ON announcements USING GIN (announcement_search);

COMMIT;



BEGIN;

-- =========================================================
-- TABLE: announcement_themes (final)
-- =========================================================
DROP TABLE IF EXISTS announcement_themes CASCADE;
CREATE TABLE IF NOT EXISTS announcement_themes (
  announcement_themes_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  announcement_themes_masjid_id  UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  -- basic
  announcement_themes_name        VARCHAR(80)  NOT NULL,
  announcement_themes_slug        VARCHAR(120) NOT NULL,
  announcement_themes_description TEXT,

  -- colors & assets
  announcement_themes_color       VARCHAR(20),
  announcement_themes_color_text  VARCHAR(20),
  announcement_themes_color_bg    VARCHAR(20),
  announcement_themes_icon_url    TEXT,
  announcement_themes_cover_url   TEXT,

  -- behavior & options
  announcement_themes_is_active   BOOLEAN NOT NULL DEFAULT TRUE,
  announcement_themes_is_default  BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_themes_order_index INT,
  announcement_themes_options     JSONB,
  announcement_themes_tags        TEXT[],
  announcement_themes_external_ref TEXT,

  -- dark mode & typography
  announcement_themes_is_dark     BOOLEAN DEFAULT FALSE,
  announcement_themes_font_family VARCHAR(120),
  announcement_themes_font_scale  NUMERIC(4,2),
  announcement_themes_design_tokens JSONB,   -- design tokens { "radius":"1rem","colors":{...} }

  -- locale & versioning
  announcement_themes_locale      VARCHAR(20),   -- 'id-ID','en-US'
  announcement_themes_is_rtl      BOOLEAN DEFAULT FALSE,
  announcement_themes_version     INT DEFAULT 1,
  announcement_themes_preview_url TEXT,
  announcement_themes_is_locked   BOOLEAN DEFAULT FALSE,

  -- audit
  announcement_themes_created_by_user_id UUID,
  announcement_themes_updated_by_user_id UUID,

  -- timestamps
  announcement_themes_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_themes_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_themes_deleted_at  TIMESTAMPTZ,

  -- validation
  CONSTRAINT ck_announcement_themes_slug CHECK (announcement_themes_slug ~ '^[a-z0-9-]+$')
);

-- =========================================================
-- INDEXES (soft-delete aware)
-- =========================================================

-- Unik per tenant (by name & slug)
CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_tenant_name_live
  ON announcement_themes (announcement_themes_masjid_id, lower(announcement_themes_name))
  WHERE announcement_themes_deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_announcement_themes_tenant_slug_live
  ON announcement_themes (announcement_themes_masjid_id, announcement_themes_slug)
  WHERE announcement_themes_deleted_at IS NULL;

-- Aktif per tenant
CREATE INDEX IF NOT EXISTS ix_announcement_themes_tenant_active_live
  ON announcement_themes (announcement_themes_masjid_id, announcement_themes_is_active)
  WHERE announcement_themes_deleted_at IS NULL;

-- Fuzzy search name
CREATE INDEX IF NOT EXISTS ix_announcement_themes_name_trgm_live
  ON announcement_themes USING GIN (announcement_themes_name gin_trgm_ops)
  WHERE announcement_themes_deleted_at IS NULL;

COMMIT;


BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- TABLE: announcement_reads (final)
-- =========================================================
DROP TABLE IF EXISTS announcement_reads CASCADE;
CREATE TABLE IF NOT EXISTS announcement_reads (
  announcement_reads_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  announcement_reads_masjid_id   UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  announcement_reads_announcement_id UUID NOT NULL
    REFERENCES announcements(announcement_id) ON DELETE CASCADE,

  -- siapa yang membaca (user, student, atau guardian — pilih salah satu; isi salah satunya saja)
  announcement_reads_user_id           UUID,
  announcement_reads_masjid_student_id UUID,
  announcement_reads_guardian_id       UUID,

  announcement_reads_first_read_at TIMESTAMPTZ,
  announcement_reads_last_read_at  TIMESTAMPTZ,
  announcement_reads_read_count    INT NOT NULL DEFAULT 1,

  announcement_reads_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_reads_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_reads_deleted_at TIMESTAMPTZ,

  -- Guards
  CONSTRAINT chk_ann_reads_positive_count CHECK (announcement_reads_read_count >= 1),
  CONSTRAINT chk_ann_reads_time_order CHECK (
    announcement_reads_last_read_at IS NULL
    OR announcement_reads_first_read_at IS NULL
    OR announcement_reads_last_read_at >= announcement_reads_first_read_at
  ),
  -- minimal satu identitas pembaca diisi
  CONSTRAINT chk_ann_reads_identity_present CHECK (
    (announcement_reads_user_id IS NOT NULL)::int
    + (announcement_reads_masjid_student_id IS NOT NULL)::int
    + (announcement_reads_guardian_id IS NOT NULL)::int >= 1
  )
);

-- Satu baris aktif per (tenant, announcement, identity) — soft-delete aware
CREATE UNIQUE INDEX IF NOT EXISTS uq_ann_reads_unique_alive
  ON announcement_reads (
    announcement_reads_masjid_id,
    announcement_reads_announcement_id,
    COALESCE(announcement_reads_user_id,            '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE(announcement_reads_masjid_student_id,  '00000000-0000-0000-0000-000000000000'::uuid),
    COALESCE(announcement_reads_guardian_id,        '00000000-0000-0000-0000-000000000000'::uuid)
  )
  WHERE announcement_reads_deleted_at IS NULL;

-- Lookups umum
CREATE INDEX IF NOT EXISTS ix_ann_reads_by_announcement
  ON announcement_reads (announcement_reads_masjid_id, announcement_reads_announcement_id)
  WHERE announcement_reads_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ann_reads_by_user
  ON announcement_reads (announcement_reads_masjid_id, announcement_reads_user_id)
  WHERE announcement_reads_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ann_reads_first_last
  ON announcement_reads (announcement_reads_first_read_at, announcement_reads_last_read_at);


-- Auto-update updated_at
CREATE OR REPLACE FUNCTION set_updated_at_announcement_reads() RETURNS trigger AS $$
BEGIN
  NEW.announcement_reads_updated_at := NOW();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_announcement_reads_updated ON announcement_reads;
CREATE TRIGGER trg_announcement_reads_updated
BEFORE UPDATE ON announcement_reads
FOR EACH ROW EXECUTE FUNCTION set_updated_at_announcement_reads();



-- =========================================================
-- TABLE: announcement_comments (final)
-- =========================================================
DROP TABLE IF EXISTS announcement_comments CASCADE;
CREATE TABLE IF NOT EXISTS announcement_comments (
  announcement_comments_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  announcement_comments_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  announcement_comments_announcement_id UUID NOT NULL
    REFERENCES announcements(announcement_id) ON DELETE CASCADE,

  -- siapa yang komentar (boleh user/student/guardian — salah satu minimal)
  announcement_comments_user_id           UUID,
  announcement_comments_masjid_student_id UUID,
  announcement_comments_guardian_id       UUID,

  -- threading sederhana
  announcement_comments_parent_id UUID,          -- reply ke komentar lain (NULL = root)

  -- isi & kontrol
  announcement_comments_content   TEXT NOT NULL,
  announcement_comments_is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_comments_is_hidden BOOLEAN NOT NULL DEFAULT FALSE,
  announcement_comments_moderation_status TEXT,  -- 'pending'|'approved'|'rejected'
  announcement_comments_rejection_reason  TEXT,

  -- metrik ringan
  announcement_comments_like_count INT DEFAULT 0 CHECK (announcement_comments_like_count >= 0),
  announcement_comments_reply_count INT DEFAULT 0 CHECK (announcement_comments_reply_count >= 0),

  -- audit
  announcement_comments_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_comments_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  announcement_comments_deleted_at TIMESTAMPTZ,

  -- Guards
  CONSTRAINT chk_ann_comments_identity_present CHECK (
    (announcement_comments_user_id IS NOT NULL)::int
    + (announcement_comments_masjid_student_id IS NOT NULL)::int
    + (announcement_comments_guardian_id IS NOT NULL)::int >= 1
  ),
  CONSTRAINT chk_ann_comments_moderation CHECK (
    announcement_comments_moderation_status IS NULL
    OR announcement_comments_moderation_status IN ('pending','approved','rejected')
  )
);

-- Threading FK opsional (parent harus di tabel & announcement sama)
-- (tanpa FK siklik agar mudah soft-delete; validasi bisa di app/trigger jika perlu)

-- Indeks umum
CREATE INDEX IF NOT EXISTS ix_ann_comments_by_announcement
  ON announcement_comments (announcement_comments_masjid_id, announcement_comments_announcement_id, announcement_comments_created_at)
  WHERE announcement_comments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ann_comments_by_user
  ON announcement_comments (announcement_comments_masjid_id, announcement_comments_user_id, announcement_comments_created_at)
  WHERE announcement_comments_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_ann_comments_parent
  ON announcement_comments (announcement_comments_parent_id)
  WHERE announcement_comments_deleted_at IS NULL;

-- Fuzzy search konten
CREATE INDEX IF NOT EXISTS gin_ann_comments_content_trgm
  ON announcement_comments USING GIN (announcement_comments_content gin_trgm_ops)
  WHERE announcement_comments_deleted_at IS NULL;

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION set_updated_at_announcement_comments() RETURNS trigger AS $$
BEGIN
  NEW.announcement_comments_updated_at := NOW();
  RETURN NEW;
END$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_announcement_comments_updated ON announcement_comments;
CREATE TRIGGER trg_announcement_comments_updated
BEFORE UPDATE ON announcement_comments
FOR EACH ROW EXECUTE FUNCTION set_updated_at_announcement_comments();

COMMIT;