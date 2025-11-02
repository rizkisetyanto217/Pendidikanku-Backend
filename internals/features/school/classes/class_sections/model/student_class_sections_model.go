// file: internals/features/school/academics/sections/model/student_class_section_model.go
package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ========================= ENUMS (app-level) =========================

type StudentClassSectionStatus string
type StudentClassSectionResult string

const (
	// Status (harus cocok dengan CHECK di SQL)
	StudentClassSectionActive    StudentClassSectionStatus = "active"
	StudentClassSectionInactive  StudentClassSectionStatus = "inactive"
	StudentClassSectionCompleted StudentClassSectionStatus = "completed"

	// Result (harus cocok dengan CHECK di SQL)
	StudentClassSectionPassed StudentClassSectionResult = "passed"
	StudentClassSectionFailed StudentClassSectionResult = "failed"
)

// ========================= MODEL =========================

type StudentClassSection struct {
	StudentClassSectionID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_class_section_id" json:"student_class_section_id"`

	// Identitas siswa & tenant
	StudentClassSectionSchoolStudentID uuid.UUID `gorm:"type:uuid;not null;column:student_class_section_school_student_id" json:"student_class_section_school_student_id"`
	StudentClassSectionSectionID       uuid.UUID `gorm:"type:uuid;not null;column:student_class_section_section_id" json:"student_class_section_section_id"`
	StudentClassSectionSchoolID        uuid.UUID `gorm:"type:uuid;not null;column:student_class_section_school_id" json:"student_class_section_school_id"`

	// Lifecycle enrolment
	StudentClassSectionStatus StudentClassSectionStatus  `gorm:"type:text;not null;default:'active';column:student_class_section_status" json:"student_class_section_status"`
	StudentClassSectionResult *StudentClassSectionResult `gorm:"type:text;column:student_class_section_result" json:"student_class_section_result,omitempty"`

	// Snapshot biaya (JSONB)
	StudentClassSectionFeeSnapshot datatypes.JSON `gorm:"type:jsonb;column:student_class_section_fee_snapshot" json:"student_class_section_fee_snapshot,omitempty"`

	// Snapshot users_profile (per siswa saat enrol ke section)
	StudentClassSectionUserProfileNameSnapshot              *string `gorm:"type:varchar(80);column:student_class_section_user_profile_name_snapshot" json:"student_class_section_user_profile_name_snapshot,omitempty"`
	StudentClassSectionUserProfileAvatarURLSnapshot         *string `gorm:"type:varchar(255);column:student_class_section_user_profile_avatar_url_snapshot" json:"student_class_section_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionUserProfileWhatsappURLSnapshot       *string `gorm:"type:varchar(50);column:student_class_section_user_profile_whatsapp_url_snapshot" json:"student_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileParentNameSnapshot        *string `gorm:"type:varchar(80);column:student_class_section_user_profile_parent_name_snapshot" json:"student_class_section_user_profile_parent_name_snapshot,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLSnapshot *string `gorm:"type:varchar(50);column:student_class_section_user_profile_parent_whatsapp_url_snapshot" json:"student_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	// Jejak waktu
	StudentClassSectionAssignedAt   time.Time  `gorm:"type:date;not null;default:current_date;column:student_class_section_assigned_at" json:"student_class_section_assigned_at"`
	StudentClassSectionUnassignedAt *time.Time `gorm:"type:date;column:student_class_section_unassigned_at" json:"student_class_section_unassigned_at,omitempty"`
	StudentClassSectionCompletedAt  *time.Time `gorm:"type:timestamptz;column:student_class_section_completed_at" json:"student_class_section_completed_at,omitempty"`

	// Audit
	StudentClassSectionCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:student_class_section_created_at" json:"student_class_section_created_at"`
	StudentClassSectionUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:student_class_section_updated_at" json:"student_class_section_updated_at"`
	StudentClassSectionDeletedAt gorm.DeletedAt `gorm:"index;column:student_class_section_deleted_at" json:"student_class_section_deleted_at,omitempty"`
}

func (StudentClassSection) TableName() string { return "student_class_sections" }

// ========================= Hooks (mirror CHECK constraints) =========================

// ensureConsistency memantulkan rule CHECK di SQL agar error terdeteksi sebelum kena DB.
func (s *StudentClassSection) ensureConsistency() error {
	// chk_scsec_dates: unassigned_at >= assigned_at
	if s.StudentClassSectionUnassignedAt != nil &&
		s.StudentClassSectionUnassignedAt.Before(s.StudentClassSectionAssignedAt) {
		return errors.New("student_class_section_unassigned_at must be >= student_class_section_assigned_at")
	}

	// chk_scsec_result_only_when_completed
	if s.StudentClassSectionStatus == StudentClassSectionCompleted {
		// saat completed → result wajib, completed_at wajib
		if s.StudentClassSectionResult == nil {
			return errors.New("student_class_section_result is required when status is 'completed'")
		}
		if s.StudentClassSectionCompletedAt == nil {
			return errors.New("student_class_section_completed_at is required when status is 'completed'")
		}
	} else {
		// saat bukan completed → result & completed_at harus kosong
		if s.StudentClassSectionResult != nil {
			return errors.New("student_class_section_result must be NULL when status is not 'completed'")
		}
		if s.StudentClassSectionCompletedAt != nil {
			return errors.New("student_class_section_completed_at must be NULL when status is not 'completed'")
		}
	}

	return nil
}

func (s *StudentClassSection) BeforeCreate(tx *gorm.DB) error { return s.ensureConsistency() }
func (s *StudentClassSection) BeforeUpdate(tx *gorm.DB) error { return s.ensureConsistency() }

// ========================= Helper opsional =========================

// MarkCompleted menutup enrolment dengan hasil akhir.
func (s *StudentClassSection) MarkCompleted(result StudentClassSectionResult, when time.Time) {
	s.StudentClassSectionStatus = StudentClassSectionCompleted
	s.StudentClassSectionResult = &result
	s.StudentClassSectionCompletedAt = &when
}

// ClearCompletion mengembalikan status ke non-completed.
func (s *StudentClassSection) ClearCompletion() {
	s.StudentClassSectionStatus = StudentClassSectionActive
	s.StudentClassSectionResult = nil
	s.StudentClassSectionCompletedAt = nil
}
