package controller

import (
	"masjidku_backend/internals/features/masjids/lectures/main/dto"
	"masjidku_backend/internals/features/masjids/lectures/main/model"

	"github.com/gofiber/fiber/v2"
)

// ✅ GET /public/lectures/by-masjid-slug/:slug
func (ctrl *LectureController) GetLectureByMasjidSlug(c *fiber.Ctx) error {
	// Ambil slug dari parameter URL
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Slug masjid tidak ditemukan di URL",
		})
	}

	// Ambil masjid_id berdasarkan slug
	var masjid struct {
		MasjidID string `gorm:"column:masjid_id"`
	}
	if err := ctrl.DB.
		Table("masjids").
		Select("masjid_id").
		Where("masjid_slug = ?", slug).
		Scan(&masjid).Error; err != nil || masjid.MasjidID == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Masjid dengan slug tersebut tidak ditemukan",
		})
	}

	// Ambil lectures berdasarkan masjid_id
	var lectures []model.LectureModel
	if err := ctrl.DB.
		Where("lecture_masjid_id = ?", masjid.MasjidID).
		Order("lecture_created_at DESC").
		Find(&lectures).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data lecture",
		})
	}

	if len(lectures) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Belum ada tema kajian untuk masjid ini",
		})
	}

	// Sukses
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Daftar tema kajian berhasil ditemukan",
		"data":    dto.ToLectureResponseList(lectures),
	})
}
