// file: internals/features/school/teachers/model/school_teacher_model.go
package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================
   Enum: teacher_employment_enum
========================= */

type TeacherEmployment string

const (
	TeacherEmploymentTetap      TeacherEmployment = "tetap"
	TeacherEmploymentKontrak    TeacherEmployment = "kontrak"
	TeacherEmploymentParuhWaktu TeacherEmployment = "paruh_waktu"
	TeacherEmploymentMagang     TeacherEmployment = "magang"
	TeacherEmploymentHonorer    TeacherEmployment = "honorer"
	TeacherEmploymentRelawan    TeacherEmployment = "relawan"
	TeacherEmploymentTamu       TeacherEmployment = "tamu"
)

var validTeacherEmployment = map[TeacherEmployment]struct{}{
	TeacherEmploymentTetap:      {},
	TeacherEmploymentKontrak:    {},
	TeacherEmploymentParuhWaktu: {},
	TeacherEmploymentMagang:     {},
	TeacherEmploymentHonorer:    {},
	TeacherEmploymentRelawan:    {},
	TeacherEmploymentTamu:       {},
}

func (e TeacherEmployment) String() string { return string(e) }
func (e TeacherEmployment) Valid() bool {
	_, ok := validTeacherEmployment[e]
	return ok
}

// DB round-trip (enum)
func (e TeacherEmployment) Value() (driver.Value, error) {
	if e == "" {
		return nil, nil
	}
	if !e.Valid() {
		return nil, fmt.Errorf("invalid teacher_employment value: %q", e)
	}
	return string(e), nil
}

func (e *TeacherEmployment) Scan(value any) error {
	if value == nil {
		*e = ""
		return nil
	}
	var s string
	switch v := value.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("unsupported Scan type for TeacherEmployment: %T", value)
	}
	s = strings.ToLower(strings.TrimSpace(s))
	ev := TeacherEmployment(s)
	if ev != "" && !ev.Valid() {
		return fmt.Errorf("invalid teacher_employment value from DB: %q", s)
	}
	*e = ev
	return nil
}

/* =========================
   Model: school_teachers
========================= */

type SchoolTeacherModel struct {
	// PK
	SchoolTeacherID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:school_teacher_id" json:"school_teacher_id"`

	// Scope/Relasi
	SchoolTeacherSchoolID      uuid.UUID `gorm:"type:uuid;not null;column:school_teacher_school_id" json:"school_teacher_school_id"`
	SchoolTeacherUserTeacherID uuid.UUID `gorm:"type:uuid;not null;column:school_teacher_user_teacher_id" json:"school_teacher_user_teacher_id"`

	// Identitas/Kepegawaian
	SchoolTeacherCode       *string            `gorm:"type:varchar(50);column:school_teacher_code" json:"school_teacher_code,omitempty"`
	SchoolTeacherSlug       *string            `gorm:"type:varchar(50);column:school_teacher_slug" json:"school_teacher_slug,omitempty"`
	SchoolTeacherEmployment *TeacherEmployment `gorm:"type:teacher_employment_enum;column:school_teacher_employment" json:"school_teacher_employment,omitempty"`
	SchoolTeacherIsActive   bool               `gorm:"type:boolean;not null;default:true;column:school_teacher_is_active" json:"school_teacher_is_active"`

	// Periode kerja (DATE)
	SchoolTeacherJoinedAt *time.Time `gorm:"type:date;column:school_teacher_joined_at" json:"school_teacher_joined_at,omitempty"`
	SchoolTeacherLeftAt   *time.Time `gorm:"type:date;column:school_teacher_left_at" json:"school_teacher_left_at,omitempty"`

	// Verifikasi
	SchoolTeacherIsVerified bool       `gorm:"type:boolean;not null;default:false;column:school_teacher_is_verified" json:"school_teacher_is_verified"`
	SchoolTeacherVerifiedAt *time.Time `gorm:"type:timestamptz;column:school_teacher_verified_at" json:"school_teacher_verified_at,omitempty"`

	// Visibilitas & Catatan
	SchoolTeacherIsPublic bool    `gorm:"type:boolean;not null;default:true;column:school_teacher_is_public" json:"school_teacher_is_public"`
	SchoolTeacherNotes    *string `gorm:"type:text;column:school_teacher_notes" json:"school_teacher_notes,omitempty"`

	// Snapshot dari user_teachers
	SchoolTeacherUserTeacherFullNameCache    *string `gorm:"type:varchar(80);column:school_teacher_user_teacher_full_name_cache" json:"school_teacher_user_teacher_full_name_cache,omitempty"`
	SchoolTeacherUserTeacherAvatarURLCache   *string `gorm:"type:varchar(255);column:school_teacher_user_teacher_avatar_url_cache" json:"school_teacher_user_teacher_avatar_url_cache,omitempty"`
	SchoolTeacherUserTeacherWhatsappURLCache *string `gorm:"type:varchar(50);column:school_teacher_user_teacher_whatsapp_url_cache" json:"school_teacher_user_teacher_whatsapp_url_cache,omitempty"`
	SchoolTeacherUserTeacherTitlePrefixCache *string `gorm:"type:varchar(20);column:school_teacher_user_teacher_title_prefix_cache" json:"school_teacher_user_teacher_title_prefix_cache,omitempty"`
	SchoolTeacherUserTeacherTitleSuffixCache *string `gorm:"type:varchar(30);column:school_teacher_user_teacher_title_suffix_cache" json:"school_teacher_user_teacher_title_suffix_cache,omitempty"`
	SchoolTeacherUserTeacherGenderCache      *string `gorm:"type:varchar(20);column:school_teacher_user_teacher_gender_cache" json:"school_teacher_user_teacher_gender_cache,omitempty"`

	// Audit & Soft Delete
	SchoolTeacherCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:school_teacher_created_at" json:"school_teacher_created_at"`
	SchoolTeacherUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:school_teacher_updated_at" json:"school_teacher_updated_at"`
	SchoolTeacherDeletedAt gorm.DeletedAt `gorm:"column:school_teacher_deleted_at;index" json:"school_teacher_deleted_at,omitempty"`
}

func (SchoolTeacherModel) TableName() string { return "school_teachers" }
