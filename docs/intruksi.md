http://localhost:3000/auth/google/callback?state=gEMUU22aaiiqqyyGGOOWW444cckkssAA&code=4%2F0AQSTgQF_bln2oJIrAnvddqhoTJt4nRUvY8sIaoPeL\_\_quimiIqnBDCDZCZjBsSw2_r3Gmg&scope=email+profile+openid+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.profile+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.email&authuser=0&prompt=consent

-- slug unik (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_schools_slug_ci
ON schools (LOWER(school_slug))
WHERE school_deleted_at IS NULL;

-- domain unik (case-insensitive)
CREATE UNIQUE INDEX IF NOT EXISTS ux_schools_domain_ci
ON schools (LOWER(school_domain))
WHERE school_domain IS NOT NULL
AND school_deleted_at IS NULL;

-- pk alive
CREATE INDEX IF NOT EXISTS idx_schools_id_alive
ON schools (school_id) WHERE school_deleted_at IS NULL;
