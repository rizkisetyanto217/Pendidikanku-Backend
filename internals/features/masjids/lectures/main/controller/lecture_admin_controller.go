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
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar kajian")
	}

	lectureResponses := dto.ToLectureResponseList(lectures)
	return helper.JsonOK(c, "Daftar kajian berhasil diambil", lectureResponses)
}


// 🟢 POST /api/a/lectures
func (ctrl *LectureController) CreateLecture(c *fiber.Ctx) error {
    // pastikan login (helper kamu sudah ada)
    if _, err := helper.GetUserIDFromToken(c); err != nil {
        return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
    }

    // 1) ambil scope dari middleware
    var masjidUUID uuid.UUID
    if v, ok := c.Locals("masjid_id").(string); ok && strings.TrimSpace(v) != "" {
        id, err := uuid.Parse(strings.TrimSpace(v))
        if err != nil {
            return helper.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid (locals.masjid_id)")
        }
        masjidUUID = id
    } else {
        // 2) fallback: dari token (teacher → union → admin)
        if id, err := helper.GetMasjidIDFromTokenPreferTeacher(c); err == nil {
            masjidUUID = id
        } else {
            // 3) fallback terakhir: header/query/body (khusus OWNER tanpa scope)
            scope := strings.TrimSpace(c.Get("X-Masjid-ID"))
            if scope == "" { scope = strings.TrimSpace(c.Query("masjid_id")) }
            if scope == "" { scope = strings.TrimSpace(c.FormValue("masjid_id")) }
            if scope == "" {
                return helper.JsonError(c, fiber.StatusBadRequest,
                    "Masjid ID tidak ter-scope. Kirim X-Masjid-ID / ?masjid_id / body.masjid_id")
            }
            id, err := uuid.Parse(scope)
            if err != nil {
                return helper.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid")
            }
            masjidUUID = id
        }
    }

    title := strings.TrimSpace(c.FormValue("lecture_title"))
    description := c.FormValue("lecture_description")
    isActive := c.FormValue("lecture_is_active") == "true"

    var imageURL *string
    if file, err := c.FormFile("lecture_image_url"); err == nil && file != nil {
        url, err := helper.UploadImageToSupabase("lectures", file)
        if err != nil {
            return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload gambar")
        }
        imageURL = &url
    } else if v := strings.TrimSpace(c.FormValue("lecture_image_url")); v != "" {
        imageURL = &v
    }

    if title == "" {
        return helper.JsonError(c, fiber.StatusBadRequest, "Judul tema kajian wajib diisi")
    }

    newLecture := model.LectureModel{
        LectureTitle:       title,
        LectureSlug:        generateSlugFromTitle(title),
        LectureDescription: description,
        LectureMasjidID:    masjidUUID,
        LectureImageURL:    imageURL,
        LectureIsActive:    isActive,
    }

    if err := ctrl.DB.Create(&newLecture).Error; err != nil {
        return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat tema kajian")
    }
    return helper.JsonCreated(c, "Tema kajian berhasil dibuat", dto.ToLectureResponse(&newLecture))
}


// ✅ GET /api/a/lectures/by-masjid
func (ctrl *LectureController) GetByMasjidID(c *fiber.Ctx) error {
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak valid atau tidak ditemukan")
	}

	var lectures []model.LectureModel
	if err := ctrl.DB.
		Where("lecture_masjid_id = ?", masjidID).
		Order("lecture_created_at DESC").
		Find(&lectures).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data lecture")
	}

	if len(lectures) == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Belum ada lecture untuk masjid ini")
	}

	// Lengkapi teacher name jika kosong
	for i := range lectures {
		if lectures[i].LectureTeachers != nil {
			var teacherList []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			if err := json.Unmarshal(lectures[i].LectureTeachers, &teacherList); err == nil {
				changed := false
				for j, t := range teacherList {
					if t.ID != "" && t.Name == "" {
						var user struct{ UserName string }
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
					if updated, err := json.Marshal(teacherList); err == nil {
						lectures[i].LectureTeachers = updated
					}
				}
			}
		}
	}

	return helper.JsonOK(c, "Daftar lecture berhasil ditemukan", dto.ToLectureResponseList(lectures))
}

// 🟢 GET /api/a/lectures/:id
func (ctrl *LectureController) GetLectureByID(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	var lecture model.LectureModel

	if err := ctrl.DB.First(&lecture, "lecture_id = ?", lectureID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Kajian tidak ditemukan")
	}

	if lecture.LectureTeachers != nil {
		var teacherList []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(lecture.LectureTeachers, &teacherList); err == nil {
			changed := false
			for i, t := range teacherList {
				if t.ID != "" && t.Name == "" {
					var user struct{ UserName string }
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

	return helper.JsonOK(c, "Berhasil mengambil detail kajian", dto.ToLectureResponse(&lecture))
}

// ✅ PUT /api/a/lectures/:id
func (ctrl *LectureController) UpdateLecture(c *fiber.Ctx) error {
	lectureID := c.Params("id")

	var existing model.LectureModel
	if err := ctrl.DB.First(&existing, "lecture_id = ?", lectureID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Tema kajian tidak ditemukan")
	}

	titleChanged := false
	if val := c.FormValue("lecture_title"); val != "" && val != existing.LectureTitle {
		existing.LectureTitle = val
		titleChanged = true
	}
	if val := c.FormValue("lecture_description"); val != "" {
		existing.LectureDescription = val
	}

	regenerate := c.FormValue("regenerate_slug")
	if titleChanged && strings.ToLower(regenerate) != "false" {
		base := generateSlugFromTitle(existing.LectureTitle)
		newSlug, err := uniqueLectureSlug(ctrl.DB, base, existing.LectureMasjidID, existing.LectureID.String())
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
		}
		existing.LectureSlug = newSlug
	}

	if err := ctrl.DB.Save(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update tema kajian")
	}

	return helper.JsonUpdated(c, "Tema kajian berhasil diperbarui", dto.ToLectureResponse(&existing))
}

// 🔴 DELETE /api/a/lectures/:id
func (ctrl *LectureController) DeleteLecture(c *fiber.Ctx) error {
	lectureID := c.Params("id")

	var existing model.LectureModel
	if err := ctrl.DB.First(&existing, "lecture_id = ?", lectureID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Kajian tidak ditemukan")
	}

	if existing.LectureImageURL != nil {
		if parsed, err := url.Parse(*existing.LectureImageURL); err == nil {
			rawPath := strings.TrimPrefix(parsed.Path, "/storage/v1/object/public/")
			if unescaped, err := url.QueryUnescape(rawPath); err == nil {
				parts := strings.SplitN(unescaped, "/", 2)
				if len(parts) == 2 {
					bucket := parts[0]
					objectPath := parts[1]
					_ = helper.DeleteFromSupabase(bucket, objectPath)
				}
			}
		}
	}

	if err := ctrl.DB.Delete(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus kajian")
	}

	return helper.JsonDeleted(c, "Kajian berhasil dihapus", dto.ToLectureResponse(&existing))
}

// ===============================
// Util
// ===============================
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
