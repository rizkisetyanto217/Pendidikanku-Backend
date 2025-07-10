package controller

import (
	"fmt"
	"log"
	"time"

	"masjidku_backend/internals/features/utils/tooltips/model"
	database "masjidku_backend/internals/databases"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// TooltipsController menangani semua operasi terkait tooltips
type TooltipsController struct {
	DB *gorm.DB
}

// NewTooltipsController membuat instance baru dari TooltipsController
func NewTooltipsController(db *gorm.DB) *TooltipsController {
	return &TooltipsController{DB: db}
}

func (tc *TooltipsController) GetTooltipsID(c *fiber.Ctx) error {
	log.Println("Fetching tooltips for given keywords")

	var request struct {
		Keywords []string `json:"keywords"`
	}
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	var tooltipIDs []uint
	for _, keyword := range request.Keywords {
		var tooltip model.Tooltip
		if err := database.DB.
			Select("tooltip_id").
			Where("tooltip_keyword = ?", keyword).
			First(&tooltip).Error; err == nil {
			tooltipIDs = append(tooltipIDs, tooltip.TooltipID)
		}
	}

	return c.JSON(fiber.Map{"tooltips_id": tooltipIDs})
}

// InsertTooltip menangani permintaan untuk menambahkan tooltips baru
func (tc *TooltipsController) CreateTooltip(c *fiber.Ctx) error {
	log.Println("[INFO] Received request to create tooltip(s)")

	var (
		single   model.Tooltip
		multiple []model.Tooltip
	)

	raw := c.Body()
	if len(raw) > 0 && raw[0] == '[' {
		if err := c.BodyParser(&multiple); err != nil {
			log.Printf("[ERROR] Failed to parse tooltip array: %v", err)
			return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON array"})
		}

		if len(multiple) == 0 {
			log.Println("[ERROR] Received empty tooltip array")
			return c.Status(400).JSON(fiber.Map{"error": "Tooltip array is empty"})
		}

		for i, tip := range multiple {
			if tip.TooltipKeyword == "" || tip.TooltipDescriptionShort == "" || tip.TooltipDescriptionLong == "" {
				log.Printf("[ERROR] Invalid tooltip at index %d: %+v\n", i, tip)
				return c.Status(400).JSON(fiber.Map{
					"error": "Each tooltip must have tooltip_keyword, tooltip_description_short, and tooltip_description_long",
					"index": i,
					"data":  tip,
				})
			}
		}

		if err := tc.DB.Create(&multiple).Error; err != nil {
			log.Printf("[ERROR] Failed to insert multiple tooltips: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create tooltips"})
		}

		log.Printf("[SUCCESS] Inserted %d tooltips", len(multiple))
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "Multiple tooltips created successfully",
			"data":    multiple,
		})
	}

	if err := c.BodyParser(&single); err != nil {
		log.Printf("[ERROR] Failed to parse single tooltip: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request format (expected object or array)",
		})
	}

	log.Printf("[DEBUG] Parsed single tooltip: %+v", single)

	if single.TooltipKeyword == "" || single.TooltipDescriptionShort == "" || single.TooltipDescriptionLong == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "tooltip_keyword, tooltip_description_short, and tooltip_description_long are required",
		})
	}

	if err := tc.DB.Create(&single).Error; err != nil {
		log.Printf("[ERROR] Failed to insert tooltip: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create tooltip"})
	}

	log.Printf("[SUCCESS] Tooltip created: ID=%d, Keyword=%s", single.TooltipID, single.TooltipKeyword)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Tooltip created successfully",
		"data":    single,
	})
}

func (tc *TooltipsController) GetTooltipByID(c *fiber.Ctx) error {
	start := time.Now()
	id := c.Params("id")
	var tooltip model.Tooltip

	if err := tc.DB.
		Select("tooltip_id", "tooltip_keyword", "tooltip_description_short", "tooltip_description_long").
		First(&tooltip, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status": false,
			"error":  "Tooltip not found",
		})
	}

	log.Printf("[PERF] Load tooltip %s in %v", id, time.Since(start))
	return c.JSON(fiber.Map{
		"status": true,
		"data": fiber.Map{
			"tooltip_id":                tooltip.TooltipID,
			"tooltip_keyword":           tooltip.TooltipKeyword,
			"tooltip_description_short": tooltip.TooltipDescriptionShort,
			"tooltip_description_long":  tooltip.TooltipDescriptionLong,
		},
	})
}

// GetAllTooltips menangani permintaan untuk mendapatkan semua data tooltips
func (tc *TooltipsController) GetAllTooltips(c *fiber.Ctx) error {
	log.Println("Fetching all tooltips")

	var tooltips []model.Tooltip

	// Ambil semua data dari database
	if err := tc.DB.Find(&tooltips).Error; err != nil {
		log.Println("Error fetching tooltips:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch tooltips"})
	}

	return c.JSON(tooltips)
}

func (tc *TooltipsController) UpdateTooltip(c *fiber.Ctx) error {
	id := c.Params("id")
	var existing model.Tooltip
	if err := tc.DB.First(&existing, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tooltip not found"})
	}

	var updated model.Tooltip
	if err := c.BodyParser(&updated); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	existing.TooltipKeyword = updated.TooltipKeyword
	existing.TooltipDescriptionShort = updated.TooltipDescriptionShort
	existing.TooltipDescriptionLong = updated.TooltipDescriptionLong

	if err := tc.DB.Save(&existing).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update tooltip"})
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Tooltip with ID %v updated successfully", existing.TooltipID),
		"data":    existing,
	})
}

func (tc *TooltipsController) DeleteTooltip(c *fiber.Ctx) error {
	id := c.Params("id")

	// ✅ Query eksplisit pakai kolom tooltip_id
	var tooltip model.Tooltip
	if err := tc.DB.First(&tooltip, "tooltip_id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tooltip not found"})
	}

	// 🗑️ Hapus berdasarkan ID yang sesuai kolom yang sudah di-declare
	if err := tc.DB.Delete(&tooltip).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete tooltip"})
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Tooltip with ID %s deleted successfully", id),
	})
}
