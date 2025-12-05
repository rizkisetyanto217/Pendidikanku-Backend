// file: internals/features/attendance/model/class_attendance_session_participant_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================
   ENUMS (selaras dgn DB)
   ========================= */

type ParticipantKind string

const (
	ParticipantKindStudent   ParticipantKind = "student"
	ParticipantKindTeacher   ParticipantKind = "teacher"
	ParticipantKindAssistant ParticipantKind = "assistant"
	ParticipantKindGuest     ParticipantKind = "guest"
)

type TeacherRole string

const (
	TeacherRolePrimary    TeacherRole = "primary"
	TeacherRoleCo         TeacherRole = "co"
	TeacherRoleSubstitute TeacherRole = "substitute"
	TeacherRoleObserver   TeacherRole = "observer"
	TeacherRoleAssistant  TeacherRole = "assistant"
)

type AttendanceState string

const (
	AttendanceStatePresent  AttendanceState = "present"
	AttendanceStateAbsent   AttendanceState = "absent"
	AttendanceStateLate     AttendanceState = "late"
	AttendanceStateExcused  AttendanceState = "excused"
	AttendanceStateSick     AttendanceState = "sick"
	AttendanceStateLeave    AttendanceState = "leave"
	AttendanceStateUnmarked AttendanceState = "unmarked" // default
)

/* =========================================
   MODEL: class_attendance_session_participants
   ========================================= */

type ClassAttendanceSessionParticipantModel struct {
	// PK
	ClassAttendanceSessionParticipantID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:class_attendance_session_participant_id" json:"class_attendance_session_participant_id"`

	// tenant & relasi utama
	ClassAttendanceSessionParticipantSchoolID  uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_participant_school_id" json:"class_attendance_session_participant_school_id"`
	ClassAttendanceSessionParticipantSessionID uuid.UUID `gorm:"type:uuid;not null;column:class_attendance_session_participant_session_id" json:"class_attendance_session_participant_session_id"`

	// jenis peserta
	ClassAttendanceSessionParticipantKind ParticipantKind `gorm:"type:participant_kind_enum;not null;column:class_attendance_session_participant_kind" json:"class_attendance_session_participant_kind"`

	// relasi detail (opsional, tergantung kind)
	ClassAttendanceSessionParticipantSchoolStudentID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_participant_school_student_id" json:"class_attendance_session_participant_school_student_id,omitempty"`
	ClassAttendanceSessionParticipantSchoolTeacherID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_participant_school_teacher_id" json:"class_attendance_session_participant_school_teacher_id,omitempty"`

	ClassAttendanceSessionParticipantTeacherRole *TeacherRole `gorm:"type:teacher_role_enum;column:class_attendance_session_participant_teacher_role" json:"class_attendance_session_participant_teacher_role,omitempty"`

	// status kehadiran (enum global)
	ClassAttendanceSessionParticipantState AttendanceState `gorm:"type:attendance_state_enum;not null;default:'unmarked';column:class_attendance_session_participant_state" json:"class_attendance_session_participant_state"`

	// jam checkin/checkout
	ClassAttendanceSessionParticipantCheckinAt  *time.Time `gorm:"type:timestamptz;column:class_attendance_session_participant_checkin_at" json:"class_attendance_session_participant_checkin_at,omitempty"`
	ClassAttendanceSessionParticipantCheckoutAt *time.Time `gorm:"type:timestamptz;column:class_attendance_session_participant_checkout_at" json:"class_attendance_session_participant_checkout_at,omitempty"`

	// jenis kegiatan/tipe absensi (opsional)
	ClassAttendanceSessionParticipantTypeID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_participant_type_id" json:"class_attendance_session_participant_type_id,omitempty"`

	// kualitas/penilaian harian (opsional)
	ClassAttendanceSessionParticipantDesc     *string  `gorm:"type:text;column:class_attendance_session_participant_desc" json:"class_attendance_session_participant_desc,omitempty"`
	ClassAttendanceSessionParticipantScore    *float64 `gorm:"type:numeric(5,2);column:class_attendance_session_participant_score" json:"class_attendance_session_participant_score,omitempty"`
	ClassAttendanceSessionParticipantIsPassed *bool    `gorm:"column:class_attendance_session_participant_is_passed" json:"class_attendance_session_participant_is_passed,omitempty"`

	// meta penandaan
	ClassAttendanceSessionParticipantMarkedAt          *time.Time `gorm:"type:timestamptz;column:class_attendance_session_participant_marked_at" json:"class_attendance_session_participant_marked_at,omitempty"`
	ClassAttendanceSessionParticipantMarkedByTeacherID *uuid.UUID `gorm:"type:uuid;column:class_attendance_session_participant_marked_by_teacher_id" json:"class_attendance_session_participant_marked_by_teacher_id,omitempty"`

	// metode absen
	ClassAttendanceSessionParticipantMethod *string `gorm:"type:varchar(16);column:class_attendance_session_participant_method" json:"class_attendance_session_participant_method,omitempty"`

	// geolocation
	ClassAttendanceSessionParticipantLat       *float64 `gorm:"column:class_attendance_session_participant_lat" json:"class_attendance_session_participant_lat,omitempty"`
	ClassAttendanceSessionParticipantLng       *float64 `gorm:"column:class_attendance_session_participant_lng" json:"class_attendance_session_participant_lng,omitempty"`
	ClassAttendanceSessionParticipantDistanceM *int     `gorm:"column:class_attendance_session_participant_distance_m" json:"class_attendance_session_participant_distance_m,omitempty"`

	// keterlambatan (detik)
	ClassAttendanceSessionParticipantLateSeconds *int `gorm:"column:class_attendance_session_participant_late_seconds" json:"class_attendance_session_participant_late_seconds,omitempty"`

	// Snapshot users_profile (per siswa saat sesi dibuat)
	ClassAttendanceSessionParticipantUserProfileNameSnapshot              *string `gorm:"type:varchar(80);column:class_attendance_session_participant_user_profile_name_snapshot" json:"class_attendance_session_participant_user_profile_name_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileAvatarURLSnapshot         *string `gorm:"type:varchar(255);column:class_attendance_session_participant_user_profile_avatar_url_snapshot" json:"class_attendance_session_participant_user_profile_avatar_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileWhatsappURLSnapshot       *string `gorm:"type:varchar(50);column:class_attendance_session_participant_user_profile_whatsapp_url_snapshot" json:"class_attendance_session_participant_user_profile_whatsapp_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileParentNameSnapshot        *string `gorm:"type:varchar(80);column:class_attendance_session_participant_user_profile_parent_name_snapshot" json:"class_attendance_session_participant_user_profile_parent_name_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileParentWhatsappURLSnapshot *string `gorm:"type:varchar(50);column:class_attendance_session_participant_user_profile_parent_whatsapp_url_snapshot" json:"class_attendance_session_participant_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	ClassAttendanceSessionParticipantUserProfileGenderSnapshot            *string `gorm:"type:varchar(20);column:class_attendance_session_participant_user_profile_gender_snapshot" json:"class_attendance_session_participant_user_profile_gender_snapshot,omitempty"`

	// catatan tambahan
	ClassAttendanceSessionParticipantUserNote    *string `gorm:"type:text;column:class_attendance_session_participant_user_note" json:"class_attendance_session_participant_user_note,omitempty"`
	ClassAttendanceSessionParticipantTeacherNote *string `gorm:"type:text;column:class_attendance_session_participant_teacher_note" json:"class_attendance_session_participant_teacher_note,omitempty"`

	// locking
	ClassAttendanceSessionParticipantLockedAt *time.Time `gorm:"type:timestamptz;column:class_attendance_session_participant_locked_at" json:"class_attendance_session_participant_locked_at,omitempty"`

	// audit
	ClassAttendanceSessionParticipantCreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_participant_created_at" json:"class_attendance_session_participant_created_at"`
	ClassAttendanceSessionParticipantUpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now();column:class_attendance_session_participant_updated_at" json:"class_attendance_session_participant_updated_at"`
	ClassAttendanceSessionParticipantDeletedAt gorm.DeletedAt `gorm:"column:class_attendance_session_participant_deleted_at;index" json:"class_attendance_session_participant_deleted_at,omitempty"`
}

func (ClassAttendanceSessionParticipantModel) TableName() string {
	return "class_attendance_session_participants"
}
