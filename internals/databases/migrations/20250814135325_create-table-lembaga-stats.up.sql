CREATE TABLE IF NOT EXISTS lembaga_stats (
  lembaga_stats_lembaga_id UUID PRIMARY KEY
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lembaga_stats_active_classes   INT NOT NULL DEFAULT 0, -- kelas aktif
  lembaga_stats_active_sections  INT NOT NULL DEFAULT 0, -- section aktif
  lembaga_stats_active_students  INT NOT NULL DEFAULT 0, -- siswa aktif
  lembaga_stats_active_teachers  INT NOT NULL DEFAULT 0, -- guru aktif (role-based)

  lembaga_stats_created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  lembaga_stats_updated_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_lembaga_stats_updated_at
  ON lembaga_stats(lembaga_stats_updated_at);
