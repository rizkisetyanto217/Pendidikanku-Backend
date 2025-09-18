// internals/features/lembaga/class_books/controller/books_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/subject_books/books/dto"
	model "masjidku_backend/internals/features/school/subject_books/books/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type BooksController struct {
	DB *gorm.DB
}

var validate = validator.New()

// =========================================================
// CREATE  - POST /admin/class-books
// Body: JSON (atau form sederhana, tanpa upload file)
// =========================================================
func (h *BooksController) Create(c *fiber.Ctx) error {
	// Pastikan DB tersedia untuk helper slug→id
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}

	// Ambil masjid context dari path/header/query/host/token
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	// Hanya DKM/Admin masjid ini yang boleh create
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req dto.BooksCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.BooksMasjidID = masjidID

	// Normalisasi + auto slug (kalau kosong)
	req.Normalize()
	if req.BooksSlug == nil || strings.TrimSpace(*req.BooksSlug) == "" {
		gen := helper.GenerateSlug(req.BooksTitle)
		req.BooksSlug = &gen
	} else {
		s := helper.GenerateSlug(*req.BooksSlug)
		req.BooksSlug = &s
	}

	if err := validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if req.BooksSlug == nil || *req.BooksSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak valid (judul terlalu kosong untuk dibentuk slug)")
	}

	// Cek unik slug per masjid (CI, soft-delete aware)
	var cnt int64
	if err := h.DB.Model(&model.BooksModel{}).
		Where(`
			books_masjid_id = ?
			AND lower(books_slug) = lower(?)
			AND books_deleted_at IS NULL
		`, masjidID, *req.BooksSlug).
		Count(&cnt).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek duplikasi slug")
	}
	if cnt > 0 {
		return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
	}

	// Simpan
	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_books_slug_per_masjid") ||
			strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat data")
	}
	return helper.JsonCreated(c, "Buku berhasil dibuat", dto.ToBooksResponse(m))
}

// =========================================================
// UPDATE - PUT /admin/class-books/:id
// Body: JSON / form sederhana (tanpa upload file)
// =========================================================
func (h *BooksController) Update(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.BooksUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalisasi & validasi
	req.Normalize()
	// Auto-regenerate slug: kalau slug dikirim, pakai itu; kalau tidak tapi title berubah → regen dari title
	if req.BooksSlug != nil {
		s := helper.GenerateSlug(*req.BooksSlug)
		req.BooksSlug = &s
	} else if req.BooksTitle != nil {
		s := helper.GenerateSlug(*req.BooksTitle)
		req.BooksSlug = &s
	}
	if err := validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil buku & guard tenant
	var m model.BooksModel
	if err := h.DB.First(&m, "books_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.BooksMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	// Cek unik slug per masjid jika berubah
	if req.BooksSlug != nil {
		needCheck := (m.BooksSlug == nil) || !strings.EqualFold(*m.BooksSlug, *req.BooksSlug)
		if needCheck {
			var cnt int64
			if err := h.DB.Model(&model.BooksModel{}).
				Where(`
					books_masjid_id = ?
					AND lower(books_slug) = lower(?)
					AND books_id <> ?
					AND books_deleted_at IS NULL
				`, masjidID, *req.BooksSlug, m.BooksID).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek duplikasi slug")
			}
			if cnt > 0 {
				return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
			}
		}
	}

	// Apply ke model & simpan
	req.ApplyToModel(&m)

	if err := h.DB.Save(&m).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_books_slug_per_masjid") ||
			strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Buku berhasil diperbarui", dto.ToBooksResponse(&m))
}

// =========================================================
// DELETE (soft) - DELETE /admin/class-books/:id
// =========================================================
func (h *BooksController) Delete(c *fiber.Ctx) error {
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.BooksModel
	if err := h.DB.First(&m, "books_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.BooksMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	// Soft delete (gorm.DeletedAt)
	if err := h.DB.Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Buku berhasil dihapus", fiber.Map{
		"books_id": id,
	})
}
