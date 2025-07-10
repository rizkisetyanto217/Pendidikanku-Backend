package controller

import (
	"fmt"
	"masjidku_backend/internals/features/home/qoutes/dto"
	"masjidku_backend/internals/features/home/qoutes/model"
	"math"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateQuote = validator.New()

type QuoteController struct {
	DB *gorm.DB
}

func NewQuoteController(db *gorm.DB) *QuoteController {
	return &QuoteController{DB: db}
}

// =============================
// ➕ Create Quote
// =============================
func (ctrl *QuoteController) CreateQuote(c *fiber.Ctx) error {
	var req dto.CreateQuoteRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateQuote.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Cari DisplayOrder tertinggi
	var maxOrder int
	if err := ctrl.DB.Model(&model.QuoteModel{}).Select("COALESCE(MAX(display_order), 0)").Scan(&maxOrder).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get max display order")
	}

	quote := req.ToModel()
	quote.DisplayOrder = maxOrder + 1 // ✅ Auto increment

	if err := ctrl.DB.Create(quote).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create quote")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToQuoteDTO(*quote))
}

// =============================
// ➕ Create Multiple Quotes
// =============================
func (ctrl *QuoteController) CreateQuotes(c *fiber.Ctx) error {
	var reqs []dto.CreateQuoteRequest
	if err := c.BodyParser(&reqs); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Validasi setiap item
	for i, req := range reqs {
		if err := validateQuote.Struct(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Error in item "+fmt.Sprint(i+1)+": "+err.Error())
		}
	}

	// Ambil DisplayOrder tertinggi saat ini
	var maxOrder int
	if err := ctrl.DB.Model(&model.QuoteModel{}).Select("COALESCE(MAX(display_order), 0)").Scan(&maxOrder).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get max display order")
	}

	// Konversi & assign DisplayOrder
	var quotes []model.QuoteModel
	for i, req := range reqs {
		quote := req.ToModel()
		quote.DisplayOrder = maxOrder + i + 1 // ✅ Increment sesuai urutan
		quotes = append(quotes, *quote)
	}

	// Simpan dalam batch
	if err := ctrl.DB.Create(&quotes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create quotes")
	}

	// Ubah ke DTO untuk respons
	var result []dto.QuoteDTO
	for _, q := range quotes {
		result = append(result, dto.ToQuoteDTO(q))
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// =============================
// 🔄 Update Quote
// =============================
func (ctrl *QuoteController) UpdateQuote(c *fiber.Ctx) error {
	id := c.Params("id")

	var body dto.UpdateQuoteRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateQuote.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var quote model.QuoteModel
	if err := ctrl.DB.First(&quote, "quote_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Quote not found")
	}

	quote.QuoteText = body.QuoteText
	quote.IsPublished = body.IsPublished
	quote.DisplayOrder = body.DisplayOrder
	quote.CreatedAt = time.Now()

	if err := ctrl.DB.Save(&quote).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update quote")
	}

	return c.JSON(dto.ToQuoteDTO(quote))
}

// =============================
// 🗑️ Delete Quote
// =============================
func (ctrl *QuoteController) DeleteQuote(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.QuoteModel{}, "quote_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete quote")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// =============================
// 📄 Get All Quotes
// =============================
func (ctrl *QuoteController) GetAllQuotes(c *fiber.Ctx) error {
	var quotes []model.QuoteModel
	if err := ctrl.DB.Order("display_order ASC").Find(&quotes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve quotes")
	}

	var result []dto.QuoteDTO
	for _, q := range quotes {
		result = append(result, dto.ToQuoteDTO(q))
	}

	return c.JSON(result)
}

// =============================
// 🔍 Get Quote By ID
// =============================
func (ctrl *QuoteController) GetQuoteByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var quote model.QuoteModel
	if err := ctrl.DB.First(&quote, "quote_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Quote not found")
	}

	return c.JSON(dto.ToQuoteDTO(quote))
}

// =============================
// 📦 Get 30 Quotes by Batch
// =============================
func (ctrl *QuoteController) GetQuotesByBatch(c *fiber.Ctx) error {
	batchParam := c.Query("batch_number", "1")
	batchNum, err := strconv.Atoi(batchParam)
	if err != nil || batchNum < 1 {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid batch_number")
	}

	const batchSize = 30
	offset := (batchNum - 1) * batchSize

	var quotes []model.QuoteModel
	if err := ctrl.DB.Order("display_order ASC").Offset(offset).Limit(batchSize).Find(&quotes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve quotes")
	}

	var totalCount int64
	ctrl.DB.Model(&model.QuoteModel{}).Count(&totalCount)
	totalBatches := int(math.Ceil(float64(totalCount) / float64(batchSize)))

	var result []dto.QuoteDTO
	for _, q := range quotes {
		result = append(result, dto.ToQuoteDTO(q))
	}

	return c.JSON(fiber.Map{
		"batch":                 batchNum,
		"total_available_batch": totalBatches,
		"data":                  result,
	})
}
