// file: schedule_rules_user_controller.go
package controller

import (
	"net/http"
	"strings"
	"time"

	ruleDTO "madinahsalam_backend/internals/features/school/class_others/class_schedules/dto"
	ruleModel "madinahsalam_backend/internals/features/school/class_others/class_schedules/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================
   Small helpers
========================= */

const (
	defLimit = 50
	maxLimit = 200
)

func clampLimitOffset(limitPtr, offsetPtr *int) (int, int) {
	limit := defLimit
	if limitPtr != nil {
		if *limitPtr < 1 {
			limit = defLimit
		} else if *limitPtr > maxLimit {
			limit = maxLimit
		} else {
			limit = *limitPtr
		}
	}
	offset := 0
	if offsetPtr != nil && *offsetPtr > 0 {
		offset = *offsetPtr
	}
	return limit, offset
}

func includeRulesFromQuery(c *fiber.Ctx) bool {
	if c.QueryBool("include_rules") {
		return true
	}
	inc := strings.TrimSpace(strings.ToLower(c.Query("include")))
	if inc == "" {
		return false
	}
	for _, part := range strings.Split(inc, ",") {
		if strings.TrimSpace(part) == "rules" {
			return true
		}
	}
	return false
}

/*
PUBLIC resolver: PRIORITAS token dulu.
1) Coba ambil school dari token (GetSchoolIDFromTokenPreferTeacher).
2) Kalau tidak ada / gagal → pakai ResolveSchoolContext (path/query: id/slug).
*/
func resolveSchoolID(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) Coba dari token dulu (kalau user login & token punya school context)
	if sid, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && sid != uuid.Nil {
		return sid, nil
	}

	// 2) Fallback ke resolver PUBLIC (params/query/slug)
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return uuid.Nil, fe
		}
		return uuid.Nil, fiber.NewError(http.StatusBadRequest, err.Error())
	}

	// 2a) Kalau ResolveSchoolContext sudah punya ID (biasanya dari :school_id / ?school_id)
	if mc.ID != uuid.Nil {
		return mc.ID, nil
	}

	// 2b) Kalau nggak ada ID tapi ada slug → resolve slug → ID
	if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil {
			return uuid.Nil, fiber.NewError(http.StatusNotFound, "School (slug) tidak ditemukan")
		}
		return id, nil
	}

	// 3) Bener-bener nggak ada context school
	return uuid.Nil, fiber.NewError(http.StatusBadRequest, helperAuth.ErrSchoolContextMissing.Error())
}

func buildScheduleOrder(sort *string) string {
	// default
	order := "class_schedule_created_at DESC"
	if sort == nil {
		return order
	}
	switch strings.ToLower(strings.TrimSpace(*sort)) {
	case "start_date_asc":
		order = "class_schedule_start_date ASC, class_schedule_end_date ASC, class_schedule_created_at DESC"
	case "start_date_desc":
		order = "class_schedule_start_date DESC, class_schedule_end_date DESC, class_schedule_created_at DESC"
	case "end_date_asc":
		order = "class_schedule_end_date ASC, class_schedule_start_date ASC, class_schedule_created_at DESC"
	case "end_date_desc":
		order = "class_schedule_end_date DESC, class_schedule_start_date DESC, class_schedule_created_at DESC"
	case "created_at_asc":
		order = "class_schedule_created_at ASC"
	case "created_at_desc":
		order = "class_schedule_created_at DESC"
	case "updated_at_asc":
		order = "class_schedule_updated_at ASC NULLS LAST"
	case "updated_at_desc":
		order = "class_schedule_updated_at DESC NULLS LAST"
	}
	return order
}

/* =========================
   Response type
========================= */

type classScheduleWithRules struct {
	Schedule ruleDTO.ClassScheduleResponse       `json:"schedule"`
	Rules    []ruleDTO.ClassScheduleRuleResponse `json:"rules,omitempty"`
}

/* =========================
   List schedules + optional rules
========================= */

func (ctl *ClassScheduleController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// parse query
	var q ruleDTO.ListClassScheduleQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// 🔓 PUBLIC school context:
	//    - Prioritas: dari token (GetSchoolIDFromTokenPreferTeacher)
	//    - Fallback: dari path/query/slug (ResolveSchoolContext)
	schoolID, err := resolveSchoolID(c)
	if err != nil {
		return err
	}

	withRules := includeRulesFromQuery(c)

	limit, offset := clampLimitOffset(q.Limit, q.Offset)
	orderExpr := buildScheduleOrder(q.Sort)

	tx := ctl.DB.Model(&ruleModel.ClassScheduleModel{})

	// alive filter
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_schedule_deleted_at IS NULL")
	}

	// tenant
	tx = tx.Where("class_schedule_school_id = ?", schoolID)

	// status filter
	if q.Status != nil {
		s := strings.ToLower(strings.TrimSpace(*q.Status))
		switch s {
		case "scheduled", "ongoing", "completed", "canceled":
			tx = tx.Where("class_schedule_status = ?", s)
		default:
			return helper.JsonError(c, http.StatusBadRequest, "status invalid")
		}
	}

	// active
	if q.IsActive != nil {
		tx = tx.Where("class_schedule_is_active = ?", *q.IsActive)
	}

	// date filters
	if q.DateFrom != nil && strings.TrimSpace(*q.DateFrom) != "" {
		dateFrom := strings.TrimSpace(*q.DateFrom)
		if _, err := time.Parse("2006-01-02", dateFrom); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "date_from invalid (YYYY-MM-DD)")
		}
		tx = tx.Where("class_schedule_end_date >= ?::date", dateFrom)
	}

	if q.DateTo != nil && strings.TrimSpace(*q.DateTo) != "" {
		dateTo := strings.TrimSpace(*q.DateTo)
		if _, err := time.Parse("2006-01-02", dateTo); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "date_to invalid (YYYY-MM-DD)")
		}
		tx = tx.Where("class_schedule_start_date <= ?::date", dateTo)
	}

	// q on slug
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		term := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("class_schedule_slug IS NOT NULL AND lower(class_schedule_slug) LIKE ?", term)
	}

	// count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// fetch schedules
	var schedRows []ruleModel.ClassScheduleModel
	if err := tx.Order(orderExpr).Limit(limit).Offset(offset).Find(&schedRows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// pagination
	pg := helper.BuildPaginationFromOffset(total, offset, limit)

	// without rules → early return
	if !withRules {
		resp := make([]ruleDTO.ClassScheduleResponse, 0, len(schedRows))
		for _, row := range schedRows {
			resp = append(resp, ruleDTO.FromModel(row))
		}
		return helper.JsonList(c, "ok", resp, pg)
	}

	// WITH RULES
	sIDs := make([]uuid.UUID, len(schedRows))
	for i := range schedRows {
		sIDs[i] = schedRows[i].ClassScheduleID
	}

	rulesBySched, err := fetchRulesGrouped(ctl.DB, schoolID, sIDs, q.WithDeleted)
	if err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	combined := make([]classScheduleWithRules, 0, len(schedRows))
	for _, sched := range schedRows {
		combined = append(combined, classScheduleWithRules{
			Schedule: ruleDTO.FromModel(sched),
			Rules:    rulesBySched[sched.ClassScheduleID],
		})
	}

	return helper.JsonListEx(c, "ok", combined, pg, []string{"rules"})
}

/*
	=========================
	  Fetch rules (pakai model + DTO mapper)
	=========================
*/

func fetchRulesGrouped(db *gorm.DB, schoolID uuid.UUID, scheduleIDs []uuid.UUID, withDeleted *bool) (map[uuid.UUID][]ruleDTO.ClassScheduleRuleResponse, error) {
	out := make(map[uuid.UUID][]ruleDTO.ClassScheduleRuleResponse)

	if len(scheduleIDs) == 0 {
		return out, nil
	}

	var rules []ruleModel.ClassScheduleRuleModel

	q := db.
		Model(&ruleModel.ClassScheduleRuleModel{}).
		Where("class_schedule_rule_school_id = ?", schoolID).
		Where("class_schedule_rule_schedule_id IN ?", scheduleIDs)

	if withDeleted == nil || !*withDeleted {
		q = q.Where("class_schedule_rule_deleted_at IS NULL")
	}

	q = q.Order(`
		class_schedule_rule_day_of_week ASC,
		class_schedule_rule_start_time ASC,
		class_schedule_rule_end_time ASC,
		class_schedule_rule_created_at ASC
	`)

	if err := q.Find(&rules).Error; err != nil {
		return nil, err
	}

	for _, r := range rules {
		schedID := r.ClassScheduleRuleScheduleID
		out[schedID] = append(out[schedID], ruleDTO.FromRuleModel(r))
	}

	return out, nil
}
