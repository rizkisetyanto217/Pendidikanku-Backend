package controller

import (
	"log"
	"masjidku_backend/internals/features/lembaga/masjids/dto"
	"masjidku_backend/internals/features/lembaga/masjids/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// 🟢 GET ALL MASJIDS
func (mc *MasjidController) GetAllMasjids(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all masjids")

	var masjids []model.MasjidModel
	if err := mc.DB.Find(&masjids).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch masjids: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal mengambil data masjid",
		})
	}

	log.Printf("[SUCCESS] Retrieved %d masjids\n", len(masjids))

	// 🔁 Transform ke DTO
	var masjidDTOs []dto.MasjidResponse
	for _, m := range masjids {
		masjidDTOs = append(masjidDTOs, dto.FromModelMasjid(&m))
	}

	return c.JSON(fiber.Map{
		"message": "Data semua masjid berhasil diambil",
		"total":   len(masjidDTOs),
		"data":    masjidDTOs,
	})
}

// 🟢 GET VERIFIED MASJIDS
func (mc *MasjidController) GetAllVerifiedMasjids(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all verified masjids")

	var masjids []model.MasjidModel
	if err := mc.DB.Where("masjid_is_verified = ?", true).Find(&masjids).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch verified masjids: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal mengambil data masjid terverifikasi",
		})
	}

	log.Printf("[SUCCESS] Retrieved %d verified masjids\n", len(masjids))

	// 🔁 Transform ke DTO
	var masjidDTOs []dto.MasjidResponse
	for _, m := range masjids {
		masjidDTOs = append(masjidDTOs, dto.FromModelMasjid(&m))
	}

	return c.JSON(fiber.Map{
		"message": "Data masjid terverifikasi berhasil diambil",
		"total":   len(masjidDTOs),
		"data":    masjidDTOs,
	})
}

// 🟢 GET VERIFIED MASJID BY ID
func (mc *MasjidController) GetVerifiedMasjidByID(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Printf("[INFO] Fetching verified masjid with ID: %s\n", id)

	masjidUUID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Format ID tidak valid",
		})
	}

	var masjid model.MasjidModel
	if err := mc.DB.
		Where("masjid_id = ? AND masjid_is_verified = ?", masjidUUID, true).
		First(&masjid).Error; err != nil {
		log.Printf("[ERROR] Verified masjid with ID %s not found\n", id)
		return c.Status(404).JSON(fiber.Map{
			"error": "Masjid terverifikasi tidak ditemukan",
		})
	}

	log.Printf("[SUCCESS] Retrieved verified masjid: %s\n", masjid.MasjidName)

	masjidDTO := dto.FromModelMasjid(&masjid)
	return c.JSON(fiber.Map{
		"message": "Data masjid terverifikasi berhasil diambil",
		"data":    masjidDTO,
	})
}


// 🟢 GET MASJID BY SLUG
func (mc *MasjidController) GetMasjidBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	log.Printf("[INFO] Fetching masjid with slug: %s\n", slug)

	var masjid model.MasjidModel
	if err := mc.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		log.Printf("[ERROR] Masjid with slug %s not found\n", slug)
		return c.Status(404).JSON(fiber.Map{
			"error": "Masjid tidak ditemukan",
		})
	}

	log.Printf("[SUCCESS] Retrieved masjid: %s\n", masjid.MasjidName)

	// 🔁 Transform ke DTO
	masjidDTO := dto.FromModelMasjid(&masjid)

	return c.JSON(fiber.Map{
		"message": "Data masjid berhasil diambil",
		"data":    masjidDTO,
	})
}
