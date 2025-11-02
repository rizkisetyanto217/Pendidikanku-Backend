package controller

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	dto "schoolku_backend/internals/features/lembaga/stats/lembaga_stats/dto"
	model "schoolku_backend/internals/features/lembaga/stats/lembaga_stats/model"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

type LembagaStatsController struct {
	DB *gorm.DB
}

func NewLembagaStatsController(db *gorm.DB) *LembagaStatsController {
	return &LembagaStatsController{DB: db}
}

/* GET /api/a/lembaga-stats  (tenant dari token) */
func (h *LembagaStatsController) GetMyLembagaStats(c *fiber.Ctx) error {
	schoolID, err := helperAuth.GetSchoolIDFromToken(c)
	if err != nil {
		return err
	}

	var m model.LembagaStats
	if err := h.DB.First(&m, "lembaga_stats_school_id = ?", schoolID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			empty := model.LembagaStats{LembagaStatsSchoolID: schoolID}
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
	schoolID, err := helperAuth.GetSchoolIDFromToken(c) // <- kalau pkg kamu "helper", ganti jadi helper.GetSchoolIDFromToken
	if err != nil {
		return err
	}

	// Cek sudah ada atau belum
	var existing model.LembagaStats
	if err := h.DB.
		First(&existing, "lembaga_stats_school_id = ?", schoolID).Error; err == nil {
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
		LembagaStatsSchoolID:       schoolID,
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

// PUT /api/a/lembaga-stats  (update lembaga stats milik tenant dari token)
func (h *LembagaStatsController) UpdateMyLembagaStats(c *fiber.Ctx) error {
	schoolID, err := helperAuth.GetSchoolIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.UpdateLembagaStatsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// validasi minimal (>=0)
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

	// pastikan baseline ada
	var m model.LembagaStats
	if err := h.DB.First(&m, "lembaga_stats_school_id = ?", schoolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			m = model.LembagaStats{LembagaStatsSchoolID: schoolID}
			if err := h.DB.Create(&m).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
		} else {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	// terapkan perubahan
	req.ApplyToModel(&m)

	if err := h.DB.Save(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Lembaga stats updated",
		"data":    dto.FromModel(m),
	})
}
