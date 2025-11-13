// file: internals/features/school/classes/class_schedules/controller/class_schedule_rule_list_controller.go
package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	ruleDTO "schoolku_backend/internals/features/school/classes/class_schedules/dto"
	ruleModel "schoolku_backend/internals/features/school/classes/class_schedules/model"
)

type ClassScheduleRuleListController struct{ DB *gorm.DB }

func NewClassScheduleRuleListController(db *gorm.DB) *ClassScheduleRuleListController {
	return &ClassScheduleRuleListController{DB: db}
}

// parse "HH:MM[:SS]" jadi *time.Time (tanggal dummy 2000-01-01, zona lokal)
func parseTODToTimePtr(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// terima "15:04:05" atau "15:04"
	if t, err := time.ParseInLocation("15:04:05", s, time.Local); err == nil {
		tt := time.Date(2000, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, time.Local)
		return &tt
	}
	if t, err := time.ParseInLocation("15:04", s, time.Local); err == nil {
		tt := time.Date(2000, 1, 1, t.Hour(), t.Minute(), 0, 0, time.Local)
		return &tt
	}
	return nil
}

type listMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type ClassScheduleRuleListResponse struct {
	Items []ruleDTO.ClassScheduleRuleResponse `json:"items"`
	Meta  listMeta                            `json:"meta"`
}

// GET /admin/class-schedule-rules
func (ctl *ClassScheduleRuleListController) List(c *fiber.Ctx) error {
	// ✅ Role guard
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// ✅ Resolve school context → tentukan tenant
	mc, er := helperAuth.ResolveSchoolContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}
	var schoolID uuid.UUID
	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		schoolID = id
	case helperAuth.IsTeacher(c):
		if mc.ID != uuid.Nil {
			schoolID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, http.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id
		} else if id, er := helperAuth.GetActiveSchoolID(c); er == nil && id != uuid.Nil {
			schoolID = id
		}
		if schoolID == uuid.Nil || !helperAuth.UserHasSchool(c, schoolID) {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope school tidak valid untuk Teacher")
		}
	default:
		return fiber.NewError(fiber.StatusUnauthorized, "Tidak diizinkan")
	}

	// ✅ Parse & validate query
	var q ruleDTO.ListClassScheduleRuleQuery
	_ = c.QueryParser(&q)
	if err := validator.New().Struct(q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	limit := 50
	offset := 0
	if q.Limit != nil {
		limit = *q.Limit
	}
	if q.Offset != nil {
		offset = *q.Offset
	}

	// ✅ Build base query
	tx := ctl.DB.Model(&ruleModel.ClassScheduleRuleModel{}).
		Where("class_schedule_rule_deleted_at IS NULL").
		Where("class_schedule_rule_school_id = ?", schoolID)

	// Filters
	if q.ScheduleID != nil {
		tx = tx.Where("class_schedule_rule_schedule_id = ?", *q.ScheduleID)
	}
	if q.DayOfWeek != nil {
		tx = tx.Where("class_schedule_rule_day_of_week = ?", *q.DayOfWeek)
	}
	if q.WeekParity != nil {
		// enum di DB: all|odd|even
		tx = tx.Where("class_schedule_rule_week_parity = ?", strings.ToLower(strings.TrimSpace(*q.WeekParity)))
	}
	if q.StudentTeacherID != nil {
		tx = tx.Where("class_schedule_rule_csst_student_teacher_id = ?", *q.StudentTeacherID)
	}
	if q.ClassSectionID != nil {
		tx = tx.Where("class_schedule_rule_csst_class_section_id = ?", *q.ClassSectionID)
	}
	if q.ClassSubjectID != nil {
		tx = tx.Where("class_schedule_rule_csst_class_subject_id = ?", *q.ClassSubjectID)
	}
	if q.ClassRoomID != nil {
		tx = tx.Where("class_schedule_rule_csst_class_room_id = ?", *q.ClassRoomID)
	}

	// Total count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	// Sorting
	orderDir := "ASC"
	if q.Order != nil && strings.EqualFold(*q.Order, "desc") {
		orderDir = "DESC"
	}
	orderBy := ""
	if q.SortBy != nil {
		switch *q.SortBy {
		case "day_of_week":
			orderBy = "class_schedule_rule_day_of_week " + orderDir
		case "start_time":
			orderBy = "class_schedule_rule_start_time " + orderDir
		case "end_time":
			orderBy = "class_schedule_rule_end_time " + orderDir
		case "created_at":
			orderBy = "class_schedule_rule_created_at " + orderDir
		case "updated_at":
			orderBy = "class_schedule_rule_updated_at " + orderDir
		}
	}
	if strings.TrimSpace(orderBy) == "" {
		// default: rapi untuk tampilan jadwal
		orderBy = "class_schedule_rule_day_of_week ASC, class_schedule_rule_start_time ASC, class_schedule_rule_end_time ASC, class_schedule_rule_created_at DESC"
	}

	// Data
	var rows []ruleModel.ClassScheduleRuleModel
	if err := tx.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data rules")
	}

	resp := ClassScheduleRuleListResponse{
		Items: ruleDTO.FromRuleModels(rows),
		Meta:  listMeta{Limit: limit, Offset: offset, Total: int(total)},
	}
	return helper.JsonOK(c, "Daftar rules berhasil diambil", resp)
}
