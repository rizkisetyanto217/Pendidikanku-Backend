// file: internals/features/academics/terms/controller/academic_term_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/academics/academic_terms/dto"
	model "masjidku_backend/internals/features/school/academics/academic_terms/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* ============================================
   Controller
============================================ */

type AcademicTermController struct {
    DB        *gorm.DB
    Validator *validator.Validate
}

func NewAcademicTermController(db *gorm.DB, v *validator.Validate) *AcademicTermController {
    if v == nil {
        v = validator.New()
    }
    return &AcademicTermController{DB: db, Validator: v}
}

/* ============================================
   RESP/ERR helpers
============================================ */

func httpErr(c *fiber.Ctx, code int, msg string) error {
	return helper.JsonError(c, code, msg)
}

func bindAndValidate[T any](c *fiber.Ctx, v *validator.Validate, dst *T) error {
	if err := c.BodyParser(dst); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if v != nil {
		if err := v.Struct(dst); err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
		}
	}
	return nil
}

/* ============================================
   CREATE (DKM only)
   POST /admin/academic-terms
============================================ */

func (ctl *AcademicTermController) Create(c *fiber.Ctx) error {
	var p dto.AcademicTermCreateDTO
	if err := bindAndValidate(c, ctl.Validator, &p); err != nil {
		return httpErr(c, err.(*fiber.Error).Code, err.Error())
	}
	p.Normalize()

	if p.AcademicTermsEndDate.Before(p.AcademicTermsStartDate) {
		return httpErr(c, fiber.StatusBadRequest, "Tanggal akhir harus >= tanggal mulai")
	}

	masjidID, err := helperAuth.GetActiveMasjidID(c)
	if err != nil {
		if id2, err2 := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err2 == nil {
			masjidID = id2
		} else {
			return httpErr(c, fiber.StatusUnauthorized, "Masjid context tidak ditemukan")
		}
	}

	// === DKM only ===
	if err := helperAuth.EnsureDKMMasjid(c, masjidID); err != nil {
		return err
	}

	// Uniqueness check (per masjid) untuk code/slug jika diisi
	if strings.TrimSpace(p.AcademicTermsCode) != "" {
		var cnt int64
		if err := ctl.DB.Model(&model.AcademicTermModel{}).
			Where("academic_terms_masjid_id = ? AND academic_terms_code = ?", masjidID, p.AcademicTermsCode).
			Count(&cnt).Error; err != nil {
			return httpErr(c, fiber.StatusInternalServerError, "Gagal memeriksa kode")
		}
		if cnt > 0 {
			return httpErr(c, fiber.StatusConflict, "Kode tahun akademik sudah dipakai")
		}
	}
	if strings.TrimSpace(p.AcademicTermsSlug) != "" {
		var cnt int64
		if err := ctl.DB.Model(&model.AcademicTermModel{}).
			Where("academic_terms_masjid_id = ? AND academic_terms_slug = ?", masjidID, p.AcademicTermsSlug).
			Count(&cnt).Error; err != nil {
			return httpErr(c, fiber.StatusInternalServerError, "Gagal memeriksa slug")
		}
		if cnt > 0 {
			return httpErr(c, fiber.StatusConflict, "Slug tahun akademik sudah dipakai")
		}
	}

	ent := p.ToModel(masjidID)
	if err := ctl.DB.Create(&ent).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal membuat data")
	}
	return helper.JsonCreated(c, "Berhasil membuat tahun akademik", dto.FromModel(ent))
}

/* ============================================
   PUT/PATCH (DKM only)
   PUT /admin/academic-terms/:id
   PATCH /admin/academic-terms/:id
============================================ */

func (ctl *AcademicTermController) Update(c *fiber.Ctx) error { return ctl.updateCommon(c, false) }
func (ctl *AcademicTermController) Patch(c *fiber.Ctx) error  { return ctl.updateCommon(c, true) }

func (ctl *AcademicTermController) updateCommon(c *fiber.Ctx, _ bool) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return httpErr(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	masjidID, err := helperAuth.GetActiveMasjidID(c)
	if err != nil {
		if id2, err2 := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err2 == nil {
			masjidID = id2
		} else {
			return httpErr(c, fiber.StatusUnauthorized, "Masjid context tidak ditemukan")
		}
	}

	// === DKM only
	if err := helperAuth.EnsureDKMMasjid(c, masjidID); err != nil {
		return err
	}

	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_terms_masjid_id = ? AND academic_terms_id = ?", masjidID, id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	var p dto.AcademicTermUpdateDTO
	if err := bindAndValidate(c, ctl.Validator, &p); err != nil {
		return httpErr(c, err.(*fiber.Error).Code, err.Error())
	}

	// Normalisasi minimal
	if p.AcademicTermsSlug != nil {
		s := strings.ToLower(strings.TrimSpace(*p.AcademicTermsSlug))
		p.AcademicTermsSlug = &s
	}
	if p.AcademicTermsCode != nil {
		s := strings.TrimSpace(*p.AcademicTermsCode)
		p.AcademicTermsCode = &s
	}
	if p.AcademicTermsDescription != nil {
		s := strings.TrimSpace(*p.AcademicTermsDescription)
		p.AcademicTermsDescription = &s
	}

	// Validasi tanggal jika diubah
	if p.AcademicTermsStartDate != nil || p.AcademicTermsEndDate != nil {
		start := ent.AcademicTermsStartDate
		end := ent.AcademicTermsEndDate
		if p.AcademicTermsStartDate != nil {
			start = *p.AcademicTermsStartDate
		}
		if p.AcademicTermsEndDate != nil {
			end = *p.AcademicTermsEndDate
		}
		if end.Before(start) {
			return httpErr(c, fiber.StatusBadRequest, "Tanggal akhir harus >= tanggal mulai")
		}
	}

	// Uniqueness check jika code/slug berubah
	if p.AcademicTermsCode != nil {
		var cnt int64
		if err := ctl.DB.Model(&model.AcademicTermModel{}).
			Where("academic_terms_masjid_id = ? AND academic_terms_code = ? AND academic_terms_id <> ?",
				masjidID, *p.AcademicTermsCode, ent.AcademicTermsID).
			Count(&cnt).Error; err != nil {
			return httpErr(c, fiber.StatusInternalServerError, "Gagal memeriksa kode")
		}
		if cnt > 0 {
			return httpErr(c, fiber.StatusConflict, "Kode tahun akademik sudah dipakai")
		}
	}
	if p.AcademicTermsSlug != nil && strings.TrimSpace(*p.AcademicTermsSlug) != "" {
		var cnt int64
		if err := ctl.DB.Model(&model.AcademicTermModel{}).
			Where("academic_terms_masjid_id = ? AND academic_terms_slug = ? AND academic_terms_id <> ?",
				masjidID, *p.AcademicTermsSlug, ent.AcademicTermsID).
			Count(&cnt).Error; err != nil {
			return httpErr(c, fiber.StatusInternalServerError, "Gagal memeriksa slug")
		}
		if cnt > 0 {
			return httpErr(c, fiber.StatusConflict, "Slug tahun akademik sudah dipakai")
		}
	}

	// Terapkan perubahan
	p.ApplyUpdates(&ent)
	ent.AcademicTermsUpdatedAt = time.Now()

	if err := ctl.DB.Save(&ent).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}
	return helper.JsonUpdated(c, "Berhasil memperbarui tahun akademik", dto.FromModel(ent))
}

/* ============================================
   DELETE (soft) — DKM only
   DELETE /admin/academic-terms/:id
============================================ */

func (ctl *AcademicTermController) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return httpErr(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	masjidID, err := helperAuth.GetActiveMasjidID(c)
	if err != nil {
		if id2, err2 := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err2 == nil {
			masjidID = id2
		} else {
			return httpErr(c, fiber.StatusUnauthorized, "Masjid context tidak ditemukan")
		}
	}

	// === DKM only
	if err := helperAuth.EnsureDKMMasjid(c, masjidID); err != nil {
		return err
	}

	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_terms_masjid_id = ? AND academic_terms_id = ?", masjidID, id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if err := ctl.DB.Delete(&ent).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	// kirim id yang dihapus biar jelas
	return helper.JsonDeleted(c, "Berhasil menghapus tahun akademik", fiber.Map{"academic_terms_id": id})
}

/* ============================================
   RESTORE (soft-deleted) — DKM only
   POST /admin/academic-terms/:id/restore
============================================ */

func (ctl *AcademicTermController) Restore(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return httpErr(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	masjidID, err := helperAuth.GetActiveMasjidID(c)
	if err != nil {
		if id2, err2 := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err2 == nil {
			masjidID = id2
		} else {
			return httpErr(c, fiber.StatusUnauthorized, "Masjid context tidak ditemukan")
		}
	}

	// === DKM only
	if err := helperAuth.EnsureDKMMasjid(c, masjidID); err != nil {
		return err
	}

	var ent model.AcademicTermModel
	if err := ctl.DB.Unscoped().
		Where("academic_terms_masjid_id = ? AND academic_terms_id = ?", masjidID, id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "Data tidak ditemukan / belum pernah ada")
		}
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if ent.AcademicTermsDeletedAt.Valid == false {
		return helper.JsonOK(c, "OK", dto.FromModel(ent))
	}

	if err := ctl.DB.Unscoped().
		Model(&ent).
		Update("academic_terms_deleted_at", nil).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal merestore data")
	}

	if err := ctl.DB.
		Where("academic_terms_masjid_id = ? AND academic_terms_id = ?", masjidID, id).
		First(&ent).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data setelah restore")
	}
	return helper.JsonUpdated(c, "Berhasil merestore tahun akademik", dto.FromModel(ent))
}

/* ============================================
   Set Active — DKM only (opsional)
   PATCH /admin/academic-terms/:id/set-active
============================================ */

func (ctl *AcademicTermController) SetActive(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return httpErr(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	masjidID, err := helperAuth.GetActiveMasjidID(c)
	if err != nil {
		if id2, err2 := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err2 == nil {
			masjidID = id2
		} else {
			return httpErr(c, fiber.StatusUnauthorized, "Masjid context tidak ditemukan")
		}
	}

	// === DKM only
	if err := helperAuth.EnsureDKMMasjid(c, masjidID); err != nil {
		return err
	}

	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_terms_masjid_id = ? AND academic_terms_id = ?", masjidID, id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Jika ingin eksklusif hanya 1 aktif per masjid, uncomment blok ini:
	// if err := ctl.DB.Model(&model.AcademicTermModel{}).
	// 	Where("academic_terms_masjid_id = ? AND academic_terms_id <> ?", masjidID, id).
	// 	Update("academic_terms_is_active", false).Error; err != nil {
	// 	return httpErr(c, fiber.StatusInternalServerError, "Gagal menonaktifkan term lain")
	// }

	if err := ctl.DB.Model(&ent).Update("academic_terms_is_active", true).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengaktifkan term")
	}
	if err := ctl.DB.First(&ent, "academic_terms_id = ?", id).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal refresh data")
	}
	return helper.JsonUpdated(c, "Berhasil mengaktifkan tahun akademik", dto.FromModel(ent))
}
