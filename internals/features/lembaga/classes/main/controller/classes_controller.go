package controller

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/lembaga/classes/main/dto"
	"masjidku_backend/internals/features/lembaga/classes/main/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
)

/* ================= Controller & Constructor ================= */

type ClassController struct {
	DB *gorm.DB
}

func NewClassController(db *gorm.DB) *ClassController {
	return &ClassController{DB: db}
}

// single validator instance for this package (tidak perlu di-inject)
var validate = validator.New()

/* ================= Handlers ================= */

// POST /admin/classes
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// paksa tenant dari token (abaikan masukan klien)
	req.ClassMasjidID = &masjidID
	req.ClassSlug = helper.NormalizeSlug(req.ClassSlug)

	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()
	// jaga2: slug unik per sistem (DB sudah ada unique)
	if err := ctrl.DB.Where("class_slug = ? AND class_deleted_at IS NULL", m.ClassSlug).
		First(&model.ClassModel{}).Error; err == nil {
		return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
	}

	if err := ctrl.DB.Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}

	return c.Status(fiber.StatusCreated).JSON(dto.NewClassResponse(m))
}

// PUT /admin/classes/:id
// PUT /admin/classes/:id
func (ctrl *ClassController) UpdateClass(c *fiber.Ctx) error {
    masjidID, err := helper.GetMasjidIDFromToken(c)
    if err != nil { return err }

    classID, err := uuid.Parse(c.Params("id"))
    if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

    var existing model.ClassModel
    if err := ctrl.DB.First(&existing, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
        }
        return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
    }
    if existing.ClassMasjidID == nil || *existing.ClassMasjidID != masjidID {
        return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah kelas di masjid lain")
    }

    var req dto.UpdateClassRequest
    if err := c.BodyParser(&req); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
    }

    // --- normalize slug/name ---
    if req.ClassSlug != nil {
        s := helper.NormalizeSlug(*req.ClassSlug)
        req.ClassSlug = &s
    } else if req.ClassName != nil { // <-- AUTO REGEN dari name jika slug tidak dikirim
        s := helper.NormalizeSlug(*req.ClassName)
        req.ClassSlug = &s
    }

    // cegah pindah tenant
    req.ClassMasjidID = &masjidID

    if err := validate.Struct(req); err != nil {
        return fiber.NewError(fiber.StatusBadRequest, err.Error())
    }

    // cek unik slug (exclude current id) bila akan mengubah slug
    if req.ClassSlug != nil && *req.ClassSlug != existing.ClassSlug {
        var cnt int64
        if err := ctrl.DB.Model(&model.ClassModel{}).
            Where("class_slug = ? AND class_id <> ? AND class_deleted_at IS NULL", *req.ClassSlug, existing.ClassID).
            Count(&cnt).Error; err == nil && cnt > 0 {
            return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
        }
    }

    // apply & save
    req.ApplyToModel(&existing)
    if err := ctrl.DB.Model(&model.ClassModel{}).
        Where("class_id = ?", existing.ClassID).
        Updates(&existing).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
    }

    return c.JSON(dto.NewClassResponse(&existing))
}

// GET /admin/classes/:id
func (ctrl *ClassController) GetClassByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.ClassModel
	if err := ctrl.DB.First(&m, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	// tenant check
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses kelas di masjid lain")
	}
	return c.JSON(dto.NewClassResponse(&m))
}

// GET /admin/classes/slug/:slug
func (ctrl *ClassController) GetClassBySlug(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	slug := helper.NormalizeSlug(c.Params("slug"))

	var m model.ClassModel
	if err := ctrl.DB.First(&m, "class_slug = ? AND class_deleted_at IS NULL", slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses kelas di masjid lain")
	}
	return c.JSON(dto.NewClassResponse(&m))
}

// GET /admin/classes
func (ctrl *ClassController) ListClasses(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q dto.ListClassQuery
	// default paging
	q.Limit = 20
	q.Offset = 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_deleted_at IS NULL").
		Where("class_masjid_id = ?", masjidID)

	if q.ActiveOnly != nil {
		tx = tx.Where("class_is_active = ?", *q.ActiveOnly)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
		tx = tx.Where("(LOWER(class_name) LIKE ? OR LOWER(class_level) LIKE ?)", s, s)
	}
	sortVal := ""
	if q.Sort != nil {
		sortVal = strings.ToLower(strings.TrimSpace(*q.Sort))
	}

	switch sortVal {
	case "name_asc":
		tx = tx.Order("class_name ASC")
	case "name_desc":
		tx = tx.Order("class_name DESC")
	case "created_at_asc":
		tx = tx.Order("class_created_at ASC")
	default:
		tx = tx.Order("class_created_at DESC")
	}

	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	var rows []model.ClassModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]*dto.ClassResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, dto.NewClassResponse(&rows[i]))
	}
	return c.JSON(resp)
}

// DELETE /admin/classes/:id  (soft delete)
func (ctrl *ClassController) SoftDeleteClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.ClassModel
	if err := ctrl.DB.First(&m, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus kelas di masjid lain")
	}

	now := time.Now()
	updates := map[string]any{
		"class_deleted_at": now,
		"class_is_active":  false,
	}
	if err := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_id = ?", m.ClassID).
		Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return c.SendStatus(fiber.StatusNoContent)
}
