// file: internals/features/lembaga/classes/user_class_sections/main/controller/user_class_section_controller.go
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
	helperAuth "masjidku_backend/internals/helpers/auth"

	ucsDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"

	// parent validators
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	ucModel "masjidku_backend/internals/features/school/classes/classes/model"
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
func (h *UserClassSectionController) ensureParentsBelongToMasjid(
	userClassID, sectionID, masjidID uuid.UUID,
) error {
	// Cek user_classes (tenant sama & belum terhapus)
	{
		var uc ucModel.UserClassesModel
		if err := h.DB.
			Select("user_classes_id, user_classes_masjid_id, user_classes_deleted_at").
			Where("user_classes_id = ? AND user_classes_deleted_at IS NULL", userClassID).
			First(&uc).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Enrolment (user_class) tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi enrolment")
		}
		if uc.UserClassesMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Enrolment bukan milik masjid Anda")
		}
	}

	// Cek class_sections (tenant sama & belum terhapus)
	{
		type clsSection struct {
			ClassSectionsID       uuid.UUID  `gorm:"column:class_sections_id"`
			ClassSectionsMasjidID uuid.UUID  `gorm:"column:class_sections_masjid_id"`
			DeletedAt             *time.Time `gorm:"column:class_sections_deleted_at"`
		}
		var sec clsSection
		if err := h.DB.
			Table("class_sections").
			Select("class_sections_id, class_sections_masjid_id, class_sections_deleted_at").
			Where("class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).
			First(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi section")
		}
		if sec.ClassSectionsMasjidID != masjidID {
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
		Where("user_class_sections_user_class_id = ? AND user_class_sections_unassigned_at IS NULL", userClassID)
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

/* =============== Handlers =============== */

// POST /admin/user-class-sections
func (h *UserClassSectionController) CreateUserClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req ucsDTO.CreateUserClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.UserClassSectionsMasjidID = &masjidID

	// Validasi DTO
	if err := validateUCS.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Validasi tanggal ringan (DB juga sudah ada CHECK)
	if req.UserClassSectionsAssignedAt != nil && req.UserClassSectionsUnassignedAt != nil {
		if req.UserClassSectionsUnassignedAt.Before(*req.UserClassSectionsAssignedAt) {
			return fiber.NewError(fiber.StatusBadRequest, "unassigned_at tidak boleh sebelum assigned_at")
		}
	}

	// Guard tenant: pastikan parent entities memang milik masjid ini
	if err := h.ensureParentsBelongToMasjid(
		req.UserClassSectionsUserClassID,
		req.UserClassSectionsSectionID,
		masjidID,
	); err != nil {
		return err
	}

	// Satu placement aktif per user_class (aktif = unassigned_at IS NULL & belum soft delete)
	if req.UserClassSectionsUnassignedAt == nil {
		if err := h.ensureSingleActivePerUserClass(req.UserClassSectionsUserClassID, uuid.Nil); err != nil {
			return err
		}
	}

	// Simpan penempatan section (no extra services / stats)
	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat penempatan section")
	}

	return helper.JsonCreated(c, "Penempatan section berhasil dibuat", ucsDTO.NewUserClassSectionResponse(m))
}

// PUT /admin/user-class-sections/:id
// UpdateUserClassSection: tanpa kolom status, aktif = unassigned_at IS NULL
func (h *UserClassSectionController) UpdateUserClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
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

	// --- Guard parent (user_class_id & section_id) tetap dalam tenant yang sama ---
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

	// --- Tentukan target unassigned_at (NULL = aktif) ---
	targetUnassigned := existing.UserClassSectionsUnassignedAt
	if req.UserClassSectionsUnassignedAt != nil {
		targetUnassigned = req.UserClassSectionsUnassignedAt
	}

	// --- Jika akan menjadi AKTIF (unassigned_at == NULL) pastikan tidak ada duplikat aktif ---
	if targetUnassigned == nil {
		if err := h.ensureSingleActivePerUserClass(targetUserClassID, existing.UserClassSectionsID); err != nil {
			return err
		}
	}

	// --- Apply & Save ---
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
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
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
// GET /admin/user-class-sections
func (h *UserClassSectionController) ListUserClassSections(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q ucsDTO.ListUserClassSectionQuery
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 20
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	tx := h.DB.Model(&secModel.UserClassSectionsModel{}).
		Where("user_class_sections_masjid_id = ?", masjidID)

	// Filters
	if q.UserClassID != nil {
		tx = tx.Where("user_class_sections_user_class_id = ?", *q.UserClassID)
	}
	if q.SectionID != nil {
		tx = tx.Where("user_class_sections_section_id = ?", *q.SectionID)
	}
	if q.ActiveOnly != nil && *q.ActiveOnly {
		tx = tx.Where("user_class_sections_unassigned_at IS NULL")
	}

	// Sort
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

	// Paging
	tx = tx.Limit(q.Limit).Offset(q.Offset)

	// Fetch
	var rows []secModel.UserClassSectionsModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if len(rows) == 0 {
		return helper.JsonOK(c, "OK", []*ucsDTO.UserClassSectionResponse{})
	}

	/* ===== Enrichment ===== */

	// 1) Kumpulkan user_class_id unik
	ucSet := make(map[uuid.UUID]struct{}, len(rows))
	userClassIDs := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		id := rows[i].UserClassSectionsUserClassID
		if _, ok := ucSet[id]; !ok {
			ucSet[id] = struct{}{}
			userClassIDs = append(userClassIDs, id)
		}
	}

	// 2) Ambil mapping user_class -> (masjid_student_id, status, joined_at)
	type ucMeta struct {
		UserClassID     uuid.UUID  `gorm:"column:user_classes_id"`
		MasjidStudentID uuid.UUID  `gorm:"column:user_classes_masjid_student_id"`
		Status          string     `gorm:"column:user_classes_status"`
		JoinedAt        *time.Time `gorm:"column:user_classes_joined_at"`
	}
	ucMetaByID := make(map[uuid.UUID]ucMeta, len(userClassIDs))
	studentIDByUC := make(map[uuid.UUID]uuid.UUID, len(userClassIDs))

	{
		var ucRows []ucMeta
		if err := h.DB.
			Table("user_classes").
			Select("user_classes_id, user_classes_masjid_student_id, user_classes_status, user_classes_joined_at").
			Where("user_classes_id IN ?", userClassIDs).
			Find(&ucRows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data user_class")
		}
		for _, r := range ucRows {
			ucMetaByID[r.UserClassID] = r
			studentIDByUC[r.UserClassID] = r.MasjidStudentID
		}
	}

	// 3) Kumpulkan masjid_student_id unik → ambil user_id dari masjid_students
	msSet := make(map[uuid.UUID]struct{}, len(userClassIDs))
	masjidStudentIDs := make([]uuid.UUID, 0, len(userClassIDs))
	for _, sid := range studentIDByUC {
		if _, ok := msSet[sid]; !ok {
			msSet[sid] = struct{}{}
			masjidStudentIDs = append(masjidStudentIDs, sid)
		}
	}

	// Map masjid_student_id → user_id
	userIDByMasjidStudent := make(map[uuid.UUID]uuid.UUID, len(masjidStudentIDs))
	if len(masjidStudentIDs) > 0 {
		var msRows []struct {
			MasjidStudentID uuid.UUID `gorm:"column:masjid_student_id"`
			UserID          uuid.UUID `gorm:"column:masjid_student_user_id"`
		}
		if err := h.DB.
			Table("masjid_students").
			Select("masjid_student_id, masjid_student_user_id").
			Where("masjid_student_id IN ?", masjidStudentIDs).
			Find(&msRows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil mapping masjid_student → user")
		}
		for _, r := range msRows {
			userIDByMasjidStudent[r.MasjidStudentID] = r.UserID
		}
	}

	// 4) Kumpulkan user_id unik
	uSet := make(map[uuid.UUID]struct{}, len(userClassIDs))
	userIDs := make([]uuid.UUID, 0, len(userClassIDs))
	for _, ucID := range userClassIDs {
		if sid, ok := studentIDByUC[ucID]; ok {
			if uid, ok2 := userIDByMasjidStudent[sid]; ok2 {
				if _, seen := uSet[uid]; !seen {
					uSet[uid] = struct{}{}
					userIDs = append(userIDs, uid)
				}
			}
		}
	}

	// 5) Ambil users -> map[user_id]UcsUser (tambahkan full_name)
	userMap := make(map[uuid.UUID]ucsDTO.UcsUser, len(userIDs))
	if len(userIDs) > 0 {
		var urs []struct {
			ID       uuid.UUID  `gorm:"column:id"`
			UserName string     `gorm:"column:user_name"`
			FullName *string    `gorm:"column:full_name"`
			Email    string     `gorm:"column:email"`
			IsActive bool       `gorm:"column:is_active"`
		}
		if err := h.DB.
			Table("users").
			Select("id, user_name, full_name, email, is_active").
			Where("id IN ?", userIDs).
			Find(&urs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data user")
		}
		for _, u := range urs {
			userMap[u.ID] = ucsDTO.UcsUser{
				ID:       u.ID,
				UserName: u.UserName,
				FullName: u.FullName, // boleh nil
				Email:    u.Email,
				IsActive: u.IsActive,
			}
		}
	}

	// 6) Ambil users_profile -> map[user_id]UcsUserProfile (sesuai kolom baru)
	profileMap := make(map[uuid.UUID]ucsDTO.UcsUserProfile, len(userIDs))
	if len(userIDs) > 0 {
		var prs []struct {
			UserID                   uuid.UUID  `gorm:"column:user_id"`
			DonationName             *string    `gorm:"column:donation_name"`
			PhotoURL                 *string    `gorm:"column:photo_url"`
			PhotoTrashURL            *string    `gorm:"column:photo_trash_url"`
			PhotoDeletePendingUntil  *time.Time `gorm:"column:photo_delete_pending_until"`
			DateOfBirth              *time.Time `gorm:"column:date_of_birth"`
			Gender                   *string    `gorm:"column:gender"`
			PhoneNumber              *string    `gorm:"column:phone_number"`
			Bio                      *string    `gorm:"column:bio"`
			Location                 *string    `gorm:"column:location"`
			Occupation               *string    `gorm:"column:occupation"`
		}
		if err := h.DB.
			Table("users_profile").
			Select(`user_id, donation_name, photo_url, photo_trash_url, photo_delete_pending_until,
			        date_of_birth, gender, phone_number, bio, location, occupation`).
			Where("user_id IN ?", userIDs).
			Where("deleted_at IS NULL").
			Find(&prs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data profile")
		}
		for _, p := range prs {
			profileMap[p.UserID] = ucsDTO.UcsUserProfile{
				UserID:                  p.UserID,
				DonationName:            p.DonationName,
				PhotoURL:                p.PhotoURL,
				PhotoTrashURL:           p.PhotoTrashURL,
				PhotoDeletePendingUntil: p.PhotoDeletePendingUntil,
				DateOfBirth:             p.DateOfBirth,
				Gender:                  p.Gender,
				PhoneNumber:             p.PhoneNumber,
				Bio:                     p.Bio,
				Location:                p.Location,
				Occupation:              p.Occupation,
			}
		}
	}

	// 7) Build response + enrichment
	resp := make([]*ucsDTO.UserClassSectionResponse, 0, len(rows))
	for i := range rows {
		r := ucsDTO.NewUserClassSectionResponse(&rows[i])

		ucID := rows[i].UserClassSectionsUserClassID
		if meta, ok := ucMetaByID[ucID]; ok {
			r.UserClassesStatus = meta.Status
		}

		if sid, ok := studentIDByUC[ucID]; ok {
			if uid, ok := userIDByMasjidStudent[sid]; ok {
				if u, ok := userMap[uid]; ok {
					uCopy := u
					r.User = &uCopy
				}
				if p, ok := profileMap[uid]; ok {
					pCopy := p
					r.Profile = &pCopy
				}
			}
		}

		resp = append(resp, r)
	}

	return helper.JsonOK(c, "OK", resp)
}

// POST /admin/user-class-sections/:id/end  -> unassign/akhiri penempatan
func (h *UserClassSectionController) EndUserClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
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

	// Idempotent
	if m.UserClassSectionsUnassignedAt != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Penempatan sudah diakhiri sebelumnya")
	}

	// Kolom DATE di DB: pakai tanggal hari ini
	today := time.Now().Truncate(24 * time.Hour)

	updates := map[string]any{
		"user_class_sections_unassigned_at": &today,
		"user_class_sections_updated_at":    time.Now(),
	}

	if err := h.DB.Model(&secModel.UserClassSectionsModel{}).
		Where("user_class_sections_id = ?", m.UserClassSectionsID).
		Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengakhiri penempatan")
	}

	return helper.JsonUpdated(c, "Penempatan diakhiri", fiber.Map{
		"user_class_sections_id":            m.UserClassSectionsID,
		"user_class_sections_unassigned_at": today,
		"is_active":                         false,
	})
}

// DELETE /admin/user-class-sections/:id
// Soft delete (default). Hard delete bila query ?hard=true.
// Tetap guard: tidak boleh menghapus jika masih aktif (unassigned_at IS NULL).
func (h *UserClassSectionController) DeleteUserClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	ucsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// hard delete?
	hard := strings.EqualFold(c.Query("hard"), "true")

	// Cari record dengan guard tenant; untuk hard, kita cari Unscoped (ikut yang sudah soft-deleted),
	// untuk soft, cukup baris hidup (deleted_at IS NULL).
	includeDeleted := hard
	m, err := h.findUCSWithTenantGuard2(ucsID, masjidID, includeDeleted)
	if err != nil {
		return err
	}
	if m == nil {
	 return fiber.NewError(fiber.StatusNotFound, "Penempatan tidak ditemukan")
	}

	// Larang hapus jika masih aktif (aktif = unassigned_at IS NULL)
	if m.UserClassSectionsUnassignedAt == nil {
		return fiber.NewError(fiber.StatusConflict, "Penempatan masih aktif, akhiri terlebih dahulu")
	}

	// Eksekusi delete
	db := h.DB
	if hard {
		// Hard delete permanen
		if err := db.Unscoped().
			Delete(&secModel.UserClassSectionsModel{}, "user_class_sections_id = ?", m.UserClassSectionsID).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus penempatan (hard)")
		}
		return helper.JsonDeleted(c, "Penempatan dihapus permanen", fiber.Map{
			"user_class_sections_id": m.UserClassSectionsID,
			"hard":                   true,
		})
	}

	// Soft delete (gorm.DeletedAt akan diisi otomatis)
	if err := db.
		Delete(&secModel.UserClassSectionsModel{}, "user_class_sections_id = ?", m.UserClassSectionsID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus penempatan (soft)")
	}

	return helper.JsonDeleted(c, "Penempatan dihapus", fiber.Map{
		"user_class_sections_id": m.UserClassSectionsID,
		"hard":                   false,
	})
}

// includeDeleted=false → hanya baris hidup (deleted_at IS NULL)
// includeDeleted=true  → cari Unscoped (termasuk yang sudah soft-deleted)
// findUCSWithTenantGuard contoh (pastikan ada di file ini)
// helper: cari UCS dengan guard tenant
// includeDeleted = true  -> Unscoped (ikut baris yang sudah soft-deleted)
// includeDeleted = false -> hanya baris hidup (deleted_at IS NULL)
func (h *UserClassSectionController) findUCSWithTenantGuard2(
	ucsID, masjidID uuid.UUID,
	includeDeleted bool,
) (*secModel.UserClassSectionsModel, error) {
	var m secModel.UserClassSectionsModel

	q := h.DB.Model(&secModel.UserClassSectionsModel{})
	if includeDeleted {
		q = q.Unscoped()
	}
	q = q.Where("user_class_sections_id = ? AND user_class_sections_masjid_id = ?", ucsID, masjidID)
	if !includeDeleted {
		q = q.Where("user_class_sections_deleted_at IS NULL")
	}

	if err := q.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Penempatan tidak ditemukan")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil penempatan")
	}
	return &m, nil
}


// POST /admin/user-class-sections/:id/restore
func (h *UserClassSectionController) RestoreUserClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	ucsID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Cari yang sudah soft-deleted
	m, err := h.findUCSWithTenantGuard2(ucsID, masjidID, true)
	if err != nil {
		return err
	}
	if m == nil || !m.UserClassSectionsDeletedAt.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "Penempatan tidak dalam status terhapus")
	}

	// Pastikan tidak melanggar aturan "single active" saat restore (jika dia aktif)
	if m.UserClassSectionsUnassignedAt == nil {
		if err := h.ensureSingleActivePerUserClass(m.UserClassSectionsUserClassID, m.UserClassSectionsID); err != nil {
			return err
		}
	}

	// Null-kan deleted_at (restore)
	if err := h.DB.Unscoped().Model(&secModel.UserClassSectionsModel{}).
		Where("user_class_sections_id = ? AND user_class_sections_masjid_id = ?", m.UserClassSectionsID, masjidID).
		Update("user_class_sections_deleted_at", nil).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memulihkan penempatan")
	}

	return helper.JsonOK(c, "Penempatan dipulihkan", ucsDTO.NewUserClassSectionResponse(m))
}
