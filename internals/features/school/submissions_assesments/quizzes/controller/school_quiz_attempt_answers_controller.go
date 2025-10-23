// file: internals/features/school/submissions_assesments/quizzes/controller/student_quiz_attempt_answer_controller.go
package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	qdto "masjidku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ============================================================
   Controller
============================================================ */

type StudentQuizAttemptAnswersController struct {
	DB *gorm.DB
	V  *validator.Validate
}

func NewStudentQuizAttemptAnswersController(db *gorm.DB) *StudentQuizAttemptAnswersController {
	return &StudentQuizAttemptAnswersController{
		DB: db,
		V:  validator.New(),
	}
}

func (ctl *StudentQuizAttemptAnswersController) ensureValidator() {
	if ctl.V == nil {
		ctl.V = validator.New()
	}
}

/* ============================================================
   Helpers — scope & relasi
============================================================ */

type attemptCore struct {
	MasjidID  uuid.UUID `gorm:"column:student_quiz_attempt_masjid_id"`
	StudentID uuid.UUID `gorm:"column:student_quiz_attempt_student_id"`
	QuizID    uuid.UUID `gorm:"column:student_quiz_attempt_quiz_id"`
}

// Load minimal kolom attempt utk cek scope
func (ctl *StudentQuizAttemptAnswersController) loadAttemptCore(id uuid.UUID) (*attemptCore, error) {
	if id == uuid.Nil {
		return nil, fiber.NewError(http.StatusBadRequest, "attempt_id wajib")
	}

	var core attemptCore
	err := ctl.DB.
		Table("student_quiz_attempts").
		Select("student_quiz_attempt_masjid_id, student_quiz_attempt_student_id, student_quiz_attempt_quiz_id").
		Where("student_quiz_attempt_id = ?", id).
		Take(&core).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(http.StatusNotFound, "attempt tidak ditemukan")
		}
		return nil, err
	}
	if core.QuizID == uuid.Nil {
		return nil, fiber.NewError(http.StatusConflict, "attempt belum terhubung ke quiz")
	}
	return &core, nil
}

// Pastikan question_id milik quiz yg sama dgn attempt & belum deleted
func (ctl *StudentQuizAttemptAnswersController) questionBelongsToQuiz(questionID, quizID uuid.UUID) (bool, error) {
	if questionID == uuid.Nil || quizID == uuid.Nil {
		return false, fiber.NewError(http.StatusBadRequest, "question_id/quiz_id wajib")
	}
	var ok bool
	if err := ctl.DB.
		Raw(`
			SELECT EXISTS(
				SELECT 1
				FROM quiz_questions
				WHERE quiz_question_id = ?
				  AND quiz_question_quiz_id = ?
				  AND quiz_question_deleted_at IS NULL
			)`, questionID, quizID).
		Scan(&ok).Error; err != nil {
		return false, err
	}
	return ok, nil
}

// Scope enforcement:
// - Student: harus terdaftar sbg student di masjid attempt & student_id harus cocok
// - Non-student: Owner diizinkan; selain itu wajib DKM/Teacher di masjid attempt
func (ctl *StudentQuizAttemptAnswersController) ensureScopeForAttempt(c *fiber.Ctx, core *attemptCore) error {
	if core == nil {
		return fiber.NewError(http.StatusInternalServerError, "internal: attemptCore nil")
	}

	// Student flow
	if helperAuth.IsStudent(c) {
		if err := helperAuth.EnsureStudentMasjid(c, core.MasjidID); err != nil {
			return err
		}
		sid, err := helperAuth.GetMasjidStudentIDForMasjid(c, core.MasjidID)
		if err != nil {
			return fiber.NewError(http.StatusForbidden, "tidak terdaftar sebagai student pada masjid attempt")
		}
		if sid != core.StudentID {
			return fiber.NewError(http.StatusForbidden, "attempt bukan milik kamu")
		}
		return nil
	}

	// Non-student:
	if helperAuth.IsOwner(c) {
		return nil
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, core.MasjidID); err != nil {
		return err
	}
	return nil
}

/* ============================================================
   Tiny helpers
============================================================ */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	s := strings.TrimSpace(c.Params(name))
	return uuid.Parse(s)
}

func validationMessage(err error) string {
	return "validasi gagal: " + err.Error()
}

/* ============================================================
   Handlers
============================================================ */

// GET /student-quiz-attempt-answers?attempt_id=...&question_id=...&sort_by=&order=&page=&per_page=
func (ctl *StudentQuizAttemptAnswersController) List(c *fiber.Ctx) error {
	// attempt_id wajib (efisien & scope)
	attemptIDStr := strings.TrimSpace(c.Query("attempt_id"))
	if attemptIDStr == "" {
		return helper.JsonError(c, http.StatusBadRequest, "attempt_id wajib diisi")
	}
	attemptID, err := uuid.Parse(attemptIDStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "attempt_id tidak valid")
	}

	core, err := ctl.loadAttemptCore(attemptID)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal memuat attempt")
	}
	if err := ctl.ensureScopeForAttempt(c, core); err != nil {
		return helper.JsonError(c, err.(*fiber.Error).Code, err.Error())
	}

	var questionID *uuid.UUID
	if s := strings.TrimSpace(c.Query("question_id")); s != "" {
		qid, err := uuid.Parse(s)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "question_id tidak valid")
		}
		// optional: validasi question belongs to quiz
		if ok, e := ctl.questionBelongsToQuiz(qid, core.QuizID); e != nil {
			return helper.JsonError(c, http.StatusInternalServerError, "gagal validasi question")
		} else if !ok {
			return helper.JsonError(c, http.StatusBadRequest, "question_id tidak milik quiz dari attempt ini")
		}
		questionID = &qid
	}

	// Pagination & sorting
	params := helper.ParseFiber(c, "answered_at", "desc", helper.DefaultOpts)
	allowed := map[string]string{
		"answered_at": "student_quiz_attempt_answer_answered_at",
		"points":      "student_quiz_attempt_answer_earned_points",
		"is_correct":  "student_quiz_attempt_answer_is_correct",
	}
	orderClause, err := params.SafeOrderClause(allowed, "answered_at")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "sort_by tidak valid")
	}

	q := ctl.DB.Model(&qmodel.StudentQuizAttemptAnswerModel{}).
		Where("student_quiz_attempt_answer_attempt_id = ?", attemptID)

	if questionID != nil {
		q = q.Where("student_quiz_attempt_answer_question_id = ?", *questionID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "gagal menghitung data")
	}

	var rows []qmodel.StudentQuizAttemptAnswerModel
	if err := q.
		Limit(params.Limit()).
		Offset(params.Offset()).
		Order(orderClause). // kolom sudah di-whitelist
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "gagal mengambil data")
	}

	resp := make([]*qdto.StudentQuizAttemptAnswerResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, qdto.FromModelStudentQuizAttemptAnswer(&rows[i]))
	}

	meta := helper.BuildMeta(total, params)
	return helper.JsonList(c, resp, meta)
}

// POST /student-quiz-attempt-answers
func (ctl *StudentQuizAttemptAnswersController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	var req qdto.CreateStudentQuizAttemptAnswerRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "payload tidak valid")
	}
	// Trim & basic sanitize untuk text
	req.StudentQuizAttemptAnswerText = strings.TrimSpace(req.StudentQuizAttemptAnswerText)
	if err := ctl.V.Struct(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, validationMessage(err))
	}
	if req.StudentQuizAttemptAnswerText == "" {
		return helper.JsonError(c, http.StatusBadRequest, "text jawaban wajib diisi")
	}

	// Load attempt & scope
	core, err := ctl.loadAttemptCore(req.StudentQuizAttemptAnswerAttemptID)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal memuat attempt")
	}
	if err := ctl.ensureScopeForAttempt(c, core); err != nil {
		return helper.JsonError(c, err.(*fiber.Error).Code, err.Error())
	}

	// Validasi: question_id harus milik quiz attempt
	if ok, e := ctl.questionBelongsToQuiz(req.StudentQuizAttemptAnswerQuestionID, core.QuizID); e != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "gagal validasi question")
	} else if !ok {
		return helper.JsonError(c, http.StatusBadRequest, "question_id tidak milik quiz dari attempt ini")
	}

	m := req.ToModel()

	// ✅ Wajib: isi quiz_id dari attempt (bukan trigger)
	m.StudentQuizAttemptAnswerQuizID = core.QuizID

	// answered_at default
	if m.StudentQuizAttemptAnswerAnsweredAt.IsZero() {
		m.StudentQuizAttemptAnswerAnsweredAt = time.Now()
	}

	if err := ctl.DB.Create(m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "jawaban untuk (attempt_id, question_id) sudah ada")
		}
		if isCheckViolation(err) {
			return helper.JsonError(c, http.StatusBadRequest, "DB menolak: text kosong atau format tidak valid")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal menyimpan data")
	}

	return helper.JsonCreated(c, "Berhasil membuat jawaban", qdto.FromModelStudentQuizAttemptAnswer(m))
}

// PATCH /student-quiz-attempt-answers/:id
func (ctl *StudentQuizAttemptAnswersController) Patch(c *fiber.Ctx) error {
	ctl.ensureValidator()

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "id tidak valid")
	}

	var req qdto.UpdateStudentQuizAttemptAnswerRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "payload tidak valid")
	}
	if req.StudentQuizAttemptAnswerText != nil {
		trimmed := strings.TrimSpace(*req.StudentQuizAttemptAnswerText)
		req.StudentQuizAttemptAnswerText = &trimmed
	}
	if err := ctl.V.Struct(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, validationMessage(err))
	}
	if req.StudentQuizAttemptAnswerText != nil && *req.StudentQuizAttemptAnswerText == "" {
		return helper.JsonError(c, http.StatusBadRequest, "text jawaban tidak boleh kosong")
	}

	var m qmodel.StudentQuizAttemptAnswerModel
	if err := ctl.DB.First(&m, "student_quiz_attempt_answer_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal mengambil data")
	}

	// Scope by attempt
	core, err := ctl.loadAttemptCore(m.StudentQuizAttemptAnswerAttemptID)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal memuat attempt")
	}
	if err := ctl.ensureScopeForAttempt(c, core); err != nil {
		return helper.JsonError(c, err.(*fiber.Error).Code, err.Error())
	}

	// Apply & save (kolom whitelist)
	req.Apply(&m)
	if err := ctl.DB.Model(&m).Select(
		"student_quiz_attempt_answer_text",
		"student_quiz_attempt_answer_is_correct",
		"student_quiz_attempt_answer_earned_points",
		"student_quiz_attempt_answer_graded_by_teacher_id",
		"student_quiz_attempt_answer_graded_at",
		"student_quiz_attempt_answer_feedback",
		"student_quiz_attempt_answer_answered_at",
	).Updates(&m).Error; err != nil {
		if isCheckViolation(err) {
			return helper.JsonError(c, http.StatusBadRequest, "DB menolak: text kosong/tidak valid")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Berhasil memperbarui jawaban", qdto.FromModelStudentQuizAttemptAnswer(&m))
}

// DELETE /student-quiz-attempt-answers/:id
func (ctl *StudentQuizAttemptAnswersController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "id tidak valid")
	}

	// Ambil attempt_id dulu untuk cek scope
	var attemptIDStr string
	if err := ctl.DB.
		Raw(`SELECT student_quiz_attempt_answer_attempt_id::text
			 FROM student_quiz_attempt_answers
			 WHERE student_quiz_attempt_answer_id = ?`, id).
		Scan(&attemptIDStr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal mengambil data")
	}
	attemptID, err := uuid.Parse(strings.TrimSpace(attemptIDStr))
	if err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "attempt_id tidak valid")
	}

	core, err := ctl.loadAttemptCore(attemptID)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal memuat attempt")
	}
	if err := ctl.ensureScopeForAttempt(c, core); err != nil {
		return helper.JsonError(c, err.(*fiber.Error).Code, err.Error())
	}

	if err := ctl.DB.Delete(&qmodel.StudentQuizAttemptAnswerModel{}, "student_quiz_attempt_answer_id = ?", id).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Berhasil menghapus jawaban", fiber.Map{"id": id})
}
