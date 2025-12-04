// file: internals/features/lembaga/teachers_students/model/user_teacher.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type UserTeacherModel struct {
	// PK & FK
	UserTeacherID     uuid.UUID `json:"user_teacher_id" gorm:"type:uuid;primaryKey;column:user_teacher_id"`
	UserTeacherUserID uuid.UUID `json:"user_teacher_user_id" gorm:"type:uuid;not null;column:user_teacher_user_id;uniqueIndex:uq_user_teachers_user"`

	// Profil ringkas
	UserTeacherUserFullNameCache   string  `json:"user_teacheru_user_full_name_cache" gorm:"type:varchar(100);not null;column:user_teacher_user_full_name_cache"`
	UserTeacherField           *string `json:"user_teacher_field,omitempty" gorm:"type:varchar(80);column:user_teacher_field"`
	UserTeacherShortBio        *string `json:"user_teacher_short_bio,omitempty" gorm:"type:varchar(300);column:user_teacher_short_bio"`
	UserTeacherLongBio         *string `json:"user_teacher_long_bio,omitempty" gorm:"type:text;column:user_teacher_long_bio"`
	UserTeacherGreeting        *string `json:"user_teacher_greeting,omitempty" gorm:"type:text;column:user_teacher_greeting"`
	UserTeacherEducation       *string `json:"user_teacher_education,omitempty" gorm:"type:text;column:user_teacher_education"`
	UserTeacherActivity        *string `json:"user_teacher_activity,omitempty" gorm:"type:text;column:user_teacher_activity"`
	UserTeacherExperienceYears *int16  `json:"user_teacher_experience_years,omitempty" gorm:"type:smallint;column:user_teacher_experience_years"`

	// Demografis (opsional)
	UserTeacherGender   *string `json:"user_teacher_gender,omitempty" gorm:"type:varchar(10);column:user_teacher_gender"`
	UserTeacherLocation *string `json:"user_teacher_location,omitempty" gorm:"type:varchar(100);column:user_teacher_location"`
	UserTeacherCity     *string `json:"user_teacher_city,omitempty" gorm:"type:varchar(100);column:user_teacher_city"`

	// Metadata fleksibel
	UserTeacherSpecialties  datatypes.JSON `json:"user_teacher_specialties,omitempty" gorm:"type:jsonb;column:user_teacher_specialties"`
	UserTeacherCertificates datatypes.JSON `json:"user_teacher_certificates,omitempty" gorm:"type:jsonb;column:user_teacher_certificates"`

	// Sosial media (opsional)
	UserTeacherInstagramURL     *string `json:"user_teacher_instagram_url,omitempty" gorm:"type:text;column:user_teacher_instagram_url"`
	UserTeacherWhatsappURL      *string `json:"user_teacher_whatsapp_url,omitempty" gorm:"type:text;column:user_teacher_whatsapp_url"`
	UserTeacherYoutubeURL       *string `json:"user_teacher_youtube_url,omitempty" gorm:"type:text;column:user_teacher_youtube_url"`
	UserTeacherLinkedinURL      *string `json:"user_teacher_linkedin_url,omitempty" gorm:"type:text;column:user_teacher_linkedin_url"`
	UserTeacherGithubURL        *string `json:"user_teacher_github_url,omitempty" gorm:"type:text;column:user_teacher_github_url"`
	UserTeacherTelegramUsername *string `json:"user_teacher_telegram_username,omitempty" gorm:"type:varchar(50);column:user_teacher_telegram_username"`

	// Avatar (single file, 2-slot + retensi 30 hari)
	UserTeacherAvatarURL                *string    `json:"user_teacher_avatar_url,omitempty" gorm:"type:text;column:user_teacher_avatar_url"`
	UserTeacherAvatarObjectKey          *string    `json:"user_teacher_avatar_object_key,omitempty" gorm:"type:text;column:user_teacher_avatar_object_key"`
	UserTeacherAvatarURLOld             *string    `json:"user_teacher_avatar_url_old,omitempty" gorm:"type:text;column:user_teacher_avatar_url_old"`
	UserTeacherAvatarObjectKeyOld       *string    `json:"user_teacher_avatar_object_key_old,omitempty" gorm:"type:text;column:user_teacher_avatar_object_key_old"`
	UserTeacherAvatarDeletePendingUntil *time.Time `json:"user_teacher_avatar_delete_pending_until,omitempty" gorm:"column:user_teacher_avatar_delete_pending_until"`

	// Title
	UserTeacherTitlePrefix *string `json:"user_teacher_title_prefix,omitempty" gorm:"type:varchar(60);column:user_teacher_title_prefix"`
	UserTeacherTitleSuffix *string `json:"user_teacher_title_suffix,omitempty" gorm:"type:varchar(60);column:user_teacher_title_suffix"`

	// Status
	UserTeacherIsVerified bool `json:"user_teacher_is_verified" gorm:"not null;default:false;column:user_teacher_is_verified"`
	UserTeacherIsActive   bool `json:"user_teacher_is_active" gorm:"not null;default:true;column:user_teacher_is_active"`

	// Completion status (onboarding/profile)
	UserTeacherIsCompleted bool       `json:"user_teacher_is_completed" gorm:"not null;default:false;column:user_teacher_is_completed"`
	UserTeacherCompletedAt *time.Time `json:"user_teacher_completed_at,omitempty" gorm:"column:user_teacher_completed_at"`

	// Audit
	UserTeacherCreatedAt time.Time      `json:"user_teacher_created_at" gorm:"column:user_teacher_created_at;autoCreateTime"`
	UserTeacherUpdatedAt time.Time      `json:"user_teacher_updated_at" gorm:"column:user_teacher_updated_at;autoUpdateTime"`
	UserTeacherDeletedAt gorm.DeletedAt `json:"user_teacher_deleted_at,omitempty" gorm:"column:user_teacher_deleted_at;index"`
}

// TableName overrides the default pluralization.
func (UserTeacherModel) TableName() string { return "user_teachers" }
