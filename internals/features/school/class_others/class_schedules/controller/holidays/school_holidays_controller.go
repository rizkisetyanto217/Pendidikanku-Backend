// file: internals/features/school/holidays/controller/school_holiday_controller.go
package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	d "madinahsalam_backend/internals/features/school/class_others/class_schedules/dto"
	m "madinahsalam_backend/internals/features/school/class_others/class_schedules/model" // mengikuti DTO kamu
)

/* =========================
   Controller & Constructor
   ========================= */

type SchoolHolidayController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewSchoolHoliday(db *gorm.DB, v *validator.Validate) *SchoolHolidayController {
	return &SchoolHolidayController{DB: db, Validate: v}
}

/* =========================
   Small helpers
   ========================= */

// (asumsikan parseDateYYYYMMDD & writePGError sudah ada di file lain / helper yang sama package)

/* =========================
   Create  (DKM/Admin School ONLY)
   Path: POST /:school_id/holidays/school
   ========================= */

func (ctl *SchoolHolidayController) Create(c *fiber.Ctx) error {
	schoolID, err := parseUUIDParam(c, "school_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// 🔐 Hanya DKM/Admin dari school ini
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}

	var req d.CreateSchoolHolidayRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	model, err := req.ToModel(schoolID)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Create(model).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonCreated(c, "School holiday created", d.FromModelSchoolHoliday(model))
}

/* =========================
   Patch  (DKM/Admin School ONLY)
   Path: PATCH /:school_id/holidays/school/:id
   ========================= */

func (ctl *SchoolHolidayController) Patch(c *fiber.Ctx) error {
	schoolID, err := parseUUIDParam(c, "school_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// 🔐 Hanya DKM/Admin dari school ini
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.SchoolHoliday
	if err := ctl.DB.
		Where("school_holiday_id = ? AND school_holiday_school_id = ? AND school_holiday_deleted_at IS NULL", id, schoolID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "school holiday not found")
		}
		return writePGError(c, err)
	}

	var req d.PatchSchoolHolidayRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	// DTO patch tidak pakai validator tag; skip/opsional

	if err := req.Apply(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "School holiday updated", d.FromModelSchoolHoliday(&existing))
}

/* =========================
   Delete (soft)  (DKM/Admin School ONLY)
   Path: DELETE /:school_id/holidays/school/:id
   ========================= */

func (ctl *SchoolHolidayController) Delete(c *fiber.Ctx) error {
	schoolID, err := parseUUIDParam(c, "school_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// 🔐 Hanya DKM/Admin dari school ini
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.SchoolHoliday
	if err := ctl.DB.
		Where("school_holiday_id = ? AND school_holiday_school_id = ? AND school_holiday_deleted_at IS NULL", id, schoolID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "school holiday not found")
		}
		return writePGError(c, err)
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonOK(c, "School holiday deleted", fiber.Map{"school_holiday_id": id})
}

/* =========================
   Get By ID  (PUBLIC)
   Path: GET /:school_id/holidays/school/:id
   ========================= */

func (ctl *SchoolHolidayController) GetByID(c *fiber.Ctx) error {
	schoolID, err := parseUUIDParam(c, "school_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var row m.SchoolHoliday
	if err := ctl.DB.
		Where("school_holiday_id = ? AND school_holiday_school_id = ? AND school_holiday_deleted_at IS NULL", id, schoolID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "school holiday not found")
		}
		return writePGError(c, err)
	}

	return helper.JsonOK(c, "OK", d.FromModelSchoolHoliday(&row))
}

/* =========================
   List (index)  (PUBLIC)
   Path: GET /:school_id/holidays/school
   Query: ?date_from&date_to&only_active&q&limit&offset
   ========================= */

func (ctl *SchoolHolidayController) List(c *fiber.Ctx) error {
	schoolID, err := parseUUIDParam(c, "school_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var q d.ListSchoolHolidaysQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	q.Normalize()
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(q); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	tx := ctl.DB.WithContext(c.Context()).Model(&m.SchoolHoliday{}).
		Where("school_holiday_school_id = ?", schoolID).
		Where("school_holiday_deleted_at IS NULL")

	// only_active
	if q.OnlyActive != nil && *q.OnlyActive {
		tx = tx.Where("school_holiday_is_active = TRUE")
	}

	// search q (slug/title)
	if q.Q != nil {
		kw := "%" + strings.ToLower(*q.Q) + "%"
		tx = tx.Where(`
			LOWER(COALESCE(school_holiday_slug, '')) LIKE ? OR
			LOWER(school_holiday_title) LIKE ?
		`, kw, kw)
	}

	// date overlap
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
		tx = tx.Where("school_holiday_end_date >= ? AND school_holiday_start_date <= ?", *dateFrom, *dateTo)
	} else if dateFrom != nil {
		tx = tx.Where("school_holiday_end_date >= ?", *dateFrom)
	} else if dateTo != nil {
		tx = tx.Where("school_holiday_start_date <= ?", *dateTo)
	}

	// default sort: created_at desc
	tx = tx.Order("school_holiday_created_at DESC")

	// total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return writePGError(c, err)
	}

	// data
	var rows []m.SchoolHoliday
	if err := tx.Limit(q.Limit).Offset(q.Offset).Find(&rows).Error; err != nil {
		return writePGError(c, err)
	}

	resp := d.SchoolHolidayListResponse{
		Data: make([]*d.SchoolHolidayResponse, 0, len(rows)),
	}
	resp.Pagination.Limit = q.Limit
	resp.Pagination.Offset = q.Offset
	resp.Pagination.Total = int(total)

	for i := range rows {
		resp.Data = append(resp.Data, d.FromModelSchoolHoliday(&rows[i]))
	}

	return helper.JsonOK(c, "OK", resp)
}
