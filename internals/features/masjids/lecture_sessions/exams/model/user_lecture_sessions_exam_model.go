package model

import (
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type UserLectureSessionsExamModel struct {
	UserLectureSessionsExamID        string    `gorm:"column:user_lecture_sessions_exam_id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserLectureSessionsExamGrade     *float64  `gorm:"column:user_lecture_sessions_exam_grade_result"` // nullable
	UserLectureSessionsExamExamID    string    `gorm:"column:user_lecture_sessions_exam_exam_id;type:uuid;not null"`
	UserLectureSessionsExamUserID    string    `gorm:"column:user_lecture_sessions_exam_user_id;type:uuid;not null"`
	UserLectureSessionsExamCreatedAt time.Time `gorm:"column:user_lecture_sessions_exam_created_at;autoCreateTime"`

	// Relations
	User *UserModel.UserModel `gorm:"foreignKey:UserLectureSessionsExamUserID"`
	// Exam *LectureSessionsExamModel `gorm:"foreignKey:UserLectureSessionsExamExamID"` // bisa aktifkan jika relasi diperlukan
}

func (UserLectureSessionsExamModel) TableName() string {
	return "user_lecture_sessions_exams"
}
