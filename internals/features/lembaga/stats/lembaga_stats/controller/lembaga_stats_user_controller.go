package controller

import (
	dto "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/dto"
	model "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/model"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GET /api/admin/lembaga-stats/:masjid_id
func (h *LembagaStatsController) GetByMasjidID(c *fiber.Ctx) error {
	masjidIDParam := c.Params("masjid_id")
	if masjidIDParam == "" {
		return fiber.NewError(fiber.StatusBadRequest, "masjid_id wajib diisi")
	}

	masjidID, err := uuid.Parse(masjidIDParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak valid")
	}

	var m model.LembagaStats
	if err := h.DB.First(&m, "lembaga_stats_lembaga_id = ?", masjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": dto.FromModel(m),
	})
}
