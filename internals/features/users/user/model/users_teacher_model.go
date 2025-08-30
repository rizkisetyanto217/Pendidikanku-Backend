package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UsersTeacherModel struct {
	ID                 uuid.UUID       `gorm:"column:users_teacher_id;type:uuid;default:gen_random_uuid();primaryKey" json:"users_teacher_id"`
	UserID             uuid.UUID       `gorm:"column:users_teacher_user_id;type:uuid;not null;unique" json:"users_teacher_user_id"`

	// Konten profil
	Field              string          `gorm:"column:users_teacher_field;type:varchar(80)" json:"users_teacher_field"`
	ShortBio           string          `gorm:"column:users_teacher_short_bio;type:varchar(300)" json:"users_teacher_short_bio"`
	Greeting           string          `gorm:"column:users_teacher_greeting;type:text" json:"users_teacher_greeting"`
	Education          string          `gorm:"column:users_teacher_education;type:text" json:"users_teacher_education"`
	Activity           string          `gorm:"column:users_teacher_activity;type:text" json:"users_teacher_activity"`
	ExperienceYears    *int16          `gorm:"column:users_teacher_experience_years;type:smallint" json:"users_teacher_experience_years"`

	// Metadata fleksibel (JSONB)
	Specialties        datatypes.JSON  `gorm:"column:users_teacher_specialties;type:jsonb" json:"users_teacher_specialties"`
	Certificates       datatypes.JSON  `gorm:"column:users_teacher_certificates;type:jsonb" json:"users_teacher_certificates"`
	Links              datatypes.JSON  `gorm:"column:users_teacher_links;type:jsonb" json:"users_teacher_links"`

	// Status
	IsVerified         bool            `gorm:"column:users_teacher_is_verified;not null;default:false" json:"users_teacher_is_verified"`
	IsActive           bool            `gorm:"column:users_teacher_is_active;not null;default:true" json:"users_teacher_is_active"`

	// Audit
	CreatedAt          time.Time       `gorm:"column:users_teacher_created_at;not null;default:now()" json:"users_teacher_created_at"`
	UpdatedAt          time.Time       `gorm:"column:users_teacher_updated_at;not null;default:now()" json:"users_teacher_updated_at"`
	DeletedAt          gorm.DeletedAt  `gorm:"column:users_teacher_deleted_at;index" json:"users_teacher_deleted_at"`
}

// Nama tabel eksplisit
func (UsersTeacherModel) TableName() string {
	return "users_teacher"
}
