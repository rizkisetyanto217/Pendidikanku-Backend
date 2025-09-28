package controller

import (
	"masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/dto"
	mModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"
	helper "masjidku_backend/internals/helpers"
	"strings"

	"github.com/gofiber/fiber/v2"
)

/* ============================== LIST (public) ============================== */

func (ctl *MasjidServicePlanController) List(c *fiber.Ctx) error {
	var q dto.ListMasjidServicePlanQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(&q); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	dbq := ctl.DB.WithContext(c.Context()).
		Model(&mModel.MasjidServicePlan{}).
		Where("masjid_service_plan_deleted_at IS NULL")

	if q.Code != nil && strings.TrimSpace(*q.Code) != "" {
		dbq = dbq.Where("LOWER(masjid_service_plan_code) = LOWER(?)", strings.TrimSpace(*q.Code))
	}
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		like := "%" + strings.ToLower(strings.TrimSpace(*q.Name)) + "%"
		dbq = dbq.Where("LOWER(masjid_service_plan_name) LIKE ?", like)
	}
	if q.Active != nil {
		dbq = dbq.Where("masjid_service_plan_is_active = ?", *q.Active)
	}
	if q.AllowCustomTheme != nil {
		dbq = dbq.Where("masjid_service_plan_allow_custom_theme = ?", *q.AllowCustomTheme)
	}
	if q.PriceMonthlyMin != nil {
		dbq = dbq.Where("(masjid_service_plan_price_monthly IS NOT NULL AND masjid_service_plan_price_monthly >= ?)", *q.PriceMonthlyMin)
	}
	if q.PriceMonthlyMax != nil {
		dbq = dbq.Where("(masjid_service_plan_price_monthly IS NOT NULL AND masjid_service_plan_price_monthly <= ?)", *q.PriceMonthlyMax)
	}

	sortVal := ""
	if q.Sort != nil {
		sortVal = strings.TrimSpace(strings.ToLower(*q.Sort))
	}
	switch sortVal {
	case "name_desc":
		dbq = dbq.Order("masjid_service_plan_name DESC")
	case "price_monthly_asc":
		dbq = dbq.Order("masjid_service_plan_price_monthly ASC NULLS LAST")
	case "price_monthly_desc":
		dbq = dbq.Order("masjid_service_plan_price_monthly DESC NULLS LAST")
	case "created_at_asc":
		dbq = dbq.Order("masjid_service_plan_created_at ASC")
	case "updated_at_desc":
		dbq = dbq.Order("masjid_service_plan_updated_at DESC")
	case "updated_at_asc":
		dbq = dbq.Order("masjid_service_plan_updated_at ASC")
	case "created_at_desc":
		fallthrough
	default:
		dbq = dbq.Order("masjid_service_plan_created_at DESC")
	}

	limit := clampLimit(q.Limit, 20, 200)
	offset := 0
	if q.Offset > 0 {
		offset = q.Offset
	}

	var rows []mModel.MasjidServicePlan
	if err := dbq.Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resp := make([]*dto.MasjidServicePlanResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, dto.NewMasjidServicePlanResponse(&rows[i]))
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"list":   resp,
		"limit":  limit,
		"offset": offset,
	})
}
