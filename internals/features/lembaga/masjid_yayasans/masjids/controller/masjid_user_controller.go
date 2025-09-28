package controller

import (
	"log"

	helper "masjidku_backend/internals/helpers"

	masjidDto "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/dto"
	masjidModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// 🟢 GET ALL MASJIDS
func (mc *MasjidController) GetAllMasjids(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all masjids")

	var masjids []masjidModel.MasjidModel
	if err := mc.DB.Find(&masjids).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch masjids: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid")
	}

	log.Printf("[SUCCESS] Retrieved %d masjids\n", len(masjids))

	// 🔁 Transform ke DTO response
	resp := make([]masjidDto.MasjidResp, 0, len(masjids))
	for i := range masjids {
		resp = append(resp, masjidDto.FromModel(&masjids[i]))
	}

	return helper.JsonList(c, resp, fiber.Map{
		"total": len(resp),
	})
}

// 🟢 GET VERIFIED MASJIDS
func (mc *MasjidController) GetAllVerifiedMasjids(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all verified masjids")

	var masjids []masjidModel.MasjidModel
	if err := mc.DB.Where("masjid_is_verified = ?", true).Find(&masjids).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch verified masjids: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid terverifikasi")
	}

	log.Printf("[SUCCESS] Retrieved %d verified masjids\n", len(masjids))

	// 🔁 Transform ke DTO response
	resp := make([]masjidDto.MasjidResp, 0, len(masjids))
	for i := range masjids {
		resp = append(resp, masjidDto.FromModel(&masjids[i]))
	}

	return helper.JsonList(c, resp, fiber.Map{
		"total": len(resp),
	})
}

// 🟢 GET VERIFIED MASJID BY ID
func (mc *MasjidController) GetVerifiedMasjidByID(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Printf("[INFO] Fetching verified masjid with ID: %s\n", id)

	masjidUUID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Format ID tidak valid")
	}

	var m masjidModel.MasjidModel
	if err := mc.DB.
		Where("masjid_id = ? AND masjid_is_verified = ?", masjidUUID, true).
		First(&m).Error; err != nil {
		log.Printf("[ERROR] Verified masjid with ID %s not found\n", id)
		return helper.JsonError(c, fiber.StatusNotFound, "Masjid terverifikasi tidak ditemukan")
	}

	log.Printf("[SUCCESS] Retrieved verified masjid: %s\n", m.MasjidName)

	return helper.JsonOK(c, "Data masjid terverifikasi berhasil diambil", masjidDto.FromModel(&m))
}

// 🟢 GET MASJID BY SLUG
func (mc *MasjidController) GetMasjidBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	log.Printf("[INFO] Fetching masjid with slug: %s\n", slug)

	var m masjidModel.MasjidModel
	if err := mc.DB.Where("masjid_slug = ?", slug).First(&m).Error; err != nil {
		log.Printf("[ERROR] Masjid with slug %s not found\n", slug)
		return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	log.Printf("[SUCCESS] Retrieved masjid: %s\n", m.MasjidName)

	return helper.JsonOK(c, "Data masjid berhasil diambil", masjidDto.FromModel(&m))
}
