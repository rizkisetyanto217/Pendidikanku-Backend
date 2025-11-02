package controller

import (
	"schoolku_backend/internals/features/home/questionnaires/dto"
	"schoolku_backend/internals/features/home/questionnaires/model"
	helper "schoolku_backend/internals/helpers"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateUserAnswer = validator.New()

type UserQuestionnaireAnswerController struct {
	DB *gorm.DB
}

func NewUserQuestionnaireAnswerController(db *gorm.DB) *UserQuestionnaireAnswerController {
	return &UserQuestionnaireAnswerController{DB: db}
}

// =============================
// ➕ Submit Banyak Jawaban Kuisioner
// =============================
func (ctrl *UserQuestionnaireAnswerController) SubmitBulkAnswers(c *fiber.Ctx) error {
	var requests []dto.CreateUserQuestionnaireAnswerRequest
	if err := c.BodyParser(&requests); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body format (should be array)")
	}
	if len(requests) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload must be a non-empty array")
	}

	// ✅ Ambil user_id dari token
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized: user_id not found in token")
	}

	// ✅ Validasi masing-masing item
	for i, req := range requests {
		if err := validateUserAnswer.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Validation failed at item "+strconv.Itoa(i)+": "+err.Error())
		}
	}

	// ✅ Konversi ke model
	answers := make([]model.UserQuestionnaireAnswerModel, 0, len(requests))
	for _, req := range requests {
		answers = append(answers, dto.ToUserQuestionnaireAnswerModel(req, userID))
	}

	// ✅ Simpan secara bulk
	if err := ctrl.DB.Create(&answers).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to save questionnaire answers")
	}

	// opsional: kumpulkan id yang tersimpan (pakai field yang benar)
	ids := make([]string, 0, len(answers))
	for _, a := range answers {
		ids = append(ids, a.UserQuestionnaireID) // <- ini yang benar
	}

	return helper.JsonCreated(c, "Jawaban kuisioner berhasil disimpan", fiber.Map{
		"count": len(answers),
		"ids":   ids,
	})

}
