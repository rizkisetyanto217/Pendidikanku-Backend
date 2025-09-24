// internals/features/lembaga/class_subject_books/controller/class_subject_book_controller.go
package controller

import (
	"errors"
	"fmt"
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
	// Normalisasi ringan pada slug (opsional)
	if req.ClassSubjectBooksSlug != nil {
		s := strings.TrimSpace(*req.ClassSubjectBooksSlug)
		if s == "" {
			req.ClassSubjectBooksSlug = nil
		} else {
			ns := helper.Slugify(s, 160)
			req.ClassSubjectBooksSlug = &ns
		}
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var created csbModel.ClassSubjectBookModel
	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		m := req.ToModel()

		// ===== SLUG: normalize + ensure-unique per tenant (alive-only) =====
		// Tentukan baseSlug: pakai request → judul buku → fallback short id
		baseSlug := ""
		if m.ClassSubjectBookSlug != nil && strings.TrimSpace(*m.ClassSubjectBookSlug) != "" {
			baseSlug = helper.Slugify(*m.ClassSubjectBookSlug, 160)
		} else {
			// coba ambil judul buku
			var bookTitle string
			if err := tx.Table("books").
				Select("books_title").
				Where("books_id = ?", m.ClassSubjectBookBookID).
				Take(&bookTitle).Error; err == nil && strings.TrimSpace(bookTitle) != "" {
				baseSlug = helper.Slugify(bookTitle, 160)
			}
			if baseSlug == "" || baseSlug == "item" {
				baseSlug = helper.Slugify(
					fmt.Sprintf("book-%s", strings.Split(m.ClassSubjectBookBookID.String(), "-")[0]),
					160,
				)
			}
		}

		uniqueSlug, uerr := helper.EnsureUniqueSlugCI(
			c.Context(),
			tx,
			"class_subject_books",
			"class_subject_book_slug",
			baseSlug,
			func(q *gorm.DB) *gorm.DB {
				// selaras dengan index uq_csb_slug_per_tenant_alive (versi singular kolom)
				return q.Where(`
					class_subject_book_masjid_id = ?
					AND class_subject_book_deleted_at IS NULL
				`, masjidID)
			},
			160,
		)
		if uerr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		m.ClassSubjectBookSlug = &uniqueSlug
		// ===== END SLUG =====

		// Create
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csb_unique") ||
				strings.Contains(msg, "duplicate") ||
				(strings.Contains(msg, "unique") &&
					strings.Contains(msg, "masjid") && strings.Contains(msg, "class_subject") && strings.Contains(msg, "book")):
				return fiber.NewError(fiber.StatusConflict, "Buku sudah terdaftar pada class_subject tersebut")

			case strings.Contains(msg, "uq_csb_slug_per_tenant_alive") ||
				(strings.Contains(msg, "class_subject_book_slug") && strings.Contains(msg, "unique")):
				return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan pada tenant ini")

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
	// Normalisasi ringan untuk slug (biar rapi; ensure-unique nanti)
	if req.ClassSubjectBooksSlug != nil {
		s := strings.TrimSpace(*req.ClassSubjectBooksSlug)
		if s == "" {
			// explicit clear
			req.ClassSubjectBooksSlug = nil
		} else {
			ns := helper.Slugify(s, 160)
			req.ClassSubjectBooksSlug = &ns
		}
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var updated csbModel.ClassSubjectBookModel
	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var m csbModel.ClassSubjectBookModel
		if err := tx.
			Where("class_subject_book_id = ?", id).
			First(&m).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectBookMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah data milik masjid lain")
		}
		if m.ClassSubjectBookDeletedAt.Valid {
			return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
		}

		// Terapkan perubahan dari DTO (tanpa menyentuh UpdatedAt; biar autoUpdateTime yang isi)
		if err := req.Apply(&m); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// ===== SLUG handling =====
		// Kasus:
		// 1) User kirim slug → pakai & ensure-unique
		// 2) User tidak kirim slug & slug di DB kosong → auto-generate dari judul buku
		if req.ClassSubjectBooksSlug != nil {
			if s := strings.TrimSpace(*req.ClassSubjectBooksSlug); s == "" {
				m.ClassSubjectBookSlug = nil
			} else {
				base := helper.Slugify(s, 160)
				unique, err := helper.EnsureUniqueSlugCI(
					c.Context(),
					tx,
					"class_subject_books",
					"class_subject_book_slug",
					base,
					func(q *gorm.DB) *gorm.DB {
						return q.Where(`
							class_subject_book_masjid_id = ?
							AND class_subject_book_deleted_at IS NULL
							AND class_subject_book_id <> ?
						`, masjidID, m.ClassSubjectBookID)
					},
					160,
				)
				if err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
				}
				m.ClassSubjectBookSlug = &unique
			}
		} else if m.ClassSubjectBookSlug == nil {
			// Auto-generate slug karena sebelumnya kosong
			var bookTitle string
			if err := tx.Table("books").
				Select("books_title").
				Where("books_id = ?", m.ClassSubjectBookBookID).
				Take(&bookTitle).Error; err == nil && strings.TrimSpace(bookTitle) != "" {
				bookTitle = helper.Slugify(bookTitle, 160)
			}
			base := bookTitle
			if base == "" || base == "item" {
				base = helper.Slugify(
					fmt.Sprintf("book-%s", strings.Split(m.ClassSubjectBookBookID.String(), "-")[0]),
					160,
				)
			}
			unique, err := helper.EnsureUniqueSlugCI(
				c.Context(),
				tx,
				"class_subject_books",
				"class_subject_book_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(`
						class_subject_book_masjid_id = ?
						AND class_subject_book_deleted_at IS NULL
						AND class_subject_book_id <> ?
					`, masjidID, m.ClassSubjectBookID)
				},
				160,
			)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			m.ClassSubjectBookSlug = &unique
		}
		// ===== END SLUG =====

		patch := map[string]interface{}{
			"class_subject_book_masjid_id":        m.ClassSubjectBookMasjidID,
			"class_subject_book_class_subject_id": m.ClassSubjectBookClassSubjectID,
			"class_subject_book_book_id":          m.ClassSubjectBookBookID,
			"class_subject_book_is_active":        m.ClassSubjectBookIsActive,
			"class_subject_book_desc":             m.ClassSubjectBookDesc,
			"class_subject_book_slug":             m.ClassSubjectBookSlug,
			// updated_at auto by DB/GORM
		}

		if err := tx.Model(&csbModel.ClassSubjectBookModel{}).
			Where("class_subject_book_id = ?", m.ClassSubjectBookID).
			Updates(patch).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csb_unique") ||
				strings.Contains(msg, "duplicate") ||
				(strings.Contains(msg, "unique") &&
					strings.Contains(msg, "masjid") && strings.Contains(msg, "class_subject") && strings.Contains(msg, "book")):
				return fiber.NewError(fiber.StatusConflict, "Buku sudah terdaftar pada class_subject tersebut")

			case strings.Contains(msg, "uq_csb_slug_per_tenant_alive") ||
				(strings.Contains(msg, "class_subject_book_slug") && strings.Contains(msg, "unique")):
				return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan pada tenant ini")

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
		if err := tx.First(&m, "class_subject_book_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectBookMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus data milik masjid lain")
		}

		if force {
			// HARD DELETE
			if err := tx.Delete(&csbModel.ClassSubjectBookModel{}, "class_subject_book_id = ?", id).Error; err != nil {
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
					return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		} else {
			// SOFT DELETE via GORM (mengisi deleted_at)
			if m.ClassSubjectBookDeletedAt.Valid {
				return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
			}
			if err := tx.Where("class_subject_book_id = ?", id).
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
