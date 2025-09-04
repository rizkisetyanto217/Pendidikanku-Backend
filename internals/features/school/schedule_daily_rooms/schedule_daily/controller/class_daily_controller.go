// file: internals/features/school/class_daily/controller/class_daily_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
"net/url" 
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"

	d "masjidku_backend/internals/features/school/schedule_daily_rooms/schedule_daily/dto"
	m "masjidku_backend/internals/features/school/schedule_daily_rooms/schedule_daily/model"
)

/* =========================
   Controller & Constructor
   ========================= */

type ClassDailyController struct {
	DB *gorm.DB
}

func NewClassDailyController(db *gorm.DB) *ClassDailyController {
	return &ClassDailyController{DB: db}
}
// convert Fiber ctx to a minimal *http.Request that only carries the query string
func stdReqFromFiber(c *fiber.Ctx) *http.Request {
	u := &url.URL{
		RawQuery: string(c.Request().URI().QueryString()),
	}
	return &http.Request{URL: u}
}


/* =========================
   Small helpers
   ========================= */

func parseDateParam(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	return time.Parse("2006-01-02", s)
}

/* =========================
   Query: List
   ========================= */

type listQueryDaily struct {
	// Filter
	MasjidID   string `query:"masjid_id"`
	SectionID  string `query:"section_id"`
	ScheduleID string `query:"schedule_id"`
	Active     *bool  `query:"active"`
	DayOfWeek  *int   `query:"dow"`     // 1..7
	OnDate     string `query:"on_date"` // YYYY-MM-DD (exact date)
	From       string `query:"from"`    // YYYY-MM-DD
	To         string `query:"to"`      // YYYY-MM-DD
}

func (ctl *ClassDailyController) List(c *fiber.Ctx) error {
	// Parse pagination & sorting (pakai preset Admin)
	// default sort: date ASC
 	p := helper.ParseWith(stdReqFromFiber(c), "date", "asc", helper.AdminOpts)

	// Whitelist kolom sorting
	allowedSort := map[string]string{
		"date":       "class_daily_date",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}
	orderCol, ok := allowedSort[strings.ToLower(p.SortBy)]
	if !ok {
		orderCol = allowedSort["date"]
	}
	orderDir := "ASC"
	if strings.ToLower(p.SortOrder) == "desc" {
		orderDir = "DESC"
	}

	// Parse filters
	var q listQueryDaily
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Query tidak valid")
	}

	db := ctl.DB.Model(&m.ClassDailyModel{}).Where("deleted_at IS NULL")

	// Masjid scope dari Locals override filter masjid jika ada
	if loc := c.Locals("masjid_id"); loc != nil {
		if s := strings.TrimSpace(fmt.Sprintf("%v", loc)); s != "" {
			q.MasjidID = s
		}
	}

	// Filters
	if q.MasjidID != "" {
		if _, err := uuid.Parse(q.MasjidID); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "masjid_id invalid")
		}
		db = db.Where("class_daily_masjid_id = ?", q.MasjidID)
	}
	if q.SectionID != "" {
		if _, err := uuid.Parse(q.SectionID); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "section_id invalid")
		}
		db = db.Where("class_daily_section_id = ?", q.SectionID)
	}
	if q.ScheduleID != "" {
		if _, err := uuid.Parse(q.ScheduleID); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "schedule_id invalid")
		}
		db = db.Where("class_daily_schedule_id = ?", q.ScheduleID)
	}
	if q.Active != nil {
		db = db.Where("class_daily_is_active = ?", *q.Active)
	}
	if q.DayOfWeek != nil {
		if *q.DayOfWeek < 1 || *q.DayOfWeek > 7 {
			return helper.JsonError(c, http.StatusBadRequest, "dow must be 1..7")
		}
		db = db.Where("class_daily_day_of_week = ?", *q.DayOfWeek)
	}

	// on_date filter (exact)
	if strings.TrimSpace(q.OnDate) != "" {
		dt, err := parseDateParam(q.OnDate)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "on_date invalid (YYYY-MM-DD)")
		}
		db = db.Where("class_daily_date = ?", dt)
	}

	// Range date
	if strings.TrimSpace(q.From) != "" || strings.TrimSpace(q.To) != "" {
		var from, to *time.Time
		if strings.TrimSpace(q.From) != "" {
			dt, err := parseDateParam(q.From)
			if err != nil {
				return helper.JsonError(c, http.StatusBadRequest, "from invalid (YYYY-MM-DD)")
			}
			from = &dt
		}
		if strings.TrimSpace(q.To) != "" {
			dt, err := parseDateParam(q.To)
			if err != nil {
				return helper.JsonError(c, http.StatusBadRequest, "to invalid (YYYY-MM-DD)")
			}
			to = &dt
		}
		if from != nil && to != nil {
			db = db.Where("class_daily_date BETWEEN ? AND ?", *from, *to)
		} else if from != nil {
			db = db.Where("class_daily_date >= ?", *from)
		} else if to != nil {
			db = db.Where("class_daily_date <= ?", *to)
		}
	}

	// Count total
	var total int64
	if err := db.Count(&total).Error; err != nil {
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	// Sorting & pagination
	db = db.Order(orderCol + " " + orderDir)
	if !p.All {
		db = db.Limit(p.Limit()).Offset(p.Offset())
	}

	// Fetch
	var rows []m.ClassDailyModel
	if err := db.Find(&rows).Error; err != nil {
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	// Map ke response
	out := make([]d.ClassDailyResponse, 0, len(rows))
	for i := range rows {
		out = append(out, d.NewClassDailyResponse(&rows[i]))
	}

	// Meta
	meta := helper.BuildMeta(total, p)

	return helper.JsonList(c, out, meta)
}

/* =========================
   GetByID
   ========================= */

func (ctl *ClassDailyController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var row m.ClassDailyModel
	if err := ctl.DB.
		Where("class_daily_id = ? AND deleted_at IS NULL", id).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "daily occurrence not found")
		}
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	return helper.JsonOK(c, "OK", d.NewClassDailyResponse(&row))
}

/* =========================
   Create
   ========================= */

func (ctl *ClassDailyController) Create(c *fiber.Ctx) error {
	var req d.CreateClassDailyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	// validator optional (nil)
	if err := req.Validate(nil); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var model m.ClassDailyModel
	if err := req.ApplyToModel(&model); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// Enforce masjid scope bila ada
	if err := enforceMasjidScope(c, &model.ClassDailyMasjidID); err != nil {
		return helper.JsonError(c, http.StatusForbidden, err.Error())
	}

	if err := ctl.DB.Create(&model).Error; err != nil {
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	return helper.JsonCreated(c, "Created", d.NewClassDailyResponse(&model))
}

/* =========================
   Update (PUT)
   ========================= */

func (ctl *ClassDailyController) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassDailyModel
	if err := ctl.DB.
		Where("class_daily_id = ? AND deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "daily occurrence not found")
		}
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	var req d.UpdateClassDailyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(nil); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.ApplyToModel(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// Enforce masjid scope
	if err := enforceMasjidScope(c, &existing.ClassDailyMasjidID); err != nil {
		return helper.JsonError(c, http.StatusForbidden, err.Error())
	}

	if err := ctl.DB.Save(&existing).Error; err != nil {
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	return helper.JsonUpdated(c, "Updated", d.NewClassDailyResponse(&existing))
}

/* =========================
   Patch (Partial)
   ========================= */

func (ctl *ClassDailyController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassDailyModel
	if err := ctl.DB.
		Where("class_daily_id = ? AND deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "daily occurrence not found")
		}
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	var req d.PatchClassDailyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(nil); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.ApplyPatch(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// Enforce masjid scope
	if err := enforceMasjidScope(c, &existing.ClassDailyMasjidID); err != nil {
		return helper.JsonError(c, http.StatusForbidden, err.Error())
	}

	if err := ctl.DB.Save(&existing).Error; err != nil {
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	return helper.JsonUpdated(c, "Updated", d.NewClassDailyResponse(&existing))
}

/* =========================
   Soft Delete
   ========================= */

func (ctl *ClassDailyController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassDailyModel
	if err := ctl.DB.
		Where("class_daily_id = ? AND deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "daily occurrence not found")
		}
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	// Enforce masjid scope
	if err := enforceMasjidScope(c, &existing.ClassDailyMasjidID); err != nil {
		return helper.JsonError(c, http.StatusForbidden, err.Error())
	}

	// GORM soft delete → set deleted_at
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	return helper.JsonDeleted(c, "Deleted", d.NewClassDailyResponse(&existing))
}
