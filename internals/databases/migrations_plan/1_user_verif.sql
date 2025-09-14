-- =========================================================
-- PRASYARAT
-- =========================================================
CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;   -- tipe CITEXT

-- =========================================================
-- 1) USER_EMAIL_VERIFICATIONS
-- =========================================================
CREATE TABLE IF NOT EXISTS user_email_verifications (
  user_email_verification_id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_email_verification_user_id              UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  user_email_verification_email                CITEXT NOT NULL,
  user_email_verification_previous_email       CITEXT,

  user_email_verification_channel              VARCHAR(20) NOT NULL
    CHECK (user_email_verification_channel IN ('email','magic_link','otp')),
  user_email_verification_purpose              VARCHAR(30),       -- signup/change_email/signin_link/reactivation
  user_email_verification_code                 VARCHAR(16),       -- OTP (opsional)
  user_email_verification_token                TEXT NOT NULL,     -- HASH token

  user_email_verification_attempt_count        INT NOT NULL DEFAULT 0,
  user_email_verification_resend_count         INT NOT NULL DEFAULT 0,
  user_email_verification_last_attempt_at      TIMESTAMPTZ,
  user_email_verification_throttle_until       TIMESTAMPTZ,

  user_email_verification_delivery_status      VARCHAR(20)
    CHECK (user_email_verification_delivery_status IN ('queued','sent','failed')),
  user_email_verification_delivery_error       TEXT,
  user_email_verification_delivered_at         TIMESTAMPTZ,

  user_email_verification_sent_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_email_verification_expires_at           TIMESTAMPTZ,
  user_email_verification_clicked_at           TIMESTAMPTZ,       -- magic-link click ts
  user_email_verification_verified_at          TIMESTAMPTZ,
  user_email_verification_invalidated_at       TIMESTAMPTZ,
  user_email_verification_invalidated_reason   VARCHAR(80),

  user_email_verification_request_ip           INET,
  user_email_verification_user_agent           TEXT,
  user_email_verification_request_id           UUID,

  user_email_verification_verified_ip          INET,
  user_email_verification_verified_user_agent  TEXT,
  user_email_verification_verified_session_id  UUID,

  user_email_verification_provider_message_id  TEXT,
  user_email_verification_provider_payload     JSONB,
  user_email_verification_redirect_url         TEXT,
  user_email_verification_locale               VARCHAR(10),
  user_email_verification_client_id            VARCHAR(40),

  user_email_verification_risk_score           SMALLINT,
  user_email_verification_risk_flags           TEXT,

  user_email_verification_created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_email_verification_deleted_at           TIMESTAMPTZ,

  CONSTRAINT chk_u_ev_risk_score
    CHECK (user_email_verification_risk_score IS NULL OR user_email_verification_risk_score BETWEEN 0 AND 100)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_u_ev_user_id             ON user_email_verifications (user_email_verification_user_id);
CREATE INDEX IF NOT EXISTS idx_u_ev_request_id          ON user_email_verifications (user_email_verification_request_id);
CREATE INDEX IF NOT EXISTS idx_u_ev_created_at          ON user_email_verifications (user_email_verification_created_at);
CREATE INDEX IF NOT EXISTS idx_u_ev_throttle_until      ON user_email_verifications (user_email_verification_throttle_until);
CREATE INDEX IF NOT EXISTS idx_u_ev_expires_at          ON user_email_verifications (user_email_verification_expires_at);
CREATE INDEX IF NOT EXISTS idx_u_ev_verified_at         ON user_email_verifications (user_email_verification_verified_at);
CREATE INDEX IF NOT EXISTS idx_u_ev_clicked_at          ON user_email_verifications (user_email_verification_clicked_at);
CREATE INDEX IF NOT EXISTS idx_u_ev_delivered_at        ON user_email_verifications (user_email_verification_delivered_at);

-- Unik untuk token AKTIF (belum verified & belum invalidated)
CREATE UNIQUE INDEX IF NOT EXISTS uq_u_ev_token_active
  ON user_email_verifications (user_email_verification_token)
  WHERE user_email_verification_verified_at IS NULL
    AND user_email_verification_invalidated_at IS NULL;

-- Satu verifikasi aktif per (user, channel, purpose)
CREATE UNIQUE INDEX IF NOT EXISTS uq_u_ev_active_per_user_channel_purpose
  ON user_email_verifications (user_email_verification_user_id,
                               user_email_verification_channel,
                               user_email_verification_purpose)
  WHERE user_email_verification_verified_at IS NULL
    AND user_email_verification_invalidated_at IS NULL;

-- Lookup cepat OTP aktif
CREATE INDEX IF NOT EXISTS idx_u_ev_code_active
  ON user_email_verifications (user_email_verification_code)
  WHERE user_email_verification_verified_at IS NULL
    AND user_email_verification_invalidated_at IS NULL
    AND user_email_verification_code IS NOT NULL;

-- =========================================================
-- 2) USER_PASSWORD_RESETS
-- =========================================================
CREATE TABLE IF NOT EXISTS user_password_resets (
  user_password_reset_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_password_reset_user_id           UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  user_password_reset_email             CITEXT,
  user_password_reset_method            VARCHAR(20) NOT NULL
    CHECK (user_password_reset_method IN ('email','otp','admin')),

  user_password_reset_code              VARCHAR(16),
  user_password_reset_token             TEXT NOT NULL,   -- HASH token

  user_password_reset_attempt_count     INT NOT NULL DEFAULT 0,
  user_password_reset_last_attempt_at   TIMESTAMPTZ,
  user_password_reset_throttle_until    TIMESTAMPTZ,

  user_password_reset_delivery_status   VARCHAR(20)
    CHECK (user_password_reset_delivery_status IN ('queued','sent','failed')),
  user_password_reset_delivery_error    TEXT,
  user_password_reset_delivered_at      TIMESTAMPTZ,

  user_password_reset_requested_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_password_reset_expires_at        TIMESTAMPTZ,
  user_password_reset_used_at           TIMESTAMPTZ,
  user_password_reset_invalidated_at    TIMESTAMPTZ,
  user_password_reset_invalidated_reason VARCHAR(80),

  user_password_reset_request_ip        INET,
  user_password_reset_request_user_agent TEXT,
  user_password_reset_consumed_ip       INET,
  user_password_reset_consumed_user_agent TEXT,

  user_password_reset_request_id        UUID,
  user_password_reset_redirect_url      TEXT,

  user_password_reset_used_by_session_id UUID,
  user_password_reset_client_id         VARCHAR(40),

  user_password_reset_password_policy_version SMALLINT,
  user_password_reset_created_by_admin_user_id UUID REFERENCES users(id) ON DELETE SET NULL,

  user_password_reset_provider_payload  JSONB,
  user_password_reset_risk_score        SMALLINT,
  user_password_reset_risk_flags        TEXT,
  user_password_reset_password_strength_score SMALLINT,

  user_password_reset_created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_password_reset_deleted_at        TIMESTAMPTZ,

  CONSTRAINT chk_u_pr_risk_score
    CHECK (user_password_reset_risk_score IS NULL OR user_password_reset_risk_score BETWEEN 0 AND 100),
  CONSTRAINT chk_u_pr_strength_score
    CHECK (user_password_reset_password_strength_score IS NULL OR user_password_reset_password_strength_score BETWEEN 0 AND 4)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_u_pr_user_id         ON user_password_resets (user_password_reset_user_id);
CREATE INDEX IF NOT EXISTS idx_u_pr_request_id      ON user_password_resets (user_password_reset_request_id);
CREATE INDEX IF NOT EXISTS idx_u_pr_created_at      ON user_password_resets (user_password_reset_created_at);
CREATE INDEX IF NOT EXISTS idx_u_pr_throttle_until  ON user_password_resets (user_password_reset_throttle_until);
CREATE INDEX IF NOT EXISTS idx_u_pr_expires_at      ON user_password_resets (user_password_reset_expires_at);
CREATE INDEX IF NOT EXISTS idx_u_pr_used_at         ON user_password_resets (user_password_reset_used_at);
CREATE INDEX IF NOT EXISTS idx_u_pr_delivered_at    ON user_password_resets (user_password_reset_delivered_at);

-- Unik untuk token AKTIF
CREATE UNIQUE INDEX IF NOT EXISTS uq_u_pr_token_active
  ON user_password_resets (user_password_reset_token)
  WHERE user_password_reset_used_at IS NULL
    AND user_password_reset_invalidated_at IS NULL;

-- Satu reset aktif per (user, method)
CREATE UNIQUE INDEX IF NOT EXISTS uq_u_pr_active_per_user_method
  ON user_password_resets (user_password_reset_user_id, user_password_reset_method)
  WHERE user_password_reset_used_at IS NULL
    AND user_password_reset_invalidated_at IS NULL;

-- Lookup cepat OTP aktif
CREATE INDEX IF NOT EXISTS idx_u_pr_code_active
  ON user_password_resets (user_password_reset_code)
  WHERE user_password_reset_used_at IS NULL
    AND user_password_reset_invalidated_at IS NULL
    AND user_password_reset_code IS NOT NULL;

-- =========================================================
-- 3) USER_SECURITY  (satu baris per user)
-- =========================================================
CREATE TABLE IF NOT EXISTS user_security (
  user_security_id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_security_user_id                       UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  user_security_two_factor_enabled            BOOLEAN NOT NULL DEFAULT FALSE,
  user_security_two_factor_secret             TEXT,
  user_security_two_factor_methods            TEXT,   -- csv/json string
  user_security_backup_codes_hashes           TEXT,   -- HASH daftar backup codes
  user_security_preferred_2fa_method          VARCHAR(20)
    CHECK (user_security_preferred_2fa_method IN ('totp','sms','email','webauthn','backup_code','none')),

  user_security_passwordless_enabled          BOOLEAN NOT NULL DEFAULT FALSE,

  user_security_last_password_changed_at      TIMESTAMPTZ,
  user_security_last_login_at                 TIMESTAMPTZ,
  user_security_last_login_ip                 INET,
  user_security_failed_login_attempts         INT NOT NULL DEFAULT 0,
  user_security_locked_until                  TIMESTAMPTZ,

  user_security_require_password_change       BOOLEAN NOT NULL DEFAULT FALSE,
  user_security_session_invalidated_at        TIMESTAMPTZ,

  user_security_password_expires_at           TIMESTAMPTZ,
  user_security_mfa_enrolled_at               TIMESTAMPTZ,
  user_security_mfa_last_challenge_at         TIMESTAMPTZ,
  user_security_compromised_password_detected_at TIMESTAMPTZ,

  user_security_allow_new_device_without_2fa  BOOLEAN NOT NULL DEFAULT FALSE,
  user_security_enforce_webauthn_only         BOOLEAN NOT NULL DEFAULT FALSE,
  user_security_login_alert_enabled           BOOLEAN NOT NULL DEFAULT FALSE,

  user_security_password_policy_version       SMALLINT,
  user_security_password_pwned_checked_at     TIMESTAMPTZ,
  user_security_password_pwned_found          BOOLEAN NOT NULL DEFAULT FALSE,
  user_security_backup_codes_rotated_at       TIMESTAMPTZ,

  user_security_max_active_sessions           SMALLINT,
  user_security_session_idle_timeout_minutes  SMALLINT,
  CONSTRAINT chk_user_security_max_active_sessions
    CHECK (user_security_max_active_sessions IS NULL OR user_security_max_active_sessions BETWEEN 0 AND 50),
  CONSTRAINT chk_user_security_idle_timeout
    CHECK (user_security_session_idle_timeout_minutes IS NULL OR user_security_session_idle_timeout_minutes BETWEEN 1 AND 10080),

  user_security_created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_security_updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Satu baris per user
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_security_user
  ON user_security (user_security_user_id);

-- =========================================================
-- 4) USER_AUDIT
-- =========================================================
CREATE TABLE IF NOT EXISTS user_audit (
  user_audit_id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_audit_actor_user_id    UUID REFERENCES users(id) ON DELETE SET NULL, -- aktor (NULL = system)
  user_audit_user_id          UUID REFERENCES users(id) ON DELETE CASCADE,  -- subjek terdampak

  user_audit_event_type       VARCHAR(50)  NOT NULL,  -- login_success/login_failed/update_profile/...
  user_audit_action           VARCHAR(120),            -- human-readable
  user_audit_resource_type    VARCHAR(50),
  user_audit_resource_id      UUID,
  user_audit_result           VARCHAR(20)  CHECK (user_audit_result IN ('success','failed')),

  user_audit_ip_address       INET,
  user_audit_user_agent       TEXT,

  user_audit_http_method      VARCHAR(10),
  user_audit_request_path     TEXT,
  user_audit_http_status      SMALLINT CHECK (user_audit_http_status BETWEEN 100 AND 599),

  user_audit_request_id       UUID,
  user_audit_correlation_id   UUID,

  user_audit_masjid_id        UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,
  user_audit_actor_masjid_id  UUID REFERENCES masjids(masjid_id) ON DELETE SET NULL,

  user_audit_service          VARCHAR(40),
  user_audit_note             TEXT,
  user_audit_metadata         JSONB,

  -- Observability & origin
  user_audit_trace_id         VARCHAR(64),
  user_audit_span_id          VARCHAR(64),
  user_audit_request_origin   VARCHAR(20) CHECK (user_audit_request_origin IN ('web','mobile','api')),
  user_audit_geo_country      CHAR(2),
  user_audit_geo_city         VARCHAR(80),
  user_audit_referrer         TEXT,
  user_audit_origin_header    TEXT,
  user_audit_request_size_bytes  INT,
  user_audit_response_size_bytes INT,
  user_audit_api_version      VARCHAR(20),

  -- Normalisasi & korelasi klien
  user_audit_client_id        VARCHAR(40),
  user_audit_endpoint_key     VARCHAR(120),
  user_audit_request_hash     TEXT,
  user_audit_response_hash    TEXT,

  user_audit_created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_u_audit_created_at       ON user_audit (user_audit_created_at);
CREATE INDEX IF NOT EXISTS idx_u_audit_user_id          ON user_audit (user_audit_user_id);
CREATE INDEX IF NOT EXISTS idx_u_audit_actor_user_id    ON user_audit (user_audit_actor_user_id);
CREATE INDEX IF NOT EXISTS idx_u_audit_event_type       ON user_audit (user_audit_event_type);
CREATE INDEX IF NOT EXISTS idx_u_audit_request_id       ON user_audit (user_audit_request_id);
CREATE INDEX IF NOT EXISTS idx_u_audit_correlation_id   ON user_audit (user_audit_correlation_id);
CREATE INDEX IF NOT EXISTS idx_u_audit_masjid_id        ON user_audit (user_audit_masjid_id);
CREATE INDEX IF NOT EXISTS idx_u_audit_actor_masjid_id  ON user_audit (user_audit_actor_masjid_id);
CREATE INDEX IF NOT EXISTS idx_u_audit_trace_id         ON user_audit (user_audit_trace_id);
CREATE INDEX IF NOT EXISTS idx_u_audit_request_origin   ON user_audit (user_audit_request_origin);
CREATE INDEX IF NOT EXISTS idx_u_audit_client_endpoint  ON user_audit (user_audit_client_id, user_audit_endpoint_key);
CREATE INDEX IF NOT EXISTS idx_u_audit_api_version      ON user_audit (user_audit_api_version);

-- =========================================================
-- 5) USER_SETTINGS
-- =========================================================
CREATE TABLE IF NOT EXISTS user_settings (
  user_setting_id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  user_setting_user_id                       UUID NOT NULL
    REFERENCES users(id) ON DELETE CASCADE,

  user_setting_scope_type                    VARCHAR(20) NOT NULL DEFAULT 'global'
    CHECK (user_setting_scope_type IN ('global','masjid')),
  user_setting_scope_id                      UUID
    REFERENCES masjids(masjid_id) ON DELETE CASCADE,

  user_setting_language                      VARCHAR(10),
  user_setting_locale                        VARCHAR(10), -- ex: id-ID
  user_setting_timezone                      VARCHAR(50),
  user_setting_theme                         VARCHAR(20) NOT NULL DEFAULT 'system',

  user_setting_notifications_enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  user_setting_email_marketing_enabled       BOOLEAN NOT NULL DEFAULT FALSE,
  user_setting_email_transactional_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
  user_setting_desktop_push_enabled          BOOLEAN NOT NULL DEFAULT FALSE,
  user_setting_mobile_push_enabled           BOOLEAN NOT NULL DEFAULT FALSE,

  user_setting_week_start                    VARCHAR(10),    -- monday/sunday
  user_setting_first_day_of_week             VARCHAR(10),
  user_setting_date_format                   VARCHAR(20),
  user_setting_time_format                   VARCHAR(10),    -- 12h/24h
  user_setting_currency                      VARCHAR(10),    -- IDR

  user_setting_number_format                 VARCHAR(10),    -- 1,234.56 / 1.234,56
  user_setting_measurement_system            VARCHAR(10)
    CHECK (user_setting_measurement_system IN ('metric','imperial')),

  user_setting_notification_email_digest     VARCHAR(10)
    CHECK (user_setting_notification_email_digest IN ('instant','daily','weekly')),

  user_setting_privacy_profile_visibility    VARCHAR(20),    -- public/private/masjid
  user_setting_privacy_last_seen_visibility  VARCHAR(20),    -- public/masjid/private
  user_setting_privacy_search_visibility     VARCHAR(20),    -- public/masjid/private
  user_setting_content_filter_level          VARCHAR(20),    -- strict/moderate/off
  user_setting_experimental_features_enabled BOOLEAN NOT NULL DEFAULT FALSE,

  user_setting_accessibility_reduced_motion  BOOLEAN NOT NULL DEFAULT FALSE,
  user_setting_in_app_sound_enabled          BOOLEAN NOT NULL DEFAULT FALSE,
  user_setting_in_app_vibration_enabled      BOOLEAN NOT NULL DEFAULT FALSE,
  user_setting_text_scale                    VARCHAR(10) NOT NULL DEFAULT 'md', -- sm/md/lg

  user_setting_accent_color                  CHAR(7),
  user_setting_theme_variant                 VARCHAR(24),

  user_setting_created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_setting_updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT chk_user_setting_accent_color
    CHECK (
      user_setting_accent_color IS NULL
      OR user_setting_accent_color ~ '^#[0-9A-Fa-f]{6}$'
    )
);

-- Indexes umum
CREATE INDEX IF NOT EXISTS idx_u_settings_user_scope
  ON user_settings (user_setting_user_id, user_setting_scope_type);

-- Unik: satu baris 'global' per user
CREATE UNIQUE INDEX IF NOT EXISTS uq_u_settings_user_global
  ON user_settings (user_setting_user_id)
  WHERE user_setting_scope_type = 'global' AND user_setting_scope_id IS NULL;

-- Unik: satu baris per (user, masjid) jika scope 'masjid'
CREATE UNIQUE INDEX IF NOT EXISTS uq_u_settings_user_masjid
  ON user_settings (user_setting_user_id, user_setting_scope_id)
  WHERE user_setting_scope_type = 'masjid' AND user_setting_scope_id IS NOT NULL;
