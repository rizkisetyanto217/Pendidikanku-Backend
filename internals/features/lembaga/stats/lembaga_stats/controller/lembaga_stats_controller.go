package controller

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/dto"
	model "masjidku_backend/internals/features/lembaga/stats/lembaga_stats/model"
	helper "masjidku_backend/internals/helpers"
)

type LembagaStatsController struct {
	DB *gorm.DB
}

func NewLembagaStatsController(db *gorm.DB) *LembagaStatsController {
	return &LembagaStatsController{DB: db}
}

/* GET /api/a/lembaga-stats  (tenant dari token) */
func (h *LembagaStatsController) GetMyLembagaStats(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var m model.LembagaStats
	if err := h.DB.First(&m, "lembaga_stats_lembaga_id = ?", masjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			empty := model.LembagaStats{LembagaStatsLembagaID: masjidID}
			return c.Status(http.StatusOK).JSON(fiber.Map{
				"data":  dto.FromModel(empty),
				"found": false,
			})
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data":  dto.FromModel(m),
		"found": true,
	})
}

// internals/features/lembaga/stats/lembaga_stats/controller/lembaga_stats_controller.go

/* POST /api/a/lembaga-stats  (create default 0; tenant dari token) */
func (h *LembagaStatsController) CreateMyLembagaStats(c *fiber.Ctx) error {
	// NOTE: samakan import helper vs helpers sesuai project-mu
	masjidID, err := helper.GetMasjidIDFromToken(c) // <- kalau pkg kamu "helper", ganti jadi helper.GetMasjidIDFromToken
	if err != nil {
		return err
	}

	// Cek sudah ada atau belum
	var existing model.LembagaStats
	if err := h.DB.
		First(&existing, "lembaga_stats_lembaga_id = ?", masjidID).Error; err == nil {
		// Sudah ada → kembalikan yang ada (idempotent)
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"message": "Lembaga stats already exists",
			"data":    dto.FromModel(existing),
			"created": false,
		})
	} else if err != gorm.ErrRecordNotFound {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Belum ada → buat baru dengan default 0
	newRow := model.LembagaStats{
		LembagaStatsLembagaID:      masjidID,
		LembagaStatsActiveClasses:  0,
		LembagaStatsActiveSections: 0,
		LembagaStatsActiveStudents: 0,
		LembagaStatsActiveTeachers: 0,
	}
	if err := h.DB.Create(&newRow).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Lembaga stats created",
		"data":    dto.FromModel(newRow),
		"created": true,
	})
}


/* PATCH /api/a/lembaga-stats  (partial update; tenant dari token) */
func (h *LembagaStatsController) PatchMyLembagaStats(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.UpdateLembagaStatsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// validasi minimum (>=0) sederhana
	if req.LembagaStatsActiveClasses != nil && *req.LembagaStatsActiveClasses < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "lembaga_stats_active_classes minimal 0")
	}
	if req.LembagaStatsActiveSections != nil && *req.LembagaStatsActiveSections < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "lembaga_stats_active_sections minimal 0")
	}
	if req.LembagaStatsActiveStudents != nil && *req.LembagaStatsActiveStudents < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "lembaga_stats_active_students minimal 0")
	}
	if req.LembagaStatsActiveTeachers != nil && *req.LembagaStatsActiveTeachers < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "lembaga_stats_active_teachers minimal 0")
	}

	// upsert baseline jika belum ada
	var m model.LembagaStats
	if err := h.DB.First(&m, "lembaga_stats_lembaga_id = ?", masjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			m = model.LembagaStats{LembagaStatsLembagaID: masjidID}
			if err := h.DB.Create(&m).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
		} else {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	// apply & save
	req.ApplyToModel(&m)
	if err := h.DB.Save(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Lembaga stats updated",
		"data":    dto.FromModel(m),
	})
}
