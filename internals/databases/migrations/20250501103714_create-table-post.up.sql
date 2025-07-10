CREATE TABLE IF NOT EXISTS posts (
  post_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_title VARCHAR(255) NOT NULL,
  post_content TEXT NOT NULL,
  post_image_url TEXT,
  post_is_published BOOLEAN DEFAULT FALSE,
  post_type VARCHAR(50) DEFAULT 'text',
  post_masjid_id UUID REFERENCES masjids(masjid_id) ON DELETE CASCADE,
  post_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  post_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  post_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  post_deleted_at TIMESTAMP
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_posts_masjid_id ON posts(post_masjid_id);
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(post_user_id);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(post_created_at);
CREATE INDEX IF NOT EXISTS idx_posts_deleted_at ON posts(post_deleted_at);


CREATE TABLE IF NOT EXISTS post_likes (
  post_like_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_like_is_liked BOOLEAN DEFAULT TRUE,
  post_like_post_id UUID NOT NULL REFERENCES posts(post_id) ON DELETE CASCADE,
  post_like_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  post_like_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT unique_post_like UNIQUE (post_like_post_id, post_like_user_id)
);

-- Indexing
CREATE INDEX IF NOT EXISTS idx_post_likes_post_id ON post_likes(post_like_post_id);
CREATE INDEX IF NOT EXISTS idx_post_likes_user_id ON post_likes(post_like_user_id);
CREATE INDEX IF NOT EXISTS idx_post_likes_updated_at ON post_likes(post_like_updated_at);
