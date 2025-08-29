package controller

import (
	dto "masjidku_backend/internals/features/school/class_books/dto"
	model "masjidku_backend/internals/features/school/class_books/model"
	helper "masjidku_backend/internals/helpers"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ----------------------------------------------------------
// GET /api/a/class-books/with-usages
// Tampilkan SEMUA buku (books = parent) + daftar pemakaian
// ----------------------------------------------------------
func (h *BooksController) ListWithUsages(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var q dto.BooksWithUsagesListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	limit := 20
	offset := 0
	if q.Limit != nil && *q.Limit > 0 && *q.Limit <= 200 { limit = *q.Limit }
	if q.Offset != nil && *q.Offset >= 0 { offset = *q.Offset }

	// order
	orderBy := "b.books_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(strings.TrimSpace(*q.OrderBy)) {
		case "books_title":
			orderBy = "b.books_title"
		case "books_author":
			orderBy = "b.books_author"
		case "created_at":
			orderBy = "b.books_created_at"
		}
	}
	sortDir := "DESC"
	if q.Sort != nil && strings.EqualFold(strings.TrimSpace(*q.Sort), "asc") {
		sortDir = "ASC"
	}

	// ---- BOOKS as driver
	base := h.DB.Table("books AS b").
		Where("b.books_masjid_id = ?", masjidID)

	if q.WithDeleted == nil || !*q.WithDeleted {
		base = base.Where("b.books_deleted_at IS NULL")
	}

	// filter/q sama seperti List biasa
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		needle := "%" + strings.TrimSpace(*q.Q) + "%"
		base = base.Where(h.DB.
			Where("b.books_title ILIKE ?", needle).
			Or("b.books_author ILIKE ?", needle).
			Or("b.books_desc ILIKE ?", needle))
	}
	if q.Author != nil && strings.TrimSpace(*q.Author) != "" {
		base = base.Where("b.books_author ILIKE ?", strings.TrimSpace(*q.Author))
	}
	if q.HasImage != nil {
		if *q.HasImage {
			base = base.Where("b.books_image_url IS NOT NULL AND b.books_image_url <> ''")
		} else {
			base = base.Where("(b.books_image_url IS NULL OR b.books_image_url = '')")
		}
	}
	if q.HasURL != nil {
		if *q.HasURL {
			base = base.Where("b.books_url IS NOT NULL AND b.books_url <> ''")
		} else {
			base = base.Where("(b.books_url IS NULL OR b.books_url = '')")
		}
	}

	// ---- LEFT JOIN pemakaian
	base = base.
		Joins(`
			LEFT JOIN class_subject_books AS csb
			  ON csb.class_subject_books_book_id = b.books_id
			 AND csb.class_subject_books_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_subjects AS cs
			  ON cs.class_subjects_id = csb.class_subject_books_class_subject_id
		`).
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_subject_id = cs.class_subjects_subject_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_sections AS sec
			  ON sec.class_sections_id = csst.class_section_subject_teachers_section_id
		`)

	// total distinct book
	var total int64
	if err := base.Session(&gorm.Session{}).
		Distinct("b.books_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// scan flat rows, lalu group di Go
	type row struct {
		// book
		BID        uuid.UUID  `gorm:"column:books_id"`
		BMasjidID  uuid.UUID  `gorm:"column:books_masjid_id"`
		BTitle     string     `gorm:"column:books_title"`
		BAuthor    *string    `gorm:"column:books_author"`
		BDesc      *string    `gorm:"column:books_desc"`
		BURL       *string    `gorm:"column:books_url"`
		BImageURL  *string    `gorm:"column:books_image_url"`
		BSlug      *string    `gorm:"column:books_slug"`

		// usage
		CSBID *uuid.UUID `gorm:"column:class_subject_books_id"`
		CSID  *uuid.UUID `gorm:"column:class_subjects_id"`
		SID   *uuid.UUID `gorm:"column:subjects_id"`
		CID   *uuid.UUID `gorm:"column:classes_id"`

		// section
		SecID   *uuid.UUID `gorm:"column:class_sections_id"`
		SecName *string    `gorm:"column:class_sections_name"`
		SecSlug *string    `gorm:"column:class_sections_slug"`
		SecCode *string    `gorm:"column:class_sections_code"`
		SecCap  *int       `gorm:"column:class_sections_capacity"`
		SecAct  *bool      `gorm:"column:class_sections_is_active"`
	}

	var rows []row
	if err := base.
		Select(`
			b.books_id,
			b.books_masjid_id,
			b.books_title,
			b.books_author,
			b.books_desc,
			b.books_url,
			b.books_image_url,
			b.books_slug,

			csb.class_subject_books_id,
			cs.class_subjects_id,
			cs.class_subjects_subject_id AS subjects_id,
			cs.class_subjects_class_id   AS classes_id,

			sec.class_sections_id,
			sec.class_sections_name,
			sec.class_sections_slug,
			sec.class_sections_code,
			sec.class_sections_capacity,
			sec.class_sections_is_active
		`).
		Order(orderBy + " " + sortDir).
		Limit(limit).Offset(offset).
		Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// group by book
	bookMap := make(map[uuid.UUID]*dto.BookWithUsagesResponse)
	order := make([]uuid.UUID, 0, len(rows))

	for _, r := range rows {
		b := bookMap[r.BID]
		if b == nil {
			b = &dto.BookWithUsagesResponse{
				BooksID:       r.BID,
				BooksMasjidID: r.BMasjidID,
				BooksTitle:    r.BTitle,
				BooksAuthor:   r.BAuthor,
				BooksDesc:     r.BDesc,
				BooksURL:      r.BURL,
				BooksImageURL: r.BImageURL,
				BooksSlug:     r.BSlug,
				Usages:        []dto.BookUsage{},
			}
			bookMap[r.BID] = b
			order = append(order, r.BID)
		}

		// jika buku belum dipakai, lewati usage
		if r.CSBID == nil {
			continue
		}

		// cari usage by CSBID
		var u *dto.BookUsage
		for i := range b.Usages {
			if b.Usages[i].ClassSubjectBooksID == *r.CSBID {
				u = &b.Usages[i]
				break
			}
		}
		if u == nil {
			u = &dto.BookUsage{
				ClassSubjectBooksID: *r.CSBID,
				ClassSubjectID:      r.CSID,
				SubjectsID:          r.SID,
				ClassesID:           r.CID,
				Sections:            []dto.BookUsageSectionLite{},
			}
			b.Usages = append(b.Usages, *u)
			u = &b.Usages[len(b.Usages)-1]
		}

		// append section (de-dup)
		if r.SecID != nil {
			found := false
			for _, s := range u.Sections {
				if s.ClassSectionsID == *r.SecID {
					found = true
					break
				}
			}
			if !found {
				u.Sections = append(u.Sections, dto.BookUsageSectionLite{
					ClassSectionsID:       *r.SecID,
					ClassSectionsName:     sPtr(r.SecName),
					ClassSectionsSlug:     sPtr(r.SecSlug),
					ClassSectionsCode:     r.SecCode,
					ClassSectionsCapacity: r.SecCap,
					ClassSectionsIsActive: bPtr(r.SecAct),
				})
			}
		}
	}

	// flatten sesuai order
	items := make([]dto.BookWithUsagesResponse, 0, len(order))
	for _, id := range order {
		items = append(items, *bookMap[id])
	}
	// edge case: jika tidak ada satupun row, ambil daftar buku polos (tanpa usages)
	if len(rows) == 0 {
		var onlyBooks []model.BooksModel
		if err := h.DB.Where("books_masjid_id = ? AND books_deleted_at IS NULL", masjidID).
			Order(orderBy + " " + sortDir).
			Limit(limit).Offset(offset).Find(&onlyBooks).Error; err == nil {
			for _, b := range onlyBooks {
				items = append(items, dto.BookWithUsagesResponse{
					BooksID:       b.BooksID,
					BooksMasjidID: b.BooksMasjidID,
					BooksTitle:    b.BooksTitle,
					BooksAuthor:   b.BooksAuthor,
					BooksDesc:     b.BooksDesc,
					BooksURL:      b.BooksURL,
					BooksImageURL: b.BooksImageURL,
					BooksSlug:     b.BooksSlug,
					Usages:        []dto.BookUsage{},
				})
			}
		}
	}

	return helper.JsonList(c, items, fiber.Map{
		"limit":  limit,
		"offset": offset,
		"total":  int(total),
	})
}

// ----------------------------------------------------------
// GET /api/a/class-books/:id/with-usages
// Bentuk hasil sama persis dgn ListWithUsages, tapi 1 buku
// ----------------------------------------------------------
func (h *BooksController) GetWithUsagesByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	bookID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// gunakan query yang sama, tapi filter b.books_id
	base := h.DB.Table("books AS b").
		Where("b.books_masjid_id = ? AND b.books_id = ?", masjidID, bookID).
		Where("b.books_deleted_at IS NULL").
		Joins(`
			LEFT JOIN class_subject_books AS csb
			  ON csb.class_subject_books_book_id = b.books_id
			 AND csb.class_subject_books_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_subjects AS cs
			  ON cs.class_subjects_id = csb.class_subject_books_class_subject_id
		`).
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_subject_id = cs.class_subjects_subject_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_sections AS sec
			  ON sec.class_sections_id = csst.class_section_subject_teachers_section_id
		`)

	type row struct {
		BID        uuid.UUID  `gorm:"column:books_id"`
		BMasjidID  uuid.UUID  `gorm:"column:books_masjid_id"`
		BTitle     string     `gorm:"column:books_title"`
		BAuthor    *string    `gorm:"column:books_author"`
		BDesc      *string    `gorm:"column:books_desc"`
		BURL       *string    `gorm:"column:books_url"`
		BImageURL  *string    `gorm:"column:books_image_url"`
		BSlug      *string    `gorm:"column:books_slug"`

		CSBID *uuid.UUID `gorm:"column:class_subject_books_id"`
		CSID  *uuid.UUID `gorm:"column:class_subjects_id"`
		SID   *uuid.UUID `gorm:"column:subjects_id"`
		CID   *uuid.UUID `gorm:"column:classes_id"`

		SecID   *uuid.UUID `gorm:"column:class_sections_id"`
		SecName *string    `gorm:"column:class_sections_name"`
		SecSlug *string    `gorm:"column:class_sections_slug"`
		SecCode *string    `gorm:"column:class_sections_code"`
		SecCap  *int       `gorm:"column:class_sections_capacity"`
		SecAct  *bool      `gorm:"column:class_sections_is_active"`
	}

	var rows []row
	if err := base.
		Select(`
			b.books_id,
			b.books_masjid_id,
			b.books_title,
			b.books_author,
			b.books_desc,
			b.books_url,
			b.books_image_url,
			b.books_slug,

			csb.class_subject_books_id,
			cs.class_subjects_id,
			cs.class_subjects_subject_id AS subjects_id,
			cs.class_subjects_class_id   AS classes_id,

			sec.class_sections_id,
			sec.class_sections_name,
			sec.class_sections_slug,
			sec.class_sections_code,
			sec.class_sections_capacity,
			sec.class_sections_is_active
		`).Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if len(rows) == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Buku tidak ditemukan")
	}

	// group 1 buku
	out := dto.BookWithUsagesResponse{
		BooksID:       rows[0].BID,
		BooksMasjidID: rows[0].BMasjidID,
		BooksTitle:    rows[0].BTitle,
		BooksAuthor:   rows[0].BAuthor,
		BooksDesc:     rows[0].BDesc,
		BooksURL:      rows[0].BURL,
		BooksImageURL: rows[0].BImageURL,
		BooksSlug:     rows[0].BSlug,
		Usages:        []dto.BookUsage{},
	}
	usageIndex := map[uuid.UUID]int{}

	for _, r := range rows {
		if r.CSBID == nil {
			continue
		}
		idx, ok := usageIndex[*r.CSBID]
		if !ok {
			out.Usages = append(out.Usages, dto.BookUsage{
				ClassSubjectBooksID: *r.CSBID,
				ClassSubjectID:      r.CSID,
				SubjectsID:          r.SID,
				ClassesID:           r.CID,
				Sections:            []dto.BookUsageSectionLite{},
			})
			idx = len(out.Usages) - 1
			usageIndex[*r.CSBID] = idx
		}
		if r.SecID != nil {
			// de-dup sederhana
			exists := false
			for _, s := range out.Usages[idx].Sections {
				if s.ClassSectionsID == *r.SecID {
					exists = true
					break
				}
			}
			if !exists {
				out.Usages[idx].Sections = append(out.Usages[idx].Sections, dto.BookUsageSectionLite{
					ClassSectionsID:       *r.SecID,
					ClassSectionsName:     sPtr(r.SecName),
					ClassSectionsSlug:     sPtr(r.SecSlug),
					ClassSectionsCode:     r.SecCode,
					ClassSectionsCapacity: r.SecCap,
					ClassSectionsIsActive: bPtr(r.SecAct),
				})
			}
		}
	}

	return helper.JsonOK(c, "Detail buku dengan usage", out)
}
