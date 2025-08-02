package controller

import (
	"log"
	"time"

	"masjidku_backend/internals/features/masjids/lectures/main/dto"
	"masjidku_backend/internals/features/masjids/lectures/main/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureSchedulesController struct {
	DB *gorm.DB
}

func NewLectureSchedulesController(db *gorm.DB) *LectureSchedulesController {
	return &LectureSchedulesController{DB: db}
}

// ✅ Create
func (ctrl *LectureSchedulesController) Create(c *fiber.Ctx) error {
	var body dto.CreateLectureScheduleRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Format data tidak valid")
	}

	// Validasi manual bisa ditambahkan jika pakai validator.v10

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
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	return c.JSON(fiber.Map{"message": "Berhasil membuat jadwal kajian", "data": data})
}

// ✅ Get All (optional: filter by masjid_id via relasi)
func (ctrl *LectureSchedulesController) GetAll(c *fiber.Ctx) error {
	var list []model.LectureSchedulesModel

	// optional: ?masjid_id=...
	masjidID := c.Query("masjid_id")
	query := ctrl.DB.Preload("Lecture")

	if masjidID != "" {
		query = query.Joins("JOIN lectures ON lectures.lecture_id = lecture_schedules.lecture_schedules_lecture_id").
			Where("lectures.lecture_masjid_id = ?", masjidID)
	}

	if err := query.Order("lecture_schedules_created_at DESC").Find(&list).Error; err != nil {
		log.Println("[ERROR] Gagal mengambil jadwal:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return c.JSON(list)
}

// ✅ Get By ID
func (ctrl *LectureSchedulesController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var data model.LectureSchedulesModel

	if err := ctrl.DB.Preload("Lecture").
		First(&data, "lecture_schedules_id = ?", id).Error; err != nil {
		log.Println("[ERROR] Jadwal tidak ditemukan:", err)
		return fiber.NewError(fiber.StatusNotFound, "Jadwal tidak ditemukan")
	}

	return c.JSON(data)
}


// ✅ Get by masjid_slug
func (ctrl *LectureSchedulesController) GetByMasjidSlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Slug tidak boleh kosong")
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
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data jadwal")
	}

	return c.JSON(list)
}


// ✅ Update
func (ctrl *LectureSchedulesController) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var body dto.UpdateLectureScheduleRequest

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Format data tidak valid")
	}

	var data model.LectureSchedulesModel
	if err := ctrl.DB.First(&data, "lecture_schedules_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}

	// ✅ Partial update
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
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update data")
	}

	return c.JSON(fiber.Map{"message": "Berhasil update jadwal", "data": data})
}

// ✅ Delete
func (ctrl *LectureSchedulesController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.DB.
		Where("lecture_schedules_id = ?", id).
		Delete(&model.LectureSchedulesModel{}).Error; err != nil {
		log.Println("[ERROR] Gagal hapus jadwal:", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal hapus data")
	}

	return c.JSON(fiber.Map{"message": "Berhasil menghapus jadwal"})
}
