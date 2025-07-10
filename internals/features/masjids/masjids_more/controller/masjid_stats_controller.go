package controller

import (
	"log"
	"masjidku_backend/internals/features/masjids/masjids_more/dto"
	"masjidku_backend/internals/features/masjids/masjids_more/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MasjidStatsController struct {
	DB *gorm.DB
}

func NewMasjidStatsController(db *gorm.DB) *MasjidStatsController {
	return &MasjidStatsController{DB: db}
}

// ✅ Tambah atau update stats masjid
func (ctrl *MasjidStatsController) UpsertStats(c *fiber.Ctx) error {
	var req dto.MasjidStatsRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Gagal parsing body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Body tidak valid",
			"error":   err.Error(),
		})
	}

	// Cek apakah sudah ada stats untuk masjid ini
	var existing model.MasjidStatsModel
	err := ctrl.DB.Where("masjid_stats_masjid_id = ?", req.MasjidStatsMasjidID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Printf("[ERROR] Gagal mengecek masjid_stats: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal memproses permintaan",
			"error":   err.Error(),
		})
	}

	if err == gorm.ErrRecordNotFound {
		// Tambah baru
		stats := req.ToModel()
		if err := ctrl.DB.Create(stats).Error; err != nil {
			log.Printf("[ERROR] Gagal menyimpan masjid_stats baru: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Gagal menyimpan data",
				"error":   err.Error(),
			})
		}
		log.Println("[SUCCESS] MasjidStats baru berhasil ditambahkan")
		return c.JSON(fiber.Map{
			"message": "MasjidStats berhasil ditambahkan",
			"data":    dto.ToMasjidStatsResponse(stats),
		})
	}

	// Update
	existing.MasjidStatsTotalLectures = req.MasjidStatsTotalLectures
	existing.MasjidStatsTotalSessions = req.MasjidStatsTotalSessions
	existing.MasjidStatsTotalParticipants = req.MasjidStatsTotalParticipants
	existing.MasjidStatsTotalDonations = req.MasjidStatsTotalDonations

	if err := ctrl.DB.Save(&existing).Error; err != nil {
		log.Printf("[ERROR] Gagal mengupdate masjid_stats: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengupdate data",
			"error":   err.Error(),
		})
	}
	log.Println("[SUCCESS] MasjidStats berhasil diperbarui")
	return c.JSON(fiber.Map{
		"message": "MasjidStats berhasil diperbarui",
		"data":    dto.ToMasjidStatsResponse(&existing),
	})
}

// ✅ Ambil stats berdasarkan masjid_id
func (ctrl *MasjidStatsController) GetStatsByMasjid(c *fiber.Ctx) error {
	masjidID := c.Query("masjid_id")
	if masjidID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "masjid_id wajib dikirim",
		})
	}

	var stats model.MasjidStatsModel
	if err := ctrl.DB.Where("masjid_stats_masjid_id = ?", masjidID).First(&stats).Error; err != nil {
		log.Printf("[ERROR] Gagal mengambil masjid_stats: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Data tidak ditemukan",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil data masjid_stats",
		"data":    dto.ToMasjidStatsResponse(&stats),
	})
}
