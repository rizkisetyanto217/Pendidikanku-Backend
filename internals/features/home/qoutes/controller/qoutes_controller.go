package controller

import (
	"fmt"
	"math"
	"strconv"

	"schoolku_backend/internals/features/home/qoutes/dto"
	"schoolku_backend/internals/features/home/qoutes/model"
	helpers "schoolku_backend/internals/helpers"

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
		return helpers.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateQuote.Struct(&req); err != nil {
		return helpers.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil MAX(order) hanya baris aktif (soft delete di-skip default)
	var maxOrder int
	if err := ctrl.DB.Model(&model.QuoteModel{}).
		Select("COALESCE(MAX(quote_display_order), 0)").
		Scan(&maxOrder).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to get max display order")
	}

	quote := req.ToModel()

	// Jika client tidak kirim order → set auto increment
	if quote.QuoteDisplayOrder == nil {
		next := maxOrder + 1
		quote.QuoteDisplayOrder = &next
	}

	if err := ctrl.DB.Create(quote).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to create quote")
	}

	return helpers.JsonCreated(c, "Quote created", dto.ToQuoteDTO(*quote))
}

// =============================
// ➕ Create Multiple Quotes
// =============================
func (ctrl *QuoteController) CreateQuotes(c *fiber.Ctx) error {
	var reqs []dto.CreateQuoteRequest
	if err := c.BodyParser(&reqs); err != nil {
		return helpers.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if len(reqs) == 0 {
		return helpers.JsonError(c, fiber.StatusBadRequest, "Payload must be a non-empty array")
	}
	for i, r := range reqs {
		if err := validateQuote.Struct(&r); err != nil {
			return helpers.JsonError(c, fiber.StatusBadRequest, "Error in item "+fmt.Sprint(i+1)+": "+err.Error())
		}
	}

	// Base MAX(order)
	var maxOrder int
	if err := ctrl.DB.Model(&model.QuoteModel{}).
		Select("COALESCE(MAX(quote_display_order), 0)").
		Scan(&maxOrder).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to get max display order")
	}

	quotes := make([]model.QuoteModel, 0, len(reqs))
	next := maxOrder
	for _, r := range reqs {
		q := r.ToModel()
		if q.QuoteDisplayOrder == nil { // hanya auto jika kosong
			next++
			q.QuoteDisplayOrder = &next
		}
		quotes = append(quotes, *q)
	}

	if err := ctrl.DB.Create(&quotes).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to create quotes")
	}

	out := make([]dto.QuoteDTO, 0, len(quotes))
	for _, q := range quotes {
		out = append(out, dto.ToQuoteDTO(q))
	}
	return helpers.JsonCreated(c, "Quotes created", out)
}

// =============================
// 🔄 Update Quote (partial)
// =============================
func (ctrl *QuoteController) UpdateQuote(c *fiber.Ctx) error {
	id := c.Params("id")

	var body dto.UpdateQuoteRequest
	if err := c.BodyParser(&body); err != nil {
		return helpers.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateQuote.Struct(&body); err != nil {
		return helpers.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var quote model.QuoteModel
	if err := ctrl.DB.First(&quote, "quote_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.JsonError(c, fiber.StatusNotFound, "Quote not found")
		}
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to get quote")
	}

	// Partial apply
	if body.QuoteText != nil {
		quote.QuoteText = *body.QuoteText
	}
	if body.QuoteIsPublished != nil {
		quote.QuoteIsPublished = *body.QuoteIsPublished
	}
	if body.QuoteDisplayOrder != nil {
		// bisa set ke nilai tertentu, atau clear ke NULL jika client kirim null
		quote.QuoteDisplayOrder = body.QuoteDisplayOrder
	}

	if err := ctrl.DB.Save(&quote).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to update quote")
	}

	return helpers.JsonUpdated(c, "Quote updated", dto.ToQuoteDTO(quote))
}

// =============================
// 🗑️ Delete Quote (soft delete)
// =============================
func (ctrl *QuoteController) DeleteQuote(c *fiber.Ctx) error {
	id := c.Params("id")

	// Opsional: cepat cek eksistensi
	var exists int64
	if err := ctrl.DB.Model(&model.QuoteModel{}).
		Where("quote_id = ?", id).
		Count(&exists).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to delete quote")
	}
	if exists == 0 {
		return helpers.JsonError(c, fiber.StatusNotFound, "Quote not found")
	}

	if err := ctrl.DB.Delete(&model.QuoteModel{}, "quote_id = ?", id).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to delete quote")
	}
	return helpers.JsonDeleted(c, "Quote deleted", fiber.Map{"id": id})
}

// =============================
// 📄 Get All Quotes
// =============================
func (ctrl *QuoteController) GetAllQuotes(c *fiber.Ctx) error {
	var quotes []model.QuoteModel
	if err := ctrl.DB.
		Order("quote_display_order ASC NULLS LAST, quote_created_at ASC").
		Find(&quotes).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve quotes")
	}

	result := make([]dto.QuoteDTO, 0, len(quotes))
	for _, q := range quotes {
		result = append(result, dto.ToQuoteDTO(q))
	}

	// simple pagination (full list)
	pagination := fiber.Map{
		"page":        1,
		"page_size":   len(result),
		"total_data":  len(result),
		"total_pages": 1,
	}
	return helpers.JsonList(c, result, pagination)
}

// =============================
// 🔍 Get Quote By ID
// =============================
func (ctrl *QuoteController) GetQuoteByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var quote model.QuoteModel
	if err := ctrl.DB.First(&quote, "quote_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.JsonError(c, fiber.StatusNotFound, "Quote not found")
		}
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve quote")
	}

	return helpers.JsonOK(c, "OK", dto.ToQuoteDTO(quote))
}

// =============================
// 📦 Get 30 Quotes by Batch
// =============================
func (ctrl *QuoteController) GetQuotesByBatch(c *fiber.Ctx) error {
	batchParam := c.Query("batch_number", "1")
	batchNum, err := strconv.Atoi(batchParam)
	if err != nil || batchNum < 1 {
		return helpers.JsonError(c, fiber.StatusBadRequest, "Invalid batch_number")
	}

	const batchSize = 30
	offset := (batchNum - 1) * batchSize

	var totalCount int64
	if err := ctrl.DB.Model(&model.QuoteModel{}).Count(&totalCount).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to count quotes")
	}
	totalBatches := int(math.Ceil(float64(totalCount) / float64(batchSize)))

	var quotes []model.QuoteModel
	if err := ctrl.DB.
		Order("quote_display_order ASC NULLS LAST, quote_created_at ASC").
		Offset(offset).Limit(batchSize).
		Find(&quotes).Error; err != nil {
		return helpers.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve quotes")
	}

	result := make([]dto.QuoteDTO, 0, len(quotes))
	for _, q := range quotes {
		result = append(result, dto.ToQuoteDTO(q))
	}

	pagination := fiber.Map{
		"page":        batchNum,
		"page_size":   batchSize,
		"total_data":  totalCount,
		"total_pages": totalBatches,
		"has_next":    batchNum < totalBatches,
		"has_prev":    batchNum > 1,
		"next_page": func() int {
			if batchNum < totalBatches {
				return batchNum + 1
			}
			return batchNum
		}(),
		"prev_page": func() int {
			if batchNum > 1 {
				return batchNum - 1
			}
			return batchNum
		}(),
		"total_available_batch": totalBatches,
	}
	return helpers.JsonList(c, result, pagination)
}
