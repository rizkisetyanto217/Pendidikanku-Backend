// file: internals/features/school/sessions_assesment/schedule_daily/controller/schedule_occurrence_controller.go
package controller

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	d "masjidku_backend/internals/features/school/sessions/schedule/dto"
	m "masjidku_backend/internals/features/school/sessions/schedule/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/*
Usage (contoh):

1) List occurrences (kalender):
   GET  /api/a/class-schedules/occurrences?from=2025-09-01&to=2025-09-30

2) Ensure CAS untuk satu hari:
   POST /api/a/class-schedules/ensure-cas?date=2025-09-06

3) Ensure CAS untuk rentang hari:
   POST /api/a/class-schedules/ensure-cas-range?from=2025-09-01&to=2025-10-15
*/

func parseLocalDate(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, err
	}
	// anchor midnight (local)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
}

// ===== 1) List Occurrences (tanpa materialisasi) =====

// GET /api/a/class-schedules/occurrences?from=YYYY-MM-DD&to=YYYY-MM-DD
type schedOccurRow struct {
	OccurDate time.Time `gorm:"column:occur_date"`
	m.ClassScheduleModel
}

type ScheduleOccurrenceResponse struct {
	OccurDate string                 `json:"occur_date"` // YYYY-MM-DD
	Schedule  d.ClassScheduleResponse `json:"schedule"`
}


func (ctl *ClassScheduleController) ListOccurrences(c *fiber.Ctx) error {
	// 🔐 akses
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidID == uuid.Nil {
		return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak ditemukan")
	}

	// Params
	fromStr := strings.TrimSpace(c.Query("from"))
	toStr := strings.TrimSpace(c.Query("to"))
	if fromStr == "" || toStr == "" {
		return helper.JsonError(c, http.StatusBadRequest, "Param from & to wajib (YYYY-MM-DD)")
	}
	from, err := parseLocalDate(fromStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "from invalid (YYYY-MM-DD)")
	}
	to, err := parseLocalDate(toStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "to invalid (YYYY-MM-DD)")
	}
	if to.Before(from) {
		return helper.JsonError(c, http.StatusBadRequest, "to harus >= from")
	}

	// Optional filter: section_id
	sectionIDStr := strings.TrimSpace(c.Query("section_id"))
	var sectionID uuid.UUID
	hasSection := false
	if sectionIDStr != "" {
		id, e := uuid.Parse(sectionIDStr)
		if e != nil {
			return helper.JsonError(c, http.StatusBadRequest, "section_id invalid")
		}
		sectionID = id
		hasSection = true
	}

	// Query: join ke generate_series via GORM builder
	q := ctl.DB.
		Table("class_schedules AS s").
		Select("days.dt AS occur_date, s.*").
		Joins(`
			JOIN generate_series(?::date, ?::date, interval '1 day') AS days(dt)
				ON s.class_schedules_is_active
			   AND s.class_schedules_deleted_at IS NULL
			   AND days.dt BETWEEN s.class_schedules_start_date AND s.class_schedules_end_date
			   AND EXTRACT(ISODOW FROM days.dt) = s.class_schedules_day_of_week
		`, from, to).
		Where("s.class_schedules_masjid_id = ?", masjidID)

	if hasSection {
		q = q.Where("s.class_schedules_section_id = ?", sectionID)
	}

	var rows []schedOccurRow
	if err := q.
		Order("occur_date, s.class_schedules_start_time").
		Scan(&rows).Error; err != nil {
		if err == gorm.ErrRecordNotFound || err == sql.ErrNoRows {
			return helper.JsonList(c, []any{}, fiber.Map{
				"from":  fromStr,
				"to":    toStr,
				"total": 0,
			})
		}
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// Map response
	out := make([]ScheduleOccurrenceResponse, 0, len(rows))
	for i := range rows {
		out = append(out, ScheduleOccurrenceResponse{
			OccurDate: rows[i].OccurDate.Format("2006-01-02"),
			Schedule:  d.NewClassScheduleResponse(&rows[i].ClassScheduleModel),
		})
	}

	return helper.JsonList(c, out, fiber.Map{
		"from":  from.Format("2006-01-02"),
		"to":    to.Format("2006-01-02"),
		"total": len(out),
	})
}

// ===== 2) Ensure CAS untuk 1 hari (idempotent, non-destructive) =====

// POST /api/a/class-schedules/ensure-cas?date=YYYY-MM-DD
func (ctl *ClassScheduleController) EnsureCASForDate(c *fiber.Ctx) error {
	// 🔐 hanya Admin/DKM/Teacher
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidID == uuid.Nil {
		return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak ditemukan")
	}

	qs := strings.TrimSpace(c.Query("date"))
	var d time.Time
	if qs == "" {
		now := time.Now().In(time.Local)
		d = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	} else {
		d, err = parseLocalDate(qs)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "Param date invalid (YYYY-MM-DD)")
		}
	}

	// INSERT: buat baris CAS dari schedule; abaikan jika sudah ada (avoid need unique name)
	insertSQL := `
	INSERT INTO class_attendance_sessions (
		class_attendance_sessions_id,
		class_attendance_sessions_masjid_id,
		class_attendance_sessions_section_id,
		class_attendance_sessions_class_subject_id,
		class_attendance_sessions_csst_id,
		class_attendance_sessions_teacher_id,
		class_attendance_sessions_class_room_id,
		class_attendance_sessions_date,
		class_attendance_sessions_general_info
	)
	SELECT
		gen_random_uuid(),
		s.class_schedules_masjid_id,
		s.class_schedules_section_id,
		s.class_schedules_class_subject_id,
		s.class_schedules_csst_id,
		s.class_schedules_teacher_id,
		s.class_schedules_room_id,
		@date::date,
		'Generated from schedule'
	FROM class_schedules s
	WHERE s.class_schedules_masjid_id = @masjid
	  AND s.class_schedules_is_active
	  AND s.class_schedules_deleted_at IS NULL
	  AND @date::date BETWEEN s.class_schedules_start_date AND s.class_schedules_end_date
	  AND EXTRACT(ISODOW FROM @date::date) = s.class_schedules_day_of_week
	ON CONFLICT DO NOTHING;
	`

	// UPDATE: sinkronkan field NULL di CAS dari schedule (non-destructive)
	updateSQL := `
	UPDATE class_attendance_sessions cas
	SET
		class_attendance_sessions_csst_id =
			COALESCE(cas.class_attendance_sessions_csst_id, s.class_schedules_csst_id),
		class_attendance_sessions_teacher_id =
			COALESCE(cas.class_attendance_sessions_teacher_id, s.class_schedules_teacher_id),
		class_attendance_sessions_class_room_id =
			COALESCE(cas.class_attendance_sessions_class_room_id, s.class_schedules_room_id)
	FROM class_schedules s
	WHERE cas.class_attendance_sessions_masjid_id = @masjid
	  AND cas.class_attendance_sessions_date = @date::date
	  AND s.class_schedules_masjid_id = cas.class_attendance_sessions_masjid_id
	  AND s.class_schedules_section_id = cas.class_attendance_sessions_section_id
	  AND s.class_schedules_class_subject_id = cas.class_attendance_sessions_class_subject_id
	  AND s.class_schedules_is_active
	  AND s.class_schedules_deleted_at IS NULL
	  AND @date::date BETWEEN s.class_schedules_start_date AND s.class_schedules_end_date
	  AND EXTRACT(ISODOW FROM @date::date) = s.class_schedules_day_of_week;
	`

	if err := ctl.DB.Exec(insertSQL, sql.Named("masjid", masjidID), sql.Named("date", d)).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}
	if err := ctl.DB.Exec(updateSQL, sql.Named("masjid", masjidID), sql.Named("date", d)).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "CAS ensured for date", fiber.Map{
		"date": d.Format("2006-01-02"),
	})
}

// ===== 3) Ensure CAS untuk rentang tanggal (idempotent, non-destructive) =====

// POST /api/a/class-schedules/ensure-cas-range?from=YYYY-MM-DD&to=YYYY-MM-DD
func (ctl *ClassScheduleController) EnsureCASForRange(c *fiber.Ctx) error {
	// 🔐 hanya Admin/DKM/Teacher
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidID == uuid.Nil {
		return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak ditemukan")
	}

	fromStr := strings.TrimSpace(c.Query("from"))
	toStr := strings.TrimSpace(c.Query("to"))
	if fromStr == "" || toStr == "" {
		return helper.JsonError(c, http.StatusBadRequest, "Param from & to wajib (YYYY-MM-DD)")
	}
	from, err := parseLocalDate(fromStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "from invalid (YYYY-MM-DD)")
	}
	to, err := parseLocalDate(toStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "to invalid (YYYY-MM-DD)")
	}
	if to.Before(from) {
		return helper.JsonError(c, http.StatusBadRequest, "to harus >= from")
	}

	// INSERT untuk seluruh hari dalam rentang
	insertSQL := `
	INSERT INTO class_attendance_sessions (
		class_attendance_sessions_id,
		class_attendance_sessions_masjid_id,
		class_attendance_sessions_section_id,
		class_attendance_sessions_class_subject_id,
		class_attendance_sessions_csst_id,
		class_attendance_sessions_teacher_id,
		class_attendance_sessions_class_room_id,
		class_attendance_sessions_date,
		class_attendance_sessions_general_info
	)
	SELECT
		gen_random_uuid(),
		s.class_schedules_masjid_id,
		s.class_schedules_section_id,
		s.class_schedules_class_subject_id,
		s.class_schedules_csst_id,
		s.class_schedules_teacher_id,
		s.class_schedules_room_id,
		d.dt,
		'Generated from schedule'
	FROM class_schedules s
	JOIN LATERAL (
		SELECT dd::date AS dt
		FROM generate_series(@from::date, @to::date, interval '1 day') dd
		WHERE dd::date BETWEEN s.class_schedules_start_date AND s.class_schedules_end_date
		  AND EXTRACT(ISODOW FROM dd) = s.class_schedules_day_of_week
	) d ON true
	WHERE s.class_schedules_masjid_id = @masjid
	  AND s.class_schedules_is_active
	  AND s.class_schedules_deleted_at IS NULL
	ON CONFLICT DO NOTHING;
	`

	// UPDATE: sinkronkan field NULL di CAS dari schedule untuk seluruh rentang
	updateSQL := `
	UPDATE class_attendance_sessions cas
	SET
		class_attendance_sessions_csst_id =
			COALESCE(cas.class_attendance_sessions_csst_id, s.class_schedules_csst_id),
		class_attendance_sessions_teacher_id =
			COALESCE(cas.class_attendance_sessions_teacher_id, s.class_schedules_teacher_id),
		class_attendance_sessions_class_room_id =
			COALESCE(cas.class_attendance_sessions_class_room_id, s.class_schedules_room_id)
	FROM class_schedules s
	WHERE cas.class_attendance_sessions_masjid_id = @masjid
	  AND cas.class_attendance_sessions_date BETWEEN @from::date AND @to::date
	  AND s.class_schedules_masjid_id = cas.class_attendance_sessions_masjid_id
	  AND s.class_schedules_section_id = cas.class_attendance_sessions_section_id
	  AND s.class_schedules_class_subject_id = cas.class_attendance_sessions_class_subject_id
	  AND s.class_schedules_is_active
	  AND s.class_schedules_deleted_at IS NULL
	  AND cas.class_attendance_sessions_date BETWEEN s.class_schedules_start_date AND s.class_schedules_end_date
	  AND EXTRACT(ISODOW FROM cas.class_attendance_sessions_date) = s.class_schedules_day_of_week;
	`

	if err := ctl.DB.Exec(insertSQL,
		sql.Named("masjid", masjidID),
		sql.Named("from", from),
		sql.Named("to", to),
	).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.Exec(updateSQL,
		sql.Named("masjid", masjidID),
		sql.Named("from", from),
		sql.Named("to", to),
	).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "CAS ensured for range", fiber.Map{
		"from": from.Format("2006-01-02"),
		"to":   to.Format("2006-01-02"),
	})
}
