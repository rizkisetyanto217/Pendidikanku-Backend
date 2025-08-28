package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	resp "masjidku_backend/internals/helpers"

	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/questions/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
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
// ➕ Create Question (mendukung bulk array atau single object)
// =============================
func (ctrl *LectureSessionsQuestionController) CreateLectureSessionsQuestion(c *fiber.Ctx) error {
	// ---- 1) Coba parse sebagai array (bulk) ----
	var bulkBody []dto.CreateLectureSessionsQuestionRequest
	if err := c.BodyParser(&bulkBody); err == nil && len(bulkBody) > 0 {
		// Ambil masjid_id dari token atau dari body[0]
		masjidID := ""
		if raw := c.Locals("masjid_id"); raw != nil {
			if s, ok := raw.(string); ok {
				masjidID = s
			}
		}
		if masjidID == "" && bulkBody[0].LectureSessionsQuestionMasjidID != "" {
			masjidID = bulkBody[0].LectureSessionsQuestionMasjidID
		}
		if masjidID == "" {
			return resp.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak tersedia di token maupun body")
		}

		questions := make([]model.LectureSessionsQuestionModel, 0, len(bulkBody))
		for i, b := range bulkBody {
			if err := validateLectureQuestion.Struct(b); err != nil {
				return resp.JsonError(c, fiber.StatusBadRequest, fmt.Sprintf("Validasi gagal pada soal ke-%d: %v", i+1, err.Error()))
			}
			questions = append(questions, model.LectureSessionsQuestionModel{
				LectureSessionsQuestion:            b.LectureSessionsQuestion,
				LectureSessionsQuestionAnswers:     b.LectureSessionsQuestionAnswers,
				LectureSessionsQuestionCorrect:     b.LectureSessionsQuestionCorrect,
				LectureSessionsQuestionExplanation: b.LectureSessionsQuestionExplanation,
				LectureSessionsQuestionQuizID:      b.LectureSessionsQuestionQuizID,
				LectureQuestionExamID:              b.LectureQuestionExamID,
				LectureSessionsQuestionMasjidID:    masjidID,
			})
		}

		if err := ctrl.DB.WithContext(c.Context()).Create(&questions).Error; err != nil {
			return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan soal")
		}

		out := make([]dto.LectureSessionsQuestionDTO, 0, len(questions))
		for _, q := range questions {
			out = append(out, dto.ToLectureSessionsQuestionDTO(q))
		}
		return resp.JsonCreated(c, "Questions created", out)
	}

	// ---- 2) Kalau bukan array, parse sebagai single object ----
	var body dto.CreateLectureSessionsQuestionRequest
	if err := c.BodyParser(&body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Format data tidak valid")
	}
	if err := validateLectureQuestion.Struct(body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	masjidID := ""
	if raw := c.Locals("masjid_id"); raw != nil {
		if s, ok := raw.(string); ok {
			masjidID = s
		}
	}
	if masjidID == "" && body.LectureSessionsQuestionMasjidID != "" {
		masjidID = body.LectureSessionsQuestionMasjidID
	}
	if masjidID == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak tersedia di token maupun body")
	}

	q := model.LectureSessionsQuestionModel{
		LectureSessionsQuestion:            body.LectureSessionsQuestion,
		LectureSessionsQuestionAnswers:     body.LectureSessionsQuestionAnswers,
		LectureSessionsQuestionCorrect:     body.LectureSessionsQuestionCorrect,
		LectureSessionsQuestionExplanation: body.LectureSessionsQuestionExplanation,
		LectureSessionsQuestionQuizID:      body.LectureSessionsQuestionQuizID,
		LectureQuestionExamID:              body.LectureQuestionExamID,
		LectureSessionsQuestionMasjidID:    masjidID,
	}
	if err := ctrl.DB.WithContext(c.Context()).Create(&q).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan soal")
	}

	return resp.JsonCreated(c, "Question created", dto.ToLectureSessionsQuestionDTO(q))
}

// =============================
// 📄 Get All Questions
// =============================
func (ctrl *LectureSessionsQuestionController) GetAllLectureSessionsQuestions(c *fiber.Ctx) error {
	var questions []model.LectureSessionsQuestionModel
	if err := ctrl.DB.WithContext(c.Context()).Find(&questions).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch questions")
	}

	out := make([]dto.LectureSessionsQuestionDTO, 0, len(questions))
	for _, q := range questions {
		out = append(out, dto.ToLectureSessionsQuestionDTO(q))
	}
	return resp.JsonOK(c, "OK", out)
}

// =============================
// 📚 Get Questions by Quiz ID
// =============================
func (ctrl *LectureSessionsQuestionController) GetLectureSessionsQuestionsByQuizID(c *fiber.Ctx) error {
	quizID := c.Params("quiz_id")
	if quizID == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "Quiz ID tidak boleh kosong")
	}

	var questions []model.LectureSessionsQuestionModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_question_quiz_id = ?", quizID).
		Find(&questions).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil pertanyaan")
	}

	out := make([]dto.LectureSessionsQuestionDTO, 0, len(questions))
	for _, q := range questions {
		out = append(out, dto.ToLectureSessionsQuestionDTO(q))
	}
	return resp.JsonOK(c, "OK", out)
}

// =============================
// 🔍 Get Question by ID
// =============================
func (ctrl *LectureSessionsQuestionController) GetLectureSessionsQuestionByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var q model.LectureSessionsQuestionModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&q, "lecture_sessions_question_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Question not found")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch question")
	}

	return resp.JsonOK(c, "OK", dto.ToLectureSessionsQuestionDTO(q))
}

// =============================
// ✏️ Update Question by ID (partial)
// =============================
func (ctrl *LectureSessionsQuestionController) UpdateLectureSessionsQuestionByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	// Ambil data existing
	var q model.LectureSessionsQuestionModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&q, "lecture_sessions_question_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Soal tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil soal")
	}

	// Bind request body
	var body dto.UpdateLectureSessionsQuestionDTO
	if err := c.BodyParser(&body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Format request tidak valid")
	}

	// Validasi nilai correct (jika dikirim)
	if body.LectureSessionsQuestionCorrect != nil {
		correct := *body.LectureSessionsQuestionCorrect
		if correct != "A" && correct != "B" && correct != "C" && correct != "D" {
			return resp.JsonError(c, fiber.StatusBadRequest, "Jawaban benar harus salah satu dari A, B, C, atau D")
		}
	}

	// Siapkan map update
	updates := map[string]any{}

	if body.LectureSessionsQuestion != nil {
		updates["lecture_sessions_question"] = *body.LectureSessionsQuestion
	}

	if body.LectureSessionsQuestionAnswers != nil {
		// Convert []string → JSON → datatypes.JSON
		jsonBytes, err := json.Marshal(*body.LectureSessionsQuestionAnswers)
		if err != nil {
			return resp.JsonError(c, fiber.StatusBadRequest, "Gagal memproses pilihan jawaban")
		}
		updates["lecture_sessions_question_answers"] = datatypes.JSON(jsonBytes)
	}

	if body.LectureSessionsQuestionCorrect != nil {
		updates["lecture_sessions_question_correct"] = *body.LectureSessionsQuestionCorrect
	}

	if body.LectureSessionsQuestionExplanation != nil {
		updates["lecture_sessions_question_explanation"] = *body.LectureSessionsQuestionExplanation
	}

	if len(updates) == 0 {
		// Tidak ada perubahan; kembalikan data existing
		return resp.JsonOK(c, "No changes", dto.ToLectureSessionsQuestionDTO(q))
	}

	// Eksekusi update
	if err := ctrl.DB.WithContext(c.Context()).
		Model(&q).
		Updates(updates).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui soal")
	}

	// Ambil ulang hasil setelah update
	if err := ctrl.DB.WithContext(c.Context()).
		First(&q, "lecture_sessions_question_id = ?", id).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Updated but failed to re-fetch")
	}

	return resp.JsonUpdated(c, "Soal berhasil diperbarui", dto.ToLectureSessionsQuestionDTO(q))
}

// =============================
// ❌ Delete LectureSessionsQuestion by ID
// =============================
func (ctrl *LectureSessionsQuestionController) DeleteLectureSessionsQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "id is required")
	}

	hard := strings.EqualFold(c.Query("hard"), "true") || c.Query("hard") == "1"

	tx := ctrl.DB.WithContext(c.Context())
	var db *gorm.DB
	if hard {
		db = tx.Unscoped().Delete(&model.LectureSessionsQuestionModel{}, "lecture_sessions_question_id = ?", id)
	} else {
		db = tx.Delete(&model.LectureSessionsQuestionModel{}, "lecture_sessions_question_id = ?", id)
	}

	if db.Error != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to delete question")
	}
	if db.RowsAffected == 0 {
		return resp.JsonError(c, fiber.StatusNotFound, "Question not found")
	}

	return resp.JsonDeleted(c, "Question deleted successfully", fiber.Map{
		"id":   id,
		"hard": hard,
	})
}