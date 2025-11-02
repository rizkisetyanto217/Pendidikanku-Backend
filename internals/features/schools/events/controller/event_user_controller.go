package controller

import (
	"log"
	"schoolku_backend/internals/features/schools/events/dto"
	"schoolku_backend/internals/features/schools/events/model"
	helper "schoolku_backend/internals/helpers"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// 🟢 GET /api/u/events/by-school/:slug?page=&limit=
func (ctrl *EventController) GetEventsBySchoolSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak boleh kosong")
	}

	// Ambil school_id dari slug
	var m struct {
		SchoolID string `gorm:"column:school_id"`
	}
	if err := ctrl.DB.
		Table("schools").
		Select("school_id").
		Where("school_slug = ?", slug).
		Take(&m).Error; err != nil || m.SchoolID == "" {
		log.Printf("[ERROR] School slug '%s' tidak ditemukan: %v", slug, err)
		return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
	}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Hitung total
	var total int64
	if err := ctrl.DB.Model(&model.EventModel{}).
		Where("event_school_id = ?", m.SchoolID).
		Count(&total).Error; err != nil {
		log.Printf("[ERROR] Count events by slug '%s': %v", slug, err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung event")
	}

	// Ambil data page
	var events []model.EventModel
	if err := ctrl.DB.
		Where("event_school_id = ?", m.SchoolID).
		Order("event_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&events).Error; err != nil {
		log.Printf("[ERROR] Gagal ambil events by slug '%s': %v", slug, err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil event")
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int((total + int64(limit) - 1) / int64(limit)),
		"has_next":    int64(page*limit) < total,
		"has_prev":    page > 1,
		"school_slug": slug,
	}

	return helper.JsonList(c, dto.ToEventResponseList(events), pagination)
}
