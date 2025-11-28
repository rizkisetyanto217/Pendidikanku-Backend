package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	attendanceModel "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	// DTO BARU (pindah namespace)
	attendanceDTO "madinahsalam_backend/internals/features/school/classes/class_attendance_sessions/dto"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

// di: internals/features/attendance/controller/class_attendance_session_participant_controller.go

/*
=========================================================
GET /class-attendance-session-participants
Query:
  - search: csv string (keyword bebas)
  - state_in: csv (present,absent,late,excused,sick,leave,unmarked)
  - method_in: csv (manual,qr,geo,import,api,self)
  - kind_in: csv (student,teacher,assistant,guest)
  - session_id, school_student_id, school_teacher_id, type_id, marked_by_teacher_id
  - created_ge, created_le, marked_ge, marked_le  (ISO date / RFC3339)
  - page, size (opsional, default: page=1,size=20)

=========================================================
*/
func (ctl *ClassAttendanceSessionParticipantController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Resolve school (tenant guard)
	var schoolID uuid.UUID
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureSchoolAccessDKM(c, mc); er == nil {
			schoolID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			schoolID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope school tidak ditemukan")
		}
	}

	// Parse query → DTO (pakai CSV dari DTO baru)
	var q attendanceDTO.ListClassAttendanceSessionParticipantQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid: "+err.Error())
	}

	// Pagination basic
	page := 1
	size := 20
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := strings.TrimSpace(c.Query("size")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			size = n
		}
	}
	offset := (page - 1) * size

	// Base query
	db := ctl.DB.WithContext(c.Context()).
		Model(&attendanceModel.ClassAttendanceSessionParticipantModel{}).
		Where("class_attendance_session_participant_deleted_at IS NULL").
		Where("class_attendance_session_participant_school_id = ?", schoolID)

	// ---- Filter by enums CSV ----
	if len(q.StateIn) > 0 {
		db = db.Where("class_attendance_session_participant_state IN ?", q.StateIn)
	}
	if len(q.MethodIn) > 0 {
		db = db.Where("class_attendance_session_participant_method IN ?", q.MethodIn)
	}
	if len(q.KindIn) > 0 {
		db = db.Where("class_attendance_session_participant_kind IN ?", q.KindIn)
	}

	// ---- Filter by IDs (string → uuid) ----
	if idStr := strings.TrimSpace(q.SessionID); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			db = db.Where("class_attendance_session_participant_session_id = ?", id)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "session_id tidak valid")
		}
	}
	if idStr := strings.TrimSpace(q.SchoolStudentID); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			db = db.Where("class_attendance_session_participant_school_student_id = ?", id)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_student_id tidak valid")
		}
	}
	if idStr := strings.TrimSpace(q.SchoolTeacherID); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			db = db.Where("class_attendance_session_participant_school_teacher_id = ?", id)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_teacher_id tidak valid")
		}
	}
	if idStr := strings.TrimSpace(q.TypeID); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			db = db.Where("class_attendance_session_participant_type_id = ?", id)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "type_id tidak valid")
		}
	}
	if idStr := strings.TrimSpace(q.MarkedByTID); idStr != "" {
		if id, err := uuid.Parse(idStr); err == nil {
			db = db.Where("class_attendance_session_participant_marked_by_teacher_id = ?", id)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "marked_by_teacher_id tidak valid")
		}
	}

	// ---- Filter by created_at range ----
	if v := strings.TrimSpace(q.CreatedGE); v != "" {
		if t, err := parseDateOrDateTime(v, true); err == nil {
			db = db.Where("class_attendance_session_participant_created_at >= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "created_ge tidak valid")
		}
	}
	if v := strings.TrimSpace(q.CreatedLE); v != "" {
		if t, err := parseDateOrDateTime(v, false); err == nil {
			db = db.Where("class_attendance_session_participant_created_at <= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "created_le tidak valid")
		}
	}

	// ---- Filter by marked_at range ----
	if v := strings.TrimSpace(q.MarkedGE); v != "" {
		if t, err := parseDateOrDateTime(v, true); err == nil {
			db = db.Where("class_attendance_session_participant_marked_at >= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "marked_ge tidak valid")
		}
	}
	if v := strings.TrimSpace(q.MarkedLE); v != "" {
		if t, err := parseDateOrDateTime(v, false); err == nil {
			db = db.Where("class_attendance_session_participant_marked_at <= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "marked_le tidak valid")
		}
	}

	// ---- Search (desc / notes) ----
	if len(q.Search) > 0 {
		for _, term := range q.Search {
			term = strings.TrimSpace(term)
			if term == "" {
				continue
			}
			pattern := "%" + strings.ToLower(term) + "%"
			db = db.Where(`
				LOWER(class_attendance_session_participant_desc) LIKE ?
				OR LOWER(class_attendance_session_participant_user_note) LIKE ?
				OR LOWER(class_attendance_session_participant_teacher_note) LIKE ?
			`, pattern, pattern, pattern)
		}
	}

	// ---- Count ----
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ---- Fetch data ----
	var rows []attendanceModel.ClassAttendanceSessionParticipantModel
	if err := db.
		Order("class_attendance_session_participant_created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Daftar peserta session absensi",
		"data":    rows,
		"pagination": fiber.Map{
			"page":  page,
			"size":  size,
			"total": total,
		},
	})
}

// parse "2025-11-16" (pakai dateLayout) atau full RFC3339.
// isStart=true → kalau cuma tanggal → jam 00:00:00
// isStart=false → kalau cuma tanggal → jam 23:59:59
func parseDateOrDateTime(s string, isStart bool) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty")
	}

	// Coba RFC3339 dulu
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Coba layout tanggal saja (YYYY-MM-DD)
	if d, err := time.Parse(dateLayout, s); err == nil {
		if isStart {
			return d, nil
		}
		// akhir hari
		return d.Add(23*time.Hour + 59*time.Minute + 59*time.Second), nil
	}

	return time.Time{}, errors.New("invalid time format")
}
