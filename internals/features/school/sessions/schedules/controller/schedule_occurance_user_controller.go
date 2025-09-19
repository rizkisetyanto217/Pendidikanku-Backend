package controller

import (
	"database/sql"
	d "masjidku_backend/internals/features/school/sessions/schedules/dto"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (ctl *ClassScheduleController) ListOccurrences(c *fiber.Ctx) error {
	// 🔁 masjid-context: siapkan DB untuk resolver slug→ID
	c.Locals("DB", ctl.DB)

	// 🔐 akses role
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

	// 🎯 tentukan masjid_id aktif:
	//    - kalau ada masjid context (path/header/cookie/query/host) ⇒ wajib DKM pada masjid tsb
	//    - kalau tidak ada ⇒ fallback ke token (admin/teacher)
	var masjidID uuid.UUID
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		masjidID = id
	} else {
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak ditemukan")
		}
		masjidID = id
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

	// Query occurrences (generate_series per hari)
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
