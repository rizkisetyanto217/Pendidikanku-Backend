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

	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

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

// include flags yang didukung: rules, csst
type includeFlags struct {
	Rules bool
	CSST  bool
}

// nested flags yang didukung: rules, csst
type nestedFlags struct {
	Rules bool
	CSST  bool
}

// Format include:
//   - include=rules
//   - include=rule
//   - include=csst
//   - include=rules,csst
//   - legacy: include_rules=1, include_csst=1
func parseIncludeFlags(c *fiber.Ctx) includeFlags {
	flags := includeFlags{}

	// legacy flags
	if c.QueryBool("include_rules") {
		flags.Rules = true
	}
	if c.QueryBool("include_csst") {
		flags.CSST = true
	}

	raw := strings.TrimSpace(strings.ToLower(c.Query("include")))
	if raw == "" {
		return flags
	}

	parts := strings.Split(raw, ",")
	for _, part := range parts {
		token := strings.TrimSpace(part)
		switch token {
		case "rule", "rules":
			flags.Rules = true
		case "csst", "class_section_subject_teacher", "class_section_subject_teachers":
			flags.CSST = true
		}
	}

	return flags
}

// Format nested:
//   - nested=rules
//   - nested=rule
//   - nested=csst
//   - nested=rules,csst
//   - legacy: nested_rules=1, nested_csst=1
func parseNestedFlags(c *fiber.Ctx) nestedFlags {
	flags := nestedFlags{}

	// legacy flags
	if c.QueryBool("nested_rules") {
		flags.Rules = true
	}
	if c.QueryBool("nested_csst") {
		flags.CSST = true
	}

	raw := strings.TrimSpace(strings.ToLower(c.Query("nested")))
	if raw == "" {
		return flags
	}

	parts := strings.Split(raw, ",")
	for _, part := range parts {
		token := strings.TrimSpace(part)
		switch token {
		case "rule", "rules":
			flags.Rules = true
		case "csst", "class_section_subject_teacher", "class_section_subject_teachers":
			flags.CSST = true
		}
	}

	return flags
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
   Response types
========================= */

// Data utama per-row
type classScheduleWithNested struct {
	Schedule ruleDTO.ClassScheduleResponse              `json:"schedule"`
	Rules    []ruleDTO.ClassScheduleRuleResponse        `json:"rules,omitempty"` // hanya jika nested=rules
	CSST     *csstModel.ClassSectionSubjectTeacherModel `json:"csst,omitempty"`  // hanya jika nested=csst
}

// Payload "includes" di root response
type classScheduleIncludesPayload struct {
	// meta: jenis include / nested
	Include []string `json:"include,omitempty"`
	Nested  []string `json:"nested,omitempty"`

	// data hasil include=...
	Rules []ruleDTO.ClassScheduleRuleResponse         `json:"rules,omitempty"`
	CSST  []csstModel.ClassSectionSubjectTeacherModel `json:"csst,omitempty"`
}

/* =========================
   List schedules + optional rules / csst
========================= */

func (ctl *ClassScheduleController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// parse query
	var q ruleDTO.ListClassScheduleQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	// 🔓 PUBLIC school context
	schoolID, err := resolveSchoolID(c)
	if err != nil {
		return err
	}

	// include=rules,csst, ...
	includes := parseIncludeFlags(c)
	// nested=rules,csst, ...
	nested := parseNestedFlags(c)

	// Flag untuk kebutuhan fetch (include OR nested)
	needRules := includes.Rules || nested.Rules
	needCSST := includes.CSST || nested.CSST

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

	// ==========================
	// Tanpa include & nested apa pun → balikin schedule saja (compat lama)
	// ==========================
	if !needRules && !needCSST {
		resp := make([]ruleDTO.ClassScheduleResponse, 0, len(schedRows))
		for _, row := range schedRows {
			// 🔁 semua time → timezone sekolah via DTO
			resp = append(resp, ruleDTO.FromModel(row).WithSchoolTime(c))
		}
		emptyIncludes := classScheduleIncludesPayload{}
		return helper.JsonListEx(c, "ok", resp, pg, emptyIncludes)
	}

	// ==========================
	// WITH INCLUDE(S) / NESTED
	// ==========================

	// Kumpulkan schedule_id untuk prefetch
	sIDs := make([]uuid.UUID, len(schedRows))
	for i := range schedRows {
		sIDs[i] = schedRows[i].ClassScheduleID
	}

	// 1) RULES (group by schedule)
	var rulesBySched map[uuid.UUID][]ruleDTO.ClassScheduleRuleResponse
	if needRules {
		var err error
		rulesBySched, err = fetchRulesGrouped(ctl.DB, schoolID, sIDs, q.WithDeleted)
		if err != nil {
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}
	}

	// 2) CSST (via RULES → CSST_ID → CSST rows), hasil: map[schedule_id]CSST
	var csstBySched map[uuid.UUID]csstModel.ClassSectionSubjectTeacherModel
	if needCSST {
		var err error
		csstBySched, err = fetchCSSTBySchedule(ctl.DB, schoolID, sIDs, q.WithDeleted)
		if err != nil {
			return helper.JsonError(c, http.StatusInternalServerError, err.Error())
		}
	}

	// ==========================
	// BUILD DATA (nested)
	// ==========================
	combined := make([]classScheduleWithNested, 0, len(schedRows))
	for _, sched := range schedRows {
		item := classScheduleWithNested{
			// ⬇️ di-convert ke school time di sini
			Schedule: ruleDTO.FromModel(sched).WithSchoolTime(c),
		}

		if nested.Rules && rulesBySched != nil {
			item.Rules = rulesBySched[sched.ClassScheduleID]
		}

		if nested.CSST && csstBySched != nil {
			if csstRow, ok := csstBySched[sched.ClassScheduleID]; ok {
				rowCopy := csstRow
				item.CSST = &rowCopy
			}
		}

		combined = append(combined, item)
	}

	// ==========================
	// BUILD INCLUDES (global)
	// ==========================
	includeMeta := []string{}
	if includes.Rules {
		includeMeta = append(includeMeta, "rules")
	}
	if includes.CSST {
		includeMeta = append(includeMeta, "csst")
	}

	nestedMeta := []string{}
	if nested.Rules {
		nestedMeta = append(nestedMeta, "rules")
	}
	if nested.CSST {
		nestedMeta = append(nestedMeta, "csst")
	}

	// data yang benar-benar di-"include"
	var rulesInclude []ruleDTO.ClassScheduleRuleResponse
	if includes.Rules && rulesBySched != nil {
		for _, arr := range rulesBySched {
			if len(arr) == 0 {
				continue
			}
			rulesInclude = append(rulesInclude, arr...)
		}
	}

	var csstInclude []csstModel.ClassSectionSubjectTeacherModel
	if includes.CSST && csstBySched != nil {
		seen := make(map[uuid.UUID]struct{})
		for _, csstRow := range csstBySched {
			if _, ok := seen[csstRow.CSSTID]; ok {
				continue
			}
			seen[csstRow.CSSTID] = struct{}{}
			csstInclude = append(csstInclude, csstRow)
		}
	}

	includesPayload := classScheduleIncludesPayload{
		Include: includeMeta,
		Nested:  nestedMeta,
		Rules:   rulesInclude,
		CSST:    csstInclude,
	}

	return helper.JsonListEx(
		c,
		"ok",
		combined,
		pg,
		includesPayload,
	)
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

/*
	=========================
	  Fetch CSST per schedule (via rules)
	=========================
*/

// Map schedule_id -> CSST (ClassSectionSubjectTeacherModel)
// Diambil via class_schedule_rules (class_schedule_rule_csst_id)
func fetchCSSTBySchedule(
	db *gorm.DB,
	schoolID uuid.UUID,
	scheduleIDs []uuid.UUID,
	withDeleted *bool,
) (map[uuid.UUID]csstModel.ClassSectionSubjectTeacherModel, error) {

	out := make(map[uuid.UUID]csstModel.ClassSectionSubjectTeacherModel)

	if len(scheduleIDs) == 0 {
		return out, nil
	}

	// 1) Ambil pasangan schedule_id <-> csst_id dari rules
	type schedCSSTPair struct {
		ScheduleID uuid.UUID `gorm:"column:class_schedule_rule_schedule_id"`
		CSSTID     uuid.UUID `gorm:"column:class_schedule_rule_csst_id"`
	}

	var pairs []schedCSSTPair
	q := db.
		Model(&ruleModel.ClassScheduleRuleModel{}).
		Select("class_schedule_rule_schedule_id, class_schedule_rule_csst_id").
		Where("class_schedule_rule_school_id = ?", schoolID).
		Where("class_schedule_rule_schedule_id IN ?", scheduleIDs)

	if withDeleted == nil || !*withDeleted {
		q = q.Where("class_schedule_rule_deleted_at IS NULL")
	}

	if err := q.Find(&pairs).Error; err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return out, nil
	}

	// 2) schedule_id -> csst_id (ambil 1 saja per schedule)
	schedToCSSTID := make(map[uuid.UUID]uuid.UUID)
	csstIDSet := make(map[uuid.UUID]struct{})

	for _, p := range pairs {
		if p.CSSTID == uuid.Nil {
			continue
		}
		if _, exists := schedToCSSTID[p.ScheduleID]; !exists {
			schedToCSSTID[p.ScheduleID] = p.CSSTID
			csstIDSet[p.CSSTID] = struct{}{}
		}
	}

	if len(csstIDSet) == 0 {
		return out, nil
	}

	// 3) Fetch CSST rows (schema baru: csst_*)
	csstIDs := make([]uuid.UUID, 0, len(csstIDSet))
	for id := range csstIDSet {
		csstIDs = append(csstIDs, id)
	}

	var csstRows []csstModel.ClassSectionSubjectTeacherModel
	dbq := db.Model(&csstModel.ClassSectionSubjectTeacherModel{}).
		Where("csst_school_id = ?", schoolID).
		Where("csst_id IN ?", csstIDs)

	// biasanya CSST list harus yang alive
	if withDeleted == nil || !*withDeleted {
		dbq = dbq.Where("csst_deleted_at IS NULL")
	}

	if err := dbq.Find(&csstRows).Error; err != nil {
		return nil, err
	}

	csstByID := make(map[uuid.UUID]csstModel.ClassSectionSubjectTeacherModel, len(csstRows))
	for _, row := range csstRows {
		csstByID[row.CSSTID] = row
	}

	// 4) schedule_id -> CSST row
	for schedID, csstID := range schedToCSSTID {
		if csstRow, ok := csstByID[csstID]; ok {
			out[schedID] = csstRow
		}
	}

	return out, nil
}
