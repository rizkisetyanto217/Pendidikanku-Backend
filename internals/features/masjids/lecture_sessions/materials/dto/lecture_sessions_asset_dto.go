package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"
	"time"
)
type LectureSessionsAssetDTO struct {
	LectureSessionsAssetID               string    `json:"lecture_sessions_asset_id"`
	LectureSessionsAssetTitle            string    `json:"lecture_sessions_asset_title"`
	LectureSessionsAssetFileURL          string    `json:"lecture_sessions_asset_file_url"`
	LectureSessionsAssetFileType         int       `json:"lecture_sessions_asset_file_type"` // 1=YouTube, 2=PDF, dst.
	LectureSessionsAssetLectureSessionID string    `json:"lecture_sessions_asset_lecture_session_id"`
	LectureSessionsAssetMasjidID         string    `json:"lecture_sessions_asset_masjid_id"`
	LectureSessionsAssetCreatedAt        time.Time `json:"lecture_sessions_asset_created_at"`
}

// dto/lecture_sessions_asset.go
type CreateLectureSessionsAssetRequest struct {
	LectureSessionsAssetTitle            string `json:"lecture_sessions_asset_title" validate:"required,min=3"`
	LectureSessionsAssetFileURL          string `json:"lecture_sessions_asset_file_url" validate:"required,url"`
	LectureSessionsAssetFileType         int    `json:"lecture_sessions_asset_file_type" validate:"required"`
	LectureSessionsAssetLectureSessionID string `json:"lecture_sessions_asset_lecture_session_id" validate:"required,uuid"`
	// LectureSessionsAssetMasjidID DIHAPUS — ambil dari token
}


func ToLectureSessionsAssetDTO(m model.LectureSessionsAssetModel) LectureSessionsAssetDTO {
	return LectureSessionsAssetDTO{
		LectureSessionsAssetID:               m.LectureSessionsAssetID,
		LectureSessionsAssetTitle:            m.LectureSessionsAssetTitle,
		LectureSessionsAssetFileURL:          m.LectureSessionsAssetFileURL,
		LectureSessionsAssetFileType:         m.LectureSessionsAssetFileType,
		LectureSessionsAssetLectureSessionID: m.LectureSessionsAssetLectureSessionID,
		LectureSessionsAssetMasjidID:         m.LectureSessionsAssetMasjidID,
		LectureSessionsAssetCreatedAt:        m.LectureSessionsAssetCreatedAt,
	}
}
