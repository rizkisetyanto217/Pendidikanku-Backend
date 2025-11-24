// file: internals/features/school/classes/class_events/controller/class_events_controller.go
package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	d "madinahsalam_backend/internals/features/school/classes/class_events/dto"
	m "madinahsalam_backend/internals/features/school/classes/class_events/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/* =========================
   List (PUBLIC)
   ========================= */

func (ctl *ClassEventsController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}

	// resolve school ID (public)
	var schoolID uuid.UUID
	if mc.ID != uuid.Nil {
		schoolID = mc.ID
	} else if strings.TrimSpace(mc.Slug) != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
		if er != nil {
			return fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		schoolID = id
	} else {
		return helperAuth.ErrSchoolContextMissing
	}

	// parse filter DTO
	var q d.ListClassEventsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	q.Normalize()
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(q); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	// pagination & sorting via helper (default: date desc)
	p := helper.ParseFiber(c, "date", "desc", helper.DefaultOpts)

	tx := ctl.DB.WithContext(c.Context()).
		Model(&m.ClassEventModel{}).
		Where("class_event_school_id = ? AND class_event_deleted_at IS NULL", schoolID)

	// only_active
	if q.OnlyActive != nil && *q.OnlyActive {
		tx = tx.Where("class_event_is_active = TRUE")
	}

	// refs
	if q.ThemeID != nil {
		tx = tx.Where("class_event_theme_id = ?", *q.ThemeID)
	}
	if q.ScheduleID != nil {
		tx = tx.Where("class_event_schedule_id = ?", *q.ScheduleID)
	}
	if q.SectionID != nil {
		tx = tx.Where("class_event_section_id = ?", *q.SectionID)
	}
	if q.ClassID != nil {
		tx = tx.Where("class_event_class_id = ?", *q.ClassID)
	}
	if q.ClassSubjectID != nil {
		tx = tx.Where("class_event_class_subject_id = ?", *q.ClassSubjectID)
	}
	if q.RoomID != nil {
		tx = tx.Where("class_event_room_id = ?", *q.RoomID)
	}
	if q.TeacherID != nil {
		tx = tx.Where("class_event_teacher_id = ?", *q.TeacherID)
	}

	// delivery mode & enrollment policy
	if q.DeliveryMode != nil && strings.TrimSpace(*q.DeliveryMode) != "" {
		tx = tx.Where("class_event_delivery_mode = ?", strings.ToLower(strings.TrimSpace(*q.DeliveryMode)))
	}
	if q.EnrollmentPolicy != nil && strings.TrimSpace(*q.EnrollmentPolicy) != "" {
		tx = tx.Where("class_event_enrollment_policy = ?", strings.ToLower(strings.TrimSpace(*q.EnrollmentPolicy)))
	}

	// search q (title/desc/teacher_name)
	if q.Q != nil {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where(`
			LOWER(class_event_title) LIKE ? OR
			LOWER(COALESCE(class_event_desc, '')) LIKE ? OR
			LOWER(COALESCE(class_event_teacher_name, '')) LIKE ?
		`, kw, kw, kw)
	}

	// date overlap: gunakan COALESCE(end_date, date)
	var dateFrom, dateTo *time.Time
	if q.DateFrom != nil && strings.TrimSpace(*q.DateFrom) != "" {
		if t, ok := parseDateYYYYMMDD(*q.DateFrom); ok {
			dateFrom = &t
		} else {
			return helper.JsonError(c, http.StatusBadRequest, "invalid date_from (YYYY-MM-DD)")
		}
	}
	if q.DateTo != nil && strings.TrimSpace(*q.DateTo) != "" {
		if t, ok := parseDateYYYYMMDD(*q.DateTo); ok {
			dateTo = &t
		} else {
			return helper.JsonError(c, http.StatusBadRequest, "invalid date_to (YYYY-MM-DD)")
		}
	}
	if dateFrom != nil && dateTo != nil {
		tx = tx.Where("COALESCE(class_event_end_date, class_event_date) >= ? AND class_event_date <= ?", *dateFrom, *dateTo)
	} else if dateFrom != nil {
		tx = tx.Where("COALESCE(class_event_end_date, class_event_date) >= ?", *dateFrom)
	} else if dateTo != nil {
		tx = tx.Where("class_event_date <= ?", *dateTo)
	}

	// sorting
	orderExpr := "class_event_date DESC, class_event_start_time ASC NULLS FIRST, class_event_title ASC"
	if q.Sort != nil && strings.TrimSpace(*q.Sort) != "" {
		switch strings.TrimSpace(*q.Sort) {
		case "date_asc":
			orderExpr = "class_event_date ASC, class_event_start_time ASC NULLS FIRST, class_event_title ASC"
		case "date_desc":
			orderExpr = "class_event_date DESC, class_event_start_time ASC NULLS FIRST, class_event_title ASC"
		case "start_time_asc":
			orderExpr = "class_event_start_time ASC NULLS FIRST, class_event_date ASC"
		case "start_time_desc":
			orderExpr = "class_event_start_time DESC NULLS LAST, class_event_date DESC"
		case "created_at_asc":
			orderExpr = "class_event_created_at ASC"
		case "created_at_desc":
			orderExpr = "class_event_created_at DESC"
		case "updated_at_asc":
			orderExpr = "class_event_updated_at ASC"
		case "updated_at_desc":
			orderExpr = "class_event_updated_at DESC"
		case "title_asc":
			orderExpr = "class_event_title ASC"
		case "title_desc":
			orderExpr = "class_event_title DESC"
		}
	} else {
		dir := "DESC"
		if strings.ToLower(p.SortOrder) == "asc" {
			dir = "ASC"
		}
		switch strings.ToLower(strings.TrimSpace(p.SortBy)) {
		case "date", "":
			orderExpr = "class_event_date " + dir + ", class_event_start_time ASC NULLS FIRST, class_event_title ASC"
		case "start_time":
			orderExpr = "class_event_start_time " + dir + " NULLS FIRST, class_event_date " + dir
		case "created_at":
			orderExpr = "class_event_created_at " + dir
		case "updated_at":
			orderExpr = "class_event_updated_at " + dir
		case "title":
			orderExpr = "class_event_title " + dir
		}
	}
	tx = tx.Order(orderExpr)

	// count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// data
	var rows []m.ClassEventModel
	if err := tx.Limit(p.Limit()).Offset(p.Offset()).Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// pagination seragam
	pg := helper.BuildPaginationFromOffset(total, p.Offset(), p.Limit())

	// response seragam
	return helper.JsonList(c, "ok", d.FromModelsClassEvent(rows), pg)
}
