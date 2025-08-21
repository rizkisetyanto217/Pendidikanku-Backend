package controller

import (
	"log"

	"masjidku_backend/internals/features/masjids/lectures/main/dto"
	"masjidku_backend/internals/features/masjids/lectures/main/model"
	helper "masjidku_backend/internals/helpers"

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
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	newStats := req.ToModel()
	if err := ctrl.DB.Create(newStats).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan stats: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan statistik")
	}

	return helper.JsonCreated(c, "Statistik berhasil ditambahkan", dto.ToLectureStatsResponse(newStats))
}

// 🔵 GET /api/a/lecture-stats/:lectureId
func (ctrl *LectureStatsController) GetLectureStatsByLectureID(c *fiber.Ctx) error {
	lectureID := c.Params("lectureId")

	var stats model.LectureStatsModel
	if err := ctrl.DB.
		Where("lecture_stats_lecture_id = ?", lectureID).
		First(&stats).Error; err != nil {
		log.Printf("[ERROR] Stats tidak ditemukan untuk lecture_id: %s, err: %v", lectureID, err)
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
	}

	return helper.JsonOK(c, "Berhasil mengambil statistik", dto.ToLectureStatsResponse(&stats))
}

// 🟡 PUT /api/a/lecture-stats/:lectureId
func (ctrl *LectureStatsController) UpdateLectureStats(c *fiber.Ctx) error {
	lectureID := c.Params("lectureId")

	// Parsel payload ke struct model agar bisa pakai GORM Updates (partial)
	var payload model.LectureStatsModel
	if err := c.BodyParser(&payload); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	// Pastikan record-nya ada dulu
	var existing model.LectureStatsModel
	if err := ctrl.DB.
		Where("lecture_stats_lecture_id = ?", lectureID).
		First(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
	}

	// Lakukan partial update berdasarkan field yang diisi di payload
	if err := ctrl.DB.Model(&existing).Updates(payload).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui statistik")
	}

	// Ambil ulang untuk response yang fresh
	if err := ctrl.DB.
		Where("lecture_stats_lecture_id = ?", lectureID).
		First(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data terkini")
	}

	return helper.JsonUpdated(c, "Statistik berhasil diperbarui", dto.ToLectureStatsResponse(&existing))
}
