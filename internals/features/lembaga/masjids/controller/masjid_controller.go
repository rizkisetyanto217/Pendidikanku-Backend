// file: internals/features/masjids/masjids/controller/masjid_controller.go
package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	dto "masjidku_backend/internals/features/lembaga/masjids/dto"
	model "masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

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

/* =========================
   Helpers
   ========================= */

func normalizePtr(v string, toLower bool) *string {
	trim := strings.TrimSpace(v)
	if trim == "" {
		return nil
	}
	if toLower {
		l := strings.ToLower(trim)
		return &l
	}
	return &trim
}

func ensureSlugUpdate(db *gorm.DB, existingSlug, candidate string) (string, error) {
	base := helper.GenerateSlug(candidate)
	if base == "" {
		return "", fmt.Errorf("slug kosong")
	}
	if base == existingSlug {
		return existingSlug, nil
	}
	return helper.EnsureUniqueSlug(db, base, "masjids", "masjid_slug")
}

/* =========================
   UPDATE MASJID (Partial)
   PUT /api/a/masjids
   ========================= */

func (mc *MasjidController) UpdateMasjid(c *fiber.Ctx) error {
	if !helperAuth.IsAdmin(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	var existing model.MasjidModel
	if err := mc.DB.First(&existing, "masjid_id = ?", masjidUUID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = ctx // (disiapkan bila butuh context ke depannya)

	// =========================
	// MULTIPART
	// =========================
	if strings.Contains(c.Get("Content-Type"), "multipart/form-data") {
		// akses langsung nilai multipart form jika key ada (termasuk empty string)
		mf, _ := c.MultipartForm()
		getField := func(key string) (string, bool) {
			if mf == nil {
				return "", false
			}
			vals, ok := mf.Value[key]
			if !ok {
				return "", false
			}
			if len(vals) == 0 {
				return "", true
			}
			return vals[0], true
		}

		// strings
		if v := strings.TrimSpace(c.FormValue("masjid_name")); v != "" {
			existing.MasjidName = v
			if newSlug, err := ensureSlugUpdate(mc.DB, existing.MasjidSlug, v); err == nil {
				existing.MasjidSlug = newSlug
			}
		}
		if raw, ok := getField("masjid_bio_short"); ok {
			existing.MasjidBioShort = normalizePtr(raw, false)
		}
		if raw, ok := getField("masjid_location"); ok {
			existing.MasjidLocation = normalizePtr(raw, false)
		}
		// domain: hanya update kalau key ada di form; "" -> NULL; else lower
		if raw, ok := getField("masjid_domain"); ok {
			existing.MasjidDomain = normalizePtr(raw, true)
		}
		if v := strings.TrimSpace(c.FormValue("masjid_slug")); v != "" {
			if newSlug, err := ensureSlugUpdate(mc.DB, existing.MasjidSlug, v); err == nil {
				existing.MasjidSlug = newSlug
			}
		}

		// numeric
		if v := c.FormValue("masjid_latitude"); v != "" {
			if lat, err := strconv.ParseFloat(v, 64); err == nil {
				existing.MasjidLatitude = &lat
			}
		}
		if v := c.FormValue("masjid_longitude"); v != "" {
			if lng, err := strconv.ParseFloat(v, 64); err == nil {
				existing.MasjidLongitude = &lng
			}
		}

		// flags & plan
		if v := c.FormValue("masjid_is_active"); v != "" {
			existing.MasjidIsActive = v == "true" || v == "1"
		}
		if v := strings.TrimSpace(c.FormValue("masjid_verification_status")); v != "" {
			switch v {
			case "pending", "approved", "rejected":
				existing.MasjidVerificationStatus = model.VerificationStatus(v)
			}
		}
		if raw, ok := getField("masjid_verification_notes"); ok {
			existing.MasjidVerificationNotes = normalizePtr(raw, false)
		}
		if v := c.FormValue("masjid_current_plan_id"); v != "" {
			if planID, err := uuid.Parse(v); err == nil {
				existing.MasjidCurrentPlanID = &planID
			}
		}
		// flag sekolah
		if v := c.FormValue("masjid_is_islamic_school"); v != "" {
			existing.MasjidIsIslamicSchool = v == "true" || v == "1"
		}

	// =========================
	// JSON (partial)
	// =========================
	} else {
		var input dto.MasjidUpdateRequest
		if err := c.BodyParser(&input); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Format JSON tidak valid")
		}

		if input.MasjidName != nil {
			existing.MasjidName = *input.MasjidName
			if newSlug, err := ensureSlugUpdate(mc.DB, existing.MasjidSlug, *input.MasjidName); err == nil {
				existing.MasjidSlug = newSlug
			}
		}
		if input.MasjidSlug != nil {
			if newSlug, err := ensureSlugUpdate(mc.DB, existing.MasjidSlug, *input.MasjidSlug); err == nil {
				existing.MasjidSlug = newSlug
			}
		}
		if input.MasjidBioShort != nil {
			existing.MasjidBioShort = normalizePtr(*input.MasjidBioShort, false)
		}
		if input.MasjidLocation != nil {
			existing.MasjidLocation = normalizePtr(*input.MasjidLocation, false)
		}
		if input.MasjidLatitude != nil {
			existing.MasjidLatitude = input.MasjidLatitude
		}
		if input.MasjidLongitude != nil {
			existing.MasjidLongitude = input.MasjidLongitude
		}

		// domain: "" → NULL; else lower
		if input.MasjidDomain != nil {
			existing.MasjidDomain = normalizePtr(*input.MasjidDomain, true)
		}

		// Flags & verif
		if input.MasjidIsActive != nil {
			existing.MasjidIsActive = *input.MasjidIsActive
		}
		if input.MasjidVerificationStatus != nil {
			v := strings.TrimSpace(*input.MasjidVerificationStatus)
			switch v {
			case "pending", "approved", "rejected":
				existing.MasjidVerificationStatus = model.VerificationStatus(v)
			}
		}
		if input.MasjidVerificationNotes != nil {
			existing.MasjidVerificationNotes = normalizePtr(*input.MasjidVerificationNotes, false)
		}
		if input.MasjidCurrentPlanID != nil {
			existing.MasjidCurrentPlanID = input.MasjidCurrentPlanID
		}
		if input.MasjidIsIslamicSchool != nil {
			existing.MasjidIsIslamicSchool = *input.MasjidIsIslamicSchool
		}
	}

	if err := mc.DB.Save(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui masjid")
	}
	return helper.JsonOK(c, "Masjid berhasil diperbarui", dto.FromModelMasjid(&existing))
}

/* =========================
   DELETE
   ========================= */

// 🗑️ DELETE /api/a/masjids           -> admin: pakai ID token; owner: 400 (perlu :id)
// 🗑️ DELETE /api/a/masjids/:id       -> owner: boleh; admin: hanya jika :id sama dgn ID token
func (mc *MasjidController) DeleteMasjid(c *fiber.Ctx) error {
	isAdmin := helperAuth.IsAdmin(c)
	isOwner := helperAuth.IsOwner(c)

	if !isAdmin && !isOwner {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	pathID := strings.TrimSpace(c.Params("id"))
	var targetID uuid.UUID

	// Masjid ID dari token untuk admin
	var tokenMasjidID uuid.UUID
	var tokenErr error
	if isAdmin {
		tokenMasjidID, tokenErr = helperAuth.GetMasjidIDFromToken(c)
		if tokenErr != nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, tokenErr.Error())
		}
	}

	// Aturan path
	if pathID == "" {
		if isOwner {
			return helper.JsonError(c, fiber.StatusBadRequest, "Owner harus menyertakan ID masjid di path")
		}
		targetID = tokenMasjidID
	} else {
		pathUUID, err := uuid.Parse(pathID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Format ID masjid tidak valid")
		}
		if isAdmin && pathUUID != tokenMasjidID {
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh menghapus masjid di luar scope Anda")
		}
		if isOwner {
			userID, err := helperAuth.GetUserIDFromToken(c)
			if err != nil {
				return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
			}
			var count int64
			if err := mc.DB.
				Table("masjid_admins_teachers").
				Where("masjid_admins_user_id = ? AND masjid_admins_masjid_id = ? AND masjid_admins_is_active = TRUE", userID, pathUUID).
				Count(&count).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memverifikasi kepemilikan")
			}
			if count == 0 {
				return helper.JsonError(c, fiber.StatusForbidden, "Anda bukan owner/admin masjid ini")
			}
		}
		targetID = pathUUID
	}

	// Ambil data existing
	var existing model.MasjidModel
	if err := mc.DB.First(&existing, "masjid_id = ?", targetID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
	}

	// Soft delete record (kolom masjid_deleted_at)
	if err := mc.DB.Delete(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus masjid")
	}

	return helper.JsonDeleted(c, "Masjid berhasil dihapus", fiber.Map{
		"masjid_id": targetID.String(),
	})
}
