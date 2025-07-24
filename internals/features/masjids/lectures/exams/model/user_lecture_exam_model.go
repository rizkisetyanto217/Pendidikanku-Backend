package model

import (
	UserModel "masjidku_backend/internals/features/users/user/model"
	"time"
)

type UserLectureExamModel struct {
	UserLectureExamID        string     `gorm:"column:user_lecture_exam_id;primaryKey;type:uuid;default:gen_random_uuid()" json:"user_lecture_exam_id"`
	UserLectureExamGrade     *float64   `gorm:"column:user_lecture_exam_grade_result" json:"user_lecture_exam_grade_result"` // nullable
	UserLectureExamExamID    string     `gorm:"column:user_lecture_exam_exam_id;type:uuid;not null" json:"user_lecture_exam_exam_id"`
	UserLectureExamUserID    string     `gorm:"column:user_lecture_exam_user_id;type:uuid;not null" json:"user_lecture_exam_user_id"`
	UserLectureExamMasjidID  string     `gorm:"column:user_lecture_exam_masjid_id;type:uuid;not null" json:"user_lecture_exam_masjid_id"` // ✅ baru
	UserLectureExamCreatedAt time.Time  `gorm:"column:user_lecture_exam_created_at;autoCreateTime" json:"user_lecture_exam_created_at"`

	// Optional relations
	User *UserModel.UserModel `gorm:"foreignKey:UserLectureExamUserID"`
	// Exam *LectureExamModel `gorm:"foreignKey:UserLectureExamExamID"` // bisa diaktifkan jika perlu
}

func (UserLectureExamModel) TableName() string {
	return "user_lecture_exams"
}
