package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/lectures/dto"
	"masjidku_backend/internals/features/masjids/lectures/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureStatsController struct {
	DB *gorm.DB
}

func NewLectureStatsController(db *gorm.DB) *LectureStatsController {
	return &LectureStatsController{DB: db}
}

// 🟢 POST /api/a/lecture-stats
func (ctrl *LectureStatsController) CreateLectureStats(c *fiber.Ctx) error {
	var req dto.LectureStatsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Permintaan tidak valid", "error": err.Error()})
	}

	newStats := req.ToModel()
	if err := ctrl.DB.Create(newStats).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan stats: %v", err)
		return c.Status(500).JSON(fiber.Map{"message": "Gagal menyimpan statistik", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Statistik berhasil ditambahkan",
		"data":    dto.ToLectureStatsResponse(newStats),
	})
}

// 🔵 GET /api/a/lecture-stats/:lectureId
func (ctrl *LectureStatsController) GetLectureStatsByLectureID(c *fiber.Ctx) error {
	lectureID := c.Params("lectureId")

	var stats model.LectureStatsModel
	if err := ctrl.DB.Where("lecture_stats_lecture_id = ?", lectureID).First(&stats).Error; err != nil {
		log.Printf("[ERROR] Stats tidak ditemukan untuk lecture_id: %s", lectureID)
		return c.Status(404).JSON(fiber.Map{"message": "Data tidak ditemukan", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil statistik",
		"data":    dto.ToLectureStatsResponse(&stats),
	})
}

// 🟡 PUT /api/a/lecture-stats/:lectureId
func (ctrl *LectureStatsController) UpdateLectureStats(c *fiber.Ctx) error {
	lectureID := c.Params("lectureId")

	var update model.LectureStatsModel
	if err := c.BodyParser(&update); err != nil {
		return c.Status(400).JSON(fiber.Map{"message": "Permintaan tidak valid", "error": err.Error()})
	}

	if err := ctrl.DB.Model(&model.LectureStatsModel{}).
		Where("lecture_stats_lecture_id = ?", lectureID).
		Updates(update).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "Gagal mengupdate statistik", "error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Statistik berhasil diperbarui",
	})
}
