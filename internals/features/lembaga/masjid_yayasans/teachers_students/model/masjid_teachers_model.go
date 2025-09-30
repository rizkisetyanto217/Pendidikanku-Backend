package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/*
=============================
  Enum: teacher_employment_enum
  (sinkron dengan enum di PostgreSQL)
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

// Driver/Scanner supaya enum Go <-> DB enum mulus
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

/*
=============================
  Model: masjid_teachers
  (sesuai SQL terbaru)
=============================
*/

type MasjidTeacherModel struct {
	// ============== PK ==============
	MasjidTeacherID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:masjid_teacher_id" json:"masjid_teacher_id"`

	// ============== Scope / Relasi ==============
	MasjidTeacherMasjidID      uuid.UUID `gorm:"type:uuid;not null;column:masjid_teacher_masjid_id" json:"masjid_teacher_masjid_id"`
	MasjidTeacherUserTeacherID uuid.UUID `gorm:"type:uuid;not null;column:masjid_teacher_user_teacher_id" json:"masjid_teacher_user_teacher_id"`

	// ============== Identitas / Kepegawaian ==============
	MasjidTeacherCode       *string            `gorm:"type:varchar(50);column:masjid_teacher_code" json:"masjid_teacher_code,omitempty"`
	MasjidTeacherSlug       *string            `gorm:"type:varchar(50);column:masjid_teacher_slug" json:"masjid_teacher_slug,omitempty"`
	MasjidTeacherEmployment *TeacherEmployment `gorm:"type:teacher_employment_enum;column:masjid_teacher_employment" json:"masjid_teacher_employment,omitempty"`
	MasjidTeacherIsActive   bool               `gorm:"type:boolean;not null;default:true;column:masjid_teacher_is_active" json:"masjid_teacher_is_active"`

	// ============== Periode Kerja ==============
	MasjidTeacherJoinedAt *time.Time `gorm:"type:date;column:masjid_teacher_joined_at" json:"masjid_teacher_joined_at,omitempty"`
	MasjidTeacherLeftAt   *time.Time `gorm:"type:date;column:masjid_teacher_left_at" json:"masjid_teacher_left_at,omitempty"`

	// ============== Verifikasi ==============
	MasjidTeacherIsVerified bool       `gorm:"type:boolean;not null;default:false;column:masjid_teacher_is_verified" json:"masjid_teacher_is_verified"`
	MasjidTeacherVerifiedAt *time.Time `gorm:"type:timestamptz;column:masjid_teacher_verified_at" json:"masjid_teacher_verified_at,omitempty"`

	// ============== Visibilitas & Catatan ==============
	MasjidTeacherIsPublic bool    `gorm:"type:boolean;not null;default:true;column:masjid_teacher_is_public" json:"masjid_teacher_is_public"`
	MasjidTeacherNotes    *string `gorm:"type:text;column:masjid_teacher_notes" json:"masjid_teacher_notes,omitempty"`

	// ============== Snapshot dari user_teachers ==============
	MasjidTeacherUserTeacherNameSnapshot        *string `gorm:"type:varchar(80);column:masjid_teacher_user_teacher_name_snapshot" json:"masjid_teacher_user_teacher_name_snapshot,omitempty"`
	MasjidTeacherUserTeacherAvatarURLSnapshot   *string `gorm:"type:varchar(255);column:masjid_teacher_user_teacher_avatar_url_snapshot" json:"masjid_teacher_user_teacher_avatar_url_snapshot,omitempty"`
	MasjidTeacherUserTeacherWhatsappURLSnapshot *string `gorm:"type:varchar(50);column:masjid_teacher_user_teacher_whatsapp_url_snapshot" json:"masjid_teacher_user_teacher_whatsapp_url_snapshot,omitempty"`
	MasjidTeacherUserTeacherTitlePrefixSnapshot *string `gorm:"type:varchar(20);column:masjid_teacher_user_teacher_title_prefix_snapshot" json:"masjid_teacher_user_teacher_title_prefix_snapshot,omitempty"`
	MasjidTeacherUserTeacherTitleSuffixSnapshot *string `gorm:"type:varchar(30);column:masjid_teacher_user_teacher_title_suffix_snapshot" json:"masjid_teacher_user_teacher_title_suffix_snapshot,omitempty"`

	// ============== MASJID SNAPSHOT (untuk render cepat /me) ==============
	MasjidTeacherMasjidNameSnapshot    *string `gorm:"type:varchar(100);column:masjid_teacher_masjid_name_snapshot" json:"masjid_teacher_masjid_name_snapshot,omitempty"`
	MasjidTeacherMasjidSlugSnapshot    *string `gorm:"type:varchar(100);column:masjid_teacher_masjid_slug_snapshot" json:"masjid_teacher_masjid_slug_snapshot,omitempty"`
	MasjidTeacherMasjidLogoURLSnapshot *string `gorm:"type:text;column:masjid_teacher_masjid_logo_url_snapshot" json:"masjid_teacher_masjid_logo_url_snapshot,omitempty"`

	// ============== JSONB (sections & csst) ==============
	// default di DB: '[]'::jsonb — pastikan backend isi [] saat create supaya tidak NULL
	MasjidTeacherSections datatypes.JSON `gorm:"type:jsonb;not null;default:'[]';column:masjid_teacher_sections" json:"masjid_teacher_sections"`
	MasjidTeacherCSST     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]';column:masjid_teacher_csst" json:"masjid_teacher_csst"`

	// ============== Audit & Soft Delete ==============
	MasjidTeacherCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:masjid_teacher_created_at" json:"masjid_teacher_created_at"`
	MasjidTeacherUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:masjid_teacher_updated_at" json:"masjid_teacher_updated_at"`
	MasjidTeacherDeletedAt gorm.DeletedAt `gorm:"index;column:masjid_teacher_deleted_at" json:"masjid_teacher_deleted_at,omitempty"`
}

func (MasjidTeacherModel) TableName() string { return "masjid_teachers" }

/*
Opsional: tipe helper untuk isi JSONB agar strongly-typed di kode aplikasi.
Kamu bisa marshal/unmarshal ke datatypes.JSON pakai json.Marshal/json.Unmarshal.
*/

type TeacherSectionItem struct {
	ClassSectionID             uuid.UUID `json:"class_section_id"`
	Role                       string    `json:"role"` // "homeroom" | "teacher" | "assistant"
	IsActive                   bool      `json:"is_active"`
	From                       *string   `json:"from,omitempty"` // "YYYY-MM-DD"
	To                         *string   `json:"to,omitempty"`   // "YYYY-MM-DD"
	ClassSectionName           *string   `json:"class_section_name,omitempty"`
	ClassSectionSlug           *string   `json:"class_section_slug,omitempty"`
	ClassSectionImageURL       *string   `json:"class_section_image_url,omitempty"`
	ClassSectionImageObjectKey *string   `json:"class_section_image_object_key,omitempty"`
}

type TeacherCSSTItem struct {
	CSSTID           uuid.UUID  `json:"csst_id"`
	IsActive         bool       `json:"is_active"`
	From             *string    `json:"from,omitempty"`
	To               *string    `json:"to,omitempty"`
	SubjectName      *string    `json:"subject_name,omitempty"`
	SubjectSlug      *string    `json:"subject_slug,omitempty"`
	ClassSectionID   *uuid.UUID `json:"class_section_id,omitempty"`
	ClassSectionName *string    `json:"class_section_name,omitempty"`
	ClassSectionSlug *string    `json:"class_section_slug,omitempty"`
}
