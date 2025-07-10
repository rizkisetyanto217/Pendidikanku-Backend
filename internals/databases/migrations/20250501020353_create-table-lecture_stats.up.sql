CREATE TABLE IF NOT EXISTS lecture_stats (
    lecture_stats_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lecture_stats_lecture_id UUID NOT NULL REFERENCES lectures(lecture_id) ON DELETE CASCADE,
    lecture_stats_total_participants INT DEFAULT 0,
    lecture_stats_average_grade FLOAT DEFAULT 0,
    lecture_stats_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(lecture_stats_lecture_id)
);

-- Index
CREATE INDEX IF NOT EXISTS idx_lecture_stats_lecture_id ON lecture_stats(lecture_stats_lecture_id);
