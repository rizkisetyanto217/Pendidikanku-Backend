// internals/features/lembaga/class_subject_books/controller/class_subject_book_controller.go
package controller

import (
	"errors"
	"strings"

	csbDTO "masjidku_backend/internals/features/school/subject_books/books/dto"
	csbModel "masjidku_backend/internals/features/school/subject_books/books/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassSubjectBookController struct {
	DB *gorm.DB
}

/*
=========================================================

	CREATE (DKM/Admin only)
	POST /admin/class-subject-books
	Body: CreateClassSubjectBookRequest
	=========================================================
*/
func (h *ClassSubjectBookController) Create(c *fiber.Ctx) error {
	// === Masjid context (eksplisit & DKM/Admin only) ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req csbDTO.CreateClassSubjectBookRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Paksa tenant dari context
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

/*
	=========================================================
	  UPDATE (partial) (DKM/Admin only)
	  PUT /admin/class-subject-books/:id

=========================================================
*/
func (h *ClassSubjectBookController) Update(c *fiber.Ctx) error {
	// === Masjid context (eksplisit & DKM/Admin only) ===
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
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req csbDTO.UpdateClassSubjectBookRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant dari context
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

		// Terapkan perubahan (setter DTO sudah isi UpdatedAt)
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

/*
	=========================================================
	  DELETE (DKM/Admin; hard delete: admin only)
	  DELETE /admin/class-subject-books/:id?force=true
	  - force=true (admin saja): hard delete
	  - default: soft delete (gorm.DeletedAt)

=========================================================
*/
func (h *ClassSubjectBookController) Delete(c *fiber.Ctx) error {
	// === Masjid context (eksplisit & DKM/Admin only) ===
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// hanya admin yang boleh hard-delete (logika eksisting)
	adminMasjidID, _ := helperAuth.GetMasjidIDFromToken(c)
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
