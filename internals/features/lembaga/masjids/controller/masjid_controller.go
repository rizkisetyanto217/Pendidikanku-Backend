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

	model "masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"
)

/*
Endpoint ringkas:

GET    /api/masjids/:id/files                    → ListMasjidFiles (list primary URLs)
POST   /api/masjids/:id/files?slot=logo|background|misc
       form-data: file                            → CreateMasjidFile (upload create)
PUT    /api/masjids/:id/files
PATCH  /api/masjids/:id/files
       JSON { "slot":"logo|background", "url":"https://..." }
                                                → UpdateMasjidFile (update metadata + object_key)
DELETE /api/masjids/:id/files
       JSON { "url":"https://..." }             → DeleteMasjidFile (pindah ke spam/ + bersihkan metadata bila perlu)
*/

type MasjidController struct {
	DB       *gorm.DB
	Validate *validator.Validate
	OSS      *helperOSS.OSSService
}

func NewMasjidController(db *gorm.DB, v *validator.Validate, oss *helperOSS.OSSService) *MasjidController {
	return &MasjidController{DB: db, Validate: v, OSS: oss}
}

const defaultRetention = 30 * 24 * time.Hour // 30 hari

// ========== helpers lokal ==========

func parseMasjidID(c *fiber.Ctx) (uuid.UUID, error) {
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

func withinMasjidScope(masjidID uuid.UUID, publicURL string) bool {
	// guard sederhana: pastikan path mengandung /masjids/{id}/
	want := "/masjids/" + masjidID.String() + "/"
	return strings.Contains(publicURL, want)
}

// ========== UPDATE (metadata URL) ==========
// PUT/PATCH /api/masjids/:id/files
// Body: { "slot":"logo|background", "url":"https://..." }
type updateFileReq struct {
	Slot string `json:"slot"`
	URL  string `json:"url"`
}

func (mc *MasjidController) UpdateMasjidFile(c *fiber.Ctx) error {
	id, err := parseMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	var body updateFileReq
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Slot) == "" || strings.TrimSpace(body.URL) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (slot & url wajib)")
	}
	slot := strings.ToLower(strings.TrimSpace(body.Slot))

	var m model.MasjidModel
	if err := mc.DB.First(&m, "masjid_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil masjid")
	}

	key, kerr := helperOSS.KeyFromPublicURL(body.URL)
	if kerr != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "URL tidak valid untuk OSS")
	}

	now := time.Now()
	switch slot {
	case "logo":
		m.MasjidLogoURL = &body.URL
		m.MasjidLogoObjectKey = &key
	case "background":
		m.MasjidBackgroundURL = &body.URL
		m.MasjidBackgroundObjectKey = &key
	default:
		return helper.JsonError(c, fiber.StatusBadRequest, "slot tidak dikenal (pakai: logo|background)")
	}
	m.MasjidUpdatedAt = now

	if err := mc.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui metadata")
	}
	return helper.JsonOK(c, "Metadata diperbarui", fiber.Map{
		"slot": slot,
		"url":  body.URL,
	})
}

// ========== DELETE (pindah ke spam/) ==========
// DELETE /api/masjids/:id/files
// Body: { "url":"https://..." }
type deleteReq struct {
	URL string `json:"url"`
}

func (mc *MasjidController) DeleteMasjidFile(c *fiber.Ctx) error {
	id, err := parseMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	var body deleteReq
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.URL) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (butuh url)")
	}

	// keamanan: batasi hanya URL di scope masjid ini
	if !withinMasjidScope(id, body.URL) {
		return helper.JsonError(c, fiber.StatusForbidden, "URL di luar scope masjid ini")
	}

	// pindahkan ke spam/<YYYY/MM/DD/HHMMSS__basename> (pakai helper baru)
	spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(body.URL, 15*time.Second)
	if mvErr != nil {
		return helper.JsonError(c, fiber.StatusBadGateway, fmt.Sprintf("Gagal memindahkan ke spam: %v", mvErr))
	}

	// jika URL yang dihapus kebetulan adalah current/old di metadata → kosongkan
	var m model.MasjidModel
	if err := mc.DB.First(&m, "masjid_id = ?", id).Error; err == nil {
		changed := false
		now := time.Now()

		if m.MasjidLogoURL != nil && *m.MasjidLogoURL == body.URL {
			m.MasjidLogoURL = nil
			m.MasjidLogoObjectKey = nil
			changed = true
		}
		if m.MasjidLogoURLOld != nil && *m.MasjidLogoURLOld == body.URL {
			m.MasjidLogoURLOld = nil
			m.MasjidLogoObjectKeyOld = nil
			m.MasjidLogoDeletePendingUntil = nil
			changed = true
		}
		if m.MasjidBackgroundURL != nil && *m.MasjidBackgroundURL == body.URL {
			m.MasjidBackgroundURL = nil
			m.MasjidBackgroundObjectKey = nil
			changed = true
		}
		if m.MasjidBackgroundURLOld != nil && *m.MasjidBackgroundURLOld == body.URL {
			m.MasjidBackgroundURLOld = nil
			m.MasjidBackgroundObjectKeyOld = nil
			m.MasjidBackgroundDeletePendingUntil = nil
			changed = true
		}
		if changed {
			m.MasjidUpdatedAt = now
			_ = mc.DB.Save(&m).Error
		}
	}

	return helper.JsonOK(c, "Dipindahkan ke spam", fiber.Map{
		"from_url": body.URL,
		"spam_url": spamURL,
	})
}

/*
(opsional) Router wiring:

func RegisterMasjidMediaRoutes(app *fiber.App, ctl *MasjidController) {
	g := app.Group("/api/masjids")
	g.Get("/:id/files", ctl.ListMasjidFiles)
	g.Post("/:id/files", ctl.CreateMasjidFile)
	g.Put("/:id/files", ctl.UpdateMasjidFile)
	g.Patch("/:id/files", ctl.UpdateMasjidFile)
	g.Delete("/:id/files", ctl.DeleteMasjidFile)
}
*/
