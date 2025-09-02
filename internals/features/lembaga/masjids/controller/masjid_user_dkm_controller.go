// file: internals/features/masjids/masjids/controller/masjid_controller.go
package controller

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	masjidAdminModel "masjidku_backend/internals/features/lembaga/masjid_admins_teachers/admins_teachers/model"
	"masjidku_backend/internals/features/lembaga/masjids/dto"
	"masjidku_backend/internals/features/lembaga/masjids/model"
	userModel "masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =========================
// Helpers lokal
// =========================

func strPtrOrNil(s string, lower bool) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	if lower {
		l := strings.ToLower(t)
		return &l
	}
	return &t
}

func boolFromForm(v string) bool {
	return v == "true" || v == "1" || strings.ToLower(v) == "yes"
}

// CreateMasjidDKM — schema terbaru (OSS + auto convert WebP + 3 slot gambar)
func (mc *MasjidController) CreateMasjidDKM(c *fiber.Ctx) error {
	log.Println("[INFO] Received request to create masjid (schema terbaru)")

	// 1) Auth
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// 2) Content-Type check
	if !strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		return helper.JsonError(c, fiber.StatusUnsupportedMediaType, "Gunakan multipart/form-data")
	}

	// 3) Ambil form (field inti)
	name := strings.TrimSpace(c.FormValue("masjid_name"))
	if name == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama masjid wajib diisi")
	}
	baseSlug := helper.GenerateSlug(name)
	if baseSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama masjid tidak valid untuk slug")
	}

	bio := c.FormValue("masjid_bio_short")
	location := c.FormValue("masjid_location")
	domain := c.FormValue("masjid_domain")
	gmapsURL := c.FormValue("masjid_google_maps_url")
	isIslamicSchool := boolFromForm(c.FormValue("masjid_is_islamic_school"))

	// Optional: Yayasan & Plan
	var yayasanID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("masjid_yayasan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			yayasanID = &id
		}
	}
	var planIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("masjid_current_plan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			planIDPtr = &id
		}
	}

	// Optional verif input; default pending
	verifStatus := strings.TrimSpace(c.FormValue("masjid_verification_status"))
	if verifStatus == "" {
		verifStatus = "pending"
	}
	verifNotes := c.FormValue("masjid_verification_notes")

	// Koordinat
	var latPtr, longPtr *float64
	if v := strings.TrimSpace(c.FormValue("masjid_latitude")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			latPtr = &f
		}
	}
	if v := strings.TrimSpace(c.FormValue("masjid_longitude")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			longPtr = &f
		}
	}

	// Sosial
	ig := c.FormValue("masjid_instagram_url")
	wa := c.FormValue("masjid_whatsapp_url")
	yt := c.FormValue("masjid_youtube_url")
	fb := c.FormValue("masjid_facebook_url")
	tiktok := c.FormValue("masjid_tiktok_url")
	waIkhwan := c.FormValue("masjid_whatsapp_group_ikhwan_url")
	waAkhwat := c.FormValue("masjid_whatsapp_group_akhwat_url")

	// 4) Transaksi
	var respDTO dto.MasjidResponse
	txErr := mc.DB.Transaction(func(tx *gorm.DB) error {
		// a) Slug unik
		slug, err := helper.EnsureUniqueSlug(tx, baseSlug, "masjids", "masjid_slug")
		if err != nil {
			return fiber.NewError(500, "Gagal membuat slug unik")
		}

		newID := uuid.New()

		// b) Siapkan OSS bila ada file upload
		var (
			svc    *helperOSS.OSSService
			ossErr error
		)
		// cek apakah ada salah satu file gambar
		if f, _ := c.FormFile("masjid_image_url"); f != nil ||
			func() bool { f, _ := c.FormFile("masjid_image_main_url"); return f != nil }() ||
			func() bool { f, _ := c.FormFile("masjid_image_bg_url"); return f != nil }() {
			svc, ossErr = helperOSS.NewOSSServiceFromEnv("")
			if ossErr != nil {
				return fiber.NewError(500, "OSS init gagal")
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		// c) Upload / set URL untuk 3 slot gambar
		var (
			imgDefaultURL *string
			imgMainURL    *string
			imgBgURL      *string
		)

		// DEFAULT
		if fh, err := c.FormFile("masjid_image_url"); err == nil && fh != nil {
			if url, uerr := helperOSS.UploadImageToOSS(ctx, svc, newID, "default", fh); uerr != nil {
				return uerr
			} else {
				imgDefaultURL = &url
			}
		} else if v := strings.TrimSpace(c.FormValue("masjid_image_url")); v != "" {
			imgDefaultURL = strPtrOrNil(v, false)
		}

		// MAIN
		if fh, err := c.FormFile("masjid_image_main_url"); err == nil && fh != nil {
			if url, uerr := helperOSS.UploadImageToOSS(ctx, svc, newID, "main", fh); uerr != nil {
				return uerr
			} else {
				imgMainURL = &url
			}
		} else if v := strings.TrimSpace(c.FormValue("masjid_image_main_url")); v != "" {
			imgMainURL = strPtrOrNil(v, false)
		}

		// BACKGROUND
		if fh, err := c.FormFile("masjid_image_bg_url"); err == nil && fh != nil {
			if url, uerr := helperOSS.UploadImageToOSS(ctx, svc, newID, "bg", fh); uerr != nil {
				return uerr
			} else {
				imgBgURL = &url
			}
		} else if v := strings.TrimSpace(c.FormValue("masjid_image_bg_url")); v != "" {
			imgBgURL = strPtrOrNil(v, false)
		}

		// d) Domain pointer (kosong → NULL), case-insensitive
		domainPtr := strPtrOrNil(domain, true)

		// e) Simpan masjid (pakai pointer-friendly fields)
		newMasjid := model.MasjidModel{
			MasjidID:                newID,
			MasjidYayasanID:         yayasanID,
			MasjidName:              name,
			MasjidBioShort:          strPtrOrNil(bio, false),
			MasjidLocation:          strPtrOrNil(location, false),
			MasjidLatitude:          latPtr,
			MasjidLongitude:         longPtr,
			MasjidGoogleMapsURL:     strPtrOrNil(gmapsURL, false),
			MasjidImageURL:          imgDefaultURL,
			MasjidImageMainURL:      imgMainURL,
			MasjidImageBgURL:        imgBgURL,
			MasjidDomain:            domainPtr,
			MasjidSlug:              slug,
			MasjidIsActive:          true,
			MasjidVerificationStatus: model.VerificationStatus(verifStatus),
			MasjidVerificationNotes:  strPtrOrNil(verifNotes, false),
			MasjidCurrentPlanID:      planIDPtr,
			MasjidIsIslamicSchool:    isIslamicSchool,

			// Sosial
			MasjidInstagramURL:           strPtrOrNil(ig, false),
			MasjidWhatsappURL:            strPtrOrNil(wa, false),
			MasjidYoutubeURL:             strPtrOrNil(yt, false),
			MasjidFacebookURL:            strPtrOrNil(fb, false),
			MasjidTiktokURL:              strPtrOrNil(tiktok, false),
			MasjidWhatsappGroupIkhwanURL: strPtrOrNil(waIkhwan, false),
			MasjidWhatsappGroupAkhwatURL: strPtrOrNil(waAkhwat, false),
		}
		if err := tx.Create(&newMasjid).Error; err != nil {
			return fiber.NewError(500, "Gagal menyimpan masjid")
		}

		// f) Jadikan pembuat sebagai admin masjid
		admin := masjidAdminModel.MasjidAdminModel{
			MasjidAdminID:       uuid.New(),
			MasjidAdminMasjidID: newMasjid.MasjidID,
			MasjidAdminUserID:   userID,
			MasjidAdminIsActive: true,
		}
		if err := tx.Create(&admin).Error; err != nil {
			return fiber.NewError(500, "Gagal membuat admin masjid")
		}

		// g) Upgrade role user menjadi dkm jika masih "user"
		if err := tx.Model(&userModel.UserModel{}).
			Where("id = ? AND role = ?", userID, "user").
			Update("role", "dkm").Error; err != nil {
			return fiber.NewError(500, "Gagal upgrade role user")
		}

		// h) Build response DTO
		respDTO = dto.FromModelMasjid(&newMasjid)
		return nil
	})

	if txErr != nil {
		if fe, ok := txErr.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Transaksi gagal")
	}

	log.Printf("[SUCCESS] Masjid created & admin assigned for user %s\n", userID)
	return helper.JsonCreated(c, "Masjid berhasil dibuat", respDTO)
}
