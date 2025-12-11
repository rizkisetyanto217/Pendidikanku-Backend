// file: internals/features/lembaga/class_subject_books/controller/class_subject_book_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"

	csbDTO "madinahsalam_backend/internals/features/school/academics/books/dto"
	csbModel "madinahsalam_backend/internals/features/school/academics/books/model"
	bookSnap "madinahsalam_backend/internals/features/school/academics/books/service"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

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
	// 🔐 Tenant scope: ambil school_id dari TOKEN
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Kalau bukan owner → wajib DKM/Admin di school ini
	if !helperAuth.IsOwner(c) {
		if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
			return err
		}
	}

	// 🔎 Parse + validate body
	var req csbDTO.CreateClassSubjectBookRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.ClassSubjectBookSchoolID = schoolID
	req.Normalize()
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var created csbModel.ClassSubjectBookModel
	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// ✅ Validasi kepemilikan tenant: CLASS_SUBJECT & BOOK
		if err := ensureClassSubjectTenantExists(tx, req.ClassSubjectBookClassSubjectID, schoolID); err != nil {
			return err
		}
		if err := ensureBookTenantExists(tx, req.ClassSubjectBookBookID, schoolID); err != nil {
			return err
		}

		// 📸 Ambil cache BOOK
		snapB, err := bookSnap.FetchBookCache(tx, req.ClassSubjectBookBookID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Buku tidak ditemukan")
		}

		// 📸 Ambil cache SUBJECT via class_subject_id
		snapS, err := fetchSubjectCacheByClassSubjectID(tx, req.ClassSubjectBookClassSubjectID, schoolID)
		if err != nil {
			return err // sudah mengembalikan fiber.Error yang pas
		}

		// 🏗️ Build model dari DTO (sudah mengisi is_primary, is_required, order, is_active, desc)
		m := req.ToModel()

		// sanity: paksa tenant
		m.ClassSubjectBookSchoolID = schoolID

		// 🧩 SLUG: normalize + ensure-unique per tenant (alive-only)
		baseSlug := ""
		if m.ClassSubjectBookSlug != nil && strings.TrimSpace(*m.ClassSubjectBookSlug) != "" {
			baseSlug = helper.Slugify(*m.ClassSubjectBookSlug, 160)
		} else {
			// prioritas pakai title buku cache
			if t := strings.TrimSpace(snapB.Title); t != "" {
				baseSlug = helper.Slugify(t, 160)
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
				return q.Where(`
					class_subject_book_school_id = ?
					AND class_subject_book_deleted_at IS NULL
				`, schoolID)
			},
			160,
		)
		if uerr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		m.ClassSubjectBookSlug = &uniqueSlug

		// 🧊 Isi cache BOOK (nil-safe)
		if snapB.Title != "" {
			m.ClassSubjectBookBookTitleCache = &snapB.Title
		}
		m.ClassSubjectBookBookAuthorCache = snapB.Author
		m.ClassSubjectBookBookSlugCache = snapB.Slug
		m.ClassSubjectBookBookPublisherCache = snapB.Publisher
		m.ClassSubjectBookBookPublicationYearCache = snapB.PublicationYear
		m.ClassSubjectBookBookImageURLCache = snapB.ImageURL

		// 🧊 Isi cache SUBJECT (pakai field baru)
		m.ClassSubjectBookSubjectID = &snapS.SubjectID
		if snapS.Code != nil {
			m.ClassSubjectBookSubjectCodeCache = snapS.Code
		}
		if snapS.Name != nil {
			m.ClassSubjectBookSubjectNameCache = snapS.Name
		}
		if snapS.Slug != nil {
			m.ClassSubjectBookSubjectSlugCache = snapS.Slug
		}

		// 💾 Create
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csb_unique") ||
				strings.Contains(msg, "duplicate") ||
				(strings.Contains(msg, "unique") &&
					strings.Contains(msg, "school") &&
					(strings.Contains(msg, "subject") || strings.Contains(msg, "class_subject")) &&
					strings.Contains(msg, "book")):
				return fiber.NewError(fiber.StatusConflict, "Buku sudah terdaftar pada subject tersebut")
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

	// ... di dalam func (h *ClassSubjectBookController) Create(c *fiber.Ctx) error {

	return helper.JsonCreated(
		c,
		"Relasi buku berhasil dibuat",
		csbDTO.FromModelWithSchoolTime(c, created),
	)
}

/* ================= Helpers: EXISTS-based tenant checks ================= */

func ensureClassSubjectTenantExists(db *gorm.DB, classSubjectID, schoolID uuid.UUID) error {
	var ok bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM class_subjects
			WHERE class_subject_id = ?
			  AND class_subject_school_id = ?
			  AND class_subject_deleted_at IS NULL
		)`, classSubjectID, schoolID).Scan(&ok).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class_subject")
	}
	if !ok {
		return fiber.NewError(fiber.StatusForbidden, "Class subject tidak ditemukan / beda tenant")
	}
	return nil
}

func ensureBookTenantExists(db *gorm.DB, bookID, schoolID uuid.UUID) error {
	var ok bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM books
			WHERE book_id = ?
			  AND book_school_id = ?
			  AND book_deleted_at IS NULL
		)`, bookID, schoolID).Scan(&ok).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi buku")
	}
	if !ok {
		return fiber.NewError(fiber.StatusForbidden, "Buku tidak ditemukan / beda tenant")
	}
	return nil
}

/* ================= Cache fetcher (SUBJECT via class_subject_id) ================= */

type subjectCache struct {
	SubjectID uuid.UUID
	Code      *string
	Name      *string
	Slug      *string
}

func fetchSubjectCacheByClassSubjectID(tx *gorm.DB, classSubjectID, schoolID uuid.UUID) (*subjectCache, error) {
	var ss subjectCache
	if err := tx.Raw(`
		SELECT 
			s.subject_id       AS subject_id,
			s.subject_code     AS code,
			s.subject_name     AS name,
			s.subject_slug     AS slug
		FROM class_subjects cs
		JOIN subjects s 
		  ON s.subject_id = cs.class_subject_subject_id
		WHERE cs.class_subject_id = ?
		  AND cs.class_subject_school_id = ?
		  AND cs.class_subject_deleted_at IS NULL
		  AND s.subject_deleted_at IS NULL
	`, classSubjectID, schoolID).Scan(&ss).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil cache subject")
	}
	if ss.SubjectID == uuid.Nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Subject untuk class_subject tidak ditemukan")
	}
	return &ss, nil
}

/*
=========================================================
UPDATE (partial) (DKM/Admin only)
PUT /admin/class-subject-books/:id
=========================================================
*/
func (h *ClassSubjectBookController) Update(c *fiber.Ctx) error {
	// 🔐 Tenant scope: ambil school_id dari TOKEN
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Guard role: owner bypass, lainnya wajib DKM/Admin
	if !helperAuth.IsOwner(c) {
		if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
			return err
		}
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req csbDTO.UpdateClassSubjectBookRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// paksa tenant
	req.ClassSubjectBookSchoolID = &schoolID

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
		if m.ClassSubjectBookSchoolID != schoolID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah data milik school lain")
		}
		if m.ClassSubjectBookDeletedAt.Valid {
			return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
		}

		// Terapkan perubahan field dasar (termasuk is_primary, is_required, order)
		if err := req.Apply(&m); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		// Jika class_subject_id diubah → validasi tenant + refresh SUBJECT cache
		if req.ClassSubjectBookClassSubjectID != nil {
			if err := ensureClassSubjectTenantExists(tx, *req.ClassSubjectBookClassSubjectID, schoolID); err != nil {
				return err
			}
			snapS, err := fetchSubjectCacheByClassSubjectID(tx, *req.ClassSubjectBookClassSubjectID, schoolID)
			if err != nil {
				return err
			}
			m.ClassSubjectBookSubjectID = &snapS.SubjectID
			m.ClassSubjectBookSubjectCodeCache = snapS.Code
			m.ClassSubjectBookSubjectNameCache = snapS.Name
			m.ClassSubjectBookSubjectSlugCache = snapS.Slug
		}

		// Jika book_id diubah → validasi tenant + refresh BOOK cache
		if req.ClassSubjectBookBookID != nil {
			if err := ensureBookTenantExists(tx, *req.ClassSubjectBookBookID, schoolID); err != nil {
				return err
			}
			snapB, err := bookSnap.FetchBookCache(tx, *req.ClassSubjectBookBookID)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Buku tidak ditemukan")
			}
			if snapB.Title != "" {
				m.ClassSubjectBookBookTitleCache = &snapB.Title
			} else {
				m.ClassSubjectBookBookTitleCache = nil
			}
			m.ClassSubjectBookBookAuthorCache = snapB.Author
			m.ClassSubjectBookBookSlugCache = snapB.Slug
			m.ClassSubjectBookBookPublisherCache = snapB.Publisher
			m.ClassSubjectBookBookPublicationYearCache = snapB.PublicationYear
			m.ClassSubjectBookBookImageURLCache = snapB.ImageURL
		}

		// SLUG handling (ensure unique jika diubah / jika kosong sebelumnya)
		if req.ClassSubjectBookSlug != nil {
			if s := strings.TrimSpace(*req.ClassSubjectBookSlug); s == "" {
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
							class_subject_book_school_id = ?
							AND class_subject_book_deleted_at IS NULL
							AND class_subject_book_id <> ?
						`, schoolID, m.ClassSubjectBookID)
					},
					160,
				)
				if err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
				}
				m.ClassSubjectBookSlug = &unique
			}
		} else if m.ClassSubjectBookSlug == nil {
			// auto-generate saat masih kosong
			var base string
			// pakai title cache terbaru bila ada
			if m.ClassSubjectBookBookTitleCache != nil && strings.TrimSpace(*m.ClassSubjectBookBookTitleCache) != "" {
				base = helper.Slugify(*m.ClassSubjectBookBookTitleCache, 160)
			}
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
						class_subject_book_school_id = ?
						AND class_subject_book_deleted_at IS NULL
						AND class_subject_book_id <> ?
					`, schoolID, m.ClassSubjectBookID)
				},
				160,
			)
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			m.ClassSubjectBookSlug = &unique
		}

		if err := tx.Model(&csbModel.ClassSubjectBookModel{}).
			Where("class_subject_book_id = ?", m.ClassSubjectBookID).
			Updates(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			switch {
			case strings.Contains(msg, "uq_csb_unique") ||
				strings.Contains(msg, "duplicate") ||
				(strings.Contains(msg, "unique") &&
					strings.Contains(msg, "school") &&
					(strings.Contains(msg, "subject") || strings.Contains(msg, "class_subject")) &&
					strings.Contains(msg, "book")):
				return fiber.NewError(fiber.StatusConflict, "Buku sudah terdaftar pada subject tersebut")
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

	// ... di dalam func (h *ClassSubjectBookController) Update(c *fiber.Ctx) error {

	return helper.JsonUpdated(
		c,
		"Relasi buku berhasil diperbarui",
		csbDTO.FromModelWithSchoolTime(c, updated),
	)
}

/*
=========================================================
DELETE (DKM/Admin; hard delete: admin only)
DELETE /admin/class-subject-books/:id?force=true
=========================================================
*/
func (h *ClassSubjectBookController) Delete(c *fiber.Ctx) error {
	// 🔐 Tenant scope: ambil school_id dari TOKEN
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Guard role: owner bypass, lainnya wajib DKM/Admin
	if !helperAuth.IsOwner(c) {
		if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
			return err
		}
	}

	// dipakai buat izin hard delete: harus admin/tenant yg sama
	adminSchoolID, _ := helperAuth.GetSchoolIDFromToken(c)
	isAdmin := adminSchoolID != uuid.Nil && adminSchoolID == schoolID
	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var deleted csbModel.ClassSubjectBookModel
	if err := h.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var m csbModel.ClassSubjectBookModel
		if err := tx.First(&m, "class_subject_book_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectBookSchoolID != schoolID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus data milik school lain")
		}

		// 🔒 GUARD: tidak boleh dihapus kalau masih dipakai CSST
		var csstCount int64
		if err := tx.
			Model(&csstModel.ClassSectionSubjectTeacherModel{}).
			Where(
				"class_section_subject_teacher_school_id = ? AND class_section_subject_teacher_class_subject_book_id = ? AND class_section_subject_teacher_deleted_at IS NULL",
				schoolID, m.ClassSubjectBookID,
			).
			Count(&csstCount).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengecek relasi ke pengampu mapel (CSST)")
		}

		if csstCount > 0 {
			return fiber.NewError(
				fiber.StatusBadRequest,
				"Tidak dapat menghapus relasi buku karena masih digunakan di pengampu mapel/rombel (CSST). Nonaktifkan atau hapus CSST terkait terlebih dahulu.",
			)
		}

		// === path hard delete (force) ===
		if force {
			if err := tx.Delete(&csbModel.ClassSubjectBookModel{}, "class_subject_book_id = ?", id).Error; err != nil {
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
					return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		} else {
			// === path soft delete ===
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

	// ... di dalam func (h *ClassSubjectBookController) Delete(c *fiber.Ctx) error {

	return helper.JsonDeleted(
		c,
		"Relasi buku berhasil dihapus",
		csbDTO.FromModelWithSchoolTime(c, deleted),
	)
}
