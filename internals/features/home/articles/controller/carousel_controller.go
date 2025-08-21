package controller

import (
	"log"
	"net/http"
	"time"

	helper "masjidku_backend/internals/helpers"

	"masjidku_backend/internals/features/home/articles/dto"
	"masjidku_backend/internals/features/home/articles/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CarouselController struct {
	DB *gorm.DB
}

func NewCarouselController(db *gorm.DB) *CarouselController {
	return &CarouselController{DB: db}
}

// ✅ GET: Ambil semua carousel aktif (untuk publik)
func (ctrl *CarouselController) GetAllActiveCarousels(c *fiber.Ctx) error {
	var carousels []model.CarouselModel
	if err := ctrl.DB.Preload("Article").
		Where("carousel_is_active = ?", true).
		Order("CASE WHEN carousel_order IS NOT NULL THEN 0 ELSE 1 END, carousel_order ASC, carousel_created_at DESC").
		Limit(3).
		Find(&carousels).Error; err != nil {
		log.Println("[ERROR] Gagal ambil data carousel:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal ambil data carousel")
	}

	return helper.JsonList(c, dto.ConvertCarouselListToDTO(carousels), nil)
}

// ✅ GET: Admin - Ambil semua carousel
func (ctrl *CarouselController) GetAllCarouselsAdmin(c *fiber.Ctx) error {
	var carousels []model.CarouselModel
	if err := ctrl.DB.Preload("Article").
		Order("carousel_order").
		Find(&carousels).Error; err != nil {
		log.Println("[ERROR] Gagal ambil semua carousel admin:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal ambil data")
	}

	return helper.JsonList(c, dto.ConvertCarouselListToDTO(carousels), nil)
}

// ✅ POST: Admin - Tambah carousel
func (ctrl *CarouselController) CreateCarousel(c *fiber.Ctx) error {
	var req model.CarouselModel
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Input tidak valid")
	}

	req.CarouselID = uuid.New()
	req.CarouselCreatedAt = time.Now()
	req.CarouselUpdatedAt = time.Now()

	if err := ctrl.DB.Create(&req).Error; err != nil {
		log.Println("[ERROR] Gagal tambah carousel:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal tambah data")
	}

	return helper.JsonCreated(c, "Carousel berhasil ditambahkan", dto.ConvertCarouselToDTO(req))
}

// ✅ PUT: Admin - Edit carousel
func (ctrl *CarouselController) UpdateCarousel(c *fiber.Ctx) error {
	id := c.Params("id")

	var existing model.CarouselModel
	if err := ctrl.DB.Where("carousel_id = ?", id).First(&existing).Error; err != nil {
		return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
	}

	var req model.CarouselModel
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Input tidak valid")
	}

	req.CarouselUpdatedAt = time.Now()

	// Hindari overwrite field yang tidak seharusnya
	if err := ctrl.DB.Model(&existing).
		Omit("carousel_id", "carousel_created_at").
		Updates(req).Error; err != nil {
		log.Println("[ERROR] Gagal update carousel:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal update data")
	}

	return helper.JsonUpdated(c, "Carousel berhasil diupdate", nil)
}

// ✅ DELETE: Admin - Hapus carousel
func (ctrl *CarouselController) DeleteCarousel(c *fiber.Ctx) error {
	id := c.Params("id")

	// optional: cek eksistensi
	var exists int64
	if err := ctrl.DB.Model(&model.CarouselModel{}).
		Where("carousel_id = ?", id).
		Count(&exists).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal hapus data")
	}
	if exists == 0 {
		return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
	}

	if err := ctrl.DB.Where("carousel_id = ?", id).
		Delete(&model.CarouselModel{}).Error; err != nil {
		log.Println("[ERROR] Gagal hapus carousel:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal hapus data")
	}

	return helper.JsonDeleted(c, "Carousel berhasil dihapus", fiber.Map{"id": id})
}
