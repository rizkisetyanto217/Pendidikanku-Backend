package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"masjidku_backend/internals/features/masjids/lectures/main/dto"
	"masjidku_backend/internals/features/masjids/lectures/main/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"
	"net/url"
	"strconv"
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
	// --- auth wajib ---
	if _, err := helper.GetUserIDFromToken(c); err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// --- scope masjid: hanya dari token (prefer teacher -> dkm -> union -> admin) ---
	masjidUUID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidUUID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid atau tidak ter-scope")
	}

	ct := strings.TrimSpace(c.Get(fiber.HeaderContentType))
	isJSON := strings.HasPrefix(ct, fiber.MIMEApplicationJSON)
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	// --- ambil payload (dukung JSON & form) ---
	type reqJSON struct {
		Title       string  `json:"lecture_title"`
		Description string  `json:"lecture_description"`
		IsActive    *bool   `json:"lecture_is_active"`
		ImageURL    *string `json:"lecture_image_url"` // jika JSON kirim URL langsung
	}

	var (
		title       string
		description string
		isActive    bool
		imageURL    *string
	)

	if isJSON {
		var body reqJSON
		if err := c.BodyParser(&body); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "JSON tidak valid")
		}
		title = strings.TrimSpace(body.Title)
		description = body.Description
		if body.IsActive != nil {
			isActive = *body.IsActive
		}
		if body.ImageURL != nil && strings.TrimSpace(*body.ImageURL) != "" {
			u := strings.TrimSpace(*body.ImageURL)
			imageURL = &u
		}
	} else {
		// form-urlencoded / multipart
		title = strings.TrimSpace(c.FormValue("lecture_title"))
		description = c.FormValue("lecture_description")
		isActive = parseBoolForm(c.FormValue("lecture_is_active"))

		// prioritas file > url text
		if isMultipart {
			if fh, err := c.FormFile("lecture_image_url"); err == nil && fh != nil {
				u, upErr := helperOSS.UploadImageToOSSScoped(masjidUUID, "lectures", fh)
				if upErr != nil {
					log.Printf("[LECTURE] upload OSS error: %v", upErr)
					return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload gambar ke OSS")
				}
				imageURL = &u
			}
		}
		if imageURL == nil {
			if v := strings.TrimSpace(c.FormValue("lecture_image_url")); v != "" {
				imageURL = &v
			}
		}
	}

	if title == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Judul tema kajian wajib diisi")
	}

	// --- slug unik per masjid ---
	baseSlug := generateSlugFromTitle(title)
	slug := baseSlug
	for i := 2; ; i++ {
		var cnt int64
		if err := ctrl.DB.
			Model(&model.LectureModel{}).
			Where("lecture_slug = ? AND lecture_masjid_id = ?", slug, masjidUUID).
			Count(&cnt).Error; err != nil {
			log.Printf("[LECTURE] count slug error: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug")
		}
		if cnt == 0 {
			break
		}
		slug = fmt.Sprintf("%s-%d", baseSlug, i)
	}

	// --- simpan ---
	newLecture := model.LectureModel{
		LectureTitle:       title,
		LectureSlug:        slug,
		LectureDescription: description,
		LectureMasjidID:    masjidUUID,
		LectureImageURL:    imageURL,
		LectureIsActive:    isActive,
	}
	if err := ctrl.DB.Create(&newLecture).Error; err != nil {
		log.Printf("[LECTURE] DB.Create error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat tema kajian")
	}

	return helper.JsonCreated(c, "Tema kajian berhasil dibuat", dto.ToLectureResponse(&newLecture))
}

// parseBoolForm: terima "true/1/on/yes" (case-insensitive)
func parseBoolForm(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "true" || v == "1" || v == "on" || v == "yes"
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


// ✅ PUT /api/a/lectures/:id — upload ke OSS (scoped) & cleanup file lama
func (ctrl *LectureController) UpdateLecture(c *fiber.Ctx) error {
	reqID := uuid.New().String()[0:8]

	// ---------- AUTH ----------
	if uid, err := helper.GetUserIDFromToken(c); err != nil {
		log.Printf("[LECTURE][%s] ❌ auth fail: %v", reqID, err)
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	} else {
		log.Printf("[LECTURE][%s] ✅ auth ok user_id=%s", reqID, uid)
	}

	// ---------- SCOPE ----------
	masjidUUID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidUUID == uuid.Nil {
		log.Printf("[LECTURE][%s] ❌ scope fail: masjid invalid (%v)", reqID, err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid ID tidak valid atau tidak ter-scope")
	}
	log.Printf("[LECTURE][%s] ✅ scope ok masjid_id=%s", reqID, masjidUUID)

	// ---------- LOAD EXISTING ----------
	lectureID := strings.TrimSpace(c.Params("id"))
	var existing model.LectureModel
	if err := ctrl.DB.First(&existing, "lecture_id = ?", lectureID).Error; err != nil {
		log.Printf("[LECTURE][%s] ❌ not found lecture_id=%s err=%v", reqID, lectureID, err)
		return helper.JsonError(c, fiber.StatusNotFound, "Tema kajian tidak ditemukan")
	}
	if existing.LectureMasjidID != masjidUUID {
		log.Printf("[LECTURE][%s] ❌ forbidden: lecture.mosque=%s active=%s", reqID, existing.LectureMasjidID, masjidUUID)
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak berhak mengubah tema kajian ini")
	}
	log.Printf("[LECTURE][%s] ✅ load ok lecture_id=%s title=%q", reqID, existing.LectureID, existing.LectureTitle)

	// Simpan URL lama (kalau ada) untuk cleanup sesudah sukses update
	var oldURL string
	if existing.LectureImageURL != nil {
		oldURL = strings.TrimSpace(*existing.LectureImageURL)
	}

	// ---------- CONTENT-TYPE ----------
	ct := strings.TrimSpace(c.Get(fiber.HeaderContentType))
	isJSON := strings.HasPrefix(ct, fiber.MIMEApplicationJSON)
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	// ---------- REQUEST PAYLOAD ----------
	type reqJSON struct {
		Title          *string `json:"lecture_title"`
		Description    *string `json:"lecture_description"`
		ImageURL       *string `json:"lecture_image_url"`
		RegenerateSlug *bool   `json:"regenerate_slug"`
		IsActive       *bool   `json:"lecture_is_active"`
		Capacity       *int    `json:"lecture_capacity"`
		Price          *int    `json:"lecture_price"`
	}

	updates := map[string]any{}
	titleChanged := false
	regenerateSlug := true // default

	// Track upload baru (untuk rollback kalau DB gagal)
	var newUploadedURL string
	var newUploadedKey string
	var uploadedNewFile bool

	// ---------- JSON ----------
	if isJSON {
		var body reqJSON
		if err := c.BodyParser(&body); err != nil {
			log.Printf("[LECTURE][%s] ❌ body parse error: %v", reqID, err)
			return helper.JsonError(c, fiber.StatusBadRequest, "JSON tidak valid")
		}

		if body.Title != nil {
			if v := strings.TrimSpace(*body.Title); v != "" && v != existing.LectureTitle {
				updates["lecture_title"] = v
				titleChanged = true
			}
		}
		if body.Description != nil {
			updates["lecture_description"] = *body.Description
		}
		if body.ImageURL != nil {
			if v := strings.TrimSpace(*body.ImageURL); v != "" {
				updates["lecture_image_url"] = v // ganti ke URL (bisa OSS atau eksternal)
			} else {
				updates["lecture_image_url"] = nil // kosongkan → nanti hapus file lama kalau dari bucket kita
			}
		}
		if body.RegenerateSlug != nil {
			regenerateSlug = *body.RegenerateSlug
		}
		if body.IsActive != nil {
			updates["lecture_is_active"] = *body.IsActive
		}
		if body.Capacity != nil {
			updates["lecture_capacity"] = *body.Capacity
		}
		if body.Price != nil {
			updates["lecture_price"] = *body.Price
		}
	} else {
		// ---------- FORM / MULTIPART ----------
		if v := strings.TrimSpace(c.FormValue("lecture_title")); v != "" && v != existing.LectureTitle {
			updates["lecture_title"] = v
			titleChanged = true
		}
		if v := c.FormValue("lecture_description"); v != "" {
			updates["lecture_description"] = v
		}
		if v := strings.ToLower(strings.TrimSpace(c.FormValue("regenerate_slug"))); v == "false" {
			regenerateSlug = false
		}
		if v := strings.TrimSpace(c.FormValue("lecture_is_active")); v != "" {
			updates["lecture_is_active"] = strings.EqualFold(v, "true")
		}
		if v := strings.TrimSpace(c.FormValue("lecture_capacity")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				updates["lecture_capacity"] = n
			}
		}
		if v := strings.TrimSpace(c.FormValue("lecture_price")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				updates["lecture_price"] = n
			}
		}

		// --- FILE UPLOAD (sama dg CreateLecture) ---
		if isMultipart {
			if fh, err := c.FormFile("lecture_image_url"); err == nil && fh != nil {
				// Upload ke OSS scoped
				url, upErr := helperOSS.UploadImageToOSSScoped(masjidUUID, "lectures", fh)
				if upErr != nil {
					log.Printf("[LECTURE][%s] ❌ upload OSS error: %v", reqID, upErr)
					return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload gambar ke OSS")
				}
				updates["lecture_image_url"] = url
				newUploadedURL = url
				newUploadedKey = ossKeyFromPublicURL(url) // parse key utk rollback jika perlu
				uploadedNewFile = true
			}
		}
		// fallback: URL string
		if _, ok := updates["lecture_image_url"]; !ok {
			if v := strings.TrimSpace(c.FormValue("lecture_image_url")); v != "" {
				updates["lecture_image_url"] = v
			}
		}
	}

	// ---------- SLUG ----------
	if titleChanged && regenerateSlug {
		base := generateSlugFromTitle(getString(updates, "lecture_title", existing.LectureTitle))
		newSlug, err := uniqueLectureSlug(ctrl.DB, base, existing.LectureMasjidID, existing.LectureID.String())
		if err != nil {
			log.Printf("[LECTURE][%s] ❌ slug gen error: %v", reqID, err)
			// rollback file baru (kalau ada)
			if uploadedNewFile && newUploadedKey != "" {
				if svc, svcErr := helperOSS.NewOSSServiceFromEnv(""); svcErr == nil {
					_ = svc.DeleteObject(context.Background(), newUploadedKey)
				}
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
		}
		updates["lecture_slug"] = newSlug
	}

	// ---------- EXECUTE UPDATE ----------
	if len(updates) == 0 {
		return helper.JsonUpdated(c, "Tidak ada perubahan", dto.ToLectureResponse(&existing))
	}

	// Simpan dulu info apakah image akan dihapus/diubah
	imageChanged := false
	imageCleared := false
	if v, ok := updates["lecture_image_url"]; ok {
		imageChanged = true
		if v == nil {
			imageCleared = true
		}
	}

	// Jalankan update DB
	if err := ctrl.DB.Model(&existing).Updates(updates).Error; err != nil {
		log.Printf("[LECTURE][%s] ❌ DB update error: %v", reqID, err)
		// rollback: hapus file baru yang sudah ter-upload kalau update DB gagal
		if uploadedNewFile && newUploadedKey != "" {
			if svc, svcErr := helperOSS.NewOSSServiceFromEnv(""); svcErr == nil {
				_ = svc.DeleteObject(context.Background(), newUploadedKey)
			}
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update tema kajian")
	}

	// reload
	if err := ctrl.DB.First(&existing, "lecture_id = ?", lectureID).Error; err != nil {
		log.Printf("[LECTURE][%s] ❌ reload error: %v", reqID, err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat data terbaru")
	}

	// ---------- CLEANUP: hapus file lama bila perlu ----------
	// Kondisi hapus:
	// 1) imageCleared = true  → kosongkan → hapus lama (jika lama milik bucket kita)
	// 2) imageChanged = true & newUploadedURL != "" → ganti file → hapus lama (jika lama milik bucket kita dan berbeda)
	if oldURL != "" && (imageCleared || (imageChanged && newUploadedURL != "" && newUploadedURL != oldURL)) {
		oldKey := ossKeyFromPublicURL(oldURL)
		if oldKey != "" {
			if svc, svcErr := helperOSS.NewOSSServiceFromEnv(""); svcErr == nil {
				// non-blocking cleanup
				go func() {
					if delErr := svc.DeleteObject(context.Background(), oldKey); delErr != nil {
						log.Printf("[LECTURE][%s] ⚠️ gagal hapus file lama key=%s err=%v", reqID, oldKey, delErr)
					} else {
						log.Printf("[LECTURE][%s] 🧹 old object deleted key=%s", reqID, oldKey)
					}
				}()
			}
		}
	}

	return helper.JsonUpdated(c, "Tema kajian berhasil diperbarui", dto.ToLectureResponse(&existing))
}

// --- Helper: parse key dari public URL OSS.
// Contoh URL: https://<bucket>.<endpoint>/<key/path/file.ext>[?query]
// Mengembalikan "" jika bukan URL bucket kita (biar aman).
func ossKeyFromPublicURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	svc, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil || svc == nil {
		return ""
	}
	end := strings.TrimPrefix(svc.Endpoint, "https://")
	end = strings.TrimPrefix(end, "http://")

	// Dua kemungkinan prefix valid
	p1 := fmt.Sprintf("https://%s.%s/", svc.BucketName, end)
	p2 := fmt.Sprintf("http://%s.%s/", svc.BucketName, end)

	var key string
	switch {
	case strings.HasPrefix(u, p1):
		key = strings.TrimPrefix(u, p1)
	case strings.HasPrefix(u, p2):
		key = strings.TrimPrefix(u, p2)
	default:
		// bukan dari bucket ini → jangan dihapus
		return ""
	}

	// buang query kalau ada
	if i := strings.IndexByte(key, '?'); i >= 0 {
		key = key[:i]
	}
	return strings.TrimSpace(key)
}



// helper ambil string
func getString(m map[string]any, key, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok2 := v.(string); ok2 {
			return s
		}
	}
	return fallback
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
