// file: internals/features/school/class_schedules/controller/class_schedule_controller.go
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

	d "masjidku_backend/internals/features/school/sessions_assesment/schedule_daily/dto"
	m "masjidku_backend/internals/features/school/sessions_assesment/schedule_daily/model"
)

/* =========================
   Controller & Constructor
   ========================= */

type ClassScheduleController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func New(db *gorm.DB, v *validator.Validate) *ClassScheduleController {
	return &ClassScheduleController{DB: db, Validate: v}
}

/* =========================
   Helpers
   ========================= */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := strings.TrimSpace(c.Params(name))
	if idStr == "" {
		return uuid.Nil, fmt.Errorf("%s is required", name)
	}
	return uuid.Parse(idStr)
}

// Ambil masjid_id dari Locals bila ada. Jika ada dan berbeda dengan request, tolak.
func enforceMasjidScope(c *fiber.Ctx, bodyMasjidID *uuid.UUID) error {
	loc := c.Locals("masjid_id")
	if loc == nil {
		return nil
	}
	locStr := strings.TrimSpace(fmt.Sprintf("%v", loc))
	if locStr == "" {
		return nil
	}
	locID, err := uuid.Parse(locStr)
	if err != nil {
		return fmt.Errorf("invalid masjid scope in context")
	}
	if bodyMasjidID != nil && locID != *bodyMasjidID {
		return fmt.Errorf("masjid scope mismatch")
	}
	return nil
}

// --- PG error mapping ---

type pgSQLErr interface {
	SQLState() string
	Error() string
}

func mapPGError(err error) (int, string) {
	// 23P01 = exclusion_violation
	// 23503 = foreign_key_violation
	// 23505 = unique_violation
	var pgErr pgSQLErr
	if errors.As(err, &pgErr) {
		switch pgErr.SQLState() {
		case "23P01":
			return http.StatusConflict, "Bentrok jadwal: time range overlap (room/section/teacher)."
		case "23503":
			return http.StatusBadRequest, "Referensi tidak ditemukan (FK violation)."
		case "23505":
			return http.StatusConflict, "Data duplikat (unique violation)."
		}
	}
	return http.StatusInternalServerError, err.Error()
}

func writePGError(c *fiber.Ctx, err error) error {
	code, msg := mapPGError(err)
	return helper.JsonError(c, code, msg)
}

func parseTimeOfDayParam(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	// support HH:mm lalu HH:mm:ss
	if t, err := time.Parse("15:04", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("15:04:05", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time format (want HH:mm or HH:mm:ss)")
}

/* =========================
   Query: List
   ========================= */

type listQuery struct {
	// Filter
	MasjidID       string `query:"masjid_id"`
	SectionID      string `query:"section_id"`
	ClassSubjectID string `query:"class_subject_id"`
	RoomID         string `query:"room_id"`
	TeacherID      string `query:"teacher_id"` // ✨ baru
	Status         string `query:"status"`     // scheduled|ongoing|completed|canceled
	Active         *bool  `query:"active"`
	DayOfWeek      *int   `query:"dow"`        // 1..7
	OnDate         string `query:"on_date"`    // YYYY-MM-DD (di antara start_date & end_date) + DOW match
	StartAfter     string `query:"start_after"`// HH:mm / HH:mm:ss → start_time >=
	EndBefore      string `query:"end_before"` // HH:mm / HH:mm:ss → end_time <=

	// Pagination & sort
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
	SortBy string `query:"sort_by"` // start_time|end_time|created_at|updated_at (default: start_time)
	Order  string `query:"order"`   // asc|desc (default: asc)
}

func (ctl *ClassScheduleController) List(c *fiber.Ctx) error {
	var q listQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	db := ctl.DB.Model(&m.ClassScheduleModel{})

	// Masjid scope dari Locals override filter masjid jika ada.
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
		db = db.Where("class_schedules_masjid_id = ?", q.MasjidID)
	}
	if q.SectionID != "" {
		if _, err := uuid.Parse(q.SectionID); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "section_id invalid")
		}
		db = db.Where("class_schedules_section_id = ?", q.SectionID)
	}
	if q.ClassSubjectID != "" {
		if _, err := uuid.Parse(q.ClassSubjectID); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "class_subject_id invalid")
		}
		db = db.Where("class_schedules_class_subject_id = ?", q.ClassSubjectID)
	}
	if q.RoomID != "" {
		if _, err := uuid.Parse(q.RoomID); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "room_id invalid")
		}
		db = db.Where("class_schedules_room_id = ?", q.RoomID)
	}
	if q.TeacherID != "" { // ✨ filter teacher
		if _, err := uuid.Parse(q.TeacherID); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "teacher_id invalid")
		}
		db = db.Where("class_schedules_teacher_id = ?", q.TeacherID)
	}
	if q.Status != "" {
		switch m.SessionStatus(q.Status) {
		case m.SessionScheduled, m.SessionOngoing, m.SessionCompleted, m.SessionCanceled:
			db = db.Where("class_schedules_status = ?", q.Status)
		default:
			return helper.JsonError(c, http.StatusBadRequest, "status invalid")
		}
	}
	if q.Active != nil {
		db = db.Where("class_schedules_is_active = ?", *q.Active)
	}
	if q.DayOfWeek != nil {
		if *q.DayOfWeek < 1 || *q.DayOfWeek > 7 {
			return helper.JsonError(c, http.StatusBadRequest, "dow must be 1..7")
		}
		db = db.Where("class_schedules_day_of_week = ?", *q.DayOfWeek)
	}

	// on_date filter → tanggal di rentang start..end, dan DOW match
	if strings.TrimSpace(q.OnDate) != "" {
		d, err := time.Parse("2006-01-02", q.OnDate)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "on_date invalid (YYYY-MM-DD)")
		}
		// Go: Sunday(0)..Saturday(6) → ISO: Monday(1)..Sunday(7)
		dow := int(d.Weekday()) // 0..6
		if dow == 0 {
			dow = 7
		}
		db = db.Where("? BETWEEN class_schedules_start_date AND class_schedules_end_date", d).
			Where("class_schedules_day_of_week = ?", dow)
	}

	// Time window
	if strings.TrimSpace(q.StartAfter) != "" {
		t, err := parseTimeOfDayParam(q.StartAfter)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "start_after invalid (HH:mm/HH:mm:ss)")
		}
		db = db.Where("class_schedules_start_time >= ?", t)
	}
	if strings.TrimSpace(q.EndBefore) != "" {
		t, err := parseTimeOfDayParam(q.EndBefore)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "end_before invalid (HH:mm/HH:mm:ss)")
		}
		db = db.Where("class_schedules_end_time <= ?", t)
	}

	// Sort & pagination
	sortBy := "class_schedules_start_time"
	if s := strings.TrimSpace(q.SortBy); s != "" {
		switch s {
		case "start_time":
			sortBy = "class_schedules_start_time"
		case "end_time":
			sortBy = "class_schedules_end_time"
		case "created_at":
			sortBy = "class_schedules_created_at"
		case "updated_at":
			sortBy = "class_schedules_updated_at"
		default:
			// keep default
		}
	}
	order := "ASC"
	if strings.EqualFold(q.Order, "desc") {
		order = "DESC"
	}

	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	var rows []m.ClassScheduleModel
	if err := db.
		Where("class_schedules_deleted_at IS NULL").
		Order(sortBy + " " + order).
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return writePGError(c, err)
	}

	// Map ke response
	out := make([]d.ClassScheduleResponse, 0, len(rows))
	for i := range rows {
		out = append(out, d.NewClassScheduleResponse(&rows[i]))
	}

	// pagination meta sederhana
	meta := fiber.Map{
		"limit":  q.Limit,
		"offset": q.Offset,
	}

	return helper.JsonList(c, out, meta)
}

/* =========================
   GetByID
   ========================= */

func (ctl *ClassScheduleController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var row m.ClassScheduleModel
	if err := ctl.DB.
		Where("class_schedule_id = ? AND class_schedules_deleted_at IS NULL", id).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "schedule not found")
		}
		return writePGError(c, err)
	}

	return helper.JsonOK(c, "OK", d.NewClassScheduleResponse(&row))
}

/* =========================
   Create
   ========================= */

func (ctl *ClassScheduleController) Create(c *fiber.Ctx) error {
	var req d.CreateClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(ctl.Validate); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var model m.ClassScheduleModel
	if err := req.ApplyToModel(&model); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// Enforce masjid scope bila ada
	if err := enforceMasjidScope(c, &model.ClassSchedulesMasjidID); err != nil {
		return helper.JsonError(c, http.StatusForbidden, err.Error())
	}

	if err := ctl.DB.Create(&model).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonCreated(c, "Schedule created", d.NewClassScheduleResponse(&model))
}

/* =========================
   Update (PUT)
   ========================= */

func (ctl *ClassScheduleController) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassScheduleModel
	if err := ctl.DB.
		Where("class_schedule_id = ? AND class_schedules_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "schedule not found")
		}
		return writePGError(c, err)
	}

	var req d.UpdateClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(ctl.Validate); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.ApplyToModel(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// Enforce masjid scope
	if err := enforceMasjidScope(c, &existing.ClassSchedulesMasjidID); err != nil {
		return helper.JsonError(c, http.StatusForbidden, err.Error())
	}

	if err := ctl.DB.Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "Schedule updated", d.NewClassScheduleResponse(&existing))
}

/* =========================
   Patch (Partial)
   ========================= */

func (ctl *ClassScheduleController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassScheduleModel
	if err := ctl.DB.
		Where("class_schedule_id = ? AND class_schedules_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "schedule not found")
		}
		return writePGError(c, err)
	}

	var req d.PatchClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(ctl.Validate); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	if err := req.ApplyPatch(&existing); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// Enforce masjid scope
	if err := enforceMasjidScope(c, &existing.ClassSchedulesMasjidID); err != nil {
		return helper.JsonError(c, http.StatusForbidden, err.Error())
	}

	if err := ctl.DB.Save(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonUpdated(c, "Schedule updated", d.NewClassScheduleResponse(&existing))
}

/* =========================
   Soft Delete
   ========================= */

func (ctl *ClassScheduleController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var existing m.ClassScheduleModel
	if err := ctl.DB.
		Where("class_schedule_id = ? AND class_schedules_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "schedule not found")
		}
		return writePGError(c, err)
	}

	// Enforce masjid scope
	if err := enforceMasjidScope(c, &existing.ClassSchedulesMasjidID); err != nil {
		return helper.JsonError(c, http.StatusForbidden, err.Error())
	}

	// GORM soft delete → akan set class_schedules_deleted_at
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonDeleted(c, "Schedule deleted", d.NewClassScheduleResponse(&existing))
}
