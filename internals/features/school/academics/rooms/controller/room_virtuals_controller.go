// file: internals/features/school/academics/rooms/controller/class_room_virtual_link_controller.go
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

/* ===================== helpers ===================== */

// Ambil {masjid_id, id} dari path params dan fetch data.
// withDeleted=true -> pakai Unscoped (ikut data soft-deleted).
func (h *ClassRoomVirtualLinkController) findByIDFromParams(c *fiber.Ctx, withDeleted bool) (*clsModel.ClassRoomVirtualLinkModel, error) {
	masjidIDStr := strings.TrimSpace(c.Params("masjid_id"))
	idStr := strings.TrimSpace(c.Params("id"))

	masjidID, err := uuid.Parse(masjidIDStr)
	if err != nil {
		return nil, helper.JsonError(c, fiber.StatusBadRequest, "masjid_id tidak valid")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	db := h.DB
	if withDeleted {
		db = db.Unscoped()
	}

	var m clsModel.ClassRoomVirtualLinkModel
	if err := db.Where(
		"class_room_virtual_link_id = ? AND class_room_virtual_link_masjid_id = ?",
		id, masjidID,
	).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return nil, helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return &m, nil
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
	if strings.TrimSpace(req.ClassRoomVirtualLinkPlatform) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Platform wajib diisi")
	}

	m := clsDTO.ToModelClassRoomVirtualLink(&req, nil)

	// Tulis
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
	if _, err := helperAuth.EnsureMasjidAccessDKM(c, mc); err != nil {
		return err
	}

	m, err := h.findByIDFromParams(c, false)
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
	if _, err := helperAuth.EnsureMasjidAccessDKM(c, mc); err != nil {
		return err
	}

	hard := strings.EqualFold(c.Query("hard"), "true")

	// Jika hard=true → ambil termasuk yang sudah terhapus (biar bisa di-hard-delete ulang)
	m, err := h.findByIDFromParams(c, hard)
	if err != nil {
		return err
	}

	if hard {
		// Hard delete (pakai Unscoped)
		if err := h.DB.Unscoped().
			Where("class_room_virtual_link_id = ? AND class_room_virtual_link_masjid_id = ?",
				m.ClassRoomVirtualLinkID, m.ClassRoomVirtualLinkMasjidID).
			Delete(&clsModel.ClassRoomVirtualLinkModel{}).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus permanen")
		}
		return helper.JsonDeleted(c, "Virtual link dihapus permanen", fiber.Map{"id": m.ClassRoomVirtualLinkID})
	}

	// Soft delete:
	// - Jika model menggunakan gorm.DeletedAt → baris di bawah akan set kolom deleted_at otomatis.
	// - Jika model masih *time.Time → ini akan hard delete; bila ingin soft delete manual,
	//   ganti ke Update kolom deleted_at = now() sesuai preferensi kamu.
	if err := h.DB.
		Where("class_room_virtual_link_id = ? AND class_room_virtual_link_masjid_id = ?",
			m.ClassRoomVirtualLinkID, m.ClassRoomVirtualLinkMasjidID).
		Delete(&clsModel.ClassRoomVirtualLinkModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus virtual link")
	}
	return helper.JsonDeleted(c, "Virtual link dihapus", fiber.Map{"id": m.ClassRoomVirtualLinkID})
}
