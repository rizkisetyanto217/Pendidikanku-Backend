// file: internals/features/school/classes/class_sections/model/class_section_model.go
package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ===========================================================
   ENUM (DB): class_section_subject_teachers_enrollment_mode
   =========================================================== */

type ClassSectionSubjectTeachersEnrollmentMode string

const (
	EnrollSelfSelect ClassSectionSubjectTeachersEnrollmentMode = "self_select"
	EnrollAssigned   ClassSectionSubjectTeachersEnrollmentMode = "assigned"
	EnrollHybrid     ClassSectionSubjectTeachersEnrollmentMode = "hybrid"
)

var validEnroll = map[ClassSectionSubjectTeachersEnrollmentMode]struct{}{
	EnrollSelfSelect: {},
	EnrollAssigned:   {},
	EnrollHybrid:     {},
}

func (e ClassSectionSubjectTeachersEnrollmentMode) String() string { return string(e) }
func (e ClassSectionSubjectTeachersEnrollmentMode) Valid() bool {
	_, ok := validEnroll[e]
	return ok
}
func (e ClassSectionSubjectTeachersEnrollmentMode) Value() (driver.Value, error) {
	if e == "" {
		return nil, nil
	}
	if !e.Valid() {
		return nil, fmt.Errorf("invalid class_section_subject_teachers_enrollment_mode: %q", e)
	}
	return string(e), nil
}
func (e *ClassSectionSubjectTeachersEnrollmentMode) Scan(v any) error {
	if v == nil {
		*e = ""
		return nil
	}
	var s string
	switch t := v.(type) {
	case []byte:
		s = string(t)
	case string:
		s = t
	default:
		return fmt.Errorf("unsupported Scan for CSST enrollment mode: %T", v)
	}
	s = strings.ToLower(strings.TrimSpace(s))
	ev := ClassSectionSubjectTeachersEnrollmentMode(s)
	if ev != "" && !ev.Valid() {
		return fmt.Errorf("invalid value from DB: %q", s)
	}
	*e = ev
	return nil
}

/* ===========================
   MODEL: class_sections
   =========================== */

type ClassSectionModel struct {
	/* ===== PK & Tenant ===== */
	ClassSectionID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_section_id;uniqueIndex:uq_class_section_id_school" json:"class_section_id"`
	ClassSectionSchoolID uuid.UUID `gorm:"type:uuid;not null;column:class_section_school_id;uniqueIndex:uq_class_section_id_school" json:"class_section_school_id"`

	/* ===== Identitas ===== */
	ClassSectionSlug string  `gorm:"type:varchar(160);not null;column:class_section_slug" json:"class_section_slug"`
	ClassSectionName string  `gorm:"type:varchar(100);not null;column:class_section_name" json:"class_section_name"`
	ClassSectionCode *string `gorm:"type:varchar(50);column:class_section_code" json:"class_section_code,omitempty"`

	/* ===== Jadwal sederhana ===== */
	ClassSectionSchedule *string `gorm:"type:text;column:class_section_schedule" json:"class_section_schedule,omitempty"`

	/* ===== Kuota (utama, sama pola dgn classes) ===== */
	// - ClassSectionQuotaTotal → kapasitas maksimal (limit)
	// - ClassSectionQuotaTaken → sudah terdaftar (count)
	ClassSectionQuotaTotal *int `gorm:"column:class_section_quota_total"                               json:"class_section_quota_total,omitempty"`
	ClassSectionQuotaTaken int  `gorm:"not null;default:0;column:class_section_quota_taken"            json:"class_section_quota_taken"`

	/* ===== Stats (ALL & ACTIVE) ===== */
	ClassSectionTotalStudentsActive       int            `gorm:"not null;default:0;column:class_section_total_students_active" json:"class_section_total_students_active"`
	ClassSectionTotalStudentsMale         int            `gorm:"not null;default:0;column:class_section_total_students_male" json:"class_section_total_students_male"`
	ClassSectionTotalStudentsFemale       int            `gorm:"not null;default:0;column:class_section_total_students_female" json:"class_section_total_students_female"`
	ClassSectionTotalStudentsMaleActive   int            `gorm:"not null;default:0;column:class_section_total_students_male_active" json:"class_section_total_students_male_active"`
	ClassSectionTotalStudentsFemaleActive int            `gorm:"not null;default:0;column:class_section_total_students_female_active" json:"class_section_total_students_female_active"`
	ClassSectionStats                     datatypes.JSON `gorm:"type:jsonb;column:class_section_stats" json:"class_section_stats,omitempty"`

	/* ===== Meeting / Group ===== */
	ClassSectionGroupURL *string `gorm:"type:text;column:class_section_group_url" json:"class_section_group_url,omitempty"`

	/* ===== Image (2-slot + retensi) ===== */
	ClassSectionImageURL                *string    `gorm:"type:text;column:class_section_image_url" json:"class_section_image_url,omitempty"`
	ClassSectionImageObjectKey          *string    `gorm:"type:text;column:class_section_image_object_key" json:"class_section_image_object_key,omitempty"`
	ClassSectionImageURLOld             *string    `gorm:"type:text;column:class_section_image_url_old" json:"class_section_image_url_old,omitempty"`
	ClassSectionImageObjectKeyOld       *string    `gorm:"type:text;column:class_section_image_object_key_old" json:"class_section_image_object_key_old,omitempty"`
	ClassSectionImageDeletePendingUntil *time.Time `gorm:"column:class_section_image_delete_pending_until" json:"class_section_image_delete_pending_until,omitempty"`

	/* ===== Join code (hash) ===== */
	ClassSectionTeacherCodeHash  []byte     `gorm:"type:bytea;column:class_section_teacher_code_hash" json:"-"`
	ClassSectionTeacherCodeSetAt *time.Time `gorm:"column:class_section_teacher_code_set_at" json:"class_section_teacher_code_set_at,omitempty"`
	ClassSectionStudentCodeHash  []byte     `gorm:"type:bytea;column:class_section_student_code_hash" json:"-"`
	ClassSectionStudentCodeSetAt *time.Time `gorm:"column:class_section_student_code_set_at" json:"class_section_student_code_set_at,omitempty"`

	/* =====================================================
	   Relasi & SNAPSHOTS (persis seperti DDL)
	   ===================================================== */

	// Class (live id + labels snapshot)
	ClassSectionClassID        *uuid.UUID `gorm:"type:uuid;column:class_section_class_id" json:"class_section_class_id,omitempty"`
	ClassSectionClassNameCache *string    `gorm:"type:varchar(160);column:class_section_class_name_cache" json:"class_section_class_name_cache,omitempty"`
	ClassSectionClassSlugCache *string    `gorm:"type:varchar(160);column:class_section_class_slug_cache" json:"class_section_class_slug_cache,omitempty"`

	// Parent (id + nama + slug + level smallint)
	ClassSectionClassParentID            *uuid.UUID `gorm:"type:uuid;column:class_section_class_parent_id" json:"class_section_class_parent_id,omitempty"`
	ClassSectionClassParentNameCache     *string    `gorm:"type:varchar(160);column:class_section_class_parent_name_cache" json:"class_section_class_parent_name_cache,omitempty"`
	ClassSectionClassParentSlugCache     *string    `gorm:"type:varchar(160);column:class_section_class_parent_slug_cache" json:"class_section_class_parent_slug_cache,omitempty"`
	ClassSectionClassParentLevelCache    *int16     `gorm:"type:smallint;column:class_section_class_parent_level_cache" json:"class_section_class_parent_level_cache,omitempty"`

	// People (teacher/assistant/leader) — ID + SLUG + JSONB snapshot
	ClassSectionSchoolTeacherID           *uuid.UUID     `gorm:"type:uuid;column:class_section_school_teacher_id" json:"class_section_school_teacher_id,omitempty"`
	ClassSectionSchoolTeacherSlugCache    *string        `gorm:"type:varchar(100);column:class_section_school_teacher_slug_cache" json:"class_section_school_teacher_slug_cache,omitempty"`
	ClassSectionSchoolTeacherCache        datatypes.JSON `gorm:"type:jsonb;column:class_section_school_teacher_cache" json:"class_section_school_teacher_cache,omitempty"`

	// Assistant teacher
	ClassSectionAssistantSchoolTeacherID        *uuid.UUID     `gorm:"type:uuid;column:class_section_assistant_school_teacher_id" json:"class_section_assistant_school_teacher_id,omitempty"`
	ClassSectionAssistantSchoolTeacherSlugCache *string        `gorm:"type:varchar(100);column:class_section_assistant_school_teacher_slug_cache" json:"class_section_assistant_school_teacher_slug_cache,omitempty"`
	ClassSectionAssistantSchoolTeacherCache     datatypes.JSON `gorm:"type:jsonb;column:class_section_assistant_school_teacher_cache" json:"class_section_assistant_school_teacher_cache,omitempty"`

	// Leader student
	ClassSectionLeaderSchoolStudentID        *uuid.UUID     `gorm:"type:uuid;column:class_section_leader_school_student_id" json:"class_section_leader_school_student_id,omitempty"`
	ClassSectionLeaderSchoolStudentSlugCache *string        `gorm:"type:varchar(100);column:class_section_leader_school_student_slug_cache" json:"class_section_leader_school_student_slug_cache,omitempty"`
	ClassSectionLeaderSchoolStudentCache     datatypes.JSON `gorm:"type:jsonb;column:class_section_leader_school_student_cache" json:"class_section_leader_school_student_cache,omitempty"`

	// Room
	ClassSectionClassRoomID              *uuid.UUID     `gorm:"type:uuid;column:class_section_class_room_id" json:"class_section_class_room_id,omitempty"`
	ClassSectionClassRoomSlugCache       *string        `gorm:"type:varchar(100);column:class_section_class_room_slug_cache" json:"class_section_class_room_slug_cache,omitempty"`
	ClassSectionClassRoomNameCache       *string        `gorm:"type:varchar(160);column:class_section_class_room_name_cache" json:"class_section_class_room_name_cache,omitempty"`
	ClassSectionClassRoomLocationCache   *string        `gorm:"type:text;column:class_section_class_room_location_cache" json:"class_section_class_room_location_cache,omitempty"`
	ClassSectionClassRoomCache           datatypes.JSON `gorm:"type:jsonb;column:class_section_class_room_cache" json:"class_section_class_room_cache,omitempty"`

	// TERM
	ClassSectionAcademicTermID                   *uuid.UUID `gorm:"type:uuid;column:class_section_academic_term_id" json:"class_section_academic_term_id,omitempty"`
	ClassSectionAcademicTermNameCache           *string    `gorm:"type:text;column:class_section_academic_term_name_cache" json:"class_section_academic_term_name_cache,omitempty"`
	ClassSectionAcademicTermSlugCache           *string    `gorm:"type:text;column:class_section_academic_term_slug_cache" json:"class_section_academic_term_slug_cache,omitempty"`
	ClassSectionAcademicTermAcademicYearCache   *string    `gorm:"type:text;column:class_section_academic_term_academic_year_cache" json:"class_section_academic_term_academic_year_cache,omitempty"`
	ClassSectionAcademicTermAngkatanCache       *int       `gorm:"column:class_section_academic_term_angkatan_cache" json:"class_section_academic_term_angkatan_cache,omitempty"`

	/* ===== SUBJECT-TEACHERS SETTINGS ===== */
	ClassSectionSubjectTeachersEnrollmentMode             ClassSectionSubjectTeachersEnrollmentMode `gorm:"type:class_section_subject_teachers_enrollment_mode;not null;default:'self_select';column:class_section_subject_teachers_enrollment_mode" json:"class_section_subject_teachers_enrollment_mode"`
	ClassSectionSubjectTeachersSelfSelectRequiresApproval bool                                      `gorm:"not null;default:false;column:class_section_subject_teachers_self_select_requires_approval" json:"class_section_subject_teachers_self_select_requires_approval"`
	ClassSectionSubjectTeachersMaxSubjectsPerStudent      *int                                      `gorm:"column:class_section_subject_teachers_max_subjects_per_student" json:"class_section_subject_teachers_max_subjects_per_student,omitempty"`

	/* ===== CSST TOTALS ===== */
	ClassSectionTotalClassClassSectionSubjectTeachers       int `gorm:"not null;default:0;column:class_section_total_class_class_section_subject_teachers" json:"class_section_total_class_class_section_subject_teachers"`
	ClassSectionTotalClassClassSectionSubjectTeachersActive int `gorm:"not null;default:0;column:class_section_total_class_class_section_subject_teachers_active" json:"class_section_total_class_class_section_subject_teachers_active"`

	/* ===== Status & audit ===== */
	ClassSectionIsActive  bool           `gorm:"not null;default:true;column:class_section_is_active" json:"class_section_is_active"`
	ClassSectionCreatedAt time.Time      `gorm:"not null;autoCreateTime;column:class_section_created_at" json:"class_section_created_at"`
	ClassSectionUpdatedAt time.Time      `gorm:"not null;autoUpdateTime;column:class_section_updated_at" json:"class_section_updated_at"`
	ClassSectionDeletedAt gorm.DeletedAt `gorm:"column:class_section_deleted_at;index" json:"class_section_deleted_at,omitempty"`
}

func (ClassSectionModel) TableName() string { return "class_sections" }

/* ================================
   Hooks ringan (validasi enum)
   ================================ */

func (m *ClassSectionModel) BeforeCreate(tx *gorm.DB) error {
	if m.ClassSectionSubjectTeachersEnrollmentMode != "" && !m.ClassSectionSubjectTeachersEnrollmentMode.Valid() {
		return errors.New("invalid class_section_subject_teachers_enrollment_mode")
	}
	return nil
}

func (m *ClassSectionModel) BeforeSave(tx *gorm.DB) error {
	if m.ClassSectionSubjectTeachersEnrollmentMode != "" && !m.ClassSectionSubjectTeachersEnrollmentMode.Valid() {
		return errors.New("invalid class_section_subject_teachers_enrollment_mode")
	}
	return nil
}
