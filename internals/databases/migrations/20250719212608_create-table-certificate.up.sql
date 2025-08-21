-- =====================================================================
-- Migration: certificates & user_certificates (optimized)
-- DB: PostgreSQL
-- =====================================================================

BEGIN;

-- -------------------------------------------------
-- Extensions (idempotent)
-- -------------------------------------------------
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- -------------------------------------------------
-- Trigger helpers
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION fn_touch_updated_at_generic()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at := CURRENT_TIMESTAMP;
  RETURN NEW;
END$$ LANGUAGE plpgsql;

-- =====================================================================
-- ===============================  UP  =================================
-- =====================================================================

-- =============================
-- 🎓 certificates (master)
-- =============================
CREATE TABLE IF NOT EXISTS certificates (
  certificate_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  certificate_title        TEXT NOT NULL,
  certificate_description  TEXT,
  certificate_lecture_id   UUID NOT NULL REFERENCES lectures(lecture_id) ON DELETE CASCADE,
  certificate_template_url TEXT,
  created_at               TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at               TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Touch updated_at on UPDATE
DROP TRIGGER IF EXISTS trg_certificates_touch ON certificates;
CREATE TRIGGER trg_certificates_touch
BEFORE UPDATE ON certificates
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at_generic();

-- Indexes (certificates)
-- 1) Listing per lecture + terbaru
CREATE INDEX IF NOT EXISTS idx_certificates_lecture_created
  ON certificates(certificate_lecture_id, created_at DESC);

-- 2) Search judul/deskripsi (ILIKE/%%)
CREATE INDEX IF NOT EXISTS idx_certificates_title_trgm
  ON certificates USING GIN (certificate_title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_certificates_desc_trgm
  ON certificates USING GIN (certificate_description gin_trgm_ops);

-- (Opsional) pastikan satu lecture hanya punya satu jenis certificate tertentu
-- aktifkan jika dibutuhkan:
-- CREATE UNIQUE INDEX IF NOT EXISTS uq_certificates_lecture_title
--   ON certificates(certificate_lecture_id, certificate_title);


-- =============================
-- 👤 user_certificates (issued to users)
-- =============================
CREATE TABLE IF NOT EXISTS user_certificates (
  user_cert_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_cert_user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_cert_certificate_id UUID NOT NULL REFERENCES certificates(certificate_id) ON DELETE CASCADE,

  user_cert_score         INTEGER CHECK (user_cert_score IS NULL OR user_cert_score BETWEEN 0 AND 100),
  user_cert_slug_url      TEXT UNIQUE NOT NULL,           -- untuk akses publik
  user_cert_is_up_to_date BOOLEAN NOT NULL DEFAULT TRUE,  -- masih valid?
  user_cert_issued_at     TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

  created_at              TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at              TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

  -- satu certificate hanya boleh sekali per user (hindari duplikat)
  CONSTRAINT uq_user_cert_user_certificate UNIQUE (user_cert_user_id, user_cert_certificate_id)
);

-- Touch updated_at on UPDATE
DROP TRIGGER IF EXISTS trg_user_certificates_touch ON user_certificates;
CREATE TRIGGER trg_user_certificates_touch
BEFORE UPDATE ON user_certificates
FOR EACH ROW
EXECUTE FUNCTION fn_touch_updated_at_generic();

-- Indexes (user_certificates)
-- 1) Query profil user: sertifikat terbaru
CREATE INDEX IF NOT EXISTS idx_user_cert_user_created
  ON user_certificates(user_cert_user_id, user_cert_issued_at DESC);

-- 2) Rekap per certificate: penerima & skor
CREATE INDEX IF NOT EXISTS idx_user_cert_cert_issued
  ON user_certificates(user_cert_certificate_id, user_cert_issued_at DESC);

-- 3) Active/valid certificates (up-to-date) – cepat untuk publikasi
CREATE INDEX IF NOT EXISTS idx_user_cert_active_by_user
  ON user_certificates(user_cert_user_id)
  WHERE user_cert_is_up_to_date = TRUE;

CREATE INDEX IF NOT EXISTS idx_user_cert_active_by_cert
  ON user_certificates(user_cert_certificate_id)
  WHERE user_cert_is_up_to_date = TRUE;

-- 4) Leaderboard cepat (jika digunakan): per certificate, skor tinggi dulu lalu waktu
CREATE INDEX IF NOT EXISTS idx_user_cert_rank
  ON user_certificates(user_cert_certificate_id, user_cert_score DESC, user_cert_issued_at ASC);

-- 5) (Opsional) pencarian slug cepat sudah di-cover UNIQUE (btree)


COMMIT;