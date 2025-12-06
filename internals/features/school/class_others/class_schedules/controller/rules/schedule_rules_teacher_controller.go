// file: internals/features/school/classes/class_schedules/controller/class_schedule_rule_list_controller.go
package controller

import (
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	ruleDTO "madinahsalam_backend/internals/features/school/class_others/class_schedules/dto"
	ruleModel "madinahsalam_backend/internals/features/school/class_others/class_schedules/model"
)

type ClassScheduleRuleListController struct{ DB *gorm.DB }

func NewClassScheduleRuleListController(db *gorm.DB) *ClassScheduleRuleListController {
	return &ClassScheduleRuleListController{DB: db}
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

/*
=========================================================

	LIST (STAFF) — Hanya DKM/Admin/Teacher di school ini
	Route contoh:
	  GET /api/a/:school_id/class-schedule-rules

	Resolver school:
	  - Pakai helperAuth.ResolveSchoolForDKMOrTeacher:
	    * Ambil school_id dari context/token
	    * Guard role DKM/Admin/Teacher

=========================================================
*/
func (ctl *ClassScheduleRuleListController) List(c *fiber.Ctx) error {
	// ===== Tenant + role guard via auth helper =====
	schoolID, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		return err
	}

	// ===== Parse & validate query =====
	var q ruleDTO.ListClassScheduleRuleQuery
	_ = c.QueryParser(&q)
	if err := validator.New().Struct(q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	limit := 50
	offset := 0
	if q.Limit != nil {
		limit = *q.Limit
	}
	if q.Offset != nil {
		offset = *q.Offset
	}

	// ===== Base query (tenant-safe) =====
	tx := ctl.DB.WithContext(c.Context()).
		Model(&ruleModel.ClassScheduleRuleModel{}).
		Where("class_schedule_rule_deleted_at IS NULL").
		Where("class_schedule_rule_school_id = ?", schoolID)

	// ===== Filters =====
	if q.ScheduleID != nil && *q.ScheduleID != uuid.Nil {
		tx = tx.Where("class_schedule_rule_schedule_id = ?", *q.ScheduleID)
	}
	if q.DayOfWeek != nil {
		tx = tx.Where("class_schedule_rule_day_of_week = ?", *q.DayOfWeek)
	}
	if q.WeekParity != nil {
		// enum di DB: all|odd|even
		tx = tx.Where("class_schedule_rule_week_parity = ?", strings.ToLower(strings.TrimSpace(*q.WeekParity)))
	}
	if q.StudentTeacherID != nil && *q.StudentTeacherID != uuid.Nil {
		tx = tx.Where("class_schedule_rule_csst_student_teacher_id = ?", *q.StudentTeacherID)
	}
	if q.ClassSectionID != nil && *q.ClassSectionID != uuid.Nil {
		tx = tx.Where("class_schedule_rule_csst_class_section_id = ?", *q.ClassSectionID)
	}
	if q.ClassSubjectID != nil && *q.ClassSubjectID != uuid.Nil {
		tx = tx.Where("class_schedule_rule_csst_class_subject_id = ?", *q.ClassSubjectID)
	}
	if q.ClassRoomID != nil && *q.ClassRoomID != uuid.Nil {
		tx = tx.Where("class_schedule_rule_csst_class_room_id = ?", *q.ClassRoomID)
	}

	// ===== Total count =====
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal menghitung total")
	}

	// ===== Sorting =====
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
		// default: rapi buat tampilan jadwal mingguan
		orderBy = "class_schedule_rule_day_of_week ASC, class_schedule_rule_start_time ASC, class_schedule_rule_end_time ASC, class_schedule_rule_created_at DESC"
	}

	// ===== Data =====
	var rows []ruleModel.ClassScheduleRuleModel
	if err := tx.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengambil data rules")
	}

	resp := ClassScheduleRuleListResponse{
		Items: ruleDTO.FromRuleModels(rows),
		Meta:  listMeta{Limit: limit, Offset: offset, Total: int(total)},
	}
	return helper.JsonOK(c, "Daftar rules berhasil diambil", resp)
}

/*
=========================================================

	LIST PUBLIC — untuk umum / murid / tamu (READ-ONLY)
	Route contoh:
	  GET /api/u/:school_id/class-schedule-rules

	Resolver school:
	  - Pakai helperAuth.ResolveSchoolIDFromContext
	    (ambil dari token/active school)

=========================================================
*/
func (ctl *ClassScheduleRuleListController) ListPublic(c *fiber.Ctx) error {
	// ===== Tenant resolvers dari auth helper =====
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// (Opsional) kalau kamu mau minimal harus member school:
	// if e := helperAuth.EnsureMemberSchool(c, schoolID); e != nil {
	//     return e
	// }

	// ===== Parse & validate query (read-only) =====
	var q ruleDTO.ListClassScheduleRuleQuery
	_ = c.QueryParser(&q)
	if err := validator.New().Struct(q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	limit := 50
	offset := 0
	if q.Limit != nil {
		limit = *q.Limit
	}
	if q.Offset != nil {
		offset = *q.Offset
	}

	// ===== Base query public (tenant + alive) =====
	tx := ctl.DB.WithContext(c.Context()).
		Model(&ruleModel.ClassScheduleRuleModel{}).
		Where("class_schedule_rule_deleted_at IS NULL").
		Where("class_schedule_rule_school_id = ?", schoolID)

	// ===== Filters sederhana =====
	if q.DayOfWeek != nil {
		tx = tx.Where("class_schedule_rule_day_of_week = ?", *q.DayOfWeek)
	}
	if q.ClassSectionID != nil && *q.ClassSectionID != uuid.Nil {
		tx = tx.Where("class_schedule_rule_csst_class_section_id = ?", *q.ClassSectionID)
	}
	if q.ClassSubjectID != nil && *q.ClassSubjectID != uuid.Nil {
		tx = tx.Where("class_schedule_rule_csst_class_subject_id = ?", *q.ClassSubjectID)
	}
	if q.ClassRoomID != nil && *q.ClassRoomID != uuid.Nil {
		tx = tx.Where("class_schedule_rule_csst_class_room_id = ?", *q.ClassRoomID)
	}
	if q.StudentTeacherID != nil && *q.StudentTeacherID != uuid.Nil {
		tx = tx.Where("class_schedule_rule_csst_student_teacher_id = ?", *q.StudentTeacherID)
	}

	// ===== Total count =====
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal menghitung total")
	}

	// ===== Sorting =====
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
		orderBy = "class_schedule_rule_day_of_week ASC, class_schedule_rule_start_time ASC, class_schedule_rule_end_time ASC, class_schedule_rule_created_at DESC"
	}

	// ===== Data =====
	var rows []ruleModel.ClassScheduleRuleModel
	if err := tx.
		Order(orderBy).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengambil data rules")
	}

	resp := ClassScheduleRuleListResponse{
		Items: ruleDTO.FromRuleModels(rows),
		Meta:  listMeta{Limit: limit, Offset: offset, Total: int(total)},
	}
	return helper.JsonOK(c, "Daftar rules berhasil diambil", resp)
}
