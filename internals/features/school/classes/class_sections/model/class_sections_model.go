package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ClassSection merepresentasikan table class_sections
type ClassSectionModel struct {
	// ================= PK & Tenant =================
	ClassSectionID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_section_id;uniqueIndex:uq_class_section_id_masjid" json:"class_section_id"`
	ClassSectionMasjidID uuid.UUID `gorm:"type:uuid;not null;column:class_section_masjid_id;uniqueIndex:uq_class_section_id_masjid" json:"class_section_masjid_id"`

	// ================ Relasi inti ==================
	ClassSectionClassID            uuid.UUID  `gorm:"type:uuid;not null;column:class_section_class_id" json:"class_section_class_id"`
	ClassSectionTeacherID          *uuid.UUID `gorm:"type:uuid;column:class_section_teacher_id" json:"class_section_teacher_id,omitempty"`
	ClassSectionAssistantTeacherID *uuid.UUID `gorm:"type:uuid;column:class_section_assistant_teacher_id" json:"class_section_assistant_teacher_id,omitempty"`
	ClassSectionClassRoomID        *uuid.UUID `gorm:"type:uuid;column:class_section_class_room_id" json:"class_section_class_room_id,omitempty"`
	ClassSectionLeaderStudentID    *uuid.UUID `gorm:"type:uuid;column:class_section_leader_student_id" json:"class_section_leader_student_id,omitempty"`

	// ================= Identitas ===================
	ClassSectionSlug string  `gorm:"type:varchar(160);not null;column:class_section_slug" json:"class_section_slug"`
	ClassSectionName string  `gorm:"type:varchar(100);not null;column:class_section_name" json:"class_section_name"`
	ClassSectionCode *string `gorm:"type:varchar(50);column:class_section_code" json:"class_section_code,omitempty"`

	// ============== Jadwal sederhana ===============
	ClassSectionSchedule *string `gorm:"type:text;column:class_section_schedule" json:"class_section_schedule,omitempty"`

	// ======= Kapasitas & counter dasar =============
	ClassSectionCapacity      *int `gorm:"column:class_section_capacity" json:"class_section_capacity,omitempty"`
	ClassSectionTotalStudents int  `gorm:"not null;default:0;column:class_section_total_students" json:"class_section_total_students"`

	// ============== Meeting / Group ================
	ClassSectionGroupURL *string `gorm:"type:text;column:class_section_group_url" json:"class_section_group_url,omitempty"`

	// ========== Image (2-slot + retensi) ===========
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

	// ============== Status & audit =================
	ClassSectionIsActive  bool           `gorm:"not null;default:true;column:class_section_is_active" json:"class_section_is_active"`
	ClassSectionCreatedAt time.Time      `gorm:"not null;autoCreateTime;column:class_section_created_at" json:"class_section_created_at"`
	ClassSectionUpdatedAt time.Time      `gorm:"not null;autoUpdateTime;column:class_section_updated_at" json:"class_section_updated_at"`
	ClassSectionDeletedAt gorm.DeletedAt `gorm:"column:class_section_deleted_at;index" json:"class_section_deleted_at,omitempty"`

	// ============== Snapshots & hints ==============
	ClassSectionClassSlugSnapshot *string `gorm:"type:varchar(160);column:class_section_class_slug_snapshot" json:"class_section_class_slug_snapshot,omitempty"`

	// ---------- Parent snapshots ----------
	ClassSectionParentNameSnapshot  *string `gorm:"type:varchar(120);column:class_section_parent_name_snapshot" json:"class_section_parent_name_snapshot,omitempty"`
	ClassSectionParentCodeSnapshot  *string `gorm:"type:varchar(50);column:class_section_parent_code_snapshot" json:"class_section_parent_code_snapshot,omitempty"`
	ClassSectionParentSlugSnapshot  *string `gorm:"type:varchar(160);column:class_section_parent_slug_snapshot" json:"class_section_parent_slug_snapshot,omitempty"`
	ClassSectionParentLevelSnapshot *string `gorm:"type:varchar(50);column:class_section_parent_level_snapshot" json:"class_section_parent_level_snapshot,omitempty"`
	ClassSectionParentURLSnapshot   *string `gorm:"type:text;column:class_section_parent_url_snapshot" json:"class_section_parent_url_snapshot,omitempty"`

	// ---------- People snapshots ----------
	ClassSectionTeacherNameSnapshot          *string `gorm:"type:varchar(120);column:class_section_teacher_name_snapshot" json:"class_section_teacher_name_snapshot,omitempty"`
	ClassSectionAssistantTeacherNameSnapshot *string `gorm:"type:varchar(120);column:class_section_assistant_teacher_name_snapshot" json:"class_section_assistant_teacher_name_snapshot,omitempty"`
	ClassSectionLeaderStudentNameSnapshot    *string `gorm:"type:varchar(120);column:class_section_leader_student_name_snapshot" json:"class_section_leader_student_name_snapshot,omitempty"`

	// kontak (snapshot)
	ClassSectionTeacherContactPhoneSnapshot          *string `gorm:"type:varchar(50);column:class_section_teacher_contact_phone_snapshot" json:"class_section_teacher_contact_phone_snapshot,omitempty"`
	ClassSectionAssistantTeacherContactPhoneSnapshot *string `gorm:"type:varchar(50);column:class_section_assistant_teacher_contact_phone_snapshot" json:"class_section_assistant_teacher_contact_phone_snapshot,omitempty"`
	ClassSectionLeaderStudentContactPhoneSnapshot    *string `gorm:"type:varchar(50);column:class_section_leader_student_contact_phone_snapshot" json:"class_section_leader_student_contact_phone_snapshot,omitempty"`

	// avatar snapshots
	ClassSectionTeacherAvatarURLSnapshot          *string `gorm:"type:text;column:class_section_teacher_avatar_url_snapshot" json:"class_section_teacher_avatar_url_snapshot,omitempty"`
	ClassSectionAssistantTeacherAvatarURLSnapshot *string `gorm:"type:text;column:class_section_assistant_teacher_avatar_url_snapshot" json:"class_section_assistant_teacher_avatar_url_snapshot,omitempty"`

	// ---------- ROOM snapshots ----------
	ClassSectionRoomNameSnapshot      *string `gorm:"type:varchar(120);column:class_section_room_name_snapshot" json:"class_section_room_name_snapshot,omitempty"`
	ClassSectionRoomSlugSnapshot      *string `gorm:"type:varchar(160);column:class_section_room_slug_snapshot" json:"class_section_room_slug_snapshot,omitempty"`
	ClassSectionRoomLocationSnapshot  *string `gorm:"type:varchar(160);column:class_section_room_location_snapshot" json:"class_section_room_location_snapshot,omitempty"`
	ClassSectionRoomJoinURLSnapshot   *string `gorm:"type:varchar(210);column:class_section_room_join_url_snapshot" json:"class_section_room_join_url_snapshot,omitempty"`
	ClassSectionRoomMeetingIDSnapshot *string `gorm:"type:varchar(50);column:class_section_room_meeting_id_snapshot" json:"class_section_room_meeting_id_snapshot,omitempty"`
	ClassSectionRoomPasscodeSnapshot  *string `gorm:"type:varchar(50);column:class_section_room_passcode_snapshot" json:"class_section_room_passcode_snapshot,omitempty"`
	ClassSectionRoomPlatformSnapshot  *string `gorm:"type:varchar(50);column:class_section_room_platform_snapshot" json:"class_section_room_platform_snapshot,omitempty"`

	// housekeeping snapshot
	ClassSectionSnapshotUpdatedAt *time.Time `gorm:"column:class_section_snapshot_updated_at" json:"class_section_snapshot_updated_at,omitempty"`

	// ---------- TERM (lean snapshots) ----------
	ClassSectionTermID                *uuid.UUID `gorm:"type:uuid;column:class_section_term_id" json:"class_section_term_id,omitempty"`
	ClassSectionTermNameSnapshot      *string    `gorm:"type:varchar(120);column:class_section_term_name_snapshot" json:"class_section_term_name_snapshot,omitempty"`
	ClassSectionTermSlugSnapshot      *string    `gorm:"type:varchar(160);column:class_section_term_slug_snapshot" json:"class_section_term_slug_snapshot,omitempty"`
	ClassSectionTermYearLabelSnapshot *string    `gorm:"type:varchar(20);column:class_section_term_year_label_snapshot" json:"class_section_term_year_label_snapshot,omitempty"`

	// ============== CSST CACHE (JSONB) ==============
	ClassSectionsCSST datatypes.JSON `gorm:"column:class_sections_csst;type:jsonb;not null;default:'[]'::jsonb" json:"class_sections_csst"`
}

// TableName memastikan GORM memakai nama tabel yang tepat
func (ClassSectionModel) TableName() string { return "class_sections" }
