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

type MasjidController struct {
	DB *gorm.DB
}

func NewMasjidController(db *gorm.DB) *MasjidController {
	return &MasjidController{DB: db}
}


func (mc *MasjidController) CreateMasjid(c *fiber.Ctx) error {
	log.Println("[INFO] Received request to create masjid")

	// =========================
	// MULTIPART (form-data)
	// =========================
	if strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		name := c.FormValue("masjid_name")
		if strings.TrimSpace(name) == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Nama masjid wajib diisi"})
		}

		// slug otomatis & unik
		baseSlug := helper.GenerateSlug(name)
		if baseSlug == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Nama masjid tidak valid untuk slug"})
		}
		slug, err := helper.EnsureUniqueSlug(mc.DB, baseSlug, "masjids", "masjid_slug")
		if err != nil {
			log.Printf("[ERROR] ensure unique slug: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Gagal membuat slug unik"})
		}

		bio := c.FormValue("masjid_bio_short")
		location := c.FormValue("masjid_location")
		domain := c.FormValue("masjid_domain")
		gmapsURL := c.FormValue("masjid_google_maps_url")
		lat, _ := strconv.ParseFloat(c.FormValue("masjid_latitude"), 64)
		long, _ := strconv.ParseFloat(c.FormValue("masjid_longitude"), 64)

		// 🔗 Sosial Media
		ig := c.FormValue("masjid_instagram_url")
		wa := c.FormValue("masjid_whatsapp_url")
		yt := c.FormValue("masjid_youtube_url")
		fb := c.FormValue("masjid_facebook_url")
		tiktok := c.FormValue("masjid_tiktok_url")
		waIkhwan := c.FormValue("masjid_whatsapp_group_ikhwan_url")
		waAkhwat := c.FormValue("masjid_whatsapp_group_akhwat_url")

		// ✅ Upload gambar jika ada
		var imageURL string
		if file, err := c.FormFile("masjid_image_url"); err == nil && file != nil {
			log.Printf("[DEBUG] File masjid_image_url ditemukan: %s (%d bytes)", file.Filename, file.Size)
			if url, upErr := helper.UploadImageToSupabase("masjids", file); upErr == nil {
				imageURL = url
			} else {
				log.Printf("[ERROR] Gagal upload gambar: %v", upErr)
				return c.Status(500).JSON(fiber.Map{"error": "Gagal upload gambar masjid"})
			}
		} else if err != nil {
			log.Printf("[DEBUG] Tidak ada file masjid_image_url: %v", err)
		}

		// pointer domain jika diisi
		var domainPtr *string
		if domain != "" {
			domainPtr = &domain
		}

		newMasjid := model.MasjidModel{
			MasjidID:                     uuid.New(),
			MasjidName:                   name,
			MasjidBioShort:               bio,
			MasjidLocation:               location,
			MasjidDomain:                 domainPtr,
			MasjidSlug:                   slug, // ← otomatis & unik
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

		if err := mc.DB.Create(&newMasjid).Error; err != nil {
			log.Printf("[ERROR] Failed to create masjid: %v\n", err)
			return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan masjid"})
		}

		log.Printf("[SUCCESS] Masjid created: %s\n", newMasjid.MasjidName)
		return c.Status(201).JSON(fiber.Map{
			"message": "Masjid berhasil dibuat",
			"data":    dto.FromModelMasjid(&newMasjid),
		})
	}

	// =========================
	// JSON (batch / single)
	// =========================
	var singleReq dto.MasjidRequest
	var multipleReq []dto.MasjidRequest

	// ---- Batch JSON ----
	if err := c.BodyParser(&multipleReq); err == nil && len(multipleReq) > 0 {
		var models []model.MasjidModel
		used := map[string]struct{}{} // cegah tabrakan di batch yang sama

		for _, req := range multipleReq {
			m := dto.ToModelMasjid(&req, uuid.New())

			// wajib name
			if strings.TrimSpace(m.MasjidName) == "" {
				return c.Status(400).JSON(fiber.Map{"error": "Nama masjid wajib diisi (batch)"})
			}

			// slug otomatis & unik
			base := helper.GenerateSlug(m.MasjidName)
			if base == "" {
				return c.Status(400).JSON(fiber.Map{"error": "Nama masjid tidak valid untuk slug (batch)"})
			}
			unique, err := helper.EnsureUniqueSlug(mc.DB, base, "masjids", "masjid_slug")
			if err != nil {
				log.Printf("[ERROR] ensure unique slug (batch): %v", err)
				return c.Status(500).JSON(fiber.Map{"error": "Gagal membuat slug unik (batch)"})
			}

			final := unique
			// jika sudah dipakai dalam batch ini, tambahkan increment lokal
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

		if err := mc.DB.Create(&models).Error; err != nil {
			log.Printf("[ERROR] Failed to create multiple masjids: %v\n", err)
			return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan banyak masjid"})
		}

		var responses []dto.MasjidResponse
		for i := range models {
			responses = append(responses, dto.FromModelMasjid(&models[i]))
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

	singleModel := dto.ToModelMasjid(&singleReq, uuid.New())

	if strings.TrimSpace(singleModel.MasjidName) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nama masjid wajib diisi"})
	}

	base := helper.GenerateSlug(singleModel.MasjidName)
	if base == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Nama masjid tidak valid untuk slug"})
	}
	unique, err := helper.EnsureUniqueSlug(mc.DB, base, "masjids", "masjid_slug")
	if err != nil {
		log.Printf("[ERROR] ensure unique slug (single): %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal membuat slug unik"})
	}
	singleModel.MasjidSlug = unique

	if err := mc.DB.Create(&singleModel).Error; err != nil {
		log.Printf("[ERROR] Failed to create masjid: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal menyimpan masjid"})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Masjid berhasil dibuat",
		"data":    dto.FromModelMasjid(singleModel),
	})
}

// 🟢 UPDATE MASJID (Partial Update)
// ✅ PUT /api/a/masjids
func (mc *MasjidController) UpdateMasjid(c *fiber.Ctx) error {
	// (opsional, kalau mau enforce admin di level handler)
	if !helper.IsAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Akses ditolak: hanya admin yang dapat memperbarui masjid"})
	}

	// 🔐 Ambil masjid_id dari token (admin scope)
	masjidUUID, err := helper.GetMasjidIDFromToken(c) // baca LocMasjidAdminIDs
	if err != nil {
		return err // helper sudah return Fiber error 401/400 yang tepat
	}

	// 🔍 Ambil data lama
	var existing model.MasjidModel
	if err := mc.DB.First(&existing, "masjid_id = ?", masjidUUID).Error; err != nil {
		log.Printf("[ERROR] Masjid with ID %s not found\n", masjidUUID.String())
		return c.Status(404).JSON(fiber.Map{"error": "Masjid tidak ditemukan"})
	}

	// util kecil: pastikan slug unik kecuali sama dengan slug existing
	ensureSlugUpdate := func(candidate string) (string, error) {
		base := helper.GenerateSlug(candidate)
		if base == "" {
			return "", fmt.Errorf("slug candidate kosong")
		}
		if base == existing.MasjidSlug {
			return existing.MasjidSlug, nil
		}
		return helper.EnsureUniqueSlug(mc.DB, base, "masjids", "masjid_slug")
	}

	contentType := c.Get("Content-Type")

	// ✅ Update via multipart/form-data
	if strings.Contains(contentType, "multipart/form-data") {
		if val := c.FormValue("masjid_name"); val != "" {
			existing.MasjidName = val
			newSlug, err := ensureSlugUpdate(val)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Nama tidak valid untuk slug"})
			}
			existing.MasjidSlug = newSlug
		}
		if val := c.FormValue("masjid_bio_short"); val != "" {
			existing.MasjidBioShort = val
		}
		if val := c.FormValue("masjid_location"); val != "" {
			existing.MasjidLocation = val
		}
		// override slug eksplisit (tetap sanitize & unik)
		if val := c.FormValue("masjid_slug"); val != "" {
			newSlug, err := ensureSlugUpdate(val)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Slug tidak valid"})
			}
			existing.MasjidSlug = newSlug
		}
		if val := c.FormValue("masjid_google_maps_url"); val != "" {
			existing.MasjidGoogleMapsURL = val
		}
		if val := c.FormValue("masjid_instagram_url"); val != "" {
			existing.MasjidInstagramURL = val
		}
		if val := c.FormValue("masjid_whatsapp_url"); val != "" {
			existing.MasjidWhatsappURL = val
		}
		if val := c.FormValue("masjid_youtube_url"); val != "" {
			existing.MasjidYoutubeURL = val
		}
		if val := c.FormValue("masjid_domain"); val != "" {
			domain := strings.TrimSpace(val)
			existing.MasjidDomain = &domain
		}
		if val := c.FormValue("masjid_facebook_url"); val != "" {
			existing.MasjidFacebookURL = val
		}
		if val := c.FormValue("masjid_tiktok_url"); val != "" {
			existing.MasjidTiktokURL = val
		}
		if val := c.FormValue("masjid_whatsapp_group_ikhwan_url"); val != "" {
			existing.MasjidWhatsappGroupIkhwanURL = val
		}
		if val := c.FormValue("masjid_whatsapp_group_akhwat_url"); val != "" {
			existing.MasjidWhatsappGroupAkhwatURL = val
		}
		if val := c.FormValue("masjid_latitude"); val != "" {
			if lat, err := strconv.ParseFloat(val, 64); err == nil {
				existing.MasjidLatitude = lat
			}
		}
		if val := c.FormValue("masjid_longitude"); val != "" {
			if lng, err := strconv.ParseFloat(val, 64); err == nil {
				existing.MasjidLongitude = lng
			}
		}
		// Upload gambar baru jika ada
		if file, err := c.FormFile("masjid_image_url"); err == nil && file != nil {
			if existing.MasjidImageURL != "" {
				if parsed, err := url.Parse(existing.MasjidImageURL); err == nil {
					raw := strings.TrimPrefix(parsed.Path, "/storage/v1/object/public/")
					if u, err := url.QueryUnescape(raw); err == nil {
						if parts := strings.SplitN(u, "/", 2); len(parts) == 2 {
							_ = helper.DeleteFromSupabase(parts[0], parts[1]) // best-effort
						}
					}
				}
			}
			newURL, err := helper.UploadImageToSupabase("masjids", file)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "Gagal upload gambar baru"})
			}
			existing.MasjidImageURL = newURL
		}
	} else {
		// ✅ Update via JSON
		var input dto.MasjidRequest
		if err := c.BodyParser(&input); err != nil {
			log.Printf("[ERROR] Invalid JSON input: %v\n", err)
			return c.Status(400).JSON(fiber.Map{"error": "Format JSON tidak valid"})
		}

		if input.MasjidName != "" {
			existing.MasjidName = input.MasjidName
			newSlug, err := ensureSlugUpdate(input.MasjidName)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Nama tidak valid untuk slug"})
			}
			existing.MasjidSlug = newSlug
		}
		if input.MasjidBioShort != "" {
			existing.MasjidBioShort = input.MasjidBioShort
		}
		if input.MasjidLocation != "" {
			existing.MasjidLocation = input.MasjidLocation
		}
		if input.MasjidSlug != "" {
			newSlug, err := ensureSlugUpdate(input.MasjidSlug)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Slug tidak valid"})
			}
			existing.MasjidSlug = newSlug
		}
		if input.MasjidGoogleMapsURL != "" {
			existing.MasjidGoogleMapsURL = input.MasjidGoogleMapsURL
		}
		if input.MasjidInstagramURL != "" {
			existing.MasjidInstagramURL = input.MasjidInstagramURL
		}
		if input.MasjidWhatsappURL != "" {
			existing.MasjidWhatsappURL = input.MasjidWhatsappURL
		}
		if input.MasjidYoutubeURL != "" {
			existing.MasjidYoutubeURL = input.MasjidYoutubeURL
		}
		if strings.TrimSpace(input.MasjidDomain) != "" {
			domain := strings.TrimSpace(input.MasjidDomain)
			existing.MasjidDomain = &domain
		}
		if input.MasjidLatitude != 0 {
			existing.MasjidLatitude = input.MasjidLatitude
		}
		if input.MasjidLongitude != 0 {
			existing.MasjidLongitude = input.MasjidLongitude
		}
		if input.MasjidFacebookURL != "" {
			existing.MasjidFacebookURL = input.MasjidFacebookURL
		}
		if input.MasjidTiktokURL != "" {
			existing.MasjidTiktokURL = input.MasjidTiktokURL
		}
		if input.MasjidWhatsappGroupIkhwanURL != "" {
			existing.MasjidWhatsappGroupIkhwanURL = input.MasjidWhatsappGroupIkhwanURL
		}
		if input.MasjidWhatsappGroupAkhwatURL != "" {
			existing.MasjidWhatsappGroupAkhwatURL = input.MasjidWhatsappGroupAkhwatURL
		}
	}

	// 💾 Simpan ke DB
	if err := mc.DB.Save(&existing).Error; err != nil {
		log.Printf("[ERROR] Failed to update masjid: %v\n", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal memperbarui masjid"})
	}

	log.Printf("[SUCCESS] Masjid updated: %s\n", existing.MasjidName)
	return c.JSON(fiber.Map{
		"message": "Masjid berhasil diperbarui",
		"data":    dto.FromModelMasjid(&existing),
	})
}



// 🗑️ DELETE /api/a/masjids           -> admin: pakai ID token; owner: 400 (perlu :id)
// 🗑️ DELETE /api/a/masjids/:id       -> owner: bebas; admin: harus sama dgn ID token
func (mc *MasjidController) DeleteMasjid(c *fiber.Ctx) error {
	// 1) Harus admin atau owner
	if !helper.IsAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Akses ditolak: hanya admin yang dapat menghapus masjid",
		})
	}

	pathID := strings.TrimSpace(c.Params("id"))
	isOwner := helper.IsOwner(c)

	var targetID uuid.UUID
	var err error

	if isOwner {
		// OWNER: harus pakai :id (biar eksplisit)
		if pathID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Owner harus menyertakan ID masjid di path",
			})
		}
		targetID, err = uuid.Parse(pathID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format ID pada path tidak valid"})
		}
	} else {
		// ADMIN biasa: ambil dari token
		tokenID, e := helper.GetMasjidIDFromToken(c)
		if e != nil {
			return e // 401/400 dari helper
		}
		// jika ada :id, wajib sama
		if pathID != "" {
			pathUUID, perr := uuid.Parse(pathID)
			if perr != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format ID pada path tidak valid"})
			}
			if pathUUID != tokenID {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Tidak boleh menghapus masjid di luar scope Anda",
				})
			}
		}
		targetID = tokenID
	}

	log.Printf("[INFO] Deleting masjid ID: %s (owner=%v)\n", targetID.String(), isOwner)

	// 2) Ambil data existing
	var existing model.MasjidModel
	if err := mc.DB.First(&existing, "masjid_id = ?", targetID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Masjid tidak ditemukan"})
	}

	// 3) Hapus file gambar (best-effort)
	if existing.MasjidImageURL != "" {
		if parsed, perr := url.Parse(existing.MasjidImageURL); perr == nil {
			raw := strings.TrimPrefix(parsed.Path, "/storage/v1/object/public/")
			if u, uerr := url.QueryUnescape(raw); uerr == nil {
				if parts := strings.SplitN(u, "/", 2); len(parts) == 2 {
					_ = helper.DeleteFromSupabase(parts[0], parts[1])
				}
			}
		}
	}

	// 4) Hapus record
	if err := mc.DB.Delete(&existing).Error; err != nil {
		log.Printf("[ERROR] Failed to delete masjid: %v\n", err)
		return c.Status(500).JSON(fiber.Map{"error": "Gagal menghapus masjid"})
	}

	log.Printf("[SUCCESS] Masjid deleted: %s\n", targetID.String())
	return c.JSON(fiber.Map{
		"message":   "Masjid berhasil dihapus",
		"masjid_id": targetID.String(),
	})
}
