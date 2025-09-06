package controller

import (
	dto "masjidku_backend/internals/features/school/subject_books/books/dto"
	model "masjidku_backend/internals/features/school/subject_books/books/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// helper kecil untuk nilai non-nil
func sPtr(v *string) string { if v == nil { return "" }; return *v }
func bPtr(v *bool) bool     { if v == nil { return false }; return *v }



// ----------------------------------------------------------
// GET /api/a/books/with-usages  (atau /api/a/class-books/books)
// Tampilkan SEMUA buku (parent) + daftar pemakaian (usages) + URL utama & cover

func (h *BooksController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var q dto.BooksWithUsagesListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// ---- buku pagination (existing) ----
	limit := 20
	offset := 0
	if q.Limit != nil && *q.Limit > 0 && *q.Limit <= 200 {
		limit = *q.Limit
	}
	if q.Offset != nil && *q.Offset >= 0 {
		offset = *q.Offset
	}

	// ---- image pagination (baru) ----
	imgPage := 1
	imgPerPage := 20
	if v := strings.TrimSpace(c.Query("img_page")); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			imgPage = n
		}
	}
	if v := strings.TrimSpace(c.Query("img_per_page")); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 && n <= 200 {
			imgPerPage = n
		}
	}
	imgTypes := []string{"cover", "desc"}
	if v := strings.TrimSpace(c.Query("img_types")); v != "" {
		var out []string
		for _, p := range strings.Split(v, ",") {
			p = strings.ToLower(strings.TrimSpace(p))
			if p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			imgTypes = out
		}
	}

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

	// ---- BOOKS as driver (existing) ----
	base := h.DB.Table("books AS b").
		Where("b.books_masjid_id = ?", masjidID)
	if q.WithDeleted == nil || !*q.WithDeleted {
		base = base.Where("b.books_deleted_at IS NULL")
	}

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

	base = base.
		Joins(`
			LEFT JOIN class_subject_books AS csb
			  ON csb.class_subject_books_book_id = b.books_id
			 AND csb.class_subject_books_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_subjects AS cs
			  ON cs.class_subjects_id = csb.class_subject_books_class_subject_id
			 AND (cs.class_subjects_deleted_at IS NULL OR cs.class_subjects_deleted_at IS NULL)
		`).
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_class_subjects_id = cs.class_subjects_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
			 AND csst.class_section_subject_teachers_masjid_id = b.books_masjid_id
		`).
		Joins(`
			LEFT JOIN class_sections AS sec
			  ON sec.class_sections_id = csst.class_section_subject_teachers_section_id
			 AND (sec.class_sections_deleted_at IS NULL OR sec.class_sections_deleted_at IS NULL)
		`).
		// URL utama
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
		// Cover
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

	// total distinct book
	var total int64
	if err := base.Session(&gorm.Session{}).
		Distinct("b.books_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// scan flat rows
	type row struct {
		BID       uuid.UUID `gorm:"column:books_id"`
		BMasjidID uuid.UUID `gorm:"column:books_masjid_id"`
		BTitle    string    `gorm:"column:books_title"`
		BAuthor   *string   `gorm:"column:books_author"`
		BDesc     *string   `gorm:"column:books_desc"`
		BSlug     *string   `gorm:"column:books_slug"`
		BURL      *string   `gorm:"column:books_url"`
		BImageURL *string   `gorm:"column:books_image_url"`

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
			b.books_slug,
			bu.book_url_href       AS books_url,
			bu_cover.book_url_href AS books_image_url,

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

	// group by book (existing)
	bookMap := make(map[uuid.UUID]*dto.BookWithUsagesResponse)
	orderIDs := make([]uuid.UUID, 0, len(rows))

	for _, r := range rows {
		b := bookMap[r.BID]
		if b == nil {
			b = &dto.BookWithUsagesResponse{
				BooksID:       r.BID,
				BooksMasjidID: r.BMasjidID,
				BooksTitle:    r.BTitle,
				BooksAuthor:   r.BAuthor,
				BooksDesc:     r.BDesc,
				BooksSlug:     r.BSlug,
				BooksURL:      r.BURL,
				BooksImageURL: r.BImageURL,
				Usages:        []dto.BookUsage{},
			}
			bookMap[r.BID] = b
			orderIDs = append(orderIDs, r.BID)
		}
		if r.CSBID == nil {
			continue
		}
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

	// —— ambil semua book_urls untuk buku yang tampil, lalu paginate per-buku ——
	type bookURLLite struct {
		BookURLID        uuid.UUID  `json:"book_url_id"         gorm:"column:book_url_id"`
		BookURLMasjidID  uuid.UUID  `json:"book_url_masjid_id"  gorm:"column:book_url_masjid_id"`
		BookURLBookID    uuid.UUID  `json:"book_url_book_id"    gorm:"column:book_url_book_id"`
		BookURLLabel     *string    `json:"book_url_label"      gorm:"column:book_url_label"`
		BookURLType      string     `json:"book_url_type"       gorm:"column:book_url_type"`
		BookURLHref      string     `json:"book_url_href"       gorm:"column:book_url_href"`
		BookURLCreatedAt time.Time  `json:"book_url_created_at" gorm:"column:book_url_created_at"`
		BookURLUpdatedAt time.Time  `json:"book_url_updated_at" gorm:"column:book_url_updated_at"`
	}

	type pagination struct {
		Page       int  `json:"page"`
		PerPage    int  `json:"per_page"`
		Total      int  `json:"total"`
		TotalPages int  `json:"total_pages"`
		HasNext    bool `json:"has_next"`
		HasPrev    bool `json:"has_prev"`
	}

	type bookURLsPage struct {
		Data       []bookURLLite `json:"data"`
		Pagination pagination    `json:"pagination"`
	}

	// query semua url untuk seluruh books pada halaman ini
	urlRows := []bookURLLite{}
	if len(orderIDs) > 0 {
		if err := h.DB.Table("book_urls").
			Select(`
				book_url_id,
				book_url_masjid_id,
				book_url_book_id,
				book_url_label,
				book_url_type,
				book_url_href,
				book_url_created_at,
				book_url_updated_at
			`).
			Where("book_url_book_id IN ?", orderIDs).
			Where("book_url_deleted_at IS NULL").
			Where("book_url_type IN ?", imgTypes).
			Order("book_url_created_at DESC").
			Find(&urlRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil gambar buku")
		}
	}

	// group urls per book_id
	urlMap := make(map[uuid.UUID][]bookURLLite, len(orderIDs))
	for _, r := range urlRows {
		urlMap[r.BookURLBookID] = append(urlMap[r.BookURLBookID], r)
	}

	// wrapper response: tambahkan "book_urls"
	type bookWithUsagesPlus struct {
		dto.BookWithUsagesResponse
		BookURLs *bookURLsPage `json:"book_urls,omitempty"`
	}

	itemsPlus := make([]bookWithUsagesPlus, 0, len(orderIDs))
	for _, id := range orderIDs {
		base := bookMap[id]
		blk := &bookURLsPage{
			Data:       []bookURLLite{},
			Pagination: pagination{Page: imgPage, PerPage: imgPerPage, Total: 0, TotalPages: 0, HasNext: false, HasPrev: false},
		}
		all := urlMap[id]
		total := len(all)
		if total > 0 {
			start := (imgPage - 1) * imgPerPage
			if start < 0 {
				start = 0
			}
			if start < total {
				end := start + imgPerPage
				if end > total {
					end = total
				}
				blk.Data = all[start:end]
			}
			tp := (total + imgPerPage - 1) / imgPerPage
			blk.Pagination.Total = total
			blk.Pagination.TotalPages = tp
			blk.Pagination.HasNext = imgPage < tp
			blk.Pagination.HasPrev = imgPage > 1
		}
		itemsPlus = append(itemsPlus, bookWithUsagesPlus{
			BookWithUsagesResponse: *base,
			BookURLs:               blk,
		})
	}

	// edge case: tidak ada row sama sekali → fallback buku polos
	if len(rows) == 0 {
		var onlyBooks []model.BooksModel
		if err := h.DB.
			Where("books_masjid_id = ? AND books_deleted_at IS NULL", masjidID).
			Order(orderBy + " " + sortDir).
			Limit(limit).Offset(offset).
			Find(&onlyBooks).Error; err == nil {
			for _, b := range onlyBooks {
				itemsPlus = append(itemsPlus, bookWithUsagesPlus{
					BookWithUsagesResponse: dto.BookWithUsagesResponse{
						BooksID:       b.BooksID,
						BooksMasjidID: b.BooksMasjidID,
						BooksTitle:    b.BooksTitle,
						BooksAuthor:   b.BooksAuthor,
						BooksDesc:     b.BooksDesc,
						BooksSlug:     b.BooksSlug,
						BooksURL:      nil,
						BooksImageURL: nil,
						Usages:        []dto.BookUsage{},
					},
					BookURLs: &bookURLsPage{
						Data:       []bookURLLite{},
						Pagination: pagination{Page: imgPage, PerPage: imgPerPage, Total: 0, TotalPages: 0, HasNext: false, HasPrev: false},
					},
				})
			}
		}
	}

	return helper.JsonList(c, itemsPlus, fiber.Map{
		"limit":  limit,
		"offset": offset,
		"total":  int(total),
	})
}


// ----------------------------------------------------------
// GET /api/a/books/:id/with-usages  (atau /api/a/class-books/books/:id)
// Detail 1 buku + usages
// ----------------------------------------------------------
func (h *BooksController) GetWithUsagesByID(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	bookID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

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
			 AND (cs.class_subjects_deleted_at IS NULL OR cs.class_subjects_deleted_at IS NULL)
		`).
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_class_subjects_id = cs.class_subjects_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
			 AND csst.class_section_subject_teachers_masjid_id = b.books_masjid_id
		`).
		Joins(`
			LEFT JOIN class_sections AS sec
			  ON sec.class_sections_id = csst.class_section_subject_teachers_section_id
			 AND (sec.class_sections_deleted_at IS NULL OR sec.class_sections_deleted_at IS NULL)
		`)

	type row struct {
		BID       uuid.UUID `gorm:"column:books_id"`
		BMasjidID uuid.UUID `gorm:"column:books_masjid_id"`
		BTitle    string    `gorm:"column:books_title"`
		BAuthor   *string   `gorm:"column:books_author"`
		BDesc     *string   `gorm:"column:books_desc"`
		BSlug     *string   `gorm:"column:books_slug"`

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

	out := dto.BookWithUsagesResponse{
		BooksID:       rows[0].BID,
		BooksMasjidID: rows[0].BMasjidID,
		BooksTitle:    rows[0].BTitle,
		BooksAuthor:   rows[0].BAuthor,
		BooksDesc:     rows[0].BDesc,
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
