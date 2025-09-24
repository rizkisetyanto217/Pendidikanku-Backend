// file: internals/features/school/sessions/events/controller/class_events_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	d "masjidku_backend/internals/features/school/others/events/dto"
	m "masjidku_backend/internals/features/school/others/events/model"
)

/* =========================
   Controller & Constructor
   ========================= */

type ClassEventsController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewClassEvents(db *gorm.DB, v *validator.Validate) *ClassEventsController {
	return &ClassEventsController{DB: db, Validate: v}
}

/* =========================
   Small helpers
   ========================= */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := strings.TrimSpace(c.Params(name))
	if idStr == "" {
		return uuid.Nil, fmt.Errorf("%s is required", name)
	}
	return uuid.Parse(idStr)
}

// parse YYYY-MM-DD → time.Time (UTC midnight)
func parseDateYYYYMMDD(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
}

// --- PG error mapping ---
type pgSQLErr interface {
	SQLState() string
	Error() string
}

func mapPGError(err error) (int, string) {
	// 23505 unique_violation, 23503 foreign_key_violation
	var pgErr pgSQLErr
	if errors.As(err, &pgErr) {
		switch pgErr.SQLState() {
		case "23505":
			return http.StatusConflict, "Data duplikat (unique violation)."
		case "23503":
			return http.StatusBadRequest, "Referensi tidak ditemukan (FK violation)."
		}
	}
	return http.StatusInternalServerError, err.Error()
}

func writePGError(c *fiber.Ctx, err error) error {
	code, msg := mapPGError(err)
	return helper.JsonError(c, code, msg)
}

/* =========================
   Create  (OWNER or DKM/Admin Masjid)
   Context via resolver (path/header/query/host/token)
   ========================= */

func (ctl *ClassEventsController) Create(c *fiber.Ctx) error {
	// sediakan DB untuk resolver slug→id
	c.Locals("DB", ctl.DB)

	// resolve masjid context
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}

	// tentukan masjid aktif
	var actMasjidID uuid.UUID
	if helperAuth.IsOwner(c) {
		// owner: tidak perlu cek role, tapi tetap perlu id masjid
		if mc.ID != uuid.Nil {
			actMasjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			actMasjidID = id
		} else {
			return helperAuth.ErrMasjidContextMissing
		}
	} else {
		// non-owner: wajib DKM/Admin
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actMasjidID = id
	}

	var req d.CreateClassEventRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	model, err := req.ToModel(actMasjidID)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&model).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonCreated(c, "Class event created", d.FromModelClassEvent(model))
}

/* =========================
   Patch  (OWNER or DKM/Admin Masjid)
   ========================= */

func (ctl *ClassEventsController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}

	var actMasjidID uuid.UUID
	if helperAuth.IsOwner(c) {
		if mc.ID != uuid.Nil {
			actMasjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			actMasjidID = id
		} else {
			return helperAuth.ErrMasjidContextMissing
		}
	} else {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actMasjidID = id
	}

	eventID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassEventModel
	if err := ctl.DB.
		Where("class_events_id = ? AND class_events_masjid_id = ? AND class_events_deleted_at IS NULL", eventID, actMasjidID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "class event not found")
		}
		return writePGError(c, err)
	}

	var req d.PatchClassEventRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.Apply(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "Class event updated", d.FromModelClassEvent(existing))
}

/* =========================
   Delete (soft)  (OWNER or DKM/Admin Masjid)
   ========================= */

func (ctl *ClassEventsController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}

	var actMasjidID uuid.UUID
	if helperAuth.IsOwner(c) {
		if mc.ID != uuid.Nil {
			actMasjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			actMasjidID = id
		} else {
			return helperAuth.ErrMasjidContextMissing
		}
	} else {
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return er
		}
		actMasjidID = id
	}

	eventID, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassEventModel
	if err := ctl.DB.
		Where("class_events_id = ? AND class_events_masjid_id = ? AND class_events_deleted_at IS NULL", eventID, actMasjidID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "class event not found")
		}
		return writePGError(c, err)
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonOK(c, "Class event deleted", fiber.Map{"class_events_id": eventID})
}

/* =========================
   List (PUBLIC)
   ========================= */

func (ctl *ClassEventsController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	// public: resolve ID saja
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if strings.TrimSpace(mc.Slug) != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
		if er != nil {
			return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	} else {
		return helperAuth.ErrMasjidContextMissing
	}

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

	tx := ctl.DB.WithContext(c.Context()).Model(&m.ClassEventModel{}).
		Where("class_events_masjid_id = ?", masjidID).
		Where("class_events_deleted_at IS NULL")

	// only_active
	if q.OnlyActive != nil && *q.OnlyActive {
		tx = tx.Where("class_events_is_active = TRUE")
	}

	// refs
	if q.ThemeID != nil {
		tx = tx.Where("class_events_theme_id = ?", *q.ThemeID)
	}
	if q.ScheduleID != nil {
		tx = tx.Where("class_events_schedule_id = ?", *q.ScheduleID)
	}
	if q.SectionID != nil {
		tx = tx.Where("class_events_section_id = ?", *q.SectionID)
	}
	if q.ClassID != nil {
		tx = tx.Where("class_events_class_id = ?", *q.ClassID)
	}
	if q.ClassSubjectID != nil {
		tx = tx.Where("class_events_class_subject_id = ?", *q.ClassSubjectID)
	}
	if q.RoomID != nil {
		tx = tx.Where("class_events_room_id = ?", *q.RoomID)
	}
	if q.TeacherID != nil {
		tx = tx.Where("class_events_teacher_id = ?", *q.TeacherID)
	}

	// delivery mode & enrollment policy
	if q.DeliveryMode != nil && strings.TrimSpace(*q.DeliveryMode) != "" {
		tx = tx.Where("class_events_delivery_mode = ?", strings.ToLower(strings.TrimSpace(*q.DeliveryMode)))
	}
	if q.EnrollmentPolicy != nil && strings.TrimSpace(*q.EnrollmentPolicy) != "" {
		tx = tx.Where("class_events_enrollment_policy = ?", strings.ToLower(strings.TrimSpace(*q.EnrollmentPolicy)))
	}

	// search q (title/desc/teacher_name)
	if q.Q != nil {
		kw := "%" + strings.ToLower(*q.Q) + "%"
		tx = tx.Where(`
			LOWER(class_events_title) LIKE ? OR
			LOWER(COALESCE(class_events_desc, '')) LIKE ? OR
			LOWER(COALESCE(class_events_teacher_name, '')) LIKE ?
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
		// overlap: COALESCE(end,date) >= from AND date <= to
		tx = tx.Where("COALESCE(class_events_end_date, class_events_date) >= ? AND class_events_date <= ?", *dateFrom, *dateTo)
	} else if dateFrom != nil {
		tx = tx.Where("COALESCE(class_events_end_date, class_events_date) >= ?", *dateFrom)
	} else if dateTo != nil {
		tx = tx.Where("class_events_date <= ?", *dateTo)
	}

	// sorting
	order := "class_events_date DESC, class_events_start_time ASC NULLS FIRST, class_events_title ASC"
	if q.Sort != nil {
		switch *q.Sort {
		case "date_asc":
			order = "class_events_date ASC, class_events_start_time ASC NULLS FIRST, class_events_title ASC"
		case "date_desc":
			order = "class_events_date DESC, class_events_start_time ASC NULLS FIRST, class_events_title ASC"
		case "start_time_asc":
			order = "class_events_start_time ASC NULLS FIRST, class_events_date ASC"
		case "start_time_desc":
			order = "class_events_start_time DESC NULLS LAST, class_events_date DESC"
		case "created_at_asc":
			order = "class_events_created_at ASC"
		case "created_at_desc":
			order = "class_events_created_at DESC"
		case "updated_at_asc":
			order = "class_events_updated_at ASC"
		case "updated_at_desc":
			order = "class_events_updated_at DESC"
		case "title_asc":
			order = "class_events_title ASC"
		case "title_desc":
			order = "class_events_title DESC"
		}
	}
	tx = tx.Order(order)

	// count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return writePGError(c, err)
	}

	// data
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	var rows []m.ClassEventModel
	if err := tx.Limit(q.Limit).Offset(q.Offset).Find(&rows).Error; err != nil {
		return writePGError(c, err)
	}

	resp := d.ClassEventListResponse{
		Items: d.FromModelsClassEvent(rows),
	}
	resp.Pagination.Limit = q.Limit
	resp.Pagination.Offset = q.Offset
	resp.Pagination.Total = int(total)

	return helper.JsonOK(c, "OK", resp)
}
