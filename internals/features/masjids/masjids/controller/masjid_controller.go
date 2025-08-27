package controller

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"image/png"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"masjidku_backend/internals/features/masjids/masjids/dto"
	"masjidku_backend/internals/features/masjids/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/chai2010/webp"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidController struct {
	DB *gorm.DB
}

func NewMasjidController(db *gorm.DB) *MasjidController {
	return &MasjidController{DB: db}
}


// 🟢 UPDATE MASJID (Partial Update)
// ✅ PUT /api/a/masjids
// 🟢 UPDATE MASJID (Partial Update) — PUT /api/a/masjids
func (mc *MasjidController) UpdateMasjid(c *fiber.Ctx) error {
	if !helper.IsAdmin(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	masjidUUID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var existing model.MasjidModel
	if err := mc.DB.First(&existing, "masjid_id = ?", masjidUUID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	ensureSlugUpdate := func(candidate string) (string, error) {
		base := helper.GenerateSlug(candidate)
		if base == "" {
			return "", fmt.Errorf("slug kosong")
		}
		if base == existing.MasjidSlug {
			return existing.MasjidSlug, nil
		}
		return helper.EnsureUniqueSlug(mc.DB, base, "masjids", "masjid_slug")
	}

	// simpan URL lama untuk cek perubahan gambar
	oldImageURL := strings.TrimSpace(existing.MasjidImageURL)

	// =========================
	// MULTIPART
	// =========================
	if strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		// strings
		if v := strings.TrimSpace(c.FormValue("masjid_name")); v != "" {
			existing.MasjidName = v
			if newSlug, err := ensureSlugUpdate(v); err == nil {
				existing.MasjidSlug = newSlug
			}
		}
		if v := c.FormValue("masjid_bio_short"); v != "" {
			existing.MasjidBioShort = v
		}
		if v := c.FormValue("masjid_location"); v != "" {
			existing.MasjidLocation = v
		}
		// domain: empty -> NULL, else lower
		// --- di dalam if strings.Contains(Content-Type, "multipart/form-data") ---
		mf, _ := c.MultipartForm()
		getField := func(key string) (string, bool) {
			if mf == nil {
				return "", false
			}
			vals, ok := mf.Value[key]
			if !ok {
				return "", false
			}
			if len(vals) == 0 {
				return "", true // key ada tapi tanpa nilai
			}
			return vals[0], true
		}

		// domain: hanya update kalau key ada di form; "" -> NULL; selain itu lower
		if raw, ok := getField("masjid_domain"); ok {
			trimLower := strings.ToLower(strings.TrimSpace(raw))
			if trimLower == "" {
				existing.MasjidDomain = nil
			} else {
				existing.MasjidDomain = &trimLower
			}
		}

		if v := c.FormValue("masjid_slug"); v != "" {
			if newSlug, err := ensureSlugUpdate(v); err == nil {
				existing.MasjidSlug = newSlug
			}
		}
		if v := c.FormValue("masjid_google_maps_url"); v != "" {
			existing.MasjidGoogleMapsURL = v
		}

		// sosial
		if v := c.FormValue("masjid_instagram_url"); v != "" {
			existing.MasjidInstagramURL = v
		}
		if v := c.FormValue("masjid_whatsapp_url"); v != "" {
			existing.MasjidWhatsappURL = v
		}
		if v := c.FormValue("masjid_youtube_url"); v != "" {
			existing.MasjidYoutubeURL = v
		}
		if v := c.FormValue("masjid_facebook_url"); v != "" {
			existing.MasjidFacebookURL = v
		}
		if v := c.FormValue("masjid_tiktok_url"); v != "" {
			existing.MasjidTiktokURL = v
		}
		if v := c.FormValue("masjid_whatsapp_group_ikhwan_url"); v != "" {
			existing.MasjidWhatsappGroupIkhwanURL = v
		}
		if v := c.FormValue("masjid_whatsapp_group_akhwat_url"); v != "" {
			existing.MasjidWhatsappGroupAkhwatURL = v
		}

		// numeric
		if v := c.FormValue("masjid_latitude"); v != "" {
			if lat, err := strconv.ParseFloat(v, 64); err == nil {
				existing.MasjidLatitude = &lat
			}
		}
		if v := c.FormValue("masjid_longitude"); v != "" {
			if lng, err := strconv.ParseFloat(v, 64); err == nil {
				existing.MasjidLongitude = &lng
			}
		}

		// flags & plan
		if v := c.FormValue("masjid_is_active"); v != "" {
			existing.MasjidIsActive = v == "true" || v == "1"
		}
		if v := strings.TrimSpace(c.FormValue("masjid_verification_status")); v != "" {
			if v == "pending" || v == "approved" || v == "rejected" {
				existing.MasjidVerificationStatus = v
			}
		}
		if v := c.FormValue("masjid_verification_notes"); v != "" {
			existing.MasjidVerificationNotes = v
		}
		if v := c.FormValue("masjid_current_plan_id"); v != "" {
			if planID, err := uuid.Parse(v); err == nil {
				existing.MasjidCurrentPlanID = &planID
			}
		}

		// IMAGE (upload & convert ke webp; webp pass-through)
		if file, err := c.FormFile("masjid_image_url"); err == nil && file != nil {
			ext := strings.ToLower(filepath.Ext(file.Filename))
			const maxBytes = 5 * 1024 * 1024
			if file.Size > maxBytes {
				return helper.JsonError(c, fiber.StatusRequestEntityTooLarge, "Ukuran gambar maksimal 5MB")
			}

			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "OSS init gagal")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			keyPrefix := "masjids/" + existing.MasjidID.String() + "/images"
			baseName := helper.GenerateSlug(strings.TrimSuffix(file.Filename, ext))
			if baseName == "" {
				baseName = "image"
			}
			key := keyPrefix + "/" + baseName + "_" + time.Now().Format("20060102_150405") + ".webp"

			src, err := file.Open()
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal membuka file upload")
			}
			defer src.Close()

			var webpBuf *bytes.Buffer
			switch ext {
			case ".jpg", ".jpeg":
				img, derr := jpeg.Decode(src)
				if derr != nil {
					return helper.JsonError(c, fiber.StatusUnsupportedMediaType, "File JPEG tidak valid")
				}
				webpBuf = new(bytes.Buffer)
				if err := webp.Encode(webpBuf, img, &webp.Options{Lossless: false, Quality: 85}); err != nil {
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal konversi JPEG ke WebP")
				}
			case ".png":
				img, derr := png.Decode(src)
				if derr != nil {
					return helper.JsonError(c, fiber.StatusUnsupportedMediaType, "File PNG tidak valid")
				}
				webpBuf = new(bytes.Buffer)
				if err := webp.Encode(webpBuf, img, &webp.Options{Lossless: false, Quality: 85}); err != nil {
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal konversi PNG ke WebP")
				}
			case ".webp":
				all := new(bytes.Buffer)
				if _, err := all.ReadFrom(src); err != nil {
					return helper.JsonError(c, fiber.StatusBadRequest, "Gagal membaca file WebP")
				}
				webpBuf = all
			default:
				return helper.JsonError(c, fiber.StatusUnsupportedMediaType, "Format tidak didukung (jpg, jpeg, png, webp)")
			}

			if err := svc.UploadStream(ctx, key, bytes.NewReader(webpBuf.Bytes()), "image/webp", true, true); err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload gambar ke OSS")
			}
			newURL := svc.PublicURL(key)

			// jika berubah, pindah-in old -> spam + set kolom trash & due
			if oldImageURL != "" && oldImageURL != newURL {
				if spamURL, mErr := helperOSS.MoveToSpamByPublicURLENV(oldImageURL, 15*time.Second); mErr == nil {
					existing.MasjidImageTrashURL = &spamURL
					due := time.Now().Add(30 * 24 * time.Hour)
					existing.MasjidImageDeletePendingUntil = &due
				} // kalau gagal, biarkan saja; bisa di-cleanup manual/trigger fallback
			}
			existing.MasjidImageURL = newURL
		} else if v := strings.TrimSpace(c.FormValue("masjid_image_url")); v != "" {
			// Update via URL langsung
			if oldImageURL != "" && oldImageURL != v {
				if spamURL, mErr := helperOSS.MoveToSpamByPublicURLENV(oldImageURL, 15*time.Second); mErr == nil {
					existing.MasjidImageTrashURL = &spamURL
					due := time.Now().Add(30 * 24 * time.Hour)
					existing.MasjidImageDeletePendingUntil = &due
				}
			}
			existing.MasjidImageURL = v
		}

	// =========================
	// JSON (partial)
	// =========================
	} else {
		var input dto.MasjidUpdateRequest
		if err := c.BodyParser(&input); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Format JSON tidak valid")
		}

		if input.MasjidName != nil {
			existing.MasjidName = *input.MasjidName
			if newSlug, err := ensureSlugUpdate(*input.MasjidName); err == nil {
				existing.MasjidSlug = newSlug
			}
		}
		if input.MasjidSlug != nil {
			if newSlug, err := ensureSlugUpdate(*input.MasjidSlug); err == nil {
				existing.MasjidSlug = newSlug
			}
		}
		if input.MasjidBioShort != nil {
			existing.MasjidBioShort = *input.MasjidBioShort
		}
		if input.MasjidLocation != nil {
			existing.MasjidLocation = *input.MasjidLocation
		}
		if input.MasjidGoogleMapsURL != nil {
			existing.MasjidGoogleMapsURL = *input.MasjidGoogleMapsURL
		}
		if input.MasjidLatitude != nil {
			existing.MasjidLatitude = input.MasjidLatitude
		}
		if input.MasjidLongitude != nil {
			existing.MasjidLongitude = input.MasjidLongitude
		}

		// domain: "" → NULL; else lower
		if input.MasjidDomain != nil {
			trim := strings.TrimSpace(*input.MasjidDomain)
			if trim == "" {
				existing.MasjidDomain = nil
			} else {
				l := strings.ToLower(trim)
				existing.MasjidDomain = &l
			}
		}

		// Sosial
		if input.MasjidInstagramURL != nil {
			existing.MasjidInstagramURL = *input.MasjidInstagramURL
		}
		if input.MasjidWhatsappURL != nil {
			existing.MasjidWhatsappURL = *input.MasjidWhatsappURL
		}
		if input.MasjidYoutubeURL != nil {
			existing.MasjidYoutubeURL = *input.MasjidYoutubeURL
		}
		if input.MasjidFacebookURL != nil {
			existing.MasjidFacebookURL = *input.MasjidFacebookURL
		}
		if input.MasjidTiktokURL != nil {
			existing.MasjidTiktokURL = *input.MasjidTiktokURL
		}
		if input.MasjidWhatsappGroupIkhwanURL != nil {
			existing.MasjidWhatsappGroupIkhwanURL = *input.MasjidWhatsappGroupIkhwanURL
		}
		if input.MasjidWhatsappGroupAkhwatURL != nil {
			existing.MasjidWhatsappGroupAkhwatURL = *input.MasjidWhatsappGroupAkhwatURL
		}

		// Flags & verif
		if input.MasjidIsActive != nil {
			existing.MasjidIsActive = *input.MasjidIsActive
		}
		if input.MasjidVerificationStatus != nil {
			v := strings.TrimSpace(*input.MasjidVerificationStatus)
			if v == "pending" || v == "approved" || v == "rejected" {
				existing.MasjidVerificationStatus = v
			}
		}
		if input.MasjidVerificationNotes != nil {
			existing.MasjidVerificationNotes = *input.MasjidVerificationNotes
		}
		if input.MasjidCurrentPlanID != nil {
			existing.MasjidCurrentPlanID = input.MasjidCurrentPlanID
		}

		// Gambar via JSON:
		if input.MasjidImageURL != nil {
			newURL := strings.TrimSpace(*input.MasjidImageURL) // boleh kosong = clear
			if oldImageURL != "" && oldImageURL != newURL {
				if spamURL, mErr := helperOSS.MoveToSpamByPublicURLENV(oldImageURL, 15*time.Second); mErr == nil {
					existing.MasjidImageTrashURL = &spamURL
					due := time.Now().Add(30 * 24 * time.Hour)
					existing.MasjidImageDeletePendingUntil = &due
				}
			}
			existing.MasjidImageURL = newURL
		}
	}

	if err := mc.DB.Save(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui masjid")
	}
	return helper.JsonOK(c, "Masjid berhasil diperbarui", dto.FromModelMasjid(&existing))
}


// 🗑️ DELETE /api/a/masjids           -> admin: pakai ID token; owner: 400 (perlu :id)
// 🗑️ DELETE /api/a/masjids/:id       -> owner: boleh; admin: hanya jika :id sama dgn ID token
func (mc *MasjidController) DeleteMasjid(c *fiber.Ctx) error {
	isAdmin := helper.IsAdmin(c)
	isOwner := helper.IsOwner(c)

	if !isAdmin && !isOwner {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	pathID := strings.TrimSpace(c.Params("id"))
	var targetID uuid.UUID

	// Masjid ID dari token untuk admin
	var tokenMasjidID uuid.UUID
	var tokenErr error
	if isAdmin {
		tokenMasjidID, tokenErr = helper.GetMasjidIDFromToken(c)
		if tokenErr != nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, tokenErr.Error())
		}
	}

	// Aturan path
	if pathID == "" {
		if isOwner {
			return helper.JsonError(c, fiber.StatusBadRequest, "Owner harus menyertakan ID masjid di path")
		}
		targetID = tokenMasjidID
	} else {
		pathUUID, err := uuid.Parse(pathID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Format ID masjid tidak valid")
		}
		if isAdmin && pathUUID != tokenMasjidID {
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh menghapus masjid di luar scope Anda")
		}
		if isOwner {
			userID, err := helper.GetUserIDFromToken(c)
			if err != nil {
				return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
			}
			var count int64
			if err := mc.DB.
				Table("masjid_admins_teachers").
				Where("masjid_admins_user_id = ? AND masjid_admins_masjid_id = ? AND masjid_admins_is_active = TRUE", userID, pathUUID).
				Count(&count).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memverifikasi kepemilikan")
			}
			if count == 0 {
				return helper.JsonError(c, fiber.StatusForbidden, "Anda bukan owner/admin masjid ini")
			}
		}
		targetID = pathUUID
	}

	log.Printf("[INFO] Deleting masjid ID: %s (isAdmin=%v isOwner=%v)\n", targetID.String(), isAdmin, isOwner)

	// Ambil data existing
	var existing model.MasjidModel
	if err := mc.DB.First(&existing, "masjid_id = ?", targetID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	// Pindahkan gambar aktif ke spam/ (biar cron reaper yang hapus kemudian)
	if existing.MasjidImageURL != "" {
		if _, err := helperOSS.MoveToSpamByPublicURLENV(existing.MasjidImageURL, 15*time.Second); err != nil {
			log.Printf("[WARN] move to spam gagal: %v\n", err)
			// best-effort: lanjutkan soft delete record meski file gagal dipindahkan
		}
	}

	// Soft delete record
	if err := mc.DB.Delete(&existing).Error; err != nil {
		log.Printf("[ERROR] Failed to delete masjid: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus masjid")
	}

	log.Printf("[SUCCESS] Masjid deleted: %s\n", targetID.String())
	return helper.JsonDeleted(c, "Masjid berhasil dihapus", fiber.Map{
		"masjid_id": targetID.String(),
	})
}
