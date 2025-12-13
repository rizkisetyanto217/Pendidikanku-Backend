// file: internals/features/school/assessments/controller/assessment_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	"madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/service"
	dto "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"

	quizDTO "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/dto"
	quizModel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	submissionsModel "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/*
========================================================

	Controller

========================================================
*/
type AssessmentController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewAssessmentController(db *gorm.DB) *AssessmentController {
	return &AssessmentController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ========================================================
   Helpers
======================================================== */

// ====== util autofill judul & deskripsi ======
func autofillTitle(current string, csstName *string, annTitle *string) string {
	if t := strings.TrimSpace(current); t != "" {
		return t
	}
	if csstName != nil && strings.TrimSpace(*csstName) != "" {
		return "Penilaian " + strings.TrimSpace(*csstName)
	}
	if annTitle != nil && strings.TrimSpace(*annTitle) != "" {
		return "Penilaian - " + strings.TrimSpace(*annTitle)
	}
	return "Penilaian"
}

func autofillDesc(curr *string, ann *sessRow, col *sessRow) *string {
	has := func(s string) bool { return strings.TrimSpace(s) != "" }
	if curr != nil && has(*curr) {
		return curr
	}
	var parts []string
	if ann != nil {
		line := "Diumumkan pada sesi"
		if ann.Title != nil && has(*ann.Title) {
			line += " \"" + strings.TrimSpace(*ann.Title) + "\""
		}
		if ann.Date != nil {
			line += ", tanggal " + ann.Date.Format("2006-01-02")
		}
		parts = append(parts, line)
	}
	if col != nil {
		line := "Dikumpulkan pada sesi"
		if col.Title != nil && has(*col.Title) {
			line += " \"" + strings.TrimSpace(*col.Title) + "\""
		}
		if col.Date != nil {
			line += ", tanggal " + col.Date.Format("2006-01-02")
		}
		parts = append(parts, line)
	}
	if len(parts) == 0 {
		return curr
	}
	s := strings.Join(parts, ". ") + "."
	return &s
}

type sessRow struct {
	ID       uuid.UUID  `gorm:"column:id"`
	SchoolID uuid.UUID  `gorm:"column:school_id"`
	StartsAt *time.Time `gorm:"column:starts_at"`
	Date     *time.Time `gorm:"column:date"`
	Deleted  *time.Time `gorm:"column:deleted_at"`
	Title    *string    `gorm:"column:title"`
}

func (ctl *AssessmentController) fetchSess(c *fiber.Ctx, id uuid.UUID) (*sessRow, error) {
	if id == uuid.Nil {
		return nil, nil
	}
	var r sessRow
	err := ctl.DB.WithContext(c.Context()).
		Table("class_attendance_sessions").
		Select(`
			class_attendance_session_id                  AS id,
			class_attendance_session_school_id           AS school_id,
			class_attendance_session_starts_at           AS starts_at,
			(class_attendance_session_date)::timestamptz AS date,
			class_attendance_session_deleted_at          AS deleted_at,
			COALESCE(
				NULLIF(TRIM(class_attendance_session_display_title), ''),
				NULLIF(TRIM(class_attendance_session_title), '')
			) AS title
		`).
		Where("class_attendance_session_id = ?", id).
		Take(&r).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func pickTime(s *sessRow) time.Time {
	if s == nil {
		return time.Now().UTC()
	}
	if s.StartsAt != nil {
		return s.StartsAt.UTC()
	}
	if s.Date != nil {
		return s.Date.UTC()
	}
	return time.Now().UTC()
}

// validasi guru milik school
func (ctl *AssessmentController) assertTeacherBelongsToSchool(c *fiber.Ctx, schoolID uuid.UUID, teacherID *uuid.UUID) error {
	if teacherID == nil || *teacherID == uuid.Nil {
		return nil
	}

	var row struct {
		M uuid.UUID `gorm:"column:m"`
	}
	if err := ctl.DB.WithContext(c.Context()).
		Table("school_teachers").
		Select("school_teacher_school_id AS m").
		Where("school_teacher_id = ? AND school_teacher_deleted_at IS NULL", *teacherID).
		Take(&row).Error; err != nil {
		return err
	}

	if row.M != schoolID {
		return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik school Anda")
	}
	return nil
}

/* ========================================================
   CSST Counter Helpers (TX-aware) ✅ pakai kolom csst_*
======================================================== */

func (ctl *AssessmentController) csstIncTotalAssessmentsTx(tx *gorm.DB, c *fiber.Ctx, schoolID, csstID uuid.UUID) error {
	return tx.WithContext(c.Context()).
		Model(&csstModel.ClassSectionSubjectTeacherModel{}).
		Where(`
			csst_school_id = ?
			AND csst_id = ?
			AND csst_deleted_at IS NULL
		`, schoolID, csstID).
		Update(
			"csst_total_assessments",
			gorm.Expr("csst_total_assessments + 1"),
		).Error
}

func (ctl *AssessmentController) csstDecTotalAssessmentsTx(tx *gorm.DB, c *fiber.Ctx, schoolID, csstID uuid.UUID) error {
	return tx.WithContext(c.Context()).
		Model(&csstModel.ClassSectionSubjectTeacherModel{}).
		Where(`
			csst_school_id = ?
			AND csst_id = ?
			AND csst_deleted_at IS NULL
		`, schoolID, csstID).
		Update(
			"csst_total_assessments",
			gorm.Expr("CASE WHEN csst_total_assessments > 0 THEN csst_total_assessments - 1 ELSE 0 END"),
		).Error
}

/* ===============================
   Handlers
=============================== */

// POST /assessments
func (ctl *AssessmentController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	var req dto.CreateAssessmentWithQuizzesRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()

	quizParts := req.FlattenQuizzes()

	// ✅ hanya wajib quiz kalau kind = quiz
	kind := strings.ToLower(strings.TrimSpace(req.Assessment.AssessmentKind))
	if kind == "" {
		kind = "quiz"
	}
	if kind == "quiz" && len(quizParts) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Minimal harus ada 1 quiz untuk assessment_kind=quiz")
	}

	mid, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	req.Assessment.AssessmentSchoolID = mid

	if req.Assessment.AssessmentQuizTotal == nil || *req.Assessment.AssessmentQuizTotal <= 0 {
		qt := len(quizParts)
		req.Assessment.AssessmentQuizTotal = &qt
	}

	if err := ctl.Validator.Struct(&req.Assessment); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	for i := range quizParts {
		if err := ctl.Validator.Struct(&quizParts[i]); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// =============== CSST (opsional) ===============
	var csstName *string
	var csstMinPassingScore *int

	if req.Assessment.AssessmentClassSectionSubjectTeacherID != nil &&
		*req.Assessment.AssessmentClassSectionSubjectTeacherID != uuid.Nil {

		cs, er := service.ValidateAndCacheCSST(
			ctl.DB.WithContext(c.Context()),
			mid,
			*req.Assessment.AssessmentClassSectionSubjectTeacherID,
		)
		if er != nil {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
		}

		csstName = cs.Name
		csstMinPassingScore = cs.MinPassingScore

		if req.Assessment.AssessmentCreatedByTeacherID == nil &&
			cs.TeacherID != nil && *cs.TeacherID != uuid.Nil {
			tid := *cs.TeacherID
			req.Assessment.AssessmentCreatedByTeacherID = &tid
		}
	}

	if err := ctl.assertTeacherBelongsToSchool(
		c,
		mid,
		req.Assessment.AssessmentCreatedByTeacherID,
	); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	hasAnn := req.Assessment.AssessmentAnnounceSessionID != nil &&
		*req.Assessment.AssessmentAnnounceSessionID != uuid.Nil
	hasCol := req.Assessment.AssessmentCollectSessionID != nil &&
		*req.Assessment.AssessmentCollectSessionID != uuid.Nil

	mode := "date"
	if hasAnn || hasCol {
		mode = "session"
	}

	tableName := (&model.AssessmentModel{}).TableName()

	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuka transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var row model.AssessmentModel

	if mode == "session" {
		var ann, col *sessRow

		if hasAnn {
			r, er := ctl.fetchSess(c, *req.Assessment.AssessmentAnnounceSessionID)
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					tx.Rollback()
					return helper.JsonError(c, fiber.StatusBadRequest, "Sesi announce tidak ditemukan")
				}
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi announce")
			}
			if r.Deleted != nil || r.SchoolID != mid {
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusForbidden, "Sesi announce bukan milik school Anda / sudah dihapus")
			}
			ann = r
		}

		if hasCol {
			r, er := ctl.fetchSess(c, *req.Assessment.AssessmentCollectSessionID)
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					tx.Rollback()
					return helper.JsonError(c, fiber.StatusBadRequest, "Sesi collect tidak ditemukan")
				}
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi collect")
			}
			if r.Deleted != nil || r.SchoolID != mid {
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusForbidden, "Sesi collect bukan milik school Anda / sudah dihapus")
			}
			col = r
		}

		if ann != nil && col != nil && pickTime(col).Before(pickTime(ann)) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "collect_session harus sama atau setelah announce_session")
		}

		row = req.Assessment.ToModel()
		row.AssessmentSubmissionMode = model.SubmissionModeSession

		if row.AssessmentStartAt == nil && ann != nil {
			if ann.StartsAt != nil {
				row.AssessmentStartAt = ann.StartsAt
			} else if ann.Date != nil {
				row.AssessmentStartAt = ann.Date
			}
		}
		if row.AssessmentDueAt == nil && col != nil {
			if col.StartsAt != nil {
				row.AssessmentDueAt = col.StartsAt
			} else if col.Date != nil {
				row.AssessmentDueAt = col.Date
			}
		}

		row.AssessmentTitle = autofillTitle(row.AssessmentTitle, csstName, func() *string {
			if ann != nil {
				return ann.Title
			}
			return nil
		}())
		row.AssessmentDescription = autofillDesc(row.AssessmentDescription, ann, col)

	} else {
		if req.Assessment.AssessmentStartAt != nil && req.Assessment.AssessmentDueAt != nil &&
			req.Assessment.AssessmentDueAt.Before(*req.Assessment.AssessmentStartAt) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at harus setelah atau sama dengan assessment_start_at")
		}

		row = req.Assessment.ToModel()
		row.AssessmentSubmissionMode = model.SubmissionModeDate
		row.AssessmentTitle = autofillTitle(row.AssessmentTitle, csstName, nil)
	}

	if row.AssessmentTypeID != nil && *row.AssessmentTypeID == uuid.Nil {
		row.AssessmentTypeID = nil
	}

	var (
		assessmentTypeTimePerQuestionSec int
		hasAssessmentTypeTimePerQuestion bool
	)

	if row.AssessmentTypeID != nil && *row.AssessmentTypeID != uuid.Nil {
		var at struct {
			Category           string  `gorm:"column:assessment_type"`
			IsGraded           bool    `gorm:"column:assessment_type_is_graded"`
			AllowLate          bool    `gorm:"column:assessment_type_allow_late_submission"`
			LatePercent        float64 `gorm:"column:assessment_type_late_penalty_percent"`
			AttemptsAllowed    int     `gorm:"column:assessment_type_attempts_allowed"`
			TimePerQuestionSec *int    `gorm:"column:assessment_type_time_per_question_sec"`
		}

		if err := tx.WithContext(c.Context()).
			Table("assessment_types").
			Select(`
				assessment_type,
				assessment_type_is_graded,
				assessment_type_allow_late_submission,
				assessment_type_late_penalty_percent,
				assessment_type_attempts_allowed,
				assessment_type_time_per_question_sec
			`).
			Where("assessment_type_id = ? AND assessment_type_deleted_at IS NULL", *row.AssessmentTypeID).
			Take(&at).Error; err != nil {

			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "assessment_type tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil assessment_type")
		}

		row.AssessmentTypeCategorySnapshot = model.AssessmentTypeCategory(at.Category)
		row.AssessmentTypeIsGradedSnapshot = at.IsGraded
		row.AssessmentAllowLateSubmissionSnapshot = at.AllowLate
		row.AssessmentLatePenaltyPercentSnapshot = at.LatePercent

		if row.AssessmentTotalAttemptsAllowed <= 0 {
			row.AssessmentTotalAttemptsAllowed = at.AttemptsAllowed
		}

		if at.TimePerQuestionSec != nil {
			assessmentTypeTimePerQuestionSec = *at.TimePerQuestionSec
			hasAssessmentTypeTimePerQuestion = true
		}
	} else {
		if row.AssessmentTypeCategorySnapshot == "" {
			row.AssessmentTypeCategorySnapshot = model.AssessmentTypeCategoryTraining
		}
		if row.AssessmentTotalAttemptsAllowed <= 0 {
			row.AssessmentTotalAttemptsAllowed = 1
		}
	}

	// FINAL RULE: passing score selalu dari CSST
	if csstMinPassingScore != nil && *csstMinPassingScore > 0 {
		row.AssessmentMinPassingScoreClassSubjectSnapshot = float64(*csstMinPassingScore)
	} else {
		row.AssessmentMinPassingScoreClassSubjectSnapshot = 0
	}

	// Auto-slug assessment
	{
		var baseSlug string
		if row.AssessmentSlug != nil && strings.TrimSpace(*row.AssessmentSlug) != "" {
			baseSlug = helper.Slugify(*row.AssessmentSlug, 100)
		} else {
			title := strings.TrimSpace(row.AssessmentTitle)
			if title == "" {
				title = "assessment"
			}
			baseSlug = helper.SuggestSlugFromName(title)
		}

		uniqueSlug, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			tx,
			tableName,
			"assessment_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				return q.Where("assessment_school_id = ?", mid).
					Where("assessment_deleted_at IS NULL")
			},
			100,
		)
		if err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal generate slug assessment")
		}
		row.AssessmentSlug = &uniqueSlug
	}

	if err := tx.Create(&row).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat assessment")
	}

	// 🔼 UP: csst_total_assessments (pakai kolom baru + tx)
	if row.AssessmentClassSectionSubjectTeacherID != nil && *row.AssessmentClassSectionSubjectTeacherID != uuid.Nil {
		if err := ctl.csstIncTotalAssessmentsTx(tx, c, mid, *row.AssessmentClassSectionSubjectTeacherID); err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel")
		}
	}

	createdQuizzes := make([]quizModel.QuizModel, 0, len(quizParts))

	for i := range quizParts {
		qm := quizParts[i].ToModel(mid, row.AssessmentID)

		if row.AssessmentTypeID != nil && *row.AssessmentTypeID != uuid.Nil {
			qm.QuizAssessmentTypeID = row.AssessmentTypeID
		}

		if qm.QuizTimeLimitSec == nil && hasAssessmentTypeTimePerQuestion {
			v := assessmentTypeTimePerQuestionSec
			qm.QuizTimeLimitSec = &v
		}

		var baseSlug string
		if qm.QuizSlug != nil && strings.TrimSpace(*qm.QuizSlug) != "" {
			baseSlug = helper.Slugify(*qm.QuizSlug, 160)
		} else {
			title := strings.TrimSpace(qm.QuizTitle)
			if title == "" {
				title = "quiz"
			}
			baseSlug = helper.Slugify(title, 160)
		}

		uniqSlug, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			tx,
			"quizzes",
			"quiz_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				return q.Where("quiz_school_id = ? AND quiz_deleted_at IS NULL", mid)
			},
			160,
		)
		if err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyiapkan slug quiz")
		}
		qm.QuizSlug = &uniqSlug

		if err := tx.Create(qm).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat quiz")
		}

		createdQuizzes = append(createdQuizzes, *qm)
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}

	msg := "Assessment (mode date) berhasil dibuat"
	if mode == "session" {
		msg = "Assessment (mode session) berhasil dibuat"
	}

	return helper.JsonCreated(c, msg, fiber.Map{
		"assessment": dto.FromModelAssesmentWithSchoolTime(c, row),
		"quizzes":    quizDTO.FromModels(createdQuizzes),
	})
}

// PATCH /assessments/:id
func (ctl *AssessmentController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	fmt.Println("==== PATCH /assessments/:id called ====")
	fmt.Printf("RAW BODY: %s\n", string(c.Body()))

	id, err := helper.ParseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
	}

	var req dto.PatchAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	fmt.Printf("PARSED REQ: %+v\n", req)

	req.Normalize()
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	mid, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var existing model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			assessment_id = ?
			AND assessment_school_id = ?
			AND assessment_deleted_at IS NULL
		`, id, mid).
		First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	oldCSSTID := existing.AssessmentClassSectionSubjectTeacherID

	if err := ctl.assertTeacherBelongsToSchool(c, mid, req.AssessmentCreatedByTeacherID); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	tableName := (&model.AssessmentModel{}).TableName()

	refreshPassingScoreSnapshotFromCSST := func(csstID *uuid.UUID) error {
		if csstID == nil || *csstID == uuid.Nil {
			existing.AssessmentMinPassingScoreClassSubjectSnapshot = 0
			return nil
		}

		cs, er := service.ValidateAndCacheCSST(
			ctl.DB.WithContext(c.Context()),
			mid,
			*csstID,
		)
		if er != nil {
			if fe, ok := er.(*fiber.Error); ok {
				return fiber.NewError(fe.Code, fe.Message)
			}
			return fiber.NewError(fiber.StatusBadRequest, er.Error())
		}

		if cs.MinPassingScore != nil && *cs.MinPassingScore > 0 {
			existing.AssessmentMinPassingScoreClassSubjectSnapshot = float64(*cs.MinPassingScore)
		} else {
			existing.AssessmentMinPassingScoreClassSubjectSnapshot = 0
		}
		return nil
	}

	if req.AssessmentClassSectionSubjectTeacherID != nil {
		if *req.AssessmentClassSectionSubjectTeacherID == uuid.Nil {
			existing.AssessmentClassSectionSubjectTeacherID = nil
		} else {
			cs, er := service.ValidateAndCacheCSST(
				ctl.DB.WithContext(c.Context()),
				mid,
				*req.AssessmentClassSectionSubjectTeacherID,
			)
			if er != nil {
				if fe, ok := er.(*fiber.Error); ok {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
			}

			existing.AssessmentClassSectionSubjectTeacherID = req.AssessmentClassSectionSubjectTeacherID

			if req.AssessmentCreatedByTeacherID == nil &&
				existing.AssessmentCreatedByTeacherID == nil &&
				cs.TeacherID != nil && *cs.TeacherID != uuid.Nil {

				tid := *cs.TeacherID
				existing.AssessmentCreatedByTeacherID = &tid
			}
		}
	}

	finalAnnID := existing.AssessmentAnnounceSessionID
	if req.AssessmentAnnounceSessionID != nil {
		if *req.AssessmentAnnounceSessionID == uuid.Nil {
			finalAnnID = nil
		} else {
			v := *req.AssessmentAnnounceSessionID
			finalAnnID = &v
		}
	}
	finalColID := existing.AssessmentCollectSessionID
	if req.AssessmentCollectSessionID != nil {
		if *req.AssessmentCollectSessionID == uuid.Nil {
			finalColID = nil
		} else {
			v := *req.AssessmentCollectSessionID
			finalColID = &v
		}
	}

	nextMode := "date"
	if finalAnnID != nil || finalColID != nil {
		nextMode = "session"
	}

	// =========================
	// MODE: session
	// =========================
	if nextMode == "session" {
		var ann, col *sessRow

		if finalAnnID != nil {
			r, er := ctl.fetchSess(c, *finalAnnID)
			if er != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Sesi announce tidak ditemukan")
			}
			if r.Deleted != nil || r.SchoolID != mid {
				return helper.JsonError(c, fiber.StatusForbidden, "Sesi announce bukan milik school Anda / sudah dihapus")
			}
			ann = r
		}

		if finalColID != nil {
			r, er := ctl.fetchSess(c, *finalColID)
			if er != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Sesi collect tidak ditemukan")
			}
			if r.Deleted != nil || r.SchoolID != mid {
				return helper.JsonError(c, fiber.StatusForbidden, "Sesi collect bukan milik school Anda / sudah dihapus")
			}
			col = r
		}

		if ann != nil && col != nil && pickTime(col).Before(pickTime(ann)) {
			return helper.JsonError(c, fiber.StatusBadRequest, "collect_session harus sama atau setelah announce_session")
		}

		req.Apply(&existing)

		if existing.AssessmentTypeID != nil && *existing.AssessmentTypeID == uuid.Nil {
			existing.AssessmentTypeID = nil
		}

		existing.AssessmentSubmissionMode = model.SubmissionModeSession
		existing.AssessmentAnnounceSessionID = finalAnnID
		existing.AssessmentCollectSessionID = finalColID

		if req.AssessmentStartAt == nil && existing.AssessmentStartAt == nil && ann != nil {
			if ann.StartsAt != nil {
				existing.AssessmentStartAt = ann.StartsAt
			} else if ann.Date != nil {
				existing.AssessmentStartAt = ann.Date
			}
		}
		if req.AssessmentDueAt == nil && existing.AssessmentDueAt == nil && col != nil {
			if col.StartsAt != nil {
				existing.AssessmentDueAt = col.StartsAt
			} else if col.Date != nil {
				existing.AssessmentDueAt = col.Date
			}
		}

		if er := refreshPassingScoreSnapshotFromCSST(existing.AssessmentClassSectionSubjectTeacherID); er != nil {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
		}

		var baseSlug string
		if req.AssessmentSlug != nil {
			slugIn := strings.TrimSpace(*req.AssessmentSlug)
			if slugIn == "" {
				title := strings.TrimSpace(existing.AssessmentTitle)
				if title == "" {
					title = "assessment"
				}
				baseSlug = helper.SuggestSlugFromName(title)
			} else {
				baseSlug = helper.Slugify(slugIn, 100)
			}
		} else {
			if existing.AssessmentSlug != nil && strings.TrimSpace(*existing.AssessmentSlug) != "" {
				baseSlug = helper.Slugify(*existing.AssessmentSlug, 100)
			} else {
				title := strings.TrimSpace(existing.AssessmentTitle)
				if title == "" {
					title = "assessment"
				}
				baseSlug = helper.SuggestSlugFromName(title)
			}
		}

		uniqueSlug, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			ctl.DB,
			tableName,
			"assessment_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				return q.Where("assessment_school_id = ?", mid).
					Where("assessment_deleted_at IS NULL").
					Where("assessment_id <> ?", id)
			},
			100,
		)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal generate slug assessment")
		}
		existing.AssessmentSlug = &uniqueSlug

		// ✅ TX khusus untuk: save assessment + down/up csst
		tx := ctl.DB.WithContext(c.Context()).Begin()
		if tx.Error != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuka transaksi")
		}
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		existing.AssessmentUpdatedAt = time.Now()
		if err := tx.WithContext(c.Context()).Save(&existing).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui assessment")
		}

		newCSSTID := existing.AssessmentClassSectionSubjectTeacherID
		if !uuidPtrEqual(oldCSSTID, newCSSTID) {
			if oldCSSTID != nil && *oldCSSTID != uuid.Nil {
				if err := ctl.csstDecTotalAssessmentsTx(tx, c, mid, *oldCSSTID); err != nil {
					tx.Rollback()
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel (csst lama)")
				}
			}
			if newCSSTID != nil && *newCSSTID != uuid.Nil {
				if err := ctl.csstIncTotalAssessmentsTx(tx, c, mid, *newCSSTID); err != nil {
					tx.Rollback()
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel (csst baru)")
				}
			}
		}

		if err := tx.Commit().Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
		}

		return helper.JsonUpdated(
			c,
			"Assessment (mode session) diperbarui",
			dto.FromModelAssesmentWithSchoolTime(c, existing),
		)
	}

	// =========================
	// MODE: date
	// =========================
	switch {
	case req.AssessmentStartAt != nil && req.AssessmentDueAt != nil:
		if req.AssessmentDueAt.Before(*req.AssessmentStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at harus setelah atau sama dengan assessment_start_at")
		}
	case req.AssessmentStartAt != nil && req.AssessmentDueAt == nil:
		if existing.AssessmentDueAt != nil && existing.AssessmentDueAt.Before(*req.AssessmentStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Tanggal due saat ini lebih awal dari start baru")
		}
	case req.AssessmentStartAt == nil && req.AssessmentDueAt != nil:
		if existing.AssessmentStartAt != nil && req.AssessmentDueAt.Before(*existing.AssessmentStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at tidak boleh sebelum assessment_start_at")
		}
	}

	req.Apply(&existing)
	existing.AssessmentSubmissionMode = model.SubmissionModeDate

	if existing.AssessmentTypeID != nil && *existing.AssessmentTypeID == uuid.Nil {
		existing.AssessmentTypeID = nil
	}

	existing.AssessmentAnnounceSessionID = finalAnnID
	existing.AssessmentCollectSessionID = finalColID

	if er := refreshPassingScoreSnapshotFromCSST(existing.AssessmentClassSectionSubjectTeacherID); er != nil {
		if fe, ok := er.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
	}

	{
		var baseSlug string
		if req.AssessmentSlug != nil {
			slugIn := strings.TrimSpace(*req.AssessmentSlug)
			if slugIn == "" {
				title := strings.TrimSpace(existing.AssessmentTitle)
				if title == "" {
					title = "assessment"
				}
				baseSlug = helper.SuggestSlugFromName(title)
			} else {
				baseSlug = helper.Slugify(slugIn, 100)
			}
		} else {
			if existing.AssessmentSlug != nil && strings.TrimSpace(*existing.AssessmentSlug) != "" {
				baseSlug = helper.Slugify(*existing.AssessmentSlug, 100)
			} else {
				title := strings.TrimSpace(existing.AssessmentTitle)
				if title == "" {
					title = "assessment"
				}
				baseSlug = helper.SuggestSlugFromName(title)
			}
		}

		uniqueSlug, err := helper.EnsureUniqueSlugCI(
			c.Context(),
			ctl.DB,
			tableName,
			"assessment_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				return q.Where("assessment_school_id = ?", mid).
					Where("assessment_deleted_at IS NULL").
					Where("assessment_id <> ?", id)
			},
			100,
		)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal generate slug assessment")
		}
		existing.AssessmentSlug = &uniqueSlug
	}

	// ✅ TX: save assessment + down/up csst
	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuka transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	existing.AssessmentUpdatedAt = time.Now()
	if err := tx.WithContext(c.Context()).Save(&existing).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui assessment")
	}

	newCSSTID := existing.AssessmentClassSectionSubjectTeacherID
	if !uuidPtrEqual(oldCSSTID, newCSSTID) {
		if oldCSSTID != nil && *oldCSSTID != uuid.Nil {
			if err := ctl.csstDecTotalAssessmentsTx(tx, c, mid, *oldCSSTID); err != nil {
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel (csst lama)")
			}
		}
		if newCSSTID != nil && *newCSSTID != uuid.Nil {
			if err := ctl.csstIncTotalAssessmentsTx(tx, c, mid, *newCSSTID); err != nil {
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel (csst baru)")
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}

	return helper.JsonUpdated(
		c,
		"Assessment (mode date) diperbarui",
		dto.FromModelAssesmentWithSchoolTime(c, existing),
	)
}

// DELETE /assessments/:id (soft delete)
func (ctl *AssessmentController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	id, err := helper.ParseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
	}

	mid, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var usedCount int64
	if err := ctl.DB.WithContext(c.Context()).
		Model(&submissionsModel.SubmissionModel{}).
		Where(`
			submission_school_id = ?
			AND submission_assessment_id = ?
			AND submission_deleted_at IS NULL
		`, mid, id).
		Count(&usedCount).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek relasi submissions")
	}

	if usedCount > 0 {
		return helper.JsonError(
			c,
			fiber.StatusBadRequest,
			"Tidak dapat menghapus assessment karena masih memiliki submission siswa.",
		)
	}

	var row model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			assessment_id = ?
			AND assessment_school_id = ?
			AND assessment_deleted_at IS NULL
		`, id, mid).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuka transaksi penghapusan")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.WithContext(c.Context()).Delete(&row).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus assessment")
	}

	// 🔽 DOWN: csst_total_assessments (pakai kolom baru + tx)
	if row.AssessmentClassSectionSubjectTeacherID != nil && *row.AssessmentClassSectionSubjectTeacherID != uuid.Nil {
		if err := ctl.csstDecTotalAssessmentsTx(tx, c, mid, *row.AssessmentClassSectionSubjectTeacherID); err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit penghapusan assessment")
	}

	return helper.JsonDeleted(c, "Assessment dihapus", fiber.Map{
		"assessment_id": id,
	})
}

/* ========================================================
   Helpers (local)
======================================================== */

func uuidPtrEqual(a, b *uuid.UUID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func atoiOr(def int, s string) int {
	if s == "" {
		return def
	}
	n := 0
	sign := 1
	for i := 0; i < len(s); i++ {
		if i == 0 && s[i] == '-' {
			sign = -1
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return def
		}
		n = n*10 + int(s[i]-'0')
	}
	n *= sign
	if n <= 0 {
		return def
	}
	return n
}

func eqTrue(s string) bool {
	v := strings.TrimSpace(strings.ToLower(s))
	return v == "1" || v == "true"
}
