// file: internals/features/lembaga/academics/academic_terms/controller/controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"schoolku_backend/internals/features/school/academics/academic_terms/dto"
	termModel "schoolku_backend/internals/features/school/academics/academic_terms/model"

	feeRuleModel "schoolku_backend/internals/features/finance/billings/model" // ⬅️ NEW
	classSectionModel "schoolku_backend/internals/features/school/classes/class_sections/model"
	classModel "schoolku_backend/internals/features/school/classes/classes/model"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* ================= Controller & Constructor ================= */

var fallbackValidator = validator.New()

// Struct khusus kalau ada include
type AcademicTermWithRelations struct {
	Term          dto.AcademicTermResponseDTO           `json:"term"`
	Classes       []classModel.ClassModel               `json:"classes,omitempty"`
	ClassSections []classSectionModel.ClassSectionModel `json:"class_sections,omitempty"`
	FeeRules      []feeRuleModel.FeeRule                `json:"fee_rules,omitempty"` // ⬅️ NEW
}

/* ================= Handlers ================= */

// List academic terms + optional include
func (ctl *AcademicTermController) List(c *fiber.Ctx) error {
	// Biar helper lain yang baca dari Locals("DB") tetap bisa jalan
	c.Locals("DB", ctl.DB)

	var schoolID uuid.UUID

	/* ========= 0) Parse include ========= */
	// ?include=classes,class_sections,fee_rules
	rawInclude := strings.TrimSpace(c.Query("include", ""))
	includeClasses := false
	includeSections := false
	includeFeeRules := false // ⬅️ NEW
	if rawInclude != "" {
		for _, part := range strings.Split(rawInclude, ",") {
			p := strings.ToLower(strings.TrimSpace(part))
			switch p {
			case "classes", "class":
				includeClasses = true
			case "class_sections", "sections", "class-section":
				includeSections = true
			case "fee_rules", "fee-rules", "feerules", "fees": // ⬅️ beberapa alias santai
				includeFeeRules = true
			}
		}
	}

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
	dbq := ctl.DB.Model(&termModel.AcademicTermModel{}).
		Where("academic_term_school_id = ? AND academic_term_deleted_at IS NULL", schoolID)

	// ===== Filters (TERM) =====
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

	// ===== Fetch data TERM =====
	var list []termModel.AcademicTermModel
	if err := dbq.
		Order(orderExpr).
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Query failed: "+err.Error())
	}

	// ===== Pagination (pakai helper) =====
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	// Kalau tidak request include sama sekali → B/C lama
	if !includeClasses && !includeSections && !includeFeeRules {
		return helper.JsonList(c, "ok", dto.FromModels(list), pg)
	}

	// Kalau nggak ada data, tapi include diminta → return kosong dengan shape baru
	if len(list) == 0 {
		return helper.JsonList(c, "ok", []AcademicTermWithRelations{}, pg)
	}

	/* ===================== INCLUDE ===================== */

	// Kumpulkan semua term_id
	termIDs := make([]uuid.UUID, 0, len(list))
	for _, t := range list {
		termIDs = append(termIDs, t.AcademicTermID)
	}

	// --- INCLUDE: classes ---
	classesByTerm := make(map[uuid.UUID][]classModel.ClassModel)
	if includeClasses {
		dbClass := ctl.DB.Model(&classModel.ClassModel{}).
			Where("class_school_id = ? AND class_deleted_at IS NULL", schoolID).
			Where("class_academic_term_id IN ?", termIDs)

		// Tambahan filter untuk classes (opsional, bisa kamu sesuaikan):
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

		var cls []classModel.ClassModel
		if err := dbClass.Find(&cls).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Query classes failed: "+err.Error())
		}

		for _, citem := range cls {
			if citem.ClassAcademicTermID == nil {
				continue
			}
			tid := *citem.ClassAcademicTermID
			classesByTerm[tid] = append(classesByTerm[tid], citem)
		}
	}

	// --- INCLUDE: class_sections ---
	sectionsByTerm := make(map[uuid.UUID][]classSectionModel.ClassSectionModel)
	if includeSections {
		dbSec := ctl.DB.Model(&classSectionModel.ClassSectionModel{}).
			Where("class_section_school_id = ? AND class_section_deleted_at IS NULL", schoolID).
			Where("class_section_academic_term_id IN ?", termIDs)

		// Tambahan filter untuk class_sections:
		if v := strings.TrimSpace(c.Query("class_section_is_active")); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				dbSec = dbSec.Where("class_section_is_active = ?", b)
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

		var secs []classSectionModel.ClassSectionModel
		if err := dbSec.Find(&secs).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Query class_sections failed: "+err.Error())
		}

		for _, sitem := range secs {
			if sitem.ClassSectionAcademicTermID == nil {
				continue
			}
			tid := *sitem.ClassSectionAcademicTermID
			sectionsByTerm[tid] = append(sectionsByTerm[tid], sitem)
		}
	}

	// --- INCLUDE: fee_rules (SPP, dll) ---
	feeRulesByTerm := make(map[uuid.UUID][]feeRuleModel.FeeRule)
	if includeFeeRules {
		dbFee := ctl.DB.Model(&feeRuleModel.FeeRule{}).
			Where("fee_rule_school_id = ? AND fee_rule_deleted_at IS NULL", schoolID).
			Where("fee_rule_term_id IN ?", termIDs)

		// (opsional) tambah filter ringan:
		if v := strings.TrimSpace(c.Query("fee_rule_scope")); v != "" {
			dbFee = dbFee.Where("fee_rule_scope = ?", v)
		}
		if v := strings.TrimSpace(c.Query("fee_rule_option_code")); v != "" {
			dbFee = dbFee.Where("LOWER(fee_rule_option_code) = ?", strings.ToLower(v))
		}

		var fees []feeRuleModel.FeeRule
		if err := dbFee.Find(&fees).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Query fee_rules failed: "+err.Error())
		}

		for _, f := range fees {
			if f.FeeRuleTermID == nil {
				continue
			}
			tid := *f.FeeRuleTermID
			feeRulesByTerm[tid] = append(feeRulesByTerm[tid], f)
		}
	}

	// ===== Build response dengan include =====
	termDTOs := dto.FromModels(list)
	items := make([]AcademicTermWithRelations, len(list))
	for i, tDTO := range termDTOs {
		tid := list[i].AcademicTermID
		item := AcademicTermWithRelations{
			Term: tDTO,
		}
		if includeClasses {
			item.Classes = classesByTerm[tid]
		}
		if includeSections {
			item.ClassSections = sectionsByTerm[tid]
		}
		if includeFeeRules {
			item.FeeRules = feeRulesByTerm[tid]
		}
		items[i] = item
	}

	return helper.JsonList(c, "ok", items, pg)
}
