// file: internals/features/home/questionnaires/controller/questionnaire_question_controller.go
package controller

import (
	"errors"

	"schoolku_backend/internals/features/home/questionnaires/dto"
	"schoolku_backend/internals/features/home/questionnaires/model"
	helper "schoolku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuestionnaireQuestionController struct {
	DB *gorm.DB
}

func NewQuestionnaireQuestionController(db *gorm.DB) *QuestionnaireQuestionController {
	return &QuestionnaireQuestionController{DB: db}
}

var validate = validator.New()

// ➕ Create
func (ctrl *QuestionnaireQuestionController) CreateQuestion(c *fiber.Ctx) error {
	var req dto.CreateQuestionnaireQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if err := validateQuestionCombination(req.QuestionType, req.QuestionOptions, req.QuestionScope, req.EventID, req.LectureSessionID); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	question := dto.ToQuestionnaireQuestionModel(req)
	if err := ctrl.DB.Create(&question).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create question")
	}
	return helper.JsonCreated(c, "Pertanyaan berhasil dibuat", dto.ToQuestionnaireQuestionDTO(question))
}

// 📄 Get All
func (ctrl *QuestionnaireQuestionController) GetAllQuestions(c *fiber.Ctx) error {
	var questions []model.QuestionnaireQuestionModel
	if err := ctrl.DB.Find(&questions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve questions")
	}
	resp := make([]dto.QuestionnaireQuestionDTO, 0, len(questions))
	for _, q := range questions {
		resp = append(resp, dto.ToQuestionnaireQuestionDTO(q))
	}
	return helper.JsonList(c, resp, nil)
}

// 🔍 Get By Scope (1=general, 2=event, 3=lecture)
func (ctrl *QuestionnaireQuestionController) GetQuestionsByScope(c *fiber.Ctx) error {
	scope := c.Params("scope")
	var questions []model.QuestionnaireQuestionModel
	if err := ctrl.DB.Where("questionnaire_question_scope = ?", scope).Find(&questions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve questions by scope")
	}
	resp := make([]dto.QuestionnaireQuestionDTO, 0, len(questions))
	for _, q := range questions {
		resp = append(resp, dto.ToQuestionnaireQuestionDTO(q))
	}
	return helper.JsonList(c, resp, nil)
}

// 🔎 Get By ID
func (ctrl *QuestionnaireQuestionController) GetQuestionByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		// hapus validasi ini kalau ID di DB bukan UUID
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	var q model.QuestionnaireQuestionModel
	if err := ctrl.DB.First(&q, "questionnaire_question_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Question not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to get question")
	}
	return helper.JsonOK(c, "OK", dto.ToQuestionnaireQuestionDTO(q))
}

// ✏️ PUT (Partial Update — field yang tidak dikirim tidak diubah)
func (ctrl *QuestionnaireQuestionController) PutQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var q model.QuestionnaireQuestionModel
	if err := ctrl.DB.First(&q, "questionnaire_question_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Question not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to get question")
	}

	var req dto.UpdateQuestionnaireQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// compose nilai final dari current + req (yang diisi)
	nextType := q.QuestionType
	nextText := q.QuestionText
	nextOptions := q.QuestionOptions
	nextScope := q.QuestionScope
	nextEventID := q.EventID
	nextLectureID := q.LectureSessionID

	if req.QuestionText != nil {
		nextText = *req.QuestionText
	}
	if req.QuestionType != nil {
		nextType = *req.QuestionType
	}
	if req.QuestionOptions != nil {
		nextOptions = *req.QuestionOptions
	}
	if req.QuestionScope != nil {
		nextScope = *req.QuestionScope
	}
	if req.EventID != nil {
		nextEventID = req.EventID
	}
	if req.LectureSessionID != nil {
		nextLectureID = req.LectureSessionID
	}

	// validasi kombinasi akhir
	if err := validateQuestionCombination(nextType, nextOptions, nextScope, nextEventID, nextLectureID); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// apply & save
	q.QuestionText = nextText
	q.QuestionType = nextType
	q.QuestionOptions = nextOptions
	q.QuestionScope = nextScope
	q.EventID = nextEventID
	q.LectureSessionID = nextLectureID

	if err := ctrl.DB.Save(&q).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update question")
	}
	return helper.JsonUpdated(c, "Pertanyaan berhasil diperbarui", dto.ToQuestionnaireQuestionDTO(q))
}

// ❌ Delete
func (ctrl *QuestionnaireQuestionController) DeleteQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	if err := ctrl.DB.Delete(&model.QuestionnaireQuestionModel{}, "questionnaire_question_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete question")
	}
	return helper.JsonDeleted(c, "Pertanyaan berhasil dihapus", fiber.Map{"id": id})
}

// 🔐 Validator kombinasi (fixed, lebih bersih)
func validateQuestionCombination(
	qType int, options []string,
	scope int, eventID *string, lectureID *string,
) error {
	// type 3 (choice) ⇒ minimal 2 opsi
	if qType == 3 && len(options) < 2 {
		return errors.New("question_options minimal 2 item untuk question_type=3 (choice)")
	}
	// type 1/2 ⇒ tidak boleh ada options
	if (qType == 1 || qType == 2) && len(options) > 0 {
		return errors.New("question_options hanya untuk question_type=3 (choice)")
	}

	switch scope {
	case 1: // general
		// OK
	case 2: // event
		if eventID == nil || *eventID == "" { // ← cukup begini
			return errors.New("event_id wajib untuk question_scope=2 (event)")
		}
	case 3: // lecture session
		if lectureID == nil || *lectureID == "" { // ← cukup begini
			return errors.New("lecture_session_id wajib untuk question_scope=3 (lecture)")
		}
	default:
		return errors.New("question_scope harus 1, 2, atau 3")
	}
	return nil
}
