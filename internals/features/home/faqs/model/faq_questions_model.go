package model

import (
	LectureModel "masjidku_backend/internals/features/masjids/lectures/model"
	LectureSessionModel "masjidku_backend/internals/features/masjids/lecture_sessions/main/model"
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type FaqQuestionModel struct {
	FaqQuestionID                 string     `gorm:"column:faq_question_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	FaqQuestionUserID             string     `gorm:"column:faq_question_user_id;type:uuid;not null"`
	FaqQuestionText               string     `gorm:"column:faq_question_text;type:text;not null"`
	FaqQuestionLectureID          *string    `gorm:"column:faq_question_lecture_id;type:uuid"`
	FaqQuestionLectureSessionID   *string    `gorm:"column:faq_question_lecture_session_id;type:uuid"`
	FaqQuestionIsAnswered         bool       `gorm:"column:faq_question_is_answered;default:false"`
	FaqQuestionCreatedAt          time.Time  `gorm:"column:faq_question_created_at;autoCreateTime"`

	// Relations
	User           *UserModel.UserModel                     `gorm:"foreignKey:FaqQuestionUserID"`
	Lecture        *LectureModel.LectureModel               `gorm:"foreignKey:FaqQuestionLectureID"`
	LectureSession *LectureSessionModel.LectureSessionModel `gorm:"foreignKey:FaqQuestionLectureSessionID"`
	FaqAnswers     []FaqAnswerModel                         `gorm:"foreignKey:FaqAnswerQuestionID"`
}

func (FaqQuestionModel) TableName() string {
	return "faq_questions"
}
