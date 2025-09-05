// internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	semstats "masjidku_backend/internals/features/lembaga/stats/semester_stats/service"
	ucsDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	classModel "masjidku_backend/internals/features/school/classes/classes/model"
)

type ClassSectionController struct {
	DB *gorm.DB
}

func NewClassSectionController(db *gorm.DB) *ClassSectionController {
	return &ClassSectionController{DB: db}
}

/* ================= Handlers (ADMIN) ================= */

// GET /admin/class-sections/:id
func (ctrl *ClassSectionController) GetClassSectionByID(c *fiber.Ctx) error {
	// Ambil Masjid ID dari token
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan dalam token")
	}

	// Parse ID section
	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil section milik masjid ini dan belum soft-deleted
	var classSection secModel.ClassSectionModel
	if err := ctrl.DB.
		Where("class_sections_id = ? AND class_sections_masjid_id = ? AND class_sections_deleted_at IS NULL",
			sectionID, masjidID).
		First(&classSection).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data section")
	}

	// Ambil nama pengajar (opsional) dari relasi masjid_teachers → users
	var teacherName string
	if classSection.ClassSectionsTeacherID != nil {
		if err := ctrl.DB.Raw(`
			SELECT u.full_name
			FROM masjid_teachers mt
			JOIN users u ON mt.masjid_teacher_user_id = u.id
			WHERE mt.masjid_teacher_id = ?`, *classSection.ClassSectionsTeacherID).
			Scan(&teacherName).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data pengajar")
		}
	}

	// Response
	resp := ucsDTO.NewClassSectionResponse(&classSection, teacherName)
	return helper.JsonOK(c, "OK", resp)
}


// GET /admin/class-sections/:id/participants
// Mengambil peserta yang TERDAFTAR (masih assigned) pada section tertentu.
// - Filter tenant by masjid
// - Hanya baris dengan user_class_sections_unassigned_at IS NULL
// - Enrich: user_classes.status/started_at/ended_at, users, users_profile
func (ctrl *ClassSectionController) ListRegisteredParticipants(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Paging opsional
	limit := 50
	offset := 0
	if v := c.QueryInt("limit"); v > 0 && v <= 200 {
		limit = v
	}
	if v := c.QueryInt("offset"); v >= 0 {
		offset = v
	}

	// Ambil user_class_sections yg masih aktif (belum di-unassign)
	var rows []secModel.UserClassSectionsModel
	if err := ctrl.DB.
		Model(&secModel.UserClassSectionsModel{}).
		Where("user_class_sections_masjid_id = ?", masjidID).
		Where("user_class_sections_section_id = ?", sectionID).
		Where("user_class_sections_unassigned_at IS NULL").
		Order("user_class_sections_assigned_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil peserta")
	}
	if len(rows) == 0 {
		return helper.JsonOK(c, "OK", []*ucsDTO.UserClassSectionResponse{})
	}

	// ===== Enrichment (mirip ListUserClassSections) =====

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
	}

	ucMetaByID := make(map[uuid.UUID]ucMeta, len(userClassIDs))
	userIDByUC := make(map[uuid.UUID]uuid.UUID, len(userClassIDs))

	{
		var ucRows []ucMeta
		if err := ctrl.DB.
			Table("user_classes").
			Select("user_classes_id, user_classes_user_id, user_classes_status, user_classes_started_at").
			Where("user_classes_id IN ?", userClassIDs).
			Find(&ucRows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data enrolment")
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
		if err := ctrl.DB.
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
		if err := ctrl.DB.
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

	// 6) Build response
	resp := make([]*ucsDTO.UserClassSectionResponse, 0, len(rows))
	for i := range rows {
		r := ucsDTO.NewUserClassSectionResponse(&rows[i])

		ucID := rows[i].UserClassSectionsUserClassID
		if meta, ok := ucMetaByID[ucID]; ok {
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


// POST /admin/class-sections
func (ctrl *ClassSectionController) CreateClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req ucsDTO.CreateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Paksa tenant dari token (VALUE, bukan pointer)
	req.ClassSectionsMasjidID = masjidID

	// Slug otomatis
	nameForSlug := strings.TrimSpace(req.ClassSectionsSlug)
	if nameForSlug == "" {
		nameForSlug = req.ClassSectionsName
	}
	req.ClassSectionsSlug = helper.GenerateSlug(nameForSlug)
	if req.ClassSectionsSlug == "" {
		req.ClassSectionsSlug = "section-" + uuid.NewString()[:8]
	}

	// Sanity checks
	if strings.TrimSpace(req.ClassSectionsName) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Nama section wajib diisi")
	}
	if len(req.ClassSectionsSlug) > 160 {
		return fiber.NewError(fiber.StatusBadRequest, "Slug terlalu panjang (maksimal 160 karakter)")
	}
	if req.ClassSectionsCapacity != nil && *req.ClassSectionsCapacity < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

	// Map ke model
	m := req.ToModel()
	// Safety override lagi di model (jaga-jaga)
	m.ClassSectionsMasjidID = masjidID

	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// Validasi class wajib: ada & se-masjid
	{
		var cls classModel.ClassModel
		if err := tx.
			Select("class_id, class_masjid_id").
			Where("class_id = ? AND class_deleted_at IS NULL", req.ClassSectionsClassID).
			First(&cls).Error; err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class")
		}
		if cls.ClassMasjidID != masjidID {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusForbidden, "Class bukan milik masjid Anda")
		}
	}

	// Jika teacher_id diisi → pastikan se-masjid
	if req.ClassSectionsTeacherID != nil {
		var teacherMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT masjid_teacher_masjid_id
			FROM masjid_teachers
			WHERE masjid_teacher_id = ? AND masjid_teacher_deleted_at IS NULL
		`, *req.ClassSectionsTeacherID).Scan(&teacherMasjid).Error; err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if teacherMasjid == uuid.Nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusBadRequest, "Pengajar tidak ditemukan")
		}
		if teacherMasjid != masjidID {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusForbidden, "Pengajar bukan milik masjid Anda")
		}
	}

	// Jika class_room_id diisi → pastikan room ada, alive, dan se-masjid
	if req.ClassSectionsClassRoomID != nil {
		var roomMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT class_rooms_masjid_id
			FROM class_rooms
			WHERE class_room_id = ? AND deleted_at IS NULL
		`, *req.ClassSectionsClassRoomID).Scan(&roomMasjid).Error; err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
		}
		if roomMasjid == uuid.Nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
		}
		if roomMasjid != masjidID {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusForbidden, "Ruang kelas bukan milik masjid Anda")
		}
	}

	// Cek unik slug per masjid (case-insensitive)
	if err := tx.
		Clauses(clause.Locking{Strength: "SHARE"}).
		Where(`
			class_sections_masjid_id = ?
			AND lower(class_sections_slug) = lower(?)
			AND class_sections_deleted_at IS NULL
		`, masjidID, m.ClassSectionsSlug).
		First(&secModel.ClassSectionModel{}).Error; err == nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Simpan
	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat section")
	}

	// Update stats jika aktif
	if m.ClassSectionsIsActive {
		statsSvc := semstats.NewLembagaStatsService()
		if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveSections(tx, masjidID, +1); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Section berhasil dibuat", ucsDTO.NewClassSectionResponse(m, ""))
}

// PUT /admin/class-sections/:id
func (ctrl *ClassSectionController) UpdateClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Parse payload
	var req ucsDTO.UpdateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalisasi slug
	if req.ClassSectionsSlug != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassSectionsSlug))
		if s == "" {
			s = "section-" + uuid.NewString()[:8]
		}
		req.ClassSectionsSlug = &s
	} else if req.ClassSectionsName != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassSectionsName))
		if s == "" {
			s = "section-" + uuid.NewString()[:8]
		}
		req.ClassSectionsSlug = &s
	}

	// Paksa tenant dari token
	req.ClassSectionsMasjidID = &masjidID

	// Sanity checks ringan
	if req.ClassSectionsSlug != nil && len(*req.ClassSectionsSlug) > 160 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug terlalu panjang (maks 160)")
	}
	if req.ClassSectionsName != nil && strings.TrimSpace(*req.ClassSectionsName) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama section wajib diisi")
	}
	if req.ClassSectionsCapacity != nil && *req.ClassSectionsCapacity < 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

	// Begin TX
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// Ambil data existing + lock
	var existing secModel.ClassSectionModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).
		First(&existing).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Pastikan tenant cocok
	if existing.ClassSectionsMasjidID != masjidID {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh mengubah section milik masjid lain")
	}

	// Jika class_id berubah → pastikan class se-masjid
	if req.ClassSectionsClassID != nil {
		var cls struct {
			ClassMasjidID uuid.UUID `gorm:"column:class_masjid_id"`
		}
		if err := tx.
			Model(&classModel.ClassModel{}).
			Select("class_masjid_id").
			Where("class_id = ? AND class_deleted_at IS NULL", *req.ClassSectionsClassID).
			Take(&cls).Error; err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi class")
		}
		if cls.ClassMasjidID != masjidID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh memindahkan section ke class milik masjid lain")
		}
	}

	// Jika teacher_id berubah → pastikan se-masjid
	if req.ClassSectionsTeacherID != nil {
		var mt struct {
			MasjidTeacherMasjidID uuid.UUID `gorm:"column:masjid_teacher_masjid_id"`
		}
		if err := tx.
			Table("masjid_teachers").
			Select("masjid_teacher_masjid_id").
			Where("masjid_teacher_id = ? AND masjid_teacher_deleted_at IS NULL", *req.ClassSectionsTeacherID).
			Take(&mt).Error; err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Pengajar tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi pengajar")
		}
		if mt.MasjidTeacherMasjidID != masjidID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Pengajar bukan milik masjid Anda")
		}
	}

	// Jika class_room_id berubah/diisi → pastikan room ada, alive, se-masjid
	if req.ClassSectionsClassRoomID != nil {
		var room struct {
			ClassRoomsMasjidID uuid.UUID `gorm:"column:class_rooms_masjid_id"`
		}
		if err := tx.
			Table("class_rooms").
			Select("class_rooms_masjid_id").
			Where("class_room_id = ? AND class_rooms_deleted_at IS NULL", *req.ClassSectionsClassRoomID).
			Take(&room).Error; err != nil {
			_ = tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
		}
		if room.ClassRoomsMasjidID != masjidID {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusForbidden, "Ruang kelas bukan milik masjid Anda")
		}
	}

	// Cek unik slug per-masjid (case-insensitive) jika slug diubah
	if req.ClassSectionsSlug != nil && !strings.EqualFold(*req.ClassSectionsSlug, existing.ClassSectionsSlug) {
		var cnt int64
		if err := tx.Model(&secModel.ClassSectionModel{}).
			Where(`
				class_sections_masjid_id = ?
				AND lower(class_sections_slug) = lower(?)
				AND class_sections_id <> ?
				AND class_sections_deleted_at IS NULL
			`, masjidID, *req.ClassSectionsSlug, existing.ClassSectionsID).
			Count(&cnt).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		if cnt > 0 {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan")
		}
	}

	// Cek unik (class_id, name) jika berubah
	targetClassID := existing.ClassSectionsClassID
	if req.ClassSectionsClassID != nil {
		targetClassID = *req.ClassSectionsClassID
	}
	targetName := existing.ClassSectionsName
	if req.ClassSectionsName != nil {
		targetName = *req.ClassSectionsName
	}
	if targetClassID != existing.ClassSectionsClassID || targetName != existing.ClassSectionsName {
		var cnt int64
		if err := tx.Model(&secModel.ClassSectionModel{}).
			Where(`
				class_sections_class_id = ?
				AND class_sections_name = ?
				AND class_sections_id <> ?
				AND class_sections_deleted_at IS NULL
			`, targetClassID, targetName, existing.ClassSectionsID).
			Count(&cnt).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		if cnt > 0 {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusConflict, "Nama section sudah dipakai pada class ini")
		}
	}

	// Track perubahan status aktif
	wasActive := existing.ClassSectionsIsActive
	newActive := wasActive
	if req.ClassSectionsIsActive != nil {
		newActive = *req.ClassSectionsIsActive
	}

	// Apply perubahan & simpan
	req.ApplyToModel(&existing)

	// Pakai Save supaya nilai nol/false ikut ter-update
	if err := tx.Save(&existing).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui section")
	}

	// Update lembaga_stats jika status aktif berubah
	if wasActive != newActive {
		stats := semstats.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveSections(tx, masjidID, delta); err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Section berhasil diperbarui", ucsDTO.NewClassSectionResponse(&existing, ""))
}



// DELETE /admin/class-sections/:id  (soft delete)
// DELETE /admin/class-sections/:id (soft delete)
func (ctrl *ClassSectionController) SoftDeleteClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// Lock row & pastikan belum terhapus
	var m secModel.ClassSectionModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&m, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Guard tenant (sekarang value, bukan pointer)
	if m.ClassSectionsMasjidID != masjidID {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus section milik masjid lain")
	}

	wasActive := m.ClassSectionsIsActive
	now := time.Now()

	// Soft delete manual (set deleted_at, inaktifkan, sentuh updated_at)
	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", m.ClassSectionsID).
		Updates(map[string]any{
			"class_sections_deleted_at": now,
			"class_sections_is_active":  false,
			"class_sections_updated_at": now,
		}).Error; err != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus section")
	}

	// Update stats bila sebelumnya aktif
	if wasActive {
		stats := semstats.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveSections(tx, masjidID, -1); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Section berhasil dihapus", fiber.Map{
		"class_sections_id": m.ClassSectionsID,
	})
}
