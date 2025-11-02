package dto

import (
	"schoolku_backend/internals/constants"
	"schoolku_backend/internals/features/schools/lecture_sessions/materials/model"
	"time"
)

// DTO utama untuk logika internal / DB
type LectureSessionsAssetDTO struct {
	LectureSessionsAssetID               string    `json:"lecture_sessions_asset_id"`
	LectureSessionsAssetTitle            string    `json:"lecture_sessions_asset_title"`
	LectureSessionsAssetFileURL          string    `json:"lecture_sessions_asset_file_url"`
	LectureSessionsAssetFileType         int       `json:"lecture_sessions_asset_file_type"` // 1 = YouTube, 2 = Audio, dll.
	LectureSessionsAssetLectureSessionID string    `json:"lecture_sessions_asset_lecture_session_id"`
	LectureSessionsAssetSchoolID         string    `json:"lecture_sessions_asset_school_id"`
	LectureSessionsAssetCreatedAt        time.Time `json:"lecture_sessions_asset_created_at"`
	LectureSessionsAssetFileTypeLabel    string    `json:"lecture_sessions_asset_file_type_label"`
}

// DTO untuk request create
type CreateLectureSessionsAssetRequest struct {
	LectureSessionsAssetTitle            string `json:"lecture_sessions_asset_title" validate:"required,min=3"`
	LectureSessionsAssetFileURL          string `json:"lecture_sessions_asset_file_url" validate:"required,url"`
	LectureSessionsAssetFileType         int    `json:"lecture_sessions_asset_file_type" validate:"required"`
	LectureSessionsAssetLectureSessionID string `json:"lecture_sessions_asset_lecture_session_id" validate:"required,uuid"`
}

// DTO untuk response (created_at dalam string agar aman di frontend)
type LectureSessionsAssetResponse struct {
	LectureSessionsAssetID               string `json:"lecture_sessions_asset_id"`
	LectureSessionsAssetTitle            string `json:"lecture_sessions_asset_title"`
	LectureSessionsAssetFileURL          string `json:"lecture_sessions_asset_file_url"`
	LectureSessionsAssetFileType         int    `json:"lecture_sessions_asset_file_type"`
	LectureSessionsAssetLectureSessionID string `json:"lecture_sessions_asset_lecture_session_id"`
	LectureSessionsAssetSchoolID         string `json:"lecture_sessions_asset_school_id"`
	LectureSessionsAssetCreatedAt        string `json:"lecture_sessions_asset_created_at"`
	LectureSessionsAssetFileTypeLabel    string `json:"lecture_sessions_asset_file_type_label"`
}

// Fungsi konversi untuk kebutuhan internal
func ToLectureSessionsAssetDTO(m model.LectureSessionsAssetModel) LectureSessionsAssetDTO {
	return LectureSessionsAssetDTO{
		LectureSessionsAssetID:               m.LectureSessionsAssetID,
		LectureSessionsAssetTitle:            m.LectureSessionsAssetTitle,
		LectureSessionsAssetFileURL:          m.LectureSessionsAssetFileURL,
		LectureSessionsAssetFileType:         m.LectureSessionsAssetFileType,
		LectureSessionsAssetLectureSessionID: m.LectureSessionsAssetLectureSessionID,
		LectureSessionsAssetSchoolID:         m.LectureSessionsAssetSchoolID,
		LectureSessionsAssetCreatedAt:        m.LectureSessionsAssetCreatedAt,
		LectureSessionsAssetFileTypeLabel:    getFileTypeLabel(m.LectureSessionsAssetFileType),
	}
}

// Fungsi konversi ke response untuk frontend
func ToLectureSessionsAssetResponse(m model.LectureSessionsAssetModel) LectureSessionsAssetResponse {
	return LectureSessionsAssetResponse{
		LectureSessionsAssetID:               m.LectureSessionsAssetID,
		LectureSessionsAssetTitle:            m.LectureSessionsAssetTitle,
		LectureSessionsAssetFileURL:          m.LectureSessionsAssetFileURL,
		LectureSessionsAssetFileType:         m.LectureSessionsAssetFileType,
		LectureSessionsAssetLectureSessionID: m.LectureSessionsAssetLectureSessionID,
		LectureSessionsAssetSchoolID:         m.LectureSessionsAssetSchoolID,
		LectureSessionsAssetCreatedAt:        m.LectureSessionsAssetCreatedAt.Format(time.RFC3339),
		LectureSessionsAssetFileTypeLabel:    getFileTypeLabel(m.LectureSessionsAssetFileType),
	}
}

// Ambil label dari constant map
func getFileTypeLabel(fileType int) string {
	if label, ok := constants.FileTypeLabels[fileType]; ok {
		return label
	}
	return "Tidak diketahui"
}
