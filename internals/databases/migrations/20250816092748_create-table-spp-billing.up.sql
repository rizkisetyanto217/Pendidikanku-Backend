-- =========================================================
-- UP MIGRATION (PostgreSQL)
-- =========================================================

-- Extension untuk UUID
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =========================================================
-- SPP BILLINGS (tambah relasi opsional ke academic_terms)
-- =========================================================
CREATE TABLE IF NOT EXISTS spp_billings (
  spp_billing_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  spp_billing_masjid_id   UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,
  spp_billing_class_id    UUID REFERENCES classes(class_id)   ON DELETE SET NULL,

  spp_billing_month       SMALLINT NOT NULL CHECK (spp_billing_month BETWEEN 1 AND 12),
  spp_billing_year        SMALLINT NOT NULL CHECK (spp_billing_year BETWEEN 2000 AND 2100),

  -- NEW (opsional): tautan ke term/semester
  spp_billing_term_id     UUID,

  spp_billing_title       TEXT NOT NULL,
  spp_billing_due_date    DATE,
  spp_billing_note        TEXT,

  spp_billing_created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  spp_billing_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  spp_billing_deleted_at  TIMESTAMPTZ
);

-- FK ke academic_terms (jika belum)
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='spp_billings' AND column_name='spp_billing_term_id'
  )
  AND NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='fk_spp_billing_term'
  ) THEN
    ALTER TABLE spp_billings
      ADD CONSTRAINT fk_spp_billing_term
      FOREIGN KEY (spp_billing_term_id)
      REFERENCES academic_terms(academic_terms_id)
      ON UPDATE CASCADE ON DELETE SET NULL;
  END IF;
END$$;

-- Validasi tenant: term.mosque == billing.mosque (DEFERRABLE)
CREATE OR REPLACE FUNCTION fn_spp_term_tenant_check()
RETURNS TRIGGER AS $$
DECLARE v_term_masjid UUID;
BEGIN
  IF NEW.spp_billing_term_id IS NULL THEN
    RETURN NEW;
  END IF;

  SELECT academic_terms_masjid_id
    INTO v_term_masjid
  FROM academic_terms
  WHERE academic_terms_id = NEW.spp_billing_term_id
    AND academic_terms_deleted_at IS NULL;

  IF v_term_masjid IS NULL THEN
    RAISE EXCEPTION 'academic_term % tidak ditemukan/terhapus', NEW.spp_billing_term_id;
  END IF;

  IF v_term_masjid IS DISTINCT FROM NEW.spp_billing_masjid_id THEN
    RAISE EXCEPTION 'Masjid mismatch: term(%) != spp_billing(%)',
      v_term_masjid, NEW.spp_billing_masjid_id;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_spp_term_tenant_check') THEN
    DROP TRIGGER trg_spp_term_tenant_check ON spp_billings;
  END IF;

  CREATE CONSTRAINT TRIGGER trg_spp_term_tenant_check
  AFTER INSERT OR UPDATE OF spp_billing_masjid_id, spp_billing_term_id
  ON spp_billings
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW
  EXECUTE FUNCTION fn_spp_term_tenant_check();
END$$;

-- Unik per masjid+kelas+bulan+tahun (tetap, soft-delete aware)
DROP INDEX IF EXISTS uq_spp_billings_batch;
CREATE UNIQUE INDEX IF NOT EXISTS uq_spp_billings_batch
ON spp_billings (spp_billing_masjid_id, spp_billing_class_id, spp_billing_month, spp_billing_year)
WHERE spp_billing_deleted_at IS NULL;

-- =========================================================
-- Percepatan Search spp_billings
-- =========================================================

-- Daftar live per tenant + terbaru
CREATE INDEX IF NOT EXISTS ix_spp_billings_tenant_created_live
  ON spp_billings (spp_billing_masjid_id, spp_billing_created_at DESC)
  WHERE spp_billing_deleted_at IS NULL;

-- Navigasi per class & due_date (live)
CREATE INDEX IF NOT EXISTS ix_spp_billings_tenant_class_due_live
  ON spp_billings (spp_billing_masjid_id, spp_billing_class_id, spp_billing_due_date DESC)
  WHERE spp_billing_deleted_at IS NULL;

-- Filter cepat per (tenant, month, year) lintas kelas (live)
CREATE INDEX IF NOT EXISTS ix_spp_billings_tenant_month_year_live
  ON spp_billings (spp_billing_masjid_id, spp_billing_year, spp_billing_month)
  WHERE spp_billing_deleted_at IS NULL;

-- Pencarian judul (ILIKE) dengan trigram (live)
CREATE INDEX IF NOT EXISTS gin_spp_billings_title_trgm_live
  ON spp_billings USING GIN (spp_billing_title gin_trgm_ops)
  WHERE spp_billing_deleted_at IS NULL;

-- Filter per term (live)
CREATE INDEX IF NOT EXISTS ix_spp_billings_term_live
  ON spp_billings (spp_billing_term_id)
  WHERE spp_billing_deleted_at IS NULL;

-- Indeks dasar (tetap)
CREATE INDEX IF NOT EXISTS idx_spp_billings_masjid ON spp_billings (spp_billing_masjid_id);
CREATE INDEX IF NOT EXISTS idx_spp_billings_class  ON spp_billings (spp_billing_class_id);


-- =========================================================
-- USER SPP BILLINGS (tetap)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_spp_billings (
  user_spp_billing_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_spp_billing_billing_id UUID NOT NULL REFERENCES spp_billings(spp_billing_id) ON DELETE CASCADE,
  user_spp_billing_user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
  user_spp_billing_amount_idr INT NOT NULL CHECK (user_spp_billing_amount_idr >= 0),
  user_spp_billing_status     VARCHAR(20) NOT NULL DEFAULT 'unpaid'
                              CHECK (user_spp_billing_status IN ('unpaid','paid','canceled')),
  user_spp_billing_paid_at    TIMESTAMPTZ,
  user_spp_billing_note       TEXT,
  user_spp_billing_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_spp_billing_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  user_spp_billing_deleted_at TIMESTAMPTZ
);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='uq_user_spp_billing_per_user'
  ) THEN
    ALTER TABLE user_spp_billings
      ADD CONSTRAINT uq_user_spp_billing_per_user
      UNIQUE (user_spp_billing_billing_id, user_spp_billing_user_id);
  END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_user_spp_billings_billing ON user_spp_billings (user_spp_billing_billing_id);
CREATE INDEX IF NOT EXISTS idx_user_spp_billings_user    ON user_spp_billings (user_spp_billing_user_id);


-- =========================================================
-- DONATIONS (link opsional ke USER SPP BILLINGS)
-- =========================================================
CREATE TABLE IF NOT EXISTS donations (
    donation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    donation_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    donation_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,

    donation_user_spp_billing_id UUID
      REFERENCES user_spp_billings(user_spp_billing_id) ON DELETE SET NULL,

    donation_parent_order_id VARCHAR(120),

    donation_name VARCHAR(50) NOT NULL,
    donation_amount INTEGER NOT NULL CHECK (donation_amount > 0),

    donation_amount_masjid INTEGER CHECK (donation_amount_masjid >= 0),
    donation_amount_masjidku INTEGER CHECK (donation_amount_masjidku >= 0),
    donation_amount_masjidku_to_masjid INTEGER CHECK (donation_amount_masjidku_to_masjid >= 0),
    donation_amount_masjidku_to_app INTEGER CHECK (donation_amount_masjidku_to_app >= 0),

    donation_message TEXT,

    donation_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (
        donation_status IN ('pending', 'paid', 'expired', 'canceled', 'completed')
    ),

    donation_order_id VARCHAR(100) NOT NULL UNIQUE CHECK (char_length(donation_order_id) <= 100),

    donation_target_type INT CHECK (donation_target_type IN (1, 2, 3, 4)),
    donation_target_id UUID,

    donation_payment_token TEXT,
    donation_payment_gateway VARCHAR(50) DEFAULT 'midtrans',
    donation_payment_method VARCHAR,

    donation_paid_at TIMESTAMP,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- XOR rule target umum vs SPP
    CONSTRAINT donations_target_xor_user_spp CHECK (
      (
        donation_user_spp_billing_id IS NOT NULL
        AND donation_target_type IS NULL
        AND donation_target_id IS NULL
      ) OR (
        donation_user_spp_billing_id IS NULL
        AND donation_target_type IS NOT NULL
        AND donation_target_id IS NOT NULL
      )
    ),

    -- Split tidak boleh melebihi total
    CONSTRAINT donations_split_le_total CHECK (
      COALESCE(donation_amount_masjid, 0)
    + COALESCE(donation_amount_masjidku, 0)
    + COALESCE(donation_amount_masjidku_to_masjid, 0)
    + COALESCE(donation_amount_masjidku_to_app, 0)
    <= donation_amount
    )
);

-- Index tambahan
CREATE INDEX IF NOT EXISTS idx_donations_status           ON donations (donation_status);
CREATE INDEX IF NOT EXISTS idx_donations_target_type      ON donations (donation_target_type);
CREATE INDEX IF NOT EXISTS idx_donations_target_id        ON donations (donation_target_id);
CREATE INDEX IF NOT EXISTS idx_donations_order_id_lower   ON donations (LOWER(donation_order_id));
CREATE INDEX IF NOT EXISTS idx_donations_user_id          ON donations (donation_user_id);
CREATE INDEX IF NOT EXISTS idx_donations_masjid_id        ON donations (donation_masjid_id);
CREATE INDEX IF NOT EXISTS idx_donations_user_spp_billing_id ON donations (donation_user_spp_billing_id);
CREATE INDEX IF NOT EXISTS idx_donations_parent_order_id  ON donations (donation_parent_order_id);

-- Sinkronisasi status USER SPP → 'paid' saat donasi SPP 'paid'
CREATE OR REPLACE FUNCTION donations_sync_user_spp_paid()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.donation_user_spp_billing_id IS NOT NULL
     AND NEW.donation_status = 'paid'
  THEN
    UPDATE user_spp_billings
       SET user_spp_billing_status = 'paid',
           user_spp_billing_paid_at = COALESCE(NEW.donation_paid_at, CURRENT_TIMESTAMP),
           user_spp_billing_updated_at = CURRENT_TIMESTAMP
     WHERE user_spp_billing_id = NEW.donation_user_spp_billing_id
       AND (user_spp_billing_status <> 'paid' OR user_spp_billing_status IS NULL);
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_donations_sync_user_spp_paid ON donations;
CREATE TRIGGER trg_donations_sync_user_spp_paid
AFTER INSERT OR UPDATE OF donation_status ON donations
FOR EACH ROW EXECUTE FUNCTION donations_sync_user_spp_paid();


-- =========================================================
-- Likes
-- =========================================================
CREATE TABLE IF NOT EXISTS donation_likes (
  donation_like_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  donation_like_is_liked BOOLEAN DEFAULT TRUE,
  donation_like_donation_id UUID NOT NULL REFERENCES donations(donation_id) ON DELETE CASCADE,
  donation_like_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  donation_like_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  donation_like_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  CONSTRAINT unique_donation_like UNIQUE (donation_like_donation_id, donation_like_user_id)
);

CREATE INDEX IF NOT EXISTS idx_donation_likes_donation_id ON donation_likes(donation_like_donation_id);
CREATE INDEX IF NOT EXISTS idx_donation_likes_user_id     ON donation_likes(donation_like_user_id);
CREATE INDEX IF NOT EXISTS idx_donation_likes_updated_at  ON donation_likes(donation_like_updated_at);