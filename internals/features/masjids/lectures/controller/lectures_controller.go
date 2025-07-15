package controller

import (
	"masjidku_backend/internals/features/masjids/lectures/dto"
	"masjidku_backend/internals/features/masjids/lectures/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureController struct {
	DB *gorm.DB
}

func NewLectureController(db *gorm.DB) *LectureController {
	return &LectureController{DB: db}
}


// 🟢 GET /api/a/lectures
func (ctrl *LectureController) GetAllLectures(c *fiber.Ctx) error {
	var lectures []model.LectureModel

	if err := ctrl.DB.Order("lecture_created_at DESC").Find(&lectures).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil daftar kajian",
			"error":   err.Error(),
		})
	}

	// Ubah ke bentuk response DTO
	lectureResponses := make([]dto.LectureResponse, len(lectures))
	for i, l := range lectures {
		lectureResponses[i] = *dto.ToLectureResponse(&l)
	}

	return c.JSON(fiber.Map{
		"message": "Daftar kajian berhasil diambil",
		"data":    lectureResponses,
	})
}

// 🟢 POST /api/a/lectures
func (ctrl *LectureController) CreateLecture(c *fiber.Ctx) error {
	var req dto.LectureRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Permintaan tidak valid", "error": err.Error()})
	}

	newLecture := req.ToModel()
	if err := ctrl.DB.Create(newLecture).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Gagal menyimpan data", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Kajian berhasil dibuat",
		"data":    dto.ToLectureResponse(newLecture),
	})
}


// ✅ GET /api/a/lectures/by-masjid
func (ctrl *LectureController) GetByMasjidID(c *fiber.Ctx) error {
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Masjid ID tidak valid atau tidak ditemukan",
		})
	}

	var lectures []model.LectureModel
	if err := ctrl.DB.
		Where("lecture_masjid_id = ?", masjidID).
		Order("lecture_created_at DESC").
		Find(&lectures).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data lecture",
		})
	}

	if len(lectures) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Belum ada lecture untuk masjid ini",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Daftar lecture berhasil ditemukan",
		"data":    dto.ToLectureResponseList(lectures),
	})
}



// 🟢 GET /api/a/lectures/:id
func (ctrl *LectureController) GetLectureByID(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	var lecture model.LectureModel

	if err := ctrl.DB.First(&lecture, "lecture_id = ?", lectureID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Kajian tidak ditemukan", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil detail kajian",
		"data":    dto.ToLectureResponse(&lecture),
	})
}

// 🟡 PUT /api/a/lectures/:id
func (ctrl *LectureController) UpdateLecture(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	var req dto.LectureRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Permintaan tidak valid", "error": err.Error()})
	}

	var lecture model.LectureModel
	if err := ctrl.DB.First(&lecture, "lecture_id = ?", lectureID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"message": "Kajian tidak ditemukan", "error": err.Error()})
	}

	// Update dengan data baru
	updatedLecture := req.ToModel()
	updatedLecture.LectureID = lecture.LectureID // tetap pakai ID lama

	if err := ctrl.DB.Model(&lecture).Updates(updatedLecture).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Gagal memperbarui data", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Kajian berhasil diperbarui",
		"data":    dto.ToLectureResponse(&lecture),
	})
}

// 🔴 DELETE /api/a/lectures/:id
func (ctrl *LectureController) DeleteLecture(c *fiber.Ctx) error {
	lectureID := c.Params("id")

	if err := ctrl.DB.Delete(&model.LectureModel{}, "lecture_id = ?", lectureID).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Gagal menghapus kajian", "error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Kajian berhasil dihapus"})
}
