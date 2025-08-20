package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"masjidku_backend/internals/features/lembaga/classes/main/dto"
	"masjidku_backend/internals/features/lembaga/classes/main/model"
	"masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
)

/* ================= Controller & Constructor ================= */

type ClassController struct {
	DB *gorm.DB
}

func NewClassController(db *gorm.DB) *ClassController {
	return &ClassController{DB: db}
}

// single validator instance for this package (tidak perlu di-inject)
var validate = validator.New()

/* ================= Handlers ================= */
// POST /admin/classes
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil { return err }

	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// paksa tenant
	req.ClassMasjidID = &masjidID

	// normalisasi & slug
	req.ClassName = strings.TrimSpace(req.ClassName)
	req.ClassSlug = strings.TrimSpace(req.ClassSlug)
	if req.ClassSlug == "" {
		req.ClassSlug = helper.GenerateSlug(req.ClassName)
	} else {
		req.ClassSlug = helper.GenerateSlug(req.ClassSlug)
	}

	// validasi
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// (opsional) upload gambar dari form field "class_image_url"
	if fh, ferr := c.FormFile("class_image_url"); ferr == nil && fh != nil {
		publicURL, upErr := helper.UploadImageToSupabase("classes", fh)
		if upErr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
		req.ClassImageURL = &publicURL
	}

	m := req.ToModel()

	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil { tx.Rollback(); panic(r) }
	}()

	// Cek slug unik PER MASJID (case-insensitive, soft-delete aware)
	var exists model.ClassModel
	findErr := tx.
		Where(
			"class_masjid_id = ? AND lower(class_slug) = lower(?) AND class_deleted_at IS NULL",
			masjidID, m.ClassSlug,
		).
		Take(&exists).Error
	if findErr == nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
	}
	if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, findErr.Error())
	}

	// Simpan
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}

	// Update lembaga_stats
	statsSvc := service.NewLembagaStatsService()
	if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
	}
	if err := statsSvc.IncActiveClasses(tx, masjidID, +1); err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Kelas berhasil dibuat", dto.NewClassResponse(m))
}



// UPDATE /admin/classes/:id  (multipart/form-data ATAU JSON)
func (ctrl *ClassController) UpdateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Parse payload (JSON / form)
	var req dto.UpdateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// --- Normalize name/slug ---
	if req.ClassSlug != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassSlug))
		req.ClassSlug = &s
	} else if req.ClassName != nil {
		// Regen slug dari name hanya kalau slug tidak dikirim
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassName))
		req.ClassSlug = &s
	}

	// Paksa tenant (tidak bisa diganti dari klien)
	req.ClassMasjidID = &masjidID

	// === (Opsional) Upload file (coba "class_image", fallback "class_image_url") ===
	if fh, err := c.FormFile("class_image"); err == nil && fh != nil {
		if publicURL, upErr := helper.UploadImageToSupabase("classes", fh); upErr == nil {
			req.ClassImageURL = &publicURL
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
	} else if fh, err := c.FormFile("class_image_url"); err == nil && fh != nil {
		if publicURL, upErr := helper.UploadImageToSupabase("classes", fh); upErr == nil {
			req.ClassImageURL = &publicURL
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
	}

	// Validasi DTO
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

	// Lock row + cek tenant
	var existing model.ClassModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&existing, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if existing.ClassMasjidID == nil || *existing.ClassMasjidID != masjidID {
		tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah kelas di masjid lain")
	}

	// Track perubahan status aktif (untuk lembaga_stats)
	wasActive := existing.ClassIsActive
	newActive := wasActive
	if req.ClassIsActive != nil {
		newActive = *req.ClassIsActive
	}

	// Cek unik slug PER MASJID saat slug berubah (case-insensitive, soft-delete aware)
	if req.ClassSlug != nil && *req.ClassSlug != existing.ClassSlug {
		var cnt int64
		if err := tx.Model(&model.ClassModel{}).
			Where(`
				class_masjid_id = ?
				AND lower(class_slug) = lower(?)
				AND class_id <> ?
				AND class_deleted_at IS NULL
			`, masjidID, *req.ClassSlug, existing.ClassID).
			Count(&cnt).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		} else if cnt > 0 {
			tx.Rollback()
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
	}

	// Jika URL gambar diganti manual, hapus file lama (best effort)
	if req.ClassImageURL != nil && existing.ClassImageURL != nil && *existing.ClassImageURL != *req.ClassImageURL {
		if bucket, path, exErr := helper.ExtractSupabasePath(*existing.ClassImageURL); exErr == nil {
			_ = helper.DeleteFromSupabase(bucket, path)
		}
	}

	// Apply perubahan ke model & simpan
	req.ApplyToModel(&existing)

	if err := tx.Model(&model.ClassModel{}).
		Where("class_id = ?", existing.ClassID).
		Updates(&existing).Error; err != nil {
		tx.Rollback()
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	// Sinkronkan lembaga_stats jika status aktif berubah
	if wasActive != newActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveClasses(tx, masjidID, delta); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// ===== TRANSACTION END =====

	return helper.JsonUpdated(c, "Kelas berhasil diperbarui", dto.NewClassResponse(&existing))
}


// GET /admin/classes/:id
func (ctrl *ClassController) GetClassByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.ClassModel
	if err := ctrl.DB.First(&m, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	// tenant check
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses kelas di masjid lain")
	}
	return helper.JsonOK(c, "Data diterima", dto.NewClassResponse(&m))
}



// GET /admin/classes
func (ctrl *ClassController) ListClasses(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q dto.ListClassQuery
	// default paging
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	// guard pagination
	if q.Limit <= 0 { q.Limit = 20 }
	if q.Limit > 200 { q.Limit = 200 }
	if q.Offset < 0 { q.Offset = 0 }

	tx := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_masjid_id = ?", masjidID).
		Where("class_deleted_at IS NULL")

	// filters
	if q.ActiveOnly != nil {
		tx = tx.Where("class_is_active = ?", *q.ActiveOnly)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
		tx = tx.Where("(LOWER(class_name) LIKE ? OR LOWER(class_level) LIKE ?)", s, s)
	}

	// total (sebelum limit/offset)
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// sorting whitelist
	sortVal := ""
	if q.Sort != nil {
		sortVal = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sortVal {
	case "name_asc":
		tx = tx.Order("class_name ASC")
	case "name_desc":
		tx = tx.Order("class_name DESC")
	case "created_at_asc":
		tx = tx.Order("class_created_at ASC")
	default:
		tx = tx.Order("class_created_at DESC")
	}

	// data
	var rows []model.ClassModel
	if err := tx.
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]*dto.ClassResponse, 0, len(rows))
	for i := range rows {
		items = append(items, dto.NewClassResponse(&rows[i]))
	}

	// gunakan JsonList agar konsisten: { data, pagination }
	return helper.JsonList(c, items, fiber.Map{
		"limit":  q.Limit,
		"offset": q.Offset,
		"total":  int(total),
	})
}




func (ctrl *ClassController) SoftDeleteClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	classID, err := uuid.Parse(c.Params("id"))
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

	// Lock row untuk hindari race
	var m model.ClassModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&m, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus kelas di masjid lain")
	}

	// Cek apakah sebelum dihapus dia aktif → nanti dipakai untuk decrement
	wasActive := m.ClassIsActive

	// Optional: hapus gambar
	deletedImage := false
	if strings.EqualFold(c.Query("delete_image"), "true") && m.ClassImageURL != nil && *m.ClassImageURL != "" {
		if bucket, path, exErr := helper.ExtractSupabasePath(*m.ClassImageURL); exErr == nil {
			_ = helper.DeleteFromSupabase(bucket, path)
			deletedImage = true
		}
		m.ClassImageURL = nil
	}

	now := time.Now()
	updates := map[string]any{
		"class_deleted_at": now,
		"class_is_active":  false,
		"class_updated_at": now,
	}
	if deletedImage {
		updates["class_image_url"] = nil
	}

	if err := tx.Model(&model.ClassModel{}).
		Where("class_id = ?", m.ClassID).
		Updates(updates).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	// Decrement stats jika sebelumnya aktif
	if wasActive {
		stats := service.NewLembagaStatsService()
		// pastikan baris stats ada (idempotent)
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		// -1 kelas aktif
		if err := stats.IncActiveClasses(tx, masjidID, -1); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Kelas berhasil dihapus", fiber.Map{
		"class_id":      m.ClassID,
		"deleted_image": deletedImage,
	})
}