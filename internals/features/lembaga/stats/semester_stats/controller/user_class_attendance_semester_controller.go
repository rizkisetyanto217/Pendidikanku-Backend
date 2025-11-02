// internals/features/lembaga/class_sections/attendance_semester_stats/controller/user_class_attendance_semester_stats_controller.go
package controller

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "schoolku_backend/internals/features/lembaga/stats/semester_stats/dto"
	m "schoolku_backend/internals/features/lembaga/stats/semester_stats/model"
)

// ====================== CONTROLLER ======================

type SemesterStatsController struct {
	DB *gorm.DB
}

func NewSemesterStatsController(db *gorm.DB) *SemesterStatsController {
	return &SemesterStatsController{DB: db}
}

// --------------------------------------------------------
// GET /api/a/semester-stats
// "Get all" dengan dukungan filter/pagination (opsional)
// Query params mengikuti dto.ListSemesterStatsQuery
// --------------------------------------------------------
func (h *SemesterStatsController) List(c *fiber.Ctx) error {
	// default paging
	var q dto.ListSemesterStatsQuery
	q.Limit, q.Offset = 20, 0

	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := h.DB.Model(&m.UserClassAttendanceSemesterStatsModel{})

	// filter opsional
	if q.SchoolID != nil {
		tx = tx.Where("school_id = ?", *q.SchoolID)
	}
	if q.UserClassID != nil {
		tx = tx.Where("user_class_id = ?", *q.UserClassID)
	}
	if q.SectionID != nil {
		tx = tx.Where("section_id = ?", *q.SectionID)
	}
	if q.Start != nil {
		tx = tx.Where("period_start >= ?", *q.Start)
	}
	if q.End != nil {
		tx = tx.Where("period_end <= ?", *q.End)
	}

	// total untuk pagination
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// ambil data berhalaman
	var rows []m.UserClassAttendanceSemesterStatsModel
	if err := tx.
		Order("period_start DESC, period_end DESC, created_at DESC").
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := dto.FromModels(rows, total)
	return c.Status(fiber.StatusOK).JSON(out)
}

// --------------------------------------------------------
// GET /api/a/semester-stats/by-user/:user_id
// "Get by user id": ambil semua semester stats milik user tertentu
// (melalui JOIN ke user_classes untuk memetakan user → user_class_id)
// Query params opsional: school_id, section_id, start, end, limit, offset
// --------------------------------------------------------
func (h *SemesterStatsController) ListByUserID(c *fiber.Ctx) error {
	userIDStr := strings.TrimSpace(c.Params("user_id"))
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "user_id tidak valid")
	}

	// default paging
	var q dto.ListSemesterStatsQuery
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	// Base: stats JOIN user_classes agar bisa filter berdasarkan user_id
	// Tabel dan kolom disesuaikan dengan konvensi kamu:
	// user_classes.user_classes_id, user_classes.user_classes_user_id
	tx := h.DB.Model(&m.UserClassAttendanceSemesterStatsModel{}).
		Joins("JOIN user_classes ON user_classes.user_classes_id = user_class_attendance_semester_stats.user_class_id").
		Where("user_classes.user_classes_user_id = ?", userID)

	// filter opsional tambahan
	if q.SchoolID != nil {
		tx = tx.Where("user_class_attendance_semester_stats.school_id = ?", *q.SchoolID)
	}
	if q.SectionID != nil {
		tx = tx.Where("user_class_attendance_semester_stats.section_id = ?", *q.SectionID)
	}
	if q.Start != nil {
		tx = tx.Where("user_class_attendance_semester_stats.period_start >= ?", *q.Start)
	}
	if q.End != nil {
		tx = tx.Where("user_class_attendance_semester_stats.period_end <= ?", *q.End)
	}

	// total untuk pagination
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// data
	var rows []m.UserClassAttendanceSemesterStatsModel
	if err := tx.
		Order("user_class_attendance_semester_stats.period_start DESC, user_class_attendance_semester_stats.period_end DESC, user_class_attendance_semester_stats.created_at DESC").
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := dto.FromModels(rows, total)
	return c.Status(fiber.StatusOK).JSON(out)
}
