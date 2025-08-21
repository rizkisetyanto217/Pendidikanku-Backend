package controller

import (
	"errors"

	"masjidku_backend/internals/features/home/faqs/dto"
	"masjidku_backend/internals/features/home/faqs/model"
	resp "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FaqQuestionController struct {
	DB *gorm.DB
}

func NewFaqQuestionController(db *gorm.DB) *FaqQuestionController {
	return &FaqQuestionController{DB: db}
}

// ======================
// Create FaqQuestion
// ======================
func (ctrl *FaqQuestionController) CreateFaqQuestion(c *fiber.Ctx) error {
	var body dto.CreateFaqQuestionRequest
	if err := c.BodyParser(&body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Ambil user_id dari token (wajib)
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "User not authenticated")
	}
	if _, err := uuid.Parse(userID); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "user_id pada token tidak valid")
	}

	newFaq := body.ToModel(userID)

	if err := ctrl.DB.WithContext(c.Context()).Create(&newFaq).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to create FAQ")
	}

	return resp.JsonCreated(c, "FAQ created", dto.ToFaqQuestionDTO(newFaq))
}

// ======================
// Get All FaqQuestions
// ======================
func (ctrl *FaqQuestionController) GetAllFaqQuestions(c *fiber.Ctx) error {
	var faqs []model.FaqQuestionModel
	if err := ctrl.DB.WithContext(c.Context()).
		Preload("FaqAnswers").
		Find(&faqs).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve FAQs")
	}

	result := make([]dto.FaqQuestionDTO, 0, len(faqs))
	for _, f := range faqs {
		result = append(result, dto.ToFaqQuestionDTO(f))
	}
	return resp.JsonOK(c, "OK", result)
}

// ======================
// Get FaqQuestion by ID
// ======================
func (ctrl *FaqQuestionController) GetFaqQuestionByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var faq model.FaqQuestionModel
	if err := ctrl.DB.WithContext(c.Context()).
		Preload("FaqAnswers").
		First(&faq, "faq_question_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "FAQ not found")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to get FAQ")
	}

	return resp.JsonOK(c, "OK", dto.ToFaqQuestionDTO(faq))
}

// ======================
// Update FaqQuestion (partial)
// ======================
// ======================
// Update FaqQuestion (partial, aman untuk *string)
// ======================
func (ctrl *FaqQuestionController) UpdateFaqQuestion(c *fiber.Ctx) error {
    id := c.Params("id")
    if _, err := uuid.Parse(id); err != nil {
        return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
    }

    var body dto.UpdateFaqQuestionRequest
    if err := c.BodyParser(&body); err != nil {
        return resp.JsonError(c, fiber.StatusBadRequest, "Invalid request")
    }

    var faq model.FaqQuestionModel
    if err := ctrl.DB.WithContext(c.Context()).
        First(&faq, "faq_question_id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return resp.JsonError(c, fiber.StatusNotFound, "FAQ not found")
        }
        return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to get FAQ")
    }

    // Kumpulkan field yang benar-benar diupdate
    updates := map[string]any{}

    if body.FaqQuestionText != "" {
        updates["faq_question_text"] = body.FaqQuestionText
    }

    if body.FaqQuestionLectureID != nil { // dikirim di JSON (bisa "" untuk clear)
        if *body.FaqQuestionLectureID == "" {
            updates["faq_question_lecture_id"] = nil // clear kolom
        } else {
            if _, err := uuid.Parse(*body.FaqQuestionLectureID); err != nil {
                return resp.JsonError(c, fiber.StatusBadRequest, "Lecture ID tidak valid")
            }
            updates["faq_question_lecture_id"] = *body.FaqQuestionLectureID
        }
    }

    if body.FaqQuestionLectureSessionID != nil {
        if *body.FaqQuestionLectureSessionID == "" {
            updates["faq_question_lecture_session_id"] = nil
        } else {
            if _, err := uuid.Parse(*body.FaqQuestionLectureSessionID); err != nil {
                return resp.JsonError(c, fiber.StatusBadRequest, "Lecture Session ID tidak valid")
            }
            updates["faq_question_lecture_session_id"] = *body.FaqQuestionLectureSessionID
        }
    }

    if body.FaqQuestionIsAnswered != nil {
        updates["faq_question_is_answered"] = *body.FaqQuestionIsAnswered
    }

    if len(updates) == 0 {
        // tidak ada perubahan — kembalikan data lama
        return resp.JsonOK(c, "Tidak ada perubahan", dto.ToFaqQuestionDTO(faq))
    }

    if err := ctrl.DB.WithContext(c.Context()).
        Model(&faq).Updates(updates).Error; err != nil {
        return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to update FAQ")
    }

    // refresh struct 'faq' supaya DTO menampilkan nilai terbaru
    if err := ctrl.DB.WithContext(c.Context()).
        Preload("FaqAnswers").
        First(&faq, "faq_question_id = ?", id).Error; err != nil {
        return resp.JsonError(c, fiber.StatusInternalServerError, "Updated but failed to re-fetch FAQ")
    }

    return resp.JsonUpdated(c, "FAQ updated", dto.ToFaqQuestionDTO(faq))
}


// ======================
// Delete FaqQuestion
// ======================
func (ctrl *FaqQuestionController) DeleteFaqQuestion(c *fiber.Ctx) error {
	id := c.Params("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Delete(&model.FaqQuestionModel{}, "faq_question_id = ?", uid).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to delete FAQ")
	}
	return resp.JsonDeleted(c, "FAQ deleted", fiber.Map{"id": uid})
}
