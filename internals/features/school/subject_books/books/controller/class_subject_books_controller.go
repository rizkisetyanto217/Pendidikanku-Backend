// internals/features/lembaga/class_subject_books/controller/class_subject_book_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

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

/* =========================================================
   CREATE
   POST /admin/class-subject-books
   Body: CreateClassSubjectBookRequest
   ========================================================= */
func (h *ClassSubjectBookController) Create(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
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
/* =========================================================
   GET BY ID
   GET /admin/class-subject-books/:id[?with_deleted=true][&section_id=...]
   ========================================================= */
func (h *ClassSubjectBookController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	withDeleted := strings.EqualFold(c.Query("with_deleted"), "true")

	// Base + tenant + id
	qBase := h.DB.Table("class_subject_books AS csb").
		Where("csb.class_subject_books_masjid_id = ?", masjidID).
		Where("csb.class_subject_books_id = ?", id)

	// exclude soft-deleted by default
	if !withDeleted {
		qBase = qBase.Where("csb.class_subject_books_deleted_at IS NULL")
	}

	// ===== JOIN (sama seperti List) =====
	qBase = qBase.
		Joins(`
			LEFT JOIN class_subjects AS cs
			  ON cs.class_subjects_id = csb.class_subject_books_class_subject_id
		`).
		// hubungkan ke section via CLASS (bukan kolom section di class_subjects)
		Joins(`
			LEFT JOIN class_sections AS sec
			  ON sec.class_sections_class_id = cs.class_subjects_class_id
		`).
		Joins(`
			LEFT JOIN books AS b
			  ON b.books_id = csb.class_subject_books_book_id
		`).
		// batasi ke section yang memang mengajar subject tsb
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_section_id = sec.class_sections_id
			 AND csst.class_section_subject_teachers_subject_id = cs.class_subjects_subject_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`)

	// (opsional) pilih section spesifik
	if sid := strings.TrimSpace(c.Query("section_id")); sid != "" {
		secID, e := uuid.Parse(sid)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("sec.class_sections_id = ?", secID)
	}

	// Scan struct ringan (sama fieldnya dengan List)
	type row struct {
		// CSB
		ID             uuid.UUID  `gorm:"column:class_subject_books_id"`
		MasjidID       uuid.UUID  `gorm:"column:class_subject_books_masjid_id"`
		ClassSubjectID uuid.UUID  `gorm:"column:class_subject_books_class_subject_id"`
		BookID         uuid.UUID  `gorm:"column:class_subject_books_book_id"`
		IsActive       bool       `gorm:"column:class_subject_books_is_active"`
		Desc           *string    `gorm:"column:class_subject_books_desc"`
		CreatedAt      time.Time  `gorm:"column:class_subject_books_created_at"`
		UpdatedAt      *time.Time `gorm:"column:class_subject_books_updated_at"`
		DeletedAt      *time.Time `gorm:"column:class_subject_books_deleted_at"`

		// Book
		BID       *uuid.UUID `gorm:"column:books_id"`
		BMasjidID *uuid.UUID `gorm:"column:books_masjid_id"`
		BTitle    *string    `gorm:"column:books_title"`
		BAuthor   *string    `gorm:"column:books_author"`
		BURL      *string    `gorm:"column:books_url"`
		BImageURL *string    `gorm:"column:books_image_url"`
		BSlug     *string    `gorm:"column:books_slug"`

		// Section
		SecID       *uuid.UUID `gorm:"column:class_sections_id"`
		SecName     *string    `gorm:"column:class_sections_name"`
		SecSlug     *string    `gorm:"column:class_sections_slug"`
		SecCode     *string    `gorm:"column:class_sections_code"`
		SecCapacity *int       `gorm:"column:class_sections_capacity"`
		SecActive   *bool      `gorm:"column:class_sections_is_active"`
	}

	var r row
	// ORDER biar deterministik kalau tanpa section_id (ambil section yang terbaru aktif duluan)
	if err := qBase.
		Select(`
			csb.class_subject_books_id,
			csb.class_subject_books_masjid_id,
			csb.class_subject_books_class_subject_id,
			csb.class_subject_books_book_id,
			csb.class_subject_books_is_active,
			csb.class_subject_books_desc,
			csb.class_subject_books_created_at,
			csb.class_subject_books_updated_at,
			csb.class_subject_books_deleted_at,

			b.books_id,
			b.books_masjid_id,
			b.books_title,
			b.books_author,
			b.books_url,
			b.books_image_url,
			b.books_slug,

			sec.class_sections_id,
			sec.class_sections_name,
			sec.class_sections_slug,
			sec.class_sections_code,
			sec.class_sections_capacity,
			sec.class_sections_is_active
		`).
		Order("sec.class_sections_created_at DESC NULLS LAST, csb.class_subject_books_created_at DESC").
		Limit(1).
		Scan(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Kalau tidak ada baris (misal karena filter section_id), anggap 404
	// r.ID zero-value? Karena kita pakai Scan, cek saja ID == uuid.Nil
	if r.ID == uuid.Nil {
		return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
	}

	// Map ke DTO (sama seperti List)
	resp := csbDTO.ClassSubjectBookResponse{
		ClassSubjectBooksID:             r.ID,
		ClassSubjectBooksMasjidID:       r.MasjidID,
		ClassSubjectBooksClassSubjectID: r.ClassSubjectID,
		ClassSubjectBooksBookID:         r.BookID,
		ClassSubjectBooksIsActive:       r.IsActive,
		ClassSubjectBooksDesc:           r.Desc,
		ClassSubjectBooksCreatedAt:      r.CreatedAt,
		ClassSubjectBooksUpdatedAt:      r.UpdatedAt,
		ClassSubjectBooksDeletedAt:      r.DeletedAt,
	}
	if r.BID != nil {
		resp.Book = &csbDTO.BookLite{
			BooksID:       *r.BID,
			BooksMasjidID: derefUUID(r.BMasjidID),
			BooksTitle:    derefString(r.BTitle),
			BooksAuthor:   r.BAuthor,
			BooksURL:      r.BURL,
			BooksImageURL: r.BImageURL,
			BooksSlug:     r.BSlug,
		}
	}
	if r.SecID != nil {
		resp.Section = &csbDTO.SectionLite{
			ClassSectionsID:       *r.SecID,
			ClassSectionsName:     derefString(r.SecName),
			ClassSectionsSlug:     derefString(r.SecSlug),
			ClassSectionsCode:     r.SecCode,
			ClassSectionsCapacity: r.SecCapacity,
			ClassSectionsIsActive: derefBool(r.SecActive),
		}
	}

	return helper.JsonOK(c, "Detail class_subject_book", resp)
}

/* =========================================================
   LIST
   GET /admin/class-subject-books
   Query:
     - q                : cari di desc
     - class_subject_id : UUID
     - class_id         : UUID (via class_sections -> class_id)
     - section_id       : UUID
     - subject_id       : UUID (via class_subjects)
     - teacher_id       : UUID (filter via CSST)
     - is_active        : bool
     - with_deleted     : bool
     - sort             : created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
     - limit, offset
   ========================================================= */
func (h *ClassSubjectBookController) List(c *fiber.Ctx) error {
	// ===== tenant: dukung teacher/student/dkm/admin (union)
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// ===== parse query sederhana
	var q csbDTO.ListClassSubjectBookQuery
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

	// ===== base query
	qBase := h.DB.Table("class_subject_books AS csb").
		Where("csb.class_subject_books_masjid_id IN ?", masjidIDs)

	// exclude soft-deleted default
	if q.WithDeleted == nil || !*q.WithDeleted {
		qBase = qBase.Where("csb.class_subject_books_deleted_at IS NULL")
	}

	// ===== JOIN yang diperlukan
	qBase = qBase.
		Joins(`
			LEFT JOIN class_subjects AS cs
			  ON cs.class_subjects_id = csb.class_subject_books_class_subject_id
		`).
		Joins(`
			LEFT JOIN class_sections AS sec
			  ON sec.class_sections_class_id = cs.class_subjects_class_id
			 AND sec.class_sections_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN books AS b
			  ON b.books_id = csb.class_subject_books_book_id
		`).
		// Perbaikan: CSST join via CLASS_SUBJECT (kolom jamak)
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_section_id = sec.class_sections_id
			 AND csst.class_section_subject_teachers_class_subjects_id = cs.class_subjects_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`)

	// ===== filters
	if q.ClassSubjectID != nil {
		qBase = qBase.Where("csb.class_subject_books_class_subject_id = ?", *q.ClassSubjectID)
	}
	if q.BookID != nil {
		qBase = qBase.Where("csb.class_subject_books_book_id = ?", *q.BookID)
	}
	if q.IsActive != nil {
		qBase = qBase.Where("csb.class_subject_books_is_active = ?", *q.IsActive)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		qq := "%" + strings.TrimSpace(*q.Q) + "%"
		qBase = qBase.Where("csb.class_subject_books_desc ILIKE ?", qq)
	}
	// filter by section_id
	if sid := strings.TrimSpace(c.Query("section_id")); sid != "" {
		secID, e := uuid.Parse(sid)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("sec.class_sections_id = ?", secID)
	}
	// filter by class_id
	if cid := strings.TrimSpace(c.Query("class_id")); cid != "" {
		classID, e := uuid.Parse(cid)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_id tidak valid")
		}
		qBase = qBase.Where("sec.class_sections_class_id = ?", classID)
	}
	// filter by subject_id (via class_subjects)
	if sub := strings.TrimSpace(c.Query("subject_id")); sub != "" {
		subID, e := uuid.Parse(sub)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "subject_id tidak valid")
		}
		qBase = qBase.Where("cs.class_subjects_subject_id = ?", subID)
	}
	// filter by teacher_id (via CSST)
	if tid := strings.TrimSpace(c.Query("teacher_id")); tid != "" {
		teacherID, e := uuid.Parse(tid)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "teacher_id tidak valid")
		}
		qBase = qBase.Where("csst.class_section_subject_teachers_teacher_id = ?", teacherID)
	}

	// ===== total distinct
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("csb.class_subject_books_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== sorting whitelist
	sort := "created_at_desc"
	if q.Sort != nil {
		sort = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sort {
	case "created_at_asc":
		qBase = qBase.Order("csb.class_subject_books_created_at ASC")
	case "updated_at_asc":
		qBase = qBase.Order("csb.class_subject_books_updated_at ASC NULLS FIRST")
	case "updated_at_desc":
		qBase = qBase.Order("csb.class_subject_books_updated_at DESC NULLS LAST")
	default:
		qBase = qBase.Order("csb.class_subject_books_created_at DESC")
	}

	// ===== tambah JOIN ke book_urls untuk ambil satu URL utama & satu cover
	qData := qBase.
		Joins(`
			LEFT JOIN LATERAL (
				SELECT bu.book_url_href
				FROM book_urls AS bu
				WHERE bu.book_url_book_id = b.books_id
				  AND bu.book_url_deleted_at IS NULL
				  AND bu.book_url_type IN ('download','purchase','desc')
				ORDER BY
				  CASE bu.book_url_type
				    WHEN 'download' THEN 1
				    WHEN 'purchase' THEN 2
				    WHEN 'desc' THEN 3
				    ELSE 9
				  END,
				  bu.book_url_created_at DESC
				LIMIT 1
			) bu ON TRUE
		`).
		Joins(`
			LEFT JOIN LATERAL (
				SELECT bu2.book_url_href
				FROM book_urls AS bu2
				WHERE bu2.book_url_book_id = b.books_id
				  AND bu2.book_url_deleted_at IS NULL
				  AND bu2.book_url_type = 'cover'
				ORDER BY bu2.book_url_created_at DESC
				LIMIT 1
			) bu_cover ON TRUE
		`)

	// ===== scan struct ringan
	type row struct {
		ID             uuid.UUID  `gorm:"column:class_subject_books_id"`
		MasjidID       uuid.UUID  `gorm:"column:class_subject_books_masjid_id"`
		ClassSubjectID uuid.UUID  `gorm:"column:class_subject_books_class_subject_id"`
		BookID         uuid.UUID  `gorm:"column:class_subject_books_book_id"`
		IsActive       bool       `gorm:"column:class_subject_books_is_active"`
		Desc           *string    `gorm:"column:class_subject_books_desc"`
		CreatedAt      time.Time  `gorm:"column:class_subject_books_created_at"`
		UpdatedAt      *time.Time `gorm:"column:class_subject_books_updated_at"`
		DeletedAt      *time.Time `gorm:"column:class_subject_books_deleted_at"`

		// book
		BID       *uuid.UUID `gorm:"column:books_id"`
		BMasjidID *uuid.UUID `gorm:"column:books_masjid_id"`
		BTitle    *string    `gorm:"column:books_title"`
		BAuthor   *string    `gorm:"column:books_author"`
		BURL      *string    `gorm:"column:books_url"`
		BImageURL *string    `gorm:"column:books_image_url"`
		BSlug     *string    `gorm:"column:books_slug"`

		// section (via class)
		SecID       *uuid.UUID `gorm:"column:class_sections_id"`
		SecName     *string    `gorm:"column:class_sections_name"`
		SecSlug     *string    `gorm:"column:class_sections_slug"`
		SecCode     *string    `gorm:"column:class_sections_code"`
		SecCapacity *int       `gorm:"column:class_sections_capacity"`
		SecActive   *bool      `gorm:"column:class_sections_is_active"`
	}

	var rows []row
	if err := qData.
		Select(`
			csb.class_subject_books_id,
			csb.class_subject_books_masjid_id,
			csb.class_subject_books_class_subject_id,
			csb.class_subject_books_book_id,
			csb.class_subject_books_is_active,
			csb.class_subject_books_desc,
			csb.class_subject_books_created_at,
			csb.class_subject_books_updated_at,
			csb.class_subject_books_deleted_at,

			b.books_id,
			b.books_masjid_id,
			b.books_title,
			b.books_author,
			bu.book_url_href AS books_url,          -- dari book_urls (utama)
			bu_cover.book_url_href AS books_image_url, -- dari book_urls (cover)
			b.books_slug,

			sec.class_sections_id,
			sec.class_sections_name,
			sec.class_sections_slug,
			sec.class_sections_code,
			sec.class_sections_capacity,
			sec.class_sections_is_active
		`).
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ===== map ke DTO (tetap sama)
	items := make([]csbDTO.ClassSubjectBookResponse, 0, len(rows))
	for _, r := range rows {
		resp := csbDTO.ClassSubjectBookResponse{
			ClassSubjectBooksID:             r.ID,
			ClassSubjectBooksMasjidID:       r.MasjidID,
			ClassSubjectBooksClassSubjectID: r.ClassSubjectID,
			ClassSubjectBooksBookID:         r.BookID,
			ClassSubjectBooksIsActive:       r.IsActive,
			ClassSubjectBooksDesc:           r.Desc,
			ClassSubjectBooksCreatedAt:      r.CreatedAt,
			ClassSubjectBooksUpdatedAt:      r.UpdatedAt,
			ClassSubjectBooksDeletedAt:      r.DeletedAt,
		}
		if r.BID != nil {
			resp.Book = &csbDTO.BookLite{
				BooksID:       *r.BID,
				BooksMasjidID: derefUUID(r.BMasjidID),
				BooksTitle:    derefString(r.BTitle),
				BooksAuthor:   r.BAuthor,
				BooksURL:      r.BURL,
				BooksImageURL: r.BImageURL,
				BooksSlug:     r.BSlug,
			}
		}
		if r.SecID != nil {
			resp.Section = &csbDTO.SectionLite{
				ClassSectionsID:       *r.SecID,
				ClassSectionsName:     derefString(r.SecName),
				ClassSectionsSlug:     derefString(r.SecSlug),
				ClassSectionsCode:     r.SecCode,
				ClassSectionsCapacity: r.SecCapacity,
				ClassSectionsIsActive: derefBool(r.SecActive),
			}
		}
		items = append(items, resp)
	}

	return helper.JsonList(c, items, csbDTO.Pagination{
		Limit:  limit,
		Offset: offset,
		Total:  int(total),
	})
}

// helpers
func derefUUID(v *uuid.UUID) uuid.UUID { if v == nil { return uuid.Nil }; return *v }
func derefString(s *string) string    { if s == nil { return "" }; return *s }
func derefBool(b *bool) bool          { if b == nil { return false }; return *b }


// UPDATE (partial)
// PUT /admin/class-subject-books/:id
func (h *ClassSubjectBookController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
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
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// hanya admin yang boleh hard-delete
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
