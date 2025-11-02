-- =========================================================
-- lembaga_stats (global snapshot, tanpa term)
-- =========================================================
CREATE TABLE IF NOT EXISTS lembaga_stats (
  lembaga_stats_school_id UUID PRIMARY KEY
    REFERENCES schools(school_id) ON DELETE CASCADE,

  lembaga_stats_active_classes  INT NOT NULL DEFAULT 0,
  lembaga_stats_active_sections INT NOT NULL DEFAULT 0,
  lembaga_stats_active_students INT NOT NULL DEFAULT 0,
  lembaga_stats_active_teachers INT NOT NULL DEFAULT 0,

  lembaga_stats_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  lembaga_stats_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Guard non-negatif (idempotent)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='chk_lembaga_stats_nonneg') THEN
    ALTER TABLE lembaga_stats
      ADD CONSTRAINT chk_lembaga_stats_nonneg
      CHECK (
        lembaga_stats_active_classes  >= 0 AND
        lembaga_stats_active_sections >= 0 AND
        lembaga_stats_active_students >= 0 AND
        lembaga_stats_active_teachers >= 0
      );
  END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_lembaga_stats_updated_at
  ON lembaga_stats(lembaga_stats_updated_at);


