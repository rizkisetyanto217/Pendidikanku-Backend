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

	// Relasi ke user_profiles
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

	// ===== SNAPSHOTS dari user_profiles =====
	SchoolStudentUserProfileNameCache             *string `gorm:"column:school_student_user_profile_name_cache;type:varchar(80)" json:"school_student_user_profile_name_cache,omitempty"`
	SchoolStudentUserProfileAvatarURLCache        *string `gorm:"column:school_student_user_profile_avatar_url_cache;type:varchar(255)" json:"school_student_user_profile_avatar_url_cache,omitempty"`
	SchoolStudentUserProfileWhatsappURLCache      *string `gorm:"column:school_student_user_profile_whatsapp_url_cache;type:varchar(50)" json:"school_student_user_profile_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileParentNameCache       *string `gorm:"column:school_student_user_profile_parent_name_cache;type:varchar(80)" json:"school_student_user_profile_parent_name_cache,omitempty"`
	SchoolStudentUserProfileParentWhatsappURLCache*string `gorm:"column:school_student_user_profile_parent_whatsapp_url_cache;type:varchar(50)" json:"school_student_user_profile_parent_whatsapp_url_cache,omitempty"`
	SchoolStudentUserProfileGenderCache           *string `gorm:"column:school_student_user_profile_gender_cache;type:varchar(20)" json:"school_student_user_profile_gender_cache,omitempty"`

	// ===== JSONB CLASS SECTIONS (NOT NULL DEFAULT '[]') =====
	// Berisi array SchoolStudentSectionItem
	SchoolStudentClassSections datatypes.JSON `gorm:"column:school_student_class_sections;type:jsonb;not null;default:'[]'" json:"school_student_class_sections"`

	// ===== JSONB CLASS SECTION SUBJECT TEACHERS (CSST) =====
	// Struktur item disamakan dengan kebutuhan FE/BE (didecode manual saat perlu)
	SchoolStudentClassSectionSubjectTeachers datatypes.JSON `gorm:"column:school_student_class_section_subject_teachers;type:jsonb;not null;default:'[]'" json:"school_student_class_section_subject_teachers"`

	// ===== STATS (ALL) =====
	SchoolStudentTotalClassSections               int `gorm:"column:school_student_total_class_sections;type:integer;not null;default:0" json:"school_student_total_class_sections"`
	SchoolStudentTotalClassSectionSubjectTeachers int `gorm:"column:school_student_total_class_section_subject_teachers;type:integer;not null;default:0" json:"school_student_total_class_section_subject_teachers"`

	// ===== STATS (ACTIVE ONLY) =====
	SchoolStudentTotalClassSectionsActive               int `gorm:"column:school_student_total_class_sections_active;type:integer;not null;default:0" json:"school_student_total_class_sections_active"`
	SchoolStudentTotalClassSectionSubjectTeachersActive int `gorm:"column:school_student_total_class_section_subject_teachers_active;type:integer;not null;default:0" json:"school_student_total_class_section_subject_teachers_active"`

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
	if len(m.SchoolStudentClassSectionSubjectTeachers) == 0 {
		m.SchoolStudentClassSectionSubjectTeachers = datatypes.JSON([]byte("[]"))
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
	if len(m.SchoolStudentClassSectionSubjectTeachers) == 0 {
		m.SchoolStudentClassSectionSubjectTeachers = datatypes.JSON([]byte("[]"))
	}

	return nil
}
