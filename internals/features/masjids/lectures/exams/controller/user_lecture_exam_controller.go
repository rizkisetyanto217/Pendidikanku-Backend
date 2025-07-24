package controller

import (
	"masjidku_backend/internals/features/masjids/lectures/exams/dto"
	"masjidku_backend/internals/features/masjids/lectures/exams/model"

	"log"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserLectureExamController struct {
	DB *gorm.DB
}

func NewUserLectureExamController(db *gorm.DB) *UserLectureExamController {
	return &UserLectureExamController{DB: db}
}

// POST - User submit hasil exam (progress)
func (ctrl *UserLectureExamController) CreateUserLectureExam(c *fiber.Ctx) error {
	var body dto.CreateUserLectureExamRequest
	if err := c.BodyParser(&body); err != nil {
		log.Printf("[ERROR] Failed to parse request body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	newExam := model.UserLectureExamModel{
		UserLectureExamGrade:  body.UserLectureExamGrade,
		UserLectureExamExamID: body.UserLectureExamExamID,
		UserLectureExamUserID: body.UserLectureExamUserID,
	}

	if err := ctrl.DB.Create(&newExam).Error; err != nil {
		log.Printf("[ERROR] Failed to create exam progress: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to save exam progress")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToUserLectureExamDTO(newExam))
}

// GET - Lihat semua hasil exam user
func (ctrl *UserLectureExamController) GetAllUserLectureExams(c *fiber.Ctx) error {
	var records []model.UserLectureExamModel
	if err := ctrl.DB.Find(&records).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch exams: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch exam data")
	}

	var response []dto.UserLectureExamDTO
	for _, r := range records {
		response = append(response, dto.ToUserLectureExamDTO(r))
	}

	return c.JSON(response)
}

// GET - Detail hasil exam user berdasarkan ID
func (ctrl *UserLectureExamController) GetUserLectureExamByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var record model.UserLectureExamModel
	if err := ctrl.DB.First(&record, "user_lecture_exam_id = ?", id).Error; err != nil {
		log.Printf("[ERROR] Exam not found: %v", err)
		return fiber.NewError(fiber.StatusNotFound, "Exam record not found")
	}

	return c.JSON(dto.ToUserLectureExamDTO(record))
}
