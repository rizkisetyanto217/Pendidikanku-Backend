// file: internals/features/school/sectionsubjectteachers/model/student_class_section_subject_teacher_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Masih pakai nama type lama biar nggak rusak import di tempat lain,
// tapi field-field disesuaikan ke kolom student_csst_* di DB.
type StudentClassSectionSubjectTeacherModel struct {
	// PK
	StudentCSSTID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_csst_id" json:"student_csst_id"`

	// Tenant
	StudentCSSTSchoolID uuid.UUID `gorm:"type:uuid;not null;column:student_csst_school_id" json:"student_csst_school_id"`

	// Anchor relations
	StudentCSSTStudentID uuid.UUID `gorm:"type:uuid;not null;column:student_csst_student_id" json:"student_csst_student_id"`
	StudentCSSTCSSTID    uuid.UUID `gorm:"type:uuid;not null;column:student_csst_csst_id" json:"student_csst_csst_id"`

	// Status mapping
	StudentCSSTIsActive bool       `gorm:"not null;default:true;column:student_csst_is_active" json:"student_csst_is_active"`
	StudentCSSTFrom     *time.Time `gorm:"type:date;column:student_csst_from" json:"student_csst_from,omitempty"`
	StudentCSSTTo       *time.Time `gorm:"type:date;column:student_csst_to" json:"student_csst_to,omitempty"`

	// Nilai terbaru (opsional; percent di-generate di DB)
	StudentCSSTScoreTotal    *float64 `gorm:"type:numeric(6,2);column:student_csst_score_total" json:"student_csst_score_total,omitempty"`
	StudentCSSTScoreMaxTotal *float64 `gorm:"type:numeric(6,2);default:100;column:student_csst_score_max_total" json:"student_csst_score_max_total,omitempty"`
	StudentCSSTScorePercent  *float64 `gorm:"type:numeric(5,2);column:student_csst_score_percent;->" json:"student_csst_score_percent,omitempty"` // generated always as
	StudentCSSTGradeLetter   *string  `gorm:"type:varchar(8);column:student_csst_grade_letter" json:"student_csst_grade_letter,omitempty"`
	StudentCSSTGradePoint    *float64 `gorm:"type:numeric(3,2);column:student_csst_grade_point" json:"student_csst_grade_point,omitempty"`
	StudentCSSTIsPassed      *bool    `gorm:"column:student_csst_is_passed" json:"student_csst_is_passed,omitempty"`

	// Cache users_profile & siswa (saat enrol)
	StudentCSSTNameCache        *string `gorm:"type:varchar(80);column:student_csst_name_cache" json:"student_csst_name_cache,omitempty"`
	StudentCSSTAvatarURLCache   *string `gorm:"type:varchar(255);column:student_csst_avatar_url_cache" json:"student_csst_avatar_url_cache,omitempty"`
	StudentCSSTWAURLCache       *string `gorm:"type:varchar(50);column:student_csst_wa_url_cache" json:"student_csst_wa_url_cache,omitempty"`
	StudentCSSTParentNameCache  *string `gorm:"type:varchar(80);column:student_csst_parent_name_cache" json:"student_csst_parent_name_cache,omitempty"`
	StudentCSSTParentWAURLCache *string `gorm:"type:varchar(50);column:student_csst_parent_wa_url_cache" json:"student_csst_parent_wa_url_cache,omitempty"`
	StudentCSSTGenderCache      *string `gorm:"type:varchar(20);column:student_csst_gender_cache" json:"student_csst_gender_cache,omitempty"`
	StudentCSSTStudentCodeCache *string `gorm:"type:varchar(50);column:student_csst_student_code_cache" json:"student_csst_student_code_cache,omitempty"`

	// Riwayat intervensi/remedial (append-only JSONB)
	StudentCSSTEditsHistory datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:student_csst_edits_history" json:"student_csst_edits_history"`

	// NOTES
	StudentCSSTStudentNotes                 *string    `gorm:"type:text;column:student_csst_student_notes" json:"student_csst_student_notes,omitempty"`
	StudentCSSTStudentNotesUpdatedAt        *time.Time `gorm:"type:timestamptz;column:student_csst_student_notes_updated_at" json:"student_csst_student_notes_updated_at,omitempty"`
	StudentCSSTHomeroomNotes                *string    `gorm:"type:text;column:student_csst_homeroom_notes" json:"student_csst_homeroom_notes,omitempty"`
	StudentCSSTHomeroomNotesUpdatedAt       *time.Time `gorm:"type:timestamptz;column:student_csst_homeroom_notes_updated_at" json:"student_csst_homeroom_notes_updated_at,omitempty"`
	StudentCSSTSubjectTeacherNotes          *string    `gorm:"type:text;column:student_csst_subject_teacher_notes" json:"student_csst_subject_teacher_notes,omitempty"`
	StudentCSSTSubjectTeacherNotesUpdatedAt *time.Time `gorm:"type:timestamptz;column:student_csst_subject_teacher_notes_updated_at" json:"student_csst_subject_teacher_notes_updated_at,omitempty"`

	// Admin & meta
	StudentCSSTSlug *string        `gorm:"type:varchar(160);column:student_csst_slug" json:"student_csst_slug,omitempty"`
	StudentCSSTMeta datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'::jsonb;column:student_csst_meta" json:"student_csst_meta"`

	// Audit & soft delete
	StudentCSSTCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_csst_created_at" json:"student_csst_created_at"`
	StudentCSSTUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_csst_updated_at" json:"student_csst_updated_at"`
	StudentCSSTDeletedAt gorm.DeletedAt `gorm:"column:student_csst_deleted_at;index" json:"student_csst_deleted_at,omitempty"`
}

func (StudentClassSectionSubjectTeacherModel) TableName() string {
	return "student_class_section_subject_teachers"
}
