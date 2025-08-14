// internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
package controller

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"

	secDTO "masjidku_backend/internals/features/lembaga/class_sections/main/dto"
	secModel "masjidku_backend/internals/features/lembaga/class_sections/main/model"
	classModel "masjidku_backend/internals/features/lembaga/classes/main/model"
)

type ClassSectionController struct {
	DB *gorm.DB
}

func NewClassSectionController(db *gorm.DB) *ClassSectionController {
	return &ClassSectionController{DB: db}
}

var validate = validator.New()

/* ================= Handlers (ADMIN) ================= */

// POST /admin/class-sections
func (ctrl *ClassSectionController) CreateClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil { return err }

	var req secDTO.CreateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// force tenant
	req.ClassSectionMasjidID = &masjidID

	// === AUTO SLUG ===
	if strings.TrimSpace(req.ClassSectionSlug) == "" {
		// generate dari name jika slug kosong
		req.ClassSectionSlug = helper.NormalizeSlug(req.ClassSectionName)
	} else {
		req.ClassSectionSlug = helper.NormalizeSlug(req.ClassSectionSlug)
	}
	// fallback kalau hasil normalisasi kosong (nama cuma simbol/spasi)
	if req.ClassSectionSlug == "" {
		req.ClassSectionSlug = "section-" + uuid.NewString()[:8]
	}

	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ... (lanjutan validasi class & cek unik seperti punyamu)
	// Cek unik slug
	if err := ctrl.DB.Where("class_sections_slug = ? AND class_sections_deleted_at IS NULL", req.ClassSectionSlug).
		First(&secModel.ClassSectionModel{}).Error; err == nil {
		return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
	}

	m := req.ToModel()
	if err := ctrl.DB.Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat section")
	}
	return helper.JsonCreated(c, "Section berhasil dibuat", secDTO.NewClassSectionResponse(m))
}


// PUT /admin/class-sections/:id
func (ctrl *ClassSectionController) UpdateClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var existing secModel.ClassSectionModel
	if err := ctrl.DB.First(&existing, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	// Guard tenant
	if existing.MasjidID == nil || *existing.MasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah section milik masjid lain")
	}

	var req secDTO.UpdateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Normalize slug jika dikirim; kalau tidak ada slug tapi ada name → auto-generate slug dari name
	if req.ClassSectionSlug != nil {
		s := helper.NormalizeSlug(*req.ClassSectionSlug)
		req.ClassSectionSlug = &s
	} else if req.ClassSectionName != nil {
		s := helper.NormalizeSlug(*req.ClassSectionName)
		req.ClassSectionSlug = &s
	}
	// Cegah ganti tenant
	req.ClassSectionMasjidID = &masjidID

	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Jika class_id diganti, validasi class milik tenant
	if req.ClassSectionClassID != nil {
		var cls classModel.ClassModel
		if err := ctrl.DB.
			Select("class_id, class_masjid_id").
			First(&cls, "class_id = ? AND class_deleted_at IS NULL", *req.ClassSectionClassID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class")
		}
		if cls.ClassMasjidID == nil || *cls.ClassMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh memindahkan section ke class milik masjid lain")
		}
	}

	// Cek unik slug (exclude current)
	if req.ClassSectionSlug != nil {
		var cnt int64
		if err := ctrl.DB.Model(&secModel.ClassSectionModel{}).
			Where("class_sections_slug = ? AND class_sections_id <> ? AND class_sections_deleted_at IS NULL",
				*req.ClassSectionSlug, existing.ClassSectionID).
			Count(&cnt).Error; err == nil && cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
		}
	}

	// Cek unik (class_id, name) exclude current
	targetClassID := existing.ClassID
	if req.ClassSectionClassID != nil {
		targetClassID = *req.ClassSectionClassID
	}
	targetName := existing.Name
	if req.ClassSectionName != nil {
		targetName = *req.ClassSectionName
	}
	{
		var cnt int64
		if err := ctrl.DB.Model(&secModel.ClassSectionModel{}).
			Where("class_sections_class_id = ? AND class_sections_name = ? AND class_sections_id <> ? AND class_sections_deleted_at IS NULL",
				targetClassID, targetName, existing.ClassSectionID).
			Count(&cnt).Error; err == nil && cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Nama section sudah dipakai pada class ini")
		}
	}

	req.ApplyToModel(&existing)

	if err := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", existing.ClassSectionID).
		Updates(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui section")
	}
	return helper.JsonUpdated(c, "Section berhasil diperbarui", secDTO.NewClassSectionResponse(&existing))
}

// GET /admin/class-sections/:id
func (ctrl *ClassSectionController) GetClassSectionByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m secModel.ClassSectionModel
	if err := ctrl.DB.First(&m, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.MasjidID == nil || *m.MasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses section milik masjid lain")
	}
	return helper.JsonOK(c, "OK", secDTO.NewClassSectionResponse(&m))
}


// GET /admin/class-sections
func (ctrl *ClassSectionController) ListClassSections(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q secDTO.ListClassSectionQuery
	q.Limit = 20
	q.Offset = 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_deleted_at IS NULL").
		Where("class_sections_masjid_id = ?", masjidID)

	if q.ActiveOnly != nil {
		tx = tx.Where("class_sections_is_active = ?", *q.ActiveOnly)
	}
	if q.ClassID != nil {
		tx = tx.Where("class_sections_class_id = ?", *q.ClassID)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
		tx = tx.Where("(LOWER(class_sections_name) LIKE ? OR LOWER(class_sections_code) LIKE ?)", s, s)
	}

	sortVal := ""
	if q.Sort != nil {
		sortVal = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sortVal {
	case "name_asc":
		tx = tx.Order("class_sections_name ASC")
	case "name_desc":
		tx = tx.Order("class_sections_name DESC")
	case "created_at_asc":
		tx = tx.Order("class_sections_created_at ASC")
	default:
		tx = tx.Order("class_sections_created_at DESC")
	}

	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]*secDTO.ClassSectionResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, secDTO.NewClassSectionResponse(&rows[i]))
	}
	return helper.JsonOK(c, "OK", resp)
}

// DELETE /admin/class-sections/:id  (soft delete)
func (ctrl *ClassSectionController) SoftDeleteClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m secModel.ClassSectionModel
	if err := ctrl.DB.First(&m, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.MasjidID == nil || *m.MasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus section milik masjid lain")
	}

	now := time.Now()
	updates := map[string]any{
		"class_sections_deleted_at": now,
		"class_sections_is_active":  false,
	}
	if err := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", m.ClassSectionID).
		Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus section")
	}
	return helper.JsonDeleted(c, "Section berhasil dihapus", fiber.Map{"class_sections_id": m.ClassSectionID})
}
