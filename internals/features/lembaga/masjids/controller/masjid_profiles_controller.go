// file: internals/features/lembaga/masjids/controller/masjid_profile_controller.go
package controller

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	d "masjidku_backend/internals/features/lembaga/masjids/dto"
	m "masjidku_backend/internals/features/lembaga/masjids/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* =======================================================
   Controller & Constructor
   ======================================================= */

type MasjidProfileController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewMasjidProfileController(db *gorm.DB, v *validator.Validate) *MasjidProfileController {
	return &MasjidProfileController{DB: db, Validate: v}
}

/* =======================================================
   Helpers
   ======================================================= */

func isUniqueViolation(err error) bool {
	// Postgres unique_violation code = 23505 (cek string aman untuk GORM)
	return err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}

func parseUUIDParam(c *fiber.Ctx, key string) (uuid.UUID, error) {
	idStr := c.Params(key)
	return uuid.Parse(idStr)
}

/* =======================================================
   Handlers
   ======================================================= */

// Create (DKM/Admin masjid). Satu masjid cuma boleh punya 1 profile.
// POST /admin/:masjid_id/masjid-profiles  (atau /admin/:masjid_slug/...)
func (ctl *MasjidProfileController) Create(c *fiber.Ctx) error {
	// Pastikan DB tersedia untuk resolver (slug→id)
	if c.Locals("DB") == nil {
		c.Locals("DB", ctl.DB)
	}

	// 🔐 Ambil context & authorize DKM/Admin di masjid ini
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req d.MasjidProfileCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
		}
	}

	// Enforce tenant dari context (anti cross-tenant injection)
	req.MasjidProfileMasjidID = masjidID.String()

	model := d.ToModelMasjidProfileCreate(&req)

	// Pastikan belum ada profile utk masjid ini (idempotent, sekaligus error message yang ramah)
	var exists int64
	if err := ctl.DB.Model(&m.MasjidProfileModel{}).
		Where("masjid_profile_masjid_id = ? AND masjid_profile_deleted_at IS NULL", model.MasjidProfileMasjidID).
		Count(&exists).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}
	if exists > 0 {
		return helper.JsonError(c, fiber.StatusConflict, "Profil untuk masjid ini sudah ada")
	}

	if err := ctl.DB.Create(model).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Duplikasi NPSN/NSS/masjid_id")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat profil: "+err.Error())
	}

	resp := d.FromModelMasjidProfile(model)
	return helper.JsonCreated(c, "Profil masjid berhasil dibuat", resp)
}

// PATCH /admin/:masjid_id/masjid-profiles/:id
func (ctl *MasjidProfileController) Update(c *fiber.Ctx) error {
	// DB untuk resolver
	if c.Locals("DB") == nil {
		c.Locals("DB", ctl.DB)
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil record dulu untuk tahu tenant-nya
	var p m.MasjidProfileModel
	if err := ctl.DB.
		Where("masjid_profile_id = ? AND masjid_profile_deleted_at IS NULL", id).
		First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// 🔐 Otorisasi:
	// Owner boleh lintas; selain Owner wajib DKM/Admin di masjid pemilik profil
	if !helperAuth.IsOwner(c) {
		if err := helperAuth.EnsureDKMMasjid(c, p.MasjidProfileMasjidID); err != nil {
			return err
		}
	}

	var req d.MasjidProfileUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
		}
	}

	// Terapkan patch ke struct in-memory
	d.ApplyPatchToModel(&p, &req)

	// Tenant guard: jangan sampai masjid_id berubah
	// (kolom ini juga di-Omit saat update)
	if err := ctl.DB.Model(&p).
		Omit(
			"masjid_profile_id",
			"masjid_profile_masjid_id",
			"masjid_profile_search",
			"masjid_profile_created_at",
		).
		Select("*").
		Updates(&p).Error; err != nil {

		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Duplikasi NPSN/NSS")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update: "+err.Error())
	}

	// Reload untuk updated_at
	if err := ctl.DB.First(&p, "masjid_profile_id = ?", id).Error; err != nil {
		log.Println("[WARN] reload after update:", err)
	}

	return helper.JsonUpdated(c, "Profil masjid berhasil diperbarui", d.FromModelMasjidProfile(&p))
}

// DELETE (soft) — hanya Owner
// DELETE /admin/:masjid_id/masjid-profiles/:id
func (ctl *MasjidProfileController) Delete(c *fiber.Ctx) error {
	// DB untuk resolver
	if c.Locals("DB") == nil {
		c.Locals("DB", ctl.DB)
	}

	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: admin saja")
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// (opsional) verifikasi record ada & alive → sekaligus dapet tenant untuk audit
	var p m.MasjidProfileModel
	if err := ctl.DB.
		Where("masjid_profile_id = ? AND masjid_profile_deleted_at IS NULL", id).
		First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// Soft delete pakai timestamp
	if err := ctl.DB.
		Model(&m.MasjidProfileModel{}).
		Where("masjid_profile_id = ?", id).
		Update("masjid_profile_deleted_at", time.Now()).
		Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus: "+err.Error())
	}

	return helper.JsonDeleted(c, "Profil masjid dihapus (soft delete)", fiber.Map{"deleted": true})
}
