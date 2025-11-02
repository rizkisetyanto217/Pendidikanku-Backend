package controller

import (
	"math"
	"strconv"
	"time"

	"schoolku_backend/internals/features/home/articles/dto"
	"schoolku_backend/internals/features/home/articles/model"
	helper "schoolku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validateArticle = validator.New()

type ArticleController struct {
	DB *gorm.DB
}

func NewArticleController(db *gorm.DB) *ArticleController {
	return &ArticleController{DB: db}
}

// =============================
// ➕ Create Article
// =============================
func (ctrl *ArticleController) CreateArticle(c *fiber.Ctx) error {
	var body dto.CreateArticleRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateArticle.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	article := model.ArticleModel{
		ArticleTitle:       body.ArticleTitle,
		ArticleDescription: body.ArticleDescription,
		ArticleImageURL:    body.ArticleImageURL,
		ArticleOrderID:     body.ArticleOrderID,
		// NOTE: kalau tabel articles mewajibkan article_school_id,
		// pastikan body/DTO-mu memuatnya dan set di sini.
		// ArticleSchoolID: body.ArticleSchoolID,
	}

	if err := ctrl.DB.Create(&article).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create article")
	}

	return helper.JsonCreated(c, "Article created", dto.ToArticleDTO(article))
}

// =============================
// 🔄 Update Article
// =============================
func (ctrl *ArticleController) UpdateArticle(c *fiber.Ctx) error {
	id := c.Params("id")

	var body dto.UpdateArticleRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateArticle.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var article model.ArticleModel
	if err := ctrl.DB.First(&article, "article_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Article not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve article")
	}

	article.ArticleTitle = body.ArticleTitle
	article.ArticleDescription = body.ArticleDescription
	article.ArticleImageURL = body.ArticleImageURL
	article.ArticleOrderID = body.ArticleOrderID
	article.ArticleUpdatedAt = time.Now()

	if err := ctrl.DB.Save(&article).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update article")
	}

	return helper.JsonUpdated(c, "Article updated", dto.ToArticleDTO(article))
}

// =============================
// 🗑️ Delete Article
// =============================
func (ctrl *ArticleController) DeleteArticle(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.ArticleModel{}, "article_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete article")
	}

	// 200 OK supaya bisa kirim message/body
	return helper.JsonDeleted(c, "Article deleted", fiber.Map{"article_id": id})
}

// =============================
// 📄 Get All Articles (with pagination)
// Query: ?page=1&limit=10
// =============================
func (ctrl *ArticleController) GetAllArticles(c *fiber.Ctx) error {
	page := parseIntDefault(c.Query("page"), 1)
	limit := parseIntDefault(c.Query("limit"), 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var total int64
	if err := ctrl.DB.Model(&model.ArticleModel{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to count articles")
	}

	var articles []model.ArticleModel
	if err := ctrl.DB.
		Order("article_order_id ASC").
		Limit(limit).
		Offset(offset).
		Find(&articles).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve articles")
	}

	result := make([]dto.ArticleDTO, 0, len(articles))
	for _, a := range articles {
		result = append(result, dto.ToArticleDTO(a))
	}

	pagination := fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	}

	return helper.JsonList(c, result, pagination)
}

// =============================
// 🔍 Get Article By ID
// =============================
func (ctrl *ArticleController) GetArticleByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var article model.ArticleModel
	if err := ctrl.DB.First(&article, "article_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Article not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve article")
	}

	return helper.JsonOK(c, "OK", dto.ToArticleDTO(article))
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
