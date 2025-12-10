// file: internals/features/lembaga/academics/academic_terms/controller/controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	academicsDTO "madinahsalam_backend/internals/features/school/academics/academic_terms/dto"
	termModel "madinahsalam_backend/internals/features/school/academics/academic_terms/model"

	feeRuleModel "madinahsalam_backend/internals/features/finance/billings/model"
	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	classModel "madinahsalam_backend/internals/features/school/classes/classes/model"

	classDTO "madinahsalam_backend/internals/features/school/classes/classes/dto"

	classSectionDTO "madinahsalam_backend/internals/features/school/classes/class_sections/dto"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/* ================= Controller & Constructor ================= */

var fallbackValidator = validator.New()

/* ================= Handlers ================= */

// List academic terms + optional include
func (ctl *AcademicTermController) List(c *fiber.Ctx) error {
	// Biar helper lain yang baca dari Locals("DB") tetap bisa jalan
	if c.Locals("DB") == nil {
		c.Locals("DB", ctl.DB)
	}

	var schoolID uuid.UUID

	/* ========= 0) Parse include ========= */
	// ?include=classes,class_sections,fee_rules
	rawInclude := strings.TrimSpace(c.Query("include", ""))
	includeClasses := false
	includeSections := false
	includeFeeRules := false
	if rawInclude != "" {
		for _, part := range strings.Split(rawInclude, ",") {
			p := strings.ToLower(strings.TrimSpace(part))
			switch p {
			case "classes", "class":
				includeClasses = true
			case "class_sections", "sections", "class-section":
				includeSections = true
			case "fee_rules", "fee-rules", "feerules", "fees":
				includeFeeRules = true
			}
		}
	}

	/* ========= 0a) Parse term_mode: compact | full (HANYA untuk academic terms) ========= */
	// Prioritas:
	// 1) term_mode (baru, lebih eksplisit)
	// 2) fallback ke mode (kompat lama)
	termMode := strings.ToLower(strings.TrimSpace(c.Query("term_mode", "")))
	if termMode == "" {
		termMode = strings.ToLower(strings.TrimSpace(c.Query("mode", "full")))
	}
	useCompact := termMode == "compact"

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
	var q academicsDTO.AcademicTermFilterDTO
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

	// ==== Paging ====
	p := helper.ResolvePaging(c, 20, 100) // default 20, max 100

	// ==== Sorting whitelist (pakai DTO.SortBy / SortDir) ====
	sortBy := "created_at"
	if q.SortBy != nil && strings.TrimSpace(*q.SortBy) != "" {
		sortBy = strings.ToLower(strings.TrimSpace(*q.SortBy))
	}
	order := "desc"
	if q.SortDir != nil && strings.TrimSpace(*q.SortDir) != "" {
		order = strings.ToLower(strings.TrimSpace(*q.SortDir))
	}
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
	dbq := ctl.DB.Model(&termModel.AcademicTermModel{}).
		Where("academic_term_school_id = ? AND academic_term_deleted_at IS NULL", schoolID)

	// ===== Filters (TERM) pakai DTO =====

	// ID (uuid) dari DTO
	if q.ID != nil && strings.TrimSpace(*q.ID) != "" {
		idStr := strings.TrimSpace(*q.ID)
		termID, err := uuid.Parse(idStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id invalid")
		}
		dbq = dbq.Where("academic_term_id = ?", termID)
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

	// ===== Fetch data TERM =====
	var list []termModel.AcademicTermModel
	if err := dbq.
		Order(orderExpr).
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// ===== Pagination =====
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	// Konversi ke DTO (compact / full)
	var termDTOs interface{}
	if useCompact {
		termDTOs = academicsDTO.FromModelsToCompact(list)
	} else {
		termDTOs = academicsDTO.FromModels(list)
	}

	// ✅ 1) Kalau tidak request include sama sekali → pure list tanpa include
	if !includeClasses && !includeSections && !includeFeeRules {
		return helper.JsonList(c, "ok", termDTOs, pg)
	}

	// ✅ 2) Kalau nggak ada data, tapi include diminta → kosong tapi tetap ada key "include"
	if len(list) == 0 {
		includeMap := fiber.Map{}
		if includeClasses {
			includeMap["classes"] = []classDTO.ClassCompact{}
		}
		if includeSections {
			includeMap["class_sections"] = []classSectionDTO.ClassSectionCompactResponse{}
		}
		if includeFeeRules {
			includeMap["fee_rules"] = []feeRuleModel.FeeRuleModel{}
		}
		return helper.JsonListWithInclude(c, "ok", termDTOs, includeMap, pg)
	}

	/* ===================== INCLUDE ===================== */

	// Kumpulkan semua term_id
	termIDs := make([]uuid.UUID, 0, len(list))
	for _, t := range list {
		termIDs = append(termIDs, t.AcademicTermID)
	}

	// --- INCLUDE: classes (flat slice) ---
	var allClasses []classModel.ClassModel
	if includeClasses {
		dbClass := ctl.DB.Model(&classModel.ClassModel{}).
			Where("class_school_id = ? AND class_deleted_at IS NULL", schoolID).
			Where("class_academic_term_id IN ?", termIDs)

		// Tambahan filter untuk classes (opsional):
		if v := strings.TrimSpace(c.Query("class_status")); v != "" {
			dbClass = dbClass.Where("class_status = ?", v)
		}
		if v := strings.TrimSpace(c.Query("class_delivery_mode")); v != "" {
			dbClass = dbClass.Where("class_delivery_mode = ?", v)
		}
		if v := strings.TrimSpace(c.Query("class_class_parent_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				dbClass = dbClass.Where("class_class_parent_id = ?", id)
			}
		}

		if err := dbClass.Find(&allClasses).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Query classes failed: "+err.Error())
		}
	}

	// --- INCLUDE: class_sections (flat slice) ---
	var allSections []classSectionModel.ClassSectionModel
	if includeSections {
		dbSec := ctl.DB.Model(&classSectionModel.ClassSectionModel{}).
			Where("class_section_school_id = ? AND class_section_deleted_at IS NULL", schoolID).
			Where("class_section_academic_term_id IN ?", termIDs)

			// Tambahan filter untuk class_sections:
		if v := strings.TrimSpace(c.Query("class_section_status")); v != "" {
			status := strings.ToLower(v)
			switch status {
			case "active", "inactive", "completed":
				dbSec = dbSec.Where("class_section_status = ?", status)
			default:
				return helper.JsonError(
					c,
					fiber.StatusBadRequest,
					"class_section_status invalid (allowed: active|inactive|completed)",
				)
			}
		}

		if v := strings.TrimSpace(c.Query("class_section_class_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				dbSec = dbSec.Where("class_section_class_id = ?", id)
			}
		}
		if v := strings.TrimSpace(c.Query("class_section_class_parent_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				dbSec = dbSec.Where("class_section_class_parent_id = ?", id)
			}
		}

		if v := strings.TrimSpace(c.Query("class_section_class_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				dbSec = dbSec.Where("class_section_class_id = ?", id)
			}
		}
		if v := strings.TrimSpace(c.Query("class_section_class_parent_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				dbSec = dbSec.Where("class_section_class_parent_id = ?", id)
			}
		}

		if err := dbSec.Find(&allSections).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Query class_sections failed: "+err.Error())
		}
	}

	// --- INCLUDE: fee_rules (SPP, dll) (flat slice) ---
	var allFeeRules []feeRuleModel.FeeRuleModel
	if includeFeeRules {
		dbFee := ctl.DB.Model(&feeRuleModel.FeeRuleModel{}).
			Where("fee_rule_school_id = ? AND fee_rule_deleted_at IS NULL", schoolID).
			Where("fee_rule_term_id IN ?", termIDs)

		// (opsional) tambah filter ringan:
		if v := strings.TrimSpace(c.Query("fee_rule_scope")); v != "" {
			dbFee = dbFee.Where("fee_rule_scope = ?", v)
		}
		if v := strings.TrimSpace(c.Query("fee_rule_option_code")); v != "" {
			dbFee = dbFee.Where("LOWER(fee_rule_option_code) = ?", strings.ToLower(v))
		}

		if err := dbFee.Find(&allFeeRules).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Query fee_rules failed: "+err.Error())
		}
	}

	// ✅ 3) Build include map (singular) dan kirim pakai JsonListWithInclude
	includeMap := fiber.Map{}
	if includeClasses {
		// sebelumnya: includeMap["classes"] = allClasses
		includeMap["classes"] = classDTO.ToClassCompactList(allClasses)
	}
	if includeSections {
		// sebelumnya: includeMap["class_sections"] = allSections
		includeMap["class_sections"] = classSectionDTO.FromSectionModelsToCompact(allSections)
	}
	if includeFeeRules {
		includeMap["fee_rules"] = allFeeRules
	}

	return helper.JsonListWithInclude(c, "ok", termDTOs, includeMap, pg)

}
