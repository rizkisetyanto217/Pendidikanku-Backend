package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserTeacher struct {
	// PK
	UserTeacherID uuid.UUID `gorm:"column:user_teacher_id;type:uuid;default:gen_random_uuid();primaryKey" json:"user_teacher_id"`

	// FK (unik per user)
	UserTeacherUserID uuid.UUID `gorm:"column:user_teacher_user_id;type:uuid;not null;uniqueIndex:uq_user_teachers_user" json:"user_teacher_user_id"`

	// Profil ringkas
	UserTeacherField              *string `gorm:"column:user_teacher_field;type:varchar(80)" json:"user_teacher_field,omitempty"`
	UserTeacherShortBio           *string `gorm:"column:user_teacher_short_bio;type:varchar(300)" json:"user_teacher_short_bio,omitempty"`
	UserTeacherLongBio            *string `gorm:"column:user_teacher_long_bio;type:text" json:"user_teacher_long_bio,omitempty"`
	UserTeacherGreeting           *string `gorm:"column:user_teacher_greeting;type:text" json:"user_teacher_greeting,omitempty"`
	UserTeacherEducation          *string `gorm:"column:user_teacher_education;type:text" json:"user_teacher_education,omitempty"`
	UserTeacherActivity           *string `gorm:"column:user_teacher_activity;type:text" json:"user_teacher_activity,omitempty"`
	UserTeacherExperienceYears    *int16  `gorm:"column:user_teacher_experience_years;type:smallint" json:"user_teacher_experience_years,omitempty"`

	// Metadata fleksibel (JSONB)
	// Catatan: gunakan *datatypes.JSON jika ingin bisa NULL berbeda dari []/{}.
	UserTeacherSpecialties  *datatypes.JSON `gorm:"column:user_teacher_specialties;type:jsonb" json:"user_teacher_specialties,omitempty"`
	UserTeacherCertificates *datatypes.JSON `gorm:"column:user_teacher_certificates;type:jsonb" json:"user_teacher_certificates,omitempty"`

	// Status
	UserTeacherIsVerified bool `gorm:"column:user_teacher_is_verified;not null;default:false" json:"user_teacher_is_verified"`
	UserTeacherIsActive   bool `gorm:"column:user_teacher_is_active;not null;default:true" json:"user_teacher_is_active"`

	// Search (GENERATED ALWAYS) → read-only
	UserTeacherSearch string `gorm:"column:user_teacher_search;type:tsvector;->" json:"user_teacher_search"`

	// Audit
	UserTeacherCreatedAt time.Time      `gorm:"column:user_teacher_created_at;not null;default:now();autoCreateTime" json:"user_teacher_created_at"`
	UserTeacherUpdatedAt time.Time      `gorm:"column:user_teacher_updated_at;not null;default:now();autoUpdateTime" json:"user_teacher_updated_at"`
	UserTeacherDeletedAt gorm.DeletedAt `gorm:"column:user_teacher_deleted_at;index" json:"user_teacher_deleted_at,omitempty"`
}

func (UserTeacher) TableName() string { return "user_teachers" }
