BEGIN;

-- =========================
-- TABLE
-- =========================
CREATE TABLE IF NOT EXISTS masjid_stats (
    masjid_stats_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_stats_total_lectures    INT    NOT NULL DEFAULT 0,   -- jumlah kajian
    masjid_stats_total_sessions    INT    NOT NULL DEFAULT 0,   -- jumlah pertemuan
    masjid_stats_total_participants INT   NOT NULL DEFAULT 0,   -- total kehadiran
    masjid_stats_total_donations   BIGINT NOT NULL DEFAULT 0,   -- total donasi (rupiah)
    masjid_stats_masjid_id         UUID   NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    masjid_stats_created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    masjid_stats_updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (masjid_stats_masjid_id)
);

-- Constraint non-negatif (idempotent)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'ck_masjid_stats_nonneg'
  ) THEN
    ALTER TABLE masjid_stats
      ADD CONSTRAINT ck_masjid_stats_nonneg CHECK (
        masjid_stats_total_lectures     >= 0 AND
        masjid_stats_total_sessions     >= 0 AND
        masjid_stats_total_participants >= 0 AND
        masjid_stats_total_donations    >= 0
      );
  END IF;
END $$;

-- =========================
-- INDEXING & OPTIMIZE
-- =========================
-- UNIQUE di atas sudah otomatis membuat index unik untuk masjid_stats_masjid_id.
-- Jadi index non-unik yang sama → tidak perlu; hapus jika sebelumnya dibuat.
DROP INDEX IF EXISTS idx_masjid_stats_masjid_id;

-- Jika sering ORDER BY updated_at per masjid (mis. histori audit)
CREATE INDEX IF NOT EXISTS idx_masjid_stats_masjid_id_updated_at
  ON masjid_stats (masjid_stats_masjid_id, masjid_stats_updated_at DESC);

-- Jika sering ambil "yang terbaru" global
CREATE INDEX IF NOT EXISTS idx_masjid_stats_updated_at_desc
  ON masjid_stats (masjid_stats_updated_at DESC);

-- =========================
-- TRIGGER updated_at
-- =========================
CREATE OR REPLACE FUNCTION set_updated_at_masjid_stats() RETURNS trigger AS $$
BEGIN
  NEW.masjid_stats_updated_at = now();
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_updated_at_masjid_stats ON masjid_stats;
CREATE TRIGGER trg_set_updated_at_masjid_stats
BEFORE UPDATE ON masjid_stats
FOR EACH ROW EXECUTE FUNCTION set_updated_at_masjid_stats();

COMMIT;
