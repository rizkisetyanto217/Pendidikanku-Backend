// file: schedule_rules_user_controller.go (refactored)
package controller

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	d "madinahsalam_backend/internals/features/school/classes/class_schedules/dto"
	m "madinahsalam_backend/internals/features/school/classes/class_schedules/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lib/pq"
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
	Schedule d.ClassScheduleResponse       `json:"schedule"`
	Rules    []d.ClassScheduleRuleResponse `json:"rules,omitempty"`
}

/* =========================
   List schedules + optional rules
========================= */

func (ctl *ClassScheduleController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// parse query
	var q d.ListClassScheduleQuery
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

	tx := ctl.DB.Model(&m.ClassScheduleModel{})

	// alive filter
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_schedule_deleted_at IS NULL")
	}

	// tenant
	tx = tx.Where("class_schedule_school_id = ?", schoolID)

	// status filter
	if q.Status != nil {
		s := strings.ToLower(strings.TrimSpace(*q.Status))
		if s != "scheduled" && s != "ongoing" && s != "completed" && s != "canceled" {
			return helper.JsonError(c, http.StatusBadRequest, "status invalid")
		}
		tx = tx.Where("class_schedule_status = ?", s)
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
	var schedRows []m.ClassScheduleModel
	if err := tx.Order(orderExpr).Limit(limit).Offset(offset).Find(&schedRows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// pagination
	pg := helper.BuildPaginationFromOffset(total, offset, limit)

	// without rules → early return
	if !withRules {
		resp := make([]d.ClassScheduleResponse, 0, len(schedRows))
		for _, row := range schedRows {
			resp = append(resp, d.FromModel(row))
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
			Schedule: d.FromModel(sched),
			Rules:    rulesBySched[sched.ClassScheduleID],
		})
	}

	return helper.JsonListEx(c, "ok", combined, pg, []string{"rules"})
}

/*
	=========================
	  Fetch rules (fixed alias to avoid conflicts)
	=========================
*/

func fetchRulesGrouped(db *gorm.DB, schoolID uuid.UUID, scheduleIDs []uuid.UUID, withDeleted *bool) (map[uuid.UUID][]d.ClassScheduleRuleResponse, error) {
	out := make(map[uuid.UUID][]d.ClassScheduleRuleResponse)

	if len(scheduleIDs) == 0 {
		return out, nil
	}

	// Struct with SAFE field names
	type ruleFlat struct {
		ID                 uuid.UUID     `gorm:"column:class_schedule_rule_id"`
		SchoolID           uuid.UUID     `gorm:"column:class_schedule_rule_school_id"`
		ScheduleID         uuid.UUID     `gorm:"column:class_schedule_rule_schedule_id"`
		DayOfWeek          int           `gorm:"column:class_schedule_rule_day_of_week"`
		StartTimeStr       string        `gorm:"column:start_time_str"` // <── FIXED
		EndTimeStr         string        `gorm:"column:end_time_str"`   // <── FIXED
		IntervalWeeks      int           `gorm:"column:class_schedule_rule_interval_weeks"`
		StartOffsetWeeks   int           `gorm:"column:class_schedule_rule_start_offset_weeks"`
		WeekParity         string        `gorm:"column:class_schedule_rule_week_parity"`
		WeeksOfMonth       pq.Int64Array `gorm:"column:class_schedule_rule_weeks_of_month"`
		LastWeekOfMonth    bool          `gorm:"column:class_schedule_rule_last_week_of_month"`
		CSSTID             uuid.UUID     `gorm:"column:class_schedule_rule_csst_id"`
		CSSTSlugSnapshot   *string       `gorm:"column:class_schedule_rule_csst_slug_snapshot"`
		CSSTSnapshotRaw    []byte        `gorm:"column:class_schedule_rule_csst_snapshot"`
		CSSTTeacherID      *uuid.UUID    `gorm:"column:class_schedule_rule_csst_student_teacher_id"`
		CSSTSectionID      *uuid.UUID    `gorm:"column:class_schedule_rule_csst_class_section_id"`
		CSSTClassSubjectID *uuid.UUID    `gorm:"column:class_schedule_rule_csst_class_subject_id"`
		CSSTRoomID         *uuid.UUID    `gorm:"column:class_schedule_rule_csst_class_room_id"`
		CreatedAt          time.Time     `gorm:"column:class_schedule_rule_created_at"`
		UpdatedAt          time.Time     `gorm:"column:class_schedule_rule_updated_at"`
	}

	q := db.
		Table("class_schedule_rules").
		Select(`
			class_schedule_rule_id,
			class_schedule_rule_school_id,
			class_schedule_rule_schedule_id,
			class_schedule_rule_day_of_week,

			-- gunakan alias berbeda agar tidak bentrok dgn kolom asli
			to_char(class_schedule_rule_start_time::time, 'HH24:MI:SS') AS start_time_str,
			to_char(class_schedule_rule_end_time::time,   'HH24:MI:SS') AS end_time_str,

			class_schedule_rule_interval_weeks,
			class_schedule_rule_start_offset_weeks,
			class_schedule_rule_week_parity,
			class_schedule_rule_weeks_of_month,
			class_schedule_rule_last_week_of_month,
			class_schedule_rule_csst_id,
			class_schedule_rule_csst_slug_snapshot,
			class_schedule_rule_csst_snapshot,
			class_schedule_rule_csst_student_teacher_id,
			class_schedule_rule_csst_class_section_id,
			class_schedule_rule_csst_class_subject_id,
			class_schedule_rule_csst_class_room_id,
			class_schedule_rule_created_at,
			class_schedule_rule_updated_at
		`).
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

	var rows []ruleFlat
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, r := range rows {
		var snap map[string]any
		if len(r.CSSTSnapshotRaw) > 0 {
			_ = json.Unmarshal(r.CSSTSnapshotRaw, &snap)
		}

		resp := d.ClassScheduleRuleResponse{
			ClassScheduleRuleID:                   r.ID,
			ClassScheduleRuleSchoolID:             r.SchoolID,
			ClassScheduleRuleScheduleID:           r.ScheduleID,
			ClassScheduleRuleDayOfWeek:            r.DayOfWeek,
			ClassScheduleRuleStartTime:            r.StartTimeStr,
			ClassScheduleRuleEndTime:              r.EndTimeStr,
			ClassScheduleRuleIntervalWeeks:        r.IntervalWeeks,
			ClassScheduleRuleStartOffsetWeeks:     r.StartOffsetWeeks,
			ClassScheduleRuleWeekParity:           r.WeekParity,
			ClassScheduleRuleWeeksOfMonth:         []int64(r.WeeksOfMonth),
			ClassScheduleRuleLastWeekOfMonth:      r.LastWeekOfMonth,
			ClassScheduleRuleCSSTID:               r.CSSTID,
			ClassScheduleRuleCSSTSlugSnapshot:     r.CSSTSlugSnapshot,
			ClassScheduleRuleCSSTSnapshot:         snap,
			ClassScheduleRuleCSSTStudentTeacherID: r.CSSTTeacherID,
			ClassScheduleRuleCSSTClassSectionID:   r.CSSTSectionID,
			ClassScheduleRuleCSSTClassSubjectID:   r.CSSTClassSubjectID,
			ClassScheduleRuleCSSTClassRoomID:      r.CSSTRoomID,
			ClassScheduleRuleCreatedAt:            r.CreatedAt,
			ClassScheduleRuleUpdatedAt:            r.UpdatedAt,
		}

		out[r.ScheduleID] = append(out[r.ScheduleID], resp)
	}

	return out, nil
}
