package model

import (
	// LectureSessionModel "masjidku_backend/internals/features/masjids/lectures_sessions/lecture_sessions/model"
	"time"
)

type LectureSessionsMaterialModel struct {
	LectureSessionsMaterialID             string    `gorm:"column:lecture_sessions_material_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	LectureSessionsMaterialTitle          string    `gorm:"column:lecture_sessions_material_title;type:varchar(255);not null"`
	LectureSessionsMaterialSummary        string    `gorm:"column:lecture_sessions_material_summary;type:text"`
	LectureSessionsMaterialTranscriptFull string    `gorm:"column:lecture_sessions_material_transcript_full;type:text"`
	LectureSessionsMaterialLectureSessionID string  `gorm:"column:lecture_sessions_material_lecture_session_id;type:uuid;not null"`
	LectureSessionsMaterialCreatedAt      time.Time `gorm:"column:lecture_sessions_material_created_at;autoCreateTime"`

	// Relations
	// Session *LectureSessionModel.LectureSessionModel `gorm:"foreignKey:LectureSessionsMaterialSessionID"`
}

func (LectureSessionsMaterialModel) TableName() string {
	return "lecture_sessions_materials"
}
