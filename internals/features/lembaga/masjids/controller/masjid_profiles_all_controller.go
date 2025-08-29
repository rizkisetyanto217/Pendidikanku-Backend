package controller

import (
	"log"
	"masjidku_backend/internals/features/lembaga/masjids/dto"
	"masjidku_backend/internals/features/lembaga/masjids/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 🟢 GET PROFILE BY MASJID_ID
func (mpc *MasjidProfileController) GetProfileByMasjidID(c *fiber.Ctx) error {
	masjidIDParam := c.Params("masjid_id")
	log.Printf("[INFO] Fetching profile for masjid ID: %s\n", masjidIDParam)

	// Validasi UUID format
	masjidUUID, err := uuid.Parse(masjidIDParam)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format UUID masjid tidak valid",
		})
	}

	var profile model.MasjidProfileModel
	err = mpc.DB.
		// Preload("Masjid"). // preload relasi opsional
		Where("masjid_profile_masjid_id = ?", masjidUUID).
		First(&profile).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[ERROR] Profile not found for masjid ID %s\n", masjidUUID)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Profil masjid tidak ditemukan",
			})
		}

		log.Printf("[ERROR] Database error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Terjadi kesalahan saat mengambil data profil",
		})
	}

	log.Printf("[SUCCESS] Retrieved profile for masjid ID %s\n", masjidUUID)
	return c.JSON(fiber.Map{
		"message": "Profil masjid berhasil diambil",
		"data":    dto.FromModelMasjidProfile(&profile),
	})
}

func (mpc *MasjidProfileController) GetProfileBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	// 1. Cari masjid berdasarkan slug
	var masjid model.MasjidModel
	if err := mpc.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Masjid tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mencari masjid"})
	}

	// 2. Cari profil masjid berdasarkan masjid_id
	var profile model.MasjidProfileModel
	if err := mpc.DB.
		Where("masjid_profile_masjid_id = ?", masjid.MasjidID).
		First(&profile).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profil masjid tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil profil masjid"})
	}

	return c.JSON(fiber.Map{
		"message": "Profil masjid berhasil diambil",
		"data":    dto.FromModelMasjidProfile(&profile),
	})
}