package controller

import (
	"bytes"
	"context"
	"image/jpeg"
	"image/png"
	"log"
	masjidAdminModel "masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	"masjidku_backend/internals/features/masjids/masjids/dto"
	"masjidku_backend/internals/features/masjids/masjids/model"
	userModel "masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateMasjidDKM (versi sederhana + OSS + auto convert WebP)
// - multipart/form-data saja
// - jpeg/jpg/png -> dikonversi ke .webp sebelum upload
// - webp -> langsung upload apa adanya
// CreateMasjidDKM (versi sesuai schema terbaru)
// - multipart/form-data
// - jpeg/jpg/png -> konversi ke .webp sebelum upload
// - webp -> langsung upload

// CreateMasjidDKM — schema terbaru (OSS + auto convert WebP)
func (mc *MasjidController) CreateMasjidDKM(c *fiber.Ctx) error {
	log.Println("[INFO] Received request to create masjid (schema terbaru)")

	// 1) Auth
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	// 2) Content-Type check
	if !strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		return c.Status(415).JSON(fiber.Map{"error": "Gunakan multipart/form-data"})
	}

	// 3) Ambil form
	name := strings.TrimSpace(c.FormValue("masjid_name"))
	if name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nama masjid wajib diisi"})
	}
	baseSlug := helper.GenerateSlug(name)
	if baseSlug == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nama masjid tidak valid untuk slug"})
	}

	bio := c.FormValue("masjid_bio_short")
	location := c.FormValue("masjid_location")
	domain := strings.ToLower(strings.TrimSpace(c.FormValue("masjid_domain")))
	gmapsURL := c.FormValue("masjid_google_maps_url")

	// Optional verif input; default pending
	verifStatus := strings.TrimSpace(c.FormValue("masjid_verification_status"))
	if verifStatus == "" {
		verifStatus = "pending"
	}
	verifNotes := c.FormValue("masjid_verification_notes")

	latStr := c.FormValue("masjid_latitude")
	longStr := c.FormValue("masjid_longitude")

	var latPtr *float64
	var longPtr *float64
	if v, err := strconv.ParseFloat(latStr, 64); err == nil {
		latPtr = &v
	}
	if v, err := strconv.ParseFloat(longStr, 64); err == nil {
		longPtr = &v
	}

	// Sosial
	ig := c.FormValue("masjid_instagram_url")
	wa := c.FormValue("masjid_whatsapp_url")
	yt := c.FormValue("masjid_youtube_url")
	fb := c.FormValue("masjid_facebook_url")
	tiktok := c.FormValue("masjid_tiktok_url")
	waIkhwan := c.FormValue("masjid_whatsapp_group_ikhwan_url")
	waAkhwat := c.FormValue("masjid_whatsapp_group_akhwat_url")

	// Plan opsional
	var planIDPtr *uuid.UUID
	if planIDStr := c.FormValue("masjid_current_plan_id"); planIDStr != "" {
		if parsed, err := uuid.Parse(planIDStr); err == nil {
			planIDPtr = &parsed
		}
	}

	// 4) Transaksi
	var respDTO dto.MasjidResponse
	txErr := mc.DB.Transaction(func(tx *gorm.DB) error {
		// a) Unik slug
		slug, err := helper.EnsureUniqueSlug(tx, baseSlug, "masjids", "masjid_slug")
		if err != nil {
			return fiber.NewError(500, "Gagal membuat slug unik")
		}

		newID := uuid.New()

		// b) Upload gambar (jpeg/png -> webp; webp pass-through)
		var imageURL string
		if file, ferr := c.FormFile("masjid_image_url"); ferr == nil && file != nil {
			ext := strings.ToLower(filepath.Ext(file.Filename))
			const maxBytes = 5 * 1024 * 1024
			if file.Size > maxBytes {
				return fiber.NewError(fiber.StatusRequestEntityTooLarge, "Ukuran gambar maksimal 5MB")
			}

			svc, err := helperOSS.NewOSSServiceFromEnv("") // root bucket sesuai env
			if err != nil {
				return fiber.NewError(500, "OSS init gagal")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			keyPrefix := "masjids/" + newID.String() + "/images"
			baseName := helper.GenerateSlug(strings.TrimSuffix(file.Filename, ext))
			if baseName == "" {
				baseName = "image"
			}
			key := keyPrefix + "/" + baseName + "_" + time.Now().Format("20060102_150405") + ".webp"

			src, err := file.Open()
			if err != nil {
				return fiber.NewError(400, "Gagal membuka file upload")
			}
			defer src.Close()

			var webpBuf *bytes.Buffer
			switch ext {
			case ".jpg", ".jpeg":
				img, derr := jpeg.Decode(src)
				if derr != nil {
					return fiber.NewError(415, "File JPEG tidak valid")
				}
				webpBuf = new(bytes.Buffer)
				if err := webp.Encode(webpBuf, img, &webp.Options{Lossless: false, Quality: 85}); err != nil {
					return fiber.NewError(500, "Gagal konversi JPEG ke WebP")
				}
			case ".png":
				img, derr := png.Decode(src)
				if derr != nil {
					return fiber.NewError(415, "File PNG tidak valid")
				}
				webpBuf = new(bytes.Buffer)
				if err := webp.Encode(webpBuf, img, &webp.Options{Lossless: false, Quality: 85}); err != nil {
					return fiber.NewError(500, "Gagal konversi PNG ke WebP")
				}
			case ".webp":
				all := new(bytes.Buffer)
				if _, err := all.ReadFrom(src); err != nil {
					return fiber.NewError(400, "Gagal membaca file WebP")
				}
				webpBuf = all
			default:
				return fiber.NewError(fiber.StatusUnsupportedMediaType, "Format tidak didukung (jpg, jpeg, png, webp)")
			}

			if err := svc.UploadStream(ctx, key, bytes.NewReader(webpBuf.Bytes()), "image/webp", true, true); err != nil {
				return fiber.NewError(500, "Gagal upload gambar ke OSS")
			}
			imageURL = svc.PublicURL(key)
		} else if v := strings.TrimSpace(c.FormValue("masjid_image_url")); v != "" {
			// Jika tidak upload file, boleh kirim URL langsung (akan dipakai apa adanya)
			imageURL = v
		}

		// c) Domain pointer (kosong → NULL), case-insensitive
		var domainPtr *string
		if domain != "" {
			domainPtr = &domain
		}

		// d) Simpan masjid
		newMasjid := model.MasjidModel{
			MasjidID:               newID,
			MasjidName:             name,
			MasjidBioShort:         bio,
			MasjidLocation:         location,
			MasjidDomain:           domainPtr,
			MasjidLatitude:         latPtr,
			MasjidLongitude:        longPtr,
			MasjidGoogleMapsURL:    gmapsURL,
			MasjidImageURL:         imageURL,
			MasjidSlug:             slug,

			// ⚠️ Jangan set IsVerified manual — trigger akan urus
			MasjidIsActive:           true,
			MasjidVerificationStatus: verifStatus,  // default "pending"
			MasjidVerificationNotes:  verifNotes,   // opsional
			MasjidCurrentPlanID:      planIDPtr,

			// Sosial
			MasjidInstagramURL:           ig,
			MasjidWhatsappURL:            wa,
			MasjidYoutubeURL:             yt,
			MasjidFacebookURL:            fb,
			MasjidTiktokURL:              tiktok,
			MasjidWhatsappGroupIkhwanURL: waIkhwan,
			MasjidWhatsappGroupAkhwatURL: waAkhwat,
		}
		if err := tx.Create(&newMasjid).Error; err != nil {
			return fiber.NewError(500, "Gagal menyimpan masjid")
		}

		// e) Jadikan pembuat sebagai admin masjid
		admin := masjidAdminModel.MasjidAdminModel{
			MasjidAdminID:       uuid.New(),
			MasjidAdminMasjidID: newMasjid.MasjidID,
			MasjidAdminUserID:   userID,
			MasjidAdminIsActive: true,
		}
		if err := tx.Create(&admin).Error; err != nil {
			return fiber.NewError(500, "Gagal membuat admin masjid")
		}

		// f) Upgrade role user menjadi dkm jika masih "user"
		if err := tx.Model(&userModel.UserModel{}).
			Where("id = ? AND role = ?", userID, "user").
			Update("role", "dkm").Error; err != nil {
			return fiber.NewError(500, "Gagal upgrade role user")
		}

		// g) Build response DTO
		respDTO = dto.FromModelMasjid(&newMasjid)
		return nil
	})

	if txErr != nil {
		if fe, ok := txErr.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Transaksi gagal"})
	}

	log.Printf("[SUCCESS] Masjid created & admin assigned for user %s\n", userID)
	return c.Status(201).JSON(fiber.Map{
		"message": "Masjid berhasil dibuat",
		"data":    respDTO,
	})
}
