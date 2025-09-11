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
	helperAuth "masjidku_backend/internals/helpers/auth"

	d "masjidku_backend/internals/features/school/sessions/schedule/dto"
	m "masjidku_backend/internals/features/school/sessions/schedule/model"
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

// Ambil masjid_id aktif dari token (teacher/admin/dkm) dan pastikan konsisten dengan body.
// Jika token tidak punya scope masjid, dilepas (public).
func enforceMasjidScopeAuth(c *fiber.Ctx, bodyMasjidID *uuid.UUID) error {
	if bodyMasjidID == nil || *bodyMasjidID == uuid.Nil {
		return nil
	}
	act, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || act == uuid.Nil {
		return nil
	}
	if act != *bodyMasjidID {
		return fiber.NewError(fiber.StatusForbidden, "masjid scope mismatch")
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
	MasjidID         string `query:"masjid_id"`
	SectionID        string `query:"section_id"`
	ClassSubjectID   string `query:"class_subject_id"`
	CSSTID           string `query:"csst_id"`
	RoomID           string `query:"room_id"`
	TeacherID        string `query:"teacher_id"`
	Status           string `query:"status"`
	Active           *bool  `query:"active"`
	DayOfWeek        *int   `query:"dow"`
	OnDate           string `query:"on_date"`
	StartAfter       string `query:"start_after"`
	EndBefore        string `query:"end_before"`
	ClassScheduleID  string `query:"class_schedule_id"`   // <— NEW (single)
	ClassScheduleIDs string `query:"class_schedule_ids"`  // <— NEW (comma-separated)

	// Pagination & sort
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
	SortBy string `query:"sort_by"`
	Order  string `query:"order"`
}

func (ctl *ClassScheduleController) List(c *fiber.Ctx) error {
	var q listQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// Tenant override dari token (teacher-aware)
	if act, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && act != uuid.Nil {
		q.MasjidID = act.String()
	}

	// Whitelist sorting
	sortCol := map[string]string{
		"start_time": "class_schedules_start_time",
		"end_time":   "class_schedules_end_time",
		"created_at": "class_schedules_created_at",
		"updated_at": "class_schedules_updated_at",
	}
	sortBy := "class_schedules_start_time"
	if s := strings.TrimSpace(q.SortBy); s != "" {
		if col, ok := sortCol[s]; ok {
			sortBy = col
		}
	}
	order := "ASC"
	if strings.EqualFold(q.Order, "desc") {
		order = "DESC"
	}

	// Pagination clamp
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	// ===== Build base query with filters =====
	tx := ctl.DB.Model(&m.ClassScheduleModel{}).
		Where("class_schedules_deleted_at IS NULL")

	// by masjid
	if s := strings.TrimSpace(q.MasjidID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "masjid_id invalid")
		}
		tx = tx.Where("class_schedules_masjid_id = ?", s)
	}

	// by ids (NEW)
	if s := strings.TrimSpace(q.ClassScheduleID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "class_schedule_id invalid")
		}
		tx = tx.Where("class_schedules_id = ?", s)
	}
	if s := strings.TrimSpace(q.ClassScheduleIDs); s != "" {
		parts := strings.Split(s, ",")
		ids := make([]uuid.UUID, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			u, err := uuid.Parse(p)
			if err != nil {
				return helper.JsonError(c, http.StatusBadRequest, "class_schedule_ids mengandung UUID tidak valid")
			}
			ids = append(ids, u)
		}
		if len(ids) > 0 {
			tx = tx.Where("class_schedules_id IN ?", ids)
		}
	}

	// by foreign keys
	if s := strings.TrimSpace(q.SectionID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "section_id invalid")
		}
		tx = tx.Where("class_schedules_section_id = ?", s)
	}
	if s := strings.TrimSpace(q.ClassSubjectID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "class_subject_id invalid")
		}
		tx = tx.Where("class_schedules_class_subject_id = ?", s)
	}
	if s := strings.TrimSpace(q.CSSTID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "csst_id invalid")
		}
		tx = tx.Where("class_schedules_csst_id = ?", s)
	}
	if s := strings.TrimSpace(q.RoomID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "room_id invalid")
		}
		tx = tx.Where("class_schedules_room_id = ?", s)
	}
	if s := strings.TrimSpace(q.TeacherID); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "teacher_id invalid")
		}
		tx = tx.Where("class_schedules_teacher_id = ?", s)
	}

	// by status
	if s := strings.TrimSpace(q.Status); s != "" {
		switch m.SessionStatus(s) {
		case m.SessionScheduled, m.SessionOngoing, m.SessionCompleted, m.SessionCanceled:
			tx = tx.Where("class_schedules_status = ?", s)
		default:
			return helper.JsonError(c, http.StatusBadRequest, "status invalid")
		}
	}

	// by active
	if q.Active != nil {
		tx = tx.Where("class_schedules_is_active = ?", *q.Active)
	}

	// by day-of-week
	if q.DayOfWeek != nil {
		if *q.DayOfWeek < 1 || *q.DayOfWeek > 7 {
			return helper.JsonError(c, http.StatusBadRequest, "dow must be 1..7")
		}
		tx = tx.Where("class_schedules_day_of_week = ?", *q.DayOfWeek)
	}

	// by on_date (toleran end_date NULL)
	if s := strings.TrimSpace(q.OnDate); s != "" {
		dt, err := time.Parse("2006-01-02", s)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "on_date invalid (YYYY-MM-DD)")
		}
		dow := int(dt.Weekday()) // Sunday(0)..Saturday(6)
		if dow == 0 {
			dow = 7 // ISO 1..7
		}
		tx = tx.
			Where("?::date BETWEEN class_schedules_start_date AND COALESCE(class_schedules_end_date, ?::date)", dt, dt).
			Where("class_schedules_day_of_week = ?", dow)
	}

	// by time windows
	if s := strings.TrimSpace(q.StartAfter); s != "" {
		tm, err := parseTimeOfDayParam(s)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "start_after invalid (HH:mm/HH:mm:ss)")
		}
		tx = tx.Where("class_schedules_start_time >= ?", tm)
	}
	if s := strings.TrimSpace(q.EndBefore); s != "" {
		tm, err := parseTimeOfDayParam(s)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "end_before invalid (HH:mm/HH:mm:ss)")
		}
		tx = tx.Where("class_schedules_end_time <= ?", tm)
	}

	// ===== Count total =====
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return writePGError(c, err)
	}

	// ===== Fetch page =====
	var rows []m.ClassScheduleModel
	if err := tx.
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

	// Meta
	nextOffset := q.Offset + q.Limit
	hasMore := nextOffset < int(total)

	meta := fiber.Map{
		"limit":       q.Limit,
		"offset":      q.Offset,
		"count":       len(out),
		"total":       total,
		"has_more":    hasMore,
		"next_offset": func() *int { if hasMore { return &nextOffset }; return nil }(),
		"sort_by":     q.SortBy,
		"order":       strings.ToLower(order),
	}

	return helper.JsonList(c, out, meta)
}

/* =========================
   Create
   ========================= */
func (ctl *ClassScheduleController) Create(c *fiber.Ctx) error {
	// 🔐 Admin/DKM/Teacher
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

	var req d.CreateClassScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// 🔒 Ambil masjid_id dari token (admin/teacher) & override body
	actMasjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || actMasjidID == uuid.Nil {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan di token")
	}
	req.ClassSchedulesMasjidID = actMasjidID.String()

	// Validasi setelah di-inject
	if err := req.Validate(ctl.Validate); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var model m.ClassScheduleModel
	if err := req.ApplyToModel(&model); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// (Opsional; redundant karena sudah di-override, tapi aman)
	// if err := enforceMasjidScopeAuth(c, &model.ClassSchedulesMasjidID); err != nil {
	// 	return helper.JsonError(c, http.StatusForbidden, err.Error())
	// }

	if err := ctl.DB.Create(&model).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonCreated(c, "Schedule created", d.NewClassScheduleResponse(&model))
}

/* =========================
   Update (PUT)
   ========================= */

func (ctl *ClassScheduleController) Update(c *fiber.Ctx) error {
	// 🔐 Admin/DKM/Teacher
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

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
	if err := enforceMasjidScopeAuth(c, &existing.ClassSchedulesMasjidID); err != nil {
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
	// 🔐 Admin/DKM/Teacher
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

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
	if err := enforceMasjidScopeAuth(c, &existing.ClassSchedulesMasjidID); err != nil {
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
	// 🔐 Admin/DKM/Teacher
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return helper.JsonError(c, http.StatusForbidden, "Akses ditolak")
	}

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
	if err := enforceMasjidScopeAuth(c, &existing.ClassSchedulesMasjidID); err != nil {
		return helper.JsonError(c, http.StatusForbidden, err.Error())
	}

	// GORM soft delete → set class_schedules_deleted_at
	if err := ctl.DB.Delete(&existing).Error; err != nil {
		return writePGError(c, err)
	}

	return helper.JsonDeleted(c, "Schedule deleted", d.NewClassScheduleResponse(&existing))
}
