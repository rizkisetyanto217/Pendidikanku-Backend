package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserTeacher struct {
	// PK
	UsersTeacherID uuid.UUID `gorm:"column:users_teacher_id;type:uuid;default:gen_random_uuid();primaryKey" json:"users_teacher_id"`

	// FK (unik per user)
	UsersTeacherUserID uuid.UUID `gorm:"column:users_teacher_user_id;type:uuid;not null" json:"users_teacher_user_id"`

	// Profil ringkas (nullable -> pakai pointer biar bedakan NULL vs "")
	UsersTeacherField            *string `gorm:"column:users_teacher_field;type:varchar(80)" json:"users_teacher_field,omitempty"`
	UsersTeacherShortBio         *string `gorm:"column:users_teacher_short_bio;type:varchar(300)" json:"users_teacher_short_bio,omitempty"`
	UsersTeacherGreeting         *string `gorm:"column:users_teacher_greeting;type:text" json:"users_teacher_greeting,omitempty"`
	UsersTeacherEducation        *string `gorm:"column:users_teacher_education;type:text" json:"users_teacher_education,omitempty"`
	UsersTeacherActivity         *string `gorm:"column:users_teacher_activity;type:text" json:"users_teacher_activity,omitempty"`
	UsersTeacherExperienceYears  *int16  `gorm:"column:users_teacher_experience_years;type:smallint" json:"users_teacher_experience_years,omitempty"`

	// Metadata fleksibel (JSONB)
	UsersTeacherSpecialties  datatypes.JSON `gorm:"column:users_teacher_specialties;type:jsonb" json:"users_teacher_specialties,omitempty"`
	UsersTeacherCertificates datatypes.JSON `gorm:"column:users_teacher_certificates;type:jsonb" json:"users_teacher_certificates,omitempty"`
	UsersTeacherLinks        datatypes.JSON `gorm:"column:users_teacher_links;type:jsonb" json:"users_teacher_links,omitempty"`

	// Status
	UsersTeacherIsVerified bool `gorm:"column:users_teacher_is_verified;not null;default:false" json:"users_teacher_is_verified"`
	UsersTeacherIsActive   bool `gorm:"column:users_teacher_is_active;not null;default:true" json:"users_teacher_is_active"`

	// Search (GENERATED ALWAYS) → read-only
	UsersTeacherSearch string `gorm:"column:users_teacher_search;type:tsvector;->" json:"users_teacher_search"`

	// Audit
	UsersTeacherCreatedAt time.Time      `gorm:"column:users_teacher_created_at;not null;default:now();autoCreateTime" json:"users_teacher_created_at"`
	UsersTeacherUpdatedAt time.Time      `gorm:"column:users_teacher_updated_at;not null;default:now();autoUpdateTime" json:"users_teacher_updated_at"`
	UsersTeacherDeletedAt gorm.DeletedAt `gorm:"column:users_teacher_deleted_at;index" json:"users_teacher_deleted_at,omitempty"`
}

// Table name
func (UserTeacher) TableName() string { return "user_teachers" }
