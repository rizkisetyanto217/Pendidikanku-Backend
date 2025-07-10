package controller

import (
	"time"

	"masjidku_backend/internals/features/home/articles/dto"
	"masjidku_backend/internals/features/home/articles/model"

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
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateArticle.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	article := model.ArticleModel{
		ArticleTitle:       body.ArticleTitle,
		ArticleDescription: body.ArticleDescription,
		ArticleImageURL:    body.ArticleImageURL,
		ArticleOrderID:     body.ArticleOrderID,
	}

	if err := ctrl.DB.Create(&article).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create article")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ToArticleDTO(article))
}

// =============================
// 🔄 Update Article
// =============================
func (ctrl *ArticleController) UpdateArticle(c *fiber.Ctx) error {
	id := c.Params("id")

	var body dto.UpdateArticleRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}
	if err := validateArticle.Struct(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var article model.ArticleModel
	if err := ctrl.DB.First(&article, "article_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Article not found")
	}

	article.ArticleTitle = body.ArticleTitle
	article.ArticleDescription = body.ArticleDescription
	article.ArticleImageURL = body.ArticleImageURL
	article.ArticleOrderID = body.ArticleOrderID
	article.ArticleUpdatedAt = time.Now()

	if err := ctrl.DB.Save(&article).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to update article")
	}

	return c.JSON(dto.ToArticleDTO(article))
}

// =============================
// 🗑️ Delete Article
// =============================
func (ctrl *ArticleController) DeleteArticle(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := ctrl.DB.Delete(&model.ArticleModel{}, "article_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete article")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// =============================
// 📄 Get All Articles
// =============================
func (ctrl *ArticleController) GetAllArticles(c *fiber.Ctx) error {
	var articles []model.ArticleModel
	if err := ctrl.DB.Order("article_order_id ASC").Find(&articles).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve articles")
	}

	var result []dto.ArticleDTO
	for _, a := range articles {
		result = append(result, dto.ToArticleDTO(a))
	}

	return c.JSON(result)
}

// =============================
// 🔍 Get Article By ID
// =============================
func (ctrl *ArticleController) GetArticleByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var article model.ArticleModel
	if err := ctrl.DB.First(&article, "article_id = ?", id).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Article not found")
	}

	return c.JSON(dto.ToArticleDTO(article))
}
