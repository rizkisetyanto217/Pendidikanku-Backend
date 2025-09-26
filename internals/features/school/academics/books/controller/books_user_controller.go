// file: internals/features/lembaga/class_books/controller/books_controller.go
package controller

import (
	"errors"
	dto "masjidku_backend/internals/features/school/academics/books/dto"
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
func sPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func bPtr(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

// ----------------------------------------------------------
// GET /api/a/books/list
// include:
//   - usages         → relasi pemakaian (class_subject_books → sections)
//   - primary_url    → ambil 1 URL utama (download/purchase/desc) via LATERAL
//   - cover          → ambil cover terbaru via LATERAL
//   - urls/images/book_urls → daftar URL (cover/desc/…) + pagination per-buku
//   - all            → semua di atas
//
// Catatan: img_* (img_types,img_page,img_per_page) akan memaksa include daftar URL.
// GET /api/a/books/list
func (h *BooksController) List(c *fiber.Ctx) error {
	// ===== Masjid context (PUBLIC): no role check =====
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}

	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve masjid dari slug")
		}
		masjidID = id
	} else {
		return helperAuth.ErrMasjidContextMissing
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
	if !wantBookURLs && (strings.TrimSpace(c.Query("img_types")) != "" ||
		strings.TrimSpace(c.Query("img_page")) != "" ||
		strings.TrimSpace(c.Query("img_per_page")) != "") {
		wantBookURLs = true
	}

	// ---------- Pagination & sorting via helper ----------
	// default: sort_by=created_at, order=desc, preset AdminOpts
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

	// Back-compat: ?order_by & ?sort dari DTO lama
	if q.OrderBy != nil {
		switch strings.ToLower(strings.TrimSpace(*q.OrderBy)) {
		case "book_title":
			p.SortBy = "title"
		case "book_author":
			p.SortBy = "author"
		case "created_at":
			p.SortBy = "created_at"
		}
	}
	if q.Sort != nil && strings.TrimSpace(*q.Sort) != "" {
		p.SortOrder = strings.ToLower(strings.TrimSpace(*q.Sort)) // asc|desc (helper sudah guard)
	}

	// Whitelist ORDER BY (aman)
	allowedSort := map[string]string{
		"created_at": "b.book_created_at",
		"title":      "b.book_title",
		"author":     "b.book_author",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

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
		idFilter, e2 = parseIDsCSV(c.Query("book_id")) // singkron ke kolom singular
		if e2 != nil {
			return helper.JsonError(c, 400, "book_id berisi UUID tidak valid")
		}
	}

	// ======================================================
	// 1) QUERY DASAR: hanya tabel books (+ optional primary/cover)
	// ======================================================
	base := h.DB.Table("books AS b").
		Where("b.book_masjid_id = ?", masjidID)
	if q.WithDeleted == nil || !*q.WithDeleted {
		base = base.Where("b.book_deleted_at IS NULL")
	}
	if len(idFilter) > 0 {
		base = base.Where("b.book_id IN ?", idFilter)
		// tampilkan semua requested ids dalam satu halaman
		p.Page = 1
		p.PerPage = len(idFilter)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		needle := "%" + strings.TrimSpace(*q.Q) + "%"
		base = base.Where(h.DB.
			Where("b.book_title ILIKE ?", needle).
			Or("b.book_author ILIKE ?", needle).
			Or("b.book_desc ILIKE ?", needle))
	}
	if q.Author != nil && strings.TrimSpace(*q.Author) != "" {
		base = base.Where("b.book_author ILIKE ?", strings.TrimSpace(*q.Author))
	}

	// LATERAL primary_url
	if wantPrimary {
		base = base.Joins(`
			LEFT JOIN LATERAL (
				SELECT bu.book_url_href
				FROM book_urls AS bu
				WHERE bu.book_url_book_id = b.book_id
				  AND bu.book_url_deleted_at IS NULL
				  AND bu.book_url_kind IN ('download','purchase','desc')
				ORDER BY
				  CASE bu.book_url_kind
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
	// LATERAL cover
	if wantCover {
		base = base.Joins(`
			LEFT JOIN LATERAL (
				SELECT bu2.book_url_href
				FROM book_urls AS bu2
				WHERE bu2.book_url_book_id = b.book_id
				  AND bu2.book_url_deleted_at IS NULL
				  AND bu2.book_url_kind = 'cover'
				ORDER BY bu2.book_url_created_at DESC
				LIMIT 1
			) bu_cover ON TRUE
		`)
	}

	// total books
	var total int64
	if err := base.Session(&gorm.Session{}).
		Distinct("b.book_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// select kolom dasar (+optional primary/cover)
	type bookRow struct {
		BID       uuid.UUID `gorm:"column:book_id"`
		BMasjidID uuid.UUID `gorm:"column:book_masjid_id"`
		BTitle    string    `gorm:"column:book_title"`
		BAuthor   *string   `gorm:"column:book_author"`
		BDesc     *string   `gorm:"column:book_desc"`
		BSlug     *string   `gorm:"column:book_slug"`
		BURL      *string   `gorm:"column:book_url"`
		BImageURL *string   `gorm:"column:book_image_url"`
	}

	selectCols := []string{
		"b.book_id",
		"b.book_masjid_id",
		"b.book_title",
		"b.book_author",
		"b.book_desc",
		"b.book_slug",
	}
	if wantPrimary {
		selectCols = append(selectCols, "bu.book_url_href AS book_url")
	}
	if wantCover {
		selectCols = append(selectCols, "bu_cover.book_url_href AS book_image_url")
	}

	var bookRows []bookRow
	if err := base.
		Select(strings.Join(selectCols, ",\n")).
		Order(orderExpr).
		Limit(p.Limit()).Offset(p.Offset()).
		Scan(&bookRows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data buku")
	}
	if len(bookRows) == 0 {
		meta := helper.BuildMeta(total, p)
		return helper.JsonList(c, []any{}, meta)
	}

	// map dasar buku
	bookMap := make(map[uuid.UUID]*dto.BookWithUsagesResponse, len(bookRows))
	orderIDs := make([]uuid.UUID, 0, len(bookRows))
	for _, r := range bookRows {
		bookMap[r.BID] = &dto.BookWithUsagesResponse{
			BookID:       r.BID,
			BookMasjidID: r.BMasjidID,
			BookTitle:    r.BTitle,
			BookAuthor:   r.BAuthor,
			BookDesc:     r.BDesc,
			BookSlug:     r.BSlug,
			BookURL:      r.BURL,
			BookImageURL: r.BImageURL,
			Usages:       []dto.BookUsage{},
		}
		orderIDs = append(orderIDs, r.BID)
	}

	// 2) (opsional) usages
	if wantUsages {
		type usageRow struct {
			BookID uuid.UUID `gorm:"column:book_id"`

			CSBID *uuid.UUID `gorm:"column:class_subject_book_id"`
			CSID  *uuid.UUID `gorm:"column:class_subject_id"`
			SID   *uuid.UUID `gorm:"column:subject_id"`
			CID   *uuid.UUID `gorm:"column:class_id"`

			SecID   *uuid.UUID `gorm:"column:class_section_id"`
			SecName *string    `gorm:"column:class_section_name"`
			SecSlug *string    `gorm:"column:class_section_slug"`
			SecCode *string    `gorm:"column:class_section_code"`
			SecCap  *int       `gorm:"column:class_section_capacity"`
			SecAct  *bool      `gorm:"column:class_section_is_active"`
		}
		var urows []usageRow
		if err := h.DB.Table("books AS b").
			Select(`
				b.book_id,
				csb.class_subject_book_id,
				cs.class_subject_id,
				cs.class_subject_subject_id AS subject_id,
				cs.class_subject_class_id   AS class_id,
				sec.class_section_id,
				sec.class_section_name,
				sec.class_section_slug,
				sec.class_section_code,
				sec.class_section_capacity,
				sec.class_section_is_active
			`).
			Joins(`LEFT JOIN class_subject_books AS csb
					ON csb.class_subject_book_book_id = b.book_id
					AND csb.class_subject_book_deleted_at IS NULL`).
			Joins(`LEFT JOIN class_subjects AS cs
					ON cs.class_subject_id = csb.class_subject_book_class_subject_id
					AND cs.class_subject_deleted_at IS NULL`).
			Joins(`LEFT JOIN class_section_subject_teachers AS csst
					ON csst.class_section_subject_teacher_class_subject_id = cs.class_subject_id
					AND csst.class_section_subject_teacher_deleted_at IS NULL
					AND csst.class_section_subject_teacher_masjid_id = b.book_masjid_id`).
			Joins(`LEFT JOIN class_sections AS sec
					ON sec.class_section_id = csst.class_section_subject_teacher_section_id
					AND sec.class_section_deleted_at IS NULL`).
			Where("b.book_masjid_id = ?", masjidID).
			Where("b.book_id IN ?", orderIDs).
			Scan(&urows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat usages")
		}

		for _, r := range urows {
			b := bookMap[r.BookID]
			if b == nil || r.CSBID == nil {
				continue
			}
			var u *dto.BookUsage
			for i := range b.Usages {
				if b.Usages[i].ClassSubjectBookID == *r.CSBID {
					u = &b.Usages[i]
					break
				}
			}
			if u == nil {
				u = &dto.BookUsage{
					ClassSubjectBookID: *r.CSBID,
					ClassSubjectID:     r.CSID,
					SubjectID:          r.SID,
					ClassID:            r.CID,
					Sections:           []dto.BookUsageSectionLite{},
				}
				b.Usages = append(b.Usages, *u)
				u = &b.Usages[len(b.Usages)-1]
			}
			if r.SecID != nil {
				exists := false
				for _, s := range u.Sections {
					if s.ClassSectionID == *r.SecID {
						exists = true
						break
					}
				}
				if !exists {
					u.Sections = append(u.Sections, dto.BookUsageSectionLite{
						ClassSectionID:       *r.SecID,
						ClassSectionName:     sPtr(r.SecName),
						ClassSectionSlug:     sPtr(r.SecSlug),
						ClassSectionCode:     r.SecCode,
						ClassSectionCapacity: r.SecCap,
						ClassSectionIsActive: bPtr(r.SecAct),
					})
				}
			}
		}
	}

	// 3) (opsional) daftar book_urls (cover/desc/…)
	type bookURLLite struct {
		BookURLID        uuid.UUID `json:"book_url_id"         gorm:"column:book_url_id"`
		BookURLMasjidID  uuid.UUID `json:"book_url_masjid_id"  gorm:"column:book_url_masjid_id"`
		BookURLBookID    uuid.UUID `json:"book_url_book_id"    gorm:"column:book_url_book_id"`
		BookURLLabel     *string   `json:"book_url_label"      gorm:"column:book_url_label"`
		BookURLKind      string    `json:"book_url_kind"       gorm:"column:book_url_kind"`
		BookURLHref      string    `json:"book_url_href"       gorm:"column:book_url_href"`
		BookURLCreatedAt time.Time `json:"book_url_created_at" gorm:"column:book_url_created_at"`
		BookURLUpdatedAt time.Time `json:"book_url_updated_at" gorm:"column:book_url_updated_at"`
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
	var imgKinds []string
	if wantBookURLs {
		imgKinds = []string{"cover", "desc"}
		if v := strings.TrimSpace(c.Query("img_types")); v != "" {
			var out []string
			for _, p2 := range strings.Split(v, ",") {
				if p2 = strings.ToLower(strings.TrimSpace(p2)); p2 != "" {
					out = append(out, p2)
				}
			}
			if len(out) > 0 {
				imgKinds = out
			}
		}
	}

	urlMap := make(map[uuid.UUID][]bookURLLite, len(orderIDs))
	if wantBookURLs && len(orderIDs) > 0 {
		var urlRows []bookURLLite
		if err := h.DB.Table("book_urls").
			Select(`
				book_url_id,
				book_url_masjid_id,
				book_url_book_id,
				book_url_label,
				book_url_kind,
				book_url_href,
				book_url_created_at,
				book_url_updated_at
			`).
			Where("book_url_book_id IN ?", orderIDs).
			Where("book_url_deleted_at IS NULL").
			Where("book_url_kind IN ?", imgKinds).
			Order("book_url_created_at DESC").
			Find(&urlRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil URL buku")
		}
		for _, r := range urlRows {
			urlMap[r.BookURLBookID] = append(urlMap[r.BookURLBookID], r)
		}
	}

	type bookWithUsagesPlus struct {
		dto.BookWithUsagesResponse
		BookURLs *bookURLsPage `json:"book_urls,omitempty"`
	}
	items := make([]bookWithUsagesPlus, 0, len(orderIDs))
	for _, id := range orderIDs {
		base := bookMap[id]
		var urlsBlk *bookURLsPage
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

	// Response meta pakai helper
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}
