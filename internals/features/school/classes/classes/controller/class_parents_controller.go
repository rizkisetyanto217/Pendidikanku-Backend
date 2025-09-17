// file: internals/features/school/classes/classes/controller/class_parent_controller.go
package controller

import (
	"strings"
	"time"

	cpdto "masjidku_backend/internals/features/school/classes/classes/dto"
	cpmodel "masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassParentController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewClassParentController(db *gorm.DB, v interface{ Struct(any) error }) *ClassParentController {
	return &ClassParentController{DB: db, Validator: v}
}

/* =========================================================
   CREATE (staff only)
   ========================================================= */
func (ctl *ClassParentController) Create(c *fiber.Ctx) error {
	var p cpdto.ClassParentCreateRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	p.Normalize()

	// ---------- MASJID CONTEXT + STAFF ONLY ----------
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	} else {
		// fallback very last resort (single-tenant teacher/admin token)
		id, er := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Masjid context tidak ditemukan")
		}
		masjidID = id
	}

	// staff check (teacher/dkm/admin/bendahara/owner/superadmin)
	if err := helperAuth.EnsureStaffMasjid(c, masjidID); err != nil {
		return err
	}

	// Paksa body sesuai context
	if p.ClassParentMasjidID == uuid.Nil {
		p.ClassParentMasjidID = masjidID
	} else if p.ClassParentMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusConflict, "class_parent_masjid_id pada body tidak cocok dengan konteks masjid")
	}

	// --- Uniqueness per masjid ---
	if p.ClassParentCode != nil {
		code := strings.TrimSpace(*p.ClassParentCode)
		if code != "" {
			var cnt int64
			if err := ctl.DB.Model(&cpmodel.ClassParentModel{}).
				Where("class_parent_masjid_id = ? AND class_parent_code = ? AND class_parent_deleted_at IS NULL",
					masjidID, code).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek kode")
			}
			if cnt > 0 {
				return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan")
			}
		}
	}
	if p.ClassParentSlug != nil {
		slug := strings.ToLower(strings.TrimSpace(*p.ClassParentSlug))
		if slug != "" {
			var cnt int64
			if err := ctl.DB.Model(&cpmodel.ClassParentModel{}).
				Where("class_parent_masjid_id = ? AND class_parent_slug = ? AND class_parent_deleted_at IS NULL",
					masjidID, slug).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek slug")
			}
			if cnt > 0 {
				return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan")
			}
		}
	}

	ent := p.ToModel()
	if err := ctl.DB.Create(ent).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}
	return helper.JsonCreated(c, "Berhasil membuat parent kelas", cpdto.FromModelClassParent(ent))
}

/* =========================================================
   PATCH (staff only)
   ========================================================= */
func (ctl *ClassParentController) Patch(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	// Ambil record dulu agar tahu masjid_id
	var ent cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Akses staff pada masjid terkait (guard utama)
	if err := helperAuth.EnsureStaffMasjid(c, ent.ClassParentMasjidID); err != nil {
		return err
	}

	var p cpdto.ClassParentPatchRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	p.Normalize()

	// Uniqueness jika code/slug diubah
	if p.ClassParentCode.Present && p.ClassParentCode.Value != nil {
		if v := strings.TrimSpace(**p.ClassParentCode.Value); v != "" {
			var cnt int64
			if err := ctl.DB.Model(&cpmodel.ClassParentModel{}).
				Where("class_parent_masjid_id = ? AND class_parent_code = ? AND class_parent_id <> ? AND class_parent_deleted_at IS NULL",
					ent.ClassParentMasjidID, v, ent.ClassParentID).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek kode")
			}
			if cnt > 0 {
				return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan")
			}
		}
	}
	if p.ClassParentSlug.Present && p.ClassParentSlug.Value != nil {
		if v := strings.ToLower(strings.TrimSpace(**p.ClassParentSlug.Value)); v != "" {
			var cnt int64
			if err := ctl.DB.Model(&cpmodel.ClassParentModel{}).
				Where("class_parent_masjid_id = ? AND class_parent_slug = ? AND class_parent_id <> ? AND class_parent_deleted_at IS NULL",
					ent.ClassParentMasjidID, v, ent.ClassParentID).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek slug")
			}
			if cnt > 0 {
				return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan")
			}
		}
	}

	// Terapkan perubahan ke entity
	p.Apply(&ent)

	if err := ctl.DB.WithContext(c.Context()).
		Model(&cpmodel.ClassParentModel{}).
		Where("class_parent_id = ?", ent.ClassParentID).
		Updates(&ent).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	// muat ulang (best effort)
	_ = ctl.DB.WithContext(c.Context()).
		First(&ent, "class_parent_id = ?", ent.ClassParentID).Error

	return helper.JsonOK(c, "Berhasil memperbarui parent kelas", cpdto.FromModelClassParent(&ent))
}

/* =========================================================
   DELETE (soft delete, staff only)
   ========================================================= */
func (ctl *ClassParentController) Delete(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var ent cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard akses staff pada masjid terkait
	if err := helperAuth.EnsureStaffMasjid(c, ent.ClassParentMasjidID); err != nil {
		return err
	}

	now := time.Now()
	if err := ctl.DB.WithContext(c.Context()).
		Model(&cpmodel.ClassParentModel{}).
		Where("class_parent_id = ?", ent.ClassParentID).
		Updates(map[string]any{
			"class_parent_deleted_at": &now,
			"class_parent_updated_at": now,
		}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonOK(c, "Berhasil menghapus parent kelas", fiber.Map{"class_parent_id": ent.ClassParentID})
}

/* =========================================================
   Util
   ========================================================= */
func clampLimit(v, def, max int) int {
	if v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}
