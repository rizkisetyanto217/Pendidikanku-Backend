-- Buat table users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_name VARCHAR(50) NOT NULL CHECK (LENGTH(user_name) >= 3 AND LENGTH(user_name) <= 50),
    email VARCHAR(255) UNIQUE NOT NULL CHECK (POSITION('@' IN email) > 1),
    password VARCHAR(250),
    google_id VARCHAR(255) UNIQUE,
    role VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (role IN ('owner', 'user', 'teacher', 'treasurer', 'admin')),
    security_question TEXT NOT NULL,
    security_answer VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Buat table users_profile
CREATE TABLE IF NOT EXISTS users_profile (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    donation_name VARCHAR(50),
    full_name VARCHAR(50),
    date_of_birth DATE,
    gender VARCHAR(10) CHECK (gender IN ('male', 'female')),
    phone_number VARCHAR(20),
    bio VARCHAR(300),
    location VARCHAR(50),
    occupation VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- ✅ Index tambahan supaya query join dan lookup cepat
CREATE INDEX IF NOT EXISTS idx_users_profile_user_id ON users_profile(user_id);