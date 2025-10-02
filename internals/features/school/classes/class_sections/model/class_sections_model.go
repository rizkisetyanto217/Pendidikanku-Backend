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

/* ============================================
   ENUM: class_section_csst_enrollment_mode
   ============================================ */

type ClassSectionCSSTEnrollmentMode string

const (
	CSSTEnrollSelfSelect ClassSectionCSSTEnrollmentMode = "self_select"
	CSSTEnrollAssigned   ClassSectionCSSTEnrollmentMode = "assigned"
	CSSTEnrollHybrid     ClassSectionCSSTEnrollmentMode = "hybrid"
)

var validCSSTEnroll = map[ClassSectionCSSTEnrollmentMode]struct{}{
	CSSTEnrollSelfSelect: {},
	CSSTEnrollAssigned:   {},
	CSSTEnrollHybrid:     {},
}

func (e ClassSectionCSSTEnrollmentMode) String() string { return string(e) }
func (e ClassSectionCSSTEnrollmentMode) Valid() bool {
	_, ok := validCSSTEnroll[e]
	return ok
}
func (e ClassSectionCSSTEnrollmentMode) Value() (driver.Value, error) {
	if e == "" {
		return nil, nil
	}
	if !e.Valid() {
		return nil, fmt.Errorf("invalid class_section_csst_enrollment_mode: %q", e)
	}
	return string(e), nil
}
func (e *ClassSectionCSSTEnrollmentMode) Scan(v any) error {
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
		return fmt.Errorf("unsupported Scan for CSSTEnrollmentMode: %T", v)
	}
	s = strings.ToLower(strings.TrimSpace(s))
	ev := ClassSectionCSSTEnrollmentMode(s)
	if ev != "" && !ev.Valid() {
		return fmt.Errorf("invalid value from DB: %q", s)
	}
	*e = ev
	return nil
}

/* ============================================
   MODEL: class_sections
   ============================================ */

type ClassSectionModel struct {
	// ============== PK & Tenant ==============
	ClassSectionID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_section_id;uniqueIndex:uq_class_section_id_masjid" json:"class_section_id"`
	ClassSectionMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_section_masjid_id;uniqueIndex:uq_class_section_id_masjid" json:"class_section_masjid_id"`

	// ============== Relasi inti ==============
	ClassSectionClassID            uuid.UUID  `gorm:"type:uuid;not null;column:class_section_class_id" json:"class_section_class_id"`
	ClassSectionTeacherID          *uuid.UUID `gorm:"type:uuid;column:class_section_teacher_id" json:"class_section_teacher_id,omitempty"`
	ClassSectionAssistantTeacherID *uuid.UUID `gorm:"type:uuid;column:class_section_assistant_teacher_id" json:"class_section_assistant_teacher_id,omitempty"`
	ClassSectionClassRoomID        *uuid.UUID `gorm:"type:uuid;column:class_section_class_room_id" json:"class_section_class_room_id,omitempty"`
	ClassSectionLeaderStudentID    *uuid.UUID `gorm:"type:uuid;column:class_section_leader_student_id" json:"class_section_leader_student_id,omitempty"`

	// ============== Identitas ==============
	ClassSectionSlug string  `gorm:"type:varchar(160);not null;column:class_section_slug" json:"class_section_slug"`
	ClassSectionName string  `gorm:"type:varchar(100);not null;column:class_section_name" json:"class_section_name"`
	ClassSectionCode *string `gorm:"type:varchar(50);column:class_section_code" json:"class_section_code,omitempty"`

	// ============== Jadwal sederhana ==============
	ClassSectionSchedule *string `gorm:"type:text;column:class_section_schedule" json:"class_section_schedule,omitempty"`

	// ============== Kapasitas & counter ==============
	ClassSectionCapacity      *int `gorm:"column:class_section_capacity" json:"class_section_capacity,omitempty"`
	ClassSectionTotalStudents int  `gorm:"not null;default:0;column:class_section_total_students" json:"class_section_total_students"`

	// ============== Meeting / Group ==============
	ClassSectionGroupURL *string `gorm:"type:text;column:class_section_group_url" json:"class_section_group_url,omitempty"`

	// ============== Image (2-slot + retensi) ==============
	ClassSectionImageURL                *string    `gorm:"type:text;column:class_section_image_url" json:"class_section_image_url,omitempty"`
	ClassSectionImageObjectKey          *string    `gorm:"type:text;column:class_section_image_object_key" json:"class_section_image_object_key,omitempty"`
	ClassSectionImageURLOld             *string    `gorm:"type:text;column:class_section_image_url_old" json:"class_section_image_url_old,omitempty"`
	ClassSectionImageObjectKeyOld       *string    `gorm:"type:text;column:class_section_image_object_key_old" json:"class_section_image_object_key_old,omitempty"`
	ClassSectionImageDeletePendingUntil *time.Time `gorm:"column:class_section_image_delete_pending_until" json:"class_section_image_delete_pending_until,omitempty"`

	// ============== Join codes (hash) ==============
	ClassSectionTeacherCodeHash  []byte     `gorm:"type:bytea;column:class_section_teacher_code_hash" json:"-"`
	ClassSectionTeacherCodeSetAt *time.Time `gorm:"column:class_section_teacher_code_set_at" json:"class_section_teacher_code_set_at,omitempty"`
	ClassSectionStudentCodeHash  []byte     `gorm:"type:bytea;column:class_section_student_code_hash" json:"-"`
	ClassSectionStudentCodeSetAt *time.Time `gorm:"column:class_section_student_code_set_at" json:"class_section_student_code_set_at,omitempty"`

	// ============== SNAPSHOTS (JSONB) ==============
	// Class
	ClassSectionClassSnapshot datatypes.JSON `gorm:"type:jsonb;column:class_section_class_snapshot" json:"class_section_class_snapshot,omitempty"`
	ClassSectionClassSlugSnap *string        `gorm:"->;type:text;column:class_section_class_slug_snap" json:"class_section_class_slug_snap,omitempty"`

	// Parent
	ClassSectionParentSnapshot  datatypes.JSON `gorm:"type:jsonb;column:class_section_parent_snapshot" json:"class_section_parent_snapshot,omitempty"`
	ClassSectionParentNameSnap  *string        `gorm:"->;type:text;column:class_section_parent_name_snap" json:"class_section_parent_name_snap,omitempty"`
	ClassSectionParentCodeSnap  *string        `gorm:"->;type:text;column:class_section_parent_code_snap" json:"class_section_parent_code_snap,omitempty"`
	ClassSectionParentSlugSnap  *string        `gorm:"->;type:text;column:class_section_parent_slug_snap" json:"class_section_parent_slug_snap,omitempty"`
	ClassSectionParentLevelSnap *string        `gorm:"->;type:text;column:class_section_parent_level_snap" json:"class_section_parent_level_snap,omitempty"`

	// People
	ClassSectionTeacherSnapshot          datatypes.JSON `gorm:"type:jsonb;column:class_section_teacher_snapshot" json:"class_section_teacher_snapshot,omitempty"`
	ClassSectionAssistantTeacherSnapshot datatypes.JSON `gorm:"type:jsonb;column:class_section_assistant_teacher_snapshot" json:"class_section_assistant_teacher_snapshot,omitempty"`
	ClassSectionLeaderStudentSnapshot    datatypes.JSON `gorm:"type:jsonb;column:class_section_leader_student_snapshot" json:"class_section_leader_student_snapshot,omitempty"`

	ClassSectionTeacherNameSnap          *string `gorm:"->;type:text;column:class_section_teacher_name_snap" json:"class_section_teacher_name_snap,omitempty"`
	ClassSectionAssistantTeacherNameSnap *string `gorm:"->;type:text;column:class_section_assistant_teacher_name_snap" json:"class_section_assistant_teacher_name_snap,omitempty"`
	ClassSectionLeaderStudentNameSnap    *string `gorm:"->;type:text;column:class_section_leader_student_name_snap" json:"class_section_leader_student_name_snap,omitempty"`

	// Room
	ClassSectionRoomSnapshot     datatypes.JSON `gorm:"type:jsonb;column:class_section_room_snapshot" json:"class_section_room_snapshot,omitempty"`
	ClassSectionRoomNameSnap     *string        `gorm:"->;type:text;column:class_section_room_name_snap" json:"class_section_room_name_snap,omitempty"`
	ClassSectionRoomSlugSnap     *string        `gorm:"->;type:text;column:class_section_room_slug_snap" json:"class_section_room_slug_snap,omitempty"`
	ClassSectionRoomLocationSnap *string        `gorm:"->;type:text;column:class_section_room_location_snap" json:"class_section_room_location_snap,omitempty"`

	// Term
	ClassSectionTermID        *uuid.UUID     `gorm:"type:uuid;column:class_section_term_id" json:"class_section_term_id,omitempty"`
	ClassSectionTermSnapshot  datatypes.JSON `gorm:"type:jsonb;column:class_section_term_snapshot" json:"class_section_term_snapshot,omitempty"`
	ClassSectionTermNameSnap  *string        `gorm:"->;type:text;column:class_section_term_name_snap" json:"class_section_term_name_snap,omitempty"`
	ClassSectionTermSlugSnap  *string        `gorm:"->;type:text;column:class_section_term_slug_snap" json:"class_section_term_slug_snap,omitempty"`
	ClassSectionTermYearLabel *string        `gorm:"->;type:text;column:class_section_term_year_label_snap" json:"class_section_term_year_label_snap,omitempty"`

	// Housekeeping snapshot
	ClassSectionSnapshotUpdatedAt *time.Time `gorm:"column:class_section_snapshot_updated_at" json:"class_section_snapshot_updated_at,omitempty"`

	// ============== CSST: cache + pengaturan ==============
	ClassSectionsCSST            datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:class_sections_csst" json:"class_sections_csst"`
	ClassSectionsCSSTCount       int            `gorm:"->;column:class_sections_csst_count" json:"class_sections_csst_count"`
	ClassSectionsCSSTActiveCount int            `gorm:"->;column:class_sections_csst_active_count" json:"class_sections_csst_active_count"`

	ClassSectionCSSTEnrollmentMode             ClassSectionCSSTEnrollmentMode `gorm:"type:class_section_csst_enrollment_mode;not null;default:'self_select';column:class_section_csst_enrollment_mode" json:"class_section_csst_enrollment_mode"`
	ClassSectionCSSTSelfSelectRequiresApproval bool                           `gorm:"not null;default:false;column:class_section_csst_self_select_requires_approval" json:"class_section_csst_self_select_requires_approval"`
	ClassSectionCSSTMaxSubjectsPerStudent      *int                           `gorm:"column:class_section_csst_max_subjects_per_student" json:"class_section_csst_max_subjects_per_student,omitempty"`
	ClassSectionCSSTSwitchDeadline             *time.Time                     `gorm:"column:class_section_csst_switch_deadline" json:"class_section_csst_switch_deadline,omitempty"`

	ClassSectionFeatures datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'::jsonb;column:class_section_features" json:"class_section_features"`

	// ============== Status & audit ==============
	ClassSectionIsActive  bool           `gorm:"not null;default:true;column:class_section_is_active" json:"class_section_is_active"`
	ClassSectionCreatedAt time.Time      `gorm:"not null;autoCreateTime;column:class_section_created_at" json:"class_section_created_at"`
	ClassSectionUpdatedAt time.Time      `gorm:"not null;autoUpdateTime;column:class_section_updated_at" json:"class_section_updated_at"`
	ClassSectionDeletedAt gorm.DeletedAt `gorm:"column:class_section_deleted_at;index" json:"class_section_deleted_at,omitempty"`
}

func (ClassSectionModel) TableName() string { return "class_sections" }

/* ============================================
   Hooks ringan: jaga JSONB tidak NULL
   ============================================ */

func (m *ClassSectionModel) BeforeCreate(tx *gorm.DB) error {
	if len(m.ClassSectionsCSST) == 0 {
		m.ClassSectionsCSST = datatypes.JSON([]byte("[]"))
	}
	if len(m.ClassSectionFeatures) == 0 {
		m.ClassSectionFeatures = datatypes.JSON([]byte("{}"))
	}
	// enum safety (optional)
	if m.ClassSectionCSSTEnrollmentMode != "" && !m.ClassSectionCSSTEnrollmentMode.Valid() {
		return errors.New("invalid class_section_csst_enrollment_mode")
	}
	return nil
}

func (m *ClassSectionModel) BeforeSave(tx *gorm.DB) error {
	if len(m.ClassSectionsCSST) == 0 {
		m.ClassSectionsCSST = datatypes.JSON([]byte("[]"))
	}
	if len(m.ClassSectionFeatures) == 0 {
		m.ClassSectionFeatures = datatypes.JSON([]byte("{}"))
	}
	if m.ClassSectionCSSTEnrollmentMode != "" && !m.ClassSectionCSSTEnrollmentMode.Valid() {
		return errors.New("invalid class_section_csst_enrollment_mode")
	}
	return nil
}
