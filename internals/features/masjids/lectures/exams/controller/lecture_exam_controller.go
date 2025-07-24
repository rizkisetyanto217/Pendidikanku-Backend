package controller

import (
	modelLectureSessionQuestion "masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"
	"masjidku_backend/internals/features/masjids/lectures/exams/dto"
	"masjidku_backend/internals/features/masjids/lectures/exams/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureExamController struct {
	DB *gorm.DB
}

func NewLectureExamController(db *gorm.DB) *LectureExamController {
	return &LectureExamController{DB: db}
}

// ➕ Create exam
func (ctrl *LectureExamController) CreateLectureExam(c *fiber.Ctx) error {
	// Ambil masjid_id dari token (middleware sebelumnya harus sudah set ini)
	masjidID := c.Locals("masjid_id")
	if masjidID == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID not found in token")
	}

	var body dto.CreateLectureExamRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	newExam := model.LectureExamModel{
		LectureExamTitle:       body.LectureExamTitle,
		LectureExamDescription: body.LectureExamDescription,
		LectureExamLectureID:   body.LectureExamLectureID,
		LectureExamMasjidID:    masjidID.(string), // casting to string
	}

	if err := ctrl.DB.Create(&newExam).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create exam")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureExamDTO(newExam))
}


// 📄 Get all exams
func (ctrl *LectureExamController) GetAllLectureExams(c *fiber.Ctx) error {
	var exams []model.LectureExamModel
	if err := ctrl.DB.Order("lecture_exam_created_at DESC").Find(&exams).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve exams")
	}

	var response []dto.LectureExamDTO
	for _, exam := range exams {
		response = append(response, dto.ToLectureExamDTO(exam))
	}

	return c.JSON(response)
}

// 📄 Get exam by ID with questions
func (ctrl *LectureExamController) GetLectureExamWithQuestions(c *fiber.Ctx) error {
	examID := c.Params("id")

	// Ambil data exam
	var exam model.LectureExamModel
	if err := ctrl.DB.First(&exam, "lecture_exam_id = ?", examID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Exam not found")
	}

	// Ambil semua soal yang berkaitan dengan exam tersebut
	var questions []modelLectureSessionQuestion.LectureSessionsQuestionModel
	if err := ctrl.DB.
		Where("lecture_question_exam_id = ?", examID).
		Order("lecture_sessions_question_created_at ASC").
		Find(&questions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch questions")
	}

	return c.JSON(fiber.Map{
		"exam":      exam,
		"questions": questions,
	})
}


// 🔍 Get exam by ID
func (ctrl *LectureExamController) GetLectureExamByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var exam model.LectureExamModel
	if err := ctrl.DB.First(&exam, "lecture_exam_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Exam not found")
	}

	return c.JSON(dto.ToLectureExamDTO(exam))
}

// ✏️ Update exam
func (ctrl *LectureExamController) UpdateLectureExam(c *fiber.Ctx) error {
	id := c.Params("id")
	var body dto.UpdateLectureExamRequest

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	var exam model.LectureExamModel
	if err := ctrl.DB.First(&exam, "lecture_exam_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Exam not found")
	}

	exam.LectureExamTitle = body.LectureExamTitle
	exam.LectureExamDescription = body.LectureExamDescription

	if err := ctrl.DB.Save(&exam).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update exam")
	}

	return c.JSON(dto.ToLectureExamDTO(exam))
}

// ❌ Delete exam
func (ctrl *LectureExamController) DeleteLectureExam(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.LectureExamModel{}, "lecture__exam_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete exam")
	}

	return c.JSON(fiber.Map{"message": "Exam deleted successfully"})
}
