// file: internals/features/lembaga/schools/controller/school_profile_controller.go
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

	d "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/dto"
	m "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/* =======================================================
   Controller & Constructor
   ======================================================= */

type SchoolProfileController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewSchoolProfileController(db *gorm.DB, v *validator.Validate) *SchoolProfileController {
	return &SchoolProfileController{DB: db, Validate: v}
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

// Create (DKM/Admin school). Satu school cuma boleh punya 1 profile.
// POST /admin/:school_id/school-profiles  (atau /admin/:school_slug/...)
func (ctl *SchoolProfileController) Create(c *fiber.Ctx) error {
	// Pastikan DB tersedia untuk resolver (slug→id)
	if c.Locals("DB") == nil {
		c.Locals("DB", ctl.DB)
	}

	// 🔐 Ambil context & authorize DKM/Admin di school ini
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	schoolID, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req d.SchoolProfileCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
		}
	}

	// Enforce tenant dari context (anti cross-tenant injection)
	req.SchoolProfileSchoolID = schoolID.String()

	model := d.ToModelSchoolProfileCreate(&req)

	// Pastikan belum ada profile utk school ini (idempotent, sekaligus error message yang ramah)
	var exists int64
	if err := ctl.DB.Model(&m.SchoolProfileModel{}).
		Where("school_profile_school_id = ? AND school_profile_deleted_at IS NULL", model.SchoolProfileSchoolID).
		Count(&exists).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}
	if exists > 0 {
		return helper.JsonError(c, fiber.StatusConflict, "Profil untuk school ini sudah ada")
	}

	if err := ctl.DB.Create(model).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Duplikasi NPSN/NSS/school_id")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat profil: "+err.Error())
	}

	resp := d.FromModelSchoolProfile(model)
	return helper.JsonCreated(c, "Profil school berhasil dibuat", resp)
}

// PATCH /admin/:school_id/school-profiles/:id
func (ctl *SchoolProfileController) Update(c *fiber.Ctx) error {
	// DB untuk resolver
	if c.Locals("DB") == nil {
		c.Locals("DB", ctl.DB)
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil record dulu untuk tahu tenant-nya
	var p m.SchoolProfileModel
	if err := ctl.DB.
		Where("school_profile_id = ? AND school_profile_deleted_at IS NULL", id).
		First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// 🔐 Otorisasi:
	// Owner boleh lintas; selain Owner wajib DKM/Admin di school pemilik profil
	if !helperAuth.IsOwner(c) {
		if err := helperAuth.EnsureDKMSchool(c, p.SchoolProfileSchoolID); err != nil {
			return err
		}
	}

	var req d.SchoolProfileUpdateRequest
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

	// Tenant guard: jangan sampai school_id berubah
	// (kolom ini juga di-Omit saat update)
	if err := ctl.DB.Model(&p).
		Omit(
			"school_profile_id",
			"school_profile_school_id",
			"school_profile_search",
			"school_profile_created_at",
		).
		Select("*").
		Updates(&p).Error; err != nil {

		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Duplikasi NPSN/NSS")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update: "+err.Error())
	}

	// Reload untuk updated_at
	if err := ctl.DB.First(&p, "school_profile_id = ?", id).Error; err != nil {
		log.Println("[WARN] reload after update:", err)
	}

	return helper.JsonUpdated(c, "Profil school berhasil diperbarui", d.FromModelSchoolProfile(&p))
}

// DELETE (soft) — hanya Owner
// DELETE /admin/:school_id/school-profiles/:id
func (ctl *SchoolProfileController) Delete(c *fiber.Ctx) error {
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
	var p m.SchoolProfileModel
	if err := ctl.DB.
		Where("school_profile_id = ? AND school_profile_deleted_at IS NULL", id).
		First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// Soft delete pakai timestamp
	if err := ctl.DB.
		Model(&m.SchoolProfileModel{}).
		Where("school_profile_id = ?", id).
		Update("school_profile_deleted_at", time.Now()).
		Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus: "+err.Error())
	}

	return helper.JsonDeleted(c, "Profil school dihapus (soft delete)", fiber.Map{"deleted": true})
}