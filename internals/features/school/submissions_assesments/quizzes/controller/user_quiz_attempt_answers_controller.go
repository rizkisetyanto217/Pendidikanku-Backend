// file: internals/features/quiz/user_attempts/controller/user_quiz_attempt_answer_controller.go
package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"masjidku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	"masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserQuizAttemptAnswersController struct {
	DB *gorm.DB
	V  *validator.Validate
}

func NewUserQuizAttemptAnswersController(db *gorm.DB) *UserQuizAttemptAnswersController {
	return &UserQuizAttemptAnswersController{
		DB: db,
		V:  validator.New(),
	}
}

func (ctl *UserQuizAttemptAnswersController) ensureValidator() {
	if ctl.V == nil {
		ctl.V = validator.New()
	}
}

// GET /user-quiz-attempt-answers?attempt_id=...&question_id=...&sort_by=&order=&page=&per_page=
func (ctl *UserQuizAttemptAnswersController) List(c *fiber.Ctx) error {
	// Wajib minimal filter attempt_id supaya efisien
	attemptIDStr := strings.TrimSpace(c.Query("attempt_id"))
	if attemptIDStr == "" {
		return helper.JsonError(c, http.StatusBadRequest, "attempt_id wajib diisi")
	}
	attemptID, err := uuid.Parse(attemptIDStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "attempt_id tidak valid")
	}

	var questionID *uuid.UUID
	if s := strings.TrimSpace(c.Query("question_id")); s != "" {
		qid, err := uuid.Parse(s)
		if err != nil {
			return helper.JsonError(c, http.StatusBadRequest, "question_id tidak valid")
		}
		questionID = &qid
	}

	// Pagination & sorting
	params := helper.ParseFiber(c, "answered_at", "desc", helper.DefaultOpts)
	allowed := map[string]string{
		"answered_at": "user_quiz_attempt_answers_answered_at",
		"points":      "user_quiz_attempt_answers_earned_points",
		"is_correct":  "user_quiz_attempt_answers_is_correct",
	}
	orderClause, err := params.SafeOrderClause(allowed, "answered_at")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "sort_by tidak valid")
	}

	q := ctl.DB.Model(&model.UserQuizAttemptAnswerModel{}).
		Where("user_quiz_attempt_answers_attempt_id = ?", attemptID)

	if questionID != nil {
		q = q.Where("user_quiz_attempt_answers_question_id = ?", *questionID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "gagal menghitung data")
	}

	var rows []model.UserQuizAttemptAnswerModel
	if err := q.
		Limit(params.Limit()).
		Offset(params.Offset()).
		// RAW order by aman dari whitelist
		Order(orderClause).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "gagal mengambil data")
	}

	// mapping ke response
	resp := make([]*dto.UserQuizAttemptAnswerResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, dto.ToUserQuizAttemptAnswerResponse(&rows[i]))
	}

	meta := helper.BuildMeta(total, params)
	return helper.JsonList(c, resp, meta)
}

// GET /user-quiz-attempt-answers/:id
func (ctl *UserQuizAttemptAnswersController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "id tidak valid")
	}

	var m model.UserQuizAttemptAnswerModel
	if err := ctl.DB.First(&m, "user_quiz_attempt_answers_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal mengambil data")
	}

	return helper.JsonOK(c, "OK", dto.ToUserQuizAttemptAnswerResponse(&m))
}

// POST /user-quiz-attempt-answers
func (ctl *UserQuizAttemptAnswersController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	var req dto.CreateUserQuizAttemptAnswerRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "payload tidak valid")
	}

	// Validasi dasar
	if err := ctl.V.Struct(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, validationMessage(err))
	}

	// Guard XOR tambahan di server (selain validator + constraint DB)
	if !exactlyOneFilled(req.UserQuizAttemptAnswersSelectedOptionID, req.UserQuizAttemptAnswersText) {
		return helper.JsonError(c, http.StatusBadRequest, "isi tepat salah satu: selected_option_id ATAU text")
	}

	m := req.ToModel()

	// Pastikan answered_at terisi (biarkan DB default jika kosong)
	if m.UserQuizAttemptAnswersAnsweredAt.IsZero() {
		m.UserQuizAttemptAnswersAnsweredAt = time.Now()
	}

	// Simpan
	if err := ctl.DB.Create(m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "jawaban untuk (attempt_id, question_id) sudah ada")
		}
		// DB constraint XOR juga akan fail → kirim pesan ramah
		if isCheckViolation(err) {
			return helper.JsonError(c, http.StatusBadRequest, "DB menolak: isi tepat salah satu antara selected_option_id atau text")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal menyimpan data")
	}

	return helper.JsonCreated(c, "Berhasil membuat jawaban", dto.ToUserQuizAttemptAnswerResponse(m))
}

// PATCH /user-quiz-attempt-answers/:id
func (ctl *UserQuizAttemptAnswersController) Patch(c *fiber.Ctx) error {
	ctl.ensureValidator()

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "id tidak valid")
	}

	var req dto.UpdateUserQuizAttemptAnswerRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "payload tidak valid")
	}
	if err := ctl.V.Struct(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, validationMessage(err))
	}

	var m model.UserQuizAttemptAnswerModel
	if err := ctl.DB.First(&m, "user_quiz_attempt_answers_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal mengambil data")
	}

	// Jika user mengirim salah satu dari option/text, pastikan XOR terpenuhi pada nilai akhir
	// (nilai akhir = existing + patch)
	if req.UserQuizAttemptAnswersSelectedOptionID != nil || req.UserQuizAttemptAnswersText != nil {
		nextOption := m.UserQuizAttemptAnswersSelectedOptionID
		nextText := m.UserQuizAttemptAnswersText
		if req.UserQuizAttemptAnswersSelectedOptionID != nil {
			nextOption = req.UserQuizAttemptAnswersSelectedOptionID
		}
		if req.UserQuizAttemptAnswersText != nil {
			nextText = req.UserQuizAttemptAnswersText
		}
		if !exactlyOneFilled(nextOption, nextText) {
			return helper.JsonError(c, http.StatusBadRequest, "isi tepat salah satu: selected_option_id ATAU text")
		}
	}

	// Apply dan simpan
	req.Apply(&m)

	// Hanya update kolom yang berubah
	if err := ctl.DB.Model(&m).Select(
		"user_quiz_attempt_answers_selected_option_id",
		"user_quiz_attempt_answers_text",
		"user_quiz_attempt_answers_is_correct",
		"user_quiz_attempt_answers_earned_points",
		"user_quiz_attempt_answers_graded_by_teacher_id",
		"user_quiz_attempt_answers_graded_at",
		"user_quiz_attempt_answers_feedback",
		"user_quiz_attempt_answers_answered_at",
	).Updates(&m).Error; err != nil {
		if isCheckViolation(err) {
			return helper.JsonError(c, http.StatusBadRequest, "DB menolak: XOR selected_option_id vs text tidak terpenuhi")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Berhasil memperbarui jawaban", dto.ToUserQuizAttemptAnswerResponse(&m))
}

// DELETE /user-quiz-attempt-answers/:id
func (ctl *UserQuizAttemptAnswersController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "id tidak valid")
	}

	if err := ctl.DB.Delete(&model.UserQuizAttemptAnswerModel{}, "user_quiz_attempt_answers_id = ?", id).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Berhasil menghapus jawaban", fiber.Map{"id": id})
}

/* ===================== Helpers ===================== */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	s := strings.TrimSpace(c.Params(name))
	return uuid.Parse(s)
}

func exactlyOneFilled(optID *uuid.UUID, text *string) bool {
	hasOpt := optID != nil
	hasText := text != nil && strings.TrimSpace(*text) != ""
	// XOR: true jika tepat satu
	return (hasOpt || hasText) && !(hasOpt && hasText)
}

func validationMessage(err error) string {
	// Bisa kamu kembangkan jadi multilanguage/detail per-field
	return "validasi gagal: " + err.Error()
}
