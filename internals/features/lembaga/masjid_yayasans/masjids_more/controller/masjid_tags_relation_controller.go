package controller

import (
	"errors"
	"strings"

	"masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids_more/dto"
	"masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids_more/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
		return helper.JsonError(c, fiber.StatusBadRequest, "Input tidak valid")
	}

	// ✅ Validasi UUID: gunakan uuid.Nil, bukan TrimSpace
	if body.MasjidTagRelationMasjidID == uuid.Nil || body.MasjidTagRelationTagID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id dan tag_id wajib diisi")
	}

	rel := body.ToModel() // *model.MasjidTagRelationModel
	if err := ctrl.DB.WithContext(c.Context()).Create(rel).Error; err != nil {
		// Tangani duplikat (unique (masjid_id, tag_id))
		lower := strings.ToLower(err.Error())
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(lower, "duplicate key") {
			return helper.JsonError(c, fiber.StatusConflict, "Relasi tag untuk masjid ini sudah ada")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan relasi tag")
	}

	return helper.JsonCreated(c, "Relasi tag berhasil ditambahkan", dto.ToMasjidTagRelationResponse(rel))
}

// ✅ Ambil semua tag yang terkait dengan masjid tertentu
func (ctrl *MasjidTagRelationController) GetTagsByMasjid(c *fiber.Ctx) error {
	masjidID := c.Query("masjid_id")
	if strings.TrimSpace(masjidID) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id wajib dikirim sebagai query parameter")
	}

	var relations []model.MasjidTagRelationModel
	if err := ctrl.DB.WithContext(c.Context()).
		Preload("MasjidTag"). // pastikan relasi didefinisikan di model
		Where("masjid_tag_relation_masjid_id = ?", masjidID).
		Order("masjid_tag_relation_created_at DESC").
		Find(&relations).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data relasi tag")
	}

	return helper.JsonOK(c, "Berhasil mengambil relasi tag masjid", dto.ToMasjidTagRelationResponseList(relations))
}
