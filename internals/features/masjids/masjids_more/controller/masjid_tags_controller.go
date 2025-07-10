package controller

import (
	"masjidku_backend/internals/features/masjids/masjids_more/dto"
	"masjidku_backend/internals/features/masjids/masjids_more/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MasjidTagController struct {
	DB *gorm.DB
}

func NewMasjidTagController(db *gorm.DB) *MasjidTagController {
	return &MasjidTagController{DB: db}
}

// ✅ Tambah tag masjid
func (ctrl *MasjidTagController) CreateTag(c *fiber.Ctx) error {
	var body dto.MasjidTagRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Input tidak valid",
			"error":   err.Error(),
		})
	}

	tag := body.ToModel()

	if err := ctrl.DB.Create(tag).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menyimpan tag",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Tag berhasil ditambahkan",
		"data":    dto.ToMasjidTagResponse(tag),
	})
}

// ✅ Ambil semua tag
func (ctrl *MasjidTagController) GetAllTags(c *fiber.Ctx) error {
	var tags []model.MasjidTagModel
	if err := ctrl.DB.Find(&tags).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data tag",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil semua tag",
		"data":    dto.ToMasjidTagResponseList(tags),
	})
}

// ✅ Hapus tag berdasarkan ID
func (ctrl *MasjidTagController) DeleteTag(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Parameter ID wajib dikirim",
		})
	}

	if err := ctrl.DB.Where("masjid_tag_id = ?", id).Delete(&model.MasjidTagModel{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menghapus tag",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Tag berhasil dihapus",
	})
}

// ✅ Ambil beberapa tag berdasarkan daftar ID (POST JSON body)
func (ctrl *MasjidTagController) GetTagsByIDs(c *fiber.Ctx) error {
	var payload struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&payload); err != nil || len(payload.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Daftar ID wajib dikirim dalam body JSON",
			"error":   err.Error(),
		})
	}

	var tags []model.MasjidTagModel
	if err := ctrl.DB.Where("masjid_tag_id IN ?", payload.IDs).Find(&tags).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data tag",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil tag",
		"data":    dto.ToMasjidTagResponseList(tags),
	})
}
