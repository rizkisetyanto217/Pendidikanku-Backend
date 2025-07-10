package controller

import (
	"fmt"
	"regexp"
	"strings"

	"masjidku_backend/internals/features/utils/tooltips/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type TooltipInjectController struct {
	DB *gorm.DB
}

func NewTooltipInjectController(db *gorm.DB) *TooltipInjectController {
	return &TooltipInjectController{DB: db}
}

// Endpoint utama: menerima teks dan inject ID tooltip ke dalamnya
func (tc *TooltipInjectController) InjectTooltipIDs(c *fiber.Ctx) error {
	var req struct {
		Text string `json:"text"`
	}

	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Text) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": false,
			"error":  "Teks tidak boleh kosong atau format request tidak valid",
		})
	}

	original := req.Text
	processed := tc.replaceWithTooltipIDs(original)

	return c.JSON(fiber.Map{
		"status":   true,
		"original": original,
		"result":   processed,
	})
}

// Ganti kata kunci[] menjadi kata kunci[ID] jika ditemukan di database
func (tc *TooltipInjectController) replaceWithTooltipIDs(text string) string {
	// Cari pola kata[ ] (misalnya: halal[], syariah[])
	re := regexp.MustCompile(`\b(\w+)\[\]`)
	matches := re.FindAllStringSubmatch(text, -1)

	seen := map[string]bool{}

	for _, match := range matches {
		keyword := match[1]
		if seen[keyword] {
			continue
		}
		seen[keyword] = true

		var tooltip model.Tooltip
		if err := tc.DB.Where("tooltip_keyword = ?", keyword).First(&tooltip).Error; err == nil {
			from := keyword + "[]"
			to := fmt.Sprintf("%s[%d]", keyword, tooltip.TooltipID)
			text = strings.Replace(text, from, to, 1)
		}
	}

	return text
}
