-- =========================================================
-- MIGRATION: lecture_stats (TIMPESTAMPTZ, auto-recalc)
-- =========================================================

-- =========================
-- ========== UP ===========
-- =========================
BEGIN;

-- Tabel utama
CREATE TABLE IF NOT EXISTS lecture_stats (
  lecture_stats_id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  lecture_stats_lecture_id          UUID NOT NULL REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  lecture_stats_masjid_id           UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  lecture_stats_total_participants  INT   NOT NULL DEFAULT 0,
  lecture_stats_average_grade       FLOAT NOT NULL DEFAULT 0,
  lecture_stats_updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT ux_lecture_stats_lecture UNIQUE (lecture_stats_lecture_id),
  CONSTRAINT chk_stats_total_nonneg CHECK (lecture_stats_total_participants >= 0),
  CONSTRAINT chk_stats_avg_range  CHECK (lecture_stats_average_grade >= 0 AND lecture_stats_average_grade <= 100)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_lecture_stats_lecture_id ON lecture_stats(lecture_stats_lecture_id);
CREATE INDEX IF NOT EXISTS idx_lecture_stats_masjid_id  ON lecture_stats(lecture_stats_masjid_id);
CREATE INDEX IF NOT EXISTS idx_lecture_stats_masjid_recent
  ON lecture_stats(lecture_stats_masjid_id, lecture_stats_updated_at DESC);


-- =========================================
-- Recalc function: hitung ulang statistik
-- =========================================
CREATE OR REPLACE FUNCTION recalc_lecture_stats(p_lecture_id UUID)
RETURNS VOID AS $$
DECLARE
  v_masjid_id UUID;
  v_total     INT;
  v_avg       FLOAT;
BEGIN
  -- Ambil masjid_id dari lectures
  SELECT lecture_masjid_id INTO v_masjid_id
  FROM lectures
  WHERE lecture_id = p_lecture_id;

  IF v_masjid_id IS NULL THEN
    -- Lecture tidak ada; baris akan terhapus via CASCADE
    RETURN;
  END IF;

  -- Total peserta (semua user_lectures untuk lecture tsb)
  SELECT COUNT(*)::INT INTO v_total
  FROM user_lectures
  WHERE user_lecture_lecture_id = p_lecture_id;

  -- Rata-rata nilai (0 jika semua NULL)
  SELECT COALESCE(AVG(user_lecture_grade_result)::FLOAT, 0) INTO v_avg
  FROM user_lectures
  WHERE user_lecture_lecture_id = p_lecture_id
    AND user_lecture_grade_result IS NOT NULL;

  -- Upsert
  INSERT INTO lecture_stats (
    lecture_stats_lecture_id,
    lecture_stats_masjid_id,
    lecture_stats_total_participants,
    lecture_stats_average_grade
  ) VALUES (
    p_lecture_id,
    v_masjid_id,
    v_total,
    v_avg
  )
  ON CONFLICT (lecture_stats_lecture_id)
  DO UPDATE SET
    lecture_stats_masjid_id          = EXCLUDED.lecture_stats_masjid_id,
    lecture_stats_total_participants = EXCLUDED.lecture_stats_total_participants,
    lecture_stats_average_grade      = EXCLUDED.lecture_stats_average_grade,
    lecture_stats_updated_at         = CURRENT_TIMESTAMPTZ;
END;
$$ LANGUAGE plpgsql;

-- =========================================
-- Triggers pada user_lectures untuk auto-recalc
-- =========================================
CREATE OR REPLACE FUNCTION trg_user_lectures_recalc_stats_fn()
RETURNS TRIGGER AS $$
DECLARE
  v_lecture_id UUID;
BEGIN
  IF (TG_OP = 'INSERT') THEN
    v_lecture_id := NEW.user_lecture_lecture_id;
  ELSIF (TG_OP = 'UPDATE') THEN
    IF NEW.user_lecture_lecture_id IS DISTINCT FROM OLD.user_lecture_lecture_id THEN
      PERFORM recalc_lecture_stats(OLD.user_lecture_lecture_id);
    END IF;
    v_lecture_id := NEW.user_lecture_lecture_id;
  ELSIF (TG_OP = 'DELETE') THEN
    v_lecture_id := OLD.user_lecture_lecture_id;
  END IF;

  IF v_lecture_id IS NOT NULL THEN
    PERFORM recalc_lecture_stats(v_lecture_id);
  END IF;

  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_user_lectures_recalc_stats_aiud ON user_lectures;
CREATE TRIGGER trg_user_lectures_recalc_stats_aiud
AFTER INSERT OR UPDATE OR DELETE ON user_lectures
FOR EACH ROW
EXECUTE FUNCTION trg_user_lectures_recalc_stats_fn();

COMMIT;
