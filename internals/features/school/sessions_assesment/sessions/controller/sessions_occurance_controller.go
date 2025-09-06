// file: internals/features/school/sessions_assesment/occurrences/controller/occurrence_controller.go
package controller

import (
	"fmt"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	// SCHEDULE (pakai path yang konsisten dengan DTO kamu)
	schedDTO "masjidku_backend/internals/features/school/sessions_assesment/schedule_daily/dto"
	schedModel "masjidku_backend/internals/features/school/sessions_assesment/schedule_daily/model"

	// ATTENDANCE
	attendanceDTO "masjidku_backend/internals/features/school/sessions_assesment/sessions/dto"
	attendanceModel "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==============================
// Controller
// ==============================
type OccurrenceController struct {
	DB *gorm.DB
}

func NewOccurrenceController(db *gorm.DB) *OccurrenceController { return &OccurrenceController{DB: db} }

// ==============================
// Helpers
// ==============================
func parseLocalYMD(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty")
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
}

// scan rows
type schedOccurRow struct {
	OccurDate time.Time `gorm:"column:occur_date"`
	schedModel.ClassScheduleModel
}
type casOccurRow struct {
	OccurDate time.Time `gorm:"column:occur_date"`
	attendanceModel.ClassAttendanceSessionModel
}

// responses
type ScheduleOccurrenceResponse struct {
	OccurDate string                        `json:"occur_date"`
	Schedule  schedDTO.ClassScheduleResponse `json:"schedule"`
}
type AttendanceOccurrenceResponse struct {
	OccurDate string                                     `json:"occur_date"`
	Session   attendanceDTO.ClassAttendanceSessionResponse `json:"session"`
}

// =====================================================
// GET /class-schedules/occurrences?from=&to=&section_id=&class_subject_id=&room_id=&teacher_id=&csst_id=
// =====================================================
func (ctl *OccurrenceController) ListScheduleOccurrences(c *fiber.Ctx) error {
	// guard role
	if !(helperAuth.IsAdmin(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak ditemukan")
	}

	// required date range
	fromStr := strings.TrimSpace(c.Query("from"))
	toStr := strings.TrimSpace(c.Query("to"))
	if fromStr == "" || toStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Param from & to wajib (YYYY-MM-DD)")
	}
	from, err := parseLocalYMD(fromStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "from invalid (YYYY-MM-DD)")
	}
	to, err := parseLocalYMD(toStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "to invalid (YYYY-MM-DD)")
	}
	if to.Before(from) {
		return helper.JsonError(c, fiber.StatusBadRequest, "to harus >= from")
	}

	// optional filters
	var (
		sectionID, classSubjectID, roomID, teacherID, csstID *uuid.UUID
	)
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "section_id invalid") }
		sectionID = &id
	}
	if s := strings.TrimSpace(c.Query("class_subject_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "class_subject_id invalid") }
		classSubjectID = &id
	}
	if s := strings.TrimSpace(c.Query("room_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "room_id invalid") }
		roomID = &id
	}
	if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "teacher_id invalid") }
		teacherID = &id
	}
	if s := strings.TrimSpace(c.Query("csst_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "csst_id invalid") }
		csstID = &id
	}

	// build SQL (parameterized)
	sb := strings.Builder{}
	sb.WriteString(`
WITH days AS (
  SELECT d::date AS dt
  FROM generate_series(?::date, ?::date, interval '1 day') d
)
SELECT
  days.dt AS occur_date,
  s.*
FROM days
JOIN class_schedules s
  ON s.class_schedules_is_active
 AND s.class_schedules_deleted_at IS NULL
 AND days.dt BETWEEN s.class_schedules_start_date AND s.class_schedules_end_date
 AND EXTRACT(ISODOW FROM days.dt) = s.class_schedules_day_of_week
WHERE s.class_schedules_masjid_id = ?
`)
	args := []any{from, to, masjidID}

	if sectionID != nil {
		sb.WriteString("  AND s.class_schedules_section_id = ?\n")
		args = append(args, *sectionID)
	}
	if classSubjectID != nil {
		sb.WriteString("  AND s.class_schedules_class_subject_id = ?\n")
		args = append(args, *classSubjectID)
	}
	if roomID != nil {
		sb.WriteString("  AND s.class_schedules_room_id = ?\n")
		args = append(args, *roomID)
	}
	if teacherID != nil {
		sb.WriteString("  AND s.class_schedules_teacher_id = ?\n")
		args = append(args, *teacherID)
	}
	if csstID != nil {
		sb.WriteString("  AND s.class_schedules_csst_id = ?\n")
		args = append(args, *csstID)
	}
	sb.WriteString("ORDER BY days.dt, s.class_schedules_start_time;")

	rawSQL := sb.String()

	var rows []schedOccurRow
	if err := ctl.DB.Raw(rawSQL, args...).Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	out := make([]ScheduleOccurrenceResponse, 0, len(rows))
	for i := range rows {
		out = append(out, ScheduleOccurrenceResponse{
			OccurDate: rows[i].OccurDate.Format("2006-01-02"),
			Schedule:  schedDTO.NewClassScheduleResponse(&rows[i].ClassScheduleModel),
		})
	}

	meta := fiber.Map{
		"from":  from.Format("2006-01-02"),
		"to":    to.Format("2006-01-02"),
		"total": len(out),
	}
	return helper.JsonList(c, out, meta)
}

// =====================================================
// GET /class-attendance-sessions/occurrences?from=&to=&section_id=&class_subject_id=&room_id=&teacher_id=&csst_id=
// =====================================================
func (ctl *OccurrenceController) ListAttendanceOccurrences(c *fiber.Ctx) error {
	// guard role
	if !(helperAuth.IsAdmin(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak ditemukan")
	}

	// required date range
	fromStr := strings.TrimSpace(c.Query("from"))
	toStr := strings.TrimSpace(c.Query("to"))
	if fromStr == "" || toStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Param from & to wajib (YYYY-MM-DD)")
	}
	from, err := parseLocalYMD(fromStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "from invalid (YYYY-MM-DD)")
	}
	to, err := parseLocalYMD(toStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "to invalid (YYYY-MM-DD)")
	}
	if to.Before(from) {
		return helper.JsonError(c, fiber.StatusBadRequest, "to harus >= from")
	}

	// optional filters
	var (
		sectionID, classSubjectID, roomID, teacherID, csstID *uuid.UUID
	)
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "section_id invalid") }
		sectionID = &id
	}
	if s := strings.TrimSpace(c.Query("class_subject_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "class_subject_id invalid") }
		classSubjectID = &id
	}
	if s := strings.TrimSpace(c.Query("room_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "room_id invalid") }
		roomID = &id
	}
	if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "teacher_id invalid") }
		teacherID = &id
	}
	if s := strings.TrimSpace(c.Query("csst_id")); s != "" {
		id, e := uuid.Parse(s); if e != nil { return helper.JsonError(c, fiber.StatusBadRequest, "csst_id invalid") }
		csstID = &id
	}

	q := ctl.DB.
		Table("class_attendance_sessions AS cas").
		Select("cas.*, cas.class_attendance_sessions_date AS occur_date").
		Where("cas.class_attendance_sessions_masjid_id = ?", masjidID).
		Where("cas.class_attendance_sessions_deleted_at IS NULL").
		Where("cas.class_attendance_sessions_date BETWEEN ? AND ?", from, to)

	if sectionID != nil {
		q = q.Where("cas.class_attendance_sessions_section_id = ?", *sectionID)
	}
	if classSubjectID != nil {
		q = q.Where("cas.class_attendance_sessions_class_subject_id = ?", *classSubjectID)
	}
	if roomID != nil {
		q = q.Where("cas.class_attendance_sessions_class_room_id = ?", *roomID)
	}
	if teacherID != nil {
		q = q.Where("cas.class_attendance_sessions_teacher_id = ?", *teacherID)
	}
	if csstID != nil {
		q = q.Where("cas.class_attendance_sessions_csst_id = ?", *csstID)
	}

	q = q.Order("cas.class_attendance_sessions_date, cas.class_attendance_sessions_id")

	var rows []casOccurRow
	if err := q.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]AttendanceOccurrenceResponse, 0, len(rows))
	for i := range rows {
		out = append(out, AttendanceOccurrenceResponse{
			OccurDate: rows[i].OccurDate.Format("2006-01-02"),
			Session:   attendanceDTO.FromClassAttendanceSessionModel(rows[i].ClassAttendanceSessionModel),
		})
	}

	meta := fiber.Map{
		"from":  from.Format("2006-01-02"),
		"to":    to.Format("2006-01-02"),
		"total": len(out),
	}
	return helper.JsonList(c, out, meta)
}
