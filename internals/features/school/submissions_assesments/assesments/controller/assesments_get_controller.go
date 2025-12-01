// file: internals/features/school/assessments/controller/assessment_list_controller.go
package controller

import (
	"log"
	"strings"
	"time"

	dto "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	quizDTO "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/dto"
	quizModel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	submissionModel "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* =========================
   Small helpers (local)
========================= */

func getSortClauseAssessment(sortBy, sortDir *string) string {
	col := "assessment_created_at"
	if sortBy != nil {
		switch strings.ToLower(strings.TrimSpace(*sortBy)) {
		case "title":
			col = "assessment_title"
		case "start_at":
			col = "assessment_start_at"
		case "due_at":
			col = "assessment_due_at"
		case "created_at":
			col = "assessment_created_at"
		case "quiz_total", "assessment_quiz_total":
			col = "assessment_quiz_total"
		}
	}
	dir := "DESC"
	if sortDir != nil && strings.EqualFold(strings.TrimSpace(*sortDir), "asc") {
		dir = "ASC"
	}
	return col + " " + dir
}

// parseYmd versi lokal (YYYY-MM-DD → time.Time di awal hari, local time)
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

// queryBoolFlag: "1"/"true"/"yes" → true
func queryBoolFlag(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	return raw == "1" || raw == "true" || raw == "yes"
}

// resolveTimelineDateRange dipakai di mode timeline (student/teacher)
func resolveTimelineDateRange(c *fiber.Ctx) (*time.Time, *time.Time, error) {
	var df, dt *time.Time
	var err error

	monthRaw := strings.TrimSpace(c.Query("month"))
	rangeRaw := strings.ToLower(strings.TrimSpace(c.Query("range")))

	// date_from/date_to (optional)
	if s := strings.TrimSpace(c.Query("date_from")); s != "" {
		df, err = parseYmd(s)
		if err != nil {
			return nil, nil, helper.JsonError(c, fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
		}
	}
	if s := strings.TrimSpace(c.Query("date_to")); s != "" {
		dt, err = parseYmd(s)
		if err != nil {
			return nil, nil, helper.JsonError(c, fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
		}
	}

	// range: today / week / sepekan / next7
	isTodayRange :=
		rangeRaw == "today" ||
			rangeRaw == "hari_ini" ||
			rangeRaw == "today_only"

	isWeekRange :=
		rangeRaw == "week" ||
			rangeRaw == "next7" ||
			rangeRaw == "sepekan"

	if isTodayRange || isWeekRange {
		now := time.Now().In(time.Local)
		todayLocal := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

		start := todayLocal
		var end time.Time
		if isTodayRange {
			end = todayLocal
		} else {
			end = todayLocal.AddDate(0, 0, 7)
		}

		df = &start
		dt = &end
		monthRaw = ""
	}

	// month=YYYY-MM → override ke full 1 bulan
	if monthRaw != "" && !isTodayRange && !isWeekRange {
		mt, err2 := time.ParseInLocation("2006-01", monthRaw, time.Local)
		if err2 != nil {
			return nil, nil, helper.JsonError(c, fiber.StatusBadRequest, "month tidak valid (YYYY-MM)")
		}
		firstOfMonth := time.Date(mt.Year(), mt.Month(), 1, 0, 0, 0, 0, time.Local)
		lastOfMonth := time.Date(mt.Year(), mt.Month()+1, 0, 0, 0, 0, 0, time.Local)

		df = &firstOfMonth
		dt = &lastOfMonth
	}

	return df, dt, nil
}

// Resolve school utk list assessment
func resolveSchoolForAssessmentList(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) Dari token: active_school
	if sid, err := helperAuth.GetActiveSchoolID(c); err == nil && sid != uuid.Nil {
		if !helperAuth.UserHasSchool(c, sid) {
			return uuid.Nil, helperAuth.ErrSchoolContextForbidden
		}
		return sid, nil
	}

	// 2) Fallback: ResolveSchoolContext (id/slug/host)
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return uuid.Nil, err
	}

	// ID langsung
	if mc.ID != uuid.Nil {
		if !helperAuth.UserHasSchool(c, mc.ID) {
			return uuid.Nil, helperAuth.ErrSchoolContextForbidden
		}
		return mc.ID, nil
	}

	// Slug → id
	if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		if !helperAuth.UserHasSchool(c, id) {
			return uuid.Nil, helperAuth.ErrSchoolContextForbidden
		}
		return id, nil
	}

	// 3) Kalau semua gagal → context missing
	return uuid.Nil, helperAuth.ErrSchoolContextMissing
}

// Ringkasan attempt quiz per student (dipakai di student_timeline)
type studentQuizAttemptLite struct {
	AttemptID uuid.UUID `json:"attempt_id"`
	QuizID    uuid.UUID `json:"quiz_id"`

	Status string `json:"status"` // in_progress | submitted | finished | abandoned | not_attempted
	Count  int    `json:"count"`

	BestRaw        *float64   `json:"best_raw,omitempty"`
	BestPercent    *float64   `json:"best_percent,omitempty"`
	BestStartedAt  *time.Time `json:"best_started_at,omitempty"`
	BestFinishedAt *time.Time `json:"best_finished_at,omitempty"`

	LastRaw        *float64   `json:"last_raw,omitempty"`
	LastPercent    *float64   `json:"last_percent,omitempty"`
	LastStartedAt  *time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt *time.Time `json:"last_finished_at,omitempty"`
}

// timelineProgress lebih informatif
type timelineProgress struct {
	State       string     `json:"state"`                  // not_opened | ongoing | overdue | submitted | submitted_late | graded | graded_late | unknown
	Overdue     bool       `json:"overdue"`                // true kalau sudah lewat due_at (dan belum submit/graded tepat waktu)
	StartAt     *time.Time `json:"start_at,omitempty"`     // copy dari assessment_start_at
	DueAt       *time.Time `json:"due_at,omitempty"`       // copy dari assessment_due_at
	SubmittedAt *time.Time `json:"submitted_at,omitempty"` // dari submission
	GradedAt    *time.Time `json:"graded_at,omitempty"`    // dari submission
	Score       *float64   `json:"score"`                  // SELALU dikirim, null kalau belum dinilai
	Status      string     `json:"status,omitempty"`       // copy dari submission_status
}

// Participant (untuk student/teacher timeline assessment)
type AssessmentParticipantLite struct {
	ParticipantID    uuid.UUID `json:"participant_id"`
	ParticipantState string    `json:"participant_state"`
}

// Quiz + attempt student (untuk mode student_timeline)
type quizWithAttempt struct {
	quizDTO.QuizResponse
	StudentAttempt *studentQuizAttemptLite `json:"student_attempt,omitempty"`
}

// helper: pilih waktu paling "akhir" dari dua pointer (boleh nil)
func latestTime(a, b *time.Time) time.Time {
	zero := time.Time{}
	if a == nil && b == nil {
		return zero
	}
	if a == nil {
		return *b
	}
	if b == nil {
		return *a
	}
	if a.After(*b) {
		return *a
	}
	return *b
}

// GET /assessments
func (ctl *AssessmentController) List(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	// 1) Resolve school
	mid, err := resolveSchoolForAssessmentList(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ============================
	// MODE TIMELINE (student/teacher)
	// ============================
	isStudentTimeline := queryBoolFlag(c.Query("student_timeline"))
	isTeacherTimeline := queryBoolFlag(c.Query("teacher_timeline"))

	if isStudentTimeline && isTeacherTimeline {
		return helper.JsonError(c, fiber.StatusBadRequest, "student_timeline dan teacher_timeline tidak boleh keduanya 1")
	}

	var (
		studentID uuid.UUID
		teacherID uuid.UUID
		df, dt    *time.Time
	)

	if isStudentTimeline {
		if err := helperAuth.EnsureStudentSchool(c, mid); err != nil {
			return err
		}
		studentID, err = helperAuth.GetSchoolStudentIDForSchool(c, mid)
		if err != nil {
			return err
		}
		if studentID == uuid.Nil {
			return helper.JsonError(c, fiber.StatusForbidden, "school_student_id tidak ditemukan di token")
		}
		df, dt, err = resolveTimelineDateRange(c)
		if err != nil {
			return err
		}
	} else if isTeacherTimeline {
		if err := helperAuth.EnsureTeacherSchool(c, mid); err != nil {
			return err
		}
		teacherID, err = helperAuth.GetSchoolTeacherIDForSchool(c, mid)
		if err != nil {
			return err
		}
		if teacherID == uuid.Nil {
			return helper.JsonError(c, fiber.StatusForbidden, "school_teacher_id tidak ditemukan di token")
		}
		df, dt, err = resolveTimelineDateRange(c)
		if err != nil {
			return err
		}
	}

	// 2) Query parameters
	var (
		typeIDStr = strings.TrimSpace(c.Query("type_id"))
		csstIDStr = strings.TrimSpace(c.Query("csst_id"))

		idStr  = strings.TrimSpace(c.Query("id"))
		idsStr = strings.TrimSpace(c.Query("ids"))

		qStr     = strings.TrimSpace(c.Query("q"))
		isPubStr = strings.TrimSpace(c.Query("is_published"))
		limit    = atoiOr(20, c.Query("limit"))
		offset   = atoiOr(0, c.Query("offset"))
		sortBy   = strings.TrimSpace(c.Query("sort_by"))
		sortDir  = strings.TrimSpace(c.Query("sort_dir"))
	)

	withURLs := eqTrue(c.Query("with_urls"))
	urlsPublishedOnly := eqTrue(c.Query("urls_published_only"))
	urlsLimitPer := atoiOr(0, c.Query("urls_limit_per"))
	urlsOrder := strings.ToLower(strings.TrimSpace(c.Query("urls_order")))

	// parse filter type & csst
	var typeID, csstID *uuid.UUID
	if typeIDStr != "" {
		if u, e := uuid.Parse(typeIDStr); e == nil {
			typeID = &u
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "type_id tidak valid")
		}
	}
	if csstIDStr != "" {
		if u, e := uuid.Parse(csstIDStr); e == nil {
			csstID = &u
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "csst_id tidak valid")
		}
	}

	// filter id & ids
	var (
		assessmentID  *uuid.UUID
		assessmentIDs []uuid.UUID
	)

	if idStr != "" {
		if u, e := uuid.Parse(idStr); e == nil {
			assessmentID = &u
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
		}
	}

	if idsStr != "" {
		parts := strings.Split(idsStr, ",")
		for _, p := range parts {
			s := strings.TrimSpace(p)
			if s == "" {
				continue
			}
			u, e := uuid.Parse(s)
			if e != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "ids mengandung UUID yang tidak valid")
			}
			assessmentIDs = append(assessmentIDs, u)
		}
	}

	// filter boolean
	var isPublished *bool
	if isPubStr != "" {
		b := strings.EqualFold(isPubStr, "true") || isPubStr == "1"
		isPublished = &b
	}

	var isGraded *bool
	if gs := strings.TrimSpace(c.Query("is_graded")); gs != "" {
		b := strings.EqualFold(gs, "true") || gs == "1"
		isGraded = &b
	}

	// sorting
	var sbPtr, sdPtr *string
	if sortBy != "" {
		sbPtr = &sortBy
	}
	if sortDir != "" {
		sdPtr = &sortDir
	}

	// 4) Base query
	qry := ctl.DB.WithContext(c.Context()).
		Model(&model.AssessmentModel{}).
		Where("assessment_school_id = ? AND assessment_deleted_at IS NULL", mid)

	// SCOPE TIMELINE
	if isStudentTimeline {
		qry = qry.Joins(`
			JOIN student_class_section_subject_teachers scst
			  ON scst.student_class_section_subject_teacher_school_id = assessment_school_id
			 AND scst.student_class_section_subject_teacher_csst_id = assessment_class_section_subject_teacher_id
			 AND scst.student_class_section_subject_teacher_student_id = ?
			 AND scst.student_class_section_subject_teacher_is_active = TRUE
			 AND scst.student_class_section_subject_teacher_deleted_at IS NULL
		`, studentID)
	}
	if isTeacherTimeline {
		qry = qry.Joins(`
			JOIN class_section_subject_teachers csst
			  ON csst.class_section_subject_teacher_school_id = assessment_school_id
			 AND csst.class_section_subject_teacher_id = assessment_class_section_subject_teacher_id
			 AND csst.class_section_subject_teacher_teacher_id = ?
			 AND csst.class_section_subject_teacher_is_active = TRUE
			 AND csst.class_section_subject_teacher_deleted_at IS NULL
		`, teacherID)
	}

	// Filter tanggal timeline
	if (isStudentTimeline || isTeacherTimeline) && (df != nil || dt != nil) {
		if df != nil && dt != nil {
			qry = qry.Where(`
				(
					assessment_start_at IS NOT NULL
					AND assessment_start_at BETWEEN ? AND ?
				)
				OR
				(
					assessment_start_at IS NULL
					AND assessment_due_at IS NOT NULL
					AND assessment_due_at BETWEEN ? AND ?
				)
				OR
				(
					assessment_start_at IS NULL
					AND assessment_due_at IS NULL
					AND assessment_created_at BETWEEN ? AND ?
				)
			`, *df, *dt, *df, *dt, *df, *dt)
		} else if df != nil {
			qry = qry.Where(`
				(
					assessment_start_at IS NOT NULL
					AND assessment_start_at >= ?
				)
				OR
				(
					assessment_start_at IS NULL
					AND assessment_due_at IS NOT NULL
					AND assessment_due_at >= ?
				)
				OR
				(
					assessment_start_at IS NULL
					AND assessment_due_at IS NULL
					AND assessment_created_at >= ?
				)
			`, *df, *df, *df)
		} else if dt != nil {
			qry = qry.Where(`
				(
					assessment_start_at IS NOT NULL
					AND assessment_start_at <= ?
				)
				OR
				(
					assessment_start_at IS NULL
					AND assessment_due_at IS NOT NULL
					AND assessment_due_at <= ?
				)
				OR
				(
					assessment_start_at IS NULL
					AND assessment_due_at IS NULL
					AND assessment_created_at <= ?
				)
			`, *dt, *dt, *dt)
		}
	}

	// APPLY FILTERS umum
	if typeID != nil {
		qry = qry.Where("assessment_type_id = ?", *typeID)
	}
	if csstID != nil {
		qry = qry.Where("assessment_class_section_subject_teacher_id = ?", *csstID)
	}
	if assessmentID != nil {
		qry = qry.Where("assessment_id = ?", *assessmentID)
	}
	if len(assessmentIDs) > 0 {
		qry = qry.Where("assessment_id IN ?", assessmentIDs)
	}
	if isPublished != nil {
		qry = qry.Where("assessment_is_published = ?", *isPublished)
	}
	if isGraded != nil {
		qry = qry.Where("assessment_type_is_graded_snapshot = ?", *isGraded)
	}
	if qStr != "" {
		q := "%" + strings.ToLower(qStr) + "%"
		qry = qry.Where(
			"(LOWER(assessment_title) LIKE ? OR LOWER(COALESCE(assessment_description, '')) LIKE ?)",
			q, q,
		)
	}

	// total
	var total int64
	if err := qry.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// page data
	var rows []model.AssessmentModel
	if limit > 0 {
		qry = qry.Limit(limit).Offset(offset)
	}
	if err := qry.
		Order(getSortClauseAssessment(sbPtr, sdPtr)).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// 5) Build response DTO
	type assessmentWithExpand struct {
		dto.AssessmentResponse
		URLsCount       *int              `json:"urls_count,omitempty"`
		Quizzes         []quizWithAttempt `json:"quizzes,omitempty"`
		Submissions     *timelineProgress `json:"submissions,omitempty"`      // ⬅️ ganti dari student_progress
		TeacherProgress *timelineProgress `json:"teacher_progress,omitempty"` // utk teacher_timeline

		Participant *AssessmentParticipantLite `json:"participant,omitempty"`
	}

	out := make([]assessmentWithExpand, 0, len(rows))
	for i := range rows {
		item := assessmentWithExpand{
			AssessmentResponse: dto.FromModelAssesment(rows[i]),
		}

		if isStudentTimeline || isTeacherTimeline {
			item.Participant = &AssessmentParticipantLite{
				ParticipantID:    uuid.Nil,
				ParticipantState: "unknown",
			}
		}

		out = append(out, item)
	}

	// ===========================================
	// PREFETCH SUBMISSIONS (khusus student_timeline)
	// ===========================================
	type submissionRow struct {
		AssessmentID uuid.UUID  `gorm:"column:submission_assessment_id"`
		SubmissionID uuid.UUID  `gorm:"column:submission_id"`
		Status       string     `gorm:"column:submission_status"`
		Score        *float64   `gorm:"column:submission_score"`
		SubmittedAt  *time.Time `gorm:"column:submission_submitted_at"`
		GradedAt     *time.Time `gorm:"column:submission_graded_at"`
	}

	submissionMap := map[uuid.UUID]submissionRow{}

	if isStudentTimeline && len(rows) > 0 {
		aIDs := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			aIDs = append(aIDs, r.AssessmentID)
		}

		log.Printf("[AssessmentList] PREFETCH SUBMISSIONS: school=%s student=%s assessment_ids=%v",
			mid.String(), studentID.String(), aIDs)

		var srows []submissionRow
		if err := ctl.DB.WithContext(c.Context()).
			Model(&submissionModel.SubmissionModel{}).
			Select(`
				submission_assessment_id,
				submission_id,
				submission_status,
				(submission_score)::float8 AS submission_score,
				submission_submitted_at,
				submission_graded_at
			`).
			Where("submission_deleted_at IS NULL").
			Where("submission_school_id = ?", mid).
			Where("submission_assessment_id IN ?", aIDs).
			Where("submission_student_id = ?", studentID).
			Scan(&srows).Error; err != nil {

			log.Printf("[AssessmentList] ERROR PREFETCH SUBMISSIONS: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil submissions")
		}

		log.Printf("[AssessmentList] PREFETCH SUBMISSIONS: got %d rows", len(srows))

		for _, sr := range srows {
			log.Printf("[AssessmentList] SUBMISSION ROW: assessment=%s submission=%s status=%s score=%v submitted_at=%v graded_at=%v",
				sr.AssessmentID, sr.SubmissionID, sr.Status, sr.Score, sr.SubmittedAt, sr.GradedAt)

			prev, ok := submissionMap[sr.AssessmentID]
			if !ok {
				submissionMap[sr.AssessmentID] = sr
				continue
			}

			prevIsGraded := strings.EqualFold(prev.Status, "graded") || prev.GradedAt != nil
			currIsGraded := strings.EqualFold(sr.Status, "graded") || sr.GradedAt != nil

			if !prevIsGraded && currIsGraded {
				submissionMap[sr.AssessmentID] = sr
				continue
			}

			prevTime := latestTime(prev.SubmittedAt, prev.GradedAt)
			currTime := latestTime(sr.SubmittedAt, sr.GradedAt)

			if currTime.After(prevTime) {
				submissionMap[sr.AssessmentID] = sr
			}
		}
	}

	// ===============================
	// HITUNG PROGRESS TIMELINE
	// ===============================
	if isStudentTimeline || isTeacherTimeline {
		now := time.Now().In(time.Local)

		for i := range rows {
			var (
				start = rows[i].AssessmentStartAt
				due   = rows[i].AssessmentDueAt
			)

			state := "unknown"
			overdue := false

			switch {
			case start != nil && now.Before(*start):
				state = "not_opened"
			case start != nil && (due == nil || now.Before(*due)) && !now.Before(*start):
				state = "ongoing"
			case start == nil && due != nil && now.Before(*due):
				state = "ongoing"
			case due != nil && now.After(*due):
				state = "overdue"
				overdue = true
			default:
				state = "unknown"
			}

			p := &timelineProgress{
				State:   state,
				Overdue: overdue,
				StartAt: start,
				DueAt:   due,
			}

			if isStudentTimeline {
				if sub, ok := submissionMap[rows[i].AssessmentID]; ok {
					p.SubmittedAt = sub.SubmittedAt
					p.GradedAt = sub.GradedAt
					p.Score = sub.Score
					p.Status = sub.Status

					isGradedSub := strings.EqualFold(sub.Status, "graded") || sub.GradedAt != nil

					switch {
					case isGradedSub:
						late := false
						if due != nil {
							tRef := latestTime(sub.SubmittedAt, sub.GradedAt)
							if !tRef.IsZero() && tRef.After(*due) {
								late = true
							}
						}
						if late {
							state = "graded_late"
						} else {
							state = "graded"
						}
						overdue = due != nil && now.After(*due)
					default:
						late := false
						if due != nil && sub.SubmittedAt != nil && sub.SubmittedAt.After(*due) {
							late = true
						}
						if late {
							state = "submitted_late"
						} else {
							state = "submitted"
						}
						overdue = due != nil && now.After(*due)
					}

					p.State = state
					p.Overdue = overdue

					if out[i].Participant == nil {
						out[i].Participant = &AssessmentParticipantLite{}
					}
					out[i].Participant.ParticipantID = sub.SubmissionID
					out[i].Participant.ParticipantState = state
				}
			}

			if isStudentTimeline {
				// ⬅️ ganti dari out[i].StudentProgress = p
				out[i].Submissions = p
			}
			if isTeacherTimeline {
				out[i].TeacherProgress = p
			}
		}
	}

	// ================================
	// SELALU: QUIZZES + STUDENT ATTEMPTS (per-quiz)
	// ================================
	if len(rows) > 0 {
		aIDs := make([]uuid.UUID, 0, len(rows))
		for i := range rows {
			aIDs = append(aIDs, rows[i].AssessmentID)
		}

		var qrows []quizModel.QuizModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&quizModel.QuizModel{}).
			Where("quiz_school_id = ? AND quiz_deleted_at IS NULL", mid).
			Where("quiz_assessment_id IN ?", aIDs).
			Find(&qrows).Error; err != nil {

			log.Printf("[AssessmentList] ERROR FETCH QUIZZES: %v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil quizzes")
		}

		log.Printf("[AssessmentList] QUIZZES: school=%s got %d quiz rows for assessments=%v",
			mid.String(), len(qrows), aIDs)

		// Map: assessment_id -> []quizWithAttempt
		quizMap := make(map[uuid.UUID][]quizWithAttempt, len(aIDs))
		for i := range qrows {
			if qrows[i].QuizAssessmentID == nil {
				log.Printf("[AssessmentList] QUIZ ROW WITHOUT assessment_id: quiz_id=%s", qrows[i].QuizID)
				continue
			}
			aid := *qrows[i].QuizAssessmentID
			qResp := quizDTO.FromModel(&qrows[i])

			quizMap[aid] = append(quizMap[aid], quizWithAttempt{
				QuizResponse:   qResp,
				StudentAttempt: nil, // diisi nanti kalau student_timeline
			})
		}

		// Prefetch attempts (map: quiz_id -> attempt)
		attemptByQuiz := map[uuid.UUID]*studentQuizAttemptLite{}

		if isStudentTimeline && len(qrows) > 0 {
			qIDs := make([]uuid.UUID, 0, len(qrows))
			for i := range qrows {
				qIDs = append(qIDs, qrows[i].QuizID)
			}

			log.Printf("[AssessmentList] PREFETCH QUIZ ATTEMPTS: school=%s student=%s quiz_ids=%v",
				mid.String(), studentID.String(), qIDs)

			var arows []quizModel.StudentQuizAttemptModel
			if err := ctl.DB.WithContext(c.Context()).
				Model(&quizModel.StudentQuizAttemptModel{}).
				Where("student_quiz_attempt_school_id = ?", mid).
				Where("student_quiz_attempt_student_id = ?", studentID).
				Where("student_quiz_attempt_quiz_id IN ?", qIDs).
				Find(&arows).Error; err != nil {

				log.Printf("[AssessmentList] ERROR FETCH QUIZ ATTEMPTS: %v", err)
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil quiz attempts")
			}

			log.Printf("[AssessmentList] QUIZ ATTEMPTS: got %d rows", len(arows))

			for i := range arows {
				ar := arows[i]

				log.Printf("[AssessmentList] QUIZ ATTEMPT ROW: quiz=%s attempt=%s status=%s count=%d",
					ar.StudentQuizAttemptQuizID, ar.StudentQuizAttemptID, ar.StudentQuizAttemptStatus, ar.StudentQuizAttemptCount)

				attemptByQuiz[ar.StudentQuizAttemptQuizID] = &studentQuizAttemptLite{
					AttemptID: ar.StudentQuizAttemptID,
					QuizID:    ar.StudentQuizAttemptQuizID,

					Status: string(ar.StudentQuizAttemptStatus),
					Count:  ar.StudentQuizAttemptCount,

					BestRaw:        ar.StudentQuizAttemptBestRaw,
					BestPercent:    ar.StudentQuizAttemptBestPercent,
					BestStartedAt:  ar.StudentQuizAttemptBestStartedAt,
					BestFinishedAt: ar.StudentQuizAttemptBestFinishedAt,

					LastRaw:        ar.StudentQuizAttemptLastRaw,
					LastPercent:    ar.StudentQuizAttemptLastPercent,
					LastStartedAt:  ar.StudentQuizAttemptLastStartedAt,
					LastFinishedAt: ar.StudentQuizAttemptLastFinishedAt,
				}
			}
		}

		// Tempel ke output
		for i := range rows {
			aid := rows[i].AssessmentID

			qs, ok := quizMap[aid]
			if !ok || len(qs) == 0 {
				continue
			}

			// Kalau bukan student_timeline → tinggal assign saja
			if !isStudentTimeline {
				out[i].Quizzes = qs
				continue
			}

			attachedCount := 0

			// Student timeline: isi StudentAttempt per quiz
			for j := range qs {
				qid := qs[j].QuizID

				if att, ok := attemptByQuiz[qid]; ok {
					qs[j].StudentAttempt = att
				} else {
					qs[j].StudentAttempt = &studentQuizAttemptLite{
						AttemptID: uuid.Nil,
						QuizID:    qid,
						Status:    "not_attempted",
						Count:     0,
					}
				}
				attachedCount++
			}

			log.Printf("[AssessmentList] ATTACH ATTEMPTS: assessment=%s attached_attempts=%d",
				aid.String(), attachedCount)

			out[i].Quizzes = qs
		}
	}

	// 6) Return response
	return helper.JsonListEx(
		c,
		"OK",
		out,
		helper.BuildPaginationFromOffset(total, offset, limit),
		fiber.Map{
			"with_urls":           withURLs,
			"urls_published_only": urlsPublishedOnly,
			"urls_limit_per":      urlsLimitPer,
			"urls_order":          urlsOrder,
		},
	)
}
