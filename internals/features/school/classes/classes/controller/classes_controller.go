package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"
	"masjidku_backend/internals/features/school/classes/classes/dto"
	"masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"

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




// POST /admin/classes
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// 🔐 Paksa tenant
	req.ClassMasjidID = masjidID

	// 🧹 Normalisasi & slug dasar
	req.ClassName = strings.TrimSpace(req.ClassName)
	req.ClassSlug = strings.TrimSpace(req.ClassSlug)
	baseSlug := req.ClassSlug
	if baseSlug == "" {
		baseSlug = req.ClassName
	}

	// ✅ Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 🖼️ (Opsional) upload gambar → otomatis konversi ke WebP
	if fh, ferr := c.FormFile("class_image_url"); ferr == nil && fh != nil {
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Init OSS gagal: "+err.Error())
		}
		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()

		// path: masjids/{masjidID}/classes
		dir := fmt.Sprintf("masjids/%s/classes", masjidID.String())

		publicURL, upErr := svc.UploadAsWebP(ctx, fh, dir)
		if upErr != nil {
			low := strings.ToLower(upErr.Error())
			if strings.Contains(low, "format tidak didukung") {
				return fiber.NewError(fiber.StatusUnsupportedMediaType, "Format tidak didukung (jpg/png/webp)")
			}
			return fiber.NewError(fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		req.ClassImageURL = &publicURL
	}

	m := req.ToModel() // -> *model.ClassModel

	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			panic(r)
		}
	}()

	// 🏷️ Generate slug unik per masjid
	slugOpts := helper.SlugOptions{
		Table:            "classes",
		SlugColumn:       "class_slug",
		SoftDeleteColumn: "class_deleted_at",
		Filters:          map[string]any{"class_masjid_id": masjidID},
		MaxLen:           160,
		DefaultBase:      "kelas",
	}
	uniqueSlug, err := helper.GenerateUniqueSlug(tx, slugOpts, baseSlug)
	if err != nil {
		_ = tx.Rollback().Error
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik: "+err.Error())
	}
	m.ClassSlug = uniqueSlug

	// 💾 Simpan
	if err := tx.Create(m).Error; err != nil {
		low := strings.ToLower(err.Error())

		// Konflik slug
		if strings.Contains(low, "uq_classes_slug_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")) {
			if reSlug, rErr := helper.GenerateUniqueSlug(tx, slugOpts, baseSlug); rErr == nil {
				m.ClassSlug = reSlug
				if e2 := tx.Create(m).Error; e2 == nil {
					goto SAVE_OK
				}
			}
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}

		// Konflik code
		if strings.Contains(low, "uq_classes_code_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_code")) {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Kode kelas sudah digunakan di masjid ini")
		}

		_ = tx.Rollback().Error
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}

SAVE_OK:
	// 📈 Update lembaga_stats
	if m.ClassIsActive {
		statsSvc := service.NewLembagaStatsService()
		if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveClasses(tx, masjidID, +1); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Kelas berhasil dibuat", dto.NewClassResponse(m))
}




// UPDATE /admin/classes/:id
// UPDATE /admin/classes/:id
func (ctrl *ClassController) UpdateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// --- Parse path param
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// --- Parse payload
	var req dto.UpdateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// --- Normalisasi slug: jika user kirim slug → slugify; jika tidak & user kirim name → turunkan slug dari name
	if req.ClassSlug != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassSlug))
		req.ClassSlug = &s
	} else if req.ClassName != nil {
		s := helper.GenerateSlug(strings.TrimSpace(*req.ClassName))
		req.ClassSlug = &s
	}

	// --- Tenant: tidak boleh dipindah masjid
	// NOTE: sesuaikan tipe DTO kamu:
	// - jika *uuid.UUID -> req.ClassMasjidID = &masjidID
	// - jika uuid.UUID  -> req.ClassMasjidID = masjidID
	req.ClassMasjidID = &masjidID

	// --- Upload gambar baru (field: class_image_url) → konversi WebP via helper
	if fh, ferr := c.FormFile("class_image_url"); ferr == nil && fh != nil {
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Init OSS gagal: "+err.Error())
		}
		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()

		dir := fmt.Sprintf("masjids/%s/classes", masjidID.String())
		publicURL, upErr := svc.UploadAsWebP(ctx, fh, dir) // jpg/png → .webp; webp passthrough
		if upErr != nil {
			low := strings.ToLower(upErr.Error())
			if strings.Contains(low, "format tidak didukung") {
				return fiber.NewError(fiber.StatusUnsupportedMediaType, "Format tidak didukung (jpg/png/webp)")
			}
			return fiber.NewError(fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		req.ClassImageURL = &publicURL
	}

	// --- Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// --- TX
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
			panic(r)
		}
	}()

	// --- Ambil existing (FOR UPDATE)
	var existing model.ClassModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&existing, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {

		_ = tx.Rollback().Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// --- Tenant guard
	if existing.ClassMasjidID != masjidID {
		_ = tx.Rollback().Error
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah kelas di masjid lain")
	}

	// --- Track perubahan aktif → update stats
	wasActive := existing.ClassIsActive
	newActive := wasActive
	if req.ClassIsActive != nil {
		newActive = *req.ClassIsActive
	}

	// --- Slug unik per masjid (abaikan soft-deleted; pending delete difilter unique partial index)
	if req.ClassSlug != nil && *req.ClassSlug != existing.ClassSlug {
		opts := helper.SlugOptions{
			Table:            "classes",
			SlugColumn:       "class_slug",
			SoftDeleteColumn: "class_deleted_at",
			Filters:          map[string]any{"class_masjid_id": masjidID},
			MaxLen:           160,
			DefaultBase:      "kelas",
		}

		uni, gErr := helper.GenerateUniqueSlug(tx, opts, *req.ClassSlug)
		if gErr != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik: "+gErr.Error())
		}

		// Jika user benar-benar mengirim slug eksplisit (bukan hasil turunan dari name) dan bentrok → tolak
		if formSlug := strings.TrimSpace(c.FormValue("class_slug")); formSlug != "" && formSlug != uni {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		req.ClassSlug = &uni
	}

	// --- Jika ada gambar baru & berbeda → pindahkan yang lama ke spam/
	if req.ClassImageURL != nil && existing.ClassImageURL != nil && *existing.ClassImageURL != *req.ClassImageURL {
		if spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(*existing.ClassImageURL, 0); mvErr == nil {
			// Catat ke trash_url bila belum diisi user
			if req.ClassTrashURL == nil {
				req.ClassTrashURL = &spamURL
			}
		}
		// best-effort: kalau gagal pindah ke spam, lanjutkan update data
	}

	// --- Apply & Save
	req.ApplyToModel(&existing)

	if err := tx.Model(&model.ClassModel{}).
		Where("class_id = ?", existing.ClassID).
		Updates(&existing).Error; err != nil {

		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())

		switch {
		case strings.Contains(low, "uq_classes_slug_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")):
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")

		case strings.Contains(low, "uq_classes_code_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_code")):
			return fiber.NewError(fiber.StatusConflict, "Kode kelas sudah digunakan di masjid ini")

		default:
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}
	}

	// --- Update statistik jika status aktif berubah
	if wasActive != newActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveClasses(tx, masjidID, delta); err != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// --- Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Kelas berhasil diperbarui", dto.NewClassResponse(&existing))
}



// GET /admin/classes/:id
func (ctrl *ClassController) GetClassByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.ClassModel
	if err := ctrl.DB.
		Where("class_id = ? AND class_masjid_id = ? AND class_deleted_at IS NULL", classID, masjidID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "Data diterima", dto.NewClassResponse(&m))
}



// DELETE /admin/classes/:id (soft delete)
func (ctrl *ClassController) SoftDeleteClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(); panic(r)
		}
	}()

	// Lock row
	var m model.ClassModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_id = ? AND class_masjid_id = ? AND class_deleted_at IS NULL", classID, masjidID).
		First(&m).Error; err != nil {

		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	wasActive := m.ClassIsActive

	// Optional: pindahkan gambar ke spam/ (OSS) jika diminta ?delete_image=true
	deletedImage := false
	newTrashURL := ""
	if strings.EqualFold(c.Query("delete_image"), "true") && m.ClassImageURL != nil && *m.ClassImageURL != "" {
		if spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(*m.ClassImageURL, 0); mvErr == nil {
			newTrashURL = spamURL
			deletedImage = true
		}
		// best-effort walau gagal memindahkan
	}

	now := time.Now()
	updates := map[string]any{
		"class_deleted_at": now,
		"class_is_active":  false,
		"class_updated_at": now,
	}
	if deletedImage {
		updates["class_image_url"] = nil
		// simpan jejak spam url jika ada
		if newTrashURL != "" {
			updates["class_trash_url"] = newTrashURL
		}
	}

	if err := tx.Model(&model.ClassModel{}).
		Where("class_id = ?", m.ClassID).
		Updates(updates).Error; err != nil {

		_ = tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	// Decrement stats jika sebelumnya aktif
	if wasActive {
		stats := service.NewLembagaStatsService()
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveClasses(tx, masjidID, -1); err != nil {
			_ = tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Kelas berhasil dihapus", fiber.Map{
		"class_id":      m.ClassID,
		"deleted_image": deletedImage,
		"trash_url":     newTrashURL,
	})
}
