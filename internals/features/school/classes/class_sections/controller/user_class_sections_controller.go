// internals/features/lembaga/classes/user_class_sections/main/controller/user_class_section_controller.go
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

	semstats "masjidku_backend/internals/features/lembaga/stats/semester_stats/service"

	// untuk validasi parent (cek tenant & eksistensi)
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
func (h *UserClassSectionController) ensureParentsBelongToMasjid(userClassID, sectionID, masjidID uuid.UUID) error {
	// --- Cek user_classes (harus belum soft-deleted & tenant sama)
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

		// ✅ uc.UserClassesMasjidID adalah uuid.UUID (value), cukup compare langsung
		if uc.UserClassesMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Enrolment bukan milik masjid Anda")
		}
	}

	// --- Cek class_sections (harus belum soft-deleted & tenant sama)
	{
		var sec secModel.ClassSectionModel
		if err := h.DB.
			Select("class_sections_id, class_sections_masjid_id, class_sections_deleted_at").
			Where("class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).
			First(&sec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi section")
		}

		// Jika di model kamu ClassSectionsMasjidID bertipe *uuid.UUID (pointer), tetap gunakan pengecekan ini:
		if sec.ClassSectionsMasjidID == nil || *sec.ClassSectionsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
		}
		// Jika nantinya kamu ubah jadi uuid.UUID (value), ganti blok di atas dengan:
		// if sec.ClassSectionsMasjidID != masjidID { ... }
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
		Where("user_class_sections_user_class_id = ? AND user_class_sections_unassigned_at IS NULL",
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


// internals/features/lembaga/classes/user_class_sections/main/controller/user_class_section_controller.go

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

	// ===== TRANSACTION START =====
	tx := h.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Simpan penempatan section
	m := req.ToModel()
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		// Tanpa AsPGError: cukup response generic (pre-check di atas sudah mencegah duplikasi aktif)
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat penempatan section")
	}

	// Gunakan assigned_at sebagai anchor utk menentukan semester kalender
	anchor := m.UserClassSectionsAssignedAt

	// Upsert 1 baris semester stats (idempotent via ON CONFLICT DO NOTHING)
	semSvc := semstats.NewSemesterStatsService()
	if err := semSvc.EnsureSemesterStatsForUserClassWithAnchor(
		tx,
		masjidID,
		req.UserClassSectionsUserClassID,
		req.UserClassSectionsSectionID,
		anchor,
	); err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi semester stats: "+err.Error())
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// ===== TRANSACTION END =====

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
	if q.Status != nil {
		switch strings.ToLower(strings.TrimSpace(*q.Status)) {
		case "active":
			tx = tx.Where("user_class_sections_unassigned_at IS NULL")
		case "inactive":
			tx = tx.Where("user_class_sections_unassigned_at IS NOT NULL")
		}
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

	// Fetch rows
	var rows []secModel.UserClassSectionsModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	// Early return kalau tidak ada data
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

	// 2) Ambil mapping user_class -> (user_id, status, started_at)
	type ucMeta struct {
		UserClassID uuid.UUID  `gorm:"column:user_classes_id"`
		UserID      uuid.UUID  `gorm:"column:user_classes_user_id"`
		Status      string     `gorm:"column:user_classes_status"`
		StartedAt   *time.Time `gorm:"column:user_classes_started_at"`
		// EndedAt dihapus karena kolomnya sudah tidak ada
	}

	ucMetaByID := make(map[uuid.UUID]ucMeta, len(userClassIDs))
	userIDByUC := make(map[uuid.UUID]uuid.UUID, len(userClassIDs))

	{
		var ucRows []ucMeta
		if err := h.DB.
			Table("user_classes").
			Select("user_classes_id, user_classes_user_id, user_classes_status, user_classes_started_at").
			Where("user_classes_id IN ?", userClassIDs).
			Find(&ucRows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data user_class")
		}
		for _, r := range ucRows {
			ucMetaByID[r.UserClassID] = r
			userIDByUC[r.UserClassID] = r.UserID
		}
	}


	// 3) Kumpulkan user_id unik
	uSet := make(map[uuid.UUID]struct{}, len(userClassIDs))
	userIDs := make([]uuid.UUID, 0, len(userClassIDs))
	for _, uc := range userClassIDs {
		if uid, ok := userIDByUC[uc]; ok {
			if _, seen := uSet[uid]; !seen {
				uSet[uid] = struct{}{}
				userIDs = append(userIDs, uid)
			}
		}
	}

	// 4) Ambil users -> map[user_id]UcsUser
	userMap := make(map[uuid.UUID]ucsDTO.UcsUser, len(userIDs))
	if len(userIDs) > 0 {
		var urs []struct {
			ID       uuid.UUID `gorm:"column:id"`
			UserName string    `gorm:"column:user_name"`
			Email    string    `gorm:"column:email"`
			IsActive bool      `gorm:"column:is_active"`
		}
		if err := h.DB.
			Table("users").
			Select("id, user_name, email, is_active").
			Where("id IN ?", userIDs).
			Find(&urs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data user")
		}
		for _, u := range urs {
			userMap[u.ID] = ucsDTO.UcsUser{
				ID:       u.ID,
				UserName: u.UserName,
				Email:    u.Email,
				IsActive: u.IsActive,
			}
		}
	}

	// 5) Ambil users_profile -> map[user_id]UcsUserProfile
	profileMap := make(map[uuid.UUID]ucsDTO.UcsUserProfile, len(userIDs))
	if len(userIDs) > 0 {
		var prs []struct {
			UserID       uuid.UUID  `gorm:"column:user_id"`
			DonationName string     `gorm:"column:donation_name"`
			FullName     string     `gorm:"column:full_name"`
			FatherName   string     `gorm:"column:father_name"`
			MotherName   string     `gorm:"column:mother_name"`
			DateOfBirth  *time.Time `gorm:"column:date_of_birth"`
			Gender       *string    `gorm:"column:gender"`
			PhoneNumber  string     `gorm:"column:phone_number"`
			Bio          string     `gorm:"column:bio"`
			Location     string     `gorm:"column:location"`
			Occupation   string     `gorm:"column:occupation"`
		}
		if err := h.DB.
			Table("users_profile").
			Select(`user_id, donation_name, full_name, father_name, mother_name,
			        date_of_birth, gender, phone_number, bio, location, occupation`).
			Where("user_id IN ?", userIDs).
			Where("deleted_at IS NULL").
			Find(&prs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data profile")
		}
		for _, p := range prs {
			profileMap[p.UserID] = ucsDTO.UcsUserProfile{
				UserID:       p.UserID,
				DonationName: p.DonationName,
				FullName:     p.FullName,
				FatherName:   p.FatherName,
				MotherName:   p.MotherName,
				DateOfBirth:  p.DateOfBirth,
				Gender:       p.Gender,
				PhoneNumber:  p.PhoneNumber,
				Bio:          p.Bio,
				Location:     p.Location,
				Occupation:   p.Occupation,
			}
		}
	}

	// 6) Build response + set user_classes status & enrichments
	resp := make([]*ucsDTO.UserClassSectionResponse, 0, len(rows))
	for i := range rows {
		r := ucsDTO.NewUserClassSectionResponse(&rows[i])

		ucID := rows[i].UserClassSectionsUserClassID
		if meta, ok := ucMetaByID[ucID]; ok {
			// isi status dari user_classes (pastikan field ada di DTO kamu)
			r.UserClassesStatus = meta.Status
		}

		if uid, ok := userIDByUC[ucID]; ok {
			if u, ok := userMap[uid]; ok {
				uCopy := u
				r.User = &uCopy
			}
			if p, ok := profileMap[uid]; ok {
				pCopy := p
				r.Profile = &pCopy
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

	// Idempotent: jika sudah diakhiri, beri pesan jelas
	if m.UserClassSectionsUnassignedAt != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Penempatan sudah diakhiri sebelumnya")
	}

	now := time.Now()
	updates := map[string]any{
		// status dihapus — cukup set unassigned_at sebagai penanda sudah diakhiri
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
		"user_class_sections_unassigned_at": now,
		"is_active":                         false, // kompas kompatibilitas: aktif = unassigned_at == NULL
	})
}

// internals/features/lembaga/classes/user_class_sections/main/controller/user_class_section_controller.go

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
func (h *UserClassSectionController) findUCSWithTenantGuard2(id, masjidID uuid.UUID, includeDeleted bool) (*secModel.UserClassSectionsModel, error) {
	var m secModel.UserClassSectionsModel
	q := h.DB.Model(&secModel.UserClassSectionsModel{})

	if includeDeleted {
		q = q.Unscoped()
	} // else: default GORM otomatis exclude soft-deleted

	if err := q.
		Where("user_class_sections_id = ? AND user_class_sections_masjid_id = ?", id, masjidID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "Penempatan tidak ditemukan/di luar tenant")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data penempatan")
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
