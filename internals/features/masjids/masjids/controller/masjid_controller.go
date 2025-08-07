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

// 🟢 GET ALL MASJIDS
func (mc *MasjidController) GetAllMasjids(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all masjids")

	var masjids []model.MasjidModel
	if err := mc.DB.Find(&masjids).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch masjids: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal mengambil data masjid",
		})
	}

	log.Printf("[SUCCESS] Retrieved %d masjids\n", len(masjids))

	// 🔁 Transform ke DTO
	var masjidDTOs []dto.MasjidResponse
	for _, m := range masjids {
		masjidDTOs = append(masjidDTOs, dto.FromModelMasjid(&m))
	}

	return c.JSON(fiber.Map{
		"message": "Data semua masjid berhasil diambil",
		"total":   len(masjidDTOs),
		"data":    masjidDTOs,
	})
}

// 🟢 GET VERIFIED MASJIDS
func (mc *MasjidController) GetAllVerifiedMasjids(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all verified masjids")

	var masjids []model.MasjidModel
	if err := mc.DB.Where("masjid_is_verified = ?", true).Find(&masjids).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch verified masjids: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal mengambil data masjid terverifikasi",
		})
	}

	log.Printf("[SUCCESS] Retrieved %d verified masjids\n", len(masjids))

	// 🔁 Transform ke DTO
	var masjidDTOs []dto.MasjidResponse
	for _, m := range masjids {
		masjidDTOs = append(masjidDTOs, dto.FromModelMasjid(&m))
	}

	return c.JSON(fiber.Map{
		"message": "Data masjid terverifikasi berhasil diambil",
		"total":   len(masjidDTOs),
		"data":    masjidDTOs,
	})
}

// 🟢 GET VERIFIED MASJID BY ID
func (mc *MasjidController) GetVerifiedMasjidByID(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Printf("[INFO] Fetching verified masjid with ID: %s\n", id)

	masjidUUID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Format ID tidak valid",
		})
	}

	var masjid model.MasjidModel
	if err := mc.DB.
		Where("masjid_id = ? AND masjid_is_verified = ?", masjidUUID, true).
		First(&masjid).Error; err != nil {
		log.Printf("[ERROR] Verified masjid with ID %s not found\n", id)
		return c.Status(404).JSON(fiber.Map{
			"error": "Masjid terverifikasi tidak ditemukan",
		})
	}

	log.Printf("[SUCCESS] Retrieved verified masjid: %s\n", masjid.MasjidName)

	masjidDTO := dto.FromModelMasjid(&masjid)
	return c.JSON(fiber.Map{
		"message": "Data masjid terverifikasi berhasil diambil",
		"data":    masjidDTO,
	})
}


// 🟢 GET MASJID BY SLUG
func (mc *MasjidController) GetMasjidBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	log.Printf("[INFO] Fetching masjid with slug: %s\n", slug)

	var masjid model.MasjidModel
	if err := mc.DB.Where("masjid_slug = ?", slug).First(&masjid).Error; err != nil {
		log.Printf("[ERROR] Masjid with slug %s not found\n", slug)
		return c.Status(404).JSON(fiber.Map{
			"error": "Masjid tidak ditemukan",
		})
	}

	log.Printf("[SUCCESS] Retrieved masjid: %s\n", masjid.MasjidName)

	// 🔁 Transform ke DTO
	masjidDTO := dto.FromModelMasjid(&masjid)

	return c.JSON(fiber.Map{
		"message": "Data masjid berhasil diambil",
		"data":    masjidDTO,
	})
}


func (mc *MasjidController) CreateMasjid(c *fiber.Ctx) error {
	log.Println("[INFO] Received request to create masjid")

	if strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		name := c.FormValue("masjid_name")
		bio := c.FormValue("masjid_bio_short")
		location := c.FormValue("masjid_location")
		domain := c.FormValue("masjid_domain")
		slug := helper.GenerateSlug(name)
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

		if name == "" || slug == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "Nama masjid dan slug wajib diisi",
			})
		}

		// ✅ Upload gambar jika ada
		var imageURL string
		file, err := c.FormFile("masjid_image_url")
		if err != nil {
			log.Printf("[DEBUG] Tidak ada file masjid_image_url: %v", err)
		} else if file != nil {
			log.Printf("[DEBUG] File masjid_image_url ditemukan: %s (%d bytes)", file.Filename, file.Size)

			url, err := helper.UploadImageToSupabase("masjids", file)
			if err != nil {
				log.Printf("[ERROR] Gagal upload gambar: %v", err)
				return c.Status(500).JSON(fiber.Map{
					"error": "Gagal upload gambar masjid",
				})
			}
			imageURL = url
		}

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

		if err := mc.DB.Create(&newMasjid).Error; err != nil {
			log.Printf("[ERROR] Failed to create masjid: %v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Gagal menyimpan masjid",
			})
		}

		log.Printf("[SUCCESS] Masjid created: %s\n", newMasjid.MasjidName)

		return c.Status(201).JSON(fiber.Map{
			"message": "Masjid berhasil dibuat",
			"data":    dto.FromModelMasjid(&newMasjid),
		})
	}

	// 🌀 Jika batch insert JSON
	var singleReq dto.MasjidRequest
	var multipleReq []dto.MasjidRequest

	if err := c.BodyParser(&multipleReq); err == nil && len(multipleReq) > 0 {
		var models []model.MasjidModel
		for _, req := range multipleReq {
			model := dto.ToModelMasjid(&req, uuid.New())
			models = append(models, *model)
		}
		if err := mc.DB.Create(&models).Error; err != nil {
			log.Printf("[ERROR] Failed to create multiple masjids: %v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Gagal menyimpan banyak masjid",
			})
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

	// 🧍 Jika single JSON
	if err := c.BodyParser(&singleReq); err != nil {
		log.Printf("[ERROR] Invalid single input: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Format input tidak valid",
		})
	}

	singleModel := dto.ToModelMasjid(&singleReq, uuid.New())
	if err := mc.DB.Create(&singleModel).Error; err != nil {
		log.Printf("[ERROR] Failed to create masjid: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal menyimpan masjid",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Masjid berhasil dibuat",
		"data":    dto.FromModelMasjid(singleModel),
	})
}


// 🟢 UPDATE MASJID (Partial Update)
// ✅ PUT /api/a/masjids/:id
func (mc *MasjidController) UpdateMasjid(c *fiber.Ctx) error {
	id := c.Locals("masjid_id")
	if id == nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Masjid ID tidak ditemukan di token",
		})
	}
	idStr := fmt.Sprintf("%v", id)
	masjidUUID, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Format ID tidak valid",
		})
	}

	// 🔍 Ambil data lama
	var existing model.MasjidModel
	if err := mc.DB.First(&existing, "masjid_id = ?", masjidUUID).Error; err != nil {
		log.Printf("[ERROR] Masjid with ID %s not found\n", id)
		return c.Status(404).JSON(fiber.Map{
			"error": "Masjid tidak ditemukan",
		})
	}

	contentType := c.Get("Content-Type")

	// ✅ Update via multipart/form-data
	if strings.Contains(contentType, "multipart/form-data") {
		// Field text
		if val := c.FormValue("masjid_name"); val != "" {
			existing.MasjidName = val
		}
		if val := c.FormValue("masjid_bio_short"); val != "" {
			existing.MasjidBioShort = val
		}
		if val := c.FormValue("masjid_location"); val != "" {
			existing.MasjidLocation = val
		}
		if val := c.FormValue("masjid_slug"); val != "" {
			existing.MasjidSlug = val
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

		// Koordinat
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
			// Hapus gambar lama dari Supabase
			if existing.MasjidImageURL != "" {
				parsed, err := url.Parse(existing.MasjidImageURL)
				if err == nil {
					rawPath := parsed.Path
					prefix := "/storage/v1/object/public/"
					cleaned := strings.TrimPrefix(rawPath, prefix)
					if unescaped, err := url.QueryUnescape(cleaned); err == nil {
						parts := strings.SplitN(unescaped, "/", 2)
						if len(parts) == 2 {
							bucket := parts[0]
							objectPath := parts[1]
							_ = helper.DeleteFromSupabase(bucket, objectPath)
						}
					}
				}
			}

			// Upload baru
			newURL, err := helper.UploadImageToSupabase("masjids", file)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": "Gagal upload gambar baru",
				})
			}
			existing.MasjidImageURL = newURL
		}

	} else {
		// ✅ Update via JSON (application/json)
		var input dto.MasjidRequest
		if err := c.BodyParser(&input); err != nil {
			log.Printf("[ERROR] Invalid JSON input: %v\n", err)
			return c.Status(400).JSON(fiber.Map{
				"error": "Format JSON tidak valid",
			})
		}

		if input.MasjidName != "" {
			existing.MasjidName = input.MasjidName
		}
		if input.MasjidBioShort != "" {
			existing.MasjidBioShort = input.MasjidBioShort
		}
		if input.MasjidLocation != "" {
			existing.MasjidLocation = input.MasjidLocation
		}
		if input.MasjidSlug != "" {
			existing.MasjidSlug = input.MasjidSlug
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
	}

	// 💾 Simpan ke DB
	if err := mc.DB.Save(&existing).Error; err != nil {
		log.Printf("[ERROR] Failed to update masjid: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal memperbarui masjid",
		})
	}

	log.Printf("[SUCCESS] Masjid updated: %s\n", existing.MasjidName)
	return c.JSON(fiber.Map{
		"message": "Masjid berhasil diperbarui",
		"data":    dto.FromModelMasjid(&existing),
	})
}



// 🗑️ DELETE /api/a/masjids/:id
func (mc *MasjidController) DeleteMasjid(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Printf("[INFO] Deleting masjid with ID: %s\n", id)

	// ✅ Validasi UUID
	masjidUUID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Format ID tidak valid",
		})
	}

	// 🔍 Cari data masjid
	var existing model.MasjidModel
	if err := mc.DB.First(&existing, "masjid_id = ?", masjidUUID).Error; err != nil {
		log.Printf("[ERROR] Masjid not found: %v\n", err)
		return c.Status(404).JSON(fiber.Map{
			"error": "Masjid tidak ditemukan",
		})
	}

	// 🧹 Hapus gambar dari Supabase jika ada
	if existing.MasjidImageURL != "" {
		parsed, err := url.Parse(existing.MasjidImageURL)
		if err == nil {
			rawPath := parsed.Path
			prefix := "/storage/v1/object/public/"
			cleaned := strings.TrimPrefix(rawPath, prefix)
			if unescaped, err := url.QueryUnescape(cleaned); err == nil {
				parts := strings.SplitN(unescaped, "/", 2)
				if len(parts) == 2 {
					bucket := parts[0]
					objectPath := parts[1]
					_ = helper.DeleteFromSupabase(bucket, objectPath)
				}
			}
		}
	}

	// 🗑️ Hapus dari DB
	if err := mc.DB.Delete(&existing).Error; err != nil {
		log.Printf("[ERROR] Failed to delete masjid: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Gagal menghapus masjid",
		})
	}

	log.Printf("[SUCCESS] Masjid with ID %s deleted\n", id)

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Masjid dengan ID %s berhasil dihapus", id),
	})
}
