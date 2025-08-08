package controller

import (
	"encoding/json"
	"fmt"
	"masjidku_backend/internals/features/masjids/lectures/main/dto"
	"masjidku_backend/internals/features/masjids/lectures/main/model"
	helper "masjidku_backend/internals/helpers"
	"net/url"
	"strings"
	"unicode"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LectureController struct {
	DB *gorm.DB
}

func NewLectureController(db *gorm.DB) *LectureController {
	return &LectureController{DB: db}
}

// 🟢 GET /api/a/lectures
func (ctrl *LectureController) GetAllLectures(c *fiber.Ctx) error {
	var lectures []model.LectureModel

	if err := ctrl.DB.Order("lecture_created_at DESC").Find(&lectures).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil daftar kajian",
			"error":   err.Error(),
		})
	}

	// Ubah ke bentuk response DTO
	lectureResponses := make([]dto.LectureResponse, len(lectures))
	for i, l := range lectures {
		lectureResponses[i] = *dto.ToLectureResponse(&l)
	}

	return c.JSON(fiber.Map{
		"message": "Daftar kajian berhasil diambil",
		"data":    lectureResponses,
	})
}


// 🟢 POST /api/a/lectures
func (ctrl *LectureController) CreateLecture(c *fiber.Ctx) error {
	// Validasi user login
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User belum login")
	}

	// Ambil masjid_id dari token
	masjidIDs, ok := c.Locals("masjid_admin_ids").([]string)
	if !ok || len(masjidIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak ditemukan di token")
	}
	masjidID, err := uuid.Parse(masjidIDs[0])
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Masjid ID tidak valid")
	}

	// Ambil nilai dari form-data
	title := c.FormValue("lecture_title")
	description := c.FormValue("lecture_description")
	isActive := c.FormValue("lecture_is_active") == "true"

	// Upload gambar jika ada
	var imageURL *string
	if file, err := c.FormFile("lecture_image_url"); err == nil && file != nil {
		url, err := helper.UploadImageToSupabase("lectures", file)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal upload gambar")
		}
		imageURL = &url
	} else if val := c.FormValue("lecture_image_url"); val != "" {
		imageURL = &val
	}

	// Validasi minimal judul
	if title == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Judul tema kajian wajib diisi")
	}

	// Generate slug dari judul
	slug := generateSlugFromTitle(title)

	// Buat model baru
	newLecture := model.LectureModel{
		LectureTitle:    title,
		LectureSlug:     slug,
		LectureDescription: description,
		LectureMasjidID: masjidID,
		LectureImageURL: imageURL,
		LectureIsActive: isActive,
	}

	// Simpan ke database
	if err := ctrl.DB.Create(&newLecture).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat tema kajian")
	}

	// Kirim response
	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureResponse(&newLecture))
}



// ✅ GET /api/a/lectures/by-masjid
func (ctrl *LectureController) GetByMasjidID(c *fiber.Ctx) error {
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Masjid ID tidak valid atau tidak ditemukan",
		})
	}

	var lectures []model.LectureModel
	if err := ctrl.DB.
		Where("lecture_masjid_id = ?", masjidID).
		Order("lecture_created_at DESC").
		Find(&lectures).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data lecture",
		})
	}

	if len(lectures) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Belum ada lecture untuk masjid ini",
		})
	}

	// Lengkapi teacher name jika kosong
	for i := range lectures {
		if lectures[i].LectureTeachers != nil {
			var teacherList []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}

			// Parse JSON
			if err := json.Unmarshal(lectures[i].LectureTeachers, &teacherList); err == nil {
				changed := false
				for j, t := range teacherList {
					if t.ID != "" && t.Name == "" {
						var user struct {
							UserName string
						}
						if err := ctrl.DB.
							Table("users").
							Select("user_name").
							Where("id = ?", t.ID).
							Scan(&user).Error; err == nil && user.UserName != "" {
							teacherList[j].Name = user.UserName
							changed = true
						}
					}
				}
				if changed {
					updated, err := json.Marshal(teacherList)
					if err == nil {
						lectures[i].LectureTeachers = updated
					}
				}
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Daftar lecture berhasil ditemukan",
		"data":    dto.ToLectureResponseList(lectures),
	})
}


// 🟢 GET /api/a/lectures/:id
func (ctrl *LectureController) GetLectureByID(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	var lecture model.LectureModel

	if err := ctrl.DB.First(&lecture, "lecture_id = ?", lectureID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{
			"message": "Kajian tidak ditemukan",
			"error":   err.Error(),
		})
	}

	// 🔄 Lengkapi nama pengajar jika kosong
	if lecture.LectureTeachers != nil {
		var teacherList []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		if err := json.Unmarshal(lecture.LectureTeachers, &teacherList); err == nil {
			changed := false
			for i, t := range teacherList {
				if t.ID != "" && t.Name == "" {
					var user struct {
						UserName string
					}
					if err := ctrl.DB.
						Table("users").
						Select("user_name").
						Where("id = ?", t.ID).
						Scan(&user).Error; err == nil && user.UserName != "" {
						teacherList[i].Name = user.UserName
						changed = true
					}
				}
			}
			if changed {
				if updated, err := json.Marshal(teacherList); err == nil {
					lecture.LectureTeachers = updated
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil detail kajian",
		"data":    dto.ToLectureResponse(&lecture),
	})
}



// ✅ PUT /api/a/lectures/:id
// ✅ PUT /api/a/lectures/:id
func (ctrl *LectureController) UpdateLecture(c *fiber.Ctx) error {
	lectureID := c.Params("id")

	var existing model.LectureModel
	if err := ctrl.DB.First(&existing, "lecture_id = ?", lectureID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Tema kajian tidak ditemukan")
	}

	titleChanged := false

	// --- fields biasa ---
	if val := c.FormValue("lecture_title"); val != "" && val != existing.LectureTitle {
		existing.LectureTitle = val
		titleChanged = true
	}
	if val := c.FormValue("lecture_description"); val != "" {
		existing.LectureDescription = val
	}

	// --- upload gambar (tetap punyamu) ---
	// ... (kode upload/hapus lama tetap seperti punyamu) ...

	// --- Regenerate slug jika judul berubah ---
	// (optional: hormati flag agar bisa mempertahankan slug lama)
	regenerate := c.FormValue("regenerate_slug")
	if titleChanged && strings.ToLower(regenerate) != "false" {
		base := generateSlugFromTitle(existing.LectureTitle)
		newSlug, err := uniqueLectureSlug(ctrl.DB, base, existing.LectureMasjidID, existing.LectureID.String())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik")
		}
		existing.LectureSlug = newSlug
	}

	if err := ctrl.DB.Save(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update tema kajian")
	}

	return c.JSON(fiber.Map{
		"message": "Tema kajian berhasil diperbarui",
		"data":    dto.ToLectureResponse(&existing),
	})
}




// 🔴 DELETE /api/a/lectures/:id
func (ctrl *LectureController) DeleteLecture(c *fiber.Ctx) error {
	lectureID := c.Params("id")

	// 🔍 Cek dulu apakah kajian ditemukan
	var existing model.LectureModel
	if err := ctrl.DB.First(&existing, "lecture_id = ?", lectureID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Kajian tidak ditemukan")
	}

	// 🗑️ Hapus gambar dari Supabase kalau ada
	if existing.LectureImageURL != nil {
		parsed, err := url.Parse(*existing.LectureImageURL)
		if err == nil {
			rawPath := parsed.Path // /storage/v1/object/public/image/lectures%2Fxxx.png
			prefix := "/storage/v1/object/public/"
			cleaned := strings.TrimPrefix(rawPath, prefix) // image/lectures%2Fxxx.png

			// Decode agar %2F jadi /
			unescaped, err := url.QueryUnescape(cleaned)
			if err == nil {
				parts := strings.SplitN(unescaped, "/", 2)
				if len(parts) == 2 {
					bucket := parts[0]        // image
					objectPath := parts[1]    // lectures/xxx.png
					_ = helper.DeleteFromSupabase(bucket, objectPath)
				}
			}
		}
	}

	// 🔴 Hapus dari database
	if err := ctrl.DB.Delete(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus kajian")
	}

	return c.JSON(fiber.Map{
		"message": "Kajian berhasil dihapus",
	})
}



func generateSlugFromTitle(title string) string {
	title = strings.ToLower(title)
	var b strings.Builder
	lastDash := false
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}


// unik per masjid, hindari bentrok dengan dirinya sendiri saat update
func uniqueLectureSlug(db *gorm.DB, base string, masjidID uuid.UUID, excludeLectureID string) (string, error) {
	slug := base
	var cnt int64
	i := 0

	for {
		q := db.Table("lectures").
			Where("lecture_slug = ? AND lecture_masjid_id = ?", slug, masjidID)
		if excludeLectureID != "" {
			q = q.Where("lecture_id <> ?", excludeLectureID)
		}
		if err := q.Count(&cnt).Error; err != nil {
			return "", err
		}
		if cnt == 0 {
			return slug, nil
		}
		i++
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}
