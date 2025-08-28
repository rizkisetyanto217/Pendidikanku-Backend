// ================================
// model/lecture_sessions_asset.go
// ================================

package model

import (
	"time"

	lsmodel "masjidku_backend/internals/features/masjids/lecture_sessions/main/model"

	"gorm.io/gorm"
)

type LectureSessionsAssetModel struct {
	LectureSessionsAssetID               string         `gorm:"column:lecture_sessions_asset_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"lecture_sessions_asset_id"`
	LectureSessionsAssetTitle            string         `gorm:"column:lecture_sessions_asset_title;type:varchar(255);not null" json:"lecture_sessions_asset_title"`
	LectureSessionsAssetFileURL          string         `gorm:"column:lecture_sessions_asset_file_url;type:text;not null" json:"lecture_sessions_asset_file_url"`
	LectureSessionsAssetFileType         int            `gorm:"column:lecture_sessions_asset_file_type;not null" json:"lecture_sessions_asset_file_type"` // 1=YouTube, 2=PDF, 3=DOCX, ...
	LectureSessionsAssetLectureSessionID string         `gorm:"column:lecture_sessions_asset_lecture_session_id;type:uuid;not null" json:"lecture_sessions_asset_lecture_session_id"`
	LectureSessionsAssetMasjidID         string         `gorm:"column:lecture_sessions_asset_masjid_id;type:uuid;not null" json:"lecture_sessions_asset_masjid_id"`

	LectureSessionsAssetCreatedAt time.Time      `gorm:"column:lecture_sessions_asset_created_at;autoCreateTime" json:"lecture_sessions_asset_created_at"`
	LectureSessionsAssetUpdatedAt time.Time      `gorm:"column:lecture_sessions_asset_updated_at;autoUpdateTime" json:"lecture_sessions_asset_updated_at"`
	LectureSessionsAssetDeletedAt gorm.DeletedAt `gorm:"column:lecture_sessions_asset_deleted_at;index" json:"-"`

	// Relasi (opsional; tidak ikut di-serialize)
	LectureSession lsmodel.LectureSessionModel `gorm:"foreignKey:LectureSessionsAssetLectureSessionID;references:LectureSessionID" json:"-"`
}

func (LectureSessionsAssetModel) TableName() string { return "lecture_sessions_assets" }
