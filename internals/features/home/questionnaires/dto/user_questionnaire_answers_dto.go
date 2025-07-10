package dto

import (
	"masjidku_backend/internals/features/home/questionnaires/model"
	"time"
)

// =============================
// 📤 Response DTO
// =============================
type UserQuestionnaireAnswerDTO struct {
	UserQuestionnaireID     string    `json:"user_questionnaire_id"`
	UserQuestionnaireUserID string    `json:"user_questionnaire_user_id"`
	UserQuestionnaireType   int       `json:"user_questionnaire_type"`             // 1 = lecture, 2 = event
	UserQuestionnaireRefID  *string   `json:"user_questionnaire_ref_id,omitempty"` // Nullable UUID
	QuestionID              *string   `json:"question_id,omitempty"`               // Nullable UUID
	Answer                  string    `json:"answer"`
	CreatedAt               time.Time `json:"created_at"`
}

// =============================
// 📥 Request DTO (Create)
// =============================
type CreateUserQuestionnaireAnswerRequest struct {
	UserQuestionnaireType  int     `json:"user_questionnaire_type" validate:"required,oneof=1 2"`
	UserQuestionnaireRefID *string `json:"user_questionnaire_ref_id" validate:"omitempty,uuid"` // Nullable UUID
	QuestionID             *string `json:"question_id" validate:"omitempty,uuid"`               // Nullable UUID
	Answer                 string  `json:"answer" validate:"required"`
}

// =============================
// 🔁 Converters
// =============================
func ToUserQuestionnaireAnswerDTO(m model.UserQuestionnaireAnswerModel) UserQuestionnaireAnswerDTO {
	return UserQuestionnaireAnswerDTO{
		UserQuestionnaireID:     m.UserQuestionnaireID,
		UserQuestionnaireUserID: m.UserQuestionnaireUserID,
		UserQuestionnaireType:   m.UserQuestionnaireType,
		UserQuestionnaireRefID:  m.UserQuestionnaireRefID,
		QuestionID:              m.UserQuestionnaireQuestionID,
		Answer:                  m.UserQuestionnaireAnswer,
		CreatedAt:               m.UserQuestionnaireCreatedAt,
	}
}

func ToUserQuestionnaireAnswerModel(req CreateUserQuestionnaireAnswerRequest, userID string) model.UserQuestionnaireAnswerModel {
	return model.UserQuestionnaireAnswerModel{
		UserQuestionnaireUserID:     userID,
		UserQuestionnaireType:       req.UserQuestionnaireType,
		UserQuestionnaireRefID:      req.UserQuestionnaireRefID,
		UserQuestionnaireQuestionID: req.QuestionID,
		UserQuestionnaireAnswer:     req.Answer,
	}
}
