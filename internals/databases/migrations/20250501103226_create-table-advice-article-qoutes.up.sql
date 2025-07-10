CREATE TABLE IF NOT EXISTS advices (
  advice_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  advice_description TEXT NOT NULL,
  advice_lecture_id UUID REFERENCES lectures(lecture_id) ON DELETE SET NULL,
  advice_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  advice_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexing (disarankan untuk filter dan relasi)
CREATE INDEX IF NOT EXISTS idx_advices_user_id ON advices(advice_user_id);
CREATE INDEX IF NOT EXISTS idx_advices_lecture_id ON advices(advice_lecture_id);
CREATE INDEX IF NOT EXISTS idx_advices_created_at ON advices(advice_created_at);


CREATE TABLE IF NOT EXISTS articles (
  article_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  article_title VARCHAR(255) NOT NULL,
  article_description TEXT NOT NULL,
  article_image_url TEXT,
  article_order_id INT,
  article_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  article_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Untuk urutan tampilan artikel
CREATE INDEX IF NOT EXISTS idx_articles_order_id ON articles(article_order_id);

-- Untuk pencarian artikel berdasarkan waktu
CREATE INDEX IF NOT EXISTS idx_articles_created_at ON articles(article_created_at);
CREATE INDEX IF NOT EXISTS idx_articles_updated_at ON articles(article_updated_at);


CREATE TABLE IF NOT EXISTS carousels (
  carousel_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  carousel_title VARCHAR(255),
  carousel_caption TEXT,
  carousel_image_url TEXT NOT NULL,
  carousel_target_url TEXT, -- optional: bisa link ke /artikel/:id atau /event/:id
  carousel_type VARCHAR(50), -- 'artikel', 'event', 'pengumuman', dsb
  carousel_article_id UUID, -- jika carousel ini terkait artikel
  carousel_order INT,
  carousel_is_active BOOLEAN DEFAULT TRUE,
  carousel_created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  carousel_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT fk_carousel_article FOREIGN KEY (carousel_article_id) REFERENCES articles(article_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_carousel_order ON carousels(carousel_order);
CREATE INDEX IF NOT EXISTS idx_carousel_active ON carousels(carousel_is_active);


CREATE TABLE IF NOT EXISTS quotes (
  quote_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  quote_text TEXT NOT NULL,
  is_published BOOLEAN DEFAULT FALSE,
  display_order INT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk performa filtering dan penampilan
CREATE INDEX IF NOT EXISTS idx_quotes_display_order ON quotes(display_order);
CREATE INDEX IF NOT EXISTS idx_quotes_created_at ON quotes(created_at);
CREATE INDEX IF NOT EXISTS idx_quotes_is_published ON quotes(is_published);