-- +migrate Down
BEGIN;

-- assessment_urls
DROP INDEX IF EXISTS gin_assessment_urls_label_trgm_live;
DROP INDEX IF EXISTS ix_assessment_urls_purge_due;
DROP INDEX IF EXISTS uq_assessment_urls_assessment_url_alive;
DROP INDEX IF EXISTS uq_assessment_urls_primary_per_kind_alive;
DROP INDEX IF EXISTS ix_assessment_urls_by_masjid_live;
DROP INDEX IF EXISTS ix_assessment_urls_by_owner_live;
DROP INDEX IF EXISTS uq_assessment_urls_id_tenant;
DROP TABLE IF EXISTS assessment_urls;

-- assessments
DROP INDEX IF EXISTS brin_assessments_created_at;
DROP INDEX IF EXISTS idx_assessments_collect_session_alive;
DROP INDEX IF EXISTS idx_assessments_announce_session_alive;
DROP INDEX IF EXISTS idx_assessments_submission_mode_alive;
DROP INDEX IF EXISTS idx_assessments_created_by_teacher;
DROP INDEX IF EXISTS idx_assessments_csst;
DROP INDEX IF EXISTS idx_assessments_masjid_created_at;
DROP INDEX IF EXISTS gin_assessments_slug_trgm_alive;
DROP INDEX IF EXISTS uq_assessments_slug_per_tenant_alive;
DROP INDEX IF EXISTS uq_assessments_id_tenant;
DROP TABLE IF EXISTS assessments;

-- assessment_types
DROP INDEX IF EXISTS brin_assessment_types_created_at;
DROP INDEX IF EXISTS idx_assessment_types_masjid_active;
DROP INDEX IF EXISTS uq_assessment_types_key_per_masjid_alive;
DROP INDEX IF EXISTS uq_assessment_types_id_tenant;
DROP TABLE IF EXISTS assessment_types;

-- index guard CSST
DROP INDEX IF EXISTS uq_csst_id_masjid;

COMMIT;
