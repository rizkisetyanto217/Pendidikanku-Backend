package controller

import (
	"net/http"
	dto "schoolku_backend/internals/features/lembaga/stats/lembaga_stats/dto"
	model "schoolku_backend/internals/features/lembaga/stats/lembaga_stats/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GET /api/admin/lembaga-stats/:school_id
func (h *LembagaStatsController) GetBySchoolID(c *fiber.Ctx) error {
	schoolIDParam := c.Params("school_id")
	if schoolIDParam == "" {
		return fiber.NewError(fiber.StatusBadRequest, "school_id wajib diisi")
	}

	schoolID, err := uuid.Parse(schoolIDParam)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "school_id tidak valid")
	}

	var m model.LembagaStats
	if err := h.DB.First(&m, "lembaga_stats_lembaga_id = ?", schoolID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": dto.FromModel(m),
	})
}
