package controller

import (
	"fmt"
	"net/http"
	"schoolku_backend/internals/features/finance/billings/dto"
	model "schoolku_backend/internals/features/finance/billings/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /:school_id/spp/fee-rules
// GET /:school_id/spp/fee-rules
func (h *Handler) ListFeeRules(c *fiber.Ctx) error {
	schoolID, err := mustSchoolID(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid school_id")
	}
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return err
	}

	// === Paging (default 20, max 200) + dukungan per_page=all ===
	pg := helper.ResolvePaging(c, 20, 200)
	perPageRaw := strings.ToLower(strings.TrimSpace(c.Query("per_page")))
	allMode := perPageRaw == "all"
	offset := (pg.Page - 1) * pg.PerPage

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
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// === Fetch (respect per_page=all) ===
	listQ := q.Order(orderClause)
	if !allMode {
		listQ = listQ.Limit(pg.PerPage).Offset(offset)
	}
	var list []model.FeeRule
	if err := listQ.Find(&list).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	out := dto.ToFeeRuleResponses(list)

	// === Pagination payload untuk JsonList ===
	var pagination helper.Pagination
	if allMode {
		pagination = helper.BuildPaginationFromPage(total, 1, int(total))
	} else {
		pagination = helper.BuildPaginationFromPage(total, pg.Page, pg.PerPage)
	}

	return helper.JsonList(c, "OK", out, pagination)
}