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

	enrolDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"

	// parents / models
	ucModel "masjidku_backend/internals/features/school/classes/classes/model"
	enrolModel "masjidku_backend/internals/features/school/classes/class_sections/model"
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
		var uc ucModel.UserClassModel
		if err := h.DB.
			Select("user_classes_id, user_classes_masjid_id, user_classes_deleted_at").
			Where("user_classes_id = ? AND user_classes_deleted_at IS NULL", userClassID).
			First(&uc).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Enrolment (user_class) tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi enrolment")
		}
		if uc.UserClassMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Enrolment bukan milik masjid Anda")
		}
	}

	// Cek class_sections (tenant sama & belum terhapus)
	{
		type clsSection struct {
			ClassSectionID       uuid.UUID  `gorm:"column:class_section_id"`
			ClassSectionMasjidID uuid.UUID  `gorm:"column:class_section_masjid_id"`
			DeletedAt            *time.Time `gorm:"column:class_section_deleted_at"`
		}
		var sec clsSection
		if err := h.DB.
			Table("class_sections").
			Select("class_section_id, class_section_masjid_id, class_section_deleted_at").
			Where("class_section_id = ? AND class_section_deleted_at IS NULL", sectionID).
			First(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi section")
		}
		if sec.ClassSectionMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
		}
	}

	return nil
}

// Ambil row user_class_sections + pastikan tenant sama
func (h *UserClassSectionController) findUCSWithTenantGuard(ucsID, masjidID uuid.UUID) (*enrolModel.UserClassSection, error) {
	var m enrolModel.UserClassSection
	if err := h.DB.
		First(&m, "user_class_section_id = ? AND user_class_section_masjid_id = ?", ucsID, masjidID).Error; err != nil {
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
	tx := h.DB.Model(&enrolModel.UserClassSection{}).
		Where("user_class_section_user_class_id = ? AND user_class_section_unassigned_at IS NULL", userClassID)
	if excludeID != uuid.Nil {
		tx = tx.Where("user_class_section_id <> ?", excludeID)
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

	var req enrolDTO.CreateUserClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.UserClassSectionMasjidID = &masjidID

	// Validasi DTO
	if err := validateUCS.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Validasi tanggal ringan (DB juga sudah ada CHECK)
	if req.UserClassSectionAssignedAt != nil && req.UserClassSectionUnassignedAt != nil {
		if req.UserClassSectionUnassignedAt.Before(*req.UserClassSectionAssignedAt) {
			return fiber.NewError(fiber.StatusBadRequest, "unassigned_at tidak boleh sebelum assigned_at")
		}
	}

	// Guard tenant: pastikan parent entities memang milik masjid ini
	if err := h.ensureParentsBelongToMasjid(
		req.UserClassSectionUserClassID,
		req.UserClassSectionSectionID,
		masjidID,
	); err != nil {
		return err
	}

	// Satu placement aktif per user_class (aktif = unassigned_at IS NULL & belum soft delete)
	if req.UserClassSectionUnassignedAt == nil {
		if err := h.ensureSingleActivePerUserClass(req.UserClassSectionUserClassID, uuid.Nil); err != nil {
			return err
		}
	}

	// Simpan penempatan section (no extra services / stats)
	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat penempatan section")
	}

	return helper.JsonCreated(c, "Penempatan section berhasil dibuat", enrolDTO.NewUserClassSectionResponse(m))
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

	var req enrolDTO.UpdateUserClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// cegah pindah tenant
	req.UserClassSectionMasjidID = &masjidID

	if err := validateUCS.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// --- Guard parent (user_class_id & section_id) tetap dalam tenant yang sama ---
	targetUserClassID := existing.UserClassSectionUserClassID
	if req.UserClassSectionUserClassID != nil {
		targetUserClassID = *req.UserClassSectionUserClassID
	}
	targetSectionID := existing.UserClassSectionSectionID
	if req.UserClassSectionSectionID != nil {
		targetSectionID = *req.UserClassSectionSectionID
	}
	if targetUserClassID != existing.UserClassSectionUserClassID || targetSectionID != existing.UserClassSectionSectionID {
		if err := h.ensureParentsBelongToMasjid(targetUserClassID, targetSectionID, masjidID); err != nil {
			return err
		}
	}

	// --- Tentukan target unassigned_at (NULL = aktif) ---
	targetUnassigned := existing.UserClassSectionUnassignedAt
	if req.UserClassSectionUnassignedAt != nil {
		targetUnassigned = req.UserClassSectionUnassignedAt
	}

	// --- Jika akan menjadi AKTIF (unassigned_at == NULL) pastikan tidak ada duplikat aktif ---
	if targetUnassigned == nil {
		if err := h.ensureSingleActivePerUserClass(targetUserClassID, existing.UserClassSectionID); err != nil {
			return err
		}
	}

	// --- Apply & Save ---
	req.ApplyToModel(existing)
	if err := h.DB.Model(&enrolModel.UserClassSection{}).
		Where("user_class_section_id = ?", existing.UserClassSectionID).
		Updates(existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui penempatan section")
	}

	return helper.JsonUpdated(c, "Penempatan section berhasil diperbarui", enrolDTO.NewUserClassSectionResponse(existing))
}

// GET /admin/user-class-sections
func (h *UserClassSectionController) ListUserClassSections(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q enrolDTO.ListUserClassSectionQuery
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

	tx := h.DB.Model(&enrolModel.UserClassSection{}).
		Where("user_class_section_masjid_id = ?", masjidID)

	// Filters
	if q.UserClassID != nil {
		tx = tx.Where("user_class_section_user_class_id = ?", *q.UserClassID)
	}
	if q.SectionID != nil {
		tx = tx.Where("user_class_section_section_id = ?", *q.SectionID)
	}
	if q.ActiveOnly != nil && *q.ActiveOnly {
		tx = tx.Where("user_class_section_unassigned_at IS NULL")
	}

	// Sort
	sort := "assigned_at_desc"
	if q.Sort != nil {
		sort = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sort {
	case "assigned_at_asc":
		tx = tx.Order("user_class_section_assigned_at ASC")
	case "created_at_asc":
		tx = tx.Order("user_class_section_created_at ASC")
	case "created_at_desc":
		tx = tx.Order("user_class_section_created_at DESC")
	default:
		tx = tx.Order("user_class_section_assigned_at DESC")
	}

	// Paging
	tx = tx.Limit(q.Limit).Offset(q.Offset)

	// Fetch
	var rows []enrolModel.UserClassSection
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if len(rows) == 0 {
		return helper.JsonOK(c, "OK", []*enrolDTO.UserClassSectionResponse{})
	}

	/* ===== Enrichment ===== */

	// 1) Kumpulkan user_class_id unik
	ucSet := make(map[uuid.UUID]struct{}, len(rows))
	userClassIDs := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		id := rows[i].UserClassSectionUserClassID
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
	userMap := make(map[uuid.UUID]enrolDTO.UcsUser, len(userIDs))
	if len(userIDs) > 0 {
		var urs []struct {
			ID       uuid.UUID `gorm:"column:id"`
			UserName string    `gorm:"column:user_name"`
			FullName *string   `gorm:"column:full_name"`
			Email    string    `gorm:"column:email"`
			IsActive bool      `gorm:"column:is_active"`
		}
		if err := h.DB.
			Table("users").
			Select("id, user_name, full_name, email, is_active").
			Where("id IN ?", userIDs).
			Find(&urs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data user")
		}
		for _, u := range urs {
			userMap[u.ID] = enrolDTO.UcsUser{
				ID:       u.ID,
				UserName: u.UserName,
				FullName: u.FullName,
				Email:    u.Email,
				IsActive: u.IsActive,
			}
		}
	}

	// 6) Ambil user_profiles -> map[user_id]UcsUserProfile
	profileMap := make(map[uuid.UUID]enrolDTO.UcsUserProfile, len(userIDs))
	if len(userIDs) > 0 {
		var prs []struct {
			UserID                  uuid.UUID  `gorm:"column:user_id"`
			DonationName            *string    `gorm:"column:donation_name"`
			PhotoURL                *string    `gorm:"column:photo_url"`
			PhotoTrashURL           *string    `gorm:"column:photo_trash_url"`
			PhotoDeletePendingUntil *time.Time `gorm:"column:photo_delete_pending_until"`
			DateOfBirth             *time.Time `gorm:"column:date_of_birth"`
			Gender                  *string    `gorm:"column:gender"`
			PhoneNumber             *string    `gorm:"column:phone_number"`
			Bio                     *string    `gorm:"column:bio"`
			Location                *string    `gorm:"column:location"`
			Occupation              *string    `gorm:"column:occupation"`
		}
		if err := h.DB.
			Table("user_profiles").
			Select(`user_id, donation_name, photo_url, photo_trash_url, photo_delete_pending_until,
			        date_of_birth, gender, phone_number, bio, location, occupation`).
			Where("user_id IN ?", userIDs).
			Where("deleted_at IS NULL").
			Find(&prs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data profile")
		}
		for _, p := range prs {
			profileMap[p.UserID] = enrolDTO.UcsUserProfile{
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
	resp := make([]*enrolDTO.UserClassSectionResponse, 0, len(rows))
	for i := range rows {
		r := enrolDTO.NewUserClassSectionResponse(&rows[i])

		ucID := rows[i].UserClassSectionUserClassID
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
	if m.UserClassSectionUnassignedAt != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Penempatan sudah diakhiri sebelumnya")
	}

	// Kolom DATE di DB: pakai tanggal hari ini
	today := time.Now().Truncate(24 * time.Hour)

	updates := map[string]any{
		"user_class_section_unassigned_at": &today,
		"user_class_section_updated_at":    time.Now(),
	}

	if err := h.DB.Model(&enrolModel.UserClassSection{}).
		Where("user_class_section_id = ?", m.UserClassSectionID).
		Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengakhiri penempatan")
	}

	return helper.JsonUpdated(c, "Penempatan diakhiri", fiber.Map{
		"user_class_section_id":            m.UserClassSectionID,
		"user_class_section_unassigned_at": today,
		"is_active":                        false,
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
	if m.UserClassSectionUnassignedAt == nil {
		return fiber.NewError(fiber.StatusConflict, "Penempatan masih aktif, akhiri terlebih dahulu")
	}

	// Eksekusi delete
	db := h.DB
	if hard {
		// Hard delete permanen
		if err := db.Unscoped().
			Delete(&enrolModel.UserClassSection{}, "user_class_section_id = ?", m.UserClassSectionID).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus penempatan (hard)")
		}
		return helper.JsonDeleted(c, "Penempatan dihapus permanen", fiber.Map{
			"user_class_section_id": m.UserClassSectionID,
			"hard":                  true,
		})
	}

	// Soft delete (gorm.DeletedAt akan diisi otomatis)
	if err := db.
		Delete(&enrolModel.UserClassSection{}, "user_class_section_id = ?", m.UserClassSectionID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus penempatan (soft)")
	}

	return helper.JsonDeleted(c, "Penempatan dihapus", fiber.Map{
		"user_class_section_id": m.UserClassSectionID,
		"hard":                  false,
	})
}

// includeDeleted=false → hanya baris hidup (deleted_at IS NULL)
// includeDeleted=true  → cari Unscoped (termasuk yang sudah soft-deleted)
func (h *UserClassSectionController) findUCSWithTenantGuard2(
	ucsID, masjidID uuid.UUID,
	includeDeleted bool,
) (*enrolModel.UserClassSection, error) {
	var m enrolModel.UserClassSection

	q := h.DB.Model(&enrolModel.UserClassSection{})
	if includeDeleted {
		q = q.Unscoped()
	}
	q = q.Where("user_class_section_id = ? AND user_class_section_masjid_id = ?", ucsID, masjidID)
	if !includeDeleted {
		q = q.Where("user_class_section_deleted_at IS NULL")
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
	if m == nil || !m.UserClassSectionDeletedAt.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "Penempatan tidak dalam status terhapus")
	}

	// Pastikan tidak melanggar aturan "single active" saat restore (jika dia aktif)
	if m.UserClassSectionUnassignedAt == nil {
		if err := h.ensureSingleActivePerUserClass(m.UserClassSectionUserClassID, m.UserClassSectionID); err != nil {
			return err
		}
	}

	// Null-kan deleted_at (restore)
	if err := h.DB.Unscoped().Model(&enrolModel.UserClassSection{}).
		Where("user_class_section_id = ? AND user_class_section_masjid_id = ?", m.UserClassSectionID, masjidID).
		Update("user_class_section_deleted_at", nil).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memulihkan penempatan")
	}

	return helper.JsonOK(c, "Penempatan dipulihkan", enrolDTO.NewUserClassSectionResponse(m))
}
