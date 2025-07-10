package controller

import (
	"fmt"
	"log"
	"net/http"

	"masjidku_backend/internals/features/masjids/masjids/dto"
	"masjidku_backend/internals/features/masjids/masjids/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidProfileController struct {
	DB *gorm.DB
}

func NewMasjidProfileController(db *gorm.DB) *MasjidProfileController {
	return &MasjidProfileController{DB: db}
}

// 🟢 GET PROFILE BY MASJID_ID
func (mpc *MasjidProfileController) GetProfileByMasjidID(c *fiber.Ctx) error {
	masjidIDParam := c.Params("masjid_id")
	log.Printf("[INFO] Fetching profile for masjid ID: %s\n", masjidIDParam)

	// Validasi UUID format
	masjidUUID, err := uuid.Parse(masjidIDParam)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format UUID masjid tidak valid",
		})
	}

	var profile model.MasjidProfileModel
	err = mpc.DB.
		Preload("Masjid"). // preload relasi opsional
		Where("masjid_profile_masjid_id = ?", masjidUUID).
		First(&profile).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[ERROR] Profile not found for masjid ID %s\n", masjidUUID)
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Profil masjid tidak ditemukan",
			})
		}

		log.Printf("[ERROR] Database error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Terjadi kesalahan saat mengambil data profil",
		})
	}

	log.Printf("[SUCCESS] Retrieved profile for masjid ID %s\n", masjidUUID)
	return c.JSON(fiber.Map{
		"message": "Profil masjid berhasil diambil",
		"data":    dto.FromModelMasjidProfile(&profile),
	})
}

func (ctrl *MasjidProfileController) GetByMasjidID(c *fiber.Ctx) error {
	
	type MasjidIDRequest struct {
		MasjidID string `json:"masjid_id"`
	}
	var payload MasjidIDRequest

	// ⛔ Validasi body
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Permintaan tidak valid",
		})
	}

	// ✅ Parse UUID
	masjidUUID, err := uuid.Parse(payload.MasjidID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Format masjid_id tidak valid",
		})
	}

	// 🔍 Ambil data dari database
	var profile model.MasjidProfileModel
	err = ctrl.DB.Where("masjid_profile_masjid_id = ? AND masjid_profile_deleted_at IS NULL", masjidUUID).First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Profil masjid tidak ditemukan",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data profil masjid",
		})
	}

	// ✅ Kirim response DTO
	return c.JSON(dto.FromModelMasjidProfile(&profile))
}

// 🟢 CREATE PROFILE
// 🟢 CREATE MASJID PROFILE (Single or Multiple)
func (mpc *MasjidProfileController) CreateMasjidProfile(c *fiber.Ctx) error {
	log.Println("[INFO] Received request to create masjid profile(s)")

	// Coba baca sebagai array terlebih dahulu
	var multipleInput []dto.MasjidProfileRequest
	if err := c.BodyParser(&multipleInput); err == nil && len(multipleInput) > 0 {
		var insertedProfiles []model.MasjidProfileModel

		for _, input := range multipleInput {
			// Validasi UUID
			masjidUUID, err := uuid.Parse(input.MasjidProfileMasjidID)
			if err != nil || masjidUUID == uuid.Nil {
				log.Printf("[SKIP] Masjid ID tidak valid: %v\n", input.MasjidProfileMasjidID)
				continue
			}

			// Cek duplikat
			var existing model.MasjidProfileModel
			if err := mpc.DB.
				Where("masjid_profile_masjid_id = ?", masjidUUID).
				First(&existing).Error; err == nil {
				log.Printf("[SKIP] Profil untuk masjid %s sudah ada\n", masjidUUID)
				continue
			}

			profile := dto.ToModelMasjidProfile(&input)
			insertedProfiles = append(insertedProfiles, *profile)
		}

		if len(insertedProfiles) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Tidak ada data profil yang valid untuk disimpan",
			})
		}

		// Simpan batch
		if err := mpc.DB.Create(&insertedProfiles).Error; err != nil {
			log.Printf("[ERROR] Gagal menyimpan banyak profil: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal menyimpan banyak profil masjid",
			})
		}

		log.Printf("[SUCCESS] %d profil masjid berhasil dibuat\n", len(insertedProfiles))
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "Berhasil membuat banyak profil masjid",
			"count":   len(insertedProfiles),
		})
	}

	// Fallback ke single insert
	var singleInput dto.MasjidProfileRequest
	if err := c.BodyParser(&singleInput); err != nil {
		log.Printf("[ERROR] Format input tidak valid: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format input tidak valid",
		})
	}

	// Validasi UUID
	masjidUUID, err := uuid.Parse(singleInput.MasjidProfileMasjidID)
	if err != nil || masjidUUID == uuid.Nil {
		log.Printf("[ERROR] Masjid ID tidak valid: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Masjid ID tidak valid atau kosong",
		})
	}

	// Cek duplikat
	var existing model.MasjidProfileModel
	if err := mpc.DB.
		Where("masjid_profile_masjid_id = ?", masjidUUID).
		First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Profil untuk masjid ini sudah ada",
		})
	}

	profile := dto.ToModelMasjidProfile(&singleInput)
	if err := mpc.DB.Create(&profile).Error; err != nil {
		log.Printf("[ERROR] Gagal menyimpan profil: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan profil masjid",
		})
	}

	log.Printf("[SUCCESS] Profil masjid berhasil dibuat untuk masjid ID: %s\n", masjidUUID)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Profil masjid berhasil dibuat",
		"data":    dto.FromModelMasjidProfile(profile),
	})
}

// 🟢 UPDATE PROFILE (Partial Update)
func (mpc *MasjidProfileController) UpdateMasjidProfile(c *fiber.Ctx) error {
	masjidID := c.Params("masjid_id")
	log.Printf("[INFO] Updating profile for masjid ID: %s\n", masjidID)

	// Ambil data lama dari DB
	var existing model.MasjidProfileModel
	if err := mpc.DB.Where("masjid_profile_masjid_id = ?", masjidID).First(&existing).Error; err != nil {
		log.Printf("[ERROR] Masjid profile not found: %s\n", masjidID)
		return c.Status(404).JSON(fiber.Map{
			"error": "Profil masjid tidak ditemukan",
		})
	}

	// Bind request ke map untuk partial update
	var inputMap map[string]interface{}
	if err := c.BodyParser(&inputMap); err != nil {
		log.Printf("[ERROR] Invalid input: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Format input tidak valid",
		})
	}

	// Hindari update kolom penting
	delete(inputMap, "masjid_profile_id")
	delete(inputMap, "masjid_profile_masjid_id")
	delete(inputMap, "masjid_profile_created_at")

	// Lakukan update
	if err := mpc.DB.Model(&existing).Updates(inputMap).Error; err != nil {
		log.Printf("[ERROR] Failed to update profile: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal memperbarui profil masjid",
		})
	}

	log.Printf("[SUCCESS] Updated profile for masjid ID: %s\n", masjidID)
	return c.JSON(fiber.Map{
		"message": "Profil masjid berhasil diperbarui",
		"data":    dto.FromModelMasjidProfile(&existing),
	})
}

// 🟢 DELETE PROFILE
func (mpc *MasjidProfileController) DeleteMasjidProfile(c *fiber.Ctx) error {
	masjidID := c.Params("masjid_id")
	log.Printf("[INFO] Deleting profile for masjid ID: %s\n", masjidID)

	if err := mpc.DB.Where("masjid_profile_masjid_id = ?", masjidID).
		Delete(&model.MasjidProfileModel{}).Error; err != nil {
		log.Printf("[ERROR] Failed to delete masjid profile: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal menghapus profil masjid",
		})
	}

	log.Printf("[SUCCESS] Masjid profile with masjid ID %s deleted\n", masjidID)
	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Profil masjid dengan ID %s berhasil dihapus", masjidID),
	})
}
