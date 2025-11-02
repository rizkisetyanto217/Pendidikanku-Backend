package controller

import (
	"log"

	"schoolku_backend/internals/features/lembaga/school_yayasans/schools_more/dto"
	"schoolku_backend/internals/features/lembaga/school_yayasans/schools_more/model"
	helper "schoolku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchoolStatsController struct {
	DB *gorm.DB
}

func NewSchoolStatsController(db *gorm.DB) *SchoolStatsController {
	return &SchoolStatsController{DB: db}
}

// ✅ Tambah atau update stats school (UPSERT sederhana: jika belum ada → create, jika ada → update)

func (ctrl *SchoolStatsController) UpsertStats(c *fiber.Ctx) error {
	var req dto.SchoolStatsRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Gagal parsing body: %v", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Body tidak valid")
	}
	// FIX: uuid check
	if req.SchoolStatsSchoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id wajib diisi")
	}

	var existing model.SchoolStatsModel
	err := ctrl.DB.WithContext(c.Context()).
		Where("school_stats_school_id = ?", req.SchoolStatsSchoolID).
		First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// CREATE
			stats := req.ToModel() // asumsi: return *model.SchoolStatsModel
			if err := ctrl.DB.WithContext(c.Context()).Create(stats).Error; err != nil {
				log.Printf("[ERROR] Gagal menyimpan school_stats baru: %v", err)
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
			}
			log.Println("[SUCCESS] SchoolStats baru berhasil ditambahkan")
			return helper.JsonCreated(c, "SchoolStats berhasil ditambahkan", dto.ToSchoolStatsResponse(stats))
		}
		// DB error lain
		log.Printf("[ERROR] Gagal mengecek school_stats: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memproses permintaan")
	}

	// UPDATE
	existing.SchoolStatsTotalLectures = req.SchoolStatsTotalLectures
	existing.SchoolStatsTotalSessions = req.SchoolStatsTotalSessions
	existing.SchoolStatsTotalParticipants = req.SchoolStatsTotalParticipants
	existing.SchoolStatsTotalDonations = req.SchoolStatsTotalDonations

	if err := ctrl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
		log.Printf("[ERROR] Gagal mengupdate school_stats: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate data")
	}

	log.Println("[SUCCESS] SchoolStats berhasil diperbarui")
	return helper.JsonUpdated(c, "SchoolStats berhasil diperbarui", dto.ToSchoolStatsResponse(&existing))
}

// ✅ Ambil stats berdasarkan school_id (query param)
func (ctrl *SchoolStatsController) GetStatsBySchool(c *fiber.Ctx) error {
	schoolID := c.Query("school_id")
	if schoolID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id wajib dikirim")
	}

	var stats model.SchoolStatsModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("school_stats_school_id = ?", schoolID).
		First(&stats).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		log.Printf("[ERROR] Gagal mengambil school_stats: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "Berhasil mengambil data school_stats", dto.ToSchoolStatsResponse(&stats))
}
