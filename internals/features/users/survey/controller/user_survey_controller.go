package controller

import (
	"fmt"
	"log"
	"time"

	model "masjidku_backend/internals/features/users/survey/model"
	helper "masjidku_backend/internals/helpers"

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

// 📩 SubmitSurveyAnswers menyimpan jawaban survei yang dikirim oleh user (bulk insert).
func (ctrl *UserSurveyController) SubmitSurveyAnswers(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Invalid user ID")
	}

	// payload harus array
	var inputs []struct {
		SurveyQuestionID int    `json:"survey_question_id"`
		UserAnswer       string `json:"user_answer"`
	}
	if err := c.BodyParser(&inputs); err != nil {
		log.Println("[ERROR] Failed to parse user survey input:", err)
	return helper.JsonError(c, fiber.StatusBadRequest, "Invalid input (expected JSON array)")
	}
	if len(inputs) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "No answers provided")
	}

	// siapkan bulk
	now := time.Now()
	answers := make([]model.UserSurvey, 0, len(inputs))
	for i, in := range inputs {
		if in.SurveyQuestionID == 0 || in.UserAnswer == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, 
				"Invalid item at index "+fmt.Sprint(i)+": survey_question_id and user_answer are required")
		}
		answers = append(answers, model.UserSurvey{
			UserSurveyUserID:     userID,
			UserSurveyQuestionID: in.SurveyQuestionID,
			UserSurveyAnswer:     in.UserAnswer,
			CreatedAt:            now,
		})
	}

	// bulk insert (gunakan CreateInBatches untuk aman di payload besar)
	if err := ctrl.DB.CreateInBatches(&answers, 100).Error; err != nil {
		log.Println("[ERROR] Failed to save user survey:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to save user survey")
	}

	return helper.JsonCreated(c, "Survey answers submitted successfully", fiber.Map{
		"count": len(answers),
	})
}
