package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"
	helper "masjidku_backend/internals/helpers"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateUserQuestion = validator.New()

type LectureSessionsUserQuestionController struct {
	DB *gorm.DB
}

func NewLectureSessionsUserQuestionController(db *gorm.DB) *LectureSessionsUserQuestionController {
	return &LectureSessionsUserQuestionController{DB: db}
}

// =============================
// ➕ Create
// =============================
func (ctrl *LectureSessionsUserQuestionController) CreateLectureSessionsUserQuestion(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionsUserQuestionRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateUserQuestion.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// masjid_id wajib (sesuai schema)
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID not found in token")
	}

	rec := model.LectureSessionsUserQuestionModel{
		LectureSessionsUserQuestionAnswer:     body.LectureSessionsUserQuestionAnswer,
		LectureSessionsUserQuestionIsCorrect:  body.LectureSessionsUserQuestionIsCorrect,
		LectureSessionsUserQuestionQuestionID: body.LectureSessionsUserQuestionQuestionID,
		LectureSessionsUserQuestionMasjidID:   masjidID,
	}

	if err := ctrl.DB.Create(&rec).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create user question answer")
	}

	return helper.JsonCreated(c, "Created", dto.ToLectureSessionsUserQuestionDTO(rec))
}

// =============================
// 📄 Get All User Question Answers
// =============================
func (ctrl *LectureSessionsUserQuestionController) GetAllLectureSessionsUserQuestions(c *fiber.Ctx) error {
	var rows []model.LectureSessionsUserQuestionModel
	if err := ctrl.DB.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve data")
	}

	resp := make([]dto.LectureSessionsUserQuestionDTO, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, dto.ToLectureSessionsUserQuestionDTO(r))
	}

	return helper.JsonList(c, resp, nil)
}

// =============================
// 🔍 Get by Question ID
// =============================
func (ctrl *LectureSessionsUserQuestionController) GetByQuestionID(c *fiber.Ctx) error {
	questionID := c.Params("question_id")
	if questionID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "question_id is required")
	}

	var rows []model.LectureSessionsUserQuestionModel
	if err := ctrl.DB.
		Where("lecture_sessions_user_question_question_id = ?", questionID).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve data by question ID")
	}

	resp := make([]dto.LectureSessionsUserQuestionDTO, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, dto.ToLectureSessionsUserQuestionDTO(r))
	}

	return helper.JsonList(c, resp, nil)
}

// =============================
// 🗑️ Delete LectureSessionsUserQuestion by ID
// =============================
func (ctrl *LectureSessionsUserQuestionController) DeleteByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "id is required")
	}

	hard := strings.EqualFold(c.Query("hard"), "true") || c.Query("hard") == "1"

	var db *gorm.DB
	if hard {
		db = ctrl.DB.Unscoped().Delete(&model.LectureSessionsUserQuestionModel{}, "lecture_sessions_user_question_id = ?", id)
	} else {
		db = ctrl.DB.Delete(&model.LectureSessionsUserQuestionModel{}, "lecture_sessions_user_question_id = ?", id)
	}

	if db.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete record")
	}
	if db.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Record not found")
	}

	return helper.JsonDeleted(c, "Deleted successfully", fiber.Map{
		"id":   id,
		"hard": hard,
	})
}