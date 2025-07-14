package model

import "time"

type LectureSessionsMaterialModel struct {
	LectureSessionsMaterialID          string    `gorm:"column:lecture_sessions_material_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	LectureSessionsMaterialTitle       string    `gorm:"column:lecture_sessions_material_title;type:varchar(255);not null"`
	LectureSessionsMaterialSummary     string    `gorm:"column:lecture_sessions_material_summary;type:text"`
	LectureSessionsMaterialTranscriptFull string `gorm:"column:lecture_sessions_material_transcript_full;type:text"`
	LectureSessionsMaterialLectureSessionID string `gorm:"column:lecture_sessions_material_lecture_session_id;type:uuid;not null"`
	LectureSessionsMaterialMasjidID    string    `gorm:"column:lecture_sessions_material_masjid_id;type:uuid;not null"`
	LectureSessionsMaterialCreatedAt   time.Time `gorm:"column:lecture_sessions_material_created_at;autoCreateTime"`

	// Optional: tambahkan relasi jika diperlukan
	// LectureSession *LectureSessionModel `gorm:"foreignKey:LectureSessionsMaterialLectureSessionID"`
	// Masjid        *MasjidModel           `gorm:"foreignKey:LectureSessionsMaterialMasjidID"`
}

func (LectureSessionsMaterialModel) TableName() string {
	return "lecture_sessions_materials"
}
