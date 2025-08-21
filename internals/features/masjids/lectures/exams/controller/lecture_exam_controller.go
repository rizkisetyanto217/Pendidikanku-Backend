package controller

import (
	modelLectureSessionQuestion "masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"
	"masjidku_backend/internals/features/masjids/lectures/exams/dto"
	"masjidku_backend/internals/features/masjids/lectures/exams/model"
	helper "masjidku_backend/internals/helpers"
	"strings"

	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureExamController struct {
	DB *gorm.DB
}

func NewLectureExamController(db *gorm.DB) *LectureExamController {
	return &LectureExamController{DB: db}
}

// ➕ POST /api/a/lecture-exams
func (ctrl *LectureExamController) CreateLectureExam(c *fiber.Ctx) error {
	// Ambil masjid_id dari token (middleware harus set)
	masjidID := c.Locals("masjid_id")
	if masjidID == nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID not found in token")
	}

	var body dto.CreateLectureExamRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	newExam := model.LectureExamModel{
		LectureExamTitle:       body.LectureExamTitle,
		LectureExamDescription: body.LectureExamDescription,
		LectureExamLectureID:   body.LectureExamLectureID,
		LectureExamMasjidID:    masjidID.(string),
	}

	if err := ctrl.DB.Create(&newExam).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create exam")
	}

	return helper.JsonCreated(c, "Exam created successfully", dto.ToLectureExamDTO(newExam))
}

// 📄 GET /api/a/lecture-exams (support pagination ?page=&page_size=)
func (ctrl *LectureExamController) GetAllLectureExams(c *fiber.Ctx) error {
	// pagination ringan
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := ctrl.DB.Model(&model.LectureExamModel{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to count exams")
	}

	var exams []model.LectureExamModel
	if err := ctrl.DB.
		Order("lecture_exam_created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&exams).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve exams")
	}

	resp := make([]dto.LectureExamDTO, len(exams))
	for i, e := range exams {
		resp[i] = dto.ToLectureExamDTO(e)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	pagination := fiber.Map{
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
	}

	return helper.JsonList(c, resp, pagination)
}

// 📄 GET /api/u/lecture-exams/:id/with-questions
func (ctrl *LectureExamController) GetLectureExamWithQuestions(c *fiber.Ctx) error {
	examID := c.Params("id")
	if examID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Exam ID is required")
	}

	var exam model.LectureExamModel
	if err := ctrl.DB.First(&exam, "lecture_exam_id = ?", examID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Exam not found")
	}

	var questions []modelLectureSessionQuestion.LectureSessionsQuestionModel
	if err := ctrl.DB.
		Where("lecture_question_exam_id = ?", examID).
		Order("lecture_sessions_question_created_at ASC").
		Find(&questions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch questions")
	}

	data := fiber.Map{
		"exam":      dto.ToLectureExamDTO(exam),
		"questions": questions, // ganti ke DTO kalau kamu sudah punya DTO untuk questions
	}
	return helper.JsonOK(c, "Exam & questions fetched successfully", data)
}

// 🔍 GET /api/u/lecture-exams/:id
func (ctrl *LectureExamController) GetLectureExamByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Exam ID is required")
	}

	var exam model.LectureExamModel
	if err := ctrl.DB.First(&exam, "lecture_exam_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Exam not found")
	}

	return helper.JsonOK(c, "Exam fetched successfully", dto.ToLectureExamDTO(exam))
}

// 📄 GET /api/a/lecture-exams/questions/by-lecture/:id
func (ctrl *LectureExamController) GetQuestionExamByLectureID(c *fiber.Ctx) error {
	lectureID := c.Params("id")
	if lectureID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Lecture ID is required")
	}

	var exams []model.LectureExamModel
	if err := ctrl.DB.
		Where("lecture_exam_lecture_id = ?", lectureID).
		Find(&exams).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to get exams by lecture ID")
	}

	if len(exams) == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "No exams found for this lecture")
	}

	// kumpulkan exam_id
	examIDs := make([]string, 0, len(exams))
	for _, e := range exams {
		examIDs = append(examIDs, e.LectureExamID)
	}

	var questions []modelLectureSessionQuestion.LectureSessionsQuestionModel
	if err := ctrl.DB.
		Where("lecture_question_exam_id IN ?", examIDs).
		Order("lecture_sessions_question_created_at ASC").
		Find(&questions).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to get questions for exams")
	}

	data := fiber.Map{
		"exam_count": len(exams),
		"exam_id":    exams[0].LectureExamID, // jika perlu id pertama
		"questions":  questions,              // ganti ke DTO jika tersedia
	}
	return helper.JsonOK(c, "Questions fetched by lecture successfully", data)
}

// ✏️ PATCH /api/a/lecture-exams/:id
func (ctrl *LectureExamController) UpdateLectureExam(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Exam ID is required")
	}

	var body dto.UpdateLectureExamRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	var exam model.LectureExamModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&exam, "lecture_exam_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Exam not found")
	}

	// Partial update
	if body.LectureExamTitle != "" {
		exam.LectureExamTitle = body.LectureExamTitle
	}
	if body.LectureExamDescription != nil { // <-- cek pointer dulu
		// kalau mau boleh kosongkan kolom saat kirim "":
		desc := strings.TrimSpace(*body.LectureExamDescription)
		if desc == "" {
			exam.LectureExamDescription = nil
		} else {
			exam.LectureExamDescription = &desc
		}
	}

	if err := ctrl.DB.WithContext(c.Context()).Save(&exam).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update exam")
	}

	return helper.JsonUpdated(c, "Exam updated successfully", dto.ToLectureExamDTO(exam))
}


// ❌ DELETE /api/a/lecture-exams/:id
func (ctrl *LectureExamController) DeleteLectureExam(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Exam ID is required")
	}

	// fix: kolom yang benar "lecture_exam_id" (bukan lecture__exam_id)
	if err := ctrl.DB.Delete(&model.LectureExamModel{}, "lecture_exam_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete exam")
	}

	return helper.JsonDeleted(c, "Exam deleted successfully", fiber.Map{"lecture_exam_id": id})
}
