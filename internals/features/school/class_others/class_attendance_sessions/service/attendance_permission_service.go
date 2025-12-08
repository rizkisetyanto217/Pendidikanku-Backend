// file: internals/features/school/classes/class_attendance_sessions/service/attendance_permission_service.go
package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/*
Service ini TIDAK membuat record absensi.
Hanya menjawab: "boleh nggak sih <kind> absen sekarang untuk session ini?"
Kind: student / teacher / assistant / guest
*/

// =============================
// Types
// =============================

// Hasil untuk controller / FE
type AttendancePermissionResult struct {
	Allowed bool   `json:"allowed"`
	Code    string `json:"code"`    // "ok", "session_canceled", "window_closed", ...
	Message string `json:"message"` // pesan human-readable

	// Info tambahan (SEMUA UTC)
	SessionID      uuid.UUID  `json:"session_id"`
	WindowMode     string     `json:"window_mode"`
	WindowStartUTC *time.Time `json:"window_start_utc,omitempty"`
	WindowEndUTC   *time.Time `json:"window_end_utc,omitempty"`
}

// Config yang diambil dari snapshot type
type attendanceTypeConfig struct {
	AllowStudentSelfAttendance bool
	AllowTeacherMarkAttendance bool
	RequireTeacherAttendance   bool

	WindowMode         string
	OpenOffsetMinutes  *int
	CloseOffsetMinutes *int
}

// Minimal kolom dari class_attendance_sessions yang dibutuhkan
type sessionPermissionRow struct {
	ID               uuid.UUID         `gorm:"column:class_attendance_session_id"`
	SchoolID         uuid.UUID         `gorm:"column:class_attendance_session_school_id"`
	Date             time.Time         `gorm:"column:class_attendance_session_date"`
	StartsAt         *time.Time        `gorm:"column:class_attendance_session_starts_at"`
	EndsAt           *time.Time        `gorm:"column:class_attendance_session_ends_at"`
	Status           string            `gorm:"column:class_attendance_session_status"`
	AttendanceStatus string            `gorm:"column:class_attendance_session_attendance_status"`
	Locked           bool              `gorm:"column:class_attendance_session_locked"`
	IsCanceled       bool              `gorm:"column:class_attendance_session_is_canceled"`
	TypeSnapshot     datatypes.JSONMap `gorm:"column:class_attendance_session_type_snapshot"`

	// tambahan buat guard relasi
	CSSTID    *uuid.UUID `gorm:"column:class_attendance_session_csst_id"`
	TeacherID *uuid.UUID `gorm:"column:class_attendance_session_teacher_id"`
}

// Service utama
type AttendancePermissionService struct {
	DB     *gorm.DB
	TZName string // timezone sekolah, default: "Asia/Jakarta"
}

func NewAttendancePermissionService(db *gorm.DB) *AttendancePermissionService {
	return &AttendancePermissionService{
		DB:     db,
		TZName: "Asia/Jakarta",
	}
}

// =============================
// Helper: baca snapshot → config
// =============================

func asBool(m datatypes.JSONMap, key string, def bool) bool {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	switch vv := v.(type) {
	case bool:
		return vv
	case string:
		s := strings.ToLower(strings.TrimSpace(vv))
		if s == "true" || s == "1" || s == "yes" {
			return true
		}
		if s == "false" || s == "0" || s == "no" {
			return false
		}
	}
	return def
}

func asIntPtr(m datatypes.JSONMap, key string) *int {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch vv := v.(type) {
	case int:
		return &vv
	case int32:
		i := int(vv)
		return &i
	case int64:
		i := int(vv)
		return &i
	case float32:
		i := int(vv)
		return &i
	case float64:
		i := int(vv)
		return &i
	case string:
		s := strings.TrimSpace(vv)
		if s == "" {
			return nil
		}
		if n, err := strconv.Atoi(s); err == nil {
			return &n
		}
	}
	return nil
}

func asString(m datatypes.JSONMap, key, def string) string {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	if s, ok2 := v.(string); ok2 {
		s2 := strings.TrimSpace(s)
		if s2 != "" {
			return s2
		}
	}
	return def
}

func extractTypeConfig(snap datatypes.JSONMap) attendanceTypeConfig {
	// default sama dengan enum SQL: same_day
	mode := strings.ToLower(asString(snap, "attendance_window_mode", "same_day"))

	return attendanceTypeConfig{
		AllowStudentSelfAttendance: asBool(snap, "allow_student_self_attendance", true),
		AllowTeacherMarkAttendance: asBool(snap, "allow_teacher_mark_attendance", true),
		RequireTeacherAttendance:   asBool(snap, "require_teacher_attendance", true),

		WindowMode:         mode,
		OpenOffsetMinutes:  asIntPtr(snap, "attendance_open_offset_minutes"),
		CloseOffsetMinutes: asIntPtr(snap, "attendance_close_offset_minutes"),
	}
}

// =============================
// Helper: timezone & window
// =============================

func (svc *AttendancePermissionService) getLocation() *time.Location {
	loc, err := time.LoadLocation(svc.TZName)
	if err != nil {
		// fallback Asia/Jakarta fixed
		return time.FixedZone("Asia/Jakarta", 7*3600)
	}
	return loc
}

// tgl sesi (date) dianggap merepresentasikan hari lokal sekolah
func (svc *AttendancePermissionService) sessionLocalDate(s sessionPermissionRow) time.Time {
	loc := svc.getLocation()
	dUTC := s.Date.UTC()
	return time.Date(dUTC.Year(), dUTC.Month(), dUTC.Day(), 0, 0, 0, 0, loc)
}

// Hitung window (hasil akhir dalam UTC)
func (svc *AttendancePermissionService) computeWindowUTC(
	s sessionPermissionRow,
	cfg attendanceTypeConfig,
) (startUTC, endUTC *time.Time) {
	loc := svc.getLocation()
	localDate := svc.sessionLocalDate(s)

	switch cfg.WindowMode {
	case "anytime":
		// tanpa batas
		return nil, nil

	case "same_day":
		// 00:00 - 23:59:59 hari H (lokal)
		startLocal := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, loc)
		endLocal := startLocal.Add(24*time.Hour - time.Second)
		su := startLocal.UTC()
		eu := endLocal.UTC()
		return &su, &eu

	case "three_days":
		// H-1 00:00 s/d H+1 23:59:59 (lokal)
		startLocal := time.Date(localDate.Year(), localDate.Month(), localDate.Day()-1, 0, 0, 0, 0, loc)
		endLocal := time.Date(localDate.Year(), localDate.Month(), localDate.Day()+1, 23, 59, 59, 0, loc)
		su := startLocal.UTC()
		eu := endLocal.UTC()
		return &su, &eu

	case "session_time":
		// pakai jam sesi langsung (sudah UTC)
		if s.StartsAt != nil && s.EndsAt != nil {
			return s.StartsAt, s.EndsAt
		}
		// kalau kosong, fallback → same_day
		fallthrough

	case "relative_window":
		if s.StartsAt == nil {
			// fallback same_day
			startLocal := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, loc)
			endLocal := startLocal.Add(24*time.Hour - time.Second)
			su := startLocal.UTC()
			eu := endLocal.UTC()
			return &su, &eu
		}

		base := s.StartsAt.UTC()
		openMin := 0
		closeMin := 0
		if cfg.OpenOffsetMinutes != nil {
			openMin = *cfg.OpenOffsetMinutes
		}
		if cfg.CloseOffsetMinutes != nil {
			closeMin = *cfg.CloseOffsetMinutes
		}

		su := base.Add(time.Duration(openMin) * time.Minute)
		eu := base.Add(time.Duration(closeMin) * time.Minute)
		if eu.Before(su) {
			tmp := su
			su = eu
			eu = tmp
		}
		return &su, &eu

	default:
		// mode tidak dikenal → fallback same_day
		startLocal := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, loc)
		endLocal := startLocal.Add(24*time.Hour - time.Second)
		su := startLocal.UTC()
		eu := endLocal.UTC()
		return &su, &eu
	}
}

// =============================
// Public API
// =============================

// kind: "student" / "teacher" / "assistant" / "guest"
// studentID / teacherID: diambil dari token (kalau relevan)
// - untuk kind=student → studentID WAJIB diisi
// - untuk kind=teacher → teacherID WAJIB diisi
func (svc *AttendancePermissionService) CheckSelfAttendancePermission(
	ctx context.Context,
	schoolID uuid.UUID,
	sessionID uuid.UUID,
	kind string,
	studentID *uuid.UUID,
	teacherID *uuid.UUID,
) (*AttendancePermissionResult, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		return nil, fmt.Errorf("kind wajib diisi (student/teacher)")
	}
	if schoolID == uuid.Nil || sessionID == uuid.Nil {
		return nil, fmt.Errorf("schoolID/sessionID kosong")
	}

	// 1) Ambil session (tenant guard)
	var row sessionPermissionRow
	if err := svc.DB.WithContext(ctx).
		Table("class_attendance_sessions").
		Where(`
			class_attendance_session_id = ?
			AND class_attendance_session_school_id = ?
			AND class_attendance_session_deleted_at IS NULL
		`, sessionID, schoolID).
		Select(`
			class_attendance_session_id,
			class_attendance_session_school_id,
			class_attendance_session_date,
			class_attendance_session_starts_at,
			class_attendance_session_ends_at,
			class_attendance_session_status,
			class_attendance_session_attendance_status,
			class_attendance_session_locked,
			class_attendance_session_is_canceled,
			class_attendance_session_type_snapshot,
			class_attendance_session_csst_id,
			class_attendance_session_teacher_id
		`).
		Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &AttendancePermissionResult{
				Allowed: false,
				Code:    "session_not_found",
				Message: "Sesi tidak ditemukan untuk sekolah ini",
			}, nil
		}
		return nil, err
	}

	res := &AttendancePermissionResult{
		Allowed:   false,
		Code:      "unknown",
		Message:   "Permission belum dievaluasi",
		SessionID: row.ID,
	}

	// 1.5) Guard: ID user sesuai kind
	switch kind {
	case "student":
		if studentID == nil || *studentID == uuid.Nil {
			res.Code = "student_id_required"
			res.Message = "ID siswa tidak ditemukan pada token untuk absensi mandiri."
			return res, nil
		}
	case "teacher":
		if teacherID == nil || *teacherID == uuid.Nil {
			res.Code = "teacher_id_required"
			res.Message = "ID guru tidak ditemukan pada token untuk absensi."
			return res, nil
		}
	}

	// 2) Status sesi
	if row.IsCanceled {
		res.Code = "session_canceled"
		res.Message = "Sesi ini sudah dibatalkan."
		return res, nil
	}
	if row.Locked {
		res.Code = "session_locked"
		res.Message = "Sesi ini sudah dikunci. Absensi tidak dapat diubah lagi."
		return res, nil
	}
	if strings.EqualFold(row.AttendanceStatus, "closed") {
		res.Code = "attendance_closed"
		res.Message = "Jendela absensi untuk sesi ini sudah ditutup."
		return res, nil
	}

	// 3) Config type dari snapshot
	cfg := extractTypeConfig(row.TypeSnapshot)
	res.WindowMode = cfg.WindowMode

	// 4) Guard relasi berdasarkan kind
	switch kind {
	case "student":
		// siswa wajib nyantol ke CSST
		if row.CSSTID == nil || *row.CSSTID == uuid.Nil {
			res.Code = "session_has_no_csst"
			res.Message = "Sesi ini tidak terhubung ke pengajar mapel (CSST), sehingga absensi siswa tidak dapat diverifikasi."
			return res, nil
		}

		// cek mapping student ↔ CSST
		// cek mapping student ↔ CSST
		var cnt int64
		if err := svc.DB.WithContext(ctx).
			Table("student_class_section_subject_teachers").
			Where(`
		student_csst_school_id = ?
		AND student_csst_student_id = ?
		AND student_csst_csst_id = ?
		AND student_csst_is_active = TRUE
		AND student_csst_deleted_at IS NULL
		AND (student_csst_from IS NULL OR student_csst_from <= ?)
		AND (student_csst_to   IS NULL OR student_csst_to   >= ?)
	`,
				schoolID,
				*studentID,
				*row.CSSTID,
				row.Date,
				row.Date,
			).
			Count(&cnt).Error; err != nil {
			return nil, err
		}

		if cnt == 0 {
			res.Code = "student_not_in_csst"
			res.Message = "Siswa ini tidak terdaftar pada kelas/mapel yang terkait sesi ini."
			return res, nil
		}

		// flag dari config type
		if !cfg.AllowStudentSelfAttendance {
			res.Code = "student_self_attendance_disabled"
			res.Message = "Sesi ini tidak mengizinkan absensi mandiri siswa."
			return res, nil
		}

	case "teacher":
		// optional: cek guru yang boleh absen harus sama dengan guru di session (jika ada)
		if row.TeacherID != nil && *row.TeacherID != uuid.Nil && teacherID != nil && *teacherID != *row.TeacherID {
			res.Code = "teacher_mismatch"
			res.Message = "Guru pada sesi ini berbeda dengan guru pada token. Absensi tidak diizinkan."
			return res, nil
		}

		if !cfg.AllowTeacherMarkAttendance {
			res.Code = "teacher_attendance_disabled"
			res.Message = "Sesi ini tidak mengizinkan absensi oleh guru."
			return res, nil
		}

	default:
		// assistant / guest: untuk sekarang ikut aturan window saja
	}

	// 5) Cek window waktu (semua dalam UTC)
	nowUTC := time.Now().UTC()
	ws, we := svc.computeWindowUTC(row, cfg)

	if ws != nil {
		wu := ws.UTC()
		res.WindowStartUTC = &wu
	}
	if we != nil {
		eu := we.UTC()
		res.WindowEndUTC = &eu
	}

	if ws != nil && nowUTC.Before(*ws) {
		res.Code = "window_not_open_yet"
		res.Message = "Jendela absensi untuk sesi ini belum dibuka."
		return res, nil
	}
	if we != nil && nowUTC.After(*we) {
		res.Code = "window_closed"
		res.Message = "Jendela absensi untuk sesi ini sudah ditutup."
		return res, nil
	}

	// 6) Lolos semua
	res.Allowed = true
	res.Code = "ok"
	res.Message = "Absensi diizinkan untuk sesi ini."
	return res, nil
}
