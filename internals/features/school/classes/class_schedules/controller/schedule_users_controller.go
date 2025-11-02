package controller

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	d "schoolku_backend/internals/features/school/classes/class_schedules/dto"
	m "schoolku_backend/internals/features/school/classes/class_schedules/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

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
		switch {
		case *limitPtr <= 0:
			limit = defLimit
		case *limitPtr > maxLimit:
			limit = maxLimit
		default:
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
	inc := strings.ToLower(strings.TrimSpace(c.Query("include")))
	if inc == "" {
		return false
	}
	for _, p := range strings.Split(inc, ",") {
		if strings.TrimSpace(p) == "rules" {
			return true
		}
	}
	return false
}

func resolveSchoolID(c *fiber.Ctx) (uuid.UUID, error) {
	// Prefer explicit school context (DKM/Admin required).
	if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		return helperAuth.EnsureSchoolAccessDKM(c, mc)
	}
	// Fallback to token (teacher-aware).
	if act, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && act != uuid.Nil {
		return act, nil
	}
	return uuid.Nil, fiber.NewError(http.StatusForbidden, "Scope school tidak ditemukan")
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
   Response type (with rules)
========================= */

type classScheduleWithRules struct {
	Schedule d.ClassScheduleResponse       `json:"schedule"`
	Rules    []d.ClassScheduleRuleResponse `json:"rules,omitempty"`
}

/* =========================
   List (filters/sort/pagination + optional rules)
========================= */

func (ctl *ClassScheduleController) List(c *fiber.Ctx) error {
	// buat helper lain bisa akses DB bila perlu (slug→ID, dll.)
	c.Locals("DB", ctl.DB)

	var q d.ListClassScheduleQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	schoolID, err := resolveSchoolID(c)
	if err != nil {
		return err
	}

	withRules := includeRulesFromQuery(c)
	limit, offset := clampLimitOffset(q.Limit, q.Offset)
	orderExpr := buildScheduleOrder(q.Sort)

	// ===== Base query: schedules =====
	tx := ctl.DB.Model(&m.ClassScheduleModel{})

	// alive only by default
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_schedule_deleted_at IS NULL")
	}

	// tenant
	tx = tx.Where("class_schedule_school_id = ?", schoolID)

	// status
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

	// date range overlap filter
	if q.DateFrom != nil && strings.TrimSpace(*q.DateFrom) != "" {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(*q.DateFrom)); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "date_from invalid (YYYY-MM-DD)")
		}
		tx = tx.Where("class_schedule_end_date >= ?::date", strings.TrimSpace(*q.DateFrom))
	}
	if q.DateTo != nil && strings.TrimSpace(*q.DateTo) != "" {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(*q.DateTo)); err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "date_to invalid (YYYY-MM-DD)")
		}
		tx = tx.Where("class_schedule_start_date <= ?::date", strings.TrimSpace(*q.DateTo))
	}

	// q on slug
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		term := strings.ToLower(strings.TrimSpace(*q.Q))
		tx = tx.Where("class_schedule_slug IS NOT NULL AND lower(class_schedule_slug) LIKE ?", "%"+term+"%")
	}

	// ===== Count first =====
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// ===== Fetch schedules =====
	var schedRows []m.ClassScheduleModel
	if err := tx.
		Order(orderExpr).
		Limit(limit).
		Offset(offset).
		Find(&schedRows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// ===== No rules? return classic response =====
	if !withRules {
		out := make([]d.ClassScheduleResponse, 0, len(schedRows))
		for i := range schedRows {
			out = append(out, d.FromModel(schedRows[i]))
		}
		meta := fiber.Map{"limit": limit, "offset": offset, "total": total}
		return helper.JsonList(c, out, meta)
	}

	// ===== WITH RULES =====
	// kumpulkan schedule_ids
	sIDs := make([]uuid.UUID, 0, len(schedRows))
	for i := range schedRows {
		sIDs = append(sIDs, schedRows[i].ClassScheduleID)
	}

	rulesBySched, err := fetchRulesGrouped(ctl.DB, schoolID, sIDs, q.WithDeleted)
	if err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, err.Error())
	}

	// gabungkan
	out := make([]classScheduleWithRules, 0, len(schedRows))
	for i := range schedRows {
		out = append(out, classScheduleWithRules{
			Schedule: d.FromModel(schedRows[i]),
			Rules:    rulesBySched[schedRows[i].ClassScheduleID],
		})
	}

	meta := fiber.Map{
		"limit":   limit,
		"offset":  offset,
		"total":   total,
		"include": []string{"rules"},
	}
	return helper.JsonList(c, out, meta)
}

/*
	=========================
	  Rules fetcher (safe for TEXT/TIME)

=========================
*/
func fetchRulesGrouped(db *gorm.DB, schoolID uuid.UUID, scheduleIDs []uuid.UUID, withDeleted *bool) (map[uuid.UUID][]d.ClassScheduleRuleResponse, error) {
	out := make(map[uuid.UUID][]d.ClassScheduleRuleResponse, len(scheduleIDs))
	if len(scheduleIDs) == 0 {
		return out, nil
	}

	// Struct flat: waktu sebagai string (HH:MM:SS), weeks array sebagai pq.Int64Array
	type ruleFlat struct {
		ID                 uuid.UUID     `gorm:"column:class_schedule_rule_id"`
		SchoolID           uuid.UUID     `gorm:"column:class_schedule_rule_school_id"`
		ScheduleID         uuid.UUID     `gorm:"column:class_schedule_rule_schedule_id"`
		DayOfWeek          int           `gorm:"column:class_schedule_rule_day_of_week"`
		StartTimeStr       string        `gorm:"column:class_schedule_rule_start_time"` // HH:MM:SS
		EndTimeStr         string        `gorm:"column:class_schedule_rule_end_time"`   // HH:MM:SS
		IntervalWeeks      int           `gorm:"column:class_schedule_rule_interval_weeks"`
		StartOffsetWeeks   int           `gorm:"column:class_schedule_rule_start_offset_weeks"`
		WeekParity         string        `gorm:"column:class_schedule_rule_week_parity"`
		WeeksOfMonth       pq.Int64Array `gorm:"column:class_schedule_rule_weeks_of_month"` // int[]
		LastWeekOfMonth    bool          `gorm:"column:class_schedule_rule_last_week_of_month"`
		CSSTID             uuid.UUID     `gorm:"column:class_schedule_rule_csst_id"`
		CSSTSchoolID       uuid.UUID     `gorm:"column:class_schedule_rule_csst_school_id"`
		CSSTSnapshotRaw    []byte        `gorm:"column:class_schedule_rule_csst_snapshot"` // jsonb
		CSSTTeacherID      *uuid.UUID    `gorm:"column:class_schedule_rule_csst_teacher_id"`
		CSSTSectionID      *uuid.UUID    `gorm:"column:class_schedule_rule_csst_section_id"`
		CSSTClassSubjectID *uuid.UUID    `gorm:"column:class_schedule_rule_csst_class_subject_id"`
		CSSTRoomID         *uuid.UUID    `gorm:"column:class_schedule_rule_csst_room_id"`
		CreatedAt          time.Time     `gorm:"column:class_schedule_rule_created_at"`
		UpdatedAt          time.Time     `gorm:"column:class_schedule_rule_updated_at"`
		DeletedAt          *time.Time    `gorm:"column:class_schedule_rule_deleted_at"`
	}

	q := db.
		Table("class_schedule_rules").
		// waktu dipaksa ke string HH24:MI:SS agar aman di-scan
		Select(`
			class_schedule_rule_id,
			class_schedule_rule_school_id,
			class_schedule_rule_schedule_id,
			class_schedule_rule_day_of_week,
			to_char(class_schedule_rule_start_time::time, 'HH24:MI:SS') AS class_schedule_rule_start_time,
			to_char(class_schedule_rule_end_time::time,   'HH24:MI:SS') AS class_schedule_rule_end_time,
			class_schedule_rule_interval_weeks,
			class_schedule_rule_start_offset_weeks,
			class_schedule_rule_week_parity,
			class_schedule_rule_weeks_of_month,
			class_schedule_rule_last_week_of_month,
			class_schedule_rule_csst_id,
			class_schedule_rule_csst_school_id,
			class_schedule_rule_csst_snapshot,
			class_schedule_rule_csst_teacher_id,
			class_schedule_rule_csst_section_id,
			class_schedule_rule_csst_class_subject_id,
			class_schedule_rule_csst_room_id,
			class_schedule_rule_created_at,
			class_schedule_rule_updated_at,
			class_schedule_rule_deleted_at
		`).
		Where("class_schedule_rule_school_id = ?", schoolID).
		Where("class_schedule_rule_schedule_id IN ?", scheduleIDs)

	if withDeleted == nil || !*withDeleted {
		q = q.Where("class_schedule_rule_deleted_at IS NULL")
	}

	q = q.Order(`
		class_schedule_rule_day_of_week ASC,
		class_schedule_rule_start_time::time ASC,
		class_schedule_rule_end_time::time   ASC,
		class_schedule_rule_created_at ASC
	`)

	var rows []ruleFlat
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}

	for i := range rows {
		r := rows[i]

		// decode snapshot jsonb -> map[string]any (optional)
		var snap map[string]any
		if len(r.CSSTSnapshotRaw) > 0 {
			_ = json.Unmarshal(r.CSSTSnapshotRaw, &snap)
		}

		resp := d.ClassScheduleRuleResponse{
			ClassScheduleRuleID:                 r.ID,
			ClassScheduleRuleSchoolID:           r.SchoolID,
			ClassScheduleRuleScheduleID:         r.ScheduleID,
			ClassScheduleRuleDayOfWeek:          r.DayOfWeek,
			ClassScheduleRuleStartTime:          r.StartTimeStr, // sudah "HH:MM:SS"
			ClassScheduleRuleEndTime:            r.EndTimeStr,   // sudah "HH:MM:SS"
			ClassScheduleRuleIntervalWeeks:      r.IntervalWeeks,
			ClassScheduleRuleStartOffsetWeeks:   r.StartOffsetWeeks,
			ClassScheduleRuleWeekParity:         r.WeekParity,
			ClassScheduleRuleWeeksOfMonth:       []int64(r.WeeksOfMonth),
			ClassScheduleRuleLastWeekOfMonth:    r.LastWeekOfMonth,
			ClassScheduleRuleCSSTID:             r.CSSTID,
			ClassScheduleRuleCSSTSchoolID:       r.CSSTSchoolID,
			ClassScheduleRuleCSSTSnapshot:       snap,
			ClassScheduleRuleCSSTTeacherID:      r.CSSTTeacherID,
			ClassScheduleRuleCSSTSectionID:      r.CSSTSectionID,
			ClassScheduleRuleCSSTClassSubjectID: r.CSSTClassSubjectID,
			ClassScheduleRuleCSSTRoomID:         r.CSSTRoomID,
			ClassScheduleRuleCreatedAt:          r.CreatedAt,
			ClassScheduleRuleUpdatedAt:          r.UpdatedAt,
		}

		out[r.ScheduleID] = append(out[r.ScheduleID], resp)
	}

	return out, nil
}
