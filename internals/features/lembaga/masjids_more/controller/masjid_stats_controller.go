package controller

import (
	"log"

	"masjidku_backend/internals/features/lembaga/masjids_more/dto"
	"masjidku_backend/internals/features/lembaga/masjids_more/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidStatsController struct {
	DB *gorm.DB
}

func NewMasjidStatsController(db *gorm.DB) *MasjidStatsController {
	return &MasjidStatsController{DB: db}
}

// ✅ Tambah atau update stats masjid (UPSERT sederhana: jika belum ada → create, jika ada → update)

func (ctrl *MasjidStatsController) UpsertStats(c *fiber.Ctx) error {
	var req dto.MasjidStatsRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Gagal parsing body: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Body tidak valid")
	}
	// FIX: uuid check
	if req.MasjidStatsMasjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id wajib diisi")
	}

	var existing model.MasjidStatsModel
	err := ctrl.DB.WithContext(c.Context()).
		Where("masjid_stats_masjid_id = ?", req.MasjidStatsMasjidID).
		First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// CREATE
			stats := req.ToModel() // asumsi: return *model.MasjidStatsModel
			if err := ctrl.DB.WithContext(c.Context()).Create(stats).Error; err != nil {
				log.Printf("[ERROR] Gagal menyimpan masjid_stats baru: %v", err)
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
			}
			log.Println("[SUCCESS] MasjidStats baru berhasil ditambahkan")
			return helper.JsonCreated(c, "MasjidStats berhasil ditambahkan", dto.ToMasjidStatsResponse(stats))
		}
		// DB error lain
		log.Printf("[ERROR] Gagal mengecek masjid_stats: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memproses permintaan")
	}

	// UPDATE
	existing.MasjidStatsTotalLectures = req.MasjidStatsTotalLectures
	existing.MasjidStatsTotalSessions = req.MasjidStatsTotalSessions
	existing.MasjidStatsTotalParticipants = req.MasjidStatsTotalParticipants
	existing.MasjidStatsTotalDonations = req.MasjidStatsTotalDonations

	if err := ctrl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
		log.Printf("[ERROR] Gagal mengupdate masjid_stats: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate data")
	}

	log.Println("[SUCCESS] MasjidStats berhasil diperbarui")
	return helper.JsonUpdated(c, "MasjidStats berhasil diperbarui", dto.ToMasjidStatsResponse(&existing))
}

// ✅ Ambil stats berdasarkan masjid_id (query param)
func (ctrl *MasjidStatsController) GetStatsByMasjid(c *fiber.Ctx) error {
	masjidID := c.Query("masjid_id")
	if masjidID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id wajib dikirim")
	}

	var stats model.MasjidStatsModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("masjid_stats_masjid_id = ?", masjidID).
		First(&stats).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		log.Printf("[ERROR] Gagal mengambil masjid_stats: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "Berhasil mengambil data masjid_stats", dto.ToMasjidStatsResponse(&stats))
}
