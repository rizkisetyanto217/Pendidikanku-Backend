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
	classmodel "masjidku_backend/internals/features/school/classes/classes/model"
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

/* =========================================================
   CREATE (DKM only)
   POST /admin/academic-terms
========================================================= */

func (ctl *AcademicTermController) Create(c *fiber.Ctx) error {
	var p dto.AcademicTermCreateDTO
	if err := bindAndValidate(c, ctl.Validator, &p); err != nil {
		return httpErr(c, err.(*fiber.Error).Code, err.Error())
	}
	p.Normalize()

	if p.AcademicTermEndDate.Before(p.AcademicTermStartDate) {
		return httpErr(c, fiber.StatusBadRequest, "Tanggal akhir harus >= tanggal mulai")
	}

	// === Masjid context (eksplisit & DKM only) ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return httpErr(c, err.(*fiber.Error).Code, err.Error())
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return httpErr(c, err.(*fiber.Error).Code, err.Error())
	}

	// Uniqueness per masjid untuk code (opsional)
	if p.AcademicTermCode != nil && strings.TrimSpace(*p.AcademicTermCode) != "" {
		var cnt int64
		if err := ctl.DB.Model(&model.AcademicTermModel{}).
			Where("academic_term_masjid_id = ? AND academic_term_code = ?", masjidID, *p.AcademicTermCode).
			Count(&cnt).Error; err != nil {
			return httpErr(c, fiber.StatusInternalServerError, "Gagal memeriksa kode")
		}
		if cnt > 0 {
			return httpErr(c, fiber.StatusConflict, "Kode tahun akademik sudah dipakai")
		}
	}

	// === Slug dari academic_year (abaikan slug dari payload) ===
	ay := strings.TrimSpace(p.AcademicTermAcademicYear)
	if ay == "" {
		return httpErr(c, fiber.StatusBadRequest, "Academic year wajib diisi")
	}
	baseSlug := helper.Slugify(ay, 50) // kolom varchar(50)

	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(),
		ctl.DB,
		"academic_terms",
		"academic_term_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("academic_term_masjid_id = ?", masjidID)
		},
		50,
	)
	if err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}

	ent := p.ToModel(masjidID)
	ent.AcademicTermSlug = &uniqueSlug

	if err := ctl.DB.Create(&ent).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal membuat data")
	}
	return helper.JsonCreated(c, "Berhasil membuat tahun akademik", dto.FromModel(ent))
}

/* =========================================================
   PUT/PATCH (DKM only)
   PUT /admin/academic-terms/:id
   PATCH /admin/academic-terms/:id
========================================================= */

func (ctl *AcademicTermController) Update(c *fiber.Ctx) error { return ctl.updateCommon(c, false) }
func (ctl *AcademicTermController) Patch(c *fiber.Ctx) error  { return ctl.updateCommon(c, true) }
func (ctl *AcademicTermController) updateCommon(c *fiber.Ctx, _ bool) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return httpErr(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// === Masjid context (eksplisit & DKM only), sama seperti Create ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return httpErr(c, err.(*fiber.Error).Code, err.Error())
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return httpErr(c, err.(*fiber.Error).Code, err.Error())
	}

	var out dto.AcademicTermResponseDTO

	txErr := ctl.DB.Transaction(func(tx *gorm.DB) error {
		var ent model.AcademicTermModel
		if err := tx.
			Where("academic_term_masjid_id = ? AND academic_term_id = ?", masjidID, id).
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

		// === Samakan normalisasi dengan Create ===
		if p.AcademicTermCode != nil {
			s := strings.TrimSpace(*p.AcademicTermCode)
			p.AcademicTermCode = &s
		}
		if p.AcademicTermDescription != nil {
			s := strings.TrimSpace(*p.AcademicTermDescription)
			p.AcademicTermDescription = &s
		}
		if p.AcademicTermAcademicYear != nil {
			s := strings.TrimSpace(*p.AcademicTermAcademicYear)
			p.AcademicTermAcademicYear = &s
		}

		// === Abaikan slug dari payload (selaras Create) ===
		// Force abaikan apapun yang dikirim user pada PATCH
		p.AcademicTermSlug = nil

		// Validasi tanggal jika diubah
		if p.AcademicTermStartDate != nil || p.AcademicTermEndDate != nil {
			start := ent.AcademicTermStartDate
			end := ent.AcademicTermEndDate
			if p.AcademicTermStartDate != nil {
				start = *p.AcademicTermStartDate
			}
			if p.AcademicTermEndDate != nil {
				end = *p.AcademicTermEndDate
			}
			if end.Before(start) {
				return httpErr(c, fiber.StatusBadRequest, "Tanggal akhir harus >= tanggal mulai")
			}
		}

		// Uniqueness kode jika berubah
		if p.AcademicTermCode != nil && strings.TrimSpace(*p.AcademicTermCode) != "" {
			var cnt int64
			if err := tx.Model(&model.AcademicTermModel{}).
				Where(`academic_term_masjid_id = ? AND academic_term_code = ? AND academic_term_id <> ?`,
					masjidID, *p.AcademicTermCode, ent.AcademicTermID).
				Count(&cnt).Error; err != nil {
				return httpErr(c, fiber.StatusInternalServerError, "Gagal memeriksa kode")
			}
			if cnt > 0 {
				return httpErr(c, fiber.StatusConflict, "Kode tahun akademik sudah dipakai")
			}
		}

		// Simpan nilai lama untuk deteksi refresh snapshot classes
		oldAY := ent.AcademicTermAcademicYear
		oldName := ent.AcademicTermName
		oldAngkatan := ent.AcademicTermAngkatan

		// Terapkan perubahan (kecuali slug, sudah dipaksa nil di atas)
		p.ApplyUpdates(&ent)

		// === Regenerasi slug hanya jika Academic Year berubah (selaras Create) ===
		if p.AcademicTermAcademicYear != nil {
			ay := strings.TrimSpace(*p.AcademicTermAcademicYear)
			if ay == "" {
				return httpErr(c, fiber.StatusBadRequest, "Academic year wajib diisi")
			}
			baseSlug := helper.Slugify(ay, 50)
			uniqueSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(), tx,
				"academic_terms", "academic_term_slug",
				baseSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where("academic_term_masjid_id = ? AND academic_term_id <> ?", masjidID, ent.AcademicTermID)
				},
				50,
			)
			if err != nil {
				return httpErr(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			ent.AcademicTermSlug = &uniqueSlug
		}

		ent.AcademicTermUpdatedAt = time.Now()

		// Simpan perubahan term
		if err := tx.Save(&ent).Error; err != nil {
			return httpErr(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
		}

		// Hitung perlu refresh snapshot classes (AY/Name/Angkatan)
		needRefresh := false
		if p.AcademicTermAcademicYear != nil && ent.AcademicTermAcademicYear != oldAY {
			needRefresh = true
		}
		if p.AcademicTermName != nil && ent.AcademicTermName != oldName {
			needRefresh = true
		}
		if p.AcademicTermAngkatan != nil {
			switch {
			case (oldAngkatan == nil && ent.AcademicTermAngkatan != nil),
				(oldAngkatan != nil && ent.AcademicTermAngkatan == nil):
				needRefresh = true
			case (oldAngkatan != nil && ent.AcademicTermAngkatan != nil) &&
				(*oldAngkatan != *ent.AcademicTermAngkatan):
				needRefresh = true
			}
		}

		if needRefresh {
			if err := tx.Model(&classmodel.ClassModel{}).
				Where("class_masjid_id = ? AND class_term_id = ?", masjidID, ent.AcademicTermID).
				Updates(map[string]any{
					"class_academic_year_term_snapshot": ent.AcademicTermAcademicYear,
					"class_name_term_snapshot":          ent.AcademicTermName,
					"class_slug_term_snapshot":          ent.AcademicTermSlug,
					"class_angkatan_term_snapshot":      ent.AcademicTermAngkatan,
					"class_updated_at":                  time.Now(),
				}).Error; err != nil {
				return httpErr(c, fiber.StatusInternalServerError, "Gagal menyegarkan snapshot kelas")
			}
		}

		out = dto.FromModel(ent)
		return nil
	})

	if txErr != nil {
		return txErr
	}

	return helper.JsonUpdated(c,
		"Berhasil memperbarui tahun akademik & menyegarkan snapshot kelas",
		out,
	)
}

/* =========================================================
   DELETE (soft) — DKM only
   DELETE /admin/academic-terms/:id
========================================================= */

func (ctl *AcademicTermController) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return httpErr(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// === Masjid context (eksplisit & DKM only) ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return httpErr(c, err.(*fiber.Error).Code, err.Error())
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return httpErr(c, err.(*fiber.Error).Code, err.Error())
	}

	var ent model.AcademicTermModel
	if err := ctl.DB.
		Where("academic_term_masjid_id = ? AND academic_term_id = ?", masjidID, id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if err := ctl.DB.Delete(&ent).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	return helper.JsonDeleted(c, "Berhasil menghapus tahun akademik", fiber.Map{
		"academic_term_id": id,
	})
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
		Where("academic_term_masjid_id = ? AND academic_term_id = ?", masjidID, id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "Data tidak ditemukan / belum pernah ada")
		}
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if err := ctl.DB.Unscoped().
		Model(&ent).
		Update("academic_term_deleted_at", nil).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal merestore data")
	}

	if err := ctl.DB.
		Where("academic_term_masjid_id = ? AND academic_term_id = ?", masjidID, id).
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
		Where("academic_term_masjid_id = ? AND academic_term_id = ?", masjidID, id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Jika ingin eksklusif hanya 1 aktif per masjid, uncomment blok ini:
	// if err := ctl.DB.Model(&model.AcademicTerm{}).
	// 	Where("academic_term_masjid_id = ? AND academic_term_id <> ?", masjidID, id).
	// 	Update("academic_term_is_active", false).Error; err != nil {
	// 	return httpErr(c, fiber.StatusInternalServerError, "Gagal menonaktifkan term lain")
	// }

	if err := ctl.DB.Model(&ent).Update("academic_term_is_active", true).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengaktifkan term")
	}
	if err := ctl.DB.First(&ent, "academic_term_id = ?", id).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal refresh data")
	}
	return helper.JsonUpdated(c, "Berhasil mengaktifkan tahun akademik", dto.FromModel(ent))
}
