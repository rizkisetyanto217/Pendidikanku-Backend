CREATE TABLE IF NOT EXISTS user_certificates (
    user_cert_id SERIAL PRIMARY KEY,
    user_cert_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_cert_subcategory_id INTEGER NOT NULL REFERENCES subcategories(subcategory_id) ON DELETE CASCADE,

    user_cert_is_up_to_date BOOLEAN NOT NULL DEFAULT true,
    user_cert_slug_url TEXT UNIQUE NOT NULL,

    user_cert_issued_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS certificate_versions (
    cert_versions_id SERIAL PRIMARY KEY,
    cert_versions_subcategory_id INTEGER NOT NULL REFERENCES subcategories(subcategory_id) ON DELETE CASCADE,

    cert_versions_number INTEGER NOT NULL,
    cert_versions_total_themes INTEGER NOT NULL DEFAULT 0,
    cert_versions_note TEXT,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE
);