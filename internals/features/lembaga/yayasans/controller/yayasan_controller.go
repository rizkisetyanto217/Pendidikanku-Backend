// internals/features/lembaga/yayasans/controller/yayasan_controller.go
package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	yDTO "masjidku_backend/internals/features/lembaga/yayasans/dto"
	yModel "masjidku_backend/internals/features/lembaga/yayasans/model"
	helper "masjidku_backend/internals/helpers"
)

type YayasanController struct {
	DB *gorm.DB
}

func NewYayasanController(db *gorm.DB) *YayasanController {
	return &YayasanController{DB: db}
}

/* ===================== HANDLERS ===================== */

// POST /admin/yayasans
func (h *YayasanController) Create(c *fiber.Ctx) error {
	var req yDTO.CreateYayasanRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// validasi ringan → bisa pakai validator global kalau perlu
	if strings.TrimSpace(req.YayasanName) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Nama yayasan wajib diisi")
	}

	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat yayasan")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Yayasan berhasil dibuat",
		"data":    yDTO.NewYayasanResponse(m),
	})
}

// PATCH /admin/yayasans/:id
func (h *YayasanController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req yDTO.UpdateYayasanRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	m, err := h.findByID(id, false)
	if err != nil {
		return err
	}
	req.ApplyToModel(m)

	if err := h.DB.Save(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui yayasan")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Yayasan diperbarui",
		"data":    yDTO.NewYayasanResponse(m),
	})
}

// DELETE /admin/yayasans/:id (soft delete default, hard=?hard=true)
func (h *YayasanController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	hard := strings.EqualFold(c.Query("hard"), "true")

	m, err := h.findByID(id, hard)
	if err != nil {
		return err
	}

	if hard {
		if err := h.DB.Unscoped().Delete(&yModel.YayasanModel{}, "yayasan_id = ?", m.YayasanID).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus permanen")
		}
		return c.JSON(fiber.Map{"message": "Yayasan dihapus permanen", "id": m.YayasanID})
	}

	if err := h.DB.Delete(&yModel.YayasanModel{}, "yayasan_id = ?", m.YayasanID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus yayasan")
	}
	return c.JSON(fiber.Map{"message": "Yayasan dihapus", "id": m.YayasanID})
}

// POST /admin/yayasans/:id/restore
func (h *YayasanController) Restore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findByID(id, true)
	if err != nil {
		return err
	}
	if !m.YayasanDeletedAt.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "Yayasan tidak dalam status terhapus")
	}

	if err := h.DB.Unscoped().
		Model(&yModel.YayasanModel{}).
		Where("yayasan_id = ?", id).
		Update("yayasan_deleted_at", nil).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memulihkan yayasan")
	}

	return c.JSON(fiber.Map{"message": "Yayasan dipulihkan", "data": yDTO.NewYayasanResponse(m)})
}

// GET /admin/yayasans/:id
func (h *YayasanController) Detail(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	m, err := h.findByID(id, false)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": yDTO.NewYayasanResponse(m)})
}



// GET /admin/yayasans dan /public/yayasans
func (h *YayasanController) List(c *fiber.Ctx) error {
    // 1) Ambil params paginasi
    req, _ := http.NewRequest("GET", "http://local"+c.OriginalURL(), nil)
    p := helper.ParseWith(req, "created_at", "desc", helper.AdminOpts)

    // 2) ORDER BY aman (whitelist)
    orderClause, err := p.SafeOrderClause(map[string]string{
        "created_at": "yayasan_created_at",
        "updated_at": "yayasan_updated_at",
        "name":       "lower(yayasan_name)",
        "city":       "lower(yayasan_city)",
        "province":   "lower(yayasan_province)",
    }, "created_at")
    if err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak dikenal")
    }

    // ⬇️ convert "ORDER BY xxx" -> "xxx"
    orderExpr := strings.TrimSpace(orderClause)
    up := strings.ToUpper(orderExpr)
    if strings.HasPrefix(up, "ORDER BY ") {
        orderExpr = strings.TrimSpace(orderExpr[len("ORDER BY "):])
    }

    dbq := h.DB.Model(&yModel.YayasanModel{})

    if v := strings.TrimSpace(c.Query("city")); v != "" {
        dbq = dbq.Where("yayasan_city ILIKE ?", "%"+v+"%")
    }
    if v := strings.TrimSpace(c.Query("province")); v != "" {
        dbq = dbq.Where("yayasan_province ILIKE ?", "%"+v+"%")
    }
    if v := c.Query("active"); v != "" {
        if b, err := strconv.ParseBool(v); err == nil {
            dbq = dbq.Where("yayasan_is_active = ?", b)
        }
    }
    if v := c.Query("verified"); v != "" {
        if b, err := strconv.ParseBool(v); err == nil {
            dbq = dbq.Where("yayasan_is_verified = ?", b)
        }
    }

    // 4) total
    var total int64
    if err := dbq.Count(&total).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung data")
    }

    // 5) data
    var rows []yModel.YayasanModel
    if err := dbq.
        Order(orderExpr). // atau: Order(clause.Expr{SQL: orderExpr})
        Limit(p.Limit()).
        Offset(p.Offset()).
        Find(&rows).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
    }

    items := make([]*yDTO.YayasanResponse, 0, len(rows))
    for i := range rows {
        items = append(items, yDTO.NewYayasanResponse(&rows[i]))
    }

    meta := helper.BuildMeta(total, p)
    return c.JSON(fiber.Map{
        "data":       items,
        "pagination": meta,
    })
}

/* ===================== HELPERS ===================== */

func (h *YayasanController) findByID(id uuid.UUID, includeDeleted bool) (*yModel.YayasanModel, error) {
	var m yModel.YayasanModel
	q := h.DB.Model(&yModel.YayasanModel{})
	if includeDeleted {
		q = q.Unscoped()
	}
	if err := q.Where("yayasan_id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Yayasan tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return &m, nil
}
