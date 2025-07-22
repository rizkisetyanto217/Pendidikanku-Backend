package controller

import (
	"fmt"
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateLectureQuestion = validator.New()

type LectureSessionsQuestionController struct {
	DB *gorm.DB
}

func NewLectureSessionsQuestionController(db *gorm.DB) *LectureSessionsQuestionController {
	return &LectureSessionsQuestionController{DB: db}
}

// =============================
// ➕ Create Question
// =============================
// =============================
// ➕ Create Question
// =============================
func (ctrl *LectureSessionsQuestionController) CreateLectureSessionsQuestion(c *fiber.Ctx) error {
	// 1. Coba parse sebagai array (bulk)
	var bulkBody []dto.CreateLectureSessionsQuestionRequest
	if err := c.BodyParser(&bulkBody); err == nil && len(bulkBody) > 0 {
		// Ambil masjid_id dari token atau body
		masjidID := ""
		if raw := c.Locals("masjid_id"); raw != nil {
			masjidID = raw.(string)
		} else if bulkBody[0].LectureSessionsQuestionMasjidID != "" {
			masjidID = bulkBody[0].LectureSessionsQuestionMasjidID
		} else {
			return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak tersedia di token maupun body")
		}

		var questions []model.LectureSessionsQuestionModel
		for i, body := range bulkBody {
			if err := validateLectureQuestion.Struct(body); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Validasi gagal pada soal ke-%d: %v", i+1, err.Error()))
			}
			questions = append(questions, model.LectureSessionsQuestionModel{
				LectureSessionsQuestion:            body.LectureSessionsQuestion,
				LectureSessionsQuestionAnswers:     body.LectureSessionsQuestionAnswers,
				LectureSessionsQuestionCorrect:     body.LectureSessionsQuestionCorrect,
				LectureSessionsQuestionExplanation: body.LectureSessionsQuestionExplanation,
				LectureSessionsQuestionQuizID:      body.LectureSessionsQuestionQuizID,
				LectureSessionsQuestionExamID:      body.LectureSessionsQuestionExamID,
				LectureSessionsQuestionMasjidID:    masjidID,
			})
		}

		if err := ctrl.DB.Create(&questions).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan soal")
		}

		var response []dto.LectureSessionsQuestionDTO
		for _, q := range questions {
			response = append(response, dto.ToLectureSessionsQuestionDTO(q))
		}
		return c.Status(fiber.StatusCreated).JSON(response)
	}

	// 2. Kalau bukan array, coba parse sebagai satuan
	var body dto.CreateLectureSessionsQuestionRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Format data tidak valid")
	}
	if err := validateLectureQuestion.Struct(body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil masjid_id dari token atau body
	masjidID := ""
	if raw := c.Locals("masjid_id"); raw != nil {
		masjidID = raw.(string)
	} else if body.LectureSessionsQuestionMasjidID != "" {
		masjidID = body.LectureSessionsQuestionMasjidID
	} else {
		return fiber.NewError(fiber.StatusUnauthorized, "Masjid ID tidak tersedia di token maupun body")
	}

	question := model.LectureSessionsQuestionModel{
		LectureSessionsQuestion:            body.LectureSessionsQuestion,
		LectureSessionsQuestionAnswers:     body.LectureSessionsQuestionAnswers,
		LectureSessionsQuestionCorrect:     body.LectureSessionsQuestionCorrect,
		LectureSessionsQuestionExplanation: body.LectureSessionsQuestionExplanation,
		LectureSessionsQuestionQuizID:      body.LectureSessionsQuestionQuizID,
		LectureSessionsQuestionExamID:      body.LectureSessionsQuestionExamID,
		LectureSessionsQuestionMasjidID:    masjidID,
	}

	if err := ctrl.DB.Create(&question).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan soal")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToLectureSessionsQuestionDTO(question))
}



// =============================
// 📄 Get All Questions
// =============================
func (ctrl *LectureSessionsQuestionController) GetAllLectureSessionsQuestions(c *fiber.Ctx) error {
	var questions []model.LectureSessionsQuestionModel

	if err := ctrl.DB.Find(&questions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to fetch questions")
	}

	var response []dto.LectureSessionsQuestionDTO
	for _, q := range questions {
		response = append(response, dto.ToLectureSessionsQuestionDTO(q))
	}

	return c.JSON(response)
}


// =============================
// 📚 Get Questions by Quiz ID
// =============================
func (ctrl *LectureSessionsQuestionController) GetLectureSessionsQuestionsByQuizID(c *fiber.Ctx) error {
	quizID := c.Params("quiz_id")
	if quizID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Quiz ID tidak boleh kosong")
	}

	var questions []model.LectureSessionsQuestionModel
	if err := ctrl.DB.
		Where("lecture_sessions_question_quiz_id = ?", quizID).
		Find(&questions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil pertanyaan")
	}

	var response []dto.LectureSessionsQuestionDTO
	for _, q := range questions {
		response = append(response, dto.ToLectureSessionsQuestionDTO(q))
	}

	return c.JSON(response)
}


// =============================
// 🔍 Get Question by ID
// =============================
func (ctrl *LectureSessionsQuestionController) GetLectureSessionsQuestionByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var question model.LectureSessionsQuestionModel
	if err := ctrl.DB.First(&question, "lecture_sessions_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Question not found")
	}

	return c.JSON(dto.ToLectureSessionsQuestionDTO(question))
}

// =============================
// ✏️ Update Question by ID (Partial)
// =============================
func (ctrl *LectureSessionsQuestionController) UpdateLectureSessionsQuestionByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	// Ambil data existing
	var question model.LectureSessionsQuestionModel
	if err := ctrl.DB.First(&question, "lecture_sessions_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Soal tidak ditemukan")
	}

	// Bind request body
	var body dto.UpdateLectureSessionsQuestionDTO
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Format request tidak valid")
	}

	// Validasi nilai correct (jika dikirim)
	if body.LectureSessionsQuestionCorrect != nil {
		correct := *body.LectureSessionsQuestionCorrect
		if correct != "A" && correct != "B" && correct != "C" && correct != "D" {
			return fiber.NewError(fiber.StatusBadRequest, "Jawaban benar harus salah satu dari A, B, C, atau D")
		}
	}

	// Siapkan map update
	updates := map[string]interface{}{}
	if body.LectureSessionsQuestion != nil {
		updates["lecture_sessions_question"] = *body.LectureSessionsQuestion
	}
	if body.LectureSessionsQuestionAnswers != nil {
		updates["lecture_sessions_question_answers"] = *body.LectureSessionsQuestionAnswers
	}
	if body.LectureSessionsQuestionCorrect != nil {
		updates["lecture_sessions_question_correct"] = *body.LectureSessionsQuestionCorrect
	}
	if body.LectureSessionsQuestionExplanation != nil {
		updates["lecture_sessions_question_explanation"] = *body.LectureSessionsQuestionExplanation
	}

	// Lakukan update hanya jika ada field yang dikirim
	if len(updates) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Tidak ada field yang dikirim untuk diperbarui")
	}
	if err := ctrl.DB.Model(&question).Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui soal")
	}

	// Ambil ulang data yang telah diperbarui
	if err := ctrl.DB.First(&question, "lecture_sessions_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data setelah update")
	}

	return c.JSON(dto.ToLectureSessionsQuestionDTO(question))
}


// =============================
// ❌ Delete Question by ID
// =============================
func (ctrl *LectureSessionsQuestionController) DeleteLectureSessionsQuestion(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.LectureSessionsQuestionModel{}, "lecture_sessions_question_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete question")
	}

	return c.JSON(fiber.Map{
	"message": "Question deleted successfully",
	})
}
