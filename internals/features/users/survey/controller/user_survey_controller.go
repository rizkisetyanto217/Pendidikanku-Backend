package controller

import (
	"log"
	"time"

	model "masjidku_backend/internals/features/users/survey/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserSurveyController struct {
	DB *gorm.DB
}

func NewUserSurveyController(db *gorm.DB) *UserSurveyController {
	return &UserSurveyController{DB: db}
}

// 📩 SubmitSurveyAnswers menyimpan jawaban survei yang dikirim oleh user.
func (ctrl *UserSurveyController) SubmitSurveyAnswers(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var inputs []struct {
		SurveyQuestionID int    `json:"survey_question_id"`
		UserAnswer       string `json:"user_answer"`
	}

	if err := c.BodyParser(&inputs); err != nil {
		log.Println("[ERROR] Failed to parse user survey input:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// Validasi isi minimal
	if len(inputs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No answers provided"})
	}

	// Simpan semua jawaban ke DB
	for _, input := range inputs {
		answer := model.UserSurvey{
			UserSurveyUserID:     userID,
			UserSurveyQuestionID: input.SurveyQuestionID,
			UserSurveyAnswer:     input.UserAnswer,
			CreatedAt:            time.Now(),
		}

		if err := ctrl.DB.Create(&answer).Error; err != nil {
			log.Println("[ERROR] Failed to save user survey:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save user survey"})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Survey answers submitted successfully",
	})
}
