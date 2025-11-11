// file: internals/features/school/sectionsubjectteachers/model/class_section_subject_teacher_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ===== ENUM (ikuti DB: class_delivery_mode_enum) ===== */
type ClassDeliveryMode string

const (
	DeliveryModeOffline ClassDeliveryMode = "offline"
	DeliveryModeOnline  ClassDeliveryMode = "online"
	DeliveryModeHybrid  ClassDeliveryMode = "hybrid"
)

/* =========================================================
   MODEL: class_section_subject_teachers (ikut SQL persis)
========================================================= */
type ClassSectionSubjectTeacherModel struct {
	/* ===== PK & Tenant ===== */
	ClassSectionSubjectTeacherID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_section_subject_teacher_id" json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherSchoolID uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_school_id" json:"class_section_subject_teacher_school_id"`

	/* ===== Identitas & Fasilitas ===== */
	ClassSectionSubjectTeacherSlug        *string `gorm:"type:varchar(160);column:class_section_subject_teacher_slug" json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string `gorm:"type:text;column:class_section_subject_teacher_description" json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string `gorm:"type:text;column:class_section_subject_teacher_group_url" json:"class_section_subject_teacher_group_url,omitempty"`

	/* ===== Agregat & kapasitas ===== */
	ClassSectionSubjectTeacherTotalAttendance int  `gorm:"not null;default:0;column:class_section_subject_teacher_total_attendance" json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherCapacity        *int `gorm:"column:class_section_subject_teacher_capacity" json:"class_section_subject_teacher_capacity,omitempty"`
	ClassSectionSubjectTeacherEnrolledCount   int  `gorm:"not null;default:0;column:class_section_subject_teacher_enrolled_count" json:"class_section_subject_teacher_enrolled_count"`

	/* ===== Delivery mode (enum) ===== */
	ClassSectionSubjectTeacherDeliveryMode ClassDeliveryMode `gorm:"type:class_delivery_mode_enum;not null;default:'offline';column:class_section_subject_teacher_delivery_mode" json:"class_section_subject_teacher_delivery_mode"`

	/* ===== SECTION snapshots (tanpa JSONB) ===== */
	ClassSectionSubjectTeacherClassSectionID            uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_class_section_id" json:"class_section_subject_teacher_class_section_id"`
	ClassSectionSubjectTeacherClassSectionSlugSnapshot  *string   `gorm:"type:varchar(160);column:class_section_subject_teacher_class_section_slug_snapshot" json:"class_section_subject_teacher_class_section_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSectionNameSnapshot  *string   `gorm:"type:varchar(160);column:class_section_subject_teacher_class_section_name_snapshot" json:"class_section_subject_teacher_class_section_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSectionCodeSnapshot  *string   `gorm:"type:varchar(50);column:class_section_subject_teacher_class_section_code_snapshot" json:"class_section_subject_teacher_class_section_code_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSectionURLSnapshot   *string   `gorm:"type:text;column:class_section_subject_teacher_class_section_url_snapshot" json:"class_section_subject_teacher_class_section_url_snapshot,omitempty"`

	/* ===== ROOM snapshot ===== */
	ClassSectionSubjectTeacherClassRoomID                *uuid.UUID     `gorm:"type:uuid;column:class_section_subject_teacher_class_room_id" json:"class_section_subject_teacher_class_room_id,omitempty"`
	ClassSectionSubjectTeacherClassRoomSlugSnapshot      *string        `gorm:"type:varchar(160);column:class_section_subject_teacher_class_room_slug_snapshot" json:"class_section_subject_teacher_class_room_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassRoomSnapshot          *datatypes.JSON`gorm:"type:jsonb;column:class_section_subject_teacher_class_room_snapshot" json:"class_section_subject_teacher_class_room_snapshot,omitempty"`
	// generated (read-only)
	ClassSectionSubjectTeacherClassRoomNameSnapshot     *string `gorm:"->;column:class_section_subject_teacher_class_room_name_snapshot" json:"class_section_subject_teacher_class_room_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassRoomSlugSnapshotGen  *string `gorm:"->;column:class_section_subject_teacher_class_room_slug_snapshot_gen" json:"class_section_subject_teacher_class_room_slug_snapshot_gen,omitempty"`
	ClassSectionSubjectTeacherClassRoomLocationSnapshot *string `gorm:"->;column:class_section_subject_teacher_class_room_location_snapshot" json:"class_section_subject_teacher_class_room_location_snapshot,omitempty"`

	/* ===== PEOPLE snapshots (teacher & assistant) ===== */
	ClassSectionSubjectTeacherSchoolTeacherID                 uuid.UUID      `gorm:"type:uuid;not null;column:class_section_subject_teacher_school_teacher_id" json:"class_section_subject_teacher_school_teacher_id"`
	ClassSectionSubjectTeacherSchoolTeacherSlugSnapshot       *string        `gorm:"type:varchar(160);column:class_section_subject_teacher_school_teacher_slug_snapshot" json:"class_section_subject_teacher_school_teacher_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherSchoolTeacherSnapshot           *datatypes.JSON`gorm:"type:jsonb;column:class_section_subject_teacher_school_teacher_snapshot" json:"class_section_subject_teacher_school_teacher_snapshot,omitempty"`

	ClassSectionSubjectTeacherAssistantSchoolTeacherID            *uuid.UUID     `gorm:"type:uuid;column:class_section_subject_teacher_assistant_school_teacher_id" json:"class_section_subject_teacher_assistant_school_teacher_id,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherSlugSnapshot  *string        `gorm:"type:varchar(160);column:class_section_subject_teacher_assistant_school_teacher_slug_snapshot" json:"class_section_subject_teacher_assistant_school_teacher_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherSnapshot      *datatypes.JSON`gorm:"type:jsonb;column:class_section_subject_teacher_assistant_school_teacher_snapshot" json:"class_section_subject_teacher_assistant_school_teacher_snapshot,omitempty"`

	// generated names (read-only)
	ClassSectionSubjectTeacherSchoolTeacherNameSnapshot          *string `gorm:"->;column:class_section_subject_teacher_school_teacher_name_snapshot" json:"class_section_subject_teacher_school_teacher_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherNameSnapshot *string `gorm:"->;column:class_section_subject_teacher_assistant_school_teacher_name_snapshot" json:"class_section_subject_teacher_assistant_school_teacher_name_snapshot,omitempty"`

	/* ===== CLASS_SUBJECT_BOOK snapshot ===== */
	ClassSectionSubjectTeacherClassSubjectBookID             uuid.UUID      `gorm:"type:uuid;not null;column:class_section_subject_teacher_class_subject_book_id" json:"class_section_subject_teacher_class_subject_book_id"`
	ClassSectionSubjectTeacherClassSubjectBookSlugSnapshot   *string        `gorm:"type:varchar(160);column:class_section_subject_teacher_class_subject_book_slug_snapshot" json:"class_section_subject_teacher_class_subject_book_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSubjectBookSnapshot       *datatypes.JSON`gorm:"type:jsonb;column:class_section_subject_teacher_class_subject_book_snapshot" json:"class_section_subject_teacher_class_subject_book_snapshot,omitempty"`

	/* ===== Derived from CSB snapshot — BOOK* ===== */
	ClassSectionSubjectTeacherBookTitleSnapshot    *string `gorm:"->;column:class_section_subject_teacher_book_title_snapshot" json:"class_section_subject_teacher_book_title_snapshot,omitempty"`
	ClassSectionSubjectTeacherBookAuthorSnapshot   *string `gorm:"->;column:class_section_subject_teacher_book_author_snapshot" json:"class_section_subject_teacher_book_author_snapshot,omitempty"`
	ClassSectionSubjectTeacherBookSlugSnapshot     *string `gorm:"->;column:class_section_subject_teacher_book_slug_snapshot" json:"class_section_subject_teacher_book_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherBookImageURLSnapshot *string `gorm:"->;column:class_section_subject_teacher_book_image_url_snapshot" json:"class_section_subject_teacher_book_image_url_snapshot,omitempty"`

	/* ===== Derived from CSB snapshot — SUBJECT* ===== */
	ClassSectionSubjectTeacherSubjectIDSnapshot   *uuid.UUID `gorm:"column:class_section_subject_teacher_subject_id_snapshot" json:"class_section_subject_teacher_subject_id_snapshot,omitempty"`
	ClassSectionSubjectTeacherSubjectNameSnapshot *string    `gorm:"->;column:class_section_subject_teacher_subject_name_snapshot" json:"class_section_subject_teacher_subject_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherSubjectCodeSnapshot *string    `gorm:"->;column:class_section_subject_teacher_subject_code_snapshot" json:"class_section_subject_teacher_subject_code_snapshot,omitempty"`
	ClassSectionSubjectTeacherSubjectSlugSnapshot *string    `gorm:"->;column:class_section_subject_teacher_subject_slug_snapshot" json:"class_section_subject_teacher_subject_slug_snapshot,omitempty"`

	/* ===== Status & audit ===== */
	ClassSectionSubjectTeacherIsActive  bool           `gorm:"not null;default:true;column:class_section_subject_teacher_is_active" json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_created_at" json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_updated_at" json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt gorm.DeletedAt `gorm:"column:class_section_subject_teacher_deleted_at;index" json:"class_section_subject_teacher_deleted_at,omitempty"`
}

func (ClassSectionSubjectTeacherModel) TableName() string {
	return "class_section_subject_teachers"
}
