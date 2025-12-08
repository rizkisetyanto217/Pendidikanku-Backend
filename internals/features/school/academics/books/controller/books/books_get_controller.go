// file: internals/features/lembaga/class_books/controller/books_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	bookdto "madinahsalam_backend/internals/features/school/academics/books/dto"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
GET /api/a/books/list  (admin, token-based)
atau versi PUBLIC yang bawa context:
  - GET /api/u/:school_id/books/list
  - GET /api/u/m/:school_slug/books/list

Skenario resolver school:

1) Kalau ada token & active school → pakai school dari token.
2) Kalau tidak ada / gagal → pakai ResolveSchoolContext:
  - mc.ID (UUID),
  - atau mc.Slug yang bisa berisi UUID / slug.

3) Kalau tetap tidak ada → ErrSchoolContextMissing.
*/
func (h *BooksController) List(c *fiber.Ctx) error {
	// Pastikan DB tersedia di Locals untuk helper lain
	c.Locals("DB", h.DB)

	// ===== School context (token-aware, PUBLIC) =====
	var schoolID uuid.UUID

	// 1) Coba dari token dulu
	if id, err := helperAuth.GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		// 2) Fallback: ResolveSchoolContext (bisa dari path / query / dsb)
		mc, err2 := helperAuth.ResolveSchoolContext(c)
		if err2 != nil {
			return err2
		}

		switch {
		case mc.ID != uuid.Nil:
			// Sudah ada ID langsung
			schoolID = mc.ID

		case strings.TrimSpace(mc.Slug) != "":
			// mc.Slug bisa:
			// - sebenarnya UUID di path (/:school_id)
			// - atau slug beneran
			s := strings.TrimSpace(mc.Slug)
			if id2, errParse := uuid.Parse(s); errParse == nil {
				// Ternyata UUID → pakai langsung
				schoolID = id2
			} else {
				// Beneran slug → resolve via DB
				id2, er := helperAuth.GetSchoolIDBySlug(c, s)
				if er != nil {
					if errors.Is(er, gorm.ErrRecordNotFound) {
						return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
					}
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
				}
				schoolID = id2
			}

		default:
			// Tidak ada ID, tidak ada slug
			return helperAuth.ErrSchoolContextMissing
		}
	}

	// ===== Query params dasar =====
	q := strings.TrimSpace(c.Query("q"))
	author := strings.TrimSpace(c.Query("author"))
	name := strings.TrimSpace(c.Query("name")) // 🔍 filter spesifik judul buku
	withDeleted := strings.EqualFold(strings.TrimSpace(c.Query("with_deleted")), "true")

	// mode: compact | full (default: full)
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	isCompact := mode == "compact"

	// ===== Pagination & sorting =====
	// default: sort_by=created_at, order=desc (helper.AdminOpts)
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

	// Back-compat: order_by/sort ala lama
	if v := strings.TrimSpace(c.Query("order_by")); v != "" {
		switch strings.ToLower(v) {
		case "book_title", "title":
			p.SortBy = "title"
		case "book_author", "author":
			p.SortBy = "author"
		case "created_at":
			p.SortBy = "created_at"
		}
	}
	if v := strings.TrimSpace(c.Query("sort")); v != "" {
		p.SortOrder = strings.ToLower(v) // asc|desc (helper sudah guard)
	}

	// Whitelist kolom ORDER BY
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

	// ===== Filter id/book_id (CSV UUID) =====
	parseIDsCSV := func(s string) ([]uuid.UUID, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		ps := strings.Split(s, ",")
		out := make([]uuid.UUID, 0, len(ps))
		for _, one := range ps {
			one = strings.TrimSpace(one)
			if one == "" {
				continue
			}
			id, e := uuid.Parse(one)
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
		if tmp, e2 := parseIDsCSV(c.Query("book_id")); e2 != nil {
			return helper.JsonError(c, 400, "book_id berisi UUID tidak valid")
		} else {
			idFilter = tmp
		}
	}

	// ===== Query dasar =====
	type row struct {
		BookID             uuid.UUID  `json:"book_id"               gorm:"column:book_id"`
		BookSchoolID       uuid.UUID  `json:"book_school_id"        gorm:"column:book_school_id"`
		BookTitle          string     `json:"book_title"            gorm:"column:book_title"`
		BookAuthor         *string    `json:"book_author,omitempty" gorm:"column:book_author"`
		BookDesc           *string    `json:"book_desc,omitempty"   gorm:"column:book_desc"`
		BookSlug           *string    `json:"book_slug,omitempty"   gorm:"column:book_slug"`
		BookImageURL       *string    `json:"book_image_url,omitempty"        gorm:"column:book_image_url"`
		BookImageObjectKey *string    `json:"book_image_object_key,omitempty" gorm:"column:book_image_object_key"`
		BookPurchaseURL    *string    `json:"book_purchase_url,omitempty"     gorm:"column:book_purchase_url"`
		BookCreatedAt      time.Time  `json:"book_created_at"       gorm:"column:book_created_at"`
		BookUpdatedAt      time.Time  `json:"book_updated_at"       gorm:"column:book_updated_at"`
		BookDeletedAt      *time.Time `json:"-"                     gorm:"column:book_deleted_at"`
		BookIsDeleted      bool       `json:"book_is_deleted"       gorm:"-"`
	}

	base := h.DB.Table("books AS b").
		Where("b.book_school_id = ?", schoolID)

	if !withDeleted {
		base = base.Where("b.book_deleted_at IS NULL")
	}
	if len(idFilter) > 0 {
		base = base.Where("b.book_id IN ?", idFilter)
		// jika by-id, tampilkan semua id dalam satu halaman
		p.Page = 1
		p.PerPage = len(idFilter)
	}
	if q != "" {
		needle := "%" + q + "%"
		base = base.Where(
			h.DB.Where("b.book_title ILIKE ?", needle).
				Or("b.book_author ILIKE ?", needle).
				Or("b.book_desc ILIKE ?", needle),
		)
	}

	// 🔍 filter spesifik by author: ?author= (sekalian aku buat contain-search)
	if author != "" {
		needle := "%" + author + "%"
		base = base.Where("b.book_author ILIKE ?", needle)
	}

	// 🔍 filter spesifik by book title: ?name=
	if name != "" {
		needle := "%" + name + "%"
		base = base.Where("b.book_title ILIKE ?", needle)
	}

	// ===== Count total (distinct book_id) =====
	var total int64
	if err := base.Session(&gorm.Session{}).
		Distinct("b.book_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// ===== Ambil data halaman =====
	var rows []row
	if err := base.
		Select(`
			b.book_id,
			b.book_school_id,
			b.book_title,
			b.book_author,
			b.book_desc,
			b.book_slug,
			b.book_image_url,
			b.book_image_object_key,
			b.book_purchase_url,
			b.book_created_at,
			b.book_updated_at,
			b.book_deleted_at
		`).
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data buku")
	}

	// post-process is_deleted
	for i := range rows {
		rows[i].BookIsDeleted = rows[i].BookDeletedAt != nil && !rows[i].BookDeletedAt.IsZero()
	}

	// ===== Pagination meta (pakai helper standar) =====
	pg := helper.BuildPaginationFromOffset(total, p.Offset(), p.Limit())

	// ===== mode compact vs full =====

	if isCompact {
		// Map ke DTO compact
		out := make([]bookdto.BookCompact, 0, len(rows))
		for _, r := range rows {
			out = append(out, bookdto.BookCompact{
				BookID:       r.BookID,
				BookSchoolID: r.BookSchoolID,

				BookTitle:  r.BookTitle,
				BookAuthor: r.BookAuthor,
				BookDesc:   r.BookDesc,
				BookSlug:   r.BookSlug,

				BookImageURL:    r.BookImageURL,
				BookPurchaseURL: r.BookPurchaseURL,

				BookCreatedAt: r.BookCreatedAt,
				BookUpdatedAt: r.BookUpdatedAt,
				BookIsDeleted: r.BookIsDeleted,
			})
		}
		return helper.JsonList(c, "ok", out, pg)
	}

	// ===== mode full (behavior lama, raw rows) =====
	return helper.JsonList(c, "ok", rows, pg)
}
