package controller

import (
	"fmt"
	"log"
	masjidAdminModel "masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	"masjidku_backend/internals/features/masjids/masjids/dto"
	"masjidku_backend/internals/features/masjids/masjids/model"
	userModel "masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"

	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)


func (mc *MasjidController) CreateMasjidDKM(c *fiber.Ctx) error {
	log.Println("[INFO] Received request to create masjid")

	// ✅ Ambil user_id dari token via helper
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return err // fiber.Error 401/400 dari helper
	}

	// =========================
	// MULTIPART (form-data)
	// =========================
	if strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		name := c.FormValue("masjid_name")
		if strings.TrimSpace(name) == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Nama masjid wajib diisi"})
		}

		baseSlug := helper.GenerateSlug(name)
		if baseSlug == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Nama masjid tidak valid untuk slug"})
		}

		bio := c.FormValue("masjid_bio_short")
		location := c.FormValue("masjid_location")
		domain := c.FormValue("masjid_domain")
		gmapsURL := c.FormValue("masjid_google_maps_url")
		lat, _ := strconv.ParseFloat(c.FormValue("masjid_latitude"), 64)
		long, _ := strconv.ParseFloat(c.FormValue("masjid_longitude"), 64)

		// Sosial
		ig := c.FormValue("masjid_instagram_url")
		wa := c.FormValue("masjid_whatsapp_url")
		yt := c.FormValue("masjid_youtube_url")
		fb := c.FormValue("masjid_facebook_url")
		tiktok := c.FormValue("masjid_tiktok_url")
		waIkhwan := c.FormValue("masjid_whatsapp_group_ikhwan_url")
		waAkhwat := c.FormValue("masjid_whatsapp_group_akhwat_url")

		// ✅ Upload file (masjid_image_file) → fallback URL teks (masjid_image_url)
		var imageURL string
		if file, ferr := c.FormFile("masjid_image_file"); ferr == nil && file != nil {
			log.Printf("[DEBUG] File masjid_image_file ditemukan: %s (%d bytes)", file.Filename, file.Size)
			if url, upErr := helper.UploadImageToSupabase("masjids", file); upErr == nil {
				imageURL = url
			} else {
				log.Printf("[ERROR] Gagal upload gambar: %v", upErr)
				return c.Status(500).JSON(fiber.Map{"error": "Gagal upload gambar masjid"})
			}
		} else if v := strings.TrimSpace(c.FormValue("masjid_image_url")); v != "" {
			imageURL = v
		}

		var domainPtr *string
		if domain != "" {
			domainPtr = &domain
		}

		var respDTO dto.MasjidResponse
		txErr := mc.DB.Transaction(func(tx *gorm.DB) error {
			// unique slug
			slug, err := helper.EnsureUniqueSlug(tx, baseSlug, "masjids", "masjid_slug")
			if err != nil {
				log.Printf("[ERROR] ensure unique slug: %v", err)
				return fiber.NewError(500, "Gagal membuat slug unik")
			}

			newMasjid := model.MasjidModel{
				MasjidID:                     uuid.New(),
				MasjidName:                   name,
				MasjidBioShort:               bio,
				MasjidLocation:               location,
				MasjidDomain:                 domainPtr,
				MasjidSlug:                   slug,
				MasjidLatitude:               lat,
				MasjidLongitude:              long,
				MasjidGoogleMapsURL:          gmapsURL,
				MasjidImageURL:               imageURL,
				MasjidInstagramURL:           ig,
				MasjidWhatsappURL:            wa,
				MasjidYoutubeURL:             yt,
				MasjidFacebookURL:            fb,
				MasjidTiktokURL:              tiktok,
				MasjidWhatsappGroupIkhwanURL: waIkhwan,
				MasjidWhatsappGroupAkhwatURL: waAkhwat,
				MasjidIsVerified:             false,
			}
			if err := tx.Create(&newMasjid).Error; err != nil {
				log.Printf("[ERROR] Failed to create masjid: %v", err)
				return fiber.NewError(500, "Gagal menyimpan masjid")
			}

			// 🆕 Init lembaga_stats
			if err := tx.Exec(`
				INSERT INTO lembaga_stats (lembaga_stats_masjid_id)
				VALUES (?)
				ON CONFLICT (lembaga_stats_masjid_id) DO NOTHING
			`, newMasjid.MasjidID).Error; err != nil {
				log.Printf("[ERROR] Init lembaga_stats: %v", err)
				return fiber.NewError(500, "Gagal inisialisasi lembaga_stats")
			}

			// 🆕 Init class_attendance_settings
			if err := tx.Exec(`
				INSERT INTO class_attendance_settings (class_attendance_setting_masjid_id)
				VALUES (?)
				ON CONFLICT (class_attendance_setting_masjid_id) DO NOTHING
			`, newMasjid.MasjidID).Error; err != nil {
				log.Printf("[ERROR] Init class_attendance_settings: %v", err)
				return fiber.NewError(500, "Gagal inisialisasi class_attendance_settings")
			}

			// masjid_admin for current user
			admin := masjidAdminModel.MasjidAdminModel{
				MasjidAdminsID:       uuid.New(),
				MasjidAdminsMasjidID: newMasjid.MasjidID,
				MasjidAdminsUserID:   userID,
				MasjidAdminsIsActive: true,
			}
			if err := tx.Create(&admin).Error; err != nil {
				log.Printf("[ERROR] Failed to create masjid_admin: %v", err)
				return fiber.NewError(500, "Gagal membuat admin masjid")
			}

			// upgrade role user -> dkm (hanya bila masih 'user')
			if err := tx.Model(&userModel.UserModel{}).
				Where("id = ? AND role = ?", userID, "user").
				Update("role", "dkm").Error; err != nil {
				log.Printf("[ERROR] Failed to upgrade user role: %v", err)
				return fiber.NewError(500, "Gagal memperbarui role user")
			}

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

	// =========================
	// JSON (batch / single)
	// =========================
	var singleReq dto.MasjidRequest
	var multipleReq []dto.MasjidRequest

	// ---- Batch JSON ----
	if err := c.BodyParser(&multipleReq); err == nil && len(multipleReq) > 0 {
		used := map[string]struct{}{}
		var responses []dto.MasjidResponse

		txErr := mc.DB.Transaction(func(tx *gorm.DB) error {
			var models []model.MasjidModel

			for _, req := range multipleReq {
				m := dto.ToModelMasjid(&req, uuid.New())
				if strings.TrimSpace(m.MasjidName) == "" {
					return fiber.NewError(400, "Nama masjid wajib diisi (batch)")
				}

				base := helper.GenerateSlug(m.MasjidName)
				if base == "" {
					return fiber.NewError(400, "Nama masjid tidak valid untuk slug (batch)")
				}
				unique, err := helper.EnsureUniqueSlug(tx, base, "masjids", "masjid_slug")
				if err != nil {
					log.Printf("[ERROR] ensure unique slug (batch): %v", err)
					return fiber.NewError(500, "Gagal membuat slug unik (batch)")
				}

				final := unique
				if _, ok := used[final]; ok {
					i := 2
					for {
						try := fmt.Sprintf("%s-%d", base, i)
						if _, ok := used[try]; !ok {
							final = try
							break
						}
						i++
					}
				}
				used[final] = struct{}{}
				m.MasjidSlug = final
				models = append(models, *m)
			}

			if err := tx.Create(&models).Error; err != nil {
				log.Printf("[ERROR] Failed to create multiple masjids: %v", err)
				return fiber.NewError(500, "Gagal menyimpan banyak masjid")
			}

			for i := range models {
				// 🆕 init lembaga_stats
				if err := tx.Exec(`
					INSERT INTO lembaga_stats (lembaga_stats_masjid_id)
					VALUES (?)
					ON CONFLICT (lembaga_stats_masjid_id) DO NOTHING
				`, models[i].MasjidID).Error; err != nil {
					log.Printf("[ERROR] Init lembaga_stats (batch): %v", err)
					return fiber.NewError(500, "Gagal inisialisasi lembaga_stats (batch)")
				}

				// 🆕 init class_attendance_settings
				if err := tx.Exec(`
					INSERT INTO class_attendance_settings (class_attendance_setting_masjid_id)
					VALUES (?)
					ON CONFLICT (class_attendance_setting_masjid_id) DO NOTHING
				`, models[i].MasjidID).Error; err != nil {
					log.Printf("[ERROR] Init class_attendance_settings (batch): %v", err)
					return fiber.NewError(500, "Gagal inisialisasi class_attendance_settings (batch)")
				}

				// admin
				admin := masjidAdminModel.MasjidAdminModel{
					MasjidAdminsID:       uuid.New(),
					MasjidAdminsMasjidID: models[i].MasjidID,
					MasjidAdminsUserID:   userID,
					MasjidAdminsIsActive: true,
				}
				if err := tx.Create(&admin).Error; err != nil {
					log.Printf("[ERROR] Failed to create masjid_admin (batch): %v", err)
					return fiber.NewError(500, "Gagal membuat admin masjid (batch)")
				}

				responses = append(responses, dto.FromModelMasjid(&models[i]))
			}

			// upgrade role sekali
			if err := tx.Model(&userModel.UserModel{}).
				Where("id = ? AND role = ?", userID, "user").
				Update("role", "dkm").Error; err != nil {
				log.Printf("[ERROR] Failed to upgrade user role (batch): %v", err)
				return fiber.NewError(500, "Gagal memperbarui role user (batch)")
			}
			return nil
		})
		if txErr != nil {
			if fe, ok := txErr.(*fiber.Error); ok {
				return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
			}
			return c.Status(500).JSON(fiber.Map{"error": "Transaksi gagal (batch)"})
		}

		return c.Status(201).JSON(fiber.Map{
			"message": "Masjid berhasil dibuat (multiple)",
			"data":    responses,
		})
	}

	// ---- Single JSON ----
	if err := c.BodyParser(&singleReq); err != nil {
		log.Printf("[ERROR] Invalid single input: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": "Format input tidak valid"})
	}

	var respDTO dto.MasjidResponse
	txErr := mc.DB.Transaction(func(tx *gorm.DB) error {
		m := dto.ToModelMasjid(&singleReq, uuid.New())
		if strings.TrimSpace(m.MasjidName) == "" {
			return fiber.NewError(400, "Nama masjid wajib diisi")
		}

		base := helper.GenerateSlug(m.MasjidName)
		if base == "" {
			return fiber.NewError(400, "Nama masjid tidak valid untuk slug")
		}
		unique, err := helper.EnsureUniqueSlug(tx, base, "masjids", "masjid_slug")
		if err != nil {
			log.Printf("[ERROR] ensure unique slug (single): %v", err)
			return fiber.NewError(500, "Gagal membuat slug unik")
		}
		m.MasjidSlug = unique

		if err := tx.Create(&m).Error; err != nil {
			log.Printf("[ERROR] Failed to create masjid: %v", err)
			return fiber.NewError(500, "Gagal menyimpan masjid")
		}

		// 🆕 init lembaga_stats
		if err := tx.Exec(`
			INSERT INTO lembaga_stats (lembaga_stats_masjid_id)
			VALUES (?)
			ON CONFLICT (lembaga_stats_masjid_id) DO NOTHING
		`, m.MasjidID).Error; err != nil {
			log.Printf("[ERROR] Init lembaga_stats (single): %v", err)
			return fiber.NewError(500, "Gagal inisialisasi lembaga_stats")
		}

		// 🆕 init class_attendance_settings
		if err := tx.Exec(`
			INSERT INTO class_attendance_settings (class_attendance_setting_masjid_id)
			VALUES (?)
			ON CONFLICT (class_attendance_setting_masjid_id) DO NOTHING
		`, m.MasjidID).Error; err != nil {
			log.Printf("[ERROR] Init class_attendance_settings (single): %v", err)
			return fiber.NewError(500, "Gagal inisialisasi class_attendance_settings")
		}

		// admin
		admin := masjidAdminModel.MasjidAdminModel{
			MasjidAdminsID:       uuid.New(),
			MasjidAdminsMasjidID: m.MasjidID,
			MasjidAdminsUserID:   userID,
			MasjidAdminsIsActive: true,
		}
		if err := tx.Create(&admin).Error; err != nil {
			log.Printf("[ERROR] Failed to create masjid_admin (single): %v", err)
			return fiber.NewError(500, "Gagal membuat admin masjid")
		}

		// upgrade role
		if err := tx.Model(&userModel.UserModel{}).
			Where("id = ? AND role = ?", userID, "user").
			Update("role", "dkm").Error; err != nil {
			log.Printf("[ERROR] Failed to upgrade user role (single): %v", err)
			return fiber.NewError(500, "Gagal memperbarui role user")
		}

		respDTO = dto.FromModelMasjid(m)
		return nil
	})
	if txErr != nil {
		if fe, ok := txErr.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Transaksi gagal"})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Masjid berhasil dibuat",
		"data":    respDTO,
	})
}
