package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	masjidDto "masjidku_backend/internals/features/lembaga/masjids/dto"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

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

// Cek scope menggunakan key hasil ekstrak dari public URL
func withinMasjidScope(masjidID uuid.UUID, publicURL string) bool {
	key, err := helperOSS.KeyFromPublicURL(publicURL)
	if err != nil {
		return false
	}
	prefix := "masjids/" + masjidID.String() + "/"
	return strings.HasPrefix(key, prefix)
}

// ------- util kecil untuk banding nilai pointer & json -------
func val(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func jsonEqual(a, b *datatypes.JSON) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return string(*a) == string(*b)
}

// PATCH /api/masjids/:id
// PATCH /api/masjids/:id
func (mc *MasjidController) Patch(c *fiber.Ctx) error {
	id, err := parseMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ===== AUTH (DKM) — pin ke path id =====
	masjidID, aerr := helperAuth.EnsureMasjidAccessDKM(c, helperAuth.MasjidContext{ID: id})
	if aerr != nil {
		return helper.JsonError(c, aerr.(*fiber.Error).Code, aerr.Error())
	}
	if masjidID != id {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: masjid tidak sesuai")
	}

	// Ambil row existing
	var m masjidDto.Masjid
	if err := mc.DB.First(&m, "masjid_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil masjid")
	}
	before := m // snapshot untuk deteksi delta

	// --- state ---
	var u masjidDto.MasjidUpdateRequest
	now := time.Now()
	changedMedia := false
	retainUntil := now.Add(helperOSS.GetRetentionDuration())

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))

	// [A] multipart/form-data
	if strings.HasPrefix(ct, "multipart/form-data") {
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			if err := json.Unmarshal([]byte(s), &u); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "payload JSON tidak valid")
			}
		} else {
			_ = c.BodyParser(&u) // best-effort untuk field sederhana
		}

		// OSS service
		svc := mc.OSS
		if svc == nil {
			tmp, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
			}
			svc = tmp
		}

		// -- icon --
		if fh, err := c.FormFile("icon"); err == nil && fh != nil {
			url, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, id, "icon", fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (icon)")
			}
			if m.MasjidIconURL != nil && *m.MasjidIconURL != "" {
				m.MasjidIconURLOld = m.MasjidIconURL
				m.MasjidIconObjectKeyOld = m.MasjidIconObjectKey
				m.MasjidIconDeletePendingUntil = &retainUntil
			}
			m.MasjidIconURL = &url
			m.MasjidIconObjectKey = &key
			changedMedia = true
		}

		// -- logo --
		if fh, err := c.FormFile("logo"); err == nil && fh != nil {
			url, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, id, "logo", fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (logo)")
			}
			if m.MasjidLogoURL != nil && *m.MasjidLogoURL != "" {
				m.MasjidLogoURLOld = m.MasjidLogoURL
				m.MasjidLogoObjectKeyOld = m.MasjidLogoObjectKey
				m.MasjidLogoDeletePendingUntil = &retainUntil
			}
			m.MasjidLogoURL = &url
			m.MasjidLogoObjectKey = &key
			changedMedia = true
		}

		// -- background --
		if fh, err := c.FormFile("background"); err == nil && fh != nil {
			url, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, id, "background", fh)
			if upErr != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, upErr.Error())
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (background)")
			}
			if m.MasjidBackgroundURL != nil && *m.MasjidBackgroundURL != "" {
				m.MasjidBackgroundURLOld = m.MasjidBackgroundURL
				m.MasjidBackgroundObjectKeyOld = m.MasjidBackgroundObjectKey
				m.MasjidBackgroundDeletePendingUntil = &retainUntil
			}
			m.MasjidBackgroundURL = &url
			m.MasjidBackgroundObjectKey = &key
			changedMedia = true
		}

	} else {
		// [B] JSON biasa
		if err := c.BodyParser(&u); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// Terapkan patch field non-file (current-only)
	masjidDto.ApplyMasjidUpdate(&m, &u)
	m.MasjidUpdatedAt = now

	// Bangun updates map hanya kolom yang berubah
	updates := map[string]any{"masjid_updated_at": m.MasjidUpdatedAt}

	// inti identitas
	if before.MasjidName != m.MasjidName {
		updates["masjid_name"] = m.MasjidName
	}
	if val(before.MasjidBioShort) != val(m.MasjidBioShort) {
		updates["masjid_bio_short"] = m.MasjidBioShort
	}
	if val(before.MasjidLocation) != val(m.MasjidLocation) {
		updates["masjid_location"] = m.MasjidLocation
	}
	if val(before.MasjidCity) != val(m.MasjidCity) {
		updates["masjid_city"] = m.MasjidCity
	}
	if val(before.MasjidDomain) != val(m.MasjidDomain) {
		updates["masjid_domain"] = m.MasjidDomain
	}
	if before.MasjidSlug != m.MasjidSlug {
		updates["masjid_slug"] = m.MasjidSlug
	}

	// verifikasi & flags
	if before.MasjidIsActive != m.MasjidIsActive {
		updates["masjid_is_active"] = m.MasjidIsActive
	}
	if string(before.MasjidVerificationStatus) != string(m.MasjidVerificationStatus) {
		updates["masjid_verification_status"] = m.MasjidVerificationStatus
	}
	if val(before.MasjidVerificationNotes) != val(m.MasjidVerificationNotes) {
		updates["masjid_verification_notes"] = m.MasjidVerificationNotes
	}
	if val(before.MasjidContactPersonName) != val(m.MasjidContactPersonName) {
		updates["masjid_contact_person_name"] = m.MasjidContactPersonName
	}
	if val(before.MasjidContactPersonPhone) != val(m.MasjidContactPersonPhone) {
		updates["masjid_contact_person_phone"] = m.MasjidContactPersonPhone
	}
	if before.MasjidIsIslamicSchool != m.MasjidIsIslamicSchool {
		updates["masjid_is_islamic_school"] = m.MasjidIsIslamicSchool
	}
	if string(before.MasjidTenantProfile) != string(m.MasjidTenantProfile) {
		updates["masjid_tenant_profile"] = m.MasjidTenantProfile
	}
	if !jsonEqual(before.MasjidLevels, m.MasjidLevels) {
		updates["masjid_levels"] = m.MasjidLevels
	}

	// media current (ICON + LOGO + BACKGROUND)
	if val(before.MasjidIconURL) != val(m.MasjidIconURL) {
		updates["masjid_icon_url"] = m.MasjidIconURL
	}
	if val(before.MasjidIconObjectKey) != val(m.MasjidIconObjectKey) {
		updates["masjid_icon_object_key"] = m.MasjidIconObjectKey
	}
	if val(before.MasjidLogoURL) != val(m.MasjidLogoURL) {
		updates["masjid_logo_url"] = m.MasjidLogoURL
	}
	if val(before.MasjidLogoObjectKey) != val(m.MasjidLogoObjectKey) {
		updates["masjid_logo_object_key"] = m.MasjidLogoObjectKey
	}
	if val(before.MasjidBackgroundURL) != val(m.MasjidBackgroundURL) {
		updates["masjid_background_url"] = m.MasjidBackgroundURL
	}
	if val(before.MasjidBackgroundObjectKey) != val(m.MasjidBackgroundObjectKey) {
		updates["masjid_background_object_key"] = m.MasjidBackgroundObjectKey
	}

	// media shadow (2-slot) — hanya untuk yang benar2 berubah
	if changedMedia {
		if val(before.MasjidIconURL) != val(m.MasjidIconURL) {
			updates["masjid_icon_url_old"] = m.MasjidIconURLOld
			updates["masjid_icon_object_key_old"] = m.MasjidIconObjectKeyOld
			updates["masjid_icon_delete_pending_until"] = m.MasjidIconDeletePendingUntil
		}
		if val(before.MasjidLogoURL) != val(m.MasjidLogoURL) {
			updates["masjid_logo_url_old"] = m.MasjidLogoURLOld
			updates["masjid_logo_object_key_old"] = m.MasjidLogoObjectKeyOld
			updates["masjid_logo_delete_pending_until"] = m.MasjidLogoDeletePendingUntil
		}
		if val(before.MasjidBackgroundURL) != val(m.MasjidBackgroundURL) {
			updates["masjid_background_url_old"] = m.MasjidBackgroundURLOld
			updates["masjid_background_object_key_old"] = m.MasjidBackgroundObjectKeyOld
			updates["masjid_background_delete_pending_until"] = m.MasjidBackgroundDeletePendingUntil
		}
	}

	if len(updates) == 1 { // cuma updated_at
		return helper.JsonOK(c, "Tidak ada perubahan", fiber.Map{
			"item": masjidDto.FromModelMasjid(&m),
		})
	}

	if err := mc.DB.Model(&m).Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	return helper.JsonOK(c, "Berhasil", fiber.Map{
		"item": masjidDto.FromModelMasjid(&m),
	})
}

// DELETE /api/masjids/:id/files { "url": "https://..." }
type deleteReq struct {
	URL string `json:"url"`
}

func (mc *MasjidController) Delete(c *fiber.Ctx) error {
	id, err := parseMasjidID(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ===== AUTH via helperAuth (DKM only) =====
	masjidID, aerr := helperAuth.EnsureMasjidAccessDKM(c, helperAuth.MasjidContext{ID: id})
	if aerr != nil {
		return helper.JsonError(c, aerr.(*fiber.Error).Code, aerr.Error())
	}
	if masjidID != id {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: masjid tidak sesuai")
	}
	// ==========================================

	var body deleteReq
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.URL) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (butuh url)")
	}

	if !withinMasjidScope(id, body.URL) {
		return helper.JsonError(c, fiber.StatusForbidden, "URL di luar scope masjid ini")
	}

	spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(body.URL, 15*time.Second)
	if mvErr != nil {
		return helper.JsonError(c, fiber.StatusBadGateway, fmt.Sprintf("Gagal memindahkan ke spam: %v", mvErr))
	}

	var m masjidDto.Masjid
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
			_ = mc.DB.Save(&m).Error // best-effort
		}
	}

	return helper.JsonOK(c, "Dipindahkan ke spam", fiber.Map{
		"from_url": body.URL,
		"spam_url": spamURL,
	})
}
