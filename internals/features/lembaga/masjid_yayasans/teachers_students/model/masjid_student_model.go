// file: internals/features/school/students/model/masjid_student_model.go
package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MasjidStudentStatus string

const (
	MasjidStudentActive   MasjidStudentStatus = "active"
	MasjidStudentInactive MasjidStudentStatus = "inactive"
	MasjidStudentAlumni   MasjidStudentStatus = "alumni"
)

var validMasjidStudentStatus = map[MasjidStudentStatus]struct{}{
	MasjidStudentActive:   {},
	MasjidStudentInactive: {},
	MasjidStudentAlumni:   {},
}

// Opsional: helper untuk manipulasi JSONB sections di service layer
type MasjidStudentSectionItem struct {
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
// Model: masjid_students
// =======================================
type MasjidStudentModel struct {
	// PK & Tenant
	MasjidStudentID       uuid.UUID `gorm:"column:masjid_student_id;type:uuid;default:gen_random_uuid();primaryKey" json:"masjid_student_id"`
	MasjidStudentMasjidID uuid.UUID `gorm:"column:masjid_student_masjid_id;type:uuid;not null;index" json:"masjid_student_masjid_id"`

	// Relasi ke users_profile
	MasjidStudentUserProfileID uuid.UUID `gorm:"column:masjid_student_user_profile_id;type:uuid;not null;index" json:"masjid_student_user_profile_id"`

	// Identitas internal (unik per masjid via migration)
	MasjidStudentSlug string  `gorm:"column:masjid_student_slug;type:varchar(50);not null" json:"masjid_student_slug"`
	MasjidStudentCode *string `gorm:"column:masjid_student_code;type:varchar(50)" json:"masjid_student_code,omitempty"`

	// Status (CHECK: active|inactive|alumni)
	MasjidStudentStatus MasjidStudentStatus `gorm:"column:masjid_student_status;type:text;not null;default:'active'" json:"masjid_student_status"`

	// Operasional
	MasjidStudentJoinedAt *time.Time `gorm:"column:masjid_student_joined_at;type:timestamptz" json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `gorm:"column:masjid_student_left_at;type:timestamptz" json:"masjid_student_left_at,omitempty"`

	// Catatan
	MasjidStudentNote *string `gorm:"column:masjid_student_note;type:text" json:"masjid_student_note,omitempty"`

	// ===== SNAPSHOTS dari users_profile =====
	MasjidStudentUserProfileNameSnapshot              *string `gorm:"column:masjid_student_user_profile_name_snapshot;type:varchar(80)" json:"masjid_student_user_profile_name_snapshot,omitempty"`
	MasjidStudentUserProfileAvatarURLSnapshot         *string `gorm:"column:masjid_student_user_profile_avatar_url_snapshot;type:varchar(255)" json:"masjid_student_user_profile_avatar_url_snapshot,omitempty"`
	MasjidStudentUserProfileWhatsappURLSnapshot       *string `gorm:"column:masjid_student_user_profile_whatsapp_url_snapshot;type:varchar(50)" json:"masjid_student_user_profile_whatsapp_url_snapshot,omitempty"`
	MasjidStudentUserProfileParentNameSnapshot        *string `gorm:"column:masjid_student_user_profile_parent_name_snapshot;type:varchar(80)" json:"masjid_student_user_profile_parent_name_snapshot,omitempty"`
	MasjidStudentUserProfileParentWhatsappURLSnapshot *string `gorm:"column:masjid_student_user_profile_parent_whatsapp_url_snapshot;type:varchar(50)" json:"masjid_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	// ===== MASJID SNAPSHOT (/me render cepat) — sesuai SQL terbaru =====
	MasjidStudentMasjidNameSnapshot          *string `gorm:"column:masjid_student_masjid_name_snapshot;type:varchar(100)" json:"masjid_student_masjid_name_snapshot,omitempty"`
	MasjidStudentMasjidSlugSnapshot          *string `gorm:"column:masjid_student_masjid_slug_snapshot;type:varchar(100)" json:"masjid_student_masjid_slug_snapshot,omitempty"`
	MasjidStudentMasjidLogoURLSnapshot       *string `gorm:"column:masjid_student_masjid_logo_url_snapshot;type:text" json:"masjid_student_masjid_logo_url_snapshot,omitempty"`
	MasjidStudentMasjidIconURLSnapshot       *string `gorm:"column:masjid_student_masjid_icon_url_snapshot;type:text" json:"masjid_student_masjid_icon_url_snapshot,omitempty"`
	MasjidStudentMasjidBackgroundURLSnapshot *string `gorm:"column:masjid_student_masjid_background_url_snapshot;type:text" json:"masjid_student_masjid_background_url_snapshot,omitempty"`

	// ===== JSONB SECTIONS (NOT NULL DEFAULT '[]') =====
	MasjidStudentSections datatypes.JSON `gorm:"column:masjid_student_sections;type:jsonb;not null" json:"masjid_student_sections"`

	// Audit & Soft delete
	MasjidStudentCreatedAt time.Time      `gorm:"column:masjid_student_created_at;type:timestamptz;not null;default:now();autoCreateTime" json:"masjid_student_created_at"`
	MasjidStudentUpdatedAt time.Time      `gorm:"column:masjid_student_updated_at;type:timestamptz;not null;default:now();autoUpdateTime" json:"masjid_student_updated_at"`
	MasjidStudentDeletedAt gorm.DeletedAt `gorm:"column:masjid_student_deleted_at;index" json:"masjid_student_deleted_at,omitempty"`
}

func (MasjidStudentModel) TableName() string { return "masjid_students" }

// Hooks ringan (mirror aturan SQL)
func (m *MasjidStudentModel) BeforeCreate(tx *gorm.DB) error {
	// JSONB guard
	if len(m.MasjidStudentSections) == 0 {
		m.MasjidStudentSections = datatypes.JSON([]byte("[]"))
	}
	// Validasi status
	if _, ok := validMasjidStudentStatus[m.MasjidStudentStatus]; !ok {
		return errors.New("invalid masjid_student_status")
	}
	return nil
}

func (m *MasjidStudentModel) BeforeSave(tx *gorm.DB) error {
	// Validasi status
	if _, ok := validMasjidStudentStatus[m.MasjidStudentStatus]; !ok {
		return errors.New("invalid masjid_student_status")
	}
	// JSONB guard
	if len(m.MasjidStudentSections) == 0 {
		m.MasjidStudentSections = datatypes.JSON([]byte("[]"))
	}
	return nil
}
