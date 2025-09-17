// file: internals/features/masjids/masjids/controller/masjid_controller.go
package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"masjidku_backend/internals/features/lembaga/masjids/dto"
	model "masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

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
	if s == nil { return "" }
	return *s
}
func jsonEqual(a, b *datatypes.JSON) bool {
	if a == nil && b == nil { return true }
	if a == nil || b == nil { return false }
	return string(*a) == string(*b)
}
func retentionDuration() time.Duration {
	// samakan dengan reaper (default 30 hari)
	d := 30
	if v, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("RETENTION_DAYS"))); v > 0 {
		d = v
	}
	return time.Duration(d) * 24 * time.Hour
}


func (mc *MasjidController) Patch(c *fiber.Ctx) error {
    id, err := parseMasjidID(c)
    if err != nil {
        return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
    }

    // ========== AUTH: resolve masjid dari token & cek DKM ==========
    masjidIDFromToken, err := helperAuth.GetActiveMasjidID(c)
    if err != nil {
        if id2, err2 := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err2 == nil {
            masjidIDFromToken = id2
        } else {
            return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid context tidak ditemukan")
        }
    }
    if masjidIDFromToken != id {
        return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: masjid tidak sesuai")
    }
    if err := helperAuth.EnsureDKMMasjid(c, masjidIDFromToken); err != nil {
        // pastikan helper ini sudah mengembalikan *fiber.Error / JSON-friendly error
        return err
    }
    // ===============================================================

    // Ambil row existing
    var m model.MasjidModel
    if err := mc.DB.First(&m, "masjid_id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
        }
        return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil masjid")
    }
    before := m // untuk deteksi perubahan

    // --- siapkan mutable state ---
    var u dto.MasjidUpdateRequest
    now := time.Now()
    changedMedia := false
    retainUntil := now.Add(retentionDuration())

    ct := strings.ToLower(c.Get("Content-Type"))

    // [A] multipart/form-data ...
    if strings.HasPrefix(ct, "multipart/form-data") {
        if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
            if err := json.Unmarshal([]byte(s), &u); err != nil {
                return helper.JsonError(c, fiber.StatusBadRequest, "payload JSON tidak valid")
            }
        } else {
            _ = c.BodyParser(&u)
        }

        svc, err := helperOSS.NewOSSServiceFromEnv("")
        if err != nil {
            return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
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

    // Terapkan patch field non-file
    dto.ApplyMasjidUpdate(&m, &u)
    m.MasjidUpdatedAt = now

    // Bangun updates map hanya kolom yang berubah
    updates := map[string]any{"masjid_updated_at": m.MasjidUpdatedAt}

    // inti identitas
    if before.MasjidName != m.MasjidName { updates["masjid_name"] = m.MasjidName }
    if val(before.MasjidBioShort) != val(m.MasjidBioShort) { updates["masjid_bio_short"] = m.MasjidBioShort }
    if val(before.MasjidLocation) != val(m.MasjidLocation) { updates["masjid_location"] = m.MasjidLocation }
    if val(before.MasjidCity) != val(m.MasjidCity) { updates["masjid_city"] = m.MasjidCity }
    if val(before.MasjidDomain) != val(m.MasjidDomain) { updates["masjid_domain"] = m.MasjidDomain }
    if before.MasjidSlug != m.MasjidSlug { updates["masjid_slug"] = m.MasjidSlug }

    // verifikasi & flags
    if before.MasjidIsActive != m.MasjidIsActive { updates["masjid_is_active"] = m.MasjidIsActive }
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

    // media current
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

    // media shadow
    if changedMedia {
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

    if len(updates) == 1 {
        return helper.JsonOK(c, "Tidak ada perubahan", fiber.Map{
            "item": dto.FromModelMasjid(&m),
        })
    }

    if err := mc.DB.Model(&m).Updates(updates).Error; err != nil {
        return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
    }

    return helper.JsonOK(c, "Berhasil", fiber.Map{
        "item": dto.FromModelMasjid(&m),
    })
}


// ========== DELETE (pindah ke spam/) ==========
// DELETE /api/masjids/:id/files
// Body: { "url":"https://..." }
type deleteReq struct {
    URL string `json:"url"`
}

func (mc *MasjidController) Delete(c *fiber.Ctx) error {
    id, err := parseMasjidID(c)
    if err != nil {
        return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
    }

    // ========== AUTH ==========
    masjidIDFromToken, err := helperAuth.GetActiveMasjidID(c)
    if err != nil {
        if id2, err2 := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err2 == nil {
            masjidIDFromToken = id2
        } else {
            return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid context tidak ditemukan")
        }
    }
    if masjidIDFromToken != id {
        return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: masjid tidak sesuai")
    }
    if err := helperAuth.EnsureDKMMasjid(c, masjidIDFromToken); err != nil {
        return err
    }
    // ==========================

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
            _ = mc.DB.Save(&m).Error // best-effort
        }
    }

    return helper.JsonOK(c, "Dipindahkan ke spam", fiber.Map{
        "from_url": body.URL,
        "spam_url": spamURL,
    })
}
