// file: internals/features/lembaga/yayasans/controller/yayasan_files_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	yModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/yayasans/model"
	helper "madinahsalam_backend/internals/helpers"
	helperOSS "madinahsalam_backend/internals/helpers/oss"
)

/*
Endpoint ringkas (disamain spt School):

GET    /api/yayasans/:id/files
POST   /api/yayasans/:id/files?slot=logo|misc     (form-data: file)   → upload
PUT    /api/yayasans/:id/files
PATCH  /api/yayasans/:id/files                    (JSON {slot,url})   → update metadata+object_key
DELETE /api/yayasans/:id/files                    (JSON {url})        → move to spam & bersihkan metadata bila perlu
*/

type YayasanController struct {
	DB       *gorm.DB
	Validate *validator.Validate
	OSS      *helperOSS.OSSService
}

func NewYayasanController(db *gorm.DB, v *validator.Validate, oss *helperOSS.OSSService) *YayasanController {
	return &YayasanController{DB: db, Validate: v, OSS: oss}
}

const defaultRetention = 30 * 24 * time.Hour // 30 hari

// ===== helpers lokal =====

func parseYayasanID(c *fiber.Ctx) (uuid.UUID, error) {
	s := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	return id, nil
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ensureOSS(oss *helperOSS.OSSService) error {
	if oss == nil {
		return fiber.NewError(fiber.StatusFailedDependency, "OSS belum dikonfigurasi")
	}
	return nil
}

func withinYayasanScope(yayasanID uuid.UUID, publicURL string) bool {
	// guard sederhana: wajib mengandung /yayasans/{id}/
	want := "/yayasans/" + yayasanID.String() + "/"
	return strings.Contains(publicURL, want)
}

// ===== LIST =====
// GET /api/yayasans/:id/files
func (yc *YayasanController) List(c *fiber.Ctx) error {
	id, err := parseYayasanID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m yModel.YayasanModel
	if err := yc.DB.First(&m, "yayasan_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Yayasan tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil yayasan")
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"logo_url":                  safeStr(m.YayasanLogoURL),
		"logo_url_old":              safeStr(m.YayasanLogoURLOld),
		"logo_delete_pending_until": m.YayasanLogoDeletePendingUntil,
	})
}

// ===== CREATE (UPLOAD) =====
// POST /api/yayasans/:id/files?slot=logo|misc
// form-data: file
func (yc *YayasanController) Create(c *fiber.Ctx) error {
	if err := ensureOSS(yc.OSS); err != nil {
		return helper.JsonError(c, fiber.StatusFailedDependency, err.Error())
	}
	id, err := parseYayasanID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	slot := strings.ToLower(strings.TrimSpace(c.Query("slot")))
	if slot == "" {
		slot = "misc"
	}
	fh, err := c.FormFile("file")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "file tidak ditemukan")
	}

	// pastikan yayasan ada
	var m yModel.YayasanModel
	if err := yc.DB.First(&m, "yayasan_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Yayasan tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil yayasan")
	}

	// Upload:
	// - jika slot == "logo" → selalu re-encode ke WebP dan taruh di "yayasans/{id}/images/logo/"
	// - selain itu → upload mentah ke "yayasans/{id}/files/{slot}/"
	var publicURL string
	if slot == "logo" {
		prefix := "yayasans/" + id.String() + "/images/logo"
		url, upErr := yc.OSS.UploadAsWebP(c.Context(), fh, prefix)
		if upErr != nil {
			var fe *fiber.Error
			if errors.As(upErr, &fe) {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload ke OSS")
		}
		publicURL = url
	} else {
		// upload mentah ke subdir files/{slot}
		prefix := "yayasans/" + id.String() + "/files/" + slot
		// manfaatkan UploadFromFormFileToDir untuk bebas ekstensi
		key, _, upErr := yc.OSS.UploadFromFormFileToDir(c.Context(), prefix, fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload ke OSS")
		}
		publicURL = yc.OSS.PublicURL(key)
	}

	// update metadata untuk slot singleton (logo) + 2-slot retensi
	now := time.Now()
	retUntil := now.Add(defaultRetention)

	if slot == "logo" {
		// shift current → old (retensi)
		if m.YayasanLogoURL != nil && strings.TrimSpace(*m.YayasanLogoURL) != "" {
			m.YayasanLogoURLOld = m.YayasanLogoURL
			m.YayasanLogoObjectKeyOld = m.YayasanLogoObjectKey
			m.YayasanLogoDeletePendingUntil = &retUntil
		}
		m.YayasanLogoURL = &publicURL
		if key, err := helperOSS.KeyFromPublicURL(publicURL); err == nil {
			m.YayasanLogoObjectKey = &key
		}
		m.YayasanUpdatedAt = now

		if err := yc.DB.Save(&m).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan metadata")
		}
	}

	return helper.JsonCreated(c, "Upload sukses", fiber.Map{
		"url":  publicURL,
		"slot": slot,
	})
}

// ===== UPDATE (metadata URL) =====
// PUT/PATCH /api/yayasans/:id/files
// Body: { "slot":"logo", "url":"https://..." }
type updateYayasanFileReq struct {
	Slot string `json:"slot"`
	URL  string `json:"url"`
}

func (yc *YayasanController) Update(c *fiber.Ctx) error {
	id, err := parseYayasanID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	var body updateYayasanFileReq
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Slot) == "" || strings.TrimSpace(body.URL) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (slot & url wajib)")
	}
	slot := strings.ToLower(strings.TrimSpace(body.Slot))

	var m yModel.YayasanModel
	if err := yc.DB.First(&m, "yayasan_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Yayasan tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil yayasan")
	}

	key, kerr := helperOSS.KeyFromPublicURL(body.URL)
	if kerr != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "URL tidak valid untuk OSS")
	}

	now := time.Now()
	switch slot {
	case "logo":
		m.YayasanLogoURL = &body.URL
		m.YayasanLogoObjectKey = &key
	default:
		return helper.JsonError(c, fiber.StatusBadRequest, "slot tidak dikenal (pakai: logo|misc)")
	}
	m.YayasanUpdatedAt = now

	if err := yc.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui metadata")
	}
	return helper.JsonOK(c, "Metadata diperbarui", fiber.Map{
		"slot": slot,
		"url":  body.URL,
	})
}

// ===== DELETE (move ke spam + cleanup metadata jika perlu) =====
// DELETE /api/yayasans/:id/files
// Body: { "url":"https://..." }
type deleteYayasanFileReq struct {
	URL string `json:"url"`
}

func (yc *YayasanController) Delete(c *fiber.Ctx) error {
	id, err := parseYayasanID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	var body deleteYayasanFileReq
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.URL) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (butuh url)")
	}

	// keamanan: hanya izinkan URL dalam scope yayasan
	if !withinYayasanScope(id, body.URL) {
		return helper.JsonError(c, fiber.StatusForbidden, "URL di luar scope yayasan ini")
	}

	// pindah ke spam/.. (pakai helper yang sama dengan School)
	spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(body.URL, 15*time.Second)
	if mvErr != nil {
		return helper.JsonError(c, fiber.StatusBadGateway, fmt.Sprintf("Gagal memindahkan ke spam: %v", mvErr))
	}

	// jika URL tsb sedang dipakai di metadata → kosongkan
	var m yModel.YayasanModel
	if err := yc.DB.First(&m, "yayasan_id = ?", id).Error; err == nil {
		changed := false
		now := time.Now()

		if m.YayasanLogoURL != nil && *m.YayasanLogoURL == body.URL {
			m.YayasanLogoURL = nil
			m.YayasanLogoObjectKey = nil
			changed = true
		}
		if m.YayasanLogoURLOld != nil && *m.YayasanLogoURLOld == body.URL {
			m.YayasanLogoURLOld = nil
			m.YayasanLogoObjectKeyOld = nil
			m.YayasanLogoDeletePendingUntil = nil
			changed = true
		}

		if changed {
			m.YayasanUpdatedAt = now
			_ = yc.DB.Save(&m).Error
		}
	}

	return helper.JsonOK(c, "Dipindahkan ke spam", fiber.Map{
		"from_url": body.URL,
		"spam_url": spamURL,
	})
}
