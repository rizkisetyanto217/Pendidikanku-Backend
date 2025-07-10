package model

import (
	"time"
)

type LectureSessionsAssetModel struct {
	LectureSessionsAssetID               string    `gorm:"column:lecture_sessions_asset_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	LectureSessionsAssetTitle            string    `gorm:"column:lecture_sessions_asset_title;type:varchar(255);not null"`
	LectureSessionsAssetFileURL          string    `gorm:"column:lecture_sessions_asset_file_url;type:text;not null"`
	LectureSessionsAssetFileType         int       `gorm:"column:lecture_sessions_asset_file_type;not null"` // 1=YouTube, 2=PDF, 3=DOCX, etc
	LectureSessionsAssetLectureSessionID string    `gorm:"column:lecture_sessions_asset_lecture_session_id;type:uuid;not null"`
	LectureSessionsAssetCreatedAt        time.Time `gorm:"column:lecture_sessions_asset_created_at;autoCreateTime"`

	// Optional: add relation to LectureSessionModel if needed
	// LectureSession *LectureSessionModel `gorm:"foreignKey:LectureSessionsAssetLectureSessionID"`
}

func (LectureSessionsAssetModel) TableName() string {
	return "lecture_sessions_assets"
}
