package controller

import (
	"math"
	"strconv"

	"masjidku_backend/internals/features/home/advices/dto"
	"masjidku_backend/internals/features/home/advices/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateAdvice = validator.New()

type AdviceController struct {
	DB *gorm.DB
}

func NewAdviceController(db *gorm.DB) *AdviceController {
	return &AdviceController{DB: db}
}

// =======================
// ➕ Create Advice
// =======================
func (ctrl *AdviceController) CreateAdvice(c *fiber.Ctx) error {
	var body dto.CreateAdviceRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateAdvice.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil user_id dari token (di-set middleware)
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "User ID not found in token")
	}

	advice := model.AdviceModel{
		AdviceDescription: body.AdviceDescription,
		AdviceLectureID:   body.AdviceLectureID,
		AdviceUserID:      userID,
	}

	if err := ctrl.DB.Create(&advice).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create advice")
	}

	return helper.JsonCreated(c, "Advice created", dto.ToAdviceDTO(advice))
}

// =======================
// 📄 Get All Advices (paginated)
// Query: ?page=1&limit=10
// =======================
func (ctrl *AdviceController) GetAllAdvices(c *fiber.Ctx) error {
	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 10)
	if limit <= 0 { limit = 10 }
	if limit > 100 { limit = 100 }
	offset := (page - 1) * limit

	var total int64
	if err := ctrl.DB.Model(&model.AdviceModel{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to count advices")
	}

	var advices []model.AdviceModel
	if err := ctrl.DB.
		Order("advice_created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&advices).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve advices")
	}

	resp := make([]dto.AdviceDTO, 0, len(advices))
	for _, a := range advices {
		resp = append(resp, dto.ToAdviceDTO(a))
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	}

	return helper.JsonList(c, resp, pagination)
}

// =============================
// 🔍 Get Advices by Lecture ID (paginated)
// Path: /advices/lecture/:lectureId
// =============================
func (ctrl *AdviceController) GetAdvicesByLectureID(c *fiber.Ctx) error {
	lectureID := c.Params("lectureId")
	if lectureID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "lectureId is required")
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 10)
	if limit <= 0 { limit = 10 }
	if limit > 100 { limit = 100 }
	offset := (page - 1) * limit

	var total int64
	if err := ctrl.DB.
		Model(&model.AdviceModel{}).
		Where("advice_lecture_id = ?", lectureID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to count advices")
	}

	var advices []model.AdviceModel
	if err := ctrl.DB.
		Where("advice_lecture_id = ?", lectureID).
		Order("advice_created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&advices).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch advices")
	}

	resp := make([]dto.AdviceDTO, 0, len(advices))
	for _, a := range advices {
		resp = append(resp, dto.ToAdviceDTO(a))
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	}

	return helper.JsonList(c, resp, pagination)
}

// =============================
// 🔍 Get Advices by User ID (paginated)
// Path: /advices/user/:userId
// =============================
func (ctrl *AdviceController) GetAdvicesByUserID(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "userId is required")
	}

	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 10)
	if limit <= 0 { limit = 10 }
	if limit > 100 { limit = 100 }
	offset := (page - 1) * limit

	var total int64
	if err := ctrl.DB.
		Model(&model.AdviceModel{}).
		Where("advice_user_id = ?", userID).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to count advices")
	}

	var advices []model.AdviceModel
	if err := ctrl.DB.
		Where("advice_user_id = ?", userID).
		Order("advice_created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&advices).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch advices")
	}

	resp := make([]dto.AdviceDTO, 0, len(advices))
	for _, a := range advices {
		resp = append(resp, dto.ToAdviceDTO(a))
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	}

	return helper.JsonList(c, resp, pagination)
}

// =============================
// 🗑️ Delete Advice by ID
// =============================
func (ctrl *AdviceController) DeleteAdvice(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "id is required")
	}

	if err := ctrl.DB.Delete(&model.AdviceModel{}, "advice_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete advice")
	}

	return helper.JsonDeleted(c, "Advice deleted successfully", fiber.Map{
		"advice_id": id,
	})
}

// =============================
// utils
// =============================
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
