BEGIN;

-- 1) Bersihkan dependensi class_term_openings supaya classes boleh di-drop
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables
             WHERE table_schema='public' AND table_name='class_term_openings') THEN

    -- Matikan trigger & FK yang mungkin ada
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_cto_quota_nonnegative') THEN
      DROP TRIGGER trg_cto_quota_nonnegative ON class_term_openings;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_touch_class_term_openings') THEN
      DROP TRIGGER trg_touch_class_term_openings ON class_term_openings;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cto_class_masjid_pair') THEN
      ALTER TABLE class_term_openings DROP CONSTRAINT fk_cto_class_masjid_pair;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname='fk_cto_term_masjid_pair') THEN
      ALTER TABLE class_term_openings DROP CONSTRAINT fk_cto_term_masjid_pair;
    END IF;

    -- Drop index-index non-essensial (aman kalau tidak ada)
    DROP INDEX IF EXISTS ix_cto_tenant_term_open_live;
    DROP INDEX IF EXISTS ix_cto_tenant_class_live;
    DROP INDEX IF EXISTS ix_cto_reg_window_live;
    DROP INDEX IF EXISTS gin_cto_notes_trgm_live;
    DROP INDEX IF EXISTS ix_cto_created_at_live;
    DROP INDEX IF EXISTS ix_cto_updated_at_live;

    -- Hapus tabelnya
    DROP TABLE IF EXISTS class_term_openings;

    -- Drop function terkait (kalau ada)
    DROP FUNCTION IF EXISTS class_term_openings_claim(UUID);
    DROP FUNCTION IF EXISTS class_term_openings_release(UUID);
    DROP FUNCTION IF EXISTS fn_cto_quota_nonnegative();
    DROP FUNCTION IF EXISTS fn_touch_class_term_openings_updated_at();
  END IF;
END$$;

COMMIT;
