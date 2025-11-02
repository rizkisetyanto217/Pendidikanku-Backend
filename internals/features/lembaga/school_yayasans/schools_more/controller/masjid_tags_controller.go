package controller

import (
	"errors"
	"strings"

	"schoolku_backend/internals/features/lembaga/school_yayasans/schools_more/dto"
	"schoolku_backend/internals/features/lembaga/school_yayasans/schools_more/model"
	helper "schoolku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type SchoolTagController struct {
	DB *gorm.DB
}

func NewSchoolTagController(db *gorm.DB) *SchoolTagController {
	return &SchoolTagController{DB: db}
}

// -------------------------------
// util: pagination lokal
// -------------------------------
// func getPagination(c *fiber.Ctx, defaultPage, defaultLimit int) (int, int) {
// 	page := defaultPage
// 	limit := defaultLimit

// 	if v := strings.TrimSpace(c.Query("page")); v != "" {
// 		if p, err := strconv.Atoi(v); err == nil && p > 0 {
// 			page = p
// 		}
// 	}
// 	if v := strings.TrimSpace(c.Query("limit")); v != "" {
// 		if l, err := strconv.Atoi(v); err == nil && l > 0 {
// 			limit = l
// 		}
// 	}
// 	if limit > 100 {
// 		limit = 100
// 	}
// 	return page, limit
// }

// ✅ Tambah tag school
func (ctrl *SchoolTagController) CreateTag(c *fiber.Ctx) error {
	var body dto.SchoolTagRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Input tidak valid")
	}

	// Trim + validasi sederhana di controller
	body.SchoolTagName = strings.TrimSpace(body.SchoolTagName)
	if body.SchoolTagName == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama tag wajib diisi")
	}
	if len(body.SchoolTagName) > 50 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama tag maksimal 50 karakter")
	}
	// description: biarkan nil jika kosong
	if body.SchoolTagDescription != nil {
		desc := strings.TrimSpace(*body.SchoolTagDescription)
		if desc == "" {
			body.SchoolTagDescription = nil
		} else {
			body.SchoolTagDescription = &desc
		}
	}

	tag := body.ToModel()

	if err := ctrl.DB.WithContext(c.Context()).Create(tag).Error; err != nil {
		// duplikasi: unique lower(name)
		msg := strings.ToLower(err.Error())
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "Tag sudah ada")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan tag")
	}

	return helper.JsonCreated(c, "Tag berhasil ditambahkan", dto.ToSchoolTagResponse(tag))
}

// ✅ Ambil semua tag (support ?q=search & pagination ?page=&limit=)
func (ctrl *SchoolTagController) GetAllTags(c *fiber.Ctx) error {
	page, limit := getPagination(c, 1, 20)
	q := strings.TrimSpace(c.Query("q"))

	tx := ctrl.DB.WithContext(c.Context()).
		Model(&model.SchoolTagModel{})

	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("lower(school_tag_name) LIKE lower(?)", like)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data tag")
	}

	var tags []model.SchoolTagModel
	if err := tx.
		Order("school_tag_created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&tags).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data tag")
	}

	return helper.JsonOK(c, "Berhasil mengambil tag", fiber.Map{
		"page":    page,
		"limit":   limit,
		"total":   total,
		"results": dto.ToSchoolTagResponseList(tags),
	})
}

// ✅ Hapus tag berdasarkan ID
func (ctrl *SchoolTagController) DeleteTag(c *fiber.Ctx) error {
	id := c.Params("id")
	if strings.TrimSpace(id) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ID wajib dikirim")
	}

	res := ctrl.DB.WithContext(c.Context()).
		Where("school_tag_id = ?", id).
		Delete(&model.SchoolTagModel{})

	if res.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus tag")
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Tag tidak ditemukan")
	}

	return helper.JsonDeleted(c, "Tag berhasil dihapus", fiber.Map{
		"school_tag_id": id,
	})
}

// ✅ Ambil beberapa tag berdasarkan daftar ID (POST JSON body)
func (ctrl *SchoolTagController) GetTagsByIDs(c *fiber.Ctx) error {
	var payload struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Body JSON tidak valid")
	}
	if len(payload.IDs) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Daftar ID wajib dikirim")
	}

	var tags []model.SchoolTagModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("school_tag_id IN ?", payload.IDs).
		Find(&tags).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data tag")
	}

	return helper.JsonOK(c, "Berhasil mengambil tag", dto.ToSchoolTagResponseList(tags))
}
