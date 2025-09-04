// file: internals/features/lembaga/masjids/controller/masjid_profile_controller.go
package controller

import (
	"log"

	helper "masjidku_backend/internals/helpers"

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

	masjidUUID, err := uuid.Parse(masjidIDParam)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Format UUID masjid tidak valid")
	}

	var profile model.MasjidProfileModel
	if err := mpc.DB.
		Where("masjid_profile_masjid_id = ?", masjidUUID).
		First(&profile).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			log.Printf("[ERROR] Profile not found for masjid ID %s\n", masjidUUID)
			return helper.JsonError(c, fiber.StatusNotFound, "Profil masjid tidak ditemukan")
		}

		log.Printf("[ERROR] Database error: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Terjadi kesalahan saat mengambil data profil")
	}

	log.Printf("[SUCCESS] Retrieved profile for masjid ID %s\n", masjidUUID)
	return helper.JsonOK(c, "Profil masjid berhasil diambil", dto.FromModelMasjidProfile(&profile))
}

// 🟢 GET PROFILE BY SLUG
func (mpc *MasjidProfileController) GetProfileBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	log.Printf("[INFO] Fetching masjid by slug: %s\n", slug)

	// 1) Cari masjid berdasarkan slug
	var masjid model.MasjidModel
	if err := mpc.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mencari masjid")
	}

	// 2) Cari profil masjid berdasarkan masjid_id
	var profile model.MasjidProfileModel
	if err := mpc.DB.
		Where("masjid_profile_masjid_id = ?", masjid.MasjidID).
		First(&profile).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil masjid tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil profil masjid")
	}

	return helper.JsonOK(c, "Profil masjid berhasil diambil", dto.FromModelMasjidProfile(&profile))
}
