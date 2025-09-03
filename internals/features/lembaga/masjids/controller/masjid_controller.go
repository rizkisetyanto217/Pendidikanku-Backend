// file: internals/features/masjids/masjids/controller/masjid_controller.go
package controller

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"image/png"
	"log"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"masjidku_backend/internals/features/lembaga/masjids/dto"
	"masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"
	helperAuth "masjidku_backend/internals/helpers/auth"

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

// =========================
// Helpers (local)
// =========================

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}
func setPtr(dst **string, v string) {
	trim := strings.TrimSpace(v)
	if trim == "" {
		*dst = nil
		return
	}
	val := trim
	*dst = &val
}
func normalizePtr(v string, toLower bool) *string {
	trim := strings.TrimSpace(v)
	if trim == "" {
		return nil
	}
	if toLower {
		l := strings.ToLower(trim)
		return &l
	}
	return &trim
}

// upload & set URL baru + pindahkan URL lama ke trash (spam/) + due 30 hari
func handleImageField(ctx context.Context, svc *helperOSS.OSSService, existing *model.MasjidModel, slot string, file *multipart.FileHeader, directURL string) (string, error) {
	// Ambil pointer ke field sesuai slot
	var (
		curr **string
		trash **string
		due **time.Time
		keyPrefix = "masjids/" + existing.MasjidID.String() + "/images"
	)

	switch slot {
	case "default":
		curr = &existing.MasjidImageURL
		trash = &existing.MasjidImageTrashURL
		due = &existing.MasjidImageDeletePendingUntil
	case "main":
		curr = &existing.MasjidImageMainURL
		trash = &existing.MasjidImageMainTrashURL
		due = &existing.MasjidImageMainDeletePendingUntil
	case "bg":
		curr = &existing.MasjidImageBgURL
		trash = &existing.MasjidImageBgTrashURL
		due = &existing.MasjidImageBgDeletePendingUntil
	default:
		return "", fmt.Errorf("slot gambar tidak dikenal: %s", slot)
	}

	oldURL := strVal(*curr)
	var newURL string

	// Path A: upload file
	if file != nil {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		const maxBytes = 5 * 1024 * 1024
		if file.Size > maxBytes {
			return "", fmt.Errorf("ukuran gambar maksimal 5MB")
		}

		keyBase := helper.GenerateSlug(strings.TrimSuffix(file.Filename, ext))
		if keyBase == "" {
			keyBase = "image"
		}
		key := keyPrefix + "/" + keyBase + "_" + time.Now().Format("20060102_150405") + ".webp"

		src, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("gagal membuka file upload")
		}
		defer src.Close()

		var webpBuf *bytes.Buffer
		switch ext {
		case ".jpg", ".jpeg":
			img, derr := jpeg.Decode(src)
			if derr != nil {
				return "", fmt.Errorf("file JPEG tidak valid")
			}
			webpBuf = new(bytes.Buffer)
			if err := webp.Encode(webpBuf, img, &webp.Options{Lossless: false, Quality: 85}); err != nil {
				return "", fmt.Errorf("gagal konversi JPEG ke WebP")
			}
		case ".png":
			img, derr := png.Decode(src)
			if derr != nil {
				return "", fmt.Errorf("file PNG tidak valid")
			}
			webpBuf = new(bytes.Buffer)
			if err := webp.Encode(webpBuf, img, &webp.Options{Lossless: false, Quality: 85}); err != nil {
				return "", fmt.Errorf("gagal konversi PNG ke WebP")
			}
		case ".webp":
			all := new(bytes.Buffer)
			if _, err := all.ReadFrom(src); err != nil {
				return "", fmt.Errorf("gagal membaca file WebP")
			}
			webpBuf = all
		default:
			return "", fmt.Errorf("format tidak didukung (jpg, jpeg, png, webp)")
		}

		if err := svc.UploadStream(ctx, key, bytes.NewReader(webpBuf.Bytes()), "image/webp", true, true); err != nil {
			return "", fmt.Errorf("gagal upload gambar ke OSS")
		}
		newURL = svc.PublicURL(key)
	} else {
		// Path B: set via direct URL (JSON/form value)
		newURL = strings.TrimSpace(directURL)
	}

	// Jika berubah → pindahkan lama ke spam + set trash & due
	if oldURL != "" && newURL != "" && oldURL != newURL {
		if spamURL, mErr := helperOSS.MoveToSpamByPublicURLENV(oldURL, 15*time.Second); mErr == nil {
			*trash = &spamURL
			d := time.Now().Add(30 * 24 * time.Hour)
			*due = &d
		}
	}
	// set URL baru
	if newURL == "" {
		*curr = nil // clear
	} else {
		setPtr(curr, newURL)
	}
	return newURL, nil
}

// 🟢 UPDATE MASJID (Partial Update) — PUT /api/a/masjids
func (mc *MasjidController) UpdateMasjid(c *fiber.Ctx) error {
	if !helperAuth.IsAdmin(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
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

	// =========================
	// MULTIPART
	// =========================
	if strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		// akses langsung nilai multipart form jika key ada (termasuk empty string)
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
				return "", true
			}
			return vals[0], true
		}

		// strings
		if v := strings.TrimSpace(c.FormValue("masjid_name")); v != "" {
			existing.MasjidName = v
			if newSlug, err := ensureSlugUpdate(v); err == nil {
				existing.MasjidSlug = newSlug
			}
		}
		if raw, ok := getField("masjid_bio_short"); ok {
			existing.MasjidBioShort = normalizePtr(raw, false)
		}
		if raw, ok := getField("masjid_location"); ok {
			existing.MasjidLocation = normalizePtr(raw, false)
		}
		// domain: hanya update kalau key ada di form; "" -> NULL; else lower
		if raw, ok := getField("masjid_domain"); ok {
			existing.MasjidDomain = normalizePtr(raw, true)
		}
		if v := strings.TrimSpace(c.FormValue("masjid_slug")); v != "" {
			if newSlug, err := ensureSlugUpdate(v); err == nil {
				existing.MasjidSlug = newSlug
			}
		}
		if raw, ok := getField("masjid_google_maps_url"); ok {
			existing.MasjidGoogleMapsURL = normalizePtr(raw, false)
		}

		// sosial
		if raw, ok := getField("masjid_instagram_url"); ok {
			existing.MasjidInstagramURL = normalizePtr(raw, false)
		}
		if raw, ok := getField("masjid_whatsapp_url"); ok {
			existing.MasjidWhatsappURL = normalizePtr(raw, false)
		}
		if raw, ok := getField("masjid_youtube_url"); ok {
			existing.MasjidYoutubeURL = normalizePtr(raw, false)
		}
		if raw, ok := getField("masjid_facebook_url"); ok {
			existing.MasjidFacebookURL = normalizePtr(raw, false)
		}
		if raw, ok := getField("masjid_tiktok_url"); ok {
			existing.MasjidTiktokURL = normalizePtr(raw, false)
		}
		if raw, ok := getField("masjid_whatsapp_group_ikhwan_url"); ok {
			existing.MasjidWhatsappGroupIkhwanURL = normalizePtr(raw, false)
		}
		if raw, ok := getField("masjid_whatsapp_group_akhwat_url"); ok {
			existing.MasjidWhatsappGroupAkhwatURL = normalizePtr(raw, false)
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
				existing.MasjidVerificationStatus = model.VerificationStatus(v)
			}
		}
		if raw, ok := getField("masjid_verification_notes"); ok {
			existing.MasjidVerificationNotes = normalizePtr(raw, false)
		}
		if v := c.FormValue("masjid_current_plan_id"); v != "" {
			if planID, err := uuid.Parse(v); err == nil {
				existing.MasjidCurrentPlanID = &planID
			}
		}
		// flag sekolah
		if v := c.FormValue("masjid_is_islamic_school"); v != "" {
			existing.MasjidIsIslamicSchool = v == "true" || v == "1"
		}

		// OSS init (jika ada file)
		var svc *helperOSS.OSSService
		var initErr error
		if fDef, _ := c.FormFile("masjid_image_url"); fDef != nil ||
			func() bool { f, _ := c.FormFile("masjid_image_main_url"); return f != nil }() ||
			func() bool { f, _ := c.FormFile("masjid_image_bg_url"); return f != nil }() {
			svc, initErr = helperOSS.NewOSSServiceFromEnv("")
			if initErr != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "OSS init gagal")
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		// Gambar DEFAULT
		if fh, err := c.FormFile("masjid_image_url"); err == nil && fh != nil {
			if _, err := handleImageField(ctx, svc, &existing, "default", fh, ""); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
		} else if raw, ok := getField("masjid_image_url"); ok {
			if _, err := handleImageField(ctx, nil, &existing, "default", nil, raw); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
		}

		// Gambar MAIN
		if fh, err := c.FormFile("masjid_image_main_url"); err == nil && fh != nil {
			if _, err := handleImageField(ctx, svc, &existing, "main", fh, ""); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
		} else if raw, ok := getField("masjid_image_main_url"); ok {
			if _, err := handleImageField(ctx, nil, &existing, "main", nil, raw); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
		}

		// Gambar BACKGROUND
		if fh, err := c.FormFile("masjid_image_bg_url"); err == nil && fh != nil {
			if _, err := handleImageField(ctx, svc, &existing, "bg", fh, ""); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
		} else if raw, ok := getField("masjid_image_bg_url"); ok {
			if _, err := handleImageField(ctx, nil, &existing, "bg", nil, raw); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
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
			existing.MasjidBioShort = normalizePtr(*input.MasjidBioShort, false)
		}
		if input.MasjidLocation != nil {
			existing.MasjidLocation = normalizePtr(*input.MasjidLocation, false)
		}
		if input.MasjidGoogleMapsURL != nil {
			existing.MasjidGoogleMapsURL = normalizePtr(*input.MasjidGoogleMapsURL, false)
		}
		if input.MasjidLatitude != nil {
			existing.MasjidLatitude = input.MasjidLatitude
		}
		if input.MasjidLongitude != nil {
			existing.MasjidLongitude = input.MasjidLongitude
		}

		// domain: "" → NULL; else lower
		if input.MasjidDomain != nil {
			existing.MasjidDomain = normalizePtr(*input.MasjidDomain, true)
		}

		// Sosial
		if input.MasjidInstagramURL != nil {
			existing.MasjidInstagramURL = normalizePtr(*input.MasjidInstagramURL, false)
		}
		if input.MasjidWhatsappURL != nil {
			existing.MasjidWhatsappURL = normalizePtr(*input.MasjidWhatsappURL, false)
		}
		if input.MasjidYoutubeURL != nil {
			existing.MasjidYoutubeURL = normalizePtr(*input.MasjidYoutubeURL, false)
		}
		if input.MasjidFacebookURL != nil {
			existing.MasjidFacebookURL = normalizePtr(*input.MasjidFacebookURL, false)
		}
		if input.MasjidTiktokURL != nil {
			existing.MasjidTiktokURL = normalizePtr(*input.MasjidTiktokURL, false)
		}
		if input.MasjidWhatsappGroupIkhwanURL != nil {
			existing.MasjidWhatsappGroupIkhwanURL = normalizePtr(*input.MasjidWhatsappGroupIkhwanURL, false)
		}
		if input.MasjidWhatsappGroupAkhwatURL != nil {
			existing.MasjidWhatsappGroupAkhwatURL = normalizePtr(*input.MasjidWhatsappGroupAkhwatURL, false)
		}

		// Flags & verif
		if input.MasjidIsActive != nil {
			existing.MasjidIsActive = *input.MasjidIsActive
		}
		if input.MasjidVerificationStatus != nil {
			v := strings.TrimSpace(*input.MasjidVerificationStatus)
			if v == "pending" || v == "approved" || v == "rejected" {
				existing.MasjidVerificationStatus = model.VerificationStatus(v)
			}
		}
		if input.MasjidVerificationNotes != nil {
			existing.MasjidVerificationNotes = normalizePtr(*input.MasjidVerificationNotes, false)
		}
		if input.MasjidCurrentPlanID != nil {
			existing.MasjidCurrentPlanID = input.MasjidCurrentPlanID
		}
		if input.MasjidIsIslamicSchool != nil {
			existing.MasjidIsIslamicSchool = *input.MasjidIsIslamicSchool
		}

		// Gambar via JSON (default/main/bg)
		// Tidak perlu init OSS; hanya pindah trash old → spam bila URL berubah
		if input.MasjidImageURL != nil {
			if _, err := handleImageField(context.Background(), nil, &existing, "default", nil, *input.MasjidImageURL); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
		}
		if input.MasjidImageMainURL != nil {
			if _, err := handleImageField(context.Background(), nil, &existing, "main", nil, *input.MasjidImageMainURL); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
		}
		if input.MasjidImageBgURL != nil {
			if _, err := handleImageField(context.Background(), nil, &existing, "bg", nil, *input.MasjidImageBgURL); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
			}
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
	isAdmin := helperAuth.IsAdmin(c)
	isOwner := helperAuth.IsOwner(c)

	if !isAdmin && !isOwner {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	pathID := strings.TrimSpace(c.Params("id"))
	var targetID uuid.UUID

	// Masjid ID dari token untuk admin
	var tokenMasjidID uuid.UUID
	var tokenErr error
	if isAdmin {
		tokenMasjidID, tokenErr = helperAuth.GetMasjidIDFromToken(c)
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
			userID, err := helperAuth.GetUserIDFromToken(c)
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

	// Pindahkan semua gambar aktif ke spam/ (biar cron reaper yang hapus kemudian)
	urls := []string{strVal(existing.MasjidImageURL), strVal(existing.MasjidImageMainURL), strVal(existing.MasjidImageBgURL)}
	for _, u := range urls {
		if u == "" {
			continue
		}
		if _, err := helperOSS.MoveToSpamByPublicURLENV(u, 15*time.Second); err != nil {
			log.Printf("[WARN] move to spam gagal: %v\n", err)
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
