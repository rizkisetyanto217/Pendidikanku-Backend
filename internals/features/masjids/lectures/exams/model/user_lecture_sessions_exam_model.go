package model

import (
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type UserLectureSessionsExamModel struct {
	UserLectureSessionsExamID        string     `gorm:"column:user_lecture_sessions_exam_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"user_lecture_sessions_exam_id"`
	UserLectureSessionsExamGrade     *float64   `gorm:"column:user_lecture_sessions_exam_grade_result" json:"user_lecture_sessions_exam_grade_result"` // nullable
	UserLectureSessionsExamExamID    string     `gorm:"column:user_lecture_sessions_exam_exam_id;type:uuid;not null" json:"user_lecture_sessions_exam_exam_id"`
	UserLectureSessionsExamUserID    string     `gorm:"column:user_lecture_sessions_exam_user_id;type:uuid;not null" json:"user_lecture_sessions_exam_user_id"`
	UserLectureSessionsExamMasjidID  string     `gorm:"column:user_lecture_sessions_exam_masjid_id;type:uuid;not null" json:"user_lecture_sessions_exam_masjid_id"` // ✅ baru
	UserLectureSessionsExamCreatedAt time.Time  `gorm:"column:user_lecture_sessions_exam_created_at;autoCreateTime" json:"user_lecture_sessions_exam_created_at"`

	// Optional relations
	User *UserModel.UserModel `gorm:"foreignKey:UserLectureSessionsExamUserID"`
	// Exam *LectureSessionsExamModel `gorm:"foreignKey:UserLectureSessionsExamExamID"` // bisa diaktifkan jika perlu
}

func (UserLectureSessionsExamModel) TableName() string {
	return "user_lecture_sessions_exams"
}
