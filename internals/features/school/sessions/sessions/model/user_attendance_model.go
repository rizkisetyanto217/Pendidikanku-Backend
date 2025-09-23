// file: internals/features/school/attendance_assesment/user_result/user_attendance/model/user_attendance_model.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ===================== ENUMS ===================== */

type UserAttendanceStatus string

const (
	UserAttendanceUnmarked UserAttendanceStatus = "unmarked"
	UserAttendancePresent  UserAttendanceStatus = "present"
	UserAttendanceAbsent   UserAttendanceStatus = "absent"
	UserAttendanceExcused  UserAttendanceStatus = "excused"
	UserAttendanceLate     UserAttendanceStatus = "late"
)

type UserAttendanceMethod string

const (
	UserAttendanceMethodManual UserAttendanceMethod = "manual"
	UserAttendanceMethodQR     UserAttendanceMethod = "qr"
	UserAttendanceMethodGeo    UserAttendanceMethod = "geo"
	UserAttendanceMethodImport UserAttendanceMethod = "import"
	UserAttendanceMethodAPI    UserAttendanceMethod = "api"
	UserAttendanceMethodSelf   UserAttendanceMethod = "self"
)

/* ===================== MODEL ===================== */

type UserAttendanceModel struct {
	// PK
	UserAttendanceID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_attendance_id" json:"user_attendance_id"`

	// Tenant & relasi utama
	UserAttendanceMasjidID        uuid.UUID `gorm:"type:uuid;not null;column:user_attendance_masjid_id" json:"user_attendance_masjid_id"`
	UserAttendanceSessionID       uuid.UUID `gorm:"type:uuid;not null;column:user_attendance_session_id;index:idx_user_attendance_session,where:user_attendance_deleted_at IS NULL" json:"user_attendance_session_id"`
	UserAttendanceMasjidStudentID uuid.UUID `gorm:"type:uuid;not null;column:user_attendance_masjid_student_id;index:idx_user_attendance_student,where:user_attendance_deleted_at IS NULL" json:"user_attendance_masjid_student_id"`

	// Status (default: 'unmarked' — sesuai CHECK di DDL)
	UserAttendanceStatus UserAttendanceStatus `gorm:"type:varchar(16);not null;default:unmarked;column:user_attendance_status;index:idx_user_attendance_status,where:user_attendance_deleted_at IS NULL" json:"user_attendance_status"`

	// Jenis kegiatan/tipe absensi (nullable)
	UserAttendanceTypeID *uuid.UUID `gorm:"type:uuid;column:user_attendance_type_id;index:idx_user_attendance_type_id,where:user_attendance_deleted_at IS NULL" json:"user_attendance_type_id,omitempty"`

	// Kualitas/penilaian harian (opsional) — DB CHECK 0..100
	UserAttendanceDesc     *string  `gorm:"type:text;column:user_attendance_desc" json:"user_attendance_desc,omitempty"`
	UserAttendanceScore    *float64 `gorm:"type:numeric(5,2);column:user_attendance_score" json:"user_attendance_score,omitempty"`
	UserAttendanceIsPassed *bool    `gorm:"column:user_attendance_is_passed" json:"user_attendance_is_passed,omitempty"`

	// Meta penandaan
	UserAttendanceMarkedAt         *time.Time `gorm:"column:user_attendance_marked_at" json:"user_attendance_marked_at,omitempty"`
	UserAttendanceMarkedByTeacherID *uuid.UUID `gorm:"type:uuid;column:user_attendance_marked_by_teacher_id" json:"user_attendance_marked_by_teacher_id,omitempty"`

	// Metode absen (nullable; restricted di DB via CHECK)
	UserAttendanceMethod *UserAttendanceMethod `gorm:"type:varchar(16);column:user_attendance_method" json:"user_attendance_method,omitempty"`

	// Geolocation (opsional)
	UserAttendanceLat        *float64 `gorm:"type:double precision;column:user_attendance_lat" json:"user_attendance_lat,omitempty"`
	UserAttendanceLng        *float64 `gorm:"type:double precision;column:user_attendance_lng" json:"user_attendance_lng,omitempty"`
	UserAttendanceDistanceM  *int     `gorm:"column:user_attendance_distance_m" json:"user_attendance_distance_m,omitempty"`

	// Keterlambatan (detik; >= 0)
	UserAttendanceLateSeconds *int `gorm:"column:user_attendance_late_seconds" json:"user_attendance_late_seconds,omitempty"`

	// Catatan tambahan
	UserAttendanceUserNote    *string `gorm:"type:text;column:user_attendance_user_note" json:"user_attendance_user_note,omitempty"`
	UserAttendanceTeacherNote *string `gorm:"type:text;column:user_attendance_teacher_note" json:"user_attendance_teacher_note,omitempty"`

	// Locking (opsional)
	UserAttendanceLockedAt *time.Time `gorm:"column:user_attendance_locked_at" json:"user_attendance_locked_at,omitempty"`

	// Audit
	UserAttendanceCreatedAt time.Time      `gorm:"column:user_attendance_created_at;autoCreateTime" json:"user_attendance_created_at"`
	UserAttendanceUpdatedAt time.Time      `gorm:"column:user_attendance_updated_at;autoUpdateTime" json:"user_attendance_updated_at"`
	UserAttendanceDeletedAt gorm.DeletedAt `gorm:"column:user_attendance_deleted_at;index" json:"user_attendance_deleted_at,omitempty"`
}

func (UserAttendanceModel) TableName() string {
	return "user_attendance"
}

/*
Catatan:
- Unique partial index uq_user_attendance_alive (masjid_id, session_id, masjid_student_id WHERE deleted_at IS NULL)
  tidak bisa didefinisikan langsung via tag GORM. Tetap dibuat via migrasi SQL (sudah ada di DDL kamu).
- CHECK constraints (status/method/score/late_seconds) juga dikelola di DDL.
- Index tags di atas meniru nama index DDL dan menambahkan WHERE agar konsisten saat auto-migrate,
  tetapi sumber kebenaran tetap di migrasi SQL kamu.
*/
