http://localhost:3000/auth/google/callback?state=gEMUU22aaiiqqyyGGOOWW444cckkssAA&code=4%2F0AQSTgQF_bln2oJIrAnvddqhoTJt4nRUvY8sIaoPeL__quimiIqnBDCDZCZjBsSw2_r3Gmg&scope=email+profile+openid+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.profile+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.email&authuser=0&prompt=consent


-- slug unik (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_masjids_slug_ci
  ON masjids (LOWER(masjid_slug))
  WHERE masjid_deleted_at IS NULL;

-- domain unik (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_masjids_domain_ci
  ON masjids (LOWER(masjid_domain))
  WHERE masjid_domain IS NOT NULL
    AND masjid_deleted_at IS NULL;

-- pk alive
CREATE INDEX IF NOT EXISTS idx_masjids_id_alive
  ON masjids (masjid_id) WHERE masjid_deleted_at IS NULL;
