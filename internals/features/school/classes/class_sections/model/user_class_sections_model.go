// file: internals/features/school/academics/sections/model/user_class_section_model.go
package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ========================= ENUMS (app-level) =========================

type UserClassSectionStatus string
type UserClassSectionResult string

const (
	// Status (harus cocok dengan CHECK di SQL)
	UserClassSectionActive    UserClassSectionStatus = "active"
	UserClassSectionInactive  UserClassSectionStatus = "inactive"
	UserClassSectionCompleted UserClassSectionStatus = "completed"

	// Result (harus cocok dengan CHECK di SQL)
	UserClassSectionPassed UserClassSectionResult = "passed"
	UserClassSectionFailed UserClassSectionResult = "failed"
)

// ========================= MODEL =========================

type UserClassSection struct {
	UserClassSectionID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_section_id" json:"user_class_section_id"`

	// Identitas siswa & tenant
	UserClassSectionMasjidStudentID uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_masjid_student_id" json:"user_class_section_masjid_student_id"`
	UserClassSectionSectionID       uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_section_id" json:"user_class_section_section_id"`
	UserClassSectionMasjidID        uuid.UUID `gorm:"type:uuid;not null;column:user_class_section_masjid_id" json:"user_class_section_masjid_id"`

	// Lifecycle enrolment
	UserClassSectionStatus UserClassSectionStatus  `gorm:"type:text;not null;default:'active';column:user_class_section_status" json:"user_class_section_status"`
	UserClassSectionResult *UserClassSectionResult `gorm:"type:text;column:user_class_section_result" json:"user_class_section_result,omitempty"`

	// Snapshot biaya (JSONB)
	UserClassSectionFeeSnapshot datatypes.JSON `gorm:"type:jsonb;column:user_class_section_fee_snapshot" json:"user_class_section_fee_snapshot,omitempty"`

	// Snapshot users_profile (per siswa saat enrol ke section)
	UserClassSectionUserProfileNameSnapshot                *string `gorm:"type:varchar(80);column:user_class_section_user_profile_name_snapshot" json:"user_class_section_user_profile_name_snapshot,omitempty"`
	UserClassSectionUserProfileAvatarURLSnapshot          *string `gorm:"type:varchar(255);column:user_class_section_user_profile_avatar_url_snapshot" json:"user_class_section_user_profile_avatar_url_snapshot,omitempty"`
	UserClassSectionUserProfileWhatsappURLSnapshot        *string `gorm:"type:varchar(50);column:user_class_section_user_profile_whatsapp_url_snapshot" json:"user_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	UserClassSectionUserProfileParentNameSnapshot         *string `gorm:"type:varchar(80);column:user_class_section_user_profile_parent_name_snapshot" json:"user_class_section_user_profile_parent_name_snapshot,omitempty"`
	UserClassSectionUserProfileParentWhatsappURLSnapshot  *string `gorm:"type:varchar(50);column:user_class_section_user_profile_parent_whatsapp_url_snapshot" json:"user_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	// Jejak waktu
	UserClassSectionAssignedAt   time.Time  `gorm:"type:date;not null;default:current_date;column:user_class_section_assigned_at" json:"user_class_section_assigned_at"`
	UserClassSectionUnassignedAt *time.Time `gorm:"type:date;column:user_class_section_unassigned_at" json:"user_class_section_unassigned_at,omitempty"`
	UserClassSectionCompletedAt  *time.Time `gorm:"type:timestamptz;column:user_class_section_completed_at" json:"user_class_section_completed_at,omitempty"`

	// Audit
	UserClassSectionCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:user_class_section_created_at" json:"user_class_section_created_at"`
	UserClassSectionUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:user_class_section_updated_at" json:"user_class_section_updated_at"`
	UserClassSectionDeletedAt gorm.DeletedAt `gorm:"index;column:user_class_section_deleted_at" json:"user_class_section_deleted_at,omitempty"`
}

func (UserClassSection) TableName() string { return "user_class_sections" }

// ========================= Hooks (mirror CHECK constraints) =========================

// ensureConsistency memantulkan rule CHECK di SQL agar error terdeteksi sebelum kena DB.
func (u *UserClassSection) ensureConsistency() error {
	// chk_ucsec_dates: unassigned_at >= assigned_at
	if u.UserClassSectionUnassignedAt != nil &&
		u.UserClassSectionUnassignedAt.Before(u.UserClassSectionAssignedAt) {
		return errors.New("user_class_section_unassigned_at must be >= user_class_section_assigned_at")
	}

	// chk_ucsec_result_only_when_completed
	if u.UserClassSectionStatus == UserClassSectionCompleted {
		// saat completed → result wajib, completed_at wajib
		if u.UserClassSectionResult == nil {
			return errors.New("user_class_section_result is required when status is 'completed'")
		}
		if u.UserClassSectionCompletedAt == nil {
			return errors.New("user_class_section_completed_at is required when status is 'completed'")
		}
	} else {
		// saat bukan completed → result & completed_at harus kosong
		if u.UserClassSectionResult != nil {
			return errors.New("user_class_section_result must be NULL when status is not 'completed'")
		}
		if u.UserClassSectionCompletedAt != nil {
			return errors.New("user_class_section_completed_at must be NULL when status is not 'completed'")
		}
	}

	return nil
}

func (u *UserClassSection) BeforeCreate(tx *gorm.DB) error { return u.ensureConsistency() }
func (u *UserClassSection) BeforeUpdate(tx *gorm.DB) error { return u.ensureConsistency() }

// ========================= Helper opsional =========================

// MarkCompleted menutup enrolment dengan hasil akhir.
func (u *UserClassSection) MarkCompleted(result UserClassSectionResult, when time.Time) {
	u.UserClassSectionStatus = UserClassSectionCompleted
	u.UserClassSectionResult = &result
	u.UserClassSectionCompletedAt = &when
}

// ClearCompletion mengembalikan status ke non-completed.
func (u *UserClassSection) ClearCompletion() {
	u.UserClassSectionStatus = UserClassSectionActive
	u.UserClassSectionResult = nil
	u.UserClassSectionCompletedAt = nil
}
