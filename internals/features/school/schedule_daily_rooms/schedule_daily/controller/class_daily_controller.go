// file: internals/features/school/class_daily/controller/class_daily_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

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

func parseDateParam(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	return time.Parse("2006-01-02", s)
}

func parseTimeOfDayParam(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	// support HH:mm and HH:mm:ss
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

type listQueryDaily struct {
	// Filter
	MasjidID        string `query:"masjid_id"`
	SectionID       string `query:"section_id"`
	TeacherID       string `query:"teacher_id"`
	RoomID          string `query:"room_id"`
	ScheduleID      string `query:"schedule_id"`
	AttendanceID    string `query:"attendance_id"`
	SubjectID       string `query:"subject_id"`
	AcademicTermsID string `query:"academic_terms_id"`
	Status          string `query:"status"` // scheduled|ongoing|finished|canceled
	Active          *bool  `query:"active"`
	DayOfWeek       *int   `query:"dow"`     // 1..7
	OnDate          string `query:"on_date"` // YYYY-MM-DD (exact date)
	From            string `query:"from"`    // YYYY-MM-DD
	To              string `query:"to"`      // YYYY-MM-DD
	StartAfter      string `query:"start_after"` // HH:mm / HH:mm:ss → start_time >=
	EndBefore       string `query:"end_before"`  // HH:mm / HH:mm:ss → end_time <=

	// Pagination & sort
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
	SortBy string `query:"sort_by"` // date|start_time|end_time|created_at|updated_at (default: date,start_time)
	Order  string `query:"order"`   // asc|desc (default: asc)
}

func (ctl *ClassDailyController) List(c *fiber.Ctx) error {
	var q listQueryDaily
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	db := ctl.DB.Model(&m.ClassDailyModel{})

	// Masjid scope dari Locals override filter masjid jika ada.
	if loc := c.Locals("masjid_id"); loc != nil {
		if s := strings.TrimSpace(fmt.Sprintf("%v", loc)); s != "" {
			q.MasjidID = s
		}
	}

	// Filters
	if q.MasjidID != "" {
		if _, err := uuid.Parse(q.MasjidID); err != nil {
			return fiber.NewError(http.StatusBadRequest, "masjid_id invalid")
		}
		db = db.Where("class_daily_masjid_id = ?", q.MasjidID)
	}
	if q.SectionID != "" {
		if _, err := uuid.Parse(q.SectionID); err != nil {
			return fiber.NewError(http.StatusBadRequest, "section_id invalid")
		}
		db = db.Where("class_daily_section_id = ?", q.SectionID)
	}
	if q.TeacherID != "" {
		if _, err := uuid.Parse(q.TeacherID); err != nil {
			return fiber.NewError(http.StatusBadRequest, "teacher_id invalid")
		}
		db = db.Where("class_daily_teacher_id = ?", q.TeacherID)
	}
	if q.RoomID != "" {
		if _, err := uuid.Parse(q.RoomID); err != nil {
			return fiber.NewError(http.StatusBadRequest, "room_id invalid")
		}
		db = db.Where("class_daily_room_id = ?", q.RoomID)
	}
	if q.ScheduleID != "" {
		if _, err := uuid.Parse(q.ScheduleID); err != nil {
			return fiber.NewError(http.StatusBadRequest, "schedule_id invalid")
		}
		db = db.Where("class_daily_schedule_id = ?", q.ScheduleID)
	}
	if q.AttendanceID != "" {
		if _, err := uuid.Parse(q.AttendanceID); err != nil {
			return fiber.NewError(http.StatusBadRequest, "attendance_id invalid")
		}
		db = db.Where("class_daily_attendance_id = ?", q.AttendanceID)
	}
	if q.SubjectID != "" {
		if _, err := uuid.Parse(q.SubjectID); err != nil {
			return fiber.NewError(http.StatusBadRequest, "subject_id invalid")
		}
		db = db.Where("class_daily_subject_id = ?", q.SubjectID)
	}
	if q.AcademicTermsID != "" {
		if _, err := uuid.Parse(q.AcademicTermsID); err != nil {
			return fiber.NewError(http.StatusBadRequest, "academic_terms_id invalid")
		}
		db = db.Where("class_daily_academic_terms_id = ?", q.AcademicTermsID)
	}
	if q.Status != "" {
		switch m.SessionStatus(q.Status) {
		case m.SessionScheduled, m.SessionOngoing, m.SessionFinished, m.SessionCanceled:
			db = db.Where("class_daily_status = ?", q.Status)
		default:
			return fiber.NewError(http.StatusBadRequest, "status invalid")
		}
	}
	if q.Active != nil {
		db = db.Where("class_daily_is_active = ?", *q.Active)
	}
	if q.DayOfWeek != nil {
		if *q.DayOfWeek < 1 || *q.DayOfWeek > 7 {
			return fiber.NewError(http.StatusBadRequest, "dow must be 1..7")
		}
		db = db.Where("class_daily_day_of_week = ?", *q.DayOfWeek)
	}

	// on_date filter (exact)
	if strings.TrimSpace(q.OnDate) != "" {
		d, err := parseDateParam(q.OnDate)
		if err != nil {
			return fiber.NewError(http.StatusBadRequest, "on_date invalid (YYYY-MM-DD)")
		}
		db = db.Where("class_daily_date = ?", d)
	}

	// Range date
	if strings.TrimSpace(q.From) != "" || strings.TrimSpace(q.To) != "" {
		var from, to *time.Time
		if strings.TrimSpace(q.From) != "" {
			d, err := parseDateParam(q.From)
			if err != nil {
				return fiber.NewError(http.StatusBadRequest, "from invalid (YYYY-MM-DD)")
			}
			from = &d
		}
		if strings.TrimSpace(q.To) != "" {
			d, err := parseDateParam(q.To)
			if err != nil {
				return fiber.NewError(http.StatusBadRequest, "to invalid (YYYY-MM-DD)")
			}
			to = &d
		}
		if from != nil && to != nil {
			db = db.Where("class_daily_date BETWEEN ? AND ?", *from, *to)
		} else if from != nil {
			db = db.Where("class_daily_date >= ?", *from)
		} else if to != nil {
			db = db.Where("class_daily_date <= ?", *to)
		}
	}

	// Time window
	if strings.TrimSpace(q.StartAfter) != "" {
		t, err := parseTimeOfDayParam(q.StartAfter)
		if err != nil {
			return fiber.NewError(http.StatusBadRequest, "start_after invalid (HH:mm/HH:mm:ss)")
		}
		db = db.Where("class_daily_start_time >= ?", t)
	}
	if strings.TrimSpace(q.EndBefore) != "" {
		t, err := parseTimeOfDayParam(q.EndBefore)
		if err != nil {
			return fiber.NewError(http.StatusBadRequest, "end_before invalid (HH:mm/HH:mm:ss)")
		}
		db = db.Where("class_daily_end_time <= ?", t)
	}

	// Sort & pagination
	sortBy := "class_daily_date"
	if s := strings.TrimSpace(q.SortBy); s != "" {
		switch s {
		case "date":
			sortBy = "class_daily_date"
		case "start_time":
			sortBy = "class_daily_start_time"
		case "end_time":
			sortBy = "class_daily_end_time"
		case "created_at":
			sortBy = "class_daily_created_at"
		case "updated_at":
			sortBy = "class_daily_updated_at"
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

	var rows []m.ClassDailyModel
	tx := db.Where("class_daily_deleted_at IS NULL").
		Order(sortBy + " " + order)
	// Stabilize sort when primary sort is date
	if sortBy == "class_daily_date" {
		tx = tx.Order("class_daily_start_time " + order)
	}
	if err := tx.Limit(q.Limit).Offset(q.Offset).Find(&rows).Error; err != nil {
		return mapPGError(err)
	}

	// Map ke response
	out := make([]d.ClassDailyResponse, 0, len(rows))
	for i := range rows {
		out = append(out, d.NewClassDailyResponse(&rows[i]))
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":   out,
		"limit":  q.Limit,
		"offset": q.Offset,
	})
}

/* =========================
   GetByID
   ========================= */

func (ctl *ClassDailyController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	var row m.ClassDailyModel
	if err := ctl.DB.
		Where("class_daily_id = ? AND class_daily_deleted_at IS NULL", id).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(http.StatusNotFound, "daily occurrence not found")
		}
		return mapPGError(err)
	}

	return c.Status(http.StatusOK).JSON(d.NewClassDailyResponse(&row))
}

/* =========================
   Create
   ========================= */

func (ctl *ClassDailyController) Create(c *fiber.Ctx) error {
	var req d.CreateClassDailyRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	// validator optional (nil)
	if err := req.Validate(nil); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	var model m.ClassDailyModel
	if err := req.ApplyToModel(&model); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	// Enforce masjid scope bila ada
	if err := enforceMasjidScope(c, &model.ClassDailyMasjidID); err != nil {
		return fiber.NewError(http.StatusForbidden, err.Error())
	}

	if err := ctl.DB.Create(&model).Error; err != nil {
		return mapPGError(err)
	}

	return c.Status(http.StatusCreated).JSON(d.NewClassDailyResponse(&model))
}

/* =========================
   Update (PUT)
   ========================= */

func (ctl *ClassDailyController) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	var existing m.ClassDailyModel
	if err := ctl.DB.
		Where("class_daily_id = ? AND class_daily_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(http.StatusNotFound, "daily occurrence not found")
		}
		return mapPGError(err)
	}

	var req d.UpdateClassDailyRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(nil); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	if err := req.ApplyToModel(&existing); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	// Enforce masjid scope
	if err := enforceMasjidScope(c, &existing.ClassDailyMasjidID); err != nil {
		return fiber.NewError(http.StatusForbidden, err.Error())
	}

	if err := ctl.DB.Save(&existing).Error; err != nil {
		return mapPGError(err)
	}

	return c.Status(http.StatusOK).JSON(d.NewClassDailyResponse(&existing))
}

/* =========================
   Patch (Partial)
   ========================= */

func (ctl *ClassDailyController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	var existing m.ClassDailyModel
	if err := ctl.DB.
		Where("class_daily_id = ? AND class_daily_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(http.StatusNotFound, "daily occurrence not found")
		}
		return mapPGError(err)
	}

	var req d.PatchClassDailyRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	if err := req.Validate(nil); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	if err := req.ApplyPatch(&existing); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	// Enforce masjid scope
	if err := enforceMasjidScope(c, &existing.ClassDailyMasjidID); err != nil {
		return fiber.NewError(http.StatusForbidden, err.Error())
	}

	if err := ctl.DB.Save(&existing).Error; err != nil {
		return mapPGError(err)
	}

	return c.Status(http.StatusOK).JSON(d.NewClassDailyResponse(&existing))
}

/* =========================
   Soft Delete
   ========================= */

func (ctl *ClassDailyController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	var existing m.ClassDailyModel
	if err := ctl.DB.
		Where("class_daily_id = ? AND class_daily_deleted_at IS NULL", id).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(http.StatusNotFound, "daily occurrence not found")
		}
		return mapPGError(err)
	}

	// Enforce masjid scope
	if err := enforceMasjidScope(c, &existing.ClassDailyMasjidID); err != nil {
		return fiber.NewError(http.StatusForbidden, err.Error())
	}

	// GORM soft delete → akan set class_daily_deleted_at
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		return mapPGError(err)
	}

	return c.SendStatus(http.StatusNoContent)
}
