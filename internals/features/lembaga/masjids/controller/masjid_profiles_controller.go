// file: internals/features/lembaga/masjids/controller/masjid_profile_controller.go
package controller

import (
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"

	"masjidku_backend/internals/features/lembaga/masjids/dto"
	"masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

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

// 🟢 CREATE PROFILE (Admin-only, masjid_id dari token)
func (mpc *MasjidProfileController) CreateMasjidProfile(c *fiber.Ctx) error {
	if !helperAuth.IsAdmin(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: hanya admin")
	}

	log.Println("[INFO] Create masjid profile")

	// Ambil masjid_id dari token
	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		// helperAuth kemungkinan sudah mengembalikan *fiber.Error; propagasi apa adanya
		return err
	}

	// Ambil form values
	desc := c.FormValue("masjid_profile_description")
	tahunStr := c.FormValue("masjid_profile_founded_year")
	tahun, _ := strconv.Atoi(strings.TrimSpace(tahunStr))

	// Cek duplikat
	var existing model.MasjidProfileModel
	if err := mpc.DB.
		Where("masjid_profile_masjid_id = ?", masjidUUID).
		First(&existing).Error; err == nil {
		// ketemu → sudah ada
		return helper.JsonError(c, fiber.StatusConflict, "Profil masjid sudah ada")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// error lain → 500
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa profil masjid")
	}
	// kalau ErrRecordNotFound → lanjut create


	// Upload file jika ada
	upload := func(field string) string {
		file, ferr := c.FormFile(field)
		if ferr != nil || file == nil {
			return ""
		}
		uploadedURL, uerr := helper.UploadImageToSupabase("masjids", file)
		if uerr != nil {
			return ""
		}
		return uploadedURL
	}
	logoURL := upload("masjid_profile_logo_url")
	ttdURL := upload("masjid_profile_ttd_ketua_dkm_url")
	stempelURL := upload("masjid_profile_stamp_url")

	// Simpan
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
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan profil")
	}

	return helper.JsonCreated(c, "Profil masjid berhasil dibuat", dto.FromModelMasjidProfile(&profile))
}

// 🟡 UPDATE PROFILE (Admin-only, masjid_id dari token)
func (mpc *MasjidProfileController) UpdateMasjidProfile(c *fiber.Ctx) error {
	if !helperAuth.IsAdmin(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: hanya admin")
	}

	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Update profil masjid: %s\n", masjidUUID.String())

	// Ambil entri lama
	var existing model.MasjidProfileModel
	if err := mpc.DB.
		Where("masjid_profile_masjid_id = ?", masjidUUID).
		First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil masjid tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil profil masjid")
	}

	// Field text
	if v := c.FormValue("masjid_profile_description"); v != "" {
		existing.MasjidProfileDescription = v
	}
	if v := c.FormValue("masjid_profile_founded_year"); v != "" {
		if tahun, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			existing.MasjidProfileFoundedYear = tahun
		}
	}

	// File handler (hapus lama → upload baru; best-effort delete)
	handleFileUpdate := func(fieldName, oldURL string, setter func(string)) error {
		file, ferr := c.FormFile(fieldName)
		if ferr == nil && file != nil {
			if oldURL != "" {
				if parsed, perr := url.Parse(oldURL); perr == nil {
					cleaned := strings.TrimPrefix(parsed.Path, "/storage/v1/object/public/")
					if u, uerr := url.QueryUnescape(cleaned); uerr == nil {
						if parts := strings.SplitN(u, "/", 2); len(parts) == 2 {
							_ = helper.DeleteFromSupabase(parts[0], parts[1])
						}
					}
				}
			}
			newURL, uerr := helper.UploadImageToSupabase("masjids", file)
			if uerr != nil {
				return uerr
			}
			setter(newURL)
		}
		return nil
	}

	if err := handleFileUpdate("masjid_profile_logo_url", existing.MasjidProfileLogoURL, func(u string) {
		existing.MasjidProfileLogoURL = u
	}); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload logo")
	}
	if err := handleFileUpdate("masjid_profile_ttd_ketua_dkm_url", existing.MasjidProfileTTDKetuaDKMURL, func(u string) {
		existing.MasjidProfileTTDKetuaDKMURL = u
	}); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload TTD Ketua DKM")
	}
	if err := handleFileUpdate("masjid_profile_stamp_url", existing.MasjidProfileStampURL, func(u string) {
		existing.MasjidProfileStampURL = u
	}); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload stempel")
	}

	// Simpan
	if err := mpc.DB.Save(&existing).Error; err != nil {
		log.Printf("[ERROR] Gagal update profil masjid: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update profil masjid")
	}

	return helper.JsonUpdated(c, "Profil masjid berhasil diperbarui", dto.FromModelMasjidProfile(&existing))
}

// 🗑️ DELETE /api/a/masjid-profiles         -> pakai ID dari token
// 🗑️ DELETE /api/a/masjid-profiles/:id     -> :id harus sama dengan ID dari token
func (mpc *MasjidProfileController) DeleteMasjidProfile(c *fiber.Ctx) error {
	if !helperAuth.IsAdmin(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: hanya admin yang dapat menghapus profil masjid")
	}

	// token scope
	tokenMasjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// cek optional :id
	targetID := tokenMasjidID
	if s := strings.TrimSpace(c.Params("id")); s != "" {
		pathUUID, perr := uuid.Parse(s)
		if perr != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Format ID pada path tidak valid")
		}
		if pathUUID != tokenMasjidID {
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh menghapus profil di luar scope Anda")
		}
		targetID = pathUUID
	}

	log.Printf("[INFO] Deleting masjid profile for masjid ID: %s\n", targetID.String())

	// cari profil
	var existing model.MasjidProfileModel
	if err := mpc.DB.Where("masjid_profile_masjid_id = ?", targetID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil masjid tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mencari profil masjid")
	}

	// hapus file (best-effort)
	deletePublicURL := func(u string) {
		if u == "" {
			return
		}
		if parsed, perr := url.Parse(u); perr == nil {
			raw := strings.TrimPrefix(parsed.Path, "/storage/v1/object/public/")
			if u2, uerr := url.QueryUnescape(raw); uerr == nil {
				if parts := strings.SplitN(u2, "/", 2); len(parts) == 2 {
					_ = helper.DeleteFromSupabase(parts[0], parts[1])
				}
			}
		}
	}
	deletePublicURL(existing.MasjidProfileLogoURL)
	deletePublicURL(existing.MasjidProfileTTDKetuaDKMURL)
	deletePublicURL(existing.MasjidProfileStampURL)

	// hapus record
	if err := mpc.DB.
		Where("masjid_profile_masjid_id = ?", targetID).
		Delete(&model.MasjidProfileModel{}).Error; err != nil {
		log.Printf("[ERROR] Failed to delete masjid profile: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus profil masjid")
	}

	log.Printf("[SUCCESS] Masjid profile deleted for masjid ID %s\n", targetID.String())
	return helper.JsonDeleted(c, "Profil masjid berhasil dihapus", fiber.Map{
		"masjid_id": targetID.String(),
	})
}
