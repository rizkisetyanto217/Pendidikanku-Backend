package controller

import (
	"errors"
	"strings"

	"schoolku_backend/internals/features/lembaga/school_yayasans/schools_more/dto"
	"schoolku_backend/internals/features/lembaga/school_yayasans/schools_more/model"
	helper "schoolku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchoolTagRelationController struct {
	DB *gorm.DB
}

func NewSchoolTagRelationController(db *gorm.DB) *SchoolTagRelationController {
	return &SchoolTagRelationController{DB: db}
}

// ✅ Tambah relasi tag ke school
func (ctrl *SchoolTagRelationController) CreateRelation(c *fiber.Ctx) error {
	var body dto.SchoolTagRelationRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Input tidak valid")
	}

	// ✅ Validasi UUID: gunakan uuid.Nil, bukan TrimSpace
	if body.SchoolTagRelationSchoolID == uuid.Nil || body.SchoolTagRelationTagID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id dan tag_id wajib diisi")
	}

	rel := body.ToModel() // *model.SchoolTagRelationModel
	if err := ctrl.DB.WithContext(c.Context()).Create(rel).Error; err != nil {
		// Tangani duplikat (unique (school_id, tag_id))
		lower := strings.ToLower(err.Error())
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(lower, "duplicate key") {
			return helper.JsonError(c, fiber.StatusConflict, "Relasi tag untuk school ini sudah ada")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan relasi tag")
	}

	return helper.JsonCreated(c, "Relasi tag berhasil ditambahkan", dto.ToSchoolTagRelationResponse(rel))
}

// ✅ Ambil semua tag yang terkait dengan school tertentu
func (ctrl *SchoolTagRelationController) GetTagsBySchool(c *fiber.Ctx) error {
	schoolID := c.Query("school_id")
	if strings.TrimSpace(schoolID) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_id wajib dikirim sebagai query parameter")
	}

	var relations []model.SchoolTagRelationModel
	if err := ctrl.DB.WithContext(c.Context()).
		Preload("SchoolTag"). // pastikan relasi didefinisikan di model
		Where("school_tag_relation_school_id = ?", schoolID).
		Order("school_tag_relation_created_at DESC").
		Find(&relations).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data relasi tag")
	}

	return helper.JsonOK(c, "Berhasil mengambil relasi tag school", dto.ToSchoolTagRelationResponseList(relations))
}
