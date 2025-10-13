package model

import (
	"time"

	LectureSessionModel "masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	LectureModel "masjidku_backend/internals/features/masjids/lectures/main/model"
	UserModel "masjidku_backend/internals/features/users/users/model"

	"gorm.io/gorm"
)

/* ===========================
   FAQ QUESTIONS (parent)
   =========================== */
type FaqQuestionModel struct {
	FaqQuestionID               string     `gorm:"column:faq_question_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	FaqQuestionUserID           string     `gorm:"column:faq_question_user_id;type:uuid;not null"`
	FaqQuestionText             string     `gorm:"column:faq_question_text;type:text;not null"`
	FaqQuestionLectureID        *string    `gorm:"column:faq_question_lecture_id;type:uuid"`
	FaqQuestionLectureSessionID *string    `gorm:"column:faq_question_lecture_session_id;type:uuid"`
	FaqQuestionIsAnswered       bool       `gorm:"column:faq_question_is_answered;not null;default:false"`

	FaqQuestionCreatedAt time.Time      `gorm:"column:faq_question_created_at;autoCreateTime"`
	FaqQuestionUpdatedAt time.Time      `gorm:"column:faq_question_updated_at;autoUpdateTime"`
	FaqQuestionDeletedAt gorm.DeletedAt `gorm:"column:faq_question_deleted_at;index"`

	// Relations
	User           *UserModel.UserModel                     `gorm:"foreignKey:FaqQuestionUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Lecture        *LectureModel.LectureModel               `gorm:"foreignKey:FaqQuestionLectureID;references:LectureID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	LectureSession *LectureSessionModel.LectureSessionModel `gorm:"foreignKey:FaqQuestionLectureSessionID;references:LectureSessionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// Child answers (FK ada di child → FaqAnswerModel.FaqAnswerQuestionID)
	FaqAnswers []FaqAnswerModel `gorm:"foreignKey:FaqAnswerQuestionID;references:FaqQuestionID"`
}

func (FaqQuestionModel) TableName() string { return "faq_questions" }