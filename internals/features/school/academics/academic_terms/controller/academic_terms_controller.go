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

	dto "madinahsalam_backend/internals/features/school/academics/academic_terms/dto"

	termModel "madinahsalam_backend/internals/features/school/academics/academic_terms/model"
	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	classModel "madinahsalam_backend/internals/features/school/classes/classes/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	academicTermsService "madinahsalam_backend/internals/features/school/academics/academic_terms/service"
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

// helper: name + academic_year (MODEL string) — anti dobel suffix year
func buildTermDisplayNameStr(name string, ay string) string {
	n := strings.TrimSpace(name)
	y := strings.TrimSpace(ay)
	if y == "" {
		return n
	}

	// anti "2027 2027" / "2026/27 2026/27"
	ln := strings.ToLower(n)
	ly := strings.ToLower(y)

	// kalau sudah mengandung year sebagai token terakhir, jangan append lagi
	if strings.HasSuffix(ln, " "+ly) || ln == ly {
		return n
	}

	return strings.TrimSpace(n + " " + y)
}

/* =========================================================
   CREATE (DKM only)
   POST /admin/academic-terms
========================================================= */

func (ctl *AcademicTermController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	var p dto.AcademicTermCreateDTO
	if err := bindAndValidate(c, ctl.Validator, &p); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return httpErr(c, fe.Code, fe.Message)
		}
		return httpErr(c, fiber.StatusBadRequest, err.Error())
	}
	p.Normalize()

	if p.AcademicTermEndDate.Before(p.AcademicTermStartDate) {
		return httpErr(c, fiber.StatusBadRequest, "Tanggal akhir harus >= tanggal mulai")
	}

	// === School context dari TOKEN (DKM only) ===
	schoolID, err := helperAuth.GetActiveSchoolIDFromToken(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return httpErr(c, fe.Code, fe.Message)
		}
		return httpErr(c, fiber.StatusUnauthorized, err.Error())
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	// Uniqueness per school untuk code (opsional)
	if p.AcademicTermCode != nil && strings.TrimSpace(*p.AcademicTermCode) != "" {
		var cnt int64
		if err := ctl.DB.Model(&termModel.AcademicTermModel{}).
			Where("academic_term_school_id = ? AND academic_term_code = ?", schoolID, *p.AcademicTermCode).
			Count(&cnt).Error; err != nil {
			return httpErr(c, fiber.StatusInternalServerError, "Gagal memeriksa kode")
		}
		if cnt > 0 {
			return httpErr(c, fiber.StatusConflict, "Kode tahun akademik sudah dipakai")
		}
	}

	// === Display name: name + academic_year ===
	name := strings.TrimSpace(p.AcademicTermName)
	if name == "" {
		return httpErr(c, fiber.StatusBadRequest, "Nama tahun akademik wajib diisi")
	}

	displayName := buildTermDisplayNameStr(name, p.AcademicTermAcademicYear)

	p.AcademicTermName = displayName // OPTIONAL (kalau mau simpan display ke DB)
	baseSlug := helper.Slugify(displayName, 50)

	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(),
		ctl.DB,
		"academic_terms",
		"academic_term_slug",
		baseSlug,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("academic_term_school_id = ?", schoolID)
		},
		50,
	)
	if err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}

	ent := p.ToModel(schoolID)
	ent.AcademicTermSlug = &uniqueSlug

	if err := ctl.DB.Create(&ent).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal membuat data")
	}
	return helper.JsonCreated(c, "Berhasil membuat tahun akademik", dto.FromModel(c, ent))
}

/*
	=========================================================
	  PATCH (DKM only)
	  PATCH /admin/academic-terms/:id

=========================================================
*/
func (ctl *AcademicTermController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return httpErr(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// === School context dari TOKEN (DKM only) ===
	schoolID, err := helperAuth.GetActiveSchoolIDFromToken(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return httpErr(c, fe.Code, fe.Message)
		}
		return httpErr(c, fiber.StatusUnauthorized, err.Error())
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	var out dto.AcademicTermResponseDTO

	txErr := ctl.DB.Transaction(func(tx *gorm.DB) error {
		var ent termModel.AcademicTermModel
		if err := tx.
			Where("academic_term_school_id = ? AND academic_term_id = ?", schoolID, id).
			First(&ent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpErr(c, fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data")
		}

		var p dto.AcademicTermUpdateDTO
		if err := bindAndValidate(c, ctl.Validator, &p); err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return httpErr(c, fe.Code, fe.Message)
			}
			return httpErr(c, fiber.StatusBadRequest, err.Error())
		}

		// ---------- Normalisasi ----------
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
		if p.AcademicTermName != nil {
			s := strings.TrimSpace(*p.AcademicTermName)
			p.AcademicTermName = &s
		}

		// Abaikan slug dari payload
		p.AcademicTermSlug = nil

		// ---------- Validasi tanggal ----------
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

		// ---------- Uniqueness kode ----------
		if p.AcademicTermCode != nil && strings.TrimSpace(*p.AcademicTermCode) != "" {
			var cnt int64
			if err := tx.Model(&termModel.AcademicTermModel{}).
				Where(`academic_term_school_id = ? AND academic_term_code = ? AND academic_term_id <> ?`,
					schoolID, *p.AcademicTermCode, ent.AcademicTermID).
				Count(&cnt).Error; err != nil {
				return httpErr(c, fiber.StatusInternalServerError, "Gagal memeriksa kode")
			}
			if cnt > 0 {
				return httpErr(c, fiber.StatusConflict, "Kode tahun akademik sudah dipakai")
			}
		}

		// ---------- Simpan nilai lama untuk deteksi refresh ----------
		oldAY := ent.AcademicTermAcademicYear
		oldName := ent.AcademicTermName
		oldAngkatan := ent.AcademicTermAngkatan

		// ---------- Apply patch ----------
		p.ApplyUpdates(&ent)

		// ✅ Pastikan name selalu "name + academic_year" kalau salah satunya berubah
		if p.AcademicTermName != nil || p.AcademicTermAcademicYear != nil {
			nm := strings.TrimSpace(ent.AcademicTermName)
			if nm == "" {
				return httpErr(c, fiber.StatusBadRequest, "Nama tahun akademik wajib diisi")
			}

			// bikin displayName anti dobel "2027 2027"
			displayName := buildTermDisplayNameStr(ent.AcademicTermName, ent.AcademicTermAcademicYear)
			ent.AcademicTermName = displayName

			baseSlug := helper.Slugify(displayName, 50)
			uniqueSlug, err := helper.EnsureUniqueSlugCI(
				c.Context(), tx,
				"academic_terms", "academic_term_slug",
				baseSlug,
				func(q *gorm.DB) *gorm.DB {
					return q.Where("academic_term_school_id = ? AND academic_term_id <> ?", schoolID, ent.AcademicTermID)
				},
				50,
			)
			if err != nil {
				return httpErr(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			ent.AcademicTermSlug = &uniqueSlug
		}

		ent.AcademicTermUpdatedAt = time.Now()

		// ---------- Save term ----------
		if err := tx.Save(&ent).Error; err != nil {
			return httpErr(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
		}

		// ---------- Need refresh classes cache? ----------
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
			cacheSvc := academicTermsService.NewAcademicTermCacheService()

			// 1) refresh classes
			if err := cacheSvc.RefreshClassesForTerm(c.Context(), tx, schoolID, &ent); err != nil {
				return httpErr(c, fiber.StatusInternalServerError, "Gagal menyegarkan cache kelas: "+err.Error())
			}

			// 2) refresh class_sections
			if err := cacheSvc.RefreshClassSectionsForTerm(c.Context(), tx, schoolID, &ent); err != nil {
				return httpErr(c, fiber.StatusInternalServerError, "Gagal menyegarkan cache rombel: "+err.Error())
			}

			// 3) refresh CSST
			if err := cacheSvc.RefreshCSSTForTerm(c.Context(), tx, schoolID, &ent); err != nil {
				return httpErr(c, fiber.StatusInternalServerError, "Gagal menyegarkan cache CSST: "+err.Error())
			}
		}

		out = dto.FromModel(c, ent)
		return nil
	})

	if txErr != nil {
		return txErr
	}

	return helper.JsonUpdated(c,
		"Berhasil memperbarui tahun akademik & menyegarkan cache kelas",
		out,
	)
}

/* =========================================================
   DELETE (soft) — DKM only
   DELETE /admin/academic-terms/:id
========================================================= */

func (ctl *AcademicTermController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return httpErr(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// === School context dari TOKEN (DKM only) ===
	schoolID, err := helperAuth.GetActiveSchoolIDFromToken(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return httpErr(c, fe.Code, fe.Message)
		}
		return httpErr(c, fiber.StatusUnauthorized, err.Error())
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	var ent termModel.AcademicTermModel
	if err := ctl.DB.
		Where("academic_term_school_id = ? AND academic_term_id = ?", schoolID, id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpErr(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// CEGAH delete kalau masih ada kelas yang pakai term ini
	var classCount int64
	if err := ctl.DB.
		Model(&classModel.ClassModel{}).
		Where("class_school_id = ? AND class_academic_term_id = ? AND class_deleted_at IS NULL", schoolID, id).
		Count(&classCount).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengecek relasi kelas")
	}

	// CEGAH delete kalau masih ada class_sections yang pakai term ini
	var sectionCount int64
	if err := ctl.DB.
		Model(&classSectionModel.ClassSectionModel{}).
		Where("class_section_school_id = ? AND class_section_academic_term_id = ? AND class_section_deleted_at IS NULL", schoolID, id).
		Count(&sectionCount).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengecek relasi rombel")
	}

	if classCount > 0 || sectionCount > 0 {
		return httpErr(c, fiber.StatusConflict,
			"Tidak dapat menghapus tahun akademik karena masih ada kelas/rombel yang menggunakan tahun akademik ini. Silakan nonaktifkan atau pindahkan kelas & rombel terlebih dahulu.",
		)
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
	c.Locals("DB", ctl.DB)

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return httpErr(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// === School context dari TOKEN (DKM only) ===
	schoolID, err := helperAuth.GetActiveSchoolIDFromToken(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return httpErr(c, fe.Code, fe.Message)
		}
		return httpErr(c, fiber.StatusUnauthorized, err.Error())
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	var ent termModel.AcademicTermModel
	if err := ctl.DB.Unscoped().
		Where("academic_term_school_id = ? AND academic_term_id = ?", schoolID, id).
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
		Where("academic_term_school_id = ? AND academic_term_id = ?", schoolID, id).
		First(&ent).Error; err != nil {
		return httpErr(c, fiber.StatusInternalServerError, "Gagal mengambil data setelah restore")
	}
	return helper.JsonUpdated(c, "Berhasil merestore tahun akademik", dto.FromModel(c, ent))
}
