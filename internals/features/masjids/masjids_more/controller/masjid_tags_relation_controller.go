package controller

import (
	"masjidku_backend/internals/features/masjids/masjids_more/dto"
	"masjidku_backend/internals/features/masjids/masjids_more/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MasjidTagRelationController struct {
	DB *gorm.DB
}

func NewMasjidTagRelationController(db *gorm.DB) *MasjidTagRelationController {
	return &MasjidTagRelationController{DB: db}
}

// ✅ Tambah relasi tag ke masjid
func (ctrl *MasjidTagRelationController) CreateRelation(c *fiber.Ctx) error {
	var body dto.MasjidTagRelationRequest
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Input tidak valid",
			"error":   err.Error(),
		})
	}

	model := body.ToModel()
	if err := ctrl.DB.Create(model).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal menyimpan relasi tag",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Relasi tag berhasil ditambahkan",
		"data":    dto.ToMasjidTagRelationResponse(model),
	})
}

// ✅ Ambil semua tag yang terkait dengan masjid tertentu
func (ctrl *MasjidTagRelationController) GetTagsByMasjid(c *fiber.Ctx) error {
	masjidID := c.Query("masjid_id")
	if masjidID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "masjid_id wajib dikirim sebagai query parameter",
		})
	}

	var relations []model.MasjidTagRelationModel
	if err := ctrl.DB.
		Preload("MasjidTag").
		Where("masjid_tag_relation_masjid_id = ?", masjidID).
		Find(&relations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Gagal mengambil data relasi tag",
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil mengambil relasi tag masjid",
		"data":    dto.ToMasjidTagRelationResponseList(relations),
	})
}
