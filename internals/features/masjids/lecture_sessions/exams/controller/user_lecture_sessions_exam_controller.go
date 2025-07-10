package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/exams/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/exams/model"

	"log"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserLectureSessionsExamController struct {
	DB *gorm.DB
}

func NewUserLectureSessionsExamController(db *gorm.DB) *UserLectureSessionsExamController {
	return &UserLectureSessionsExamController{DB: db}
}

// POST - User submit hasil exam (progress)
func (ctrl *UserLectureSessionsExamController) CreateUserLectureSessionsExam(c *fiber.Ctx) error {
	var body dto.CreateUserLectureSessionsExamRequest
	if err := c.BodyParser(&body); err != nil {
		log.Printf("[ERROR] Failed to parse request body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	newExam := model.UserLectureSessionsExamModel{
		UserLectureSessionsExamGrade:  body.UserLectureSessionsExamGrade,
		UserLectureSessionsExamExamID: body.UserLectureSessionsExamExamID,
		UserLectureSessionsExamUserID: body.UserLectureSessionsExamUserID,
	}

	if err := ctrl.DB.Create(&newExam).Error; err != nil {
		log.Printf("[ERROR] Failed to create exam progress: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to save exam progress")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureSessionsExamDTO(newExam))
}

// GET - Lihat semua hasil exam user
func (ctrl *UserLectureSessionsExamController) GetAllUserLectureSessionsExams(c *fiber.Ctx) error {
	var records []model.UserLectureSessionsExamModel
	if err := ctrl.DB.Find(&records).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch exams: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch exam data")
	}

	var response []dto.UserLectureSessionsExamDTO
	for _, r := range records {
		response = append(response, dto.ToUserLectureSessionsExamDTO(r))
	}

	return c.JSON(response)
}

// GET - Detail hasil exam user berdasarkan ID
func (ctrl *UserLectureSessionsExamController) GetUserLectureSessionsExamByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var record model.UserLectureSessionsExamModel
	if err := ctrl.DB.First(&record, "user_lecture_sessions_exam_id = ?", id).Error; err != nil {
		log.Printf("[ERROR] Exam not found: %v", err)
		return fiber.NewError(fiber.StatusNotFound, "Exam record not found")
	}

	return c.JSON(dto.ToUserLectureSessionsExamDTO(record))
}
