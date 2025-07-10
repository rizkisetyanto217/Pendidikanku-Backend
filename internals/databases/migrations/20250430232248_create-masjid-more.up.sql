CREATE TABLE IF NOT EXISTS masjid_profile_teacher_dkm (
    masjid_profile_teacher_dkm_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_profile_teacher_dkm_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    masjid_profile_teacher_dkm_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    masjid_profile_teacher_dkm_name VARCHAR(100) NOT NULL,
    masjid_profile_teacher_dkm_role VARCHAR(100) NOT NULL,
    masjid_profile_teacher_dkm_description TEXT,
    masjid_profile_teacher_dkm_message TEXT,
    masjid_profile_teacher_dkm_image_url TEXT,
    masjid_profile_teacher_dkm_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS masjid_tags (
    masjid_tag_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_tag_name VARCHAR(50) NOT NULL,
    masjid_tag_description TEXT,
    masjid_tag_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS masjid_tag_relations  (
    masjid_tag_relation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    masjid_tag_relation_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    masjid_tag_relation_tag_id UUID NOT NULL REFERENCES masjid_tags(masjid_tag_id) ON DELETE CASCADE,
    masjid_tag_relation_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(masjid_tag_relation_masjid_id, masjid_tag_relation_tag_id)
);


-- Index untuk pencarian cepat profil DKM/pengajar berdasarkan masjid
CREATE INDEX IF NOT EXISTS idx_profile_teacher_dkm_masjid_id
ON masjid_profile_teacher_dkm(masjid_profile_teacher_dkm_masjid_id);


-- Untuk tag relations
CREATE INDEX IF NOT EXISTS idx_tag_relations_masjid_id ON masjid_tag_relations(masjid_tag_relation_masjid_id);
CREATE INDEX IF NOT EXISTS idx_tag_relations_tag_id ON masjid_tag_relations(masjid_tag_relation_tag_id);
