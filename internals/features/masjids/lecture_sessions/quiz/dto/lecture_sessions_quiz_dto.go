package dto

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/quiz/model"
	"time"
)

// ============================
// Response DTO
// ============================
type LectureSessionsQuizDTO struct {
	LectureSessionsQuizID               string    `json:"lecture_sessions_quiz_id"`
	LectureSessionsQuizTitle            string    `json:"lecture_sessions_quiz_title"`
	LectureSessionsQuizDescription      string    `json:"lecture_sessions_quiz_description"`
	LectureSessionsQuizLectureSessionID string    `json:"lecture_sessions_quiz_lecture_session_id"`
	LectureSessionsQuizMasjidID         string    `json:"lecture_sessions_quiz_masjid_id"` // ⬅️ Tambahan
	LectureSessionsQuizCreatedAt        time.Time `json:"lecture_sessions_quiz_created_at"`
}


// ============================
// Create Request DTO
// ============================
type CreateLectureSessionsQuizRequest struct {
	LectureSessionsQuizTitle            string `json:"lecture_sessions_quiz_title" validate:"required"`
	LectureSessionsQuizDescription      string `json:"lecture_sessions_quiz_description" validate:"required"`
	LectureSessionsQuizLectureSessionID string `json:"lecture_sessions_quiz_lecture_session_id" validate:"required,uuid"`
	LectureSessionsQuizMasjidID         string `json:"lecture_sessions_quiz_masjid_id" validate:"required,uuid"` // ⬅️ Tambahan
}


// ============================
// Converter
// ============================
func (r *CreateLectureSessionsQuizRequest) ToModel() *model.LectureSessionsQuizModel {
	return &model.LectureSessionsQuizModel{
		LectureSessionsQuizTitle:            r.LectureSessionsQuizTitle,
		LectureSessionsQuizDescription:      r.LectureSessionsQuizDescription,
		LectureSessionsQuizLectureSessionID: r.LectureSessionsQuizLectureSessionID,
		LectureSessionsQuizMasjidID:         r.LectureSessionsQuizMasjidID,
	}
}


func ToLectureSessionsQuizDTO(m model.LectureSessionsQuizModel) LectureSessionsQuizDTO {
	return LectureSessionsQuizDTO{
		LectureSessionsQuizID:               m.LectureSessionsQuizID,
		LectureSessionsQuizTitle:            m.LectureSessionsQuizTitle,
		LectureSessionsQuizDescription:      m.LectureSessionsQuizDescription,
		LectureSessionsQuizLectureSessionID: m.LectureSessionsQuizLectureSessionID,
		LectureSessionsQuizMasjidID:         m.LectureSessionsQuizMasjidID, // ⬅️ Tambahan
		LectureSessionsQuizCreatedAt:        m.LectureSessionsQuizCreatedAt,
	}
}
