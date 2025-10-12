// file: internals/features/school/sectionsubjectteachers/model/class_section_subject_teacher_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Enum delivery mode (map ke type di DB: class_delivery_mode_enum)
type ClassDeliveryMode string

const (
	DeliveryModeOffline ClassDeliveryMode = "offline"
	DeliveryModeOnline  ClassDeliveryMode = "online"
	DeliveryModeHybrid  ClassDeliveryMode = "hybrid"
)

type ClassSectionSubjectTeacherModel struct {
	// PK
	ClassSectionSubjectTeacherID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_section_subject_teacher_id" json:"class_section_subject_teacher_id"`

	// Tenant
	ClassSectionSubjectTeacherMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_masjid_id" json:"class_section_subject_teacher_masjid_id"`

	// Relations (IDs)
	ClassSectionSubjectTeacherSectionID      uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_section_id" json:"class_section_subject_teacher_section_id"`
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_class_subject_id" json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherTeacherID      uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_teacher_id" json:"class_section_subject_teacher_teacher_id"`

	// Identitas / fasilitas
	ClassSectionSubjectTeacherSlug        *string   `gorm:"type:varchar(160);column:class_section_subject_teacher_slug" json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string   `gorm:"type:text;column:class_section_subject_teacher_description" json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherRoomID      *uuid.UUID`gorm:"type:uuid;column:class_section_subject_teacher_room_id" json:"class_section_subject_teacher_room_id,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string   `gorm:"type:text;column:class_section_subject_teacher_group_url" json:"class_section_subject_teacher_group_url,omitempty"`

	// Agregat & kapasitas
	ClassSectionSubjectTeacherTotalAttendance int              `gorm:"not null;default:0;column:class_section_subject_teacher_total_attendance" json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherCapacity        *int            `gorm:"column:class_section_subject_teacher_capacity" json:"class_section_subject_teacher_capacity,omitempty"` // NULL/<=0 = unlimited
	ClassSectionSubjectTeacherEnrolledCount   int              `gorm:"not null;default:0;column:class_section_subject_teacher_enrolled_count" json:"class_section_subject_teacher_enrolled_count"`
	// NOTE: nama kolom mengikuti SQL terbaru (ada 'sections' jamak)
	ClassSectionsSubjectTeacherDeliveryMode  ClassDeliveryMode `gorm:"type:class_delivery_mode_enum;not null;default:'offline';column:class_sections_subject_teacher_delivery_mode" json:"class_sections_subject_teacher_delivery_mode"`

	// Room snapshot + kolom turunan (generated)
	ClassSectionSubjectTeacherRoomSnapshot       *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_room_snapshot" json:"class_section_subject_teacher_room_snapshot,omitempty"`
	ClassSectionSubjectTeacherRoomNameSnap       *string         `gorm:"->;column:class_section_subject_teacher_room_name_snap" json:"class_section_subject_teacher_room_name_snap,omitempty"`
	ClassSectionSubjectTeacherRoomSlugSnap       *string         `gorm:"->;column:class_section_subject_teacher_room_slug_snap" json:"class_section_subject_teacher_room_slug_snap,omitempty"`
	ClassSectionSubjectTeacherRoomLocationSnap   *string         `gorm:"->;column:class_section_subject_teacher_room_location_snap" json:"class_section_subject_teacher_room_location_snap,omitempty"`

	// People snapshots + kolom turunan (generated)
	ClassSectionSubjectTeacherTeacherSnapshot          *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_teacher_snapshot" json:"class_section_subject_teacher_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantTeacherSnapshot *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_assistant_teacher_snapshot" json:"class_section_subject_teacher_assistant_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherTeacherNameSnap          *string         `gorm:"->;column:class_section_subject_teacher_teacher_name_snap" json:"class_section_subject_teacher_teacher_name_snap,omitempty"`
	ClassSectionSubjectTeacherAssistantTeacherNameSnap *string         `gorm:"->;column:class_section_subject_teacher_assistant_teacher_name_snap" json:"class_section_subject_teacher_assistant_teacher_name_snap,omitempty"`

	// Status & audit
	ClassSectionSubjectTeacherIsActive  bool           `gorm:"not null;default:true;column:class_section_subject_teacher_is_active" json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_created_at" json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_updated_at" json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt gorm.DeletedAt `gorm:"column:class_section_subject_teacher_deleted_at;index" json:"class_section_subject_teacher_deleted_at,omitempty"`
}

func (ClassSectionSubjectTeacherModel) TableName() string { return "class_section_subject_teachers" }
