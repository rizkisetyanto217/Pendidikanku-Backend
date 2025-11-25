// file: internals/features/school/sectionsubjectteachers/model/student_class_section_subject_teacher_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Selaras dengan tabel: student_class_section_subject_teachers
type StudentClassSectionSubjectTeacher struct {
	// PK
	StudentClassSectionSubjectTeacherID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:student_class_section_subject_teacher_id" json:"student_class_section_subject_teacher_id"`

	// Tenant
	StudentClassSectionSubjectTeacherSchoolID uuid.UUID `gorm:"type:uuid;not null;column:student_class_section_subject_teacher_school_id" json:"student_class_section_subject_teacher_school_id"`

	// Anchor relations
	StudentClassSectionSubjectTeacherStudentID uuid.UUID `gorm:"type:uuid;not null;column:student_class_section_subject_teacher_student_id" json:"student_class_section_subject_teacher_student_id"`
	StudentClassSectionSubjectTeacherCSSTID    uuid.UUID `gorm:"type:uuid;not null;column:student_class_section_subject_teacher_csst_id" json:"student_class_section_subject_teacher_csst_id"`

	// Status mapping
	StudentClassSectionSubjectTeacherIsActive bool       `gorm:"not null;default:true;column:student_class_section_subject_teacher_is_active" json:"student_class_section_subject_teacher_is_active"`
	StudentClassSectionSubjectTeacherFrom     *time.Time `gorm:"type:date;column:student_class_section_subject_teacher_from" json:"student_class_section_subject_teacher_from,omitempty"`
	StudentClassSectionSubjectTeacherTo       *time.Time `gorm:"type:date;column:student_class_section_subject_teacher_to" json:"student_class_section_subject_teacher_to,omitempty"`

	// Nilai terbaru (opsional; percent di-generate di DB)
	StudentClassSectionSubjectTeacherScoreTotal    *float64 `gorm:"type:numeric(6,2);column:student_class_section_subject_teacher_score_total" json:"student_class_section_subject_teacher_score_total,omitempty"`
	StudentClassSectionSubjectTeacherScoreMaxTotal *float64 `gorm:"type:numeric(6,2);default:100;column:student_class_section_subject_teacher_score_max_total" json:"student_class_section_subject_teacher_score_max_total,omitempty"`
	StudentClassSectionSubjectTeacherScorePercent  *float64 `gorm:"type:numeric(5,2);column:student_class_section_subject_teacher_score_percent;->" json:"student_class_section_subject_teacher_score_percent,omitempty"` // read-only (generated always as)
	StudentClassSectionSubjectTeacherGradeLetter   *string  `gorm:"type:varchar(8);column:student_class_section_subject_teacher_grade_letter" json:"student_class_section_subject_teacher_grade_letter,omitempty"`
	StudentClassSectionSubjectTeacherGradePoint    *float64 `gorm:"type:numeric(3,2);column:student_class_section_subject_teacher_grade_point" json:"student_class_section_subject_teacher_grade_point,omitempty"`
	StudentClassSectionSubjectTeacherIsPassed      *bool    `gorm:"column:student_class_section_subject_teacher_is_passed" json:"student_class_section_subject_teacher_is_passed,omitempty"`

	// Snapshot users_profile & siswa (saat enrol)
	StudentClassSectionSubjectTeacherUserProfileNameSnapshot              *string `gorm:"type:varchar(80);column:student_class_section_subject_teacher_user_profile_name_snapshot" json:"student_class_section_subject_teacher_user_profile_name_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileAvatarURLSnapshot         *string `gorm:"type:varchar(255);column:student_class_section_subject_teacher_user_profile_avatar_url_snapshot" json:"student_class_section_subject_teacher_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileWhatsappURLSnapshot       *string `gorm:"type:varchar(50);column:student_class_section_subject_teacher_user_profile_whatsapp_url_snapshot" json:"student_class_section_subject_teacher_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileParentNameSnapshot        *string `gorm:"type:varchar(80);column:student_class_section_subject_teacher_user_profile_parent_name_snapshot" json:"student_class_section_subject_teacher_user_profile_parent_name_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileParentWhatsappURLSnapshot *string `gorm:"type:varchar(50);column:student_class_section_subject_teacher_user_profile_parent_whatsapp_url_snapshot" json:"student_class_section_subject_teacher_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherUserProfileGenderSnapshot            *string `gorm:"type:varchar(20);column:student_class_section_subject_teacher_user_profile_gender_snapshot" json:"student_class_section_subject_teacher_user_profile_gender_snapshot,omitempty"`
	StudentClassSectionSubjectTeacherStudentCodeSnapshot                  *string `gorm:"type:varchar(50);column:student_class_section_subject_teacher_student_code_snapshot" json:"student_class_section_subject_teacher_student_code_snapshot,omitempty"`

	// Riwayat intervensi/remedial (append-only JSONB)
	StudentClassSectionSubjectTeacherEditsHistory datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:student_class_section_subject_teacher_edits_history" json:"student_class_section_subject_teacher_edits_history"`

	// NOTES
	StudentClassSectionSubjectTeacherStudentNotes                 *string    `gorm:"type:text;column:student_class_section_subject_teacher_student_notes" json:"student_class_section_subject_teacher_student_notes,omitempty"`
	StudentClassSectionSubjectTeacherStudentNotesUpdatedAt        *time.Time `gorm:"type:timestamptz;column:student_class_section_subject_teacher_student_notes_updated_at" json:"student_class_section_subject_teacher_student_notes_updated_at,omitempty"`
	StudentClassSectionSubjectTeacherHomeroomNotes                *string    `gorm:"type:text;column:student_class_section_subject_teacher_homeroom_notes" json:"student_class_section_subject_teacher_homeroom_notes,omitempty"`
	StudentClassSectionSubjectTeacherHomeroomNotesUpdatedAt       *time.Time `gorm:"type:timestamptz;column:student_class_section_subject_teacher_homeroom_notes_updated_at" json:"student_class_section_subject_teacher_homeroom_notes_updated_at,omitempty"`
	StudentClassSectionSubjectTeacherSubjectTeacherNotes          *string    `gorm:"type:text;column:student_class_section_subject_teacher_subject_teacher_notes" json:"student_class_section_subject_teacher_subject_teacher_notes,omitempty"`
	StudentClassSectionSubjectTeacherSubjectTeacherNotesUpdatedAt *time.Time `gorm:"type:timestamptz;column:student_class_section_subject_teacher_subject_teacher_notes_updated_at" json:"student_class_section_subject_teacher_subject_teacher_notes_updated_at,omitempty"`

	// Admin & meta
	StudentClassSectionSubjectTeacherSlug *string        `gorm:"type:varchar(160);column:student_class_section_subject_teacher_slug" json:"student_class_section_subject_teacher_slug,omitempty"`
	StudentClassSectionSubjectTeacherMeta datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'::jsonb;column:student_class_section_subject_teacher_meta" json:"student_class_section_subject_teacher_meta"`

	// Audit & soft delete
	StudentClassSectionSubjectTeacherCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_class_section_subject_teacher_created_at" json:"student_class_section_subject_teacher_created_at"`
	StudentClassSectionSubjectTeacherUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:student_class_section_subject_teacher_updated_at" json:"student_class_section_subject_teacher_updated_at"`
	StudentClassSectionSubjectTeacherDeletedAt gorm.DeletedAt `gorm:"column:student_class_section_subject_teacher_deleted_at;index" json:"student_class_section_subject_teacher_deleted_at,omitempty"`
}

func (StudentClassSectionSubjectTeacher) TableName() string {
	return "student_class_section_subject_teachers"
}
