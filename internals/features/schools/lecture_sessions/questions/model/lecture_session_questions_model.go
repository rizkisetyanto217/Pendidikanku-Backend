package model

import (
	"time"

	"gorm.io/gorm"
)

type LectureSessionsQuestionModel struct {
	LectureSessionsQuestionID string `gorm:"column:lecture_sessions_question_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_sessions_question_id"`
	LectureSessionsQuestion   string `gorm:"column:lecture_sessions_question;type:text;not null" json:"lecture_sessions_question"`

	// JSONB: array/object diperbolehkan di DB; di app kita gunakan array string (A/B/C/D).
	// Jika nanti butuh bentuk object (map[string]string), tinggal ganti tipenya ke map[string]string.
	LectureSessionsQuestionAnswers []string `gorm:"column:lecture_sessions_question_answers;type:jsonb;serializer:json" json:"lecture_sessions_question_answers"`

	LectureSessionsQuestionCorrect     string `gorm:"column:lecture_sessions_question_correct;type:char(1);not null" json:"lecture_sessions_question_correct"` // A/B/C/D
	LectureSessionsQuestionExplanation string `gorm:"column:lecture_sessions_question_explanation;type:text" json:"lecture_sessions_question_explanation"`

	LectureSessionsQuestionQuizID *string `gorm:"column:lecture_sessions_question_quiz_id;type:uuid" json:"lecture_sessions_question_quiz_id"`
	LectureQuestionExamID         *string `gorm:"column:lecture_question_exam_id;type:uuid" json:"lecture_question_exam_id"`

	LectureSessionsQuestionSchoolID string `gorm:"column:lecture_sessions_question_school_id;type:uuid;not null" json:"lecture_sessions_question_school_id"`

	LectureSessionsQuestionCreatedAt time.Time      `gorm:"column:lecture_sessions_question_created_at;autoCreateTime" json:"lecture_sessions_question_created_at"`
	LectureSessionsQuestionUpdatedAt time.Time      `gorm:"column:lecture_sessions_question_updated_at;autoUpdateTime" json:"lecture_sessions_question_updated_at"`
	LectureSessionsQuestionDeletedAt gorm.DeletedAt `gorm:"column:lecture_sessions_question_deleted_at;index" json:"-"`

	// --- Optional relations (aktifkan bila perlu) ---
	// Quiz   *LectureSessionsQuizModel `gorm:"foreignKey:LectureSessionsQuestionQuizID;references:LectureSessionsQuizID"`
	// Exam   *LectureExamModel         `gorm:"foreignKey:LectureQuestionExamID;references:LectureExamID"`
	// School *SchoolModel              `gorm:"foreignKey:LectureSessionsQuestionSchoolID;references:SchoolID"`
}

func (LectureSessionsQuestionModel) TableName() string { return "lecture_sessions_questions" }
