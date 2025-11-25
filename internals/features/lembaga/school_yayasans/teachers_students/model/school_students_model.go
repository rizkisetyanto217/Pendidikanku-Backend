// file: internals/features/school/students/model/school_student_model.go
package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// =======================================
// ENUM & VALIDATOR
// =======================================

type SchoolStudentStatus string

const (
	SchoolStudentActive   SchoolStudentStatus = "active"
	SchoolStudentInactive SchoolStudentStatus = "inactive"
	SchoolStudentAlumni   SchoolStudentStatus = "alumni"
)

var validSchoolStudentStatus = map[SchoolStudentStatus]struct{}{
	SchoolStudentActive:   {},
	SchoolStudentInactive: {},
	SchoolStudentAlumni:   {},
}

// =======================================
// Helper: item JSONB class sections
// Disimpan di kolom: school_student_class_sections (jsonb, NOT NULL, default '[]')
// =======================================

type SchoolStudentSectionItem struct {
	ClassSectionID             uuid.UUID `json:"class_section_id"`
	IsActive                   bool      `json:"is_active"`
	From                       *string   `json:"from,omitempty"` // YYYY-MM-DD
	To                         *string   `json:"to,omitempty"`   // YYYY-MM-DD | null
	ClassSectionName           *string   `json:"class_section_name,omitempty"`
	ClassSectionSlug           *string   `json:"class_section_slug,omitempty"`
	ClassSectionImageURL       *string   `json:"class_section_image_url,omitempty"`
	ClassSectionImageObjectKey *string   `json:"class_section_image_object_key,omitempty"`
}

// =======================================
// Model: school_students
// =======================================

type SchoolStudentModel struct {
	// PK & Tenant
	SchoolStudentID       uuid.UUID `gorm:"column:school_student_id;type:uuid;default:gen_random_uuid();primaryKey" json:"school_student_id"`
	SchoolStudentSchoolID uuid.UUID `gorm:"column:school_student_school_id;type:uuid;not null;index" json:"school_student_school_id"`

	// Relasi ke users_profile
	SchoolStudentUserProfileID uuid.UUID `gorm:"column:school_student_user_profile_id;type:uuid;not null;index" json:"school_student_user_profile_id"`

	// Identitas internal (unik per school via migration)
	SchoolStudentSlug string  `gorm:"column:school_student_slug;type:varchar(50);not null" json:"school_student_slug"`
	SchoolStudentCode *string `gorm:"column:school_student_code;type:varchar(50)" json:"school_student_code,omitempty"`

	// Status (CHECK: active|inactive|alumni)
	SchoolStudentStatus SchoolStudentStatus `gorm:"column:school_student_status;type:text;not null;default:'active'" json:"school_student_status"`

	// Operasional
	SchoolStudentJoinedAt *time.Time `gorm:"column:school_student_joined_at;type:timestamptz" json:"school_student_joined_at,omitempty"`
	SchoolStudentLeftAt   *time.Time `gorm:"column:school_student_left_at;type:timestamptz" json:"school_student_left_at,omitempty"`

	// Flag: butuh penempatan ke class_sections?
	SchoolStudentNeedsClassSections bool `gorm:"column:school_student_needs_class_sections;type:boolean;not null;default:false" json:"school_student_needs_class_sections"`

	// Catatan
	SchoolStudentNote *string `gorm:"column:school_student_note;type:text" json:"school_student_note,omitempty"`

	// ===== SNAPSHOTS dari users_profile =====
	SchoolStudentUserProfileNameSnapshot              *string `gorm:"column:school_student_user_profile_name_snapshot;type:varchar(80)" json:"school_student_user_profile_name_snapshot,omitempty"`
	SchoolStudentUserProfileAvatarURLSnapshot         *string `gorm:"column:school_student_user_profile_avatar_url_snapshot;type:varchar(255)" json:"school_student_user_profile_avatar_url_snapshot,omitempty"`
	SchoolStudentUserProfileWhatsappURLSnapshot       *string `gorm:"column:school_student_user_profile_whatsapp_url_snapshot;type:varchar(50)" json:"school_student_user_profile_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileParentNameSnapshot        *string `gorm:"column:school_student_user_profile_parent_name_snapshot;type:varchar(80)" json:"school_student_user_profile_parent_name_snapshot,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLSnapshot *string `gorm:"column:school_student_user_profile_parent_whatsapp_url_snapshot;type:varchar(50)" json:"school_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	SchoolStudentUserProfileGenderSnapshot            *string `gorm:"column:school_student_user_profile_gender_snapshot;type:varchar(20)" json:"school_student_user_profile_gender_snapshot,omitempty"` // NEW

	// ===== SCHOOL SNAPSHOT (/me render cepat) =====
	SchoolStudentSchoolNameSnapshot          *string `gorm:"column:school_student_school_name_snapshot;type:varchar(100)" json:"school_student_school_name_snapshot,omitempty"`
	SchoolStudentSchoolSlugSnapshot          *string `gorm:"column:school_student_school_slug_snapshot;type:varchar(100)" json:"school_student_school_slug_snapshot,omitempty"`
	SchoolStudentSchoolLogoURLSnapshot       *string `gorm:"column:school_student_school_logo_url_snapshot;type:text" json:"school_student_school_logo_url_snapshot,omitempty"`
	SchoolStudentSchoolIconURLSnapshot       *string `gorm:"column:school_student_school_icon_url_snapshot;type:text" json:"school_student_school_icon_url_snapshot,omitempty"`
	SchoolStudentSchoolBackgroundURLSnapshot *string `gorm:"column:school_student_school_background_url_snapshot;type:text" json:"school_student_school_background_url_snapshot,omitempty"`

	// ===== JSONB CLASS SECTIONS (NOT NULL DEFAULT '[]') =====
	// Berisi array SchoolStudentSectionItem
	SchoolStudentClassSections datatypes.JSON `gorm:"column:school_student_class_sections;type:jsonb;not null;default:'[]'" json:"school_student_class_sections"`

	// Audit & Soft delete
	SchoolStudentCreatedAt time.Time      `gorm:"column:school_student_created_at;type:timestamptz;not null;default:now();autoCreateTime" json:"school_student_created_at"`
	SchoolStudentUpdatedAt time.Time      `gorm:"column:school_student_updated_at;type:timestamptz;not null;default:now();autoUpdateTime" json:"school_student_updated_at"`
	SchoolStudentDeletedAt gorm.DeletedAt `gorm:"column:school_student_deleted_at;index" json:"school_student_deleted_at,omitempty"`
}

func (SchoolStudentModel) TableName() string { return "school_students" }

// =======================================
// Hooks ringan (mirror aturan SQL)
// =======================================

func (m *SchoolStudentModel) BeforeCreate(tx *gorm.DB) error {
	// JSONB guard
	if len(m.SchoolStudentClassSections) == 0 {
		m.SchoolStudentClassSections = datatypes.JSON([]byte("[]"))
	}
	// Validasi status
	if _, ok := validSchoolStudentStatus[m.SchoolStudentStatus]; !ok {
		return errors.New("invalid school_student_status")
	}
	return nil
}

func (m *SchoolStudentModel) BeforeSave(tx *gorm.DB) error {
	// Validasi status
	if _, ok := validSchoolStudentStatus[m.SchoolStudentStatus]; !ok {
		return errors.New("invalid school_student_status")
	}
	// JSONB guard
	if len(m.SchoolStudentClassSections) == 0 {
		m.SchoolStudentClassSections = datatypes.JSON([]byte("[]"))
	}
	return nil
}
