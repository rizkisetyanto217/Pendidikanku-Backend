// internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "masjidku_backend/internals/helpers"

	secDTO "masjidku_backend/internals/features/lembaga/class_sections/main/dto"
	secModel "masjidku_backend/internals/features/lembaga/class_sections/main/model"
	classModel "masjidku_backend/internals/features/lembaga/classes/main/model"
	"masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"
)

type ClassSectionController struct {
	DB *gorm.DB
}

func NewClassSectionController(db *gorm.DB) *ClassSectionController {
	return &ClassSectionController{DB: db}
}

var validate = validator.New()

/* ================= Handlers (ADMIN) ================= */

// POST /admin/class-sections
func (ctrl *ClassSectionController) CreateClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req secDTO.CreateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// force tenant
	req.ClassSectionsMasjidID = &masjidID

	// === AUTO SLUG ===
	if strings.TrimSpace(req.ClassSectionsSlug) == "" {
		req.ClassSectionsSlug = helper.NormalizeSlug(req.ClassSectionsName)
	} else {
		req.ClassSectionsSlug = helper.NormalizeSlug(req.ClassSectionsSlug)
	}
	if req.ClassSectionsSlug == "" {
		req.ClassSectionsSlug = "section-" + uuid.NewString()[:8]
	}

	// Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Mapping ke model
	m := req.ToModel()

	// === TRANSACTION START ===
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Cek unik slug (lock ringan biar anti-race)
	if err := tx.
		Clauses(clause.Locking{Strength: "SHARE"}).
		Where("class_sections_slug = ? AND class_sections_deleted_at IS NULL", m.ClassSectionsSlug).
		First(&secModel.ClassSectionModel{}).Error; err == nil {
		// ada row ⇒ slug sudah dipakai
		tx.Rollback()
		return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// error lain (bukan "tidak ditemukan")
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Simpan section
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat section")
	}

	// === Update lembaga_stats: +1 jika section AKTIF ===
	// Asumsi field boolean di model: m.ClassSectionsIsActive
	if m.ClassSectionsIsActive {
		statsSvc := service.NewLembagaStatsService()
		if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveSections(tx, masjidID, +1); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// === TRANSACTION END ===

	return helper.JsonCreated(c, "Section berhasil dibuat", secDTO.NewClassSectionResponse(m))
}


// PUT /admin/class-sections/:id
func (ctrl *ClassSectionController) UpdateClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Parse & normalisasi payload lebih dulu
	var req secDTO.UpdateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalisasi slug bila dikirim; atau auto dari name bila name dikirim
	if req.ClassSectionsSlug != nil {
		s := helper.NormalizeSlug(*req.ClassSectionsSlug)
		if s == "" {
			s = "section-" + uuid.NewString()[:8]
		}
		req.ClassSectionsSlug = &s
	} else if req.ClassSectionsName != nil {
		s := helper.NormalizeSlug(*req.ClassSectionsName)
		if s == "" {
			s = "section-" + uuid.NewString()[:8]
		}
		req.ClassSectionsSlug = &s
	}

	// Cegah ganti tenant dari luar
	req.ClassSectionsMasjidID = &masjidID

	// Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ===== TRANSACTION START =====
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Ambil existing + LOCK (hindari race)
	var existing secModel.ClassSectionModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&existing, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Guard tenant
	if existing.ClassSectionsMasjidID == nil || *existing.ClassSectionsMasjidID != masjidID {
		tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah section milik masjid lain")
	}

	// Jika class_id diganti, validasi class milik tenant
	if req.ClassSectionsClassID != nil {
		var cls classModel.ClassModel
		if err := tx.
			Select("class_id, class_masjid_id").
			First(&cls, "class_id = ? AND class_deleted_at IS NULL", *req.ClassSectionsClassID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class")
		}
		if cls.ClassMasjidID == nil || *cls.ClassMasjidID != masjidID {
			tx.Rollback()
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh memindahkan section ke class milik masjid lain")
		}
	}

	// Cek unik slug (exclude current)
	if req.ClassSectionsSlug != nil && *req.ClassSectionsSlug != existing.ClassSectionsSlug {
		var cnt int64
		if err := tx.Model(&secModel.ClassSectionModel{}).
			Where("class_sections_slug = ? AND class_sections_id <> ? AND class_sections_deleted_at IS NULL",
				*req.ClassSectionsSlug, existing.ClassSectionsID).
			Count(&cnt).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		} else if cnt > 0 {
			tx.Rollback()
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
		}
	}

	// Cek unik (class_id, name) exclude current
	targetClassID := existing.ClassSectionsClassID
	if req.ClassSectionsClassID != nil {
		targetClassID = *req.ClassSectionsClassID
	}
	targetName := existing.ClassSectionsName
	if req.ClassSectionsName != nil {
		targetName = *req.ClassSectionsName
	}
	{
		var cnt int64
		if err := tx.Model(&secModel.ClassSectionModel{}).
			Where(`class_sections_class_id = ?
			       AND class_sections_name = ?
			       AND class_sections_id <> ?
			       AND class_sections_deleted_at IS NULL`,
				targetClassID, targetName, existing.ClassSectionsID).
			Count(&cnt).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		} else if cnt > 0 {
			tx.Rollback()
			return fiber.NewError(fiber.StatusConflict, "Nama section sudah dipakai pada class ini")
		}
	}

	// Hitung perubahan status aktif
	wasActive := existing.ClassSectionsIsActive
	newActive := wasActive
	if req.ClassSectionsIsActive != nil {
		newActive = *req.ClassSectionsIsActive
	}

	// Apply & save
	req.ApplyToModel(&existing)
	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", existing.ClassSectionsID).
		Updates(&existing).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui section")
	}

	// Sinkronkan lembaga_stats jika status aktif berubah
	if wasActive != newActive {
		stats := service.NewLembagaStatsService()
		// pastikan baris stats ada (idempotent)
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveSections(tx, masjidID, delta); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// ===== TRANSACTION END =====

	return helper.JsonUpdated(c, "Section berhasil diperbarui", secDTO.NewClassSectionResponse(&existing))
}


// GET /admin/class-sections/:id
func (ctrl *ClassSectionController) GetClassSectionByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m secModel.ClassSectionModel
	if err := ctrl.DB.First(&m, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassSectionsMasjidID == nil || *m.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses section milik masjid lain")
	}
	return helper.JsonOK(c, "OK", secDTO.NewClassSectionResponse(&m))
}


// GET /admin/class-sections
func (ctrl *ClassSectionController) ListClassSections(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q secDTO.ListClassSectionQuery
	q.Limit = 20
	q.Offset = 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_deleted_at IS NULL").
		Where("class_sections_masjid_id = ?", masjidID)

	if q.ActiveOnly != nil {
		tx = tx.Where("class_sections_is_active = ?", *q.ActiveOnly)
	}
	if q.ClassID != nil {
		tx = tx.Where("class_sections_class_id = ?", *q.ClassID)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
		tx = tx.Where("(LOWER(class_sections_name) LIKE ? OR LOWER(class_sections_code) LIKE ?)", s, s)
	}

	sortVal := ""
	if q.Sort != nil {
		sortVal = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sortVal {
	case "name_asc":
		tx = tx.Order("class_sections_name ASC")
	case "name_desc":
		tx = tx.Order("class_sections_name DESC")
	case "created_at_asc":
		tx = tx.Order("class_sections_created_at ASC")
	default:
		tx = tx.Order("class_sections_created_at DESC")
	}

	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]*secDTO.ClassSectionResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, secDTO.NewClassSectionResponse(&rows[i]))
	}
	return helper.JsonOK(c, "OK", resp)
}


// DELETE /admin/class-sections/:id  (soft delete)
func (ctrl *ClassSectionController) SoftDeleteClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
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
			tx.Rollback()
			panic(r)
		}
	}()

	// Lock row agar anti race & pastikan belum soft-deleted
	var m secModel.ClassSectionModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&m, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Guard tenant
	if m.ClassSectionsMasjidID == nil || *m.ClassSectionsMasjidID != masjidID {
		tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus section milik masjid lain")
	}

	// simpan status aktif sebelum delete
	wasActive := m.ClassSectionsIsActive

	// Soft delete
	now := time.Now()
	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", m.ClassSectionsID).
		Updates(map[string]any{
			"class_sections_deleted_at": now,
			"class_sections_is_active":  false,
			"class_sections_updated_at": now,
		}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus section")
	}

	// Decrement stats hanya jika sebelumnya aktif
	if wasActive {
		stats := service.NewLembagaStatsService()
		// pastikan baris stats ada
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveSections(tx, masjidID, -1); err != nil {
			tx.Rollback()
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
