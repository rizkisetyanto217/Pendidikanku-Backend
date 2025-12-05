// file: internals/features/finance/billings/controller/fee_rule_list_controller.go
package controller

import (
	"fmt"
	"strings"
	"time"

	"madinahsalam_backend/internals/features/finance/billings/dto"
	model "madinahsalam_backend/internals/features/finance/billings/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// lite-term untuk include/nested opsional
type termLite struct {
	ID           uuid.UUID `json:"academic_term_id"                 gorm:"column:academic_term_id"`
	Name         string    `json:"academic_term_name"               gorm:"column:academic_term_name"`
	AcademicYear string    `json:"academic_term_academic_year"      gorm:"column:academic_term_academic_year"`
	StartDate    time.Time `json:"academic_term_start_date"         gorm:"column:academic_term_start_date"`
	EndDate      time.Time `json:"academic_term_end_date"           gorm:"column:academic_term_end_date"`
	IsActive     bool      `json:"academic_term_is_active"          gorm:"column:academic_term_is_active"`
	Angkatan     *int      `json:"academic_term_angkatan,omitempty" gorm:"column:academic_term_angkatan"`
}

// GET /api/u/fee-rules/list
// GET /:school_id/spp/fee-rules (kompat lama)
func (h *Handler) ListFeeRules(c *fiber.Ctx) error {
	// === School context: UTAMA dari token, baru fallback ke path ===
	var schoolID uuid.UUID

	// 1) Coba ambil dari token (school_ids di claim)
	if ids, errTok := helperAuth.GetSchoolIDsFromToken(c); errTok == nil && len(ids) > 0 {
		// asumsi: untuk staff/murid biasanya 1 active school,
		// kalau nanti mau lebih strict bisa diganti pakai active_school_id
		schoolID = ids[0]
	} else {
		// 2) Fallback: path param :school_id (kompatibel versi lama)
		raw := strings.TrimSpace(c.Params("school_id"))
		if raw == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_id tidak ditemukan di token maupun path")
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "invalid school_id")
		}
		schoolID = id
	}

	// Member sekolah (student/teacher/dkm/admin/bendahara) boleh lihat fee-rules
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return err
	}

	// === Paging (default 20, max 200) + dukungan per_page=all ===
	pg := helper.ResolvePaging(c, 20, 200)
	perPageRaw := strings.ToLower(strings.TrimSpace(c.Query("per_page")))
	allMode := perPageRaw == "all"
	offset := (pg.Page - 1) * pg.PerPage

	// === Include & Nested flags (opsional) ===
	includeRaw := strings.TrimSpace(c.Query("include"))
	nestedRaw := strings.TrimSpace(c.Query("nested"))

	includeTerm := false // academic_terms di top-level "include"
	nestedTerm := false  // academic_terms nested di tiap item "data"

	if includeRaw != "" {
		for _, part := range strings.Split(includeRaw, ",") {
			switch strings.ToLower(strings.TrimSpace(part)) {
			case "term", "academic_term", "academic_terms":
				includeTerm = true
			}
		}
	}
	if nestedRaw != "" {
		for _, part := range strings.Split(nestedRaw, ",") {
			switch strings.ToLower(strings.TrimSpace(part)) {
			case "term", "academic_term", "academic_terms":
				nestedTerm = true
			}
		}
	}

	// === Base query (tenant-scoped & alive) ===
	q := h.DB.Model(&model.FeeRule{}).
		Where("fee_rule_deleted_at IS NULL").
		Where("fee_rule_school_id = ?", schoolID)

	// === Filters ===
	if oc := strings.TrimSpace(c.Query("option_code")); oc != "" {
		q = q.Where("LOWER(fee_rule_option_code) = ?", strings.ToLower(oc))
	}
	if sc := strings.TrimSpace(c.Query("scope")); sc != "" {
		q = q.Where("fee_rule_scope = ?", sc)
	}
	if tid := strings.TrimSpace(c.Query("term_id")); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			q = q.Where("fee_rule_term_id = ?", id)
		}
	} else if ym := strings.TrimSpace(c.Query("ym")); ym != "" {
		var y, m int
		if _, err := fmt.Sscanf(ym, "%d-%d", &y, &m); err == nil && y > 0 && m >= 1 && m <= 12 {
			q = q.Where("fee_rule_year = ? AND fee_rule_month = ?", y, m)
		}
	}

	// === Sorting whitelist ===
	allowed := map[string]string{
		"created_at": "fee_rule_created_at",
		"updated_at": "fee_rule_updated_at",
		"amount":     "fee_rule_amount_idr",
		"option":     "fee_rule_option_code",
	}
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by")))
	sortCol, ok := allowed[sortBy]
	if !ok {
		sortCol = allowed["created_at"]
	}
	dir := "DESC"
	if strings.EqualFold(strings.TrimSpace(c.Query("order")), "asc") {
		dir = "ASC"
	}
	orderClause := sortCol + " " + dir

	// === Count ===
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// === Fetch (respect per_page=all) ===
	listQ := q.Order(orderClause)
	if !allMode {
		listQ = listQ.Limit(pg.PerPage).Offset(offset)
	}
	var list []model.FeeRule
	if err := listQ.Find(&list).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ============================
	// Prefetch academic_terms (dipakai untuk include / nested)
	// ============================
	termMap := map[uuid.UUID]termLite{}

	if (includeTerm || nestedTerm) && len(list) > 0 {
		termSet := map[uuid.UUID]struct{}{}
		for _, fr := range list {
			// asumsi field di model: FeeRuleTermID *uuid.UUID
			if fr.FeeRuleTermID != nil {
				termSet[*fr.FeeRuleTermID] = struct{}{}
			}
		}

		if len(termSet) > 0 {
			ids := make([]uuid.UUID, 0, len(termSet))
			for id := range termSet {
				ids = append(ids, id)
			}

			var trs []termLite
			if err := h.DB.
				Table("academic_terms").
				Select(`
					academic_term_id,
					academic_term_name,
					academic_term_academic_year,
					academic_term_start_date,
					academic_term_end_date,
					academic_term_is_active,
					academic_term_angkatan
				`).
				Where("academic_term_id IN ? AND academic_term_deleted_at IS NULL", ids).
				Scan(&trs).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil academic_terms: "+err.Error())
			}
			for _, t := range trs {
				termMap[t.ID] = t
			}
		}
	}

	// ============================
	// Compose DATA (plain / nested)
	// ============================
	base := dto.ToFeeRuleResponses(list)

	// asumsikan ada type dto.FeeRuleResponse
	type FeeRuleWithTerm struct {
		dto.FeeRuleResponse `json:",inline"`
		AcademicTerm        *termLite `json:"academic_terms,omitempty"`
	}

	var data any
	if nestedTerm {
		// mode: NESTED → academic_terms nempel di tiap item data[]
		out := make([]FeeRuleWithTerm, 0, len(base))
		for i, fr := range list {
			item := FeeRuleWithTerm{
				FeeRuleResponse: base[i],
			}
			if fr.FeeRuleTermID != nil {
				if t, ok := termMap[*fr.FeeRuleTermID]; ok {
					tCopy := t
					item.AcademicTerm = &tCopy
				}
			}
			out = append(out, item)
		}
		data = out
	} else {
		// default: tanpa academic_terms nested
		data = base
	}

	// ============================
	// Compose INCLUDE (top-level)
	// ============================
	type FeeRuleInclude struct {
		AcademicTerms []termLite `json:"academic_terms,omitempty"`
		// next:
		// Classes       []ClassLite          `json:"classes,omitempty"`
		// ClassSections []ClassSectionLite   `json:"class_sections,omitempty"`
		// FeeRules      []dto.FeeRuleResponse `json:"fee_rules,omitempty"`
	}

	var include any
	if includeTerm && len(termMap) > 0 {
		terms := make([]termLite, 0, len(termMap))
		for _, t := range termMap {
			terms = append(terms, t)
		}
		include = FeeRuleInclude{
			AcademicTerms: terms,
		}
	}

	// === Pagination payload ===
	var pagination helper.Pagination
	if allMode {
		pagination = helper.BuildPaginationFromPage(total, 1, int(total))
	} else {
		pagination = helper.BuildPaginationFromPage(total, pg.Page, pg.PerPage)
	}

	// ============================
	// Final JSON response (pakai helper)
	// ============================
	return helper.JsonListWithInclude(c, "OK", data, include, pagination)
}
