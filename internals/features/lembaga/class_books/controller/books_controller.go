// internals/features/lembaga/class_books/controller/books_controller.go
package controller

import (
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/lembaga/class_books/dto"
	model "masjidku_backend/internals/features/lembaga/class_books/model"
	helper "masjidku_backend/internals/helpers"
)

type BooksController struct {
	DB *gorm.DB
}

var validate = validator.New()

/* =========================================================
   CREATE  - POST /admin/class-books
   Body: form-data / JSON (support upload)
   ========================================================= */
func (h *BooksController) Create(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var req dto.BooksCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.BooksMasjidID = masjidID

	// Ambil URL gambar dari form jika ada
	if v := strings.TrimSpace(c.FormValue("books_image_url")); v != "" && req.BooksImageURL == nil {
		req.BooksImageURL = &v
	}

	// Upload file (prioritas di atas URL teks)
	var fileHeader *multipart.FileHeader
	for _, key := range []string{"books_image", "books_image_url", "image", "file"} {
		if fh, ferr := c.FormFile(key); ferr == nil && fh != nil {
			fileHeader = fh
			break
		}
	}
	if fileHeader != nil {
		publicURL, upErr := helper.UploadImageToSupabase("books", fileHeader)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
		req.BooksImageURL = &publicURL
	}

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
	// Wajib ada slug hasil generate
	if req.BooksSlug == nil || *req.BooksSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Slug tidak valid (judul terlalu kosong untuk dibentuk slug)")
	}

	// Cek unik slug per masjid (case-insensitive, soft-delete aware)
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
	now := time.Now()
	m.BooksCreatedAt = now
	if err := h.DB.Create(m).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_books_slug_per_masjid") || strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat data")
	}
	return helper.JsonCreated(c, "Buku berhasil dibuat", dto.ToBooksResponse(m))
}

/* =========================================================
   LIST  - GET /admin/class-books
   Query: q, author, has_image, has_url, order_by, sort, limit, offset
   ========================================================= */
func (h *BooksController) List(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var q dto.BooksListQuery
	if err := c.QueryParser(&q); err != nil {
	 return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	limit := 20
	offset := 0
	if q.Limit != nil && *q.Limit > 0 && *q.Limit <= 200 {
		limit = *q.Limit
	}
	if q.Offset != nil && *q.Offset >= 0 {
		offset = *q.Offset
	}

	orderBy := "books_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(strings.TrimSpace(*q.OrderBy)) {
		case "books_title":
			orderBy = "books_title"
		case "books_author":
			orderBy = "books_author"
		case "created_at":
			orderBy = "books_created_at"
		}
	}
	sortDir := "DESC"
	if q.Sort != nil && strings.EqualFold(strings.TrimSpace(*q.Sort), "asc") {
		sortDir = "ASC"
	}

	tx := h.DB.Model(&model.BooksModel{}).
		Where("books_masjid_id = ?", masjidID)

	// Pencarian sederhana pada title/author/desc
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		needle := "%" + strings.TrimSpace(*q.Q) + "%"
		tx = tx.Where(h.DB.
			Where("books_title ILIKE ?", needle).
			Or("books_author ILIKE ?", needle).
			Or("books_desc ILIKE ?", needle))
	}
	// Filter author
	if q.Author != nil && strings.TrimSpace(*q.Author) != "" {
		tx = tx.Where("books_author ILIKE ?", strings.TrimSpace(*q.Author))
	}
	// Filter has image
	if q.HasImage != nil {
		if *q.HasImage {
			tx = tx.Where("books_image_url IS NOT NULL AND books_image_url <> ''")
		} else {
			tx = tx.Where("(books_image_url IS NULL OR books_image_url = '')")
		}
	}
	// Filter has url
	if q.HasURL != nil {
		if *q.HasURL {
			tx = tx.Where("books_url IS NOT NULL AND books_url <> ''")
		} else {
			tx = tx.Where("(books_url IS NULL OR books_url = '')")
		}
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var rows []model.BooksModel
	if err := tx.Order(fmt.Sprintf("%s %s", orderBy, sortDir)).
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]dto.BooksResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, dto.ToBooksResponse(&rows[i]))
	}

	return helper.JsonList(c, resp, fiber.Map{
		"limit":  limit,
		"offset": offset,
		"total":  total,
	})
}

/* =========================================================
   GET BY ID - GET /admin/class-books/:id
   ========================================================= */
func (h *BooksController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
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

	return helper.JsonOK(c, "OK", dto.ToBooksResponse(&m))
}

/* =========================================================
   UPDATE - PUT /admin/class-books/:id
   Body: form-data / JSON (support upload dan clear image)
   ========================================================= */
func (h *BooksController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.BooksUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Ambil URL teks dari form jika ada
	if v := strings.TrimSpace(c.FormValue("books_image_url")); v != "" && req.BooksImageURL == nil {
		req.BooksImageURL = &v
	}

	// Upload file (prioritas)
	var fileHeader *multipart.FileHeader
	for _, key := range []string{"books_image", "books_image_url", "image", "file"} {
		if fh, ferr := c.FormFile(key); ferr == nil && fh != nil {
			fileHeader = fh
			break
		}
	}
	var newUploadedURL *string
	if fileHeader != nil {
		publicURL, upErr := helper.UploadImageToSupabase("books", fileHeader)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
		newUploadedURL = &publicURL
		req.BooksImageURL = newUploadedURL
	}

	// Clear image jika dikirim explicit kosong
	wantClearImage := false
	if req.BooksImageURL != nil && strings.TrimSpace(*req.BooksImageURL) == "" {
		req.BooksImageURL = nil
		wantClearImage = true
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

	// Simpan URL lama untuk cleanup
	oldURL := m.BooksImageURL

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

	// Apply ke model
	req.ApplyToModel(&m)

	// Handle clear image
	if wantClearImage {
		if oldURL != nil && *oldURL != "" {
			if bucket, path, exErr := helper.ExtractSupabasePath(*oldURL); exErr == nil {
				_ = helper.DeleteFromSupabase(bucket, path)
			}
		}
		m.BooksImageURL = nil
	}

	// Jika upload baru & beda dari lama → hapus lama
	if newUploadedURL != nil && oldURL != nil && *oldURL != *newUploadedURL {
		if bucket, path, exErr := helper.ExtractSupabasePath(*oldURL); exErr == nil {
			_ = helper.DeleteFromSupabase(bucket, path)
		}
	}

	// Save
	if err := h.DB.Save(&m).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "uq_books_slug_per_masjid") || strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Buku berhasil diperbarui", dto.ToBooksResponse(&m))
}

/* =========================================================
   DELETE (soft) - DELETE /admin/class-books/:id?delete_image=true
   ========================================================= */
func (h *BooksController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
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

	// Opsional: hapus file di storage
	deletedImage := false
	if strings.EqualFold(c.Query("delete_image"), "true") && m.BooksImageURL != nil && *m.BooksImageURL != "" {
		if bucket, path, exErr := helper.ExtractSupabasePath(*m.BooksImageURL); exErr == nil {
			_ = helper.DeleteFromSupabase(bucket, path)
			deletedImage = true
		}
		m.BooksImageURL = nil
		_ = h.DB.Model(&model.BooksModel{}).
			Where("books_id = ?", id).
			Update("books_image_url", nil).Error // best-effort
	}

	// Soft delete (gorm.DeletedAt)
	if err := h.DB.Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Buku berhasil dihapus", fiber.Map{
		"books_id":      id,
		"deleted_image": deletedImage,
	})
}
