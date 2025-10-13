// file: internals/features/system/holidays/controller/national_holiday_controller.go
package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	d "masjidku_backend/internals/features/school/classes/class_schedules/dto"
	m "masjidku_backend/internals/features/school/classes/class_schedules/model"
)

/* =========================
   Controller & Constructor
   ========================= */

type NationalHolidayController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewNationHoliday(db *gorm.DB, v *validator.Validate) *NationalHolidayController {
	return &NationalHolidayController{DB: db, Validate: v}
}

/* =========================
   Small helpers
   ========================= */

// parse YYYY-MM-DD → time.Time UTC midnight
func parseDateYYYYMMDD(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
}

/* =========================
   Create  (OWNER ONLY)
   ========================= */

func (ctl *NationalHolidayController) Create(c *fiber.Ctx) error {
	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, http.StatusForbidden, "Hanya owner yang diizinkan")
	}

	var req d.NationalHolidayCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	model, err := req.ToModel()
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&model).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonCreated(c, "Holiday created", d.FromModelNationHoliday(model))
}

/* =========================
   Patch  (OWNER ONLY)
   ========================= */

func (ctl *NationalHolidayController) Patch(c *fiber.Ctx) error {
	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, http.StatusForbidden, "Hanya owner yang diizinkan")
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.NationalHolidayModel
	if err := ctl.DB.
		Where("national_holiday_id = ? AND national_holiday_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "holiday not found")
		}
		return writePGError(c, err)
	}

	var req d.NationalHolidayUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	if err := req.Apply(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "Holiday updated", d.FromModelNationHoliday(existing))
}

/* =========================
   Delete (soft)  (OWNER ONLY)
   ========================= */

func (ctl *NationalHolidayController) Delete(c *fiber.Ctx) error {
	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, http.StatusForbidden, "Hanya owner yang diizinkan")
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.NationalHolidayModel
	if err := ctl.DB.
		Where("national_holiday_id = ? AND national_holiday_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "holiday not found")
		}
		return writePGError(c, err)
	}

	// Soft delete by GORM
	if err := ctl.DB.WithContext(c.Context()).Delete(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonOK(c, "Holiday deleted", fiber.Map{"national_holiday_id": id})
}

/* =========================
   Get By ID  (PUBLIC)
   ========================= */

func (ctl *NationalHolidayController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	withDeleted := strings.TrimSpace(c.Query("with_deleted")) == "true"

	q := ctl.DB.WithContext(c.Context()).Model(&m.NationalHolidayModel{})
	if withDeleted {
		q = q.Unscoped()
	}

	var row m.NationalHolidayModel
	where := "national_holiday_id = ?"
	if !withDeleted {
		where += " AND national_holiday_deleted_at IS NULL"
	}
	if err := q.Where(where, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "holiday not found")
		}
		return writePGError(c, err)
	}

	return helper.JsonOK(c, "OK", d.FromModelNationHoliday(row))
}

/* =========================
   List (index)  (PUBLIC)
   ========================= */

func (ctl *NationalHolidayController) List(c *fiber.Ctx) error {
	var q d.NationalHolidayListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(q); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	tx := ctl.DB.WithContext(c.Context()).Model(&m.NationalHolidayModel{})

	// with_deleted
	if q.WithDeleted != nil && *q.WithDeleted {
		tx = tx.Unscoped()
	} else {
		tx = tx.Where("national_holiday_deleted_at IS NULL")
	}

	// is_active
	if q.IsActive != nil {
		tx = tx.Where("national_holiday_is_active = ?", *q.IsActive)
	}

	// is_recurring
	if q.IsRecurring != nil {
		tx = tx.Where("national_holiday_is_recurring_yearly = ?", *q.IsRecurring)
	}

	// q search (slug/title)
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where(`
			LOWER(COALESCE(national_holiday_slug, '')) LIKE ? OR
			LOWER(national_holiday_title) LIKE ?
		`, kw, kw)
	}

	// date overlap filter
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
		// overlap: end >= from AND start <= to
		tx = tx.Where("national_holiday_end_date >= ? AND national_holiday_start_date <= ?", *dateFrom, *dateTo)
	} else if dateFrom != nil {
		tx = tx.Where("national_holiday_end_date >= ?", *dateFrom)
	} else if dateTo != nil {
		tx = tx.Where("national_holiday_start_date <= ?", *dateTo)
	}

	// sorting
	order := "national_holiday_created_at DESC"
	if q.Sort != nil {
		switch *q.Sort {
		case "start_date_asc":
			order = "national_holiday_start_date ASC, national_holiday_end_date ASC"
		case "start_date_desc":
			order = "national_holiday_start_date DESC, national_holiday_end_date DESC"
		case "end_date_asc":
			order = "national_holiday_end_date ASC"
		case "end_date_desc":
			order = "national_holiday_end_date DESC"
		case "created_at_asc":
			order = "national_holiday_created_at ASC"
		case "created_at_desc":
			order = "national_holiday_created_at DESC"
		case "updated_at_asc":
			order = "national_holiday_updated_at ASC"
		case "updated_at_desc":
			order = "national_holiday_updated_at DESC"
		}
	}
	tx = tx.Order(order)

	// pagination
	limit := 20
	offset := 0
	if q.Limit != nil {
		limit = *q.Limit
	}
	if q.Offset != nil {
		offset = *q.Offset
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return writePGError(c, err)
	}

	var rows []m.NationalHolidayModel
	if err := tx.Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return writePGError(c, err)
	}

	resp := d.NationalHolidayListResponse{
		Items: d.FromModelNationHolidays(rows),
		Pagination: d.Pagination{
			Limit:  limit,
			Offset: offset,
			Total:  int(total),
		},
	}
	return helper.JsonOK(c, "OK", resp)
}
