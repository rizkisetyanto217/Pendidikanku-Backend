// file: internals/features/school/class_daily/controller/class_daily_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	d "masjidku_backend/internals/features/school/sessions_assesment/schedule_daily/dto"
	m "masjidku_backend/internals/features/school/sessions_assesment/schedule_daily/model"
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
	u := &url.URL{RawQuery: string(c.Request().URI().QueryString())}
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

func containsUUID(list []uuid.UUID, id uuid.UUID) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
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
	// Pagination & sorting (default: date ASC)
	p := helper.ParseWith(stdReqFromFiber(c), "date", "asc", helper.AdminOpts)

	// Whitelist kolom sorting → pakai nama kolom di DB
	allowedSort := map[string]string{
		"date":       "class_daily_date",
		"created_at": "class_daily_created_at",
		"updated_at": "class_daily_updated_at",
	}
	orderCol := allowedSort["date"]
	if col, ok := allowedSort[strings.ToLower(p.SortBy)]; ok {
		orderCol = col
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

	// Masjid scope
	accessibleMasjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil || len(accessibleMasjidIDs) == 0 {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}

	// masjid_id explicit → must be in scope
	var filterMasjidIDs []uuid.UUID
	if s := strings.TrimSpace(q.MasjidID); s != "" {
		mid, e := uuid.Parse(s)
		if e != nil {
			return helper.JsonError(c, http.StatusBadRequest, "masjid_id invalid")
		}
		if !containsUUID(accessibleMasjidIDs, mid) {
			return helper.JsonError(c, http.StatusForbidden, "Tidak punya akses ke masjid tersebut")
		}
		filterMasjidIDs = []uuid.UUID{mid}
	} else {
		filterMasjidIDs = accessibleMasjidIDs
	}

	// ❌ HAPUS filter manual deleted_at — GORM soft delete sudah otomatis
	db := ctl.DB.Model(&m.ClassDailyModel{})

	// Scope masjid
	if len(filterMasjidIDs) == 1 {
		db = db.Where("class_daily_masjid_id = ?", filterMasjidIDs[0])
	} else {
		db = db.Where("class_daily_masjid_id IN ?", filterMasjidIDs)
	}

	// Filters
	if s := strings.TrimSpace(q.SectionID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "section_id invalid")
		}
		db = db.Where("class_daily_section_id = ?", s)
	}

	// NOTE: Kolom class_daily_schedule_id belum ada di DDL class_daily.
	// Jika belum kamu tambahkan, JANGAN aktifkan filter ini.
	if s := strings.TrimSpace(q.ScheduleID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "schedule_id invalid")
		}
		// Uncomment HANYA jika kolom sudah ada:
		// db = db.Where("class_daily_schedule_id = ?", s)
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

	// on_date (exact)
	if s := strings.TrimSpace(q.OnDate); s != "" {
		dt, err := parseDateParam(s)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "on_date invalid (YYYY-MM-DD)")
		}
		db = db.Where("class_daily_date = ?", dt)
	}

	// Range date
	if strings.TrimSpace(q.From) != "" || strings.TrimSpace(q.To) != "" {
		var from, to *time.Time
		if s := strings.TrimSpace(q.From); s != "" {
			dt, err := parseDateParam(s)
			if err != nil {
				return helper.JsonError(c, http.StatusBadRequest, "from invalid (YYYY-MM-DD)")
			}
			from = &dt
		}
		if s := strings.TrimSpace(q.To); s != "" {
			dt, err := parseDateParam(s)
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

	// Count
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

	// Map response
	out := make([]d.ClassDailyResponse, 0, len(rows))
	for i := range rows {
		out = append(out, d.NewClassDailyResponse(&rows[i]))
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}

/* =========================
   GetByID
   ========================= */

func (ctl *ClassDailyController) GetByID(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	// Ambil data
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

	// Enforce scope via helper
	accessibleMasjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil || len(accessibleMasjidIDs) == 0 {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	if !containsUUID(accessibleMasjidIDs, row.ClassDailyMasjidID) {
		return helper.JsonError(c, http.StatusForbidden, "Tidak punya akses ke masjid tersebut")
	}

	return helper.JsonOK(c, "OK", d.NewClassDailyResponse(&row))
}

/* =========================
   Create
   ========================= */

func (ctl *ClassDailyController) Create(c *fiber.Ctx) error {
	var req d.CreateClassDailyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}
	if err := req.Validate(nil); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var model m.ClassDailyModel
	if err := req.ApplyToModel(&model); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// Tentukan masjid_id dengan helper:
	// - kalau tidak diisi request → pakai active masjid dari token
	// - apapun nilainya harus belong ke scope user
	accessibleMasjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil || len(accessibleMasjidIDs) == 0 {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	if model.ClassDailyMasjidID == uuid.Nil {
		if mid, e := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); e == nil && mid != uuid.Nil {
			model.ClassDailyMasjidID = mid
		}
	}
	if model.ClassDailyMasjidID == uuid.Nil || !containsUUID(accessibleMasjidIDs, model.ClassDailyMasjidID) {
		return helper.JsonError(c, http.StatusForbidden, "Tidak punya akses ke masjid tersebut")
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
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
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

	// Enforce scope
	accessibleMasjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil || len(accessibleMasjidIDs) == 0 {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	if !containsUUID(accessibleMasjidIDs, existing.ClassDailyMasjidID) {
		return helper.JsonError(c, http.StatusForbidden, "Tidak punya akses ke masjid tersebut")
	}

	var req d.UpdateClassDailyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}
	if err := req.Validate(nil); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.ApplyToModel(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// (Opsional) kalau request mau mengubah masjid_id, pastikan masih in-scope
	if existing.ClassDailyMasjidID != uuid.Nil && !containsUUID(accessibleMasjidIDs, existing.ClassDailyMasjidID) {
		return helper.JsonError(c, http.StatusForbidden, "Tidak punya akses ke masjid tersebut")
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
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
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

	// Enforce scope
	accessibleMasjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil || len(accessibleMasjidIDs) == 0 {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	if !containsUUID(accessibleMasjidIDs, existing.ClassDailyMasjidID) {
		return helper.JsonError(c, http.StatusForbidden, "Tidak punya akses ke masjid tersebut")
	}

	var req d.PatchClassDailyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}
	if err := req.Validate(nil); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.ApplyPatch(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// (Opsional) kalau patch mengubah masjid_id, cek scope lagi
	if existing.ClassDailyMasjidID != uuid.Nil && !containsUUID(accessibleMasjidIDs, existing.ClassDailyMasjidID) {
		return helper.JsonError(c, http.StatusForbidden, "Tidak punya akses ke masjid tersebut")
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
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
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

	// Enforce scope
	accessibleMasjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil || len(accessibleMasjidIDs) == 0 {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	if !containsUUID(accessibleMasjidIDs, existing.ClassDailyMasjidID) {
		return helper.JsonError(c, http.StatusForbidden, "Tidak punya akses ke masjid tersebut")
	}

	// GORM soft delete → set deleted_at
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		code, msg := mapPGError(err)
		return helper.JsonError(c, code, msg)
	}

	return helper.JsonDeleted(c, "Deleted", d.NewClassDailyResponse(&existing))
}
