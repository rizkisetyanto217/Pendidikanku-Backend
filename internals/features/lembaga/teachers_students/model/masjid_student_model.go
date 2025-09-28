package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidStudentStatus string

const (
	MasjidStudentActive   MasjidStudentStatus = "active"
	MasjidStudentInactive MasjidStudentStatus = "inactive"
	MasjidStudentAlumni   MasjidStudentStatus = "alumni"
)

type MasjidStudent struct {
	MasjidStudentID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:masjid_student_id" json:"masjid_student_id"`

	MasjidStudentMasjidID uuid.UUID `gorm:"type:uuid;not null;column:masjid_student_masjid_id" json:"masjid_student_masjid_id"`
	MasjidStudentUserID   uuid.UUID `gorm:"type:uuid;not null;column:masjid_student_user_id" json:"masjid_student_user_id"`

	MasjidStudentSlug string  `gorm:"type:varchar(50);uniqueIndex;not null;column:masjid_student_slug" json:"masjid_student_slug"`
	MasjidStudentCode *string `gorm:"type:varchar(50);column:masjid_student_code" json:"masjid_student_code,omitempty"`

	MasjidStudentStatus MasjidStudentStatus `gorm:"type:text;not null;default:'active';column:masjid_student_status" json:"masjid_student_status"`

	// Operasional
	MasjidStudentJoinedAt *time.Time `gorm:"type:timestamptz;column:masjid_student_joined_at" json:"masjid_student_joined_at,omitempty"`
	MasjidStudentLeftAt   *time.Time `gorm:"type:timestamptz;column:masjid_student_left_at" json:"masjid_student_left_at,omitempty"`

	// Catatan umum
	MasjidStudentNote *string `gorm:"type:text;column:masjid_student_note" json:"masjid_student_note,omitempty"`

	// Snapshots (ringkasan untuk lookup cepat)
	MasjidStudentNameUserSnapshot              *string `gorm:"type:varchar(80);column:masjid_student_name_user_snapshot" json:"masjid_student_name_user_snapshot,omitempty"`
	MasjidStudentAvatarURLUserSnapshot         *string `gorm:"type:varchar(255);column:masjid_student_avatar_url_user_snapshot" json:"masjid_student_avatar_url_user_snapshot,omitempty"`
	MasjidStudentWhatsappURLUserSnapshot       *string `gorm:"type:varchar(50);column:masjid_student_whatsapp_url_user_snapshot" json:"masjid_student_whatsapp_url_user_snapshot,omitempty"`
	MasjidStudentParentNameUserSnapshot        *string `gorm:"type:varchar(80);column:masjid_student_parent_name_user_snapshot" json:"masjid_student_parent_name_user_snapshot,omitempty"`
	MasjidStudentParentWhatsappURLUserSnapshot *string `gorm:"type:varchar(50);column:masjid_student_parent_whatsapp_url_user_snapshot" json:"masjid_student_parent_whatsapp_url_user_snapshot,omitempty"`

	// Audit & soft delete
	MasjidStudentCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:masjid_student_created_at" json:"masjid_student_created_at"`
	MasjidStudentUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:masjid_student_updated_at" json:"masjid_student_updated_at"`
	MasjidStudentDeletedAt gorm.DeletedAt `gorm:"index;column:masjid_student_deleted_at" json:"masjid_student_deleted_at,omitempty"`
}

func (MasjidStudent) TableName() string {
	return "masjid_students"
}
