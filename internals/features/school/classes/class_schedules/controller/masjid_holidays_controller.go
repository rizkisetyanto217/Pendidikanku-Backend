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

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	d "masjidku_backend/internals/features/school/classes/class_schedules/dto"
	m "masjidku_backend/internals/features/school/classes/class_schedules/model" // mengikuti DTO kamu
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

/* =========================
   Create  (OWNER or DKM/Admin Masjid)
   Path: POST /:masjid_id/holidays/school
   ========================= */

func (ctl *SchoolHolidayController) Create(c *fiber.Ctx) error {
	masjidID, err := parseUUIDParam(c, "masjid_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	// owner bypass, selain itu wajib DKM/Admin masjid tsb
	if !helperAuth.IsOwner(c) {
		if er := helperAuth.EnsureDKMMasjid(c, masjidID); er != nil {
			return er
		}
	}

	var req d.CreateMasjidHolidayRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(req); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	model, err := req.ToModel(masjidID)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Create(model).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonCreated(c, "School holiday created", d.FromModelMasjidHoliday(model))
}

/* =========================
   Patch  (OWNER or DKM/Admin Masjid)
   Path: PATCH /:masjid_id/holidays/school/:id
   ========================= */

func (ctl *SchoolHolidayController) Patch(c *fiber.Ctx) error {
	masjidID, err := parseUUIDParam(c, "masjid_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if !helperAuth.IsOwner(c) {
		if er := helperAuth.EnsureDKMMasjid(c, masjidID); er != nil {
			return er
		}
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.MasjidHoliday
	if err := ctl.DB.
		Where("masjid_holiday_id = ? AND masjid_holiday_masjid_id = ? AND masjid_holiday_deleted_at IS NULL", id, masjidID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "school holiday not found")
		}
		return writePGError(c, err)
	}

	var req d.PatchMasjidHolidayRequest
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

	return helper.JsonUpdated(c, "School holiday updated", d.FromModelMasjidHoliday(&existing))
}

/* =========================
   Delete (soft)  (OWNER or DKM/Admin Masjid)
   Path: DELETE /:masjid_id/holidays/school/:id
   ========================= */

func (ctl *SchoolHolidayController) Delete(c *fiber.Ctx) error {
	masjidID, err := parseUUIDParam(c, "masjid_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if !helperAuth.IsOwner(c) {
		if er := helperAuth.EnsureDKMMasjid(c, masjidID); er != nil {
			return er
		}
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.MasjidHoliday
	if err := ctl.DB.
		Where("masjid_holiday_id = ? AND masjid_holiday_masjid_id = ? AND masjid_holiday_deleted_at IS NULL", id, masjidID).
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
   Path: GET /:masjid_id/holidays/school/:id
   ========================= */

func (ctl *SchoolHolidayController) GetByID(c *fiber.Ctx) error {
	masjidID, err := parseUUIDParam(c, "masjid_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var row m.MasjidHoliday
	if err := ctl.DB.
		Where("masjid_holiday_id = ? AND masjid_holiday_masjid_id = ? AND masjid_holiday_deleted_at IS NULL", id, masjidID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "school holiday not found")
		}
		return writePGError(c, err)
	}

	return helper.JsonOK(c, "OK", d.FromModelMasjidHoliday(&row))
}

/* =========================
   List (index)  (PUBLIC)
   Path: GET /:masjid_id/holidays/school
   Query: ?date_from&date_to&only_active&q&limit&offset
   ========================= */

func (ctl *SchoolHolidayController) List(c *fiber.Ctx) error {
	masjidID, err := parseUUIDParam(c, "masjid_id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var q d.ListMasjidHolidaysQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	q.Normalize()
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(q); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, err.Error())
		}
	}

	tx := ctl.DB.WithContext(c.Context()).Model(&m.MasjidHoliday{}).
		Where("masjid_holiday_masjid_id = ?", masjidID).
		Where("masjid_holiday_deleted_at IS NULL")

	// only_active
	if q.OnlyActive != nil && *q.OnlyActive {
		tx = tx.Where("masjid_holiday_is_active = TRUE")
	}

	// search q (slug/title)
	if q.Q != nil {
		kw := "%" + strings.ToLower(*q.Q) + "%"
		tx = tx.Where(`
			LOWER(COALESCE(masjid_holiday_slug, '')) LIKE ? OR
			LOWER(masjid_holiday_title) LIKE ?
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
		tx = tx.Where("masjid_holiday_end_date >= ? AND masjid_holiday_start_date <= ?", *dateFrom, *dateTo)
	} else if dateFrom != nil {
		tx = tx.Where("masjid_holiday_end_date >= ?", *dateFrom)
	} else if dateTo != nil {
		tx = tx.Where("masjid_holiday_start_date <= ?", *dateTo)
	}

	// default sort: created_at desc
	tx = tx.Order("masjid_holiday_created_at DESC")

	// total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return writePGError(c, err)
	}

	// data
	var rows []m.MasjidHoliday
	if err := tx.Limit(q.Limit).Offset(q.Offset).Find(&rows).Error; err != nil {
		return writePGError(c, err)
	}

	resp := d.MasjidHolidayListResponse{
		Data: make([]*d.MasjidHolidayResponse, 0, len(rows)),
	}
	resp.Pagination.Limit = q.Limit
	resp.Pagination.Offset = q.Offset
	resp.Pagination.Total = int(total)

	for i := range rows {
		resp.Data = append(resp.Data, d.FromModelMasjidHoliday(&rows[i]))
	}

	return helper.JsonOK(c, "OK", resp)
}
