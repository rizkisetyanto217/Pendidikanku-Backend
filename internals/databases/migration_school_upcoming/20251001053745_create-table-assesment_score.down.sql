-- ============================================
-- DOWN Migration — Assessment Scores & Snapshots
-- ============================================
BEGIN;

-- 1) SNAPSHOTS
-- Drop trigger dulu supaya gak ada dependensi
DROP TRIGGER IF EXISTS trg_assessment_score_snapshot_updated
  ON assessment_score_snapshots;

-- Drop exclusion constraint (anti overlap)
ALTER TABLE IF EXISTS assessment_score_snapshots
  DROP CONSTRAINT IF EXISTS assessment_score_snapshot_final_no_overlap;

-- Drop indexes (aman meski tabel belum ada; tapi urutkan sebelum DROP TABLE)
DROP INDEX IF EXISTS pidx_assessment_score_snapshot_final;
DROP INDEX IF EXISTS gin_assessment_score_snapshot_grade_hist;
DROP INDEX IF EXISTS gin_assessment_score_snapshot_status_counts;
DROP INDEX IF EXISTS brin_assessment_score_snapshot_start;
DROP INDEX IF EXISTS brin_assessment_score_snapshot_created;
DROP INDEX IF EXISTS idx_assessment_score_snapshot_tenant_grain_start;

-- Drop table snapshots
DROP TABLE IF EXISTS assessment_score_snapshots;

-- Drop enum type (harus setelah tabel yang pakai enum di-drop)
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'assessment_score_grain') THEN
    DROP TYPE assessment_score_grain;
  END IF;
END$$;


-- 2) LIFETIME PER ASSESSMENT
-- Drop trigger
DROP TRIGGER IF EXISTS trg_assessment_score_updated
  ON assessment_scores;

-- Drop indexes
DROP INDEX IF EXISTS pidx_assessment_score_final;
DROP INDEX IF EXISTS gin_assessment_score_grade_hist;
DROP INDEX IF EXISTS gin_assessment_score_status_counts;
DROP INDEX IF EXISTS brin_assessment_score_created;
DROP INDEX IF EXISTS idx_assessment_score_tenant_assessment;

-- Drop table
DROP TABLE IF EXISTS assessment_scores;


-- 3) Helper functions untuk updated_at
DROP FUNCTION IF EXISTS set_assessment_score_snapshot_updated_at();
DROP FUNCTION IF EXISTS set_assessment_score_updated_at();

COMMIT;
