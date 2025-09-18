package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	clsDTO "masjidku_backend/internals/features/school/academics/rooms/dto"
	clsModel "masjidku_backend/internals/features/school/academics/rooms/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
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

// POST /admin/:masjid_id/class-room-virtual-links
func (h *ClassRoomVirtualLinkController) Create(c *fiber.Ctx) error {
	// === Masjid context (DKM only) ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req clsDTO.ClassRoomVirtualLinkCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force scope dari context, abaikan masjid_id di body
	req.ClassRoomVirtualLinkMasjidID = masjidID

	// Validasi minimal
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

	// Pastikan setiap transaksi menulis masjid_id
	if err := h.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat virtual link")
	}

	return helper.JsonCreated(c, "Virtual link dibuat", clsDTO.FromModelClassRoomVirtualLink(m))
}

// PATCH /admin/:masjid_id/class-room-virtual-links/:id
func (h *ClassRoomVirtualLinkController) Update(c *fiber.Ctx) error {
	// === Masjid context (DKM only) ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findByID(masjidID, id, false)
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

// DELETE /admin/:masjid_id/class-room-virtual-links/:id?hard=true
func (h *ClassRoomVirtualLinkController) Delete(c *fiber.Ctx) error {
	// === Masjid context (DKM only) ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	hard := strings.EqualFold(c.Query("hard"), "true")

	m, err := h.findByID(masjidID, id, hard)
	if err != nil {
		return err
	}

	if hard {
		if err := h.DB.Unscoped().
			Where("class_room_virtual_link_id = ? AND class_room_virtual_link_masjid_id = ?", m.ClassRoomVirtualLinkID, masjidID).
			Delete(&clsModel.ClassRoomVirtualLinkModel{}).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus permanen")
		}
		return helper.JsonDeleted(c, "Virtual link dihapus permanen", fiber.Map{"id": m.ClassRoomVirtualLinkID})
	}

	if err := h.DB.
		Where("class_room_virtual_link_id = ? AND class_room_virtual_link_masjid_id = ?", m.ClassRoomVirtualLinkID, masjidID).
		Delete(&clsModel.ClassRoomVirtualLinkModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus virtual link")
	}
	return helper.JsonDeleted(c, "Virtual link dihapus", fiber.Map{"id": m.ClassRoomVirtualLinkID})
}

// POST /admin/:masjid_id/class-room-virtual-links/:id/restore
func (h *ClassRoomVirtualLinkController) Restore(c *fiber.Ctx) error {
	// === Masjid context (DKM only) ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findByID(masjidID, id, true)
	if err != nil {
		return err
	}
	if !m.ClassRoomVirtualLinkDeletedAt.Valid {
		return helper.JsonError(c, fiber.StatusBadRequest, "Data tidak dalam status terhapus")
	}

	if err := h.DB.Unscoped().
		Model(&clsModel.ClassRoomVirtualLinkModel{}).
		Where("class_room_virtual_link_id = ? AND class_room_virtual_link_masjid_id = ?", id, masjidID).
		Update("class_room_virtual_link_deleted_at", nil).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulihkan virtual link")
	}

	m, _ = h.findByID(masjidID, id, false)
	return helper.JsonOK(c, "Virtual link dipulihkan", clsDTO.FromModelClassRoomVirtualLink(m))
}

// GET /admin/:masjid_id/class-room-virtual-links/:id
func (h *ClassRoomVirtualLinkController) Detail(c *fiber.Ctx) error {
	// === Masjid context (DKM only) ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}
	m, err := h.findByID(masjidID, id, false)
	if err != nil {
		return err
	}
	return helper.JsonOK(c, "Detail virtual link", clsDTO.FromModelClassRoomVirtualLink(m))
}

/* ===================== HELPERS ===================== */

// findByID: selalu filter ke masjid_id agar tenant-safe.
func (h *ClassRoomVirtualLinkController) findByID(masjidID, id uuid.UUID, includeDeleted bool) (*clsModel.ClassRoomVirtualLinkModel, error) {
	var m clsModel.ClassRoomVirtualLinkModel
	dbq := h.DB.Model(&clsModel.ClassRoomVirtualLinkModel{})
	if includeDeleted {
		dbq = dbq.Unscoped()
	}
	if err := dbq.
		Where("class_room_virtual_link_id = ? AND class_room_virtual_link_masjid_id = ?", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Virtual link tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return &m, nil
}
