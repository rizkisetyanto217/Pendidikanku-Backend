CREATE TABLE IF NOT EXISTS events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_title VARCHAR(255) NOT NULL,
    event_slug VARCHAR(100) NOT NULL,
    event_description TEXT,
    event_location VARCHAR(255),
    event_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    event_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_event_masjid_id ON events(event_masjid_id);
CREATE INDEX IF NOT EXISTS idx_event_slug ON events(event_slug);


CREATE TABLE IF NOT EXISTS event_sessions (
    event_session_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_session_event_id UUID NOT NULL REFERENCES events(event_id) ON DELETE CASCADE,
    event_session_title VARCHAR(255) NOT NULL,
    event_session_description TEXT,
    event_session_start_time TIMESTAMP NOT NULL,
    event_session_end_time TIMESTAMP NOT NULL,
    event_session_location VARCHAR(255),
    event_session_image_url TEXT,
    event_session_capacity INT,
    event_session_is_public BOOLEAN DEFAULT TRUE,
    event_session_is_registration_required BOOLEAN DEFAULT FALSE,
    event_session_masjid_id UUID NOT NULL REFERENCES masjids(masjid_id) ON DELETE CASCADE,
    event_session_created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    event_session_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    event_session_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_event_sessions_event_id ON event_sessions(event_session_event_id);
CREATE INDEX IF NOT EXISTS idx_event_sessions_start_time ON event_sessions(event_session_start_time);



CREATE TABLE IF NOT EXISTS user_event_registrations (
    user_event_registration_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_event_registration_event_session_id UUID NOT NULL REFERENCES event_sessions(event_session_id) ON DELETE CASCADE,
    user_event_registration_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_event_registration_status VARCHAR(50) DEFAULT 'registered',
    user_event_registration_registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_event_registration_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_event_registration_event_session_id, user_event_registration_user_id)
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_user_event_registrations_event_session_id ON user_event_registrations(user_event_registration_event_session_id);
CREATE INDEX IF NOT EXISTS idx_user_event_registrations_user_id ON user_event_registrations(user_event_registration_user_id);
