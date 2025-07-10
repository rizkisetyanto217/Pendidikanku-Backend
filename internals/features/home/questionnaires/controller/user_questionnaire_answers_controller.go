package controller

import (
	"masjidku_backend/internals/features/home/questionnaires/dto"
	"masjidku_backend/internals/features/home/questionnaires/model"
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
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body format (should be array)")
	}

	// ✅ Ambil user_id dari token
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized: user_id not found in token")
	}

	// ✅ Validasi masing-masing item
	// ✅ Validasi masing-masing item
	for i, req := range requests {
		if err := validateUserAnswer.Struct(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Validation failed at item "+strconv.Itoa(i)+": "+err.Error())
		}
	}

	// ✅ Konversi ke model
	var answers []model.UserQuestionnaireAnswerModel
	for _, req := range requests {
		answer := dto.ToUserQuestionnaireAnswerModel(req, userID)
		answers = append(answers, answer)
	}

	// ✅ Simpan secara bulk
	if err := ctrl.DB.Create(&answers).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to save questionnaire answers")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Jawaban kuisioner berhasil disimpan",
		"count":   len(answers),
	})
}
