package controller

import (
	"errors"
	"strings"

	"masjidku_backend/internals/features/masjids/masjids_more/dto"
	"masjidku_backend/internals/features/masjids/masjids_more/model"
	helper "masjidku_backend/internals/helpers"

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
		return helper.JsonError(c, fiber.StatusBadRequest, "Input tidak valid")
	}

	tag := body.ToModel() // asumsi: *model.MasjidTagModel

	if err := ctrl.DB.WithContext(c.Context()).Create(tag).Error; err != nil {
		// duplikasi: unique lower(name)
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return helper.JsonError(c, fiber.StatusConflict, "Tag sudah ada")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan tag")
	}

	return helper.JsonCreated(c, "Tag berhasil ditambahkan", dto.ToMasjidTagResponse(tag))
}

// ✅ Ambil semua tag
func (ctrl *MasjidTagController) GetAllTags(c *fiber.Ctx) error {
	var tags []model.MasjidTagModel
	if err := ctrl.DB.WithContext(c.Context()).
		Order("masjid_tag_created_at DESC").
		Find(&tags).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data tag")
	}

	return helper.JsonOK(c, "Berhasil mengambil semua tag", dto.ToMasjidTagResponseList(tags))
}

// ✅ Hapus tag berdasarkan ID
func (ctrl *MasjidTagController) DeleteTag(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ID wajib dikirim")
	}

	res := ctrl.DB.WithContext(c.Context()).
		Where("masjid_tag_id = ?", id).
		Delete(&model.MasjidTagModel{})

	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus tag")
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Tag tidak ditemukan")
	}

	return helper.JsonDeleted(c, "Tag berhasil dihapus", fiber.Map{
		"masjid_tag_id": id,
	})
}

// ✅ Ambil beberapa tag berdasarkan daftar ID (POST JSON body)
func (ctrl *MasjidTagController) GetTagsByIDs(c *fiber.Ctx) error {
	var payload struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Body JSON tidak valid")
	}
	if len(payload.IDs) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Daftar ID wajib dikirim")
	}

	var tags []model.MasjidTagModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("masjid_tag_id IN ?", payload.IDs).
		Find(&tags).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data tag")
	}

	return helper.JsonOK(c, "Berhasil mengambil tag", dto.ToMasjidTagResponseList(tags))
}
