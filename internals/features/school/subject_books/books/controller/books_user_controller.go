package controller

import (
	dto "masjidku_backend/internals/features/school/subject_books/books/dto"
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
// GET /api/a/books/list
// List ringan by default. Expansion on-demand lewat ?include=...
// include:
//   - usages         → muat relasi pemakaian (class_subject_books → sections)
//   - primary_url    → ambil 1 URL utama (download/purchase/desc) via LATERAL
//   - cover          → ambil cover terbaru via LATERAL
//   - urls / images / book_urls → daftar URL (cover/desc/…) + pagination per-buku
//   - all            → semua di atas
//
// Catatan: kirim img_* (img_types,img_page,img_per_page) juga akan memaksa include daftar URL.
func (h *BooksController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var q dto.BooksWithUsagesListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// ---------- includes (on-demand) ----------
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includes := map[string]bool{}
	for _, s := range strings.Split(includeStr, ",") {
		if s = strings.TrimSpace(s); s != "" {
			includes[s] = true
		}
	}
	includeAll := includeStr == "all" || includes["all"]

	wantUsages := includeAll || includes["usage"] || includes["usages"]
	wantPrimary := includeAll || includes["primary_url"] || includes["main_url"]
	wantCover := includeAll || includes["cover"]

	wantBookURLs := includeAll || includes["urls"] || includes["images"] || includes["book_urls"]
	// presence of img_* params also implies wanting book URLs
	if !wantBookURLs && (strings.TrimSpace(c.Query("img_types")) != "" ||
		strings.TrimSpace(c.Query("img_page")) != "" ||
		strings.TrimSpace(c.Query("img_per_page")) != "") {
		wantBookURLs = true
	}

	// ---------- buku pagination ----------
	limit := 20
	offset := 0
	if q.Limit != nil && *q.Limit > 0 && *q.Limit <= 200 {
		limit = *q.Limit
	}
	if q.Offset != nil && *q.Offset >= 0 {
		offset = *q.Offset
	}

	// ---------- order ----------
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

	// ---------- Filter by ID (single/multi) ----------
	parseIDsCSV := func(s string) ([]uuid.UUID, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		parts := strings.Split(s, ",")
		out := make([]uuid.UUID, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p == "" {
				continue
			}
			id, e := uuid.Parse(p)
			if e != nil {
				return nil, e
			}
			out = append(out, id)
		}
		return out, nil
	}
	idFilter, e1 := parseIDsCSV(c.Query("id"))
	if e1 != nil {
		return helper.JsonError(c, 400, "id berisi UUID tidak valid")
	}
	if len(idFilter) == 0 {
		var e2 error
		idFilter, e2 = parseIDsCSV(c.Query("books_id"))
		if e2 != nil {
			return helper.JsonError(c, 400, "books_id berisi UUID tidak valid")
		}
	}

	// ======================================================
	// 1) QUERY DASAR: hanya tabel books (+ optional primary/cover)
	//    → Hemat: TANPA join usages sama sekali
	// ======================================================
	base := h.DB.Table("books AS b").
		Where("b.books_masjid_id = ?", masjidID)
	if q.WithDeleted == nil || !*q.WithDeleted {
		base = base.Where("b.books_deleted_at IS NULL")
	}
	if len(idFilter) > 0 {
		base = base.Where("b.books_id IN ?", idFilter)
		limit = len(idFilter)
		offset = 0
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

	// Tambahkan LATERAL join hanya jika diminta
	if wantPrimary {
		base = base.Joins(`
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
		`)
	}
	if wantCover {
		base = base.Joins(`
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
	}

	// total books
	var total int64
	if err := base.Session(&gorm.Session{}).
		Distinct("b.books_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// select kolom dasar (+optional primary/cover)
	type bookRow struct {
		BID       uuid.UUID `gorm:"column:books_id"`
		BMasjidID uuid.UUID `gorm:"column:books_masjid_id"`
		BTitle    string    `gorm:"column:books_title"`
		BAuthor   *string   `gorm:"column:books_author"`
		BDesc     *string   `gorm:"column:books_desc"`
		BSlug     *string   `gorm:"column:books_slug"`
		BURL      *string   `gorm:"column:books_url"`
		BImageURL *string   `gorm:"column:books_image_url"`
	}

	// build SELECT dinamis
	selectCols := []string{
		"b.books_id",
		"b.books_masjid_id",
		"b.books_title",
		"b.books_author",
		"b.books_desc",
		"b.books_slug",
	}
	if wantPrimary {
		selectCols = append(selectCols, "bu.book_url_href AS books_url")
	}
	if wantCover {
		selectCols = append(selectCols, "bu_cover.book_url_href AS books_image_url")
	}

	var bookRows []bookRow
	if err := base.
		Select(strings.Join(selectCols, ",\n")).
		Order(orderBy + " " + sortDir).
		Limit(limit).Offset(offset).
		Scan(&bookRows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data buku")
	}
	if len(bookRows) == 0 {
		// kosong: balikin list kosong + meta
		return helper.JsonList(c, []any{}, fiber.Map{"limit": limit, "offset": offset, "total": int(total)})
	}

	// map dasar buku
	bookMap := make(map[uuid.UUID]*dto.BookWithUsagesResponse, len(bookRows))
	orderIDs := make([]uuid.UUID, 0, len(bookRows))
	for _, r := range bookRows {
		bookMap[r.BID] = &dto.BookWithUsagesResponse{
			BooksID:       r.BID,
			BooksMasjidID: r.BMasjidID,
			BooksTitle:    r.BTitle,
			BooksAuthor:   r.BAuthor,
			BooksDesc:     r.BDesc,
			BooksSlug:     r.BSlug,
			BooksURL:      r.BURL,      // hanya ada jika wantPrimary
			BooksImageURL: r.BImageURL, // hanya ada jika wantCover
			Usages:        []dto.BookUsage{},
		}
		orderIDs = append(orderIDs, r.BID)
	}

	// ======================================================
	// 2) (OPSIONAL) MUAT USAGES untuk book_ids pada halaman
	//    → query terpisah supaya base query tidak meledak row-nya
	// ======================================================
	if wantUsages {
		type usageRow struct {
			BookID uuid.UUID `gorm:"column:books_id"`

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
		var urows []usageRow
		if err := h.DB.Table("books AS b").
			Select(`
				b.books_id,
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
			Joins(`LEFT JOIN class_subject_books AS csb
					ON csb.class_subject_books_book_id = b.books_id
					AND csb.class_subject_books_deleted_at IS NULL`).
			Joins(`LEFT JOIN class_subjects AS cs
					ON cs.class_subjects_id = csb.class_subject_books_class_subject_id
					AND cs.class_subjects_deleted_at IS NULL`).
			Joins(`LEFT JOIN class_section_subject_teachers AS csst
					ON csst.class_section_subject_teachers_class_subjects_id = cs.class_subjects_id
					AND csst.class_section_subject_teachers_deleted_at IS NULL
					AND csst.class_section_subject_teachers_masjid_id = b.books_masjid_id`).
			Joins(`LEFT JOIN class_sections AS sec
					ON sec.class_sections_id = csst.class_section_subject_teachers_section_id
					AND sec.class_sections_deleted_at IS NULL`).
			Where("b.books_masjid_id = ?", masjidID).
			Where("b.books_id IN ?", orderIDs).
			Scan(&urows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat usages")
		}

		for _, r := range urows {
			b := bookMap[r.BookID]
			if b == nil || r.CSBID == nil {
				continue
			}
			// cari usage masuk
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
				exists := false
				for _, s := range u.Sections {
					if s.ClassSectionsID == *r.SecID {
						exists = true
						break
					}
				}
				if !exists {
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
	}

	// ======================================================
	// 3) (OPSIONAL) MUAT daftar book_urls (cover/desc/...) dgn pagination per-buku
	// ======================================================
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

	// img paging & types (aktif hanya jika diminta)
	imgPage := 1
	imgPerPage := 20
	if wantBookURLs {
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
	}
	var imgTypes []string
	if wantBookURLs {
		imgTypes = []string{"cover", "desc"}
		if v := strings.TrimSpace(c.Query("img_types")); v != "" {
			var out []string
			for _, p := range strings.Split(v, ",") {
				if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
					out = append(out, p)
				}
			}
			if len(out) > 0 {
				imgTypes = out
			}
		}
	}

	// kumpulkan url hanya jika perlu
	urlMap := make(map[uuid.UUID][]bookURLLite, len(orderIDs))
	if wantBookURLs && len(orderIDs) > 0 {
		var urlRows []bookURLLite
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
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil URL buku")
		}
		for _, r := range urlRows {
			urlMap[r.BookURLBookID] = append(urlMap[r.BookURLBookID], r)
		}
	}

	// compose response
	type bookWithUsagesPlus struct {
		dto.BookWithUsagesResponse
		BookURLs *bookURLsPage `json:"book_urls,omitempty"`
	}
	items := make([]bookWithUsagesPlus, 0, len(orderIDs))
	for _, id := range orderIDs {
		base := bookMap[id]
		var urlsBlk *bookURLsPage // nil → omitempty
		if wantBookURLs {
			all := urlMap[id]
			blk := &bookURLsPage{
				Data:       []bookURLLite{},
				Pagination: pagination{Page: imgPage, PerPage: imgPerPage, Total: 0, TotalPages: 0, HasNext: false, HasPrev: false},
			}
			totalURLs := len(all)
			if totalURLs > 0 {
				start := (imgPage - 1) * imgPerPage
				if start < 0 {
					start = 0
				}
				if start < totalURLs {
					end := start + imgPerPage
					if end > totalURLs {
						end = totalURLs
					}
					blk.Data = all[start:end]
				}
				tp := (totalURLs + imgPerPage - 1) / imgPerPage
				blk.Pagination.Total = totalURLs
				blk.Pagination.TotalPages = tp
				blk.Pagination.HasNext = imgPage < tp
				blk.Pagination.HasPrev = imgPage > 1
			}
			urlsBlk = blk
		}
		items = append(items, bookWithUsagesPlus{
			BookWithUsagesResponse: *base,
			BookURLs:               urlsBlk,
		})
	}

	return helper.JsonList(c, items, fiber.Map{
		"limit":  limit,
		"offset": offset,
		"total":  int(total),
	})
}
