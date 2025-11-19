// file: internals/features/school/classes/class_attendance_sessions/controller/list_controller.go
package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	sessiondto "schoolku_backend/internals/features/school/classes/class_attendance_sessions/dto"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* =========================
   Utils (tanpa perubahan besar)
========================= */

func parseYmd(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, err
	}
	tt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return &tt, nil
}

func parseUUIDPtr(s string, field string) (*uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, field+" tidak valid")
	}
	return &id, nil
}

func parseUUIDList(raw string) ([]uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := uuid.Parse(part)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "id tidak valid")
		}
		out = append(out, id)
	}
	return out, nil
}

func jsonToMap(j datatypes.JSON) map[string]any {
	if len(j) == 0 {
		return nil
	}
	var m map[string]any
	_ = json.Unmarshal(j, &m)
	return m
}

// ===============================
// Handlers
// ===============================

// parse "HH:MM[:SS]" jadi *time.Time (tanggal dummy 2000-01-01)
func parseRuleTimeToPtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	str := strings.TrimSpace(*s)
	if str == "" {
		return nil
	}

	layouts := []string{"15:04:05", "15:04"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, str); err == nil {
			// pakai tanggal dummy supaya hanya time-of-day yang dipakai
			tt := time.Date(2000, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, time.Local)
			return &tt
		}
	}
	// kalau gagal parse, kita fallback nil aja (nggak hard error)
	return nil
}

/* =================================================================
   LIST /admin/class-attendance-sessions — updated to DTO terbaru
   + support mode "nearest" (3 hari ke depan, urut paling dekat)
================================================================= */

func (ctrl *ClassAttendanceSessionController) ListClassAttendanceSessions(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// ===== School context: ambil dari token dulu =====
	var schoolID uuid.UUID

	// Prioritas: token teacher / active-school style
	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else if id, err := helperAuth.GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		// fallback terakhir (kalau memang masih pakai path/slug di beberapa route)
		if mc, err := helperAuth.ResolveSchoolContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
			if mc.ID != uuid.Nil {
				schoolID = mc.ID
			} else if strings.TrimSpace(mc.Slug) != "" {
				if sid, e2 := helperAuth.GetSchoolIDBySlug(c, mc.Slug); e2 == nil {
					schoolID = sid
				}
			}
		}
	}

	if schoolID == uuid.Nil {
		return helper.JsonError(c, http.StatusForbidden, "Scope school tidak ditemukan")
	}

	// ===== Guard: hanya DKM/Admin/Teacher di school ini =====
	if err := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); err != nil {
		return err
	}

	// ===== Roles (dipakai untuk scope participants) =====
	userID, _ := helperAuth.GetUserIDFromToken(c)
	adminSchoolID, _ := helperAuth.GetSchoolIDFromToken(c)
	teacherSchoolID, _ := helperAuth.GetSchoolIDFromTokenPreferTeacher(c)

	isAdmin := (adminSchoolID != uuid.Nil && adminSchoolID == schoolID) ||
		helperAuth.HasRoleInSchool(c, schoolID, "admin") ||
		helperAuth.HasRoleInSchool(c, schoolID, "dkm") ||
		helperAuth.IsDKMInSchool(c, schoolID)

	isTeacher := (teacherSchoolID != uuid.Nil && teacherSchoolID == schoolID) ||
		helperAuth.HasRoleInSchool(c, schoolID, "teacher") ||
		helperAuth.IsTeacherInSchool(c, schoolID)

	// ===== Includes =====
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeAll := includeStr == "all"
	includeSet := map[string]bool{}
	for _, part := range strings.Split(includeStr, ",") {
		if p := strings.TrimSpace(part); p != "" {
			includeSet[p] = true
		}
	}
	wantParticipants :=
		includeAll ||
			includeSet["participants"] ||
			includeSet["participant"] ||
			includeSet["session_participants"] ||
			includeSet["session_participant"]

	// ===== Pagination =====
	p := helper.ResolvePaging(c, 20, 200)

	// ===== Sorting whitelist =====
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by")))
	order := strings.ToLower(strings.TrimSpace(c.Query("sort")))
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	col := "cas.class_attendance_session_date" // default
	switch sortBy {
	case "title":
		col = "cas.class_attendance_session_title"
	case "date", "":
		col = "cas.class_attendance_session_date"
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// ===== MODE NEAREST (3 hari ke depan, urut paling dekat sekarang) =====
	nearestRaw := strings.ToLower(strings.TrimSpace(c.Query("nearest")))
	isNearest := nearestRaw == "1" || nearestRaw == "true" || nearestRaw == "yes"

	// ===== Filters dasar =====
	var df, dt *time.Time
	var err error

	if !isNearest {
		// mode biasa → pakai date_from/date_to
		df, err = parseYmd(c.Query("date_from"))
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
		}
		dt, err = parseYmd(c.Query("date_to"))
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
		}
	}

	teacherIdPtr, err := parseUUIDPtr(c.Query("teacher_id"), "teacher_id")
	if err != nil {
		return err
	}
	teacherUserIDPtr, err := parseUUIDPtr(c.Query("teacher_user_id"), "teacher_user_id")
	if err != nil {
		return err
	}

	if strings.TrimSpace(c.Query("section_id")) != "" || strings.TrimSpace(c.Query("class_subject_id")) != "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Filter section_id / class_subject_id belum didukung skema jadwal saat ini")
	}

	// schedule (opsional) & dukung null
	scheduleRaw := strings.TrimSpace(c.Query("schedule_id"))
	scheduleIDPtr, err := parseUUIDPtr(scheduleRaw, "schedule_id")
	if err != nil {
		return err
	}
	wantScheduleNull := strings.EqualFold(scheduleRaw, "null") || strings.EqualFold(scheduleRaw, "nil")

	// keyword
	var like *string
	if kw := strings.TrimSpace(c.Query("q")); kw != "" {
		pat := "%" + kw + "%"
		like = &pat
	}

	// filter by id(s)
	var sessionIDs []uuid.UUID
	if ids, err := parseUUIDList(c.Query("id")); err != nil {
		return err
	} else if len(ids) > 0 {
		sessionIDs = ids
	} else if ids, err := parseUUIDList(c.Query("session_id")); err != nil {
		return err
	} else if len(ids) > 0 {
		sessionIDs = ids
	} else if ids, err := parseUUIDList(c.Query("cas_id")); err != nil {
		return err
	} else if len(ids) > 0 {
		sessionIDs = ids
	}

	// ===== Base query =====
	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Where("cas.class_attendance_session_school_id = ?", schoolID).
		Where("cas.class_attendance_session_deleted_at IS NULL")

	// ===== Filter waktu =====
	if isNearest {
		// Mode terdekat: gunakan starts_at sebagai acuan,
		// ambil sesi dengan starts_at antara sekarang dan 3 hari ke depan.
		now := time.Now()
		threeDaysLater := now.AddDate(0, 0, 3)

		qBase = qBase.Where(`
			cas.class_attendance_session_starts_at IS NOT NULL
			AND cas.class_attendance_session_starts_at >= ?
			AND cas.class_attendance_session_starts_at <= ?
		`, now, threeDaysLater)

		// Override sorting: paling dekat dengan jam sekarang di atas
		orderExpr = "cas.class_attendance_session_starts_at ASC, cas.class_attendance_session_date ASC, cas.class_attendance_session_id ASC"
	} else {
		// Mode biasa: pakai date_from/date_to
		if df != nil && dt != nil {
			qBase = qBase.Where("cas.class_attendance_session_date BETWEEN ? AND ?", *df, *dt)
		} else if df != nil {
			qBase = qBase.Where("cas.class_attendance_session_date >= ?", *df)
		} else if dt != nil {
			qBase = qBase.Where("cas.class_attendance_session_date <= ?", *dt)
		}
	}

	if scheduleIDPtr != nil {
		qBase = qBase.Where("cas.class_attendance_session_schedule_id = ?", *scheduleIDPtr)
	} else if wantScheduleNull {
		qBase = qBase.Where(`cas.class_attendance_session_schedule_id IS NULL OR cas.class_attendance_session_schedule_id = '00000000-0000-0000-0000-000000000000'`)
	}

	if len(sessionIDs) > 0 {
		qBase = qBase.Where("cas.class_attendance_session_id IN ?", sessionIDs)
	}
	if teacherIdPtr != nil {
		qBase = qBase.Where("cas.class_attendance_session_teacher_id = ?", *teacherIdPtr)
	}
	if teacherUserIDPtr != nil {
		qBase = qBase.
			Joins(`LEFT JOIN school_teachers mt_cas
                     ON mt_cas.school_teacher_id = cas.class_attendance_session_teacher_id
                    AND mt_cas.school_teacher_deleted_at IS NULL
                    AND mt_cas.school_teacher_school_id = cas.class_attendance_session_school_id`).
			Where(`mt_cas.school_teacher_user_id = ?`, *teacherUserIDPtr)
	}
	if like != nil {
		qBase = qBase.Where(`
			(cas.class_attendance_session_title ILIKE ?
			 OR cas.class_attendance_session_general_info ILIKE ?
             OR cas.class_attendance_session_display_title ILIKE ?)`, *like, *like, *like)
	}

	// ===== Total =====
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("cas.class_attendance_session_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Page data (UPDATED to DTO, lengkap) =====
	type row struct {
		// kunci utama
		ID       uuid.UUID `gorm:"column:class_attendance_session_id"`
		SchoolID uuid.UUID `gorm:"column:class_attendance_session_school_id"`

		// relasi jadwal & rule
		ScheduleID *uuid.UUID `gorm:"column:class_attendance_session_schedule_id"`
		RuleID     *uuid.UUID `gorm:"column:class_attendance_session_rule_id"`

		// slug
		Slug *string `gorm:"column:class_attendance_session_slug"`

		// waktu
		Date     time.Time  `gorm:"column:class_attendance_session_date"`
		StartsAt *time.Time `gorm:"column:class_attendance_session_starts_at"`
		EndsAt   *time.Time `gorm:"column:class_attendance_session_ends_at"`

		// status session
		Status           string `gorm:"column:class_attendance_session_status"`
		AttendanceStatus string `gorm:"column:class_attendance_session_attendance_status"`
		Locked           bool   `gorm:"column:class_attendance_session_locked"`
		IsOverride       bool   `gorm:"column:class_attendance_session_is_override"`
		IsCanceled       bool   `gorm:"column:class_attendance_session_is_canceled"`

		OriginalStartAt *time.Time `gorm:"column:class_attendance_session_original_start_at"`
		OriginalEndAt   *time.Time `gorm:"column:class_attendance_session_original_end_at"`
		Kind            *string    `gorm:"column:class_attendance_session_kind"`
		OverrideReason  *string    `gorm:"column:class_attendance_session_override_reason"`
		OverrideEventID *uuid.UUID `gorm:"column:class_attendance_session_override_event_id"`

		// relasi guru & ruang
		TeacherID *uuid.UUID `gorm:"column:class_attendance_session_teacher_id"`
		RoomID    *uuid.UUID `gorm:"column:class_attendance_session_class_room_id"`

		// relasi CSST (header class-Subject-Teacher)
		CSSTID *uuid.UUID `gorm:"column:class_attendance_session_csst_id"`

		// teks tampilan
		Title        *string `gorm:"column:class_attendance_session_title"`
		DisplayTitle *string `gorm:"column:class_attendance_session_display_title"`
		Gen          *string `gorm:"column:class_attendance_session_general_info"`
		Note         *string `gorm:"column:class_attendance_session_note"`

		// counters attendance (rekap)
		PresentCount *int `gorm:"column:class_attendance_session_present_count"`
		AbsentCount  *int `gorm:"column:class_attendance_session_absent_count"`
		LateCount    *int `gorm:"column:class_attendance_session_late_count"`
		ExcusedCount *int `gorm:"column:class_attendance_session_excused_count"`
		SickCount    *int `gorm:"column:class_attendance_session_sick_count"`
		LeaveCount   *int `gorm:"column:class_attendance_session_leave_count"`

		// snapshot CSST (JSON) + turunan kolom snapshot
		CSSTSnap datatypes.JSON `gorm:"column:class_attendance_session_csst_snapshot"`

		CSSTIDSnapshot      *uuid.UUID `gorm:"column:class_attendance_session_csst_id_snapshot"`
		SubjectIDSnapshot   *uuid.UUID `gorm:"column:class_attendance_session_subject_id_snapshot"`
		SectionIDSnapshot   *uuid.UUID `gorm:"column:class_attendance_session_section_id_snapshot"`
		TeacherIDSnapshot   *uuid.UUID `gorm:"column:class_attendance_session_teacher_id_snapshot"`
		RoomIDSnapshot      *uuid.UUID `gorm:"column:class_attendance_session_room_id_snapshot"`
		SubjectCodeSnapshot *string    `gorm:"column:class_attendance_session_subject_code_snapshot"`
		SubjectNameSnapshot *string    `gorm:"column:class_attendance_session_subject_name_snapshot"`
		SectionNameSnapshot *string    `gorm:"column:class_attendance_session_section_name_snapshot"`
		TeacherNameSnapshot *string    `gorm:"column:class_attendance_session_teacher_name_snapshot"`
		RoomNameSnapshot    *string    `gorm:"column:class_attendance_session_room_name_snapshot"`

		// rule snapshot (JSON & turunan)
		RuleSnapshot           datatypes.JSON `gorm:"column:class_attendance_session_rule_snapshot"`
		RuleDayOfWeekSnapshot  *int           `gorm:"column:class_attendance_session_rule_day_of_week_snapshot"`
		RuleStartTimeSnapshot  *string        `gorm:"column:class_attendance_session_rule_start_time_snapshot"`
		RuleEndTimeSnapshot    *string        `gorm:"column:class_attendance_session_rule_end_time_snapshot"`
		RuleWeekParitySnapshot *string        `gorm:"column:class_attendance_session_rule_week_parity_snapshot"`

		// audit
		CreatedAt time.Time  `gorm:"column:class_attendance_session_created_at"`
		UpdatedAt time.Time  `gorm:"column:class_attendance_session_updated_at"`
		DeletedAt *time.Time `gorm:"column:class_attendance_session_deleted_at"`
	}

	var rows []row

	// build query select + order (order kedua hanya dipakai di mode biasa)
	qSelect := qBase.
		Select(`
			cas.class_attendance_session_id,
			cas.class_attendance_session_school_id,
			cas.class_attendance_session_schedule_id,
			cas.class_attendance_session_rule_id,
			cas.class_attendance_session_slug,
			cas.class_attendance_session_date,
			cas.class_attendance_session_starts_at,
			cas.class_attendance_session_ends_at,
			cas.class_attendance_session_status,
			cas.class_attendance_session_attendance_status,
			cas.class_attendance_session_locked,
			cas.class_attendance_session_is_override,
			cas.class_attendance_session_is_canceled,
			cas.class_attendance_session_original_start_at,
			cas.class_attendance_session_original_end_at,
			cas.class_attendance_session_kind,
			cas.class_attendance_session_override_reason,
			cas.class_attendance_session_override_event_id,
			cas.class_attendance_session_teacher_id,
			cas.class_attendance_session_class_room_id,
			cas.class_attendance_session_csst_id,
			cas.class_attendance_session_title,
			cas.class_attendance_session_display_title,
			cas.class_attendance_session_general_info,
			cas.class_attendance_session_note,
			cas.class_attendance_session_present_count,
			cas.class_attendance_session_absent_count,
			cas.class_attendance_session_late_count,
			cas.class_attendance_session_excused_count,
			cas.class_attendance_session_sick_count,
			cas.class_attendance_session_leave_count,
			cas.class_attendance_session_csst_snapshot,
			cas.class_attendance_session_csst_id_snapshot,
			cas.class_attendance_session_subject_id_snapshot,
			cas.class_attendance_session_section_id_snapshot,
			cas.class_attendance_session_teacher_id_snapshot,
			cas.class_attendance_session_room_id_snapshot,
			cas.class_attendance_session_subject_code_snapshot,
			cas.class_attendance_session_subject_name_snapshot,
			cas.class_attendance_session_section_name_snapshot,
			cas.class_attendance_session_teacher_name_snapshot,
			cas.class_attendance_session_room_name_snapshot,
			cas.class_attendance_session_rule_snapshot,
			cas.class_attendance_session_rule_day_of_week_snapshot,
			cas.class_attendance_session_rule_start_time_snapshot,
			cas.class_attendance_session_rule_end_time_snapshot,
			cas.class_attendance_session_rule_week_parity_snapshot,
			cas.class_attendance_session_created_at,
			cas.class_attendance_session_updated_at,
			cas.class_attendance_session_deleted_at
		`).
		Order(orderExpr)

	if !isNearest {
		// mode biasa: tambahkan order tambahan by date desc
		qSelect = qSelect.Order("cas.class_attendance_session_date DESC, cas.class_attendance_session_id DESC")
	}

	if err := qSelect.
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	pageIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		pageIDs = append(pageIDs, r.ID)
	}

	// ===== Prefetch UA (opsional) =====
	// ===== Prefetch Participants (opsional) =====
	type SessionParticipantLite struct {
		ParticipantID     uuid.UUID  `json:"participant_id"`
		SessionID         uuid.UUID  `json:"participant_session_id"`
		SchoolStudentID   *uuid.UUID `json:"participant_school_student_id,omitempty"`
		Kind              string     `json:"participant_kind"`
		State             string     `json:"participant_state"`
		TypeID            *uuid.UUID `json:"participant_type_id,omitempty"`
		Desc              *string    `json:"participant_desc,omitempty"`
		Score             *float64   `json:"participant_score,omitempty"`
		IsPassed          *bool      `json:"participant_is_passed,omitempty"`
		UserNote          *string    `json:"participant_user_note,omitempty"`
		TeacherNote       *string    `json:"participant_teacher_note,omitempty"`
		CheckinAt         *time.Time `json:"participant_checkin_at,omitempty"`
		CheckoutAt        *time.Time `json:"participant_checkout_at,omitempty"`
		LateSeconds       *int       `json:"participant_late_seconds,omitempty"`
		CreatedAt         time.Time  `json:"participant_created_at"`
		UpdatedAt         time.Time  `json:"participant_updated_at"`
		MarkedAt          *time.Time `json:"participant_marked_at,omitempty"`
		MarkedByTeacherID *uuid.UUID `json:"participant_marked_by_teacher_id,omitempty"`
		Method            *string    `json:"participant_method,omitempty"`
		TeacherRole       *string    `json:"participant_teacher_role,omitempty"`
		Lat               *float64   `json:"participant_lat,omitempty"`
		Lng               *float64   `json:"participant_lng,omitempty"`
		DistanceM         *int       `json:"participant_distance_m,omitempty"`
	}

	partMap := map[uuid.UUID][]SessionParticipantLite{}

	if wantParticipants && len(rows) > 0 {
		// filter via query param baru
		state := strings.ToLower(strings.TrimSpace(c.Query("participant_state")))
		kind := strings.ToLower(strings.TrimSpace(c.Query("participant_kind")))

		typeIDPtr, err := parseUUIDPtr(c.Query("participant_type_id"), "participant_type_id")
		if err != nil {
			return err
		}

		// filter by student
		var studentIDs []uuid.UUID
		if ids, err := parseUUIDList(c.Query("participant_student_id")); err != nil {
			return err
		} else if len(ids) > 0 {
			studentIDs = ids
		} else if ids, err := parseUUIDList(c.Query("school_student_id")); err != nil {
			return err
		} else if len(ids) > 0 {
			studentIDs = ids
		}

		// text search
		var like *string
		if q := strings.TrimSpace(c.Query("participant_q")); q != "" {
			pat := "%" + q + "%"
			like = &pat
		}

		// is_passed
		var isPassedPtr *bool
		if s := strings.TrimSpace(c.Query("participant_is_passed")); s != "" {
			if b, e := strconv.ParseBool(s); e == nil {
				isPassedPtr = &b
			} else {
				return helper.JsonError(c, fiber.StatusBadRequest, "participant_is_passed tidak valid (true/false)")
			}
		}

		// ==== QUERY ke tabel baru: class_attendance_session_participants ====
		paQ := ctrl.DB.Table("class_attendance_session_participants AS p").
			Where("p.class_attendance_session_participant_deleted_at IS NULL").
			Where("p.class_attendance_session_participant_school_id = ?", schoolID).
			Where("p.class_attendance_session_participant_session_id IN ?", pageIDs)

		if state != "" {
			paQ = paQ.Where("LOWER(p.class_attendance_session_participant_state) = ?", state)
		}
		if kind != "" {
			paQ = paQ.Where("LOWER(p.class_attendance_session_participant_kind) = ?", kind)
		}
		if typeIDPtr != nil {
			paQ = paQ.Where("p.class_attendance_session_participant_type_id = ?", *typeIDPtr)
		}
		if len(studentIDs) > 0 {
			paQ = paQ.Where("p.class_attendance_session_participant_school_student_id IN ?", studentIDs)
		}
		if like != nil {
			paQ = paQ.Where(`
				(p.class_attendance_session_participant_desc ILIKE ?
				 OR p.class_attendance_session_participant_user_note ILIKE ?
				 OR p.class_attendance_session_participant_teacher_note ILIKE ?)`,
				*like, *like, *like)
		}
		if isPassedPtr != nil {
			paQ = paQ.Where("p.class_attendance_session_participant_is_passed = ?", *isPassedPtr)
		}

		// Role-scope Student/Parent (hanya boleh lihat participant miliknya)
		if !isAdmin && !isTeacher {
			if userID == uuid.Nil {
				return helper.JsonError(c, fiber.StatusUnauthorized, "User tidak terautentik")
			}
			paQ = paQ.Joins(`
				JOIN school_students ms
				  ON ms.school_student_id = p.class_attendance_session_participant_school_student_id
				 AND ms.school_student_deleted_at IS NULL
				 AND ms.school_student_user_id = ?
				 AND ms.school_student_school_id = ?
			`, userID, schoolID)
		}

		type paRow struct {
			ID, SessionID     uuid.UUID
			SchoolStudentID   *uuid.UUID
			Kind, State       string
			TypeID            *uuid.UUID
			Desc, UserNote    *string
			TeacherNote       *string
			Score             *float64
			IsPassed          *bool
			CheckinAt         *time.Time
			CheckoutAt        *time.Time
			LateSeconds       *int
			CreatedAt         time.Time
			UpdatedAt         time.Time
			MarkedAt          *time.Time
			MarkedByTeacherID *uuid.UUID
			Method            *string
			TeacherRole       *string
			Lat               *float64
			Lng               *float64
			DistanceM         *int
		}

		var paRows []paRow
		if err := paQ.
			Select(`
				p.class_attendance_session_participant_id,
				p.class_attendance_session_participant_session_id,
				p.class_attendance_session_participant_school_student_id,
				p.class_attendance_session_participant_kind,
				p.class_attendance_session_participant_state,
				p.class_attendance_session_participant_type_id,
				p.class_attendance_session_participant_desc,
				p.class_attendance_session_participant_score,
				p.class_attendance_session_participant_is_passed,
				p.class_attendance_session_participant_user_note,
				p.class_attendance_session_participant_teacher_note,
				p.class_attendance_session_participant_checkin_at,
				p.class_attendance_session_participant_checkout_at,
				p.class_attendance_session_participant_late_seconds,
				p.class_attendance_session_participant_marked_at,
				p.class_attendance_session_participant_marked_by_teacher_id,
				p.class_attendance_session_participant_method,
				p.class_attendance_session_participant_teacher_role,
				p.class_attendance_session_participant_lat,
				p.class_attendance_session_participant_lng,
				p.class_attendance_session_participant_distance_m,
				p.class_attendance_session_participant_created_at,
				p.class_attendance_session_participant_updated_at
			`).
			Order(`
				p.class_attendance_session_participant_session_id ASC,
				p.class_attendance_session_participant_created_at ASC,
				p.class_attendance_session_participant_id ASC
			`).
			Find(&paRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data peserta absensi")
		}

		for _, r := range paRows {
			partMap[r.SessionID] = append(partMap[r.SessionID], SessionParticipantLite{
				ParticipantID:     r.ID,
				SessionID:         r.SessionID,
				SchoolStudentID:   r.SchoolStudentID,
				Kind:              r.Kind,
				State:             r.State,
				TypeID:            r.TypeID,
				Desc:              r.Desc,
				Score:             r.Score,
				IsPassed:          r.IsPassed,
				UserNote:          r.UserNote,
				TeacherNote:       r.TeacherNote,
				CheckinAt:         r.CheckinAt,
				CheckoutAt:        r.CheckoutAt,
				LateSeconds:       r.LateSeconds,
				CreatedAt:         r.CreatedAt,
				UpdatedAt:         r.UpdatedAt,
				MarkedAt:          r.MarkedAt,
				MarkedByTeacherID: r.MarkedByTeacherID,
				Method:            r.Method,
				TeacherRole:       r.TeacherRole,
				Lat:               r.Lat,
				Lng:               r.Lng,
				DistanceM:         r.DistanceM,
			})
		}
	}

	// ===== Compose response (UPDATED to DTO) =====
	buildBase := func(r row) sessiondto.ClassAttendanceSessionResponse {
		gen := ""
		if r.Gen != nil {
			gen = *r.Gen
		}

		return sessiondto.ClassAttendanceSessionResponse{
			// kunci & tenant
			ClassAttendanceSessionId:       r.ID,
			ClassAttendanceSessionSchoolId: r.SchoolID,

			// relasi jadwal
			ClassAttendanceSessionScheduleId: r.ScheduleID,
			ClassAttendanceSessionSlug:       r.Slug,

			// waktu
			ClassAttendanceSessionDate:     r.Date,
			ClassAttendanceSessionStartsAt: r.StartsAt,
			ClassAttendanceSessionEndsAt:   r.EndsAt,

			// status
			ClassAttendanceSessionStatus:           r.Status,
			ClassAttendanceSessionAttendanceStatus: r.AttendanceStatus,
			ClassAttendanceSessionLocked:           r.Locked,
			ClassAttendanceSessionIsOverride:       r.IsOverride,
			ClassAttendanceSessionIsCanceled:       r.IsCanceled,

			ClassAttendanceSessionOriginalStartAt: r.OriginalStartAt,
			ClassAttendanceSessionOriginalEndAt:   r.OriginalEndAt,
			ClassAttendanceSessionKind:            r.Kind,
			ClassAttendanceSessionOverrideReason:  r.OverrideReason,
			ClassAttendanceSessionOverrideEventId: r.OverrideEventID,

			// relasi guru & ruang
			ClassAttendanceSessionTeacherId:   r.TeacherID,
			ClassAttendanceSessionClassRoomId: r.RoomID,

			// CSST relasi
			ClassAttendanceSessionCSSTId: r.CSSTID,

			// teks tampilan
			ClassAttendanceSessionTitle:        r.Title,
			ClassAttendanceSessionDisplayTitle: r.DisplayTitle,
			ClassAttendanceSessionGeneralInfo:  gen,
			ClassAttendanceSessionNote:         r.Note,

			// counters
			ClassAttendanceSessionPresentCount: r.PresentCount,
			ClassAttendanceSessionAbsentCount:  r.AbsentCount,
			ClassAttendanceSessionLateCount:    r.LateCount,
			ClassAttendanceSessionExcusedCount: r.ExcusedCount,
			ClassAttendanceSessionSickCount:    r.SickCount,
			ClassAttendanceSessionLeaveCount:   r.LeaveCount,

			// snapshot CSST — dikembalikan sebagai map (bukan string mentah)
			ClassAttendanceSessionCSSTSnapshot: jsonToMap(r.CSSTSnap),

			ClassAttendanceSessionCSSTIdSnapshot:      r.CSSTIDSnapshot,
			ClassAttendanceSessionSubjectIdSnapshot:   r.SubjectIDSnapshot,
			ClassAttendanceSessionSectionIdSnapshot:   r.SectionIDSnapshot,
			ClassAttendanceSessionTeacherIdSnapshot:   r.TeacherIDSnapshot,
			ClassAttendanceSessionRoomIdSnapshot:      r.RoomIDSnapshot,
			ClassAttendanceSessionSubjectCodeSnapshot: r.SubjectCodeSnapshot,
			ClassAttendanceSessionSubjectNameSnapshot: r.SubjectNameSnapshot,
			ClassAttendanceSessionSectionNameSnapshot: r.SectionNameSnapshot,
			ClassAttendanceSessionTeacherNameSnapshot: r.TeacherNameSnapshot,
			ClassAttendanceSessionRoomNameSnapshot:    r.RoomNameSnapshot,

			// rule snapshot — juga dikembalikan sebagai map
			ClassAttendanceSessionRuleSnapshot:           jsonToMap(r.RuleSnapshot),
			ClassAttendanceSessionRuleDayOfWeekSnapshot:  r.RuleDayOfWeekSnapshot,
			ClassAttendanceSessionRuleStartTimeSnapshot:  parseRuleTimeToPtr(r.RuleStartTimeSnapshot),
			ClassAttendanceSessionRuleEndTimeSnapshot:    parseRuleTimeToPtr(r.RuleEndTimeSnapshot),
			ClassAttendanceSessionRuleWeekParitySnapshot: r.RuleWeekParitySnapshot,

			// audit
			ClassAttendanceSessionCreatedAt: r.CreatedAt,
			ClassAttendanceSessionUpdatedAt: r.UpdatedAt,
			ClassAttendanceSessionDeletedAt: r.DeletedAt,
		}
	}

	// ===== Meta =====
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	if wantParticipants {
		type SessionWithParticipants struct {
			sessiondto.ClassAttendanceSessionResponse
			Participants []SessionParticipantLite `json:"participants,omitempty"`
		}
		out := make([]SessionWithParticipants, 0, len(rows))
		for _, r := range rows {
			out = append(out, SessionWithParticipants{
				ClassAttendanceSessionResponse: buildBase(r),
				Participants:                   partMap[r.ID],
			})
		}
		return helper.JsonList(c, "ok", out, pg)
	}

	items := make([]sessiondto.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, buildBase(r))
	}
	return helper.JsonList(c, "ok", items, pg)
}
