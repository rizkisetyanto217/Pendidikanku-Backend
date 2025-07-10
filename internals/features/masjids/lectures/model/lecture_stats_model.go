package model

import (
	"time"

	"github.com/google/uuid"
)

type LectureStatsModel struct {
	LectureStatsID                uuid.UUID `gorm:"column:lecture_stats_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_stats_id"`
	LectureStatsLectureID         uuid.UUID `gorm:"column:lecture_stats_lecture_id;type:uuid;not null" json:"lecture_stats_lecture_id"`
	LectureStatsTotalParticipants int       `gorm:"column:lecture_stats_total_participants;default:0" json:"lecture_stats_total_participants"`
	LectureStatsAverageGrade      float64   `gorm:"column:lecture_stats_average_grade;default:0" json:"lecture_stats_average_grade"`
	LectureStatsUpdatedAt         time.Time `gorm:"column:lecture_stats_updated_at;autoUpdateTime" json:"lecture_stats_updated_at"`
}

// TableName overrides the table name for GORM
func (LectureStatsModel) TableName() string {
	return "lecture_stats"
}
