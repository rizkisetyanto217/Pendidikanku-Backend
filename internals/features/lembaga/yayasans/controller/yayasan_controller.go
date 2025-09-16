// internals/features/lembaga/yayasans/controller/yayasan_controller.go
package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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

func isUniqueViolation(err error) bool {
	// Postgres unique_violation code = 23505
	return err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}


/* ===================== HANDLERS ===================== */

// POST /admin/yayasans
func (h *YayasanController) Create(c *fiber.Ctx) error {
	var req yDTO.CreateYayasanRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// validasi minimal
	if strings.TrimSpace(req.YayasanName) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama yayasan wajib diisi")
	}
	if strings.TrimSpace(req.YayasanSlug) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug wajib diisi")
	}

	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		// opsional: deteksi unique violation slug/domain
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Slug/domain sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat yayasan")
	}

	return helper.JsonCreated(c, "Yayasan berhasil dibuat", yDTO.NewYayasanResponse(m))
}

// PATCH /admin/yayasans/:id
// Mendukung tri-state PATCH via dto.UpdateYayasanRequest + ApplyToModel
func (h *YayasanController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req yDTO.UpdateYayasanRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	m, err := h.findByID(id, false)
	if err != nil {
		return err
	}

	// apply tri-state patch ke model
	req.ApplyToModel(m)

	// set updated_at (jaga-jaga; GORM autoUpdateTime juga akan set)
	now := time.Now()
	m.YayasanUpdatedAt = now

	if err := h.DB.Save(m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Slug/domain sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui yayasan")
	}

	return helper.JsonUpdated(c, "Yayasan diperbarui", yDTO.NewYayasanResponse(m))
}

// DELETE /admin/yayasans/:id (soft default, ?hard=true untuk hard delete)
func (h *YayasanController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	hard := strings.EqualFold(c.Query("hard"), "true")

	m, err := h.findByID(id, hard)
	if err != nil {
		return err
	}

	if hard {
		if err := h.DB.Unscoped().Delete(&yModel.YayasanModel{}, "yayasan_id = ?", m.YayasanID).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus permanen")
		}
		return helper.JsonDeleted(c, "Yayasan dihapus permanen", fiber.Map{"id": m.YayasanID})
	}

	if err := h.DB.Delete(&yModel.YayasanModel{}, "yayasan_id = ?", m.YayasanID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus yayasan")
	}
	return helper.JsonDeleted(c, "Yayasan dihapus", fiber.Map{"id": m.YayasanID})
}

// POST /admin/yayasans/:id/restore
func (h *YayasanController) Restore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findByID(id, true)
	if err != nil {
		return err
	}
	if !m.YayasanDeletedAt.Valid {
		return helper.JsonError(c, fiber.StatusBadRequest, "Yayasan tidak dalam status terhapus")
	}

	if err := h.DB.Unscoped().
		Model(&yModel.YayasanModel{}).
		Where("yayasan_id = ?", id).
		Update("yayasan_deleted_at", nil).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulihkan yayasan")
	}

	// refresh
	m, _ = h.findByID(id, false)
	return helper.JsonOK(c, "Yayasan dipulihkan", yDTO.NewYayasanResponse(m))
}

// GET /admin/yayasans/:id
func (h *YayasanController) Detail(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	m, err := h.findByID(id, false)
	if err != nil {
		return err
	}
	return helper.JsonOK(c, "Detail yayasan", yDTO.NewYayasanResponse(m))
}

// GET /admin/yayasans dan /public/yayasans
// Filter: slug, domain, city, province, active, verified, verification_status, q (FTS/ILIKE)
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
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak dikenal")
	}

	orderExpr := strings.TrimSpace(orderClause)
	if strings.HasPrefix(strings.ToUpper(orderExpr), "ORDER BY ") {
		orderExpr = strings.TrimSpace(orderExpr[len("ORDER BY "):])
	}

	dbq := h.DB.Model(&yModel.YayasanModel{})

	// Filters
	if v := strings.TrimSpace(c.Query("slug")); v != "" {
		dbq = dbq.Where("yayasan_slug = ?", v)
	}
	if v := strings.TrimSpace(c.Query("domain")); v != "" {
		dbq = dbq.Where("LOWER(yayasan_domain) = LOWER(?)", v)
	}
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
	if v := strings.TrimSpace(c.Query("verification_status")); v != "" {
		// normalisasi input
		v = strings.ToLower(v)
		if v == "pending" || v == "approved" || v == "rejected" {
			dbq = dbq.Where("yayasan_verification_status = ?", v)
		}
	}
	// q: gunakan FTS pada yayasan_search; fallback ke ILIKE name
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		// gunakan plainto_tsquery(simple)
		dbq = dbq.Where(
			"(yayasan_search @@ plainto_tsquery('simple', ?)) OR (yayasan_name ILIKE ?)",
			q, "%"+q+"%",
		)
	}

	// total
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// data
	var rows []yModel.YayasanModel
	if err := dbq.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]*yDTO.YayasanResponse, 0, len(rows))
	for i := range rows {
		items = append(items, yDTO.NewYayasanResponse(&rows[i]))
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
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
