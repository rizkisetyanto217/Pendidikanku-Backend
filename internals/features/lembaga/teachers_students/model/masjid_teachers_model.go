package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
	=============================
	  Enum: teacher_employment_enum

=============================
*/
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

type MasjidTeacherModel struct {
	MasjidTeacherID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:masjid_teacher_id" json:"masjid_teacher_id"`

	// Scope/relasi
	MasjidTeacherMasjidID uuid.UUID `gorm:"type:uuid;not null;column:masjid_teacher_masjid_id" json:"masjid_teacher_masjid_id"`
	// DDL saat ini: kolom ini mereferensikan user_teachers(id) (bukan user_teacher_id).
	// Model mengikuti DDL apa adanya.
	MasjidTeacherUserTeacherID uuid.UUID `gorm:"type:uuid;not null;column:masjid_teacher_user_teacher_id" json:"masjid_teacher_user_teacher_id"`

	// Identitas/kepegawaian
	MasjidTeacherCode       *string            `gorm:"type:varchar(50);column:masjid_teacher_code" json:"masjid_teacher_code,omitempty"`
	MasjidTeacherSlug       *string            `gorm:"type:varchar(50);column:masjid_teacher_slug" json:"masjid_teacher_slug,omitempty"`
	MasjidTeacherEmployment *TeacherEmployment `gorm:"type:teacher_employment_enum;column:masjid_teacher_employment" json:"masjid_teacher_employment,omitempty"`
	MasjidTeacherIsActive   bool               `gorm:"type:boolean;not null;default:true;column:masjid_teacher_is_active" json:"masjid_teacher_is_active"`

	// Periode kerja
	MasjidTeacherJoinedAt *time.Time `gorm:"type:date;column:masjid_teacher_joined_at" json:"masjid_teacher_joined_at,omitempty"`
	MasjidTeacherLeftAt   *time.Time `gorm:"type:date;column:masjid_teacher_left_at" json:"masjid_teacher_left_at,omitempty"`

	// Verifikasi internal
	MasjidTeacherIsVerified bool       `gorm:"type:boolean;not null;default:false;column:masjid_teacher_is_verified" json:"masjid_teacher_is_verified"`
	MasjidTeacherVerifiedAt *time.Time `gorm:"type:timestamptz;column:masjid_teacher_verified_at" json:"masjid_teacher_verified_at,omitempty"`

	// Visibilitas & catatan
	MasjidTeacherIsPublic bool    `gorm:"type:boolean;not null;default:true;column:masjid_teacher_is_public" json:"masjid_teacher_is_public"`
	MasjidTeacherNotes    *string `gorm:"type:text;column:masjid_teacher_notes" json:"masjid_teacher_notes,omitempty"`

	// Snapshot dari user_teachers
	MasjidTeacherNameUserSnapshot        *string `gorm:"type:varchar(80);column:masjid_teacher_name_user_snapshot" json:"masjid_teacher_name_user_snapshot,omitempty"`
	MasjidTeacherAvatarURLUserSnapshot   *string `gorm:"type:varchar(255);column:masjid_teacher_avatar_url_user_snapshot" json:"masjid_teacher_avatar_url_user_snapshot,omitempty"`
	MasjidTeacherWhatsappURLUserSnapshot *string `gorm:"type:varchar(20);column:masjid_teacher_whatsapp_url_user_snapshot" json:"masjid_teacher_whatsapp_url_user_snapshot,omitempty"`
	MasjidTeacherTitlePrefixUserSnapshot *string `gorm:"type:varchar(20);column:masjid_teacher_title_prefix_user_snapshot" json:"masjid_teacher_title_prefix_user_snapshot,omitempty"`
	MasjidTeacherTitleSuffixUserSnapshot *string `gorm:"type:varchar(30);column:masjid_teacher_title_suffix_user_snapshot" json:"masjid_teacher_title_suffix_user_snapshot,omitempty"`

	// Audit & soft delete
	MasjidTeacherCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:masjid_teacher_created_at" json:"masjid_teacher_created_at"`
	MasjidTeacherUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:masjid_teacher_updated_at" json:"masjid_teacher_updated_at"`
	MasjidTeacherDeletedAt gorm.DeletedAt `gorm:"index;column:masjid_teacher_deleted_at" json:"masjid_teacher_deleted_at,omitempty"`
}

func (MasjidTeacherModel) TableName() string {
	return "masjid_teachers"
}
