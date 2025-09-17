// internals/features/lembaga/teachers/model/masjid_teacher_model.go
package model

import (
	"database/sql/driver"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
   ENUM: teacher_employment_enum  (harus sama dengan di DB)
   ========================================================= */

type TeacherEmploymentStatus string

const (
	TeacherEmploymentTetap      TeacherEmploymentStatus = "tetap"
	TeacherEmploymentKontrak    TeacherEmploymentStatus = "kontrak"
	TeacherEmploymentParuhWaktu TeacherEmploymentStatus = "paruh_waktu"
	TeacherEmploymentMagang     TeacherEmploymentStatus = "magang"
	TeacherEmploymentHonorer    TeacherEmploymentStatus = "honorer"
	TeacherEmploymentRelawan    TeacherEmploymentStatus = "relawan"
	TeacherEmploymentTamu       TeacherEmploymentStatus = "tamu"
)

// Scan & Value → jaga konsistensi lowercase + trim
func (s *TeacherEmploymentStatus) Scan(value any) error {
	switch v := value.(type) {
	case string:
		*s = TeacherEmploymentStatus(strings.ToLower(strings.TrimSpace(v)))
	case []byte:
		*s = TeacherEmploymentStatus(strings.ToLower(strings.TrimSpace(string(v))))
	case nil:
		*s = ""
	default:
		*s = TeacherEmploymentStatus(strings.ToLower(strings.TrimSpace(v.(string))))
	}
	return nil
}
func (s TeacherEmploymentStatus) Value() (driver.Value, error) {
	return string(TeacherEmploymentStatus(strings.ToLower(strings.TrimSpace(string(s))))), nil
}

/* =========================================================
   MODEL: masjid_teachers
   ========================================================= */

type MasjidTeacherModel struct {
	// PK
	MasjidTeacherID uuid.UUID `gorm:"column:masjid_teacher_id;type:uuid;primaryKey;default:gen_random_uuid()" json:"masjid_teacher_id"`

	// Scope/Relasi
	MasjidTeacherMasjidID uuid.UUID `gorm:"column:masjid_teacher_masjid_id;type:uuid;not null;index" json:"masjid_teacher_masjid_id"`
	MasjidTeacherUserID   uuid.UUID `gorm:"column:masjid_teacher_user_id;type:uuid;not null;index"   json:"masjid_teacher_user_id"`

	// Identitas/Kepegawaian
	MasjidTeacherCode       *string                  `gorm:"column:masjid_teacher_code;type:varchar(50)"      json:"masjid_teacher_code,omitempty"`
	MasjidTeacherSlug       *string                  `gorm:"column:masjid_teacher_slug;type:varchar(50)"      json:"masjid_teacher_slug,omitempty"`

	MasjidTeacherEmployment *TeacherEmploymentStatus `gorm:"column:masjid_teacher_employment;type:teacher_employment_enum" json:"masjid_teacher_employment,omitempty"`
	MasjidTeacherIsActive   bool                     `gorm:"column:masjid_teacher_is_active;not null;default:true"        json:"masjid_teacher_is_active"`

	// Periode kerja
	MasjidTeacherJoinedAt *time.Time `gorm:"column:masjid_teacher_joined_at;type:date" json:"masjid_teacher_joined_at,omitempty"`
	MasjidTeacherLeftAt   *time.Time `gorm:"column:masjid_teacher_left_at;type:date"   json:"masjid_teacher_left_at,omitempty"`

	// Verifikasi internal
	MasjidTeacherIsVerified bool       `gorm:"column:masjid_teacher_is_verified;not null;default:false" json:"masjid_teacher_is_verified"`
	MasjidTeacherVerifiedAt *time.Time `gorm:"column:masjid_teacher_verified_at"                        json:"masjid_teacher_verified_at,omitempty"`

	// Visibilitas & catatan
	MasjidTeacherIsPublic bool    `gorm:"column:masjid_teacher_is_public;not null;default:true" json:"masjid_teacher_is_public"`
	MasjidTeacherNotes    *string `gorm:"column:masjid_teacher_notes"                           json:"masjid_teacher_notes,omitempty"`

	// Audit
	MasjidTeacherCreatedAt time.Time      `gorm:"column:masjid_teacher_created_at;autoCreateTime" json:"masjid_teacher_created_at"`
	MasjidTeacherUpdatedAt time.Time      `gorm:"column:masjid_teacher_updated_at;autoUpdateTime"  json:"masjid_teacher_updated_at"`
	MasjidTeacherDeletedAt gorm.DeletedAt `gorm:"column:masjid_teacher_deleted_at;index"           json:"masjid_teacher_deleted_at,omitempty"`
}

func (MasjidTeacherModel) TableName() string { return "masjid_teachers" }
