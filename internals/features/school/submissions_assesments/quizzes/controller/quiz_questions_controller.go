// file: internals/features/school/submissions_assesments/quizzes/controller/quiz_questions_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	qdto "schoolku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "schoolku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

/* =========================================================
   Controller
========================================================= */

type QuizQuestionsController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewQuizQuestionsController(db *gorm.DB) *QuizQuestionsController {
	return &QuizQuestionsController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* =========================================================
   Tenant helpers
========================================================= */

func (ctl *QuizQuestionsController) applyFilters(db *gorm.DB, schoolID uuid.UUID, quizID *uuid.UUID, qType string, q string) *gorm.DB {
	db = db.Where("quiz_question_school_id = ? AND quiz_question_deleted_at IS NULL", schoolID)
	if quizID != nil && *quizID != uuid.Nil {
		db = db.Where("quiz_question_quiz_id = ?", *quizID)
	}
	if t := strings.ToLower(strings.TrimSpace(qType)); t == "single" || t == "essay" {
		db = db.Where("quiz_question_type = ?", t)
	}
	if s := strings.TrimSpace(q); s != "" {
		like := "%" + strings.ToLower(s) + "%"
		db = db.Where("(LOWER(quiz_question_text) LIKE ? OR LOWER(COALESCE(quiz_question_explanation,'')) LIKE ?)", like, like)
	}
	return db
}

func (ctl *QuizQuestionsController) applySort(db *gorm.DB, sort string) *gorm.DB {
	switch strings.TrimSpace(sort) {
	case "created_at":
		return db.Order("quiz_question_created_at ASC")
	case "desc_created_at", "":
		return db.Order("quiz_question_created_at DESC")
	case "points":
		return db.Order("quiz_question_points ASC")
	case "desc_points":
		return db.Order("quiz_question_points DESC")
	case "type":
		return db.Order("quiz_question_type ASC")
	case "desc_type":
		return db.Order("quiz_question_type DESC")
	default:
		return db.Order("quiz_question_created_at DESC")
	}
}

/* =========================================================
   WRITE (DKM/Admin)
========================================================= */

// POST /quiz-questions
func (ctl *QuizQuestionsController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	var req qdto.CreateQuizQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔒 Resolve school + wajib DKM/Admin
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	mid, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Force school_id dari tenant context
	req.QuizQuestionSchoolID = mid

	// Safety: pastikan quiz_id milik school tenant
	var ok bool
	if err := ctl.DB.Raw(`
		SELECT EXISTS(
		  SELECT 1 FROM quizzes
		  WHERE quiz_id = ? AND quiz_school_id = ?
		)
	`, req.QuizQuestionQuizID, mid).Scan(&ok).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	if !ok {
		return helper.JsonError(c, fiber.StatusForbidden, "Quiz tidak milik tenant aktif")
	}

	m, err := req.ToModel()
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := ctl.DB.Create(m).Error; err != nil {
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar aturan bentuk data (CHECK)")
		}
		if isForeignKeyViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Relasi tidak valid (quiz/school)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonCreated(c, "Soal berhasil dibuat", qdto.FromModelQuizQuestion(m))
}

// PATCH /quiz-questions/:id
func (ctl *QuizQuestionsController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// 🔒 Resolve school + wajib DKM/Admin
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	mid, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m qmodel.QuizQuestionModel
	if err := ctl.DB.
		First(&m, "quiz_question_id = ? AND quiz_question_school_id = ? AND quiz_question_deleted_at IS NULL", id, mid).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Soal tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var req qdto.PatchQuizQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	if err := req.ApplyToModel(&m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Jika quiz_id berubah, validasi quiz baru milik tenant
	if req.QuizQuestionQuizID.ShouldUpdate() && !req.QuizQuestionQuizID.IsNull() {
		newQID := req.QuizQuestionQuizID.Val()
		var ok bool
		if err := ctl.DB.Raw(`
			SELECT EXISTS(SELECT 1 FROM quizzes WHERE quiz_id = ? AND quiz_school_id = ?)
		`, newQID, mid).Scan(&ok).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		if !ok {
			return helper.JsonError(c, fiber.StatusForbidden, "Quiz tidak milik tenant aktif")
		}
	}

	// Save all changed fields (optimistic)
	if err := ctl.DB.
		Model(&qmodel.QuizQuestionModel{}).
		Where("quiz_question_id = ?", m.QuizQuestionID).
		Select("*").
		Updates(&m).Error; err != nil {
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar aturan bentuk data (CHECK)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Reload
	if err := ctl.DB.First(&m, "quiz_question_id = ?", m.QuizQuestionID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonUpdated(c, "Soal diperbarui", qdto.FromModelQuizQuestion(&m))
}

// DELETE /quiz-questions/:id (soft delete: set deleted_at)
func (ctl *QuizQuestionsController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// 🔒 Resolve school + wajib DKM/Admin
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	mid, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// pastikan exist dan milik tenant
	var m qmodel.QuizQuestionModel
	if err := ctl.DB.Select("quiz_question_id").
		First(&m, "quiz_question_id = ? AND quiz_question_school_id = ? AND quiz_question_deleted_at IS NULL", id, mid).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Soal tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	now := time.Now()
	if err := ctl.DB.Model(&qmodel.QuizQuestionModel{}).
		Where("quiz_question_id = ?", id).
		Update("quiz_question_deleted_at", now).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonDeleted(c, "Soal dihapus", nil)
}

/* =========================================================
   Small utils
========================================================= */

func pageOffset(page, perPage int) int {
	if page <= 0 {
		return 0
	}
	return page * perPage
}

/* =========================================================
   DB error helpers
========================================================= */

func isCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "check constraint")
}

func isForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "foreign key")
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unique")
}
