

-- ============================ --
-- TABLE MASJID SERVICE SUBSCRIPTIONS
-- ============================ --
CREATE TABLE IF NOT EXISTS masjid_service_subscriptions (
  masjid_service_subscription_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  masjid_service_subscription_masjid_id UUID NOT NULL
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  masjid_service_subscription_plan_id UUID NOT NULL
    REFERENCES masjid_service_plans(masjid_service_plan_id) ON DELETE RESTRICT,

  masjid_service_subscription_status masjid_subscription_status_enum NOT NULL DEFAULT 'active',
  masjid_service_subscription_is_auto_renew BOOLEAN NOT NULL DEFAULT FALSE,

  masjid_service_subscription_start_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_subscription_end_at   TIMESTAMPTZ,
  masjid_service_subscription_trial_end_at TIMESTAMPTZ,

  -- Snapshot harga saat checkout
  masjid_service_subscription_price_monthly NUMERIC(12,2),
  masjid_service_subscription_price_yearly  NUMERIC(12,2),

  -- Metadata billing
  masjid_service_subscription_provider        VARCHAR(40),
  masjid_service_subscription_provider_ref_id VARCHAR(100),
  masjid_service_subscription_canceled_at     TIMESTAMPTZ,

  -- Override kuota (NULL = ikut plan)
  masjid_service_subscription_max_teachers_override      INT,
  masjid_service_subscription_max_students_override      INT,
  masjid_service_subscription_max_storage_mb_override    INT,
  masjid_service_subscription_max_custom_themes_override INT,

  masjid_service_subscription_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_subscription_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  masjid_service_subscription_deleted_at TIMESTAMPTZ,

  -- Konsistensi
  CONSTRAINT ck_mss_time_order CHECK (
    masjid_service_subscription_end_at IS NULL
    OR masjid_service_subscription_end_at >= masjid_service_subscription_start_at
  ),
  CONSTRAINT ck_mss_overrides_nonneg CHECK (
    (masjid_service_subscription_max_teachers_override      IS NULL OR masjid_service_subscription_max_teachers_override      >= 0) AND
    (masjid_service_subscription_max_students_override      IS NULL OR masjid_service_subscription_max_students_override      >= 0) AND
    (masjid_service_subscription_max_storage_mb_override    IS NULL OR masjid_service_subscription_max_storage_mb_override    >= 0) AND
    (masjid_service_subscription_max_custom_themes_override IS NULL OR masjid_service_subscription_max_custom_themes_override >= 0)
  ),

  -- Generated column: window periode (untuk query overlap/now())
  masjid_service_subscription_period tstzrange
    GENERATED ALWAYS AS (
      tstzrange(
        masjid_service_subscription_start_at,
        COALESCE(masjid_service_subscription_end_at, 'infinity'::timestamptz),
        '[)'
      )
    ) STORED
);

-- Indexes: subscriptions
-- Maks. 1 langganan "current" (end_at IS NULL) per masjid
CREATE UNIQUE INDEX IF NOT EXISTS uq_mss_masjid_current_alive
  ON masjid_service_subscriptions (masjid_service_subscription_masjid_id)
  WHERE masjid_service_subscription_deleted_at IS NULL
    AND masjid_service_subscription_end_at IS NULL;

-- Lookup umum
CREATE INDEX IF NOT EXISTS idx_mss_masjid_alive
  ON masjid_service_subscriptions (masjid_service_subscription_masjid_id)
  WHERE masjid_service_subscription_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mss_plan_alive
  ON masjid_service_subscriptions (masjid_service_subscription_plan_id)
  WHERE masjid_service_subscription_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mss_status_alive
  ON masjid_service_subscriptions (masjid_service_subscription_status)
  WHERE masjid_service_subscription_deleted_at IS NULL;

-- Performa query waktu-kini / window
CREATE INDEX IF NOT EXISTS idx_mss_current_window
  ON masjid_service_subscriptions (masjid_service_subscription_masjid_id, masjid_service_subscription_start_at DESC)
  WHERE masjid_service_subscription_deleted_at IS NULL;

-- Range index untuk period (bisa dipakai cek "NOW() ∈ period")
CREATE INDEX IF NOT EXISTS gist_mss_period
  ON masjid_service_subscriptions
  USING gist (masjid_service_subscription_period)
  WHERE masjid_service_subscription_deleted_at IS NULL;

-- Yang mau/sudah habis (monitoring/cron)
CREATE INDEX IF NOT EXISTS idx_mss_end_at_alive
  ON masjid_service_subscriptions (masjid_service_subscription_end_at)
  WHERE masjid_service_subscription_deleted_at IS NULL;

-- Provider ref unik (hindari duplikasi webhook)
CREATE UNIQUE INDEX IF NOT EXISTS ux_mss_provider_ref_alive
  ON masjid_service_subscriptions (masjid_service_subscription_provider, masjid_service_subscription_provider_ref_id)
  WHERE masjid_service_subscription_deleted_at IS NULL
    AND masjid_service_subscription_provider IS NOT NULL
    AND masjid_service_subscription_provider_ref_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_mss_created_at
  ON masjid_service_subscriptions USING brin (masjid_service_subscription_created_at);

CREATE INDEX IF NOT EXISTS brin_mss_updated_at
  ON masjid_service_subscriptions USING brin (masjid_service_subscription_updated_at);




-- ============================ --
-- TABLE USER SERVICE SUBSCRIPTIONS (riwayat)
-- ============================ --
CREATE TABLE IF NOT EXISTS user_service_subscriptions (
  user_service_subscription_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_service_subscription_user_id UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  user_service_subscription_plan_id UUID NOT NULL
    REFERENCES user_service_plans(user_service_plan_id) ON DELETE RESTRICT,

  user_service_subscription_status user_subscription_status_enum NOT NULL DEFAULT 'active',
  user_service_subscription_is_auto_renew BOOLEAN NOT NULL DEFAULT FALSE,

  user_service_subscription_start_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_service_subscription_end_at   TIMESTAMPTZ,      -- NULL = current/berjalan
  user_service_subscription_trial_end_at TIMESTAMPTZ,

  -- Snapshot harga saat checkout
  user_service_subscription_price_monthly NUMERIC(12,2),
  user_service_subscription_price_yearly  NUMERIC(12,2),

  -- Metadata billing
  user_service_subscription_provider        VARCHAR(40),
  user_service_subscription_provider_ref_id VARCHAR(100),
  user_service_subscription_canceled_at     TIMESTAMPTZ,

  -- Overrides (NULL = ikut plan)
  user_service_subscription_max_masjids_owned_override   INT,
  user_service_subscription_max_storage_mb_override      INT,
  user_service_subscription_max_custom_themes_override   INT,

  user_service_subscription_created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_service_subscription_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_service_subscription_deleted_at TIMESTAMPTZ,

  -- Konsistensi
  CONSTRAINT ck_uss_time_order CHECK (
    user_service_subscription_end_at IS NULL
    OR user_service_subscription_end_at >= user_service_subscription_start_at
  ),
  CONSTRAINT ck_uss_overrides_nonneg CHECK (
    (user_service_subscription_max_masjids_owned_override IS NULL OR user_service_subscription_max_masjids_owned_override >= 0) AND
    (user_service_subscription_max_storage_mb_override    IS NULL OR user_service_subscription_max_storage_mb_override    >= 0) AND
    (user_service_subscription_max_custom_themes_override IS NULL OR user_service_subscription_max_custom_themes_override >= 0)
  ),

  -- Generated column: window periode (untuk anti-overlap & cek NOW() ∈ period)
  user_service_subscription_period tstzrange
    GENERATED ALWAYS AS (
      tstzrange(
        user_service_subscription_start_at,
        COALESCE(user_service_subscription_end_at, 'infinity'::timestamptz),
        '[)'
      )
    ) STORED
);

-- Index subscriptions (selaras dgn masjid)
-- Maks. 1 langganan "current" (end_at IS NULL) per user
CREATE UNIQUE INDEX IF NOT EXISTS uq_uss_user_current_alive
  ON user_service_subscriptions (user_service_subscription_user_id)
  WHERE user_service_subscription_deleted_at IS NULL
    AND user_service_subscription_end_at IS NULL;

-- Lookup umum
CREATE INDEX IF NOT EXISTS idx_uss_user_alive
  ON user_service_subscriptions (user_service_subscription_user_id)
  WHERE user_service_subscription_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uss_plan_alive
  ON user_service_subscriptions (user_service_subscription_plan_id)
  WHERE user_service_subscription_deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_uss_status_alive
  ON user_service_subscriptions (user_service_subscription_status)
  WHERE user_service_subscription_deleted_at IS NULL;

-- Performa query waktu-kini / window
CREATE INDEX IF NOT EXISTS idx_uss_current_window
  ON user_service_subscriptions (user_service_subscription_user_id, user_service_subscription_start_at DESC)
  WHERE user_service_subscription_deleted_at IS NULL;

-- Range index utk period (aksellerasi cek overlap/now-in-range)
CREATE INDEX IF NOT EXISTS gist_uss_period
  ON user_service_subscriptions
  USING gist (user_service_subscription_period)
  WHERE user_service_subscription_deleted_at IS NULL;

-- Yang akan/sudah habis
CREATE INDEX IF NOT EXISTS idx_uss_end_at_alive
  ON user_service_subscriptions (user_service_subscription_end_at)
  WHERE user_service_subscription_deleted_at IS NULL;

-- Provider ref unik (hindari duplikasi webhook/charge)
CREATE UNIQUE INDEX IF NOT EXISTS ux_uss_provider_ref_alive
  ON user_service_subscriptions (user_service_subscription_provider, user_service_subscription_provider_ref_id)
  WHERE user_service_subscription_deleted_at IS NULL
    AND user_service_subscription_provider IS NOT NULL
    AND user_service_subscription_provider_ref_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS brin_uss_created_at
  ON user_service_subscriptions USING brin (user_service_subscription_created_at);

CREATE INDEX IF NOT EXISTS brin_uss_updated_at
  ON user_service_subscriptions USING brin (user_service_subscription_updated_at);
