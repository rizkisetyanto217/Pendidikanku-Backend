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
	helperAuth "masjidku_backend/internals/helpers/auth"
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

var validate = validator.New()

/* =========================== CREATE =========================== */
// POST /admin/classes
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// 🔐 Paksa tenant
	req.ClassMasjidID = masjidID

	// 🧹 Normalisasi
	req.Normalize()

	// ✅ Validasi payload
	if err := req.Validate(); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
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
	baseSlug := strings.TrimSpace(m.ClassSlug)
	if baseSlug == "" {
		baseSlug = "kelas"
	}
	uniqueSlug, err := helper.GenerateUniqueSlug(tx, slugOpts, baseSlug)
	if err != nil {
		_ = tx.Rollback().Error
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik: "+err.Error())
	}
	m.ClassSlug = uniqueSlug

	// 💾 Simpan
	if err := tx.Create(m).Error; err != nil {
		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "uq_classes_slug_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")) {
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}

	// 📈 Update lembaga_stats bila status = active
	if m.ClassStatus == model.ClassStatusActive {
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

	return helper.JsonCreated(c, "Kelas berhasil dibuat", dto.FromModel(m))
}

/* ============================ PATCH ============================ */
// PATCH /admin/classes/:id
func (ctrl *ClassController) PatchClass(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// --- Parse path param
	classID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// --- Parse payload (PATCH tri-state)
	var req dto.PatchClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// --- Upload gambar baru (multipart) → override patch field
	if fh, ferr := c.FormFile("class_image_url"); ferr == nil && fh != nil {
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Init OSS gagal: "+err.Error())
		}
		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()

		dir := fmt.Sprintf("masjids/%s/classes", masjidID.String())
		publicURL, upErr := svc.UploadAsWebP(ctx, fh, dir)
		if upErr != nil {
			low := strings.ToLower(upErr.Error())
			if strings.Contains(low, "format tidak didukung") {
				return fiber.NewError(fiber.StatusUnsupportedMediaType, "Format tidak didukung (jpg/png/webp)")
			}
			return fiber.NewError(fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		if req.ClassImageURL == nil {
			req.ClassImageURL = &dto.PatchField[*string]{Set: true, Value: &publicURL}
		} else {
			req.ClassImageURL.Set = true
			req.ClassImageURL.Value = &publicURL
		}
	}

	// --- Normalisasi & Validasi
	req.Normalize()
	if err := req.Validate(); err != nil {
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

	// --- Track perubahan ACTIVE → update stats jika berubah
	wasActive := (existing.ClassStatus == model.ClassStatusActive)
	newActive := wasActive
	if req.ClassStatus != nil && req.ClassStatus.Set {
		newActive = (req.ClassStatus.Value == model.ClassStatusActive)
	}

	// --- Slug unik per masjid (jika di-patch & berbeda)
	if req.ClassSlug != nil && req.ClassSlug.Set && req.ClassSlug.Value != existing.ClassSlug {
		opts := helper.SlugOptions{
			Table:            "classes",
			SlugColumn:       "class_slug",
			SoftDeleteColumn: "class_deleted_at",
			Filters:          map[string]any{"class_masjid_id": masjidID},
			MaxLen:           160,
			DefaultBase:      "kelas",
		}
		base := strings.TrimSpace(req.ClassSlug.Value)
		if base == "" {
			base = "kelas"
		}
		uni, gErr := helper.GenerateUniqueSlug(tx, opts, base)
		if gErr != nil {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik: "+gErr.Error())
		}
		// Jika user minta slug spesifik tapi bentrok → 409
		if req.ClassSlug.Value != "" && uni != req.ClassSlug.Value {
			_ = tx.Rollback().Error
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		req.ClassSlug.Value = uni
	}

	// --- Jika ada gambar baru & berbeda → pindahkan yang lama ke spam/
	if req.ClassImageURL != nil && req.ClassImageURL.Set && req.ClassImageURL.Value != nil &&
		existing.ClassImageURL != nil && *existing.ClassImageURL != *req.ClassImageURL.Value {

		if spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(*existing.ClassImageURL, 0); mvErr == nil {
			if req.ClassTrashURL == nil {
				req.ClassTrashURL = &dto.PatchField[*string]{Set: true, Value: &spamURL}
			} else if !req.ClassTrashURL.Set || req.ClassTrashURL.Value == nil || *req.ClassTrashURL.Value == "" {
				req.ClassTrashURL.Set = true
				req.ClassTrashURL.Value = &spamURL
			}
		}
		// best-effort
	}

	// --- Apply & Save
	req.Apply(&existing)

	if err := tx.Model(&model.ClassModel{}).
		Where("class_id = ?", existing.ClassID).
		Updates(&existing).Error; err != nil {

		_ = tx.Rollback().Error
		low := strings.ToLower(err.Error())
		switch {
		case strings.Contains(low, "uq_classes_slug_per_masjid_active") ||
			(strings.Contains(low, "duplicate") && strings.Contains(low, "class_slug")):
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}
	}

	// --- Update statistik jika transisi active berubah
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

	return helper.JsonUpdated(c, "Kelas berhasil diperbarui", dto.FromModel(&existing))
}

/* ========================== GET BY ID ========================== */
// GET /admin/classes/:id
func (ctrl *ClassController) GetClassByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
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

	return helper.JsonOK(c, "Data diterima", dto.FromModel(&m))
}

/* =========================== SOFT DELETE =========================== */
// DELETE /admin/classes/:id
func (ctrl *ClassController) SoftDeleteClass(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
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

	wasActive := (m.ClassStatus == model.ClassStatusActive)

	// Optional: pindahkan gambar ke spam/ (OSS) jika diminta ?delete_image=true
	deletedImage := false
	newTrashURL := ""
	if strings.EqualFold(c.Query("delete_image"), "true") && m.ClassImageURL != nil && *m.ClassImageURL != "" {
		if spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(*m.ClassImageURL, 0); mvErr == nil {
			newTrashURL = spamURL
			deletedImage = true
		}
		// best-effort
	}

	now := time.Now()
	updates := map[string]any{
		"class_deleted_at": now,
		"class_updated_at": now,
		// opsional: tandai non-aktif saat dihapus (tidak wajib karena row sudah soft-delete)
		"class_status": "inactive",
	}
	if deletedImage {
		updates["class_image_url"] = nil
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

	// Decrement stats jika sebelumnya ACTIVE
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
