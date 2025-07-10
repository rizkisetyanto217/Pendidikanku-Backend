package controller

import (
	"fmt"
	"log"

	"masjidku_backend/internals/features/masjids/masjids/dto"
	"masjidku_backend/internals/features/masjids/masjids/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidController struct {
	DB *gorm.DB
}

func NewMasjidController(db *gorm.DB) *MasjidController {
	return &MasjidController{DB: db}
}

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

// 🟢 CREATE MASJID (Single or Multiple)
func (mc *MasjidController) CreateMasjid(c *fiber.Ctx) error {
	log.Println("[INFO] Received request to create masjid")

	var singleReq dto.MasjidRequest
	var multipleReq []dto.MasjidRequest

	// 🌀 Multiple insert
	if err := c.BodyParser(&multipleReq); err == nil && len(multipleReq) > 0 {
		var multipleModels []model.MasjidModel
		for _, req := range multipleReq {
			m := dto.ToModelMasjid(&req, uuid.New())
			multipleModels = append(multipleModels, *m)
		}

		if err := mc.DB.Create(&multipleModels).Error; err != nil {
			log.Printf("[ERROR] Failed to create multiple masjids: %v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Gagal menyimpan banyak masjid",
			})
		}

		log.Printf("[SUCCESS] %d masjids created\n", len(multipleModels))

		// 🔁 Convert to DTO response
		var responses []dto.MasjidResponse
		for i := range multipleModels {
			responses = append(responses, dto.FromModelMasjid(&multipleModels[i]))
		}

		return c.Status(201).JSON(fiber.Map{
			"message": "Masjid berhasil dibuat (multiple)",
			"data":    responses,
		})
	}

	// 🌀 Single insert
	if err := c.BodyParser(&singleReq); err != nil {
		log.Printf("[ERROR] Invalid input: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Format input tidak valid",
		})
	}

	singleModel := dto.ToModelMasjid(&singleReq, uuid.New())

	if err := mc.DB.Create(&singleModel).Error; err != nil {
		log.Printf("[ERROR] Failed to create masjid: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal menyimpan masjid",
		})
	}

	log.Printf("[SUCCESS] Masjid created: %s\n", singleModel.MasjidName)

	return c.Status(201).JSON(fiber.Map{
		"message": "Masjid berhasil dibuat",
		"data":    dto.FromModelMasjid(singleModel),
	})
}

// 🟢 UPDATE MASJID
// 🟢 UPDATE MASJID (Partial Update)
func (mc *MasjidController) UpdateMasjid(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Printf("[INFO] Updating masjid with ID: %s\n", id)

	masjidUUID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Format ID tidak valid",
		})
	}

	var existing model.MasjidModel
	if err := mc.DB.First(&existing, "masjid_id = ?", masjidUUID).Error; err != nil {
		log.Printf("[ERROR] Masjid with ID %s not found\n", id)
		return c.Status(404).JSON(fiber.Map{
			"error": "Masjid tidak ditemukan",
		})
	}

	// Parsing input ke map agar bisa partial update
	var inputMap map[string]interface{}
	if err := c.BodyParser(&inputMap); err != nil {
		log.Printf("[ERROR] Invalid input: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Input tidak valid",
		})
	}

	// Hindari update field sensitif
	delete(inputMap, "masjid_id")
	delete(inputMap, "masjid_created_at")

	if err := mc.DB.Model(&existing).Updates(inputMap).Error; err != nil {
		log.Printf("[ERROR] Failed to update masjid: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal memperbarui masjid",
		})
	}

	log.Printf("[SUCCESS] Masjid updated: %s\n", existing.MasjidName)

	return c.JSON(fiber.Map{
		"message": "Masjid berhasil diperbarui",
		"data":    dto.FromModelMasjid(&existing),
	})
}

// 🟢 DELETE MASJID
func (mc *MasjidController) DeleteMasjid(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Printf("[INFO] Deleting masjid with ID: %s\n", id)

	if err := mc.DB.Delete(&model.MasjidModel{}, "masjid_id = ?", id).Error; err != nil {
		log.Printf("[ERROR] Failed to delete masjid: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal menghapus masjid",
		})
	}

	log.Printf("[SUCCESS] Masjid with ID %s deleted successfully\n", id)
	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Masjid dengan ID %s berhasil dihapus", id),
	})
}
