package controller

import (
	"masjidku_backend/internals/features/masjids/lecture_sessions/exams/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/exams/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureSessionsExamController struct {
	DB *gorm.DB
}

func NewLectureSessionsExamController(db *gorm.DB) *LectureSessionsExamController {
	return &LectureSessionsExamController{DB: db}
}

// ➕ Create exam
func (ctrl *LectureSessionsExamController) CreateLectureSessionsExam(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionsExamRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	newExam := model.LectureSessionsExamModel{
		LectureSessionsExamTitle:       body.LectureSessionsExamTitle,
		LectureSessionsExamDescription: body.LectureSessionsExamDescription,
		LectureSessionsExamLectureID:   body.LectureSessionsExamLectureID,
	}

	if err := ctrl.DB.Create(&newExam).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create exam")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionsExamDTO(newExam))
}

// 📄 Get all exams
func (ctrl *LectureSessionsExamController) GetAllLectureSessionsExams(c *fiber.Ctx) error {
	var exams []model.LectureSessionsExamModel
	if err := ctrl.DB.Order("lecture_sessions_exam_created_at DESC").Find(&exams).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve exams")
	}

	var response []dto.LectureSessionsExamDTO
	for _, exam := range exams {
		response = append(response, dto.ToLectureSessionsExamDTO(exam))
	}

	return c.JSON(response)
}

// 🔍 Get exam by ID
func (ctrl *LectureSessionsExamController) GetLectureSessionsExamByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var exam model.LectureSessionsExamModel
	if err := ctrl.DB.First(&exam, "lecture_sessions_exam_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Exam not found")
	}

	return c.JSON(dto.ToLectureSessionsExamDTO(exam))
}

// ✏️ Update exam
func (ctrl *LectureSessionsExamController) UpdateLectureSessionsExam(c *fiber.Ctx) error {
	id := c.Params("id")
	var body dto.UpdateLectureSessionsExamRequest

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	var exam model.LectureSessionsExamModel
	if err := ctrl.DB.First(&exam, "lecture_sessions_exam_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Exam not found")
	}

	exam.LectureSessionsExamTitle = body.LectureSessionsExamTitle
	exam.LectureSessionsExamDescription = body.LectureSessionsExamDescription

	if err := ctrl.DB.Save(&exam).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update exam")
	}

	return c.JSON(dto.ToLectureSessionsExamDTO(exam))
}

// ❌ Delete exam
func (ctrl *LectureSessionsExamController) DeleteLectureSessionsExam(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.LectureSessionsExamModel{}, "lecture_sessions_exam_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete exam")
	}

	return c.JSON(fiber.Map{"message": "Exam deleted successfully"})
}
