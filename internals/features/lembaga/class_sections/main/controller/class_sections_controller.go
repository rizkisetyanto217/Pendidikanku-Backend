// internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
package controller

import (
	"errors"
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
	if err != nil {
		return err
	}

	var req secDTO.CreateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// force tenant
	req.ClassSectionsMasjidID = &masjidID

	// === AUTO SLUG ===
	if strings.TrimSpace(req.ClassSectionsSlug) == "" {
		// generate dari name jika slug kosong
		req.ClassSectionsSlug = helper.NormalizeSlug(req.ClassSectionsName)
	} else {
		req.ClassSectionsSlug = helper.NormalizeSlug(req.ClassSectionsSlug)
	}
	// fallback kalau hasil normalisasi kosong (nama cuma simbol/spasi)
	if req.ClassSectionsSlug == "" {
		req.ClassSectionsSlug = "section-" + uuid.NewString()[:8]
	}

	// Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Cek unik slug
	var exists secModel.ClassSectionModel
	if err := ctrl.DB.Where("class_sections_slug = ? AND class_sections_deleted_at IS NULL", req.ClassSectionsSlug).
		First(&exists).Error; err == nil {
		return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
	}

	// Mapping ke model
	m := req.ToModel()

	// Simpan ke DB
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

	// Ambil existing (pastikan belum soft-deleted)
	var existing secModel.ClassSectionModel
	if err := ctrl.DB.
		First(&existing, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Guard tenant
	if existing.ClassSectionsMasjidID == nil || *existing.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah section milik masjid lain")
	}

	// Parse & normalisasi payload
	var req secDTO.UpdateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalisasi slug bila dikirim; atau auto dari name bila name dikirim
	if req.ClassSectionsSlug != nil {
		s := helper.NormalizeSlug(*req.ClassSectionsSlug)
		if s == "" { s = "section-" + uuid.NewString()[:8] }
		req.ClassSectionsSlug = &s
	} else if req.ClassSectionsName != nil {
		s := helper.NormalizeSlug(*req.ClassSectionsName)
		if s == "" { s = "section-" + uuid.NewString()[:8] }
		req.ClassSectionsSlug = &s
	}

	// Cegah ganti tenant dari luar
	req.ClassSectionsMasjidID = &masjidID

	// Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Jika class_id diganti, validasi class milik tenant
	if req.ClassSectionsClassID != nil {
		var cls classModel.ClassModel
		if err := ctrl.DB.
			Select("class_id, class_masjid_id").
			First(&cls, "class_id = ? AND class_deleted_at IS NULL", *req.ClassSectionsClassID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class")
		}
		if cls.ClassMasjidID == nil || *cls.ClassMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh memindahkan section ke class milik masjid lain")
		}
	}

	// Cek unik slug (exclude current)
	if req.ClassSectionsSlug != nil {
		var cnt int64
		if err := ctrl.DB.Model(&secModel.ClassSectionModel{}).
			Where("class_sections_slug = ? AND class_sections_id <> ? AND class_sections_deleted_at IS NULL",
				*req.ClassSectionsSlug, existing.ClassSectionsID).
			Count(&cnt).Error; err == nil && cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
		}
	}

	// Cek unik (class_id, name) exclude current
	targetClassID := existing.ClassSectionsClassID
	if req.ClassSectionsClassID != nil {
		targetClassID = *req.ClassSectionsClassID
	}
	targetName := existing.ClassSectionsName
	if req.ClassSectionsName != nil {
		targetName = *req.ClassSectionsName
	}
	{
		var cnt int64
		if err := ctrl.DB.Model(&secModel.ClassSectionModel{}).
			Where(`class_sections_class_id = ? 
			       AND class_sections_name = ? 
			       AND class_sections_id <> ? 
			       AND class_sections_deleted_at IS NULL`,
				targetClassID, targetName, existing.ClassSectionsID).
			Count(&cnt).Error; err == nil && cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Nama section sudah dipakai pada class ini")
		}
	}

	// Apply perubahan ke model
	req.ApplyToModel(&existing)

	// Update by PK
	if err := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", existing.ClassSectionsID).
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
	if m.ClassSectionsMasjidID == nil || *m.ClassSectionsMasjidID != masjidID {
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
	if m.ClassSectionsMasjidID == nil || *m.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus section milik masjid lain")
	}

	now := time.Now()
	updates := map[string]any{
		"class_sections_deleted_at": now,
		"class_sections_is_active":  false,
	}
	if err := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", m.ClassSectionsClassID).
		Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus section")
	}
	return helper.JsonDeleted(c, "Section berhasil dihapus", fiber.Map{"class_sections_id": m.ClassSectionsID})
}
