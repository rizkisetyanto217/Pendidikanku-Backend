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

// (Opsional) tipe item untuk bantu manipulasi sections di service layer
type MasjidStudentSectionItem struct {
	ClassSectionID             uuid.UUID `json:"class_section_id"`
	IsActive                   bool      `json:"is_active"`
	From                       *string   `json:"from"` // YYYY-MM-DD
	To                         *string   `json:"to"`   // YYYY-MM-DD | null
	ClassSectionName           *string   `json:"class_section_name"`
	ClassSectionSlug           *string   `json:"class_section_slug"`
	ClassSectionImageURL       *string   `json:"class_section_image_url"`
	ClassSectionImageObjectKey *string   `json:"class_section_image_object_key"`
}

// MasjidStudentModel merepresentasikan tabel masjid_students
type MasjidStudentModel struct {
	// ============== PK & Tenant ==============
	MasjidStudentID       uuid.UUID `gorm:"column:masjid_student_id;type:uuid;default:gen_random_uuid();primaryKey" json:"masjid_student_id"`
	MasjidStudentMasjidID uuid.UUID `gorm:"column:masjid_student_masjid_id;type:uuid;not null;index" json:"masjid_student_masjid_id"`

	// Relasi ke users_profile
	MasjidStudentUserProfileID uuid.UUID `gorm:"column:masjid_student_user_profile_id;type:uuid;not null;index" json:"masjid_student_user_profile_id"`

	// Identitas internal (unik per masjid dikelola oleh SQL migration)
	MasjidStudentSlug string  `gorm:"column:masjid_student_slug;type:varchar(50);not null" json:"masjid_student_slug"`
	MasjidStudentCode *string `gorm:"column:masjid_student_code;type:varchar(50)" json:"masjid_student_code,omitempty"`

	// Status (active, inactive, alumni)
	MasjidStudentStatus MasjidStudentStatus `gorm:"column:masjid_student_status;type:text;not null;default:'active'" json:"masjid_student_status"`

	// Operasional
	MasjidStudentJoinedAt *time.Time `gorm:"column:masjid_student_joined_at;type:timestamptz" json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `gorm:"column:masjid_student_left_at;type:timestamptz" json:"masjid_student_left_at,omitempty"`

	// Catatan umum santri
	MasjidStudentNote *string `gorm:"column:masjid_student_note;type:text" json:"masjid_student_note,omitempty"`

	// ============== SNAPSHOTS (dari users_profile) ==============
	MasjidStudentUserProfileNameSnapshot              *string `gorm:"column:masjid_student_user_profile_name_snapshot;type:varchar(80)" json:"masjid_student_user_profile_name_snapshot,omitempty"`
	MasjidStudentUserProfileAvatarURLSnapshot         *string `gorm:"column:masjid_student_user_profile_avatar_url_snapshot;type:varchar(255)" json:"masjid_student_user_profile_avatar_url_snapshot,omitempty"`
	MasjidStudentUserProfileWhatsappURLSnapshot       *string `gorm:"column:masjid_student_user_profile_whatsapp_url_snapshot;type:varchar(50)" json:"masjid_student_user_profile_whatsapp_url_snapshot,omitempty"`
	MasjidStudentUserProfileParentNameSnapshot        *string `gorm:"column:masjid_student_user_profile_parent_name_snapshot;type:varchar(80)" json:"masjid_student_user_profile_parent_name_snapshot,omitempty"`
	MasjidStudentUserProfileParentWhatsappURLSnapshot *string `gorm:"column:masjid_student_user_profile_parent_whatsapp_url_snapshot;type:varchar(50)" json:"masjid_student_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	// ============== MASJID SNAPSHOT (untuk render cepat /me) ==============
	MasjidStudentMasjidNameSnapshot    *string `gorm:"column:masjid_student_masjid_name_snapshot;type:varchar(100)" json:"masjid_student_masjid_name_snapshot,omitempty"`
	MasjidStudentMasjidSlugSnapshot    *string `gorm:"column:masjid_student_masjid_slug_snapshot;type:varchar(100)" json:"masjid_student_masjid_slug_snapshot,omitempty"`
	MasjidStudentMasjidLogoURLSnapshot *string `gorm:"column:masjid_student_masjid_logo_url_snapshot;type:text" json:"masjid_student_masjid_logo_url_snapshot,omitempty"`

	// ============== JSONB SECTIONS (dipelihara backend) ==============
	MasjidStudentSections datatypes.JSON `gorm:"column:masjid_student_sections;type:jsonb;not null;default:'[]'::jsonb" json:"masjid_student_sections"`

	// Audit
	MasjidStudentCreatedAt time.Time      `gorm:"column:masjid_student_created_at;not null;autoCreateTime" json:"masjid_student_created_at"`
	MasjidStudentUpdatedAt time.Time      `gorm:"column:masjid_student_updated_at;not null;autoUpdateTime" json:"masjid_student_updated_at"`
	MasjidStudentDeletedAt gorm.DeletedAt `gorm:"column:masjid_student_deleted_at;index" json:"masjid_student_deleted_at,omitempty"`
}

// TableName override
func (MasjidStudentModel) TableName() string {
	return "masjid_students"
}

// ============== Hooks ringan (mirror CHECK & default JSON) ==============
func (m *MasjidStudentModel) BeforeSave(tx *gorm.DB) error {
	if _, ok := validMasjidStudentStatus[m.MasjidStudentStatus]; !ok {
		return errors.New("invalid masjid_student_status")
	}
	if len(m.MasjidStudentSections) == 0 {
		m.MasjidStudentSections = datatypes.JSON([]byte("[]"))
	}
	return nil
}
