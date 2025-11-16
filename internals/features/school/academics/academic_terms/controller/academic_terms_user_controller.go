// file: internals/features/lembaga/academics/academic_terms/controller/controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"schoolku_backend/internals/features/school/academics/academic_terms/dto"
	"schoolku_backend/internals/features/school/academics/academic_terms/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* ================= Controller & Constructor ================= */

var fallbackValidator = validator.New()

/* ================= Handlers ================= */

// PUBLIC (tapi aware token)
//
// Skenario:
// 1) Kalau user login dan token punya active_school → pakai school dari token.
// 2) Kalau tidak ada / gagal baca token → pakai konteks PUBLIC:
//   - /api/u/:school_id/academic-terms/list       (UUID di path)
//   - /api/u/:school_slug/academic-terms/list     (slug di path)
//
// 3) Kalau semua sumber gagal → ErrSchoolContextMissing.
func (ctl *AcademicTermController) List(c *fiber.Ctx) error {
	// Biar helper lain yang baca dari Locals("DB") tetap bisa jalan
	c.Locals("DB", ctl.DB)

	var schoolID uuid.UUID

	/* ========= 1) Coba dari TOKEN dulu (jika ada) ========= */
	if id, err := helperAuth.GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		// Kalau berhasil baca active school dari token → langsung pakai ini
		schoolID = id
	} else {
		/* ========= 2) Fallback: PUBLIC context (ID / slug) ========= */

		mc, err2 := helperAuth.ResolveSchoolContext(c)
		if err2 != nil {
			// ErrSchoolContextMissing atau fiber.Error dari helper
			return err2
		}

		if mc.ID != uuid.Nil {
			// Sudah ada ID langsung dari context
			schoolID = mc.ID
		} else if s := strings.TrimSpace(mc.Slug); s != "" {
			// mc.Slug bisa berisi:
			// - beneran slug,
			// - atau sebenarnya UUID yang dikirim via path (/:school_id)
			if id2, errParse := uuid.Parse(s); errParse == nil {
				// Kalau ternyata valid UUID → treat sebagai school_id
				schoolID = id2
			} else {
				// Beneran slug → resolve via DB
				id2, er := helperAuth.GetSchoolIDBySlug(c, s)
				if er != nil {
					if errors.Is(er, gorm.ErrRecordNotFound) {
						return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
					}
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
				}
				schoolID = id2
			}
		} else {
			// Tidak ada ID, tidak ada slug → context kurang
			return helperAuth.ErrSchoolContextMissing
		}
	}

	// ==== Query → DTO + validasi ====
	var q dto.AcademicTermFilterDTO
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid query: "+err.Error())
	}
	q.Normalize()

	v := ctl.Validator
	if v == nil {
		v = fallbackValidator
	}
	if err := v.Struct(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validation failed: "+err.Error())
	}

	// ==== Paging (helper baru) ====
	p := helper.ResolvePaging(c, 20, 100) // default 20, max 100

	// ==== Sorting whitelist (manual) ====
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by", "created_at")))
	order := strings.ToLower(strings.TrimSpace(c.Query("sort", "desc")))
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	colMap := map[string]string{
		"created_at": "academic_term_created_at",
		"updated_at": "academic_term_updated_at",
		"start_date": "academic_term_start_date",
		"end_date":   "academic_term_end_date",
		"name":       "academic_term_name",
		"year":       "academic_term_academic_year",
		"angkatan":   "academic_term_angkatan",
		"code":       "academic_term_code",
		"slug":       "academic_term_slug",
	}
	col, ok := colMap[sortBy]
	if !ok {
		col = colMap["created_at"]
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// ===== Base query =====
	dbq := ctl.DB.Model(&model.AcademicTermModel{}).
		Where("academic_term_school_id = ? AND academic_term_deleted_at IS NULL", schoolID)

	// ===== Filters =====
	// id (single; toleransi quotes / null / undefined)
	if rawID, has := c.Queries()["id"]; has {
		cleaned := strings.Trim(rawID, `"' {}`)
		low := strings.ToLower(strings.TrimSpace(cleaned))
		if cleaned != "" && low != "null" && low != "undefined" {
			if termID, err := uuid.Parse(cleaned); err == nil {
				dbq = dbq.Where("academic_term_id = ?", termID)
			}
		}
	}
	if q.Year != nil && strings.TrimSpace(*q.Year) != "" {
		dbq = dbq.Where("academic_term_academic_year = ?", strings.TrimSpace(*q.Year))
	}
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		dbq = dbq.Where("academic_term_name = ?", strings.TrimSpace(*q.Name))
	}
	if q.Code != nil && strings.TrimSpace(*q.Code) != "" {
		dbq = dbq.Where("academic_term_code = ?", strings.TrimSpace(*q.Code))
	}
	if q.Slug != nil && strings.TrimSpace(*q.Slug) != "" {
		dbq = dbq.Where("academic_term_slug = ?", strings.TrimSpace(*q.Slug))
	}
	if q.Active != nil {
		dbq = dbq.Where("academic_term_is_active = ?", *q.Active)
	}
	if q.Angkatan != nil {
		dbq = dbq.Where("academic_term_angkatan = ?", *q.Angkatan)
	}

	// ===== Count =====
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Count failed: "+err.Error())
	}

	// ===== Fetch data =====
	var list []model.AcademicTermModel
	if err := dbq.
		Order(orderExpr).
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// ===== Pagination (pakai helper) =====
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	// ===== Response (JsonList standar) =====
	return helper.JsonList(c, "ok", dto.FromModels(list), pg)
}
