package model

import (
	"time"

	"gorm.io/gorm"
)

type LectureSessionsUserQuestionModel struct {
	LectureSessionsUserQuestionID        string `gorm:"column:lecture_sessions_user_question_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"lecture_sessions_user_question_id"`
	LectureSessionsUserQuestionAnswer    string `gorm:"column:lecture_sessions_user_question_answer;type:char(1);not null" json:"lecture_sessions_user_question_answer"` // A/B/C/D
	LectureSessionsUserQuestionIsCorrect bool   `gorm:"column:lecture_sessions_user_question_is_correct;not null" json:"lecture_sessions_user_question_is_correct"`

	LectureSessionsUserQuestionQuestionID string `gorm:"column:lecture_sessions_user_question_question_id;type:uuid;not null" json:"lecture_sessions_user_question_question_id"`
	LectureSessionsUserQuestionSchoolID   string `gorm:"column:lecture_sessions_user_question_school_id;type:uuid;not null" json:"lecture_sessions_user_question_school_id"`

	LectureSessionsUserQuestionCreatedAt time.Time      `gorm:"column:lecture_sessions_user_question_created_at;autoCreateTime" json:"lecture_sessions_user_question_created_at"`
	LectureSessionsUserQuestionUpdatedAt time.Time      `gorm:"column:lecture_sessions_user_question_updated_at;autoUpdateTime" json:"lecture_sessions_user_question_updated_at"`
	LectureSessionsUserQuestionDeletedAt gorm.DeletedAt `gorm:"column:lecture_sessions_user_question_deleted_at;index" json:"-"`

	// --- Optional relations (aktifkan bila perlu) ---
	// Question *LectureSessionsQuestionModel `gorm:"foreignKey:LectureSessionsUserQuestionQuestionID;references:LectureSessionsQuestionID"`
	// School   *SchoolModel                  `gorm:"foreignKey:LectureSessionsUserQuestionSchoolID;references:SchoolID"`
}

func (LectureSessionsUserQuestionModel) TableName() string { return "lecture_sessions_user_questions" }
