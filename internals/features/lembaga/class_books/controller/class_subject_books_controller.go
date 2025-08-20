// internals/features/lembaga/class_subject_books/controller/class_subject_book_controller.go
package controller

import (
	"errors"
	"strings"

	csbDTO "masjidku_backend/internals/features/lembaga/class_books/dto"
	csbModel "masjidku_backend/internals/features/lembaga/class_books/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectBookController struct {
	DB *gorm.DB
}

func intPtr(v int) *int { return &v }

/* =========================================================
   CREATE
   POST /admin/class-subject-books
   Body: CreateClassSubjectBookRequest
   ========================================================= */
func (h *ClassSubjectBookController) Create(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var req csbDTO.CreateClassSubjectBookRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Paksa tenant dari token
	req.ClassSubjectBooksMasjidID = masjidID

	// Normalisasi ringan pada desc
	if req.ClassSubjectBooksDesc != nil {
		d := strings.TrimSpace(*req.ClassSubjectBooksDesc)
		if d == "" {
			req.ClassSubjectBooksDesc = nil
		} else {
			req.ClassSubjectBooksDesc = &d
		}
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var created csbModel.ClassSubjectBookModel
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csb_unique") ||
				strings.Contains(msg, "duplicate") ||
				strings.Contains(msg, "unique"):
				return fiber.NewError(fiber.StatusConflict, "Buku sudah terdaftar pada class_subject tersebut")
			case strings.Contains(msg, "foreign"):
				return fiber.NewError(fiber.StatusBadRequest, "FK gagal: pastikan class_subject & book valid dan satu tenant")
			default:
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat relasi buku")
			}
		}
		created = m
		return nil
	}); err != nil {
		return err
	}

	return helper.JsonCreated(c, "Relasi buku berhasil dibuat", csbDTO.FromModel(created))
}

/* =========================================================
   GET BY ID
   GET /admin/class-subject-books/:id[?with_deleted=true]
   ========================================================= */
func (h *ClassSubjectBookController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	withDeleted := strings.EqualFold(c.Query("with_deleted"), "true")

	var m csbModel.ClassSubjectBookModel
	if err := h.DB.First(&m, "class_subject_books_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if m.ClassSubjectBooksMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Akses ditolak")
	}
	// soft delete guard (pakai gorm.DeletedAt.Valid)
	if !withDeleted && m.ClassSubjectBooksDeletedAt.Valid {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}

	return helper.JsonOK(c, "Detail class_subject_book", csbDTO.FromModel(m))
}

/* =========================================================
   LIST
   GET /admin/class-subject-books
     ?class_subject_id=&book_id=&is_active=&q=&sort=&limit=&offset=&with_deleted=
   sort: created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
   ========================================================= */
func (h *ClassSubjectBookController) List(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var q csbDTO.ListClassSubjectBookQuery
	// default pagination
	q.Limit, q.Offset = intPtr(20), intPtr(0)

	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	if q.Limit == nil || *q.Limit <= 0 || *q.Limit > 200 {
		q.Limit = intPtr(20)
	}
	if q.Offset == nil || *q.Offset < 0 {
		q.Offset = intPtr(0)
	}

	tx := h.DB.Model(&csbModel.ClassSubjectBookModel{}).
		Where("class_subject_books_masjid_id = ?", masjidID)

	// exclude soft-deleted by default
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_subject_books_deleted_at IS NULL")
	}

	// filters
	if q.ClassSubjectID != nil {
		tx = tx.Where("class_subject_books_class_subject_id = ?", *q.ClassSubjectID)
	}
	if q.BookID != nil {
		tx = tx.Where("class_subject_books_book_id = ?", *q.BookID)
	}
	if q.IsActive != nil {
		tx = tx.Where("class_subject_books_is_active = ?", *q.IsActive)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		qq := "%" + strings.TrimSpace(*q.Q) + "%"
		tx = tx.Where("class_subject_books_desc ILIKE ?", qq)
	}

	// total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// sort
	sort := "created_at_desc"
	if q.Sort != nil {
		sort = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sort {
	case "created_at_asc":
		tx = tx.Order("class_subject_books_created_at ASC")
	case "updated_at_asc":
		tx = tx.Order("class_subject_books_updated_at ASC NULLS FIRST")
	case "updated_at_desc":
		tx = tx.Order("class_subject_books_updated_at DESC NULLS LAST")
	default: // "created_at_desc"
		tx = tx.Order("class_subject_books_created_at DESC")
	}

	// data
	var rows []csbModel.ClassSubjectBookModel
	if err := tx.
		Select(`
			class_subject_books_id,
			class_subject_books_masjid_id,
			class_subject_books_class_subject_id,
			class_subject_books_book_id,
			class_subject_books_is_active,
			class_subject_books_desc,
			class_subject_books_created_at,
			class_subject_books_updated_at,
			class_subject_books_deleted_at
		`).
		Limit(*q.Limit).Offset(*q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonList(c,
		csbDTO.FromModels(rows),
		csbDTO.Pagination{
			Limit:  *q.Limit,
			Offset: *q.Offset,
			Total:  int(total),
		},
	)
}

// UPDATE (partial)
// PUT /admin/class-subject-books/:id
func (h *ClassSubjectBookController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req csbDTO.UpdateClassSubjectBookRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant
	req.ClassSubjectBooksMasjidID = &masjidID

	// Normalisasi ringan untuk desc
	if req.ClassSubjectBooksDesc != nil {
		d := strings.TrimSpace(*req.ClassSubjectBooksDesc)
		if d == "" {
			req.ClassSubjectBooksDesc = nil
		} else {
			req.ClassSubjectBooksDesc = &d
		}
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var updated csbModel.ClassSubjectBookModel
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var m csbModel.ClassSubjectBookModel
		if err := tx.First(&m, "class_subject_books_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectBooksMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah data milik masjid lain")
		}
		if m.ClassSubjectBooksDeletedAt.Valid {
			return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
		}

		// Apply perubahan (setter DTO sudah isi UpdatedAt)
		req.Apply(&m)

		patch := map[string]interface{}{
			"class_subject_books_masjid_id":        m.ClassSubjectBooksMasjidID,
			"class_subject_books_class_subject_id": m.ClassSubjectBooksClassSubjectID,
			"class_subject_books_book_id":          m.ClassSubjectBooksBookID,
			"class_subject_books_is_active":        m.ClassSubjectBooksIsActive,
			"class_subject_books_desc":             m.ClassSubjectBooksDesc,
			"class_subject_books_updated_at":       m.ClassSubjectBooksUpdatedAt,
		}

		if err := tx.Model(&csbModel.ClassSubjectBookModel{}).
			Where("class_subject_books_id = ?", m.ClassSubjectBooksID).
			Updates(patch).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csb_unique") ||
				strings.Contains(msg, "duplicate") ||
				strings.Contains(msg, "unique"):
				return fiber.NewError(fiber.StatusConflict, "Buku sudah terdaftar pada class_subject tersebut")
			case strings.Contains(msg, "foreign"):
				return fiber.NewError(fiber.StatusBadRequest, "FK gagal: pastikan class_subject & book valid dan satu tenant")
			default:
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
			}
		}

		updated = m
		return nil
	}); err != nil {
		return err
	}

	return helper.JsonUpdated(c, "Relasi buku berhasil diperbarui", csbDTO.FromModel(updated))
}


// DELETE
// DELETE /admin/class-subject-books/:id?force=true
// - force=true (admin saja): hard delete
// - default: soft delete (gorm.DeletedAt)
func (h *ClassSubjectBookController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// hanya admin yang boleh hard-delete
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var deleted csbModel.ClassSubjectBookModel
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var m csbModel.ClassSubjectBookModel
		if err := tx.First(&m, "class_subject_books_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectBooksMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus data milik masjid lain")
		}

		if force {
			// HARD DELETE
			if err := tx.Delete(&csbModel.ClassSubjectBookModel{}, "class_subject_books_id = ?", id).Error; err != nil {
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
					return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		} else {
			// SOFT DELETE via GORM (mengisi deleted_at)
			if m.ClassSubjectBooksDeletedAt.Valid {
				return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
			}
			if err := tx.Where("class_subject_books_id = ?", id).
				Delete(&csbModel.ClassSubjectBookModel{}).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		}

		deleted = m
		return nil
	}); err != nil {
		return err
	}

	return helper.JsonDeleted(c, "Relasi buku berhasil dihapus", csbDTO.FromModel(deleted))
}
