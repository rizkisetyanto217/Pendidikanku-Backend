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

// ✅ POST /api/a/lectures/by-masjid-latest
func (ctrl *LectureController) GetByMasjidID(c *fiber.Ctx) error {
	type RequestBody struct {
		MasjidID string `json:"masjid_id"`
	}

	var body RequestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Permintaan tidak valid",
		})
	}

	if body.MasjidID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Masjid ID wajib diisi",
		})
	}

	userIDRaw := c.Locals("user_id")

	var lecture model.LectureModel
	if err := ctrl.DB.
		Where("lecture_masjid_id = ?", body.MasjidID).
		Order("lecture_created_at DESC").
		First(&lecture).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Belum ada lecture untuk masjid ini",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data lecture",
		})
	}

	// Jika user login, cari data user_lecture
	if userIDRaw != nil {
		userIDStr, ok := userIDRaw.(string)
		if ok && userIDStr != "" {
			var userLecture model.UserLectureModel
			if err := ctrl.DB.Where("user_lecture_user_id = ? AND user_lecture_lecture_id = ?", userIDStr, lecture.LectureID).
				First(&userLecture).Error; err == nil {
				// Gabungkan data
				lectureMap := map[string]interface{}{
					"lecture":      dto.ToLectureResponse(&lecture),
					"user_lecture": userLecture,
				}
				return c.Status(fiber.StatusOK).JSON(fiber.Map{
					"message": "Lecture dan partisipasi user ditemukan",
					"data":    lectureMap,
				})
			}
		}
	}

	// Jika user tidak login atau tidak ada data user_lecture
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Lecture terbaru berhasil ditemukan",
		"data":    dto.ToLectureResponse(&lecture),
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
