package controller

import (
	"log"
	"strconv"
	"time"

	"masjidku_backend/internals/features/masjids/lectures/main/dto"
	"masjidku_backend/internals/features/masjids/lectures/main/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureSchedulesController struct {
	DB *gorm.DB
}

func NewLectureSchedulesController(db *gorm.DB) *LectureSchedulesController {
	return &LectureSchedulesController{DB: db}
}

// ✅ Create  | POST /api/a/lecture-schedules
func (ctrl *LectureSchedulesController) Create(c *fiber.Ctx) error {
	var body dto.CreateLectureScheduleRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Format data tidak valid")
	}

	data := model.LectureSchedulesModel{
		LectureSchedulesLectureID:              body.LectureSchedulesLectureID,
		LectureSchedulesTitle:                  body.LectureSchedulesTitle,
		LectureSchedulesDayOfWeek:              body.LectureSchedulesDayOfWeek,
		LectureSchedulesStartTime:              body.LectureSchedulesStartTime,
		LectureSchedulesEndTime:                body.LectureSchedulesEndTime,
		LectureSchedulesPlace:                  body.LectureSchedulesPlace,
		LectureSchedulesNotes:                  body.LectureSchedulesNotes,
		LectureSchedulesIsActive:               body.LectureSchedulesIsActive != nil && *body.LectureSchedulesIsActive,
		LectureSchedulesIsPaid:                 body.LectureSchedulesIsPaid != nil && *body.LectureSchedulesIsPaid,
		LectureSchedulesPrice:                  body.LectureSchedulesPrice,
		LectureSchedulesCapacity:               body.LectureSchedulesCapacity,
		LectureSchedulesIsRegistrationRequired: body.LectureSchedulesIsRegistrationRequired != nil && *body.LectureSchedulesIsRegistrationRequired,
	}

	if err := ctrl.DB.Create(&data).Error; err != nil {
		log.Println("[ERROR] Gagal membuat jadwal:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	return helper.JsonCreated(c, "Berhasil membuat jadwal kajian", data)
}

// ✅ Get All (opsional filter ?masjid_id=...) | GET /api/a/lecture-schedules
// ✅ Get All (with pagination & optional ?masjid_id=...) | GET /api/a/lecture-schedules
func (ctrl *LectureSchedulesController) GetAll(c *fiber.Ctx) error {
	// ---- Pagination params ----
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}
	// Batas aman supaya query tidak membebani server
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// ---- Optional filter ----
	masjidID := c.Query("masjid_id")

	// ---- Base query (untuk data) ----
	dataQ := ctrl.DB.Model(&model.LectureSchedulesModel{}).
		Preload("Lecture")

	if masjidID != "" {
		dataQ = dataQ.
			Joins("JOIN lectures ON lectures.lecture_id = lecture_schedules.lecture_schedules_lecture_id").
			Where("lectures.lecture_masjid_id = ?", masjidID)
	}

	// ---- Count total ----
	var total int64
	countQ := ctrl.DB.Model(&model.LectureSchedulesModel{})
	if masjidID != "" {
		countQ = countQ.Joins("JOIN lectures ON lectures.lecture_id = lecture_schedules.lecture_schedules_lecture_id").
			Where("lectures.lecture_masjid_id = ?", masjidID)
	}
	if err := countQ.Count(&total).Error; err != nil {
		log.Println("[ERROR] Count jadwal:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// ---- Fetch data page ----
	var list []model.LectureSchedulesModel
	if err := dataQ.
		Order("lecture_schedules_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&list).Error; err != nil {
		log.Println("[ERROR] Gagal mengambil jadwal:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ---- Build pagination meta ----
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := fiber.Map{
		"page":         page,
		"limit":        limit,
		"total":        total,
		"total_pages":  totalPages,
		"has_next":     page < totalPages,
		"has_prev":     page > 1,
		"next_page":    func() int { if page < totalPages { return page + 1 }; return page }(),
		"prev_page":    func() int { if page > 1 { return page - 1 }; return page }(),
		"masjid_id":    masjidID,
	}

	return helper.JsonList(c, list, pagination)
}


// ✅ Get By ID | GET /api/a/lecture-schedules/:id
func (ctrl *LectureSchedulesController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var data model.LectureSchedulesModel

	if err := ctrl.DB.Preload("Lecture").
		First(&data, "lecture_schedules_id = ?", id).Error; err != nil {
		log.Println("[ERROR] Jadwal tidak ditemukan:", err)
		return helper.JsonError(c, fiber.StatusNotFound, "Jadwal tidak ditemukan")
	}

	return helper.JsonOK(c, "Berhasil mengambil detail jadwal", data)
}

// ✅ Get by masjid_slug | GET /public/lecture-schedules/by-masjid/:slug
func (ctrl *LectureSchedulesController) GetByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak boleh kosong")
	}

	var list []model.LectureSchedulesModel
	err := ctrl.DB.
		Joins("JOIN lectures ON lectures.lecture_id = lecture_schedules.lecture_schedules_lecture_id").
		Where("lectures.lecture_masjid_id = (SELECT masjid_id FROM masjids WHERE masjid_slug = ?)", slug).
		Preload("Lecture").
		Order("lecture_schedules_day_of_week ASC, lecture_schedules_start_time ASC").
		Find(&list).Error
	if err != nil {
		log.Println("[ERROR] Gagal ambil jadwal by slug:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data jadwal")
	}

	return helper.JsonOK(c, "Daftar jadwal berhasil diambil", list)
}

// ✅ Update | PUT /api/a/lecture-schedules/:id
func (ctrl *LectureSchedulesController) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var body dto.UpdateLectureScheduleRequest

	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Format data tidak valid")
	}

	var data model.LectureSchedulesModel
	if err := ctrl.DB.First(&data, "lecture_schedules_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
	}

	// Partial update
	if body.LectureSchedulesTitle != nil {
		data.LectureSchedulesTitle = *body.LectureSchedulesTitle
	}
	if body.LectureSchedulesDayOfWeek != nil {
		data.LectureSchedulesDayOfWeek = *body.LectureSchedulesDayOfWeek
	}
	if body.LectureSchedulesStartTime != nil {
		data.LectureSchedulesStartTime = *body.LectureSchedulesStartTime
	}
	if body.LectureSchedulesEndTime != nil {
		data.LectureSchedulesEndTime = body.LectureSchedulesEndTime
	}
	if body.LectureSchedulesPlace != nil {
		data.LectureSchedulesPlace = *body.LectureSchedulesPlace
	}
	if body.LectureSchedulesNotes != nil {
		data.LectureSchedulesNotes = *body.LectureSchedulesNotes
	}
	if body.LectureSchedulesIsActive != nil {
		data.LectureSchedulesIsActive = *body.LectureSchedulesIsActive
	}
	if body.LectureSchedulesIsPaid != nil {
		data.LectureSchedulesIsPaid = *body.LectureSchedulesIsPaid
	}
	if body.LectureSchedulesPrice != nil {
		data.LectureSchedulesPrice = body.LectureSchedulesPrice
	}
	if body.LectureSchedulesCapacity != nil {
		data.LectureSchedulesCapacity = body.LectureSchedulesCapacity
	}
	if body.LectureSchedulesIsRegistrationRequired != nil {
		data.LectureSchedulesIsRegistrationRequired = *body.LectureSchedulesIsRegistrationRequired
	}

	now := time.Now()
	data.LectureSchedulesUpdatedAt = &now

	if err := ctrl.DB.Save(&data).Error; err != nil {
		log.Println("[ERROR] Gagal update:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Berhasil memperbarui jadwal", data)
}

// ✅ Delete | DELETE /api/a/lecture-schedules/:id
func (ctrl *LectureSchedulesController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.
		Where("lecture_schedules_id = ?", id).
		Delete(&model.LectureSchedulesModel{}).Error; err != nil {
		log.Println("[ERROR] Gagal hapus jadwal:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Berhasil menghapus jadwal", fiber.Map{"lecture_schedules_id": id})
}
