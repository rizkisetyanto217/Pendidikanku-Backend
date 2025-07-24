-- Tabel sertifikat utama
CREATE TABLE IF NOT EXISTS certificates (
    certificate_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    certificate_title TEXT NOT NULL,
    certificate_description TEXT,
    certificate_lecture_id UUID NOT NULL REFERENCES lectures(lecture_id) ON DELETE CASCADE,
    certificate_template_url TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk pencarian berdasarkan lecture
CREATE INDEX IF NOT EXISTS idx_certificates_lecture_id ON certificates(certificate_lecture_id);


-- Tabel user yang menerima sertifikat
CREATE TABLE IF NOT EXISTS user_certificates (
    user_cert_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_cert_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    user_cert_certificate_id UUID NOT NULL REFERENCES certificates(certificate_id) ON DELETE CASCADE,

    user_cert_score INTEGER,
    user_cert_slug_url TEXT UNIQUE NOT NULL,
    user_cert_is_up_to_date BOOLEAN NOT NULL DEFAULT true,
    user_cert_issued_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk pencarian efisien
CREATE INDEX IF NOT EXISTS idx_user_cert_user_id ON user_certificates(user_cert_user_id);
CREATE INDEX IF NOT EXISTS idx_user_cert_certificate_id ON user_certificates(user_cert_certificate_id);
