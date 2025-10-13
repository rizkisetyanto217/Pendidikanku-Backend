// =================================
// model/user_lecture_exam_model.go
// =================================

package model

import (
	"time"

	UserModel "masjidku_backend/internals/features/users/users/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserLectureExamModel struct {
	UserLectureExamID        uuid.UUID      `gorm:"column:user_lecture_exam_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"user_lecture_exam_id"`
	UserLectureExamGrade     *float64       `gorm:"column:user_lecture_exam_grade_result" json:"user_lecture_exam_grade_result"`

	UserLectureExamExamID    uuid.UUID      `gorm:"column:user_lecture_exam_exam_id;type:uuid;not null" json:"user_lecture_exam_exam_id"`
	UserLectureExamUserID    uuid.UUID      `gorm:"column:user_lecture_exam_user_id;type:uuid;not null" json:"user_lecture_exam_user_id"`
	UserLectureExamUserName  *string        `gorm:"column:user_lecture_exam_user_name;type:varchar(100)" json:"user_lecture_exam_user_name,omitempty"`
	UserLectureExamMasjidID  uuid.UUID      `gorm:"column:user_lecture_exam_masjid_id;type:uuid;not null" json:"user_lecture_exam_masjid_id"`

	UserLectureExamCreatedAt time.Time       `gorm:"column:user_lecture_exam_created_at;autoCreateTime" json:"user_lecture_exam_created_at"`
	UserLectureExamUpdatedAt time.Time       `gorm:"column:user_lecture_exam_updated_at;autoUpdateTime" json:"user_lecture_exam_updated_at"`
	UserLectureExamDeletedAt gorm.DeletedAt  `gorm:"column:user_lecture_exam_deleted_at;index" json:"-"`

	// Relasi (opsional)
	User *UserModel.UserModel `gorm:"foreignKey:UserLectureExamUserID" json:"user,omitempty"`
}

func (UserLectureExamModel) TableName() string { return "user_lecture_exams" }
