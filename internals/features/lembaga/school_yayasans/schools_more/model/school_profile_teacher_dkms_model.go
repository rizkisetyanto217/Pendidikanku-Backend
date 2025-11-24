package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	School "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"
	User "madinahsalam_backend/internals/features/users/users/model"
)

type SchoolProfileTeacherDkmModel struct {
	// PK
	SchoolProfileTeacherDkmID uuid.UUID `gorm:"column:school_profile_teacher_dkm_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"school_profile_teacher_dkm_id"`

	// FK ke schools (NOT NULL, ON DELETE CASCADE)
	SchoolProfileTeacherDkmSchoolID uuid.UUID          `gorm:"column:school_profile_teacher_dkm_school_id;type:uuid;not null;index" json:"school_profile_teacher_dkm_school_id"`
	School                          School.SchoolModel `gorm:"foreignKey:SchoolProfileTeacherDkmSchoolID;references:SchoolID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"school,omitempty"`

	// FK ke users (NULLABLE, ON DELETE SET NULL)
	SchoolProfileTeacherDkmUserID *uuid.UUID      `gorm:"column:school_profile_teacher_dkm_user_id;type:uuid;index" json:"school_profile_teacher_dkm_user_id,omitempty"`
	User                          *User.UserModel `gorm:"foreignKey:SchoolProfileTeacherDkmUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"user,omitempty"`

	// Data profil
	SchoolProfileTeacherDkmName        string  `gorm:"column:school_profile_teacher_dkm_name;type:varchar(100);not null" json:"school_profile_teacher_dkm_name"`
	SchoolProfileTeacherDkmRole        string  `gorm:"column:school_profile_teacher_dkm_role;type:varchar(100);not null" json:"school_profile_teacher_dkm_role"`
	SchoolProfileTeacherDkmDescription *string `gorm:"column:school_profile_teacher_dkm_description;type:text" json:"school_profile_teacher_dkm_description,omitempty"`
	SchoolProfileTeacherDkmMessage     *string `gorm:"column:school_profile_teacher_dkm_message;type:text" json:"school_profile_teacher_dkm_message,omitempty"`
	SchoolProfileTeacherDkmImageURL    *string `gorm:"column:school_profile_teacher_dkm_image_url;type:text" json:"school_profile_teacher_dkm_image_url,omitempty"`

	// Timestamps
	SchoolProfileTeacherDkmCreatedAt time.Time      `gorm:"column:school_profile_teacher_dkm_created_at;not null;autoCreateTime" json:"school_profile_teacher_dkm_created_at"`
	SchoolProfileTeacherDkmUpdatedAt time.Time      `gorm:"column:school_profile_teacher_dkm_updated_at;not null;autoUpdateTime" json:"school_profile_teacher_dkm_updated_at"`
	SchoolProfileTeacherDkmDeletedAt gorm.DeletedAt `gorm:"column:school_profile_teacher_dkm_deleted_at;index" json:"school_profile_teacher_dkm_deleted_at,omitempty"`
}

func (SchoolProfileTeacherDkmModel) TableName() string {
	return "school_profile_teacher_dkm"
}
