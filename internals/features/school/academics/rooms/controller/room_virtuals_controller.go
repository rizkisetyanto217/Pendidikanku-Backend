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

	clsDTO "masjidku_backend/internals/features/school/academics/rooms/dto"
	clsModel "masjidku_backend/internals/features/school/academics/rooms/model"
	helper "masjidku_backend/internals/helpers"
)

/* =========================================================
   Controller
========================================================= */

type ClassRoomVirtualLinkController struct {
	DB *gorm.DB
}

func NewClassRoomVirtualLinkController(db *gorm.DB) *ClassRoomVirtualLinkController {
	return &ClassRoomVirtualLinkController{DB: db}
}

/* ===================== HANDLERS ===================== */

// POST /admin/class-room-virtual-links
func (h *ClassRoomVirtualLinkController) Create(c *fiber.Ctx) error {
	var req clsDTO.ClassRoomVirtualLinkCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Validasi minimal
	if req.ClassRoomVirtualLinkMasjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Masjid ID wajib diisi")
	}
	if req.ClassRoomVirtualLinkRoomID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Room ID wajib diisi")
	}
	if strings.TrimSpace(req.ClassRoomVirtualLinkLabel) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Label wajib diisi")
	}
	if strings.TrimSpace(req.ClassRoomVirtualLinkJoinURL) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Join URL wajib diisi")
	}

	m := clsDTO.ToModelClassRoomVirtualLink(&req, nil)
	if err := h.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat virtual link")
	}

	return helper.JsonCreated(c, "Virtual link dibuat", clsDTO.FromModelClassRoomVirtualLink(m))
}

// PATCH /admin/class-room-virtual-links/:id
func (h *ClassRoomVirtualLinkController) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findByID(id, false)
	if err != nil {
		return err
	}

	var req clsDTO.ClassRoomVirtualLinkUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	clsDTO.ApplyUpdateClassRoomVirtualLink(m, &req)
	m.ClassRoomVirtualLinkUpdatedAt = time.Now()

	if err := h.DB.Save(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui virtual link")
	}
	return helper.JsonUpdated(c, "Virtual link diperbarui", clsDTO.FromModelClassRoomVirtualLink(m))
}

// DELETE /admin/class-room-virtual-links/:id?hard=true
func (h *ClassRoomVirtualLinkController) Delete(c *fiber.Ctx) error {
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
		if err := h.DB.Unscoped().Delete(&clsModel.ClassRoomVirtualLinkModel{}, "class_room_virtual_link_id = ?", m.ClassRoomVirtualLinkID).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus permanen")
		}
		return helper.JsonDeleted(c, "Virtual link dihapus permanen", fiber.Map{"id": m.ClassRoomVirtualLinkID})
	}

	if err := h.DB.Delete(&clsModel.ClassRoomVirtualLinkModel{}, "class_room_virtual_link_id = ?", m.ClassRoomVirtualLinkID).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus virtual link")
	}
	return helper.JsonDeleted(c, "Virtual link dihapus", fiber.Map{"id": m.ClassRoomVirtualLinkID})
}

// POST /admin/class-room-virtual-links/:id/restore
func (h *ClassRoomVirtualLinkController) Restore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findByID(id, true)
	if err != nil {
		return err
	}
	if !m.ClassRoomVirtualLinkDeletedAt.Valid {
		return helper.JsonError(c, fiber.StatusBadRequest, "Data tidak dalam status terhapus")
	}

	if err := h.DB.Unscoped().
		Model(&clsModel.ClassRoomVirtualLinkModel{}).
		Where("class_room_virtual_link_id = ?", id).
		Update("class_room_virtual_link_deleted_at", nil).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulihkan virtual link")
	}

	m, _ = h.findByID(id, false)
	return helper.JsonOK(c, "Virtual link dipulihkan", clsDTO.FromModelClassRoomVirtualLink(m))
}

// GET /admin/class-room-virtual-links/:id
func (h *ClassRoomVirtualLinkController) Detail(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	m, err := h.findByID(id, false)
	if err != nil {
		return err
	}
	return helper.JsonOK(c, "Detail virtual link", clsDTO.FromModelClassRoomVirtualLink(m))
}

// GET /admin/class-room-virtual-links
// Query:
//   masjid_id, room_id, active, q
//   sort_by: created_at|updated_at|label (default: created_at)
//   order: asc|desc
//   limit, page
func (h *ClassRoomVirtualLinkController) List(c *fiber.Ctx) error {
	req, _ := http.NewRequest("GET", "http://local"+c.OriginalURL(), nil)
	p := helper.ParseWith(req, "created_at", "desc", helper.AdminOpts)

	orderClause, err := p.SafeOrderClause(map[string]string{
		"created_at": "class_room_virtual_link_created_at",
		"updated_at": "class_room_virtual_link_updated_at",
		"label":      "LOWER(class_room_virtual_link_label)",
	}, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak dikenal")
	}
	orderExpr := strings.TrimPrefix(strings.TrimSpace(orderClause), "ORDER BY ")

	q := h.DB.Model(&clsModel.ClassRoomVirtualLinkModel{})

	if v := strings.TrimSpace(c.Query("masjid_id")); v != "" {
		if uid, e := uuid.Parse(v); e == nil {
			q = q.Where("class_room_virtual_link_masjid_id = ?", uid)
		}
	}
	if v := strings.TrimSpace(c.Query("room_id")); v != "" {
		if uid, e := uuid.Parse(v); e == nil {
			q = q.Where("class_room_virtual_link_room_id = ?", uid)
		}
	}
	if v := c.Query("active"); v != "" {
		if b, e := strconv.ParseBool(v); e == nil {
			q = q.Where("class_room_virtual_link_is_active = ?", b)
		}
	}
	if kw := strings.TrimSpace(c.Query("q")); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("class_room_virtual_link_label ILIKE ? OR class_room_virtual_link_join_url ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var rows []clsModel.ClassRoomVirtualLinkModel
	if err := q.Order(orderExpr).Limit(p.Limit()).Offset(p.Offset()).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]clsDTO.ClassRoomVirtualLinkResponse, 0, len(rows))
	for i := range rows {
		items = append(items, clsDTO.FromModelClassRoomVirtualLink(&rows[i]))
	}
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}

/* ===================== HELPERS ===================== */

func (h *ClassRoomVirtualLinkController) findByID(id uuid.UUID, includeDeleted bool) (*clsModel.ClassRoomVirtualLinkModel, error) {
	var m clsModel.ClassRoomVirtualLinkModel
	dbq := h.DB.Model(&clsModel.ClassRoomVirtualLinkModel{})
	if includeDeleted {
		dbq = dbq.Unscoped()
	}
	if err := dbq.Where("class_room_virtual_link_id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Virtual link tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return &m, nil
}
