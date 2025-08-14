// internals/features/lembaga/classes/user_class_sections/main/controller/user_class_section_controller.go
package controller

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"

	ucsDTO "masjidku_backend/internals/features/lembaga/class_sections/main/dto"

	// untuk validasi parent (cek tenant & eksistensi)
	secModel "masjidku_backend/internals/features/lembaga/class_sections/main/model"
	ucModel "masjidku_backend/internals/features/lembaga/classes/main/model"
)

type UserClassSectionController struct {
	DB *gorm.DB
}

func NewUserClassSectionController(db *gorm.DB) *UserClassSectionController {
	return &UserClassSectionController{DB: db}
}

var validateUCS = validator.New()

/* =============== Helpers =============== */

// Pastikan user_class (enrolment) & class_section berada pada masjid yg sama (masjidID token)
func (h *UserClassSectionController) ensureParentsBelongToMasjid(userClassID, sectionID, masjidID uuid.UUID) error {
	// cek user_classes
	{
		var uc ucModel.UserClassesModel
		if err := h.DB.
			Select("user_classes_id, user_classes_masjid_id").
			First(&uc, "user_classes_id = ?", userClassID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fiber.NewError(fiber.StatusBadRequest, "Enrolment (user_class) tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi enrolment")
		}
		if uc.UserClassesMasjidID == nil || *uc.UserClassesMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Enrolment bukan milik masjid Anda")
		}
	}
	// cek class_sections (tidak boleh deleted & harus tenant sama)
	{
		var sec secModel.ClassSectionModel
		if err := h.DB.
			Select("class_sections_id, class_sections_masjid_id, class_sections_deleted_at").
			First(&sec, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi section")
		}
		if sec.MasjidID == nil || *sec.MasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
		}
	}
	return nil
}

// Ambil row user_class_sections + pastikan tenant sama
func (h *UserClassSectionController) findUCSWithTenantGuard(ucsID, masjidID uuid.UUID) (*secModel.UserClassSectionsModel, error) {
	var m secModel.UserClassSectionsModel
	if err := h.DB.
		First(&m, "user_class_sections_id = ? AND user_class_sections_masjid_id = ?", ucsID, masjidID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "Penempatan section tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	return &m, nil
}

// Validasi: hanya boleh ada 1 penempatan aktif per enrolment (user_class_id)
func (h *UserClassSectionController) ensureSingleActivePerUserClass(userClassID, excludeID uuid.UUID) error {
	var cnt int64
	tx := h.DB.Model(&secModel.UserClassSectionsModel{}).
		Where("user_class_sections_user_class_id = ? AND user_class_sections_status = 'active' AND user_class_sections_unassigned_at IS NULL",
			userClassID)
	if excludeID != uuid.Nil {
		tx = tx.Where("user_class_sections_id <> ?", excludeID)
	}
	if err := tx.Count(&cnt).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi penempatan aktif")
	}
	if cnt > 0 {
		return fiber.NewError(fiber.StatusConflict, "Enrolment ini sudah memiliki penempatan aktif")
	}
	return nil
}

/* =============== Handlers (ADMIN) =============== */

// POST /admin/user-class-sections
func (h *UserClassSectionController) CreateUserClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req ucsDTO.CreateUserClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// force tenant dari token
	req.UserClassSectionsMasjidID = &masjidID

	if err := validateUCS.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Guard parent tenant
	if err := h.ensureParentsBelongToMasjid(req.UserClassSectionsUserClassID, req.UserClassSectionsSectionID, masjidID); err != nil {
		return err
	}

	// Jika status aktif (default aktif) & belum di-unassign, pastikan tidak ada yg aktif lain
	targetStatus := secModel.UserClassSectionStatusActive
	if req.UserClassSectionsStatus != nil && strings.TrimSpace(*req.UserClassSectionsStatus) != "" {
		targetStatus = strings.TrimSpace(*req.UserClassSectionsStatus)
	}
	if strings.EqualFold(targetStatus, secModel.UserClassSectionStatusActive) &&
		(req.UserClassSectionsUnassignedAt == nil) {
		if err := h.ensureSingleActivePerUserClass(req.UserClassSectionsUserClassID, uuid.Nil); err != nil {
			return err
		}
	}

	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat penempatan section")
	}

	return helper.JsonCreated(c, "Penempatan section berhasil dibuat", ucsDTO.NewUserClassSectionResponse(m))
}

// PUT /admin/user-class-sections/:id
func (h *UserClassSectionController) UpdateUserClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	ucsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	existing, err := h.findUCSWithTenantGuard(ucsID, masjidID)
	if err != nil {
		return err
	}

	var req ucsDTO.UpdateUserClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// cegah pindah tenant
	req.UserClassSectionsMasjidID = &masjidID

	if err := validateUCS.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Jika mengganti parent, guard tenantnya
	targetUserClassID := existing.UserClassSectionsUserClassID
	if req.UserClassSectionsUserClassID != nil {
		targetUserClassID = *req.UserClassSectionsUserClassID
	}
	targetSectionID := existing.UserClassSectionsSectionID
	if req.UserClassSectionsSectionID != nil {
		targetSectionID = *req.UserClassSectionsSectionID
	}
	if targetUserClassID != existing.UserClassSectionsUserClassID || targetSectionID != existing.UserClassSectionsSectionID {
		if err := h.ensureParentsBelongToMasjid(targetUserClassID, targetSectionID, masjidID); err != nil {
			return err
		}
	}

	// Hitung status/unassigned target untuk cek unik aktif
	targetStatus := existing.UserClassSectionsStatus
	if req.UserClassSectionsStatus != nil {
		targetStatus = *req.UserClassSectionsStatus
	}
	targetUnassigned := existing.UserClassSectionsUnassignedAt
	if req.UserClassSectionsUnassignedAt != nil {
		targetUnassigned = req.UserClassSectionsUnassignedAt
	}

	// Jika akan menjadi aktif & belum di-unassign, pastikan tidak duplikat aktif
	if strings.EqualFold(targetStatus, secModel.UserClassSectionStatusActive) && targetUnassigned == nil {
		if err := h.ensureSingleActivePerUserClass(targetUserClassID, existing.UserClassSectionsID); err != nil {
			return err
		}
	}

	// apply & save
	req.ApplyToModel(existing)
	if err := h.DB.Model(&secModel.UserClassSectionsModel{}).
		Where("user_class_sections_id = ?", existing.UserClassSectionsID).
		Updates(existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui penempatan section")
	}

	return helper.JsonUpdated(c, "Penempatan section berhasil diperbarui", ucsDTO.NewUserClassSectionResponse(existing))
}

// GET /admin/user-class-sections/:id
func (h *UserClassSectionController) GetUserClassSectionByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	ucsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	m, err := h.findUCSWithTenantGuard(ucsID, masjidID)
	if err != nil {
		return err
	}
	return helper.JsonOK(c, "OK", ucsDTO.NewUserClassSectionResponse(m))
}

// GET /admin/user-class-sections
func (h *UserClassSectionController) ListUserClassSections(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q ucsDTO.ListUserClassSectionQuery
	q.Limit = 20
	q.Offset = 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := h.DB.Model(&secModel.UserClassSectionsModel{}).
		Where("user_class_sections_masjid_id = ?", masjidID)

	if q.UserClassID != nil {
		tx = tx.Where("user_class_sections_user_class_id = ?", *q.UserClassID)
	}
	if q.SectionID != nil {
		tx = tx.Where("user_class_sections_section_id = ?", *q.SectionID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		tx = tx.Where("user_class_sections_status = ?", strings.TrimSpace(*q.Status))
	}
	if q.ActiveOnly != nil && *q.ActiveOnly {
		tx = tx.Where("user_class_sections_status = 'active' AND user_class_sections_unassigned_at IS NULL")
	}

	sort := "assigned_at_desc"
	if q.Sort != nil {
		sort = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sort {
	case "assigned_at_asc":
		tx = tx.Order("user_class_sections_assigned_at ASC")
	case "created_at_asc":
		tx = tx.Order("user_class_sections_created_at ASC")
	case "created_at_desc":
		tx = tx.Order("user_class_sections_created_at DESC")
	default:
		tx = tx.Order("user_class_sections_assigned_at DESC")
	}

	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	var rows []secModel.UserClassSectionsModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]*ucsDTO.UserClassSectionResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, ucsDTO.NewUserClassSectionResponse(&rows[i]))
	}
	return helper.JsonOK(c, "OK", resp)
}

// POST /admin/user-class-sections/:id/end  -> unassign/akhiri penempatan
func (h *UserClassSectionController) EndUserClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	ucsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findUCSWithTenantGuard(ucsID, masjidID)
	if err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]any{
		"user_class_sections_status":       secModel.UserClassSectionStatusEnded,
		"user_class_sections_unassigned_at": now,
		"user_class_sections_updated_at":    now,
	}
	if err := h.DB.Model(&secModel.UserClassSectionsModel{}).
		Where("user_class_sections_id = ?", m.UserClassSectionsID).
		Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengakhiri penempatan")
	}

	return helper.JsonUpdated(c, "Penempatan diakhiri", fiber.Map{
		"user_class_sections_id":            m.UserClassSectionsID,
		"user_class_sections_status":        secModel.UserClassSectionStatusEnded,
		"user_class_sections_unassigned_at": now,
	})
}

// DELETE /admin/user-class-sections/:id  (hard delete dgn guard aman)
func (h *UserClassSectionController) DeleteUserClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	ucsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	m, err := h.findUCSWithTenantGuard(ucsID, masjidID)
	if err != nil {
		return err
	}

	// Larang hapus jika masih aktif & belum di-unassign
	if strings.EqualFold(m.UserClassSectionsStatus, secModel.UserClassSectionStatusActive) &&
		m.UserClassSectionsUnassignedAt == nil {
		return fiber.NewError(fiber.StatusConflict, "Penempatan masih aktif, akhiri terlebih dahulu")
	}

	if err := h.DB.Delete(&secModel.UserClassSectionsModel{}, "user_class_sections_id = ?", m.UserClassSectionsID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus penempatan")
	}

	return helper.JsonDeleted(c, "Penempatan dihapus", fiber.Map{
		"user_class_sections_id": m.UserClassSectionsID,
	})
}
