package controller

import (
	"log"
	"masjidku_backend/internals/features/home/articles/dto"
	"masjidku_backend/internals/features/home/articles/model"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CarouselController struct {
	DB *gorm.DB
}

func NewCarouselController(db *gorm.DB) *CarouselController {
	return &CarouselController{
		DB: db,
	}
}

// ✅ GET: Ambil semua carousel aktif (untuk publik)
func (ctrl *CarouselController) GetAllActiveCarousels(c *fiber.Ctx) error {
	var carousels []model.CarouselModel
	err := ctrl.DB.Preload("Article").
		Where("carousel_is_active = ?", true).
		Order("CASE WHEN carousel_order IS NOT NULL THEN 0 ELSE 1 END, carousel_order ASC, carousel_created_at DESC").
		Limit(3).
		Find(&carousels).Error
	if err != nil {
		log.Println("[ERROR] Gagal ambil data carousel:", err)
		return fiber.NewError(http.StatusInternalServerError, "Gagal ambil data carousel")
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil ambil carousel",
		"data":    dto.ConvertCarouselListToDTO(carousels),
	})
}

// ✅ GET: Admin - Ambil semua carousel
func (ctrl *CarouselController) GetAllCarouselsAdmin(c *fiber.Ctx) error {
	var carousels []model.CarouselModel
	err := ctrl.DB.Preload("Article").
		Order("carousel_order").
		Find(&carousels).Error
	if err != nil {
		log.Println("[ERROR] Gagal ambil semua carousel admin:", err)
		return fiber.NewError(http.StatusInternalServerError, "Gagal ambil data")
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil ambil data",
		"data":    dto.ConvertCarouselListToDTO(carousels),
	})
}

// ✅ POST: Admin - Tambah carousel
func (ctrl *CarouselController) CreateCarousel(c *fiber.Ctx) error {
	var req model.CarouselModel
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "Input tidak valid")
	}
	req.CarouselID = uuid.New()
	req.CarouselCreatedAt = time.Now()
	req.CarouselUpdatedAt = time.Now()

	if err := ctrl.DB.Create(&req).Error; err != nil {
		log.Println("[ERROR] Gagal tambah carousel:", err)
		return fiber.NewError(http.StatusInternalServerError, "Gagal tambah data")
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Carousel berhasil ditambahkan",
		"data":    dto.ConvertCarouselToDTO(req),
	})
}

// ✅ PUT: Admin - Edit carousel
func (ctrl *CarouselController) UpdateCarousel(c *fiber.Ctx) error {
	id := c.Params("id")
	var req model.CarouselModel
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "Input tidak valid")
	}

	var existing model.CarouselModel
	if err := ctrl.DB.Where("carousel_id = ?", id).First(&existing).Error; err != nil {
		return fiber.NewError(http.StatusNotFound, "Data tidak ditemukan")
	}

	req.CarouselUpdatedAt = time.Now()
	if err := ctrl.DB.Model(&existing).Updates(req).Error; err != nil {
		log.Println("[ERROR] Gagal update carousel:", err)
		return fiber.NewError(http.StatusInternalServerError, "Gagal update data")
	}

	return c.JSON(fiber.Map{
		"message": "Carousel berhasil diupdate",
	})
}

// ✅ DELETE: Admin - Hapus carousel
func (ctrl *CarouselController) DeleteCarousel(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.DB.Where("carousel_id = ?", id).Delete(&model.CarouselModel{}).Error; err != nil {
		log.Println("[ERROR] Gagal hapus carousel:", err)
		return fiber.NewError(http.StatusInternalServerError, "Gagal hapus data")
	}

	return c.JSON(fiber.Map{
		"message": "Carousel berhasil dihapus",
	})
}
