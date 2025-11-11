// file: internals/features/school/academics/sections/model/student_class_section_model.go
package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
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
	StudentClassSectionSchoolID        uuid.UUID `gorm:"type:uuid;not null;column:student_class_section_school_id" json:"student_class_section_school_id"`

	// Section & snapshot slug
	StudentClassSectionSectionID           uuid.UUID `gorm:"type:uuid;not null;column:student_class_section_section_id" json:"student_class_section_section_id"`
	StudentClassSectionSectionSlugSnapshot string    `gorm:"type:varchar(160);not null;column:student_class_section_section_slug_snapshot" json:"student_class_section_section_slug_snapshot"`

	// Lifecycle enrolment
	StudentClassSectionStatus StudentClassSectionStatus  `gorm:"type:text;not null;default:'active';column:student_class_section_status" json:"student_class_section_status"`
	StudentClassSectionResult *StudentClassSectionResult `gorm:"type:text;column:student_class_section_result" json:"student_class_section_result,omitempty"`

	// ==========================
	// NILAI AKHIR (grades)
	// ==========================
	StudentClassSectionFinalScore        *float64   `gorm:"type:numeric(5,2);column:student_class_section_final_score" json:"student_class_section_final_score,omitempty"` // 0..100
	StudentClassSectionFinalGradeLetter  *string    `gorm:"type:varchar(3);column:student_class_section_final_grade_letter" json:"student_class_section_final_grade_letter,omitempty"`
	StudentClassSectionFinalGradePoint   *float64   `gorm:"type:numeric(3,2);column:student_class_section_final_grade_point" json:"student_class_section_final_grade_point,omitempty"` // 0..4
	StudentClassSectionFinalRank         *int       `gorm:"type:int;column:student_class_section_final_rank" json:"student_class_section_final_rank,omitempty"`                        // > 0
	StudentClassSectionFinalRemarks      *string    `gorm:"type:text;column:student_class_section_final_remarks" json:"student_class_section_final_remarks,omitempty"`
	StudentClassSectionGradedByTeacherID *uuid.UUID `gorm:"type:uuid;column:student_class_section_graded_by_teacher_id" json:"student_class_section_graded_by_teacher_id,omitempty"`
	StudentClassSectionGradedAt          *time.Time `gorm:"type:timestamptz;column:student_class_section_graded_at" json:"student_class_section_graded_at,omitempty"`

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

func (s *StudentClassSection) ensureConsistency() error {
	// chk_scsec_dates: unassigned_at >= assigned_at
	if s.StudentClassSectionUnassignedAt != nil &&
		s.StudentClassSectionUnassignedAt.Before(s.StudentClassSectionAssignedAt) {
		return errors.New("student_class_section_unassigned_at must be >= student_class_section_assigned_at")
	}

	// Completed → result & completed_at wajib
	if s.StudentClassSectionStatus == StudentClassSectionCompleted {
		if s.StudentClassSectionResult == nil {
			return errors.New("student_class_section_result is required when status is 'completed'")
		}
		if s.StudentClassSectionCompletedAt == nil {
			return errors.New("student_class_section_completed_at is required when status is 'completed'")
		}
		// At least one of score/letter/point is present
		if s.StudentClassSectionFinalScore == nil &&
			s.StudentClassSectionFinalGradeLetter == nil &&
			s.StudentClassSectionFinalGradePoint == nil {
			return errors.New("at least one of final_score, final_grade_letter, or final_grade_point is required when status is 'completed'")
		}
	} else {
		// Non-completed → fields ini harus NULL (mirror chk_scsec_grades_only_when_completed)
		if s.StudentClassSectionFinalScore != nil ||
			s.StudentClassSectionFinalGradeLetter != nil ||
			s.StudentClassSectionFinalGradePoint != nil ||
			s.StudentClassSectionFinalRank != nil ||
			s.StudentClassSectionFinalRemarks != nil ||
			s.StudentClassSectionGradedAt != nil {
			return errors.New("final grade fields must be NULL when status is not 'completed'")
		}
		if s.StudentClassSectionResult != nil {
			return errors.New("student_class_section_result must be NULL when status is not 'completed'")
		}
		if s.StudentClassSectionCompletedAt != nil {
			return errors.New("student_class_section_completed_at must be NULL when status is not 'completed'")
		}
	}

	// Optional: validate ranges to fail fast before DB check
	if s.StudentClassSectionFinalScore != nil {
		if *s.StudentClassSectionFinalScore < 0 || *s.StudentClassSectionFinalScore > 100 {
			return errors.New("final_score must be between 0 and 100")
		}
	}
	if s.StudentClassSectionFinalGradePoint != nil {
		if *s.StudentClassSectionFinalGradePoint < 0 || *s.StudentClassSectionFinalGradePoint > 4 {
			return errors.New("final_grade_point must be between 0 and 4")
		}
	}
	if s.StudentClassSectionFinalRank != nil && *s.StudentClassSectionFinalRank <= 0 {
		return errors.New("final_rank must be > 0")
	}

	return nil
}

func (s *StudentClassSection) BeforeCreate(tx *gorm.DB) error { return s.ensureConsistency() }
func (s *StudentClassSection) BeforeUpdate(tx *gorm.DB) error { return s.ensureConsistency() }

// ========================= Helper opsional =========================

// MarkCompleted menutup enrolment dengan hasil akhir minimal (result + salah satu metrik nilai).
func (s *StudentClassSection) MarkCompleted(result StudentClassSectionResult, when time.Time) {
	s.StudentClassSectionStatus = StudentClassSectionCompleted
	s.StudentClassSectionResult = &result
	s.StudentClassSectionCompletedAt = &when
}

// SetFinalGrades mengisi nilai akhir (opsional pilih mana yang ada).
func (s *StudentClassSection) SetFinalGrades(score *float64, letter *string, point *float64, rank *int, remarks *string, gradedBy *uuid.UUID, gradedAt *time.Time) {
	s.StudentClassSectionFinalScore = score
	s.StudentClassSectionFinalGradeLetter = letter
	s.StudentClassSectionFinalGradePoint = point
	s.StudentClassSectionFinalRank = rank
	s.StudentClassSectionFinalRemarks = remarks
	s.StudentClassSectionGradedByTeacherID = gradedBy
	s.StudentClassSectionGradedAt = gradedAt
}

// ClearCompletion mengembalikan status ke non-completed dan membersihkan kolom final grade.
func (s *StudentClassSection) ClearCompletion() {
	s.StudentClassSectionStatus = StudentClassSectionActive
	s.StudentClassSectionResult = nil
	s.StudentClassSectionCompletedAt = nil

	s.StudentClassSectionFinalScore = nil
	s.StudentClassSectionFinalGradeLetter = nil
	s.StudentClassSectionFinalGradePoint = nil
	s.StudentClassSectionFinalRank = nil
	s.StudentClassSectionFinalRemarks = nil
	s.StudentClassSectionGradedByTeacherID = nil
	s.StudentClassSectionGradedAt = nil
}
