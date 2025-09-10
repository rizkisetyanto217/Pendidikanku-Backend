// file: internals/features/school/submissions_assesments/quizzes/controller/quiz_items_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	validator "github.com/go-playground/validator/v10"

	qdto "masjidku_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "masjidku_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "masjidku_backend/internals/helpers" // <— pakai helper kamu
)

type QuizItemsController struct {
	DB        *gorm.DB
	validator *validator.Validate
}

func NewQuizItemsController(db *gorm.DB) *QuizItemsController { return &QuizItemsController{DB: db} }

func (ctl *QuizItemsController) ensureValidator() {
	if ctl.validator == nil {
		ctl.validator = validator.New()
	}
}

// POST /quizzes/items
func (ctl *QuizItemsController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	var req qdto.CreateQuizItemRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal")
	}

	m, err := req.ToModel()
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Shape baris tidak valid")
	}

	if err := ctl.DB.Create(m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Data duplikat / melanggar unique index")
		}
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar CHECK constraint")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan")
	}
	return helper.JsonCreated(c, "Berhasil membuat quiz item", qdto.FromModelQuizItem(m))
}

// POST /quizzes/items/bulk-single
func (ctl *QuizItemsController) BulkCreateSingle(c *fiber.Ctx) error {
	ctl.ensureValidator()

	var req qdto.CreateSingleQuestionWithOptionsRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal")
	}
	models, err := req.ToModels()
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi domain gagal")
	}

	if err := ctl.DB.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&models).Error
	}); err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Melanggar aturan unik (tepat satu opsi benar / duplikasi option)")
		}
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar CHECK constraint")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan (transaksi dibatalkan)")
	}

	return helper.JsonCreated(c, "Berhasil membuat soal single beserta opsi", qdto.FromModelsQuizItems(models))
}



// PATCH /quiz-items/:id
func (ctl *QuizItemsController) Patch(c *fiber.Ctx) error {
	ctl.ensureValidator()

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var req qdto.UpdateQuizItemRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal")
	}

	// load current row
	var m qmodel.QuizItemModel
	if err := ctl.DB.First(&m, "quiz_items_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Quiz item tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// clone untuk validasi shape
	mPatched := m
	if err := req.ApplyToModel(&mPatched); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Patch tidak valid")
	}
	if err := mPatched.RowShapeValid(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Patch melanggar bentuk data (single/essay)")
	}

	// build updates hanya dari field yang dikirim
	updates := map[string]any{}
	if req.QuizItemsQuizID != nil {
		updates["quiz_items_quiz_id"] = *req.QuizItemsQuizID
	}
	if req.QuizItemsQuestionID != nil {
		updates["quiz_items_question_id"] = *req.QuizItemsQuestionID
	}
	if req.QuizItemsQuestionType != nil {
		updates["quiz_items_question_type"] = *req.QuizItemsQuestionType
	}
	if req.QuizItemsQuestionText != nil {
		updates["quiz_items_question_text"] = strings.TrimSpace(*req.QuizItemsQuestionText)
	}
	if req.QuizItemsPoints != nil {
		updates["quiz_items_points"] = *req.QuizItemsPoints
	}
	// kolom opsi (boleh NULL utk ESSAY; pointer di DTO mengindikasikan field dikirim)
	if req.QuizItemsOptionID != nil {
		updates["quiz_items_option_id"] = req.QuizItemsOptionID // bisa nil
	}
	if req.QuizItemsOptionText != nil {
		txt := strings.TrimSpace(*req.QuizItemsOptionText) // bisa kosong string
		updates["quiz_items_option_text"] = &txt
	}
	if req.QuizItemsOptionIsCorrect != nil {
		updates["quiz_items_option_is_correct"] = req.QuizItemsOptionIsCorrect // bisa nil
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", qdto.FromModelQuizItem(&m))
	}

	if err := ctl.DB.Model(&m).Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Melanggar aturan unik")
		}
		if isCheckViolation(err) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Melanggar CHECK constraint")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan")
	}

	// reload untuk response
	if err := ctl.DB.First(&m, "quiz_items_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat ulang data")
	}

	return helper.JsonUpdated(c, "Berhasil memperbarui quiz item", qdto.FromModelQuizItem(&m))
}



// GET /quizzes/items?quiz_id=...
func (ctl *QuizItemsController) ListByQuiz(c *fiber.Ctx) error {
	quizIDStr := strings.TrimSpace(c.Query("quiz_id"))
	if quizIDStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter quiz_id wajib diisi")
	}
	quizID, err := uuid.Parse(quizIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "quiz_id tidak valid")
	}

	var total int64
	if err := ctl.DB.Model(&qmodel.QuizItemModel{}).
		Where("quiz_items_quiz_id = ?", quizID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// pakai pagination helper
	p := helper.ParseFiber(c, "quiz_items_question_id", "asc", helper.DefaultOpts)

	var rows []*qmodel.QuizItemModel
	if err := ctl.DB.
		Where("quiz_items_quiz_id = ?", quizID).
		Order("quiz_items_question_id, quiz_items_option_is_correct DESC NULLS LAST").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, qdto.FromModelsQuizItems(rows), meta)
}

// GET /quizzes/items/by-question/:question_id
func (ctl *QuizItemsController) ListByQuestion(c *fiber.Ctx) error {
	qid, err := uuid.Parse(strings.TrimSpace(c.Params("question_id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "question_id tidak valid")
	}

	var total int64
	if err := ctl.DB.Model(&qmodel.QuizItemModel{}).
		Where("quiz_items_question_id = ?", qid).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	p := helper.ParseFiber(c, "quiz_items_option_is_correct", "desc", helper.DefaultOpts)

	var rows []*qmodel.QuizItemModel
	if err := ctl.DB.
		Where("quiz_items_question_id = ?", qid).
		Order("quiz_items_option_is_correct DESC NULLS LAST").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, qdto.FromModelsQuizItems(rows), meta)
}

// GET /quizzes/items/:id
func (ctl *QuizItemsController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}
	var m qmodel.QuizItemModel
	if err := ctl.DB.First(&m, "quiz_items_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Quiz item tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return helper.JsonOK(c, "OK", qdto.FromModelQuizItem(&m))
}

// DELETE /quizzes/items/:id
func (ctl *QuizItemsController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}
	if err := ctl.DB.Delete(&qmodel.QuizItemModel{}, "quiz_items_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus")
	}
	return helper.JsonDeleted(c, "Berhasil menghapus", fiber.Map{"deleted_id": id})
}

/* ====== DB error helpers (string-based, tetap dipakai) ====== */

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	ls := strings.ToLower(s)
	return strings.Contains(s, "SQLSTATE 23505") ||
		strings.Contains(ls, "unique violation") ||
		strings.Contains(ls, "unique constraint") ||
		strings.Contains(ls, "duplicate key")
}

func isCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	ls := strings.ToLower(s)
	return strings.Contains(s, "SQLSTATE 23514") ||
		strings.Contains(ls, "check violation") ||
		strings.Contains(ls, "violates check constraint")
}
