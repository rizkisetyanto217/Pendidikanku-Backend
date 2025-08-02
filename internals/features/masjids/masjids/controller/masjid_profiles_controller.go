package controller

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"masjidku_backend/internals/features/masjids/masjids/dto"
	"masjidku_backend/internals/features/masjids/masjids/model"
	helper "masjidku_backend/internals/helpers"

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
		// Preload("Masjid"). // preload relasi opsional
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

func (mpc *MasjidProfileController) GetProfileBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	// 1. Cari masjid berdasarkan slug
	var masjid model.MasjidModel
	if err := mpc.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Masjid tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mencari masjid"})
	}

	// 2. Cari profil masjid berdasarkan masjid_id
	var profile model.MasjidProfileModel
	if err := mpc.DB.
		Where("masjid_profile_masjid_id = ?", masjid.MasjidID).
		First(&profile).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profil masjid tidak ditemukan"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil profil masjid"})
	}

	return c.JSON(fiber.Map{
		"message": "Profil masjid berhasil diambil",
		"data":    dto.FromModelMasjidProfile(&profile),
	})
}


func (mpc *MasjidProfileController) CreateMasjidProfile(c *fiber.Ctx) error {
	log.Println("[INFO] Create masjid profile")

	// ✅ Ambil masjid_id dari token
	masjidIDStr, ok := c.Locals("masjid_id").(string)
	if !ok || masjidIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "masjid_id tidak ditemukan di token",
		})
	}

	masjidUUID, err := uuid.Parse(masjidIDStr)
	if err != nil || masjidUUID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "masjid_id dari token tidak valid",
		})
	}

	// ✅ Ambil form values
	desc := c.FormValue("masjid_profile_description")
	tahunStr := c.FormValue("masjid_profile_founded_year")
	tahun, _ := strconv.Atoi(tahunStr)

	// 🔎 Cek duplikat
	var existing model.MasjidProfileModel
	if err := mpc.DB.Where("masjid_profile_masjid_id = ?", masjidUUID).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Profil masjid sudah ada",
		})
	}

	// ⬆️ Upload file jika ada
	upload := func(field string) string {
		file, err := c.FormFile(field)
		if err != nil || file == nil {
			return ""
		}
		url, err := helper.UploadImageToSupabase("masjids", file)
		if err != nil {
			return ""
		}
		return url
	}

	logoURL := upload("masjid_profile_logo_url")
	ttdURL := upload("masjid_profile_ttd_ketua_dkm_url")
	stempelURL := upload("masjid_profile_stamp_url")

	// 💾 Simpan ke DB
	profile := model.MasjidProfileModel{
		MasjidProfileMasjidID:       masjidUUID,
		MasjidProfileDescription:    desc,
		MasjidProfileFoundedYear:    tahun,
		MasjidProfileLogoURL:        logoURL,
		MasjidProfileTTDKetuaDKMURL: ttdURL,
		MasjidProfileStampURL:       stempelURL,
	}

	if err := mpc.DB.Create(&profile).Error; err != nil {
		log.Printf("[ERROR] Gagal simpan profil: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan profil"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Profil masjid berhasil dibuat",
		"data":    dto.FromModelMasjidProfile(&profile),
	})
}


func (mpc *MasjidProfileController) UpdateMasjidProfile(c *fiber.Ctx) error {
	// ✅ Ambil masjid_id dari token
	masjidIDStr, ok := c.Locals("masjid_id").(string)
	if !ok || masjidIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Masjid ID tidak ditemukan di token",
		})
	}
	masjidUUID, err := uuid.Parse(masjidIDStr)
	if err != nil || masjidUUID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Masjid ID tidak valid",
		})
	}

	log.Printf("[INFO] Update profil masjid: %s\n", masjidUUID)

	// 🔍 Cari entri lama
	var existing model.MasjidProfileModel
	if err := mpc.DB.Where("masjid_profile_masjid_id = ?", masjidUUID).First(&existing).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Profil masjid tidak ditemukan"})
	}

	// ✅ Form value
	if val := c.FormValue("masjid_profile_description"); val != "" {
		existing.MasjidProfileDescription = val
	}
	if val := c.FormValue("masjid_profile_founded_year"); val != "" {
		if tahun, err := strconv.Atoi(val); err == nil {
			existing.MasjidProfileFoundedYear = tahun
		}
	}

	// ✅ File handler
	handleFileUpdate := func(fieldName string, oldURL string, setter func(string)) error {
		file, err := c.FormFile(fieldName)
		if err == nil && file != nil {
			if oldURL != "" {
				if parsed, err := url.Parse(oldURL); err == nil {
					cleaned := strings.TrimPrefix(parsed.Path, "/storage/v1/object/public/")
					parts := strings.SplitN(cleaned, "/", 2)
					if len(parts) == 2 {
						_ = helper.DeleteFromSupabase(parts[0], parts[1])
					}
				}
			}
			newURL, err := helper.UploadImageToSupabase("masjids", file)
			if err != nil {
				return err
			}
			setter(newURL)
		}
		return nil
	}

	if err := handleFileUpdate("masjid_profile_logo_url", existing.MasjidProfileLogoURL, func(url string) {
		existing.MasjidProfileLogoURL = url
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal upload logo"})
	}

	if err := handleFileUpdate("masjid_profile_ttd_ketua_dkm_url", existing.MasjidProfileTTDKetuaDKMURL, func(url string) {
		existing.MasjidProfileTTDKetuaDKMURL = url
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal upload TTD Ketua DKM"})
	}

	if err := handleFileUpdate("masjid_profile_stamp_url", existing.MasjidProfileStampURL, func(url string) {
		existing.MasjidProfileStampURL = url
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Gagal upload stempel"})
	}

	// 💾 Simpan
	if err := mpc.DB.Save(&existing).Error; err != nil {
		log.Printf("[ERROR] Gagal update profil masjid: %v\n", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal update profil masjid"})
	}

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
