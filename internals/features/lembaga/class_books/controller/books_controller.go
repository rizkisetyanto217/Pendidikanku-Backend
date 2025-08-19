package controller

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/lembaga/class_books/dto"
	"masjidku_backend/internals/features/lembaga/class_books/model"
	helper "masjidku_backend/internals/helpers"
)

type BooksController struct {
	DB *gorm.DB
}

var validate = validator.New()

// =====================================================
// CREATE  - POST /admin/class-books
// =====================================================
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
	req.Normalize()

	if err := validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()

	if err := h.DB.Create(m).Error; err != nil {
		// cek duplikat unik via error string
		if strings.Contains(err.Error(), "duplicate key value") {
			return helper.JsonError(c, fiber.StatusConflict, "Judul + Edisi sudah terdaftar untuk masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat data")
	}

	return helper.JsonCreated(c, "Buku berhasil dibuat", dto.ToBooksResponse(m))
}

// =====================================================
// LIST  - GET /admin/class-books
// =====================================================
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
		switch strings.ToLower(*q.OrderBy) {
		case "books_title":
			orderBy = "books_title"
		case "books_year":
			orderBy = "books_year"
		}
	}
	sortDir := "DESC"
	if q.Sort != nil && strings.EqualFold(*q.Sort, "asc") {
		sortDir = "ASC"
	}

	tx := h.DB.Model(&model.BooksModel{}).Where("books_masjid_id = ?", masjidID)

	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		needle := "%" + strings.TrimSpace(*q.Q) + "%"
		tx = tx.Where(
			h.DB.Where("books_title ILIKE ?", needle).
				Or("books_author ILIKE ?", needle).
				Or("books_publisher ILIKE ?", needle),
		)
	}
	if q.Publisher != nil && strings.TrimSpace(*q.Publisher) != "" {
		tx = tx.Where("books_publisher ILIKE ?", strings.TrimSpace(*q.Publisher))
	}
	if q.YearMin != nil {
		tx = tx.Where("books_year >= ?", *q.YearMin)
	}
	if q.YearMax != nil {
		tx = tx.Where("books_year <= ?", *q.YearMax)
	}
	if q.HasImage != nil {
		if *q.HasImage {
			tx = tx.Where("books_image_url IS NOT NULL AND books_image_url <> ''")
		} else {
			tx = tx.Where("(books_image_url IS NULL OR books_image_url = '')")
		}
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var rows []model.BooksModel
	if err := tx.Order(fmt.Sprintf("%s %s", orderBy, sortDir)).
		Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
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

// =====================================================
// GET BY ID
// =====================================================
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

// =====================================================
// UPDATE
// =====================================================
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
	req.Normalize()

	if err := validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
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

	req.ApplyToModel(&m)

	if err := h.DB.Save(&m).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			return helper.JsonError(c, fiber.StatusConflict, "Judul + Edisi sudah terdaftar untuk masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Buku berhasil diperbarui", dto.ToBooksResponse(&m))
}

// =====================================================
// DELETE (soft delete)
// =====================================================
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

	if err := h.DB.Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Buku berhasil dihapus", fiber.Map{
		"books_id": id,
	})
}
