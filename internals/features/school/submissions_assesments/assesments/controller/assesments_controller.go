// file: internals/features/school/assessments/controller/assessment_controller.go
package controller

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	"madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/snapshot"
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

func makeSessionSnap(s *sessRow) datatypes.JSONMap {
	if s == nil {
		return datatypes.JSONMap{}
	}
	m := map[string]any{
		"captured_at": time.Now().UTC(),
		"session_id":  s.ID,
	}
	if s.StartsAt != nil {
		m["starts_at"] = s.StartsAt.UTC()
	}
	if s.Date != nil {
		m["date"] = s.Date.UTC()
	}
	if s.Title != nil && strings.TrimSpace(*s.Title) != "" {
		m["title"] = strings.TrimSpace(*s.Title)
	}
	return datatypes.JSONMap(m)
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
   Helpers untuk AssessmentType snapshot (baru, tanpa JSON)
======================================================== */

func resetAssessmentTypeSnapshotFields(a *model.AssessmentModel) {
	a.AssessmentTypeIsGradedSnapshot = false

	a.AssessmentShuffleQuestionsSnapshot = false
	a.AssessmentShuffleOptionsSnapshot = false
	a.AssessmentShowCorrectAfterSubmitSnapshot = false
	a.AssessmentStrictModeSnapshot = false
	a.AssessmentTimeLimitMinSnapshot = nil
	a.AssessmentAttemptsAllowedSnapshot = 0
	a.AssessmentRequireLoginSnapshot = false
	a.AssessmentScoreAggregationModeSnapshot = ""

	a.AssessmentAllowLateSubmissionSnapshot = false
	a.AssessmentLatePenaltyPercentSnapshot = 0
	a.AssessmentPassingScorePercentSnapshot = 0
	a.AssessmentShowScoreAfterSubmitSnapshot = false
	a.AssessmentShowCorrectAfterClosedSnapshot = false
	a.AssessmentAllowReviewBeforeSubmitSnapshot = false
	a.AssessmentRequireCompleteAttemptSnapshot = false
	a.AssessmentShowDetailsAfterAllAttemptsSnapshot = false
}

func (ctl *AssessmentController) hydrateAssessmentTypeSnapshot(
	c *fiber.Ctx,
	schoolID uuid.UUID,
	a *model.AssessmentModel,
	typeID uuid.UUID,
) error {
	var at model.AssessmentTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			assessment_type_id = ?
			AND assessment_type_school_id = ?
			AND assessment_type_deleted_at IS NULL
		`, typeID, schoolID).
		Take(&at).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "Tipe assessment tidak ditemukan")
		}
		return err
	}

	a.AssessmentTypeID = &typeID
	a.AssessmentTypeIsGradedSnapshot = at.AssessmentTypeIsGraded

	a.AssessmentShuffleQuestionsSnapshot = at.AssessmentTypeShuffleQuestions
	a.AssessmentShuffleOptionsSnapshot = at.AssessmentTypeShuffleOptions
	a.AssessmentShowCorrectAfterSubmitSnapshot = at.AssessmentTypeShowCorrectAfterSubmit
	a.AssessmentStrictModeSnapshot = at.AssessmentTypeStrictMode
	a.AssessmentTimeLimitMinSnapshot = at.AssessmentTypeTimeLimitMin
	a.AssessmentAttemptsAllowedSnapshot = at.AssessmentTypeAttemptsAllowed
	a.AssessmentRequireLoginSnapshot = at.AssessmentTypeRequireLogin
	a.AssessmentScoreAggregationModeSnapshot = at.AssessmentTypeScoreAggregationMode

	a.AssessmentAllowLateSubmissionSnapshot = at.AssessmentTypeAllowLateSubmission
	a.AssessmentLatePenaltyPercentSnapshot = at.AssessmentTypeLatePenaltyPercent
	a.AssessmentPassingScorePercentSnapshot = at.AssessmentTypePassingScorePercent
	a.AssessmentShowScoreAfterSubmitSnapshot = at.AssessmentTypeShowScoreAfterSubmit
	a.AssessmentShowCorrectAfterClosedSnapshot = at.AssessmentTypeShowCorrectAfterClosed
	a.AssessmentAllowReviewBeforeSubmitSnapshot = at.AssessmentTypeAllowReviewBeforeSubmit
	a.AssessmentRequireCompleteAttemptSnapshot = at.AssessmentTypeRequireCompleteAttempt
	a.AssessmentShowDetailsAfterAllAttemptsSnapshot = at.AssessmentTypeShowDetailsAfterAllAttempts

	return nil
}

/* ===============================
   Handlers
=============================== */

// POST /assessments
// Body: CreateAssessmentWithQuizzesRequest
// - "assessment": data assessment
// - "quiz": 1 quiz, atau
// - "quizzes": array quiz (kalau >1)
func (ctl *AssessmentController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	var req dto.CreateAssessmentWithQuizzesRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()

	// Flatten quiz: pakai "quizzes" kalau ada; else "quiz" tunggal
	quizParts := req.FlattenQuizzes()
	if len(quizParts) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Minimal harus ada 1 quiz")
	}

	// 🔒 resolve & authorize (DKM/Admin atau Teacher) DARI TOKEN
	mid, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	// Enforce tenant di assessment
	req.Assessment.AssessmentSchoolID = mid

	// ====== Auto isi assessment_quiz_total kalau belum diisi di payload ======
	if req.Assessment.AssessmentQuizTotal == nil || *req.Assessment.AssessmentQuizTotal <= 0 {
		qt := len(quizParts)
		req.Assessment.AssessmentQuizTotal = &qt
	}

	// DTO validation
	if err := ctl.Validator.Struct(&req.Assessment); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	for i := range quizParts {
		if err := ctl.Validator.Struct(&quizParts[i]); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// =============== CSST SNAPSHOT (opsional, +auto teacher & auto-title) ===============
	var csstSnap datatypes.JSONMap
	var csstName *string

	if req.Assessment.AssessmentClassSectionSubjectTeacherID != nil &&
		*req.Assessment.AssessmentClassSectionSubjectTeacherID != uuid.Nil {

		cs, er := snapshot.ValidateAndSnapshotCSST(
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

		// simpan csst name utk autofill judul
		csstName = cs.Name

		// Auto-isi created_by_teacher_id kalau belum ada
		if req.Assessment.AssessmentCreatedByTeacherID == nil &&
			cs.TeacherID != nil && *cs.TeacherID != uuid.Nil {
			tid := *cs.TeacherID
			req.Assessment.AssessmentCreatedByTeacherID = &tid
		}

		// Simpan snapshot FULL (sama persis dengan yg dipakai attendance session)
		if jb := snapshot.ToJSON(cs); len(jb) > 0 {
			var m map[string]any
			_ = json.Unmarshal(jb, &m)
			csstSnap = datatypes.JSONMap(m)
		}
	}

	// Validasi creator teacher (opsional)
	if err := ctl.assertTeacherBelongsToSchool(
		c,
		mid,
		req.Assessment.AssessmentCreatedByTeacherID,
	); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ====== Tentukan mode dari presence sesi ======
	hasAnn := req.Assessment.AssessmentAnnounceSessionID != nil &&
		*req.Assessment.AssessmentAnnounceSessionID != uuid.Nil
	hasCol := req.Assessment.AssessmentCollectSessionID != nil &&
		*req.Assessment.AssessmentCollectSessionID != uuid.Nil
	mode := "date"
	if hasAnn || hasCol {
		mode = "session"
	}

	tableName := (&model.AssessmentModel{}).TableName()

	// Mulai transaksi
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

	// ====== MODE: session ======
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

		// Turunkan start/due dari sesi bila belum diisi
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

		// ===== auto-title & auto-desc jika kosong =====
		row.AssessmentTitle = autofillTitle(row.AssessmentTitle, csstName, func() *string {
			if ann != nil {
				return ann.Title
			}
			return nil
		}())
		row.AssessmentDescription = autofillDesc(row.AssessmentDescription, ann, col)

		// Snapshots
		if csstSnap != nil {
			row.AssessmentCSSTSnapshot = csstSnap
		}
		if ann != nil {
			row.AssessmentAnnounceSessionSnapshot = makeSessionSnap(ann)
		}
		if col != nil {
			row.AssessmentCollectSessionSnapshot = makeSessionSnap(col)
		}

	} else {
		// ===== MODE: date =====
		if req.Assessment.AssessmentStartAt != nil && req.Assessment.AssessmentDueAt != nil &&
			req.Assessment.AssessmentDueAt.Before(*req.Assessment.AssessmentStartAt) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at harus setelah atau sama dengan assessment_start_at")
		}

		row = req.Assessment.ToModel()
		row.AssessmentSubmissionMode = model.SubmissionModeDate

		// Auto-title (pakai csstName bila title kosong)
		row.AssessmentTitle = autofillTitle(row.AssessmentTitle, csstName, nil)

		// Snapshot CSST bila ada
		if csstSnap != nil {
			row.AssessmentCSSTSnapshot = csstSnap
		}
	}

	// ==== Sync assessment type snapshot (tanpa JSON) ====
	if row.AssessmentTypeID != nil && *row.AssessmentTypeID != uuid.Nil {
		if err := ctl.hydrateAssessmentTypeSnapshot(c, mid, &row, *row.AssessmentTypeID); err != nil {
			tx.Rollback()
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
	} else {
		row.AssessmentTypeID = nil
		resetAssessmentTypeSnapshotFields(&row)
	}

	// ===== Auto-slug assessment (unik per school) =====
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

	// Simpan assessment dulu
	if err := tx.Create(&row).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat assessment")
	}

	// 🔼 UP: tambah total_assessments (+ graded / ungraded) di CSST kalau ada relasi
	if row.AssessmentClassSectionSubjectTeacherID != nil && *row.AssessmentClassSectionSubjectTeacherID != uuid.Nil {
		isGraded := row.AssessmentTypeIsGradedSnapshot // bool

		updates := map[string]any{
			"class_section_subject_teacher_total_assessments": gorm.Expr(
				"class_section_subject_teacher_total_assessments + 1",
			),
		}

		if isGraded {
			updates["class_section_subject_teacher_total_assessments_graded"] = gorm.Expr(
				"class_section_subject_teacher_total_assessments_graded + 1",
			)
		} else {
			updates["class_section_subject_teacher_total_assessments_ungraded"] = gorm.Expr(
				"class_section_subject_teacher_total_assessments_ungraded + 1",
			)
		}

		if err := tx.WithContext(c.Context()).
			Model(&csstModel.ClassSectionSubjectTeacherModel{}).
			Where(`
				class_section_subject_teacher_school_id = ?
				AND class_section_subject_teacher_id = ?
				AND class_section_subject_teacher_deleted_at IS NULL
			`, mid, *row.AssessmentClassSectionSubjectTeacherID).
			Updates(updates).Error; err != nil {

			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel")
		}
	}

	// ==============================
	// 2) Buat semua quiz (1..N)
	// ==============================
	createdQuizzes := make([]quizModel.QuizModel, 0, len(quizParts))

	for i := range quizParts {
		qm := quizParts[i].ToModel(mid, row.AssessmentID)

		// Slug quiz (unik per school; fallback "quiz")
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

	// Commit transaksi
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}

	// Response: assessment + quizzes
	msg := "Assessment (mode date) berhasil dibuat"
	if mode == "session" {
		msg = "Assessment (mode session) berhasil dibuat"
	}

	return helper.JsonCreated(c, msg, fiber.Map{
		"assessment": dto.FromModelAssesment(row),
		"quizzes":    quizDTO.FromModels(createdQuizzes),
	})
}

// PATCH /assessments/:id
func (ctl *AssessmentController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	id, err := helper.ParseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
	}

	var req dto.PatchAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔒 resolve & authorize (DKM/Admin atau Teacher) DARI TOKEN
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

	// Simpan CSST lama untuk hitung UP/DOWN
	oldCSSTID := existing.AssessmentClassSectionSubjectTeacherID

	// validasi guru bila diubah
	if err := ctl.assertTeacherBelongsToSchool(c, mid, req.AssessmentCreatedByTeacherID); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	tableName := (&model.AssessmentModel{}).TableName()

	// ==== (Opsional) update CSST snapshot bila CSST diubah ====
	if req.AssessmentClassSectionSubjectTeacherID != nil {
		if *req.AssessmentClassSectionSubjectTeacherID == uuid.Nil {
			// clear relasi CSST
			existing.AssessmentClassSectionSubjectTeacherID = nil
			existing.AssessmentCSSTSnapshot = datatypes.JSONMap{}
		} else {
			cs, er := snapshot.ValidateAndSnapshotCSST(
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

			// Simpan snapshot FULL (sama seperti Create)
			if jb := snapshot.ToJSON(cs); len(jb) > 0 {
				var m map[string]any
				_ = json.Unmarshal(jb, &m)
				existing.AssessmentCSSTSnapshot = datatypes.JSONMap(m)
			}

			// Auto-isi created_by_teacher_id kalau:
			// - user TIDAK mengirim AssessmentCreatedByTeacherID di PATCH
			// - existing belum punya creator
			if req.AssessmentCreatedByTeacherID == nil &&
				existing.AssessmentCreatedByTeacherID == nil &&
				cs.TeacherID != nil && *cs.TeacherID != uuid.Nil {

				tid := *cs.TeacherID
				existing.AssessmentCreatedByTeacherID = &tid
			}
		}
	}

	// ==== Hitung FINAL session IDs (memperhatikan niat PATCH untuk clear) ====
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

	// Tentukan mode hasil
	nextMode := "date"
	if finalAnnID != nil || finalColID != nil {
		nextMode = "session"
	}

	// ===== MODE: session =====
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

		// Terapkan PATCH ke existing (tanpa simpan dulu)
		req.Apply(&existing)

		// Set mode & session ids final
		existing.AssessmentSubmissionMode = model.SubmissionModeSession
		existing.AssessmentAnnounceSessionID = finalAnnID
		existing.AssessmentCollectSessionID = finalColID

		// Jika user TIDAK kirim start/due di PATCH, turunkan dari sesi (agar konsisten)
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

		// Snapshots sesi
		if ann != nil {
			existing.AssessmentAnnounceSessionSnapshot = makeSessionSnap(ann)
		} else {
			existing.AssessmentAnnounceSessionSnapshot = datatypes.JSONMap{}
		}
		if col != nil {
			existing.AssessmentCollectSessionSnapshot = makeSessionSnap(col)
		} else {
			existing.AssessmentCollectSessionSnapshot = datatypes.JSONMap{}
		}

		// ==== Update snapshot tipe assessment bila perlu ====
		if err := ctl.applyAssessmentTypePatch(c, mid, &existing, &req); err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		// ===== Auto-slug (jaga unik, exclude diri sendiri) =====
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

		existing.AssessmentUpdatedAt = time.Now()
		if err := ctl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui assessment")
		}

		// 🔁 DOWN/UP kalau CSST berubah
		newCSSTID := existing.AssessmentClassSectionSubjectTeacherID
		if !uuidPtrEqual(oldCSSTID, newCSSTID) {
			// CSST lama: -1
			if oldCSSTID != nil && *oldCSSTID != uuid.Nil {
				if err := ctl.DB.WithContext(c.Context()).
					Model(&csstModel.ClassSectionSubjectTeacherModel{}).
					Where(`
						class_section_subject_teacher_school_id = ?
						AND class_section_subject_teacher_id = ?
						AND class_section_subject_teacher_deleted_at IS NULL
					`, mid, *oldCSSTID).
					Update(
						"class_section_subject_teacher_total_assessments",
						gorm.Expr("CASE WHEN class_section_subject_teacher_total_assessments > 0 THEN class_section_subject_teacher_total_assessments - 1 ELSE 0 END"),
					).Error; err != nil {

					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel (csst lama)")
				}
			}
			// CSST baru: +1
			if newCSSTID != nil && *newCSSTID != uuid.Nil {
				if err := ctl.DB.WithContext(c.Context()).
					Model(&csstModel.ClassSectionSubjectTeacherModel{}).
					Where(`
						class_section_subject_teacher_school_id = ?
						AND class_section_subject_teacher_id = ?
						AND class_section_subject_teacher_deleted_at IS NULL
					`, mid, *newCSSTID).
					Update(
						"class_section_subject_teacher_total_assessments",
						gorm.Expr("class_section_subject_teacher_total_assessments + 1"),
					).Error; err != nil {

					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel (csst baru)")
				}
			}
		}

		return helper.JsonUpdated(c, "Assessment (mode session) diperbarui", dto.FromModelAssesment(existing))
	}

	// ===== MODE: date =====
	// Validasi kombinasi waktu
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

	// Terapkan PATCH & set mode/date
	req.Apply(&existing)
	existing.AssessmentSubmissionMode = model.SubmissionModeDate

	// Jika pindah ke date → bersihkan session IDs & snapshots bila user clear
	existing.AssessmentAnnounceSessionID = finalAnnID // akan nil jika user kirim UUID nil
	existing.AssessmentCollectSessionID = finalColID
	if finalAnnID == nil {
		existing.AssessmentAnnounceSessionSnapshot = datatypes.JSONMap{}
	}
	if finalColID == nil {
		existing.AssessmentCollectSessionSnapshot = datatypes.JSONMap{}
	}

	// ==== Update snapshot tipe assessment bila perlu ====
	if err := ctl.applyAssessmentTypePatch(c, mid, &existing, &req); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ===== Auto-slug (jaga unik, exclude diri sendiri) =====
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

	existing.AssessmentUpdatedAt = time.Now()
	if err := ctl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui assessment")
	}

	// 🔁 DOWN/UP kalau CSST berubah
	newCSSTID := existing.AssessmentClassSectionSubjectTeacherID
	if !uuidPtrEqual(oldCSSTID, newCSSTID) {
		// CSST lama: -1
		if oldCSSTID != nil && *oldCSSTID != uuid.Nil {
			if err := ctl.DB.WithContext(c.Context()).
				Model(&csstModel.ClassSectionSubjectTeacherModel{}).
				Where(`
					class_section_subject_teacher_school_id = ?
					AND class_section_subject_teacher_id = ?
					AND class_section_subject_teacher_deleted_at IS NULL
				`, mid, *oldCSSTID).
				Update(
					"class_section_subject_teacher_total_assessments",
					gorm.Expr("CASE WHEN class_section_subject_teacher_total_assessments > 0 THEN class_section_subject_teacher_total_assessments - 1 ELSE 0 END"),
				).Error; err != nil {

				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel (csst lama)")
			}
		}
		// CSST baru: +1
		if newCSSTID != nil && *newCSSTID != uuid.Nil {
			if err := ctl.DB.WithContext(c.Context()).
				Model(&csstModel.ClassSectionSubjectTeacherModel{}).
				Where(`
					class_section_subject_teacher_school_id = ?
					AND class_section_subject_teacher_id = ?
					AND class_section_subject_teacher_deleted_at IS NULL
				`, mid, *newCSSTID).
				Update(
					"class_section_subject_teacher_total_assessments",
					gorm.Expr("class_section_subject_teacher_total_assessments + 1"),
				).Error; err != nil {

				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate total assessment mapel (csst baru)")
			}
		}
	}

	return helper.JsonUpdated(c, "Assessment (mode date) diperbarui", dto.FromModelAssesment(existing))
}

/* ========================================================
   Helpers tambahan untuk Assessment Type Snapshot (PATCH)
======================================================== */

func (ctl *AssessmentController) applyAssessmentTypePatch(
	c *fiber.Ctx,
	schoolID uuid.UUID,
	existing *model.AssessmentModel,
	req *dto.PatchAssessmentRequest,
) error {
	// Kalau field tidak dikirim → jangan apa-apa
	if req.AssessmentTypeID == nil {
		return nil
	}

	// Kalau dikirim UUID nil → clear type + semua snapshot
	if *req.AssessmentTypeID == uuid.Nil {
		existing.AssessmentTypeID = nil
		resetAssessmentTypeSnapshotFields(existing)
		return nil
	}

	// Else: load type & set snapshot scalar
	return ctl.hydrateAssessmentTypeSnapshot(c, schoolID, existing, *req.AssessmentTypeID)
}

// DELETE /assessments/:id (soft delete)
func (ctl *AssessmentController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	id, err := helper.ParseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
	}

	// 🔒 resolve & authorize (DKM/Admin atau Teacher) DARI TOKEN
	mid, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔒 GUARD: cek apakah assessment masih dipakai oleh submissions
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

	// 🔎 Ambil row assessment yang masih hidup
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

	// 🗑 Soft delete + DOWN agregat dalam transaksi
	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuka transaksi penghapusan")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Soft delete assessment
	if err := tx.WithContext(c.Context()).Delete(&row).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus assessment")
	}

	// 🔽 DOWN: turunkan total_assessments (+ graded / ungraded) di CSST jika ada relasi
	if row.AssessmentClassSectionSubjectTeacherID != nil && *row.AssessmentClassSectionSubjectTeacherID != uuid.Nil {
		isGraded := row.AssessmentTypeIsGradedSnapshot

		updates := map[string]any{
			"class_section_subject_teacher_total_assessments": gorm.Expr(
				"CASE WHEN class_section_subject_teacher_total_assessments > 0 " +
					"THEN class_section_subject_teacher_total_assessments - 1 ELSE 0 END",
			),
		}

		if isGraded {
			updates["class_section_subject_teacher_total_assessments_graded"] = gorm.Expr(
				"CASE WHEN class_section_subject_teacher_total_assessments_graded > 0 " +
					"THEN class_section_subject_teacher_total_assessments_graded - 1 ELSE 0 END",
			)
		} else {
			updates["class_section_subject_teacher_total_assessments_ungraded"] = gorm.Expr(
				"CASE WHEN class_section_subject_teacher_total_assessments_ungraded > 0 " +
					"THEN class_section_subject_teacher_total_assessments_ungraded - 1 ELSE 0 END",
			)
		}

		if err := tx.WithContext(c.Context()).
			Model(&csstModel.ClassSectionSubjectTeacherModel{}).
			Where(`
			class_section_subject_teacher_school_id = ?
			AND class_section_subject_teacher_id = ?
			AND class_section_subject_teacher_deleted_at IS NULL
		`, mid, *row.AssessmentClassSectionSubjectTeacherID).
			Updates(updates).Error; err != nil {

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
