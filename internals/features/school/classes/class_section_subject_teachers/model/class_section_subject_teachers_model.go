// file: internals/features/school/sectionsubjectteachers/model/class_section_subject_teacher_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/*
	============================
	  ENUM delivery mode (map ke DB type: class_delivery_mode_enum)

============================
*/
type ClassDeliveryMode string

const (
	DeliveryModeOffline ClassDeliveryMode = "offline"
	DeliveryModeOnline  ClassDeliveryMode = "online"
	DeliveryModeHybrid  ClassDeliveryMode = "hybrid"
)

/*
	=========================================================
	  MODEL: class_section_subject_teachers
	  Sinkron dengan SQL:
	  - FK: section_id, class_subject_book_id, teacher_id (tenant-safe)
	  - Kolom delivery mode: class_section_subject_teacher_delivery_mode
	  - Snapshot: class_subject_book_snapshot (gabungan book+subject)
	  - Generated columns: room_*, teacher_*, book_*, subject_*

=========================================================
*/
type ClassSectionSubjectTeacherModel struct {
	// ===== PK
	ClassSectionSubjectTeacherID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_section_subject_teacher_id" json:"class_section_subject_teacher_id"`

	// ===== Tenant
	ClassSectionSubjectTeacherSchoolID uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_school_id" json:"class_section_subject_teacher_school_id"`

	// ===== Relations (IDs)
	ClassSectionSubjectTeacherSectionID          uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_section_id" json:"class_section_subject_teacher_section_id"`
	ClassSectionSubjectTeacherClassSubjectBookID uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_class_subject_book_id" json:"class_section_subject_teacher_class_subject_book_id"`
	ClassSectionSubjectTeacherTeacherID          uuid.UUID `gorm:"type:uuid;not null;column:class_section_subject_teacher_teacher_id" json:"class_section_subject_teacher_teacher_id"`

	// ===== Identitas
	ClassSectionSubjectTeacherName string  `gorm:"type:varchar(160);not null;column:class_section_subject_teacher_name" json:"class_section_subject_teacher_name"`
	ClassSectionSubjectTeacherSlug *string `gorm:"type:varchar(160);column:class_section_subject_teacher_slug" json:"class_section_subject_teacher_slug,omitempty"`

	// ===== Fasilitas
	ClassSectionSubjectTeacherDescription *string    `gorm:"type:text;column:class_section_subject_teacher_description" json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherRoomID      *uuid.UUID `gorm:"type:uuid;column:class_section_subject_teacher_room_id" json:"class_section_subject_teacher_room_id,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string    `gorm:"type:text;column:class_section_subject_teacher_group_url" json:"class_section_subject_teacher_group_url,omitempty"`

	// ===== Agregat & kapasitas
	ClassSectionSubjectTeacherTotalAttendance int  `gorm:"not null;default:0;column:class_section_subject_teacher_total_attendance" json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherCapacity        *int `gorm:"column:class_section_subject_teacher_capacity" json:"class_section_subject_teacher_capacity,omitempty"` // NULL = unlimited
	ClassSectionSubjectTeacherEnrolledCount   int  `gorm:"not null;default:0;column:class_section_subject_teacher_enrolled_count" json:"class_section_subject_teacher_enrolled_count"`

	// ===== Delivery mode (per DDL baru: class_section_subject_teacher_delivery_mode)
	ClassSectionSubjectTeacherDeliveryMode ClassDeliveryMode `gorm:"type:class_delivery_mode_enum;not null;default:'offline';column:class_section_subject_teacher_delivery_mode" json:"class_section_subject_teacher_delivery_mode"`

	/* =======================
	   SNAPSHOTS (JSONB)
	======================= */

	// ---- Room snapshot + generated
	ClassSectionSubjectTeacherRoomSnapshot     *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_room_snapshot" json:"class_section_subject_teacher_room_snapshot,omitempty"`
	ClassSectionSubjectTeacherRoomNameSnap     *string         `gorm:"->;column:class_section_subject_teacher_room_name_snap" json:"class_section_subject_teacher_room_name_snap,omitempty"`
	ClassSectionSubjectTeacherRoomSlugSnap     *string         `gorm:"->;column:class_section_subject_teacher_room_slug_snap" json:"class_section_subject_teacher_room_slug_snap,omitempty"`
	ClassSectionSubjectTeacherRoomLocationSnap *string         `gorm:"->;column:class_section_subject_teacher_room_location_snap" json:"class_section_subject_teacher_room_location_snap,omitempty"`

	// ---- People snapshots + generated
	ClassSectionSubjectTeacherTeacherSnapshot          *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_teacher_snapshot" json:"class_section_subject_teacher_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantTeacherSnapshot *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_assistant_teacher_snapshot" json:"class_section_subject_teacher_assistant_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherTeacherNameSnap          *string         `gorm:"->;column:class_section_subject_teacher_teacher_name_snap" json:"class_section_subject_teacher_teacher_name_snap,omitempty"`
	ClassSectionSubjectTeacherAssistantTeacherNameSnap *string         `gorm:"->;column:class_section_subject_teacher_assistant_teacher_name_snap" json:"class_section_subject_teacher_assistant_teacher_name_snap,omitempty"`

	// ---- CLASS_SUBJECT_BOOK snapshot (gabungan book + subject)
	// Rekomendasi struktur JSON:
	// { "book": {"title","author","slug","image_url"}, "subject": {"name","code","slug","url"} }
	ClassSectionSubjectTeacherClassSubjectBookSnapshot *datatypes.JSON `gorm:"type:jsonb;column:class_section_subject_teacher_class_subject_book_snapshot" json:"class_section_subject_teacher_class_subject_book_snapshot,omitempty"`

	// ---- Generated dari CLASS_SUBJECT_BOOK snapshot: BOOK*
	ClassSectionSubjectTeacherBookTitleSnap    *string `gorm:"->;column:class_section_subject_teacher_book_title_snap" json:"class_section_subject_teacher_book_title_snap,omitempty"`
	ClassSectionSubjectTeacherBookAuthorSnap   *string `gorm:"->;column:class_section_subject_teacher_book_author_snap" json:"class_section_subject_teacher_book_author_snap,omitempty"`
	ClassSectionSubjectTeacherBookSlugSnap     *string `gorm:"->;column:class_section_subject_teacher_book_slug_snap" json:"class_section_subject_teacher_book_slug_snap,omitempty"`
	ClassSectionSubjectTeacherBookImageURLSnap *string `gorm:"->;column:class_section_subject_teacher_book_image_url_snap" json:"class_section_subject_teacher_book_image_url_snap,omitempty"`

	// ---- Generated dari CLASS_SUBJECT_BOOK snapshot: SUBJECT*
	ClassSectionSubjectTeacherSubjectNameSnap *string `gorm:"->;column:class_section_subject_teacher_subject_name_snap" json:"class_section_subject_teacher_subject_name_snap,omitempty"`
	ClassSectionSubjectTeacherSubjectCodeSnap *string `gorm:"->;column:class_section_subject_teacher_subject_code_snap" json:"class_section_subject_teacher_subject_code_snap,omitempty"`
	ClassSectionSubjectTeacherSubjectSlugSnap *string `gorm:"->;column:class_section_subject_teacher_subject_slug_snap" json:"class_section_subject_teacher_subject_slug_snap,omitempty"`

	// ---- Books snapshot array (tetap, untuk daftar buku terpakai)
	ClassSectionSubjectTeacherBooksSnapshot datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb;column:class_section_subject_teacher_books_snapshot" json:"class_section_subject_teacher_books_snapshot"`

	// (Opsional) kolom turunan jika kamu menambahkannya di DB (tidak wajib ada)
	ClassSectionSubjectTeacherBooksCount       *int    `gorm:"->;column:class_section_subject_teacher_books_count" json:"class_section_subject_teacher_books_count,omitempty"`
	ClassSectionSubjectTeacherPrimaryBookTitle *string `gorm:"->;column:class_section_subject_teacher_primary_book_title" json:"class_section_subject_teacher_primary_book_title,omitempty"`

	// ===== Status & audit
	ClassSectionSubjectTeacherIsActive  bool           `gorm:"not null;default:true;column:class_section_subject_teacher_is_active" json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_created_at" json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_section_subject_teacher_updated_at" json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt gorm.DeletedAt `gorm:"column:class_section_subject_teacher_deleted_at;index" json:"class_section_subject_teacher_deleted_at,omitempty"`
}

func (ClassSectionSubjectTeacherModel) TableName() string {
	return "class_section_subject_teachers"
}
