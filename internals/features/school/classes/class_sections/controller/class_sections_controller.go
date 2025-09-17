// file: internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
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

// POST /admin/class-sections
func (ctrl *ClassSectionController) CreateClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req ucsDTO.ClassSectionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// paksa tenant
	req.ClassSectionsMasjidID = masjidID

	// generate slug bila kosong / normalisasi
	slugBase := strings.TrimSpace(req.ClassSectionsSlug)
	if slugBase == "" {
		slugBase = req.ClassSectionsName
	}
	req.ClassSectionsSlug = helper.GenerateSlug(slugBase)
	if req.ClassSectionsSlug == "" {
		req.ClassSectionsSlug = "section-" + uuid.NewString()[:8]
	}

	// sanity
	if strings.TrimSpace(req.ClassSectionsName) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Nama section wajib diisi")
	}
	if len(req.ClassSectionsSlug) > 160 {
		return fiber.NewError(fiber.StatusBadRequest, "Slug terlalu panjang (maksimal 160)")
	}
	if req.ClassSectionsCapacity != nil && *req.ClassSectionsCapacity < 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

	// map ke model
	m := req.ToModel()
	m.ClassSectionsMasjidID = masjidID // enforce

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

	// validasi class se-masjid
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

	// validasi teacher se-masjid
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

	// validasi room se-masjid
	if req.ClassSectionsClassRoomID != nil {
		var roomMasjid uuid.UUID
		if err := tx.Raw(`
			SELECT class_rooms_masjid_id
			FROM class_rooms
			WHERE class_room_id = ? AND class_rooms_deleted_at IS NULL
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

	// unik slug per masjid
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

	// simpan
	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat section")
	}

	// update stats bila aktif
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

	return helper.JsonCreated(c, "Section berhasil dibuat", ucsDTO.FromModelClassSection(m))
}

// PATCH /admin/class-sections/:id   (PATCH semantics)
// PATCH /admin/class-sections/:id   (PATCH semantics)
func (ctrl *ClassSectionController) UpdateClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req ucsDTO.ClassSectionPatchRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalisasi/generasi slug hanya jika slug atau name dikirim
	if req.ClassSectionsSlug.Present && req.ClassSectionsSlug.Value != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassSectionsSlug.Value))
		if s == "" {
			s = "section-" + uuid.NewString()[:8]
		}
		req.ClassSectionsSlug.Value = &s
	} else if req.ClassSectionsName.Present && req.ClassSectionsName.Value != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassSectionsName.Value))
		if s == "" {
			s = "section-" + uuid.NewString()[:8]
		}
		// set juga slug agar konsisten
		req.ClassSectionsSlug.Present = true
		req.ClassSectionsSlug.Value = &s
	}

	// Sanity ringan
	if req.ClassSectionsSlug.Present && req.ClassSectionsSlug.Value != nil && len(*req.ClassSectionsSlug.Value) > 160 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug terlalu panjang (maks 160)")
	}
	if req.ClassSectionsName.Present && req.ClassSectionsName.Value != nil && strings.TrimSpace(*req.ClassSectionsName.Value) == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama section wajib diisi")
	}
	if req.ClassSectionsCapacity.Present && req.ClassSectionsCapacity.Value != nil && *req.ClassSectionsCapacity.Value < 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Capacity tidak boleh negatif")
	}

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

	// Tenant guard
	if existing.ClassSectionsMasjidID != masjidID {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusForbidden, "Tidak boleh mengubah section milik masjid lain")
	}

	// Validasi teacher jika berubah
	if req.ClassSectionsTeacherID.Present && req.ClassSectionsTeacherID.Value != nil {
		var mt struct{ MasjidTeacherMasjidID uuid.UUID `gorm:"column:masjid_teacher_masjid_id"` }
		if err := tx.
			Table("masjid_teachers").
			Select("masjid_teacher_masjid_id").
			Where("masjid_teacher_id = ? AND masjid_teacher_deleted_at IS NULL", *req.ClassSectionsTeacherID.Value).
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

	// Validasi room jika berubah
	if req.ClassSectionsClassRoomID.Present && req.ClassSectionsClassRoomID.Value != nil {
		var room struct{ ClassRoomsMasjidID uuid.UUID `gorm:"column:class_rooms_masjid_id"` }
		if err := tx.
			Table("class_rooms").
			Select("class_rooms_masjid_id").
			Where("class_room_id = ? AND class_rooms_deleted_at IS NULL", *req.ClassSectionsClassRoomID.Value).
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

	// Cek unik slug per masjid jika slug diubah
	if req.ClassSectionsSlug.Present && req.ClassSectionsSlug.Value != nil &&
		!strings.EqualFold(*req.ClassSectionsSlug.Value, existing.ClassSectionsSlug) {
		var cnt int64
		if err := tx.Model(&secModel.ClassSectionModel{}).
			Where(`
				class_sections_masjid_id = ?
				AND lower(class_sections_slug) = lower(?)
				AND class_sections_id <> ?
				AND class_sections_deleted_at IS NULL
			`, masjidID, *req.ClassSectionsSlug.Value, existing.ClassSectionsID).
			Count(&cnt).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		if cnt > 0 {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan")
		}
	}

	// Cek unik (class_id, name) — hanya jika NAME berubah (class_id tidak diubah via PATCH)
	targetName := existing.ClassSectionsName
	if req.ClassSectionsName.Present && req.ClassSectionsName.Value != nil {
		targetName = strings.TrimSpace(*req.ClassSectionsName.Value)
	}
	if !strings.EqualFold(targetName, existing.ClassSectionsName) {
		var cnt int64
		if err := tx.Model(&secModel.ClassSectionModel{}).
			Where(`
				class_sections_class_id = ?
				AND class_sections_name = ?
				AND class_sections_id <> ?
				AND class_sections_deleted_at IS NULL
			`, existing.ClassSectionsClassID, targetName, existing.ClassSectionsID).
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
	if req.ClassSectionsIsActive.Present && req.ClassSectionsIsActive.Value != nil {
		newActive = *req.ClassSectionsIsActive.Value
	}

	// Apply patch & save
	req.Normalize()
	req.Apply(&existing)
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

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Section berhasil diperbarui", ucsDTO.FromModelClassSection(&existing))
}


// DELETE /admin/class-sections/:id (soft delete)
func (ctrl *ClassSectionController) SoftDeleteClassSection(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
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

	if m.ClassSectionsMasjidID != masjidID {
		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus section milik masjid lain")
	}

	wasActive := m.ClassSectionsIsActive
	now := time.Now()

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
