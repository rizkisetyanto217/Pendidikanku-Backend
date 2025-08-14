package model

import (
	"time"

	"github.com/google/uuid"
)

/*
Mapping status (SMALLINT):
0 = absent
1 = present
2 = sick
3 = leave
*/
type AttendanceStatus int16

const (
	AttendanceAbsent  AttendanceStatus = 0
	AttendancePresent AttendanceStatus = 1
	AttendanceSick    AttendanceStatus = 2
	AttendanceLeave   AttendanceStatus = 3
)

type UserClassAttendanceEntryModel struct {
	UserClassAttendanceEntriesID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_class_attendance_entries_id" json:"user_class_attendance_entries_id"`

	UserClassAttendanceEntriesSessionID   uuid.UUID `gorm:"type:uuid;not null;column:user_class_attendance_entries_session_id;index:idx_ucae_session_status,priority:1;index:idx_ucae_masjid_session,priority:2;uniqueIndex:uq_ucae_session_userclass,priority:1" json:"user_class_attendance_entries_session_id"`
	UserClassAttendanceEntriesUserClassID uuid.UUID `gorm:"type:uuid;not null;column:user_class_attendance_entries_user_class_id;index:idx_ucae_userclass_created_at,priority:1;index:idx_ucae_userclass_score,priority:1;uniqueIndex:uq_ucae_session_userclass,priority:2" json:"user_class_attendance_entries_user_class_id"`
	UserClassAttendanceEntriesMasjidID    uuid.UUID `gorm:"type:uuid;not null;column:user_class_attendance_entries_masjid_id;index:idx_ucae_masjid_created_at,priority:1;index:idx_ucae_masjid_session,priority:1" json:"user_class_attendance_entries_masjid_id"`

	// SMALLINT di DB (0=absent,1=present,2=sick,3=leave)
	UserClassAttendanceEntriesAttendanceStatus AttendanceStatus `gorm:"type:smallint;not null;column:user_class_attendance_entries_attendance_status;index:idx_ucae_session_status,priority:2" json:"user_class_attendance_entries_attendance_status"`

	// Skor 0..100 (nullable)
	UserClassAttendanceEntriesScore *int `gorm:"column:user_class_attendance_entries_score;index:idx_ucae_userclass_score,priority:2" json:"user_class_attendance_entries_score,omitempty"`

	// Lulus/tidak (nullable)
	UserClassAttendanceEntriesGradePassed *bool `gorm:"column:user_class_attendance_entries_grade_passed;index:idx_ucae_grade_passed" json:"user_class_attendance_entries_grade_passed,omitempty"`

	UserClassAttendanceEntriesMaterialPersonal *string `gorm:"column:user_class_attendance_entries_material_personal" json:"user_class_attendance_entries_material_personal,omitempty"`
	UserClassAttendanceEntriesPersonalNote     *string `gorm:"column:user_class_attendance_entries_personal_note" json:"user_class_attendance_entries_personal_note,omitempty"`
	UserClassAttendanceEntriesMemorization     *string `gorm:"column:user_class_attendance_entries_memorization" json:"user_class_attendance_entries_memorization,omitempty"`
	UserClassAttendanceEntriesHomework         *string `gorm:"column:user_class_attendance_entries_homework" json:"user_class_attendance_entries_homework,omitempty"`

	// Generated tsvector (read-only)
	UserClassAttendanceEntriesSearch *string `gorm:"type:tsvector;column:user_class_attendance_entries_search;->" json:"-"`

	UserClassAttendanceEntriesCreatedAt time.Time  `gorm:"column:user_class_attendance_entries_created_at;autoCreateTime" json:"user_class_attendance_entries_created_at"`
	UserClassAttendanceEntriesUpdatedAt *time.Time `gorm:"column:user_class_attendance_entries_updated_at;autoUpdateTime" json:"user_class_attendance_entries_updated_at,omitempty"`
}

func (UserClassAttendanceEntryModel) TableName() string {
	return "user_class_attendance_entries"
}
