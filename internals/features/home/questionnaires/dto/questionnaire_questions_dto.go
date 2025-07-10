package dto

import (
	"masjidku_backend/internals/features/home/questionnaires/model"
	"time"
)

// =============================
// 📤 Response DTO
// =============================
type QuestionnaireQuestionDTO struct {
	QuestionID       string    `json:"question_id"`
	QuestionText     string    `json:"question_text"`
	QuestionType     int       `json:"question_type"` // 1=rating, 2=text, 3=choice
	QuestionOptions  []string  `json:"question_options,omitempty"`
	EventID          *string   `json:"event_id,omitempty"`
	LectureSessionID *string   `json:"lecture_session_id,omitempty"`
	QuestionScope    int       `json:"question_scope"` // 1=general, 2=event, 3=lecture
	CreatedAt        time.Time `json:"created_at"`
}

// =============================
// 📥 Request DTO (Create / Update)
// =============================
type CreateQuestionnaireQuestionRequest struct {
	QuestionText     string   `json:"question_text" validate:"required"`
	QuestionType     int      `json:"question_type" validate:"required,oneof=1 2 3"` // rating/text/choice
	QuestionOptions  []string `json:"question_options,omitempty"`                    // optional, only if type == 3
	EventID          *string  `json:"event_id,omitempty"`
	LectureSessionID *string  `json:"lecture_session_id,omitempty"`
	QuestionScope    int      `json:"question_scope" validate:"required,oneof=1 2 3"` // general/event/lecture
}

// =============================
// 🔁 Converters
// =============================
func ToQuestionnaireQuestionDTO(m model.QuestionnaireQuestionModel) QuestionnaireQuestionDTO {
	return QuestionnaireQuestionDTO{
		QuestionID:       m.QuestionID,
		QuestionText:     m.QuestionText,
		QuestionType:     m.QuestionType,
		QuestionOptions:  m.QuestionOptions,
		EventID:          m.EventID,
		LectureSessionID: m.LectureSessionID,
		QuestionScope:    m.QuestionScope,
		CreatedAt:        m.CreatedAt,
	}
}

func ToQuestionnaireQuestionModel(req CreateQuestionnaireQuestionRequest) model.QuestionnaireQuestionModel {
	return model.QuestionnaireQuestionModel{
		QuestionText:     req.QuestionText,
		QuestionType:     req.QuestionType,
		QuestionOptions:  req.QuestionOptions,
		EventID:          req.EventID,
		LectureSessionID: req.LectureSessionID,
		QuestionScope:    req.QuestionScope,
	}
}
