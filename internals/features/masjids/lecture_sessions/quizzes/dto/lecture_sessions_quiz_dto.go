package dto

import (
	"time"

	"masjidku_backend/internals/features/masjids/lecture_sessions/quizzes/model"
)

// ============================
// Response DTO
// ============================
type LectureSessionsQuizDTO struct {
	LectureSessionsQuizID               string     `json:"lecture_sessions_quiz_id"`
	LectureSessionsQuizTitle            string     `json:"lecture_sessions_quiz_title"`
	LectureSessionsQuizDescription      *string    `json:"lecture_sessions_quiz_description,omitempty"`
	LectureSessionsQuizLectureSessionID string     `json:"lecture_sessions_quiz_lecture_session_id"`
	LectureSessionsQuizMasjidID         string     `json:"lecture_sessions_quiz_masjid_id"`
	LectureSessionsQuizCreatedAt        time.Time  `json:"lecture_sessions_quiz_created_at"`
	LectureSessionsQuizUpdatedAt        time.Time  `json:"lecture_sessions_quiz_updated_at"`
}

// ============================
// Create Request DTO
// ============================
// catatan: masjid_id diambil dari token scope di controller, bukan dari request body
type CreateLectureSessionsQuizRequest struct {
	LectureSessionsQuizTitle            string  `json:"lecture_sessions_quiz_title" validate:"required,min=1,max=255"`
	LectureSessionsQuizDescription      *string `json:"lecture_sessions_quiz_description"` // optional/null
	LectureSessionsQuizLectureSessionID string  `json:"lecture_sessions_quiz_lecture_session_id" validate:"required,uuid"`
}

// Converter: Create -> Model
func (r *CreateLectureSessionsQuizRequest) ToModel(masjidID string) *model.LectureSessionsQuizModel {
	return &model.LectureSessionsQuizModel{
		LectureSessionsQuizTitle:            r.LectureSessionsQuizTitle,
		LectureSessionsQuizDescription:      r.LectureSessionsQuizDescription,
		LectureSessionsQuizLectureSessionID: r.LectureSessionsQuizLectureSessionID,
		LectureSessionsQuizMasjidID:         masjidID, // dari token scope
	}
}

// ============================
// Update Request DTO (partial)
// ============================
// gunakan pointer agar bisa bedakan: tidak dikirim vs set kosong
type UpdateLectureSessionsQuizRequest struct {
	LectureSessionsQuizTitle       *string `json:"lecture_sessions_quiz_title" validate:"omitempty,min=1,max=255"`
	LectureSessionsQuizDescription *string `json:"lecture_sessions_quiz_description"` // boleh null utk hapus deskripsi
}

// ============================
// Converters: Model <-> DTO
// ============================
func ToLectureSessionsQuizDTO(m model.LectureSessionsQuizModel) LectureSessionsQuizDTO {
	return LectureSessionsQuizDTO{
		LectureSessionsQuizID:               m.LectureSessionsQuizID,
		LectureSessionsQuizTitle:            m.LectureSessionsQuizTitle,
		LectureSessionsQuizDescription:      m.LectureSessionsQuizDescription,
		LectureSessionsQuizLectureSessionID: m.LectureSessionsQuizLectureSessionID,
		LectureSessionsQuizMasjidID:         m.LectureSessionsQuizMasjidID,
		LectureSessionsQuizCreatedAt:        m.LectureSessionsQuizCreatedAt,
		LectureSessionsQuizUpdatedAt:        m.LectureSessionsQuizUpdatedAt,
	}
}
