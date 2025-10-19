// file: internals/features/lembaga/class_books/controller/books_controller.go
package controller

import (
	"errors"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
GET /api/a/books/list (versi sederhana)
- Filter: q (title/author/desc, ILIKE), author, id/book_id (CSV UUID), with_deleted
- Sort: order_by=created_at|title|author + sort=asc|desc (whitelist)
- Pagination: pakai helper.ParseFiber + helper.BuildMeta
- Tanpa DTO eksternal (struct lokal) & tanpa preload/joins
*/
func (h *BooksController) List(c *fiber.Ctx) error {
	// ===== Masjid context (PUBLIC): no role check =====
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}

	var masjidID uuid.UUID
	switch {
	case mc.ID != uuid.Nil:
		masjidID = mc.ID
	case strings.TrimSpace(mc.Slug) != "":
		id, er := helperAuth.GetMasjidIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve masjid dari slug")
		}
		masjidID = id
	default:
		return helperAuth.ErrMasjidContextMissing
	}

	// ===== Query params dasar =====
	q := strings.TrimSpace(c.Query("q"))
	author := strings.TrimSpace(c.Query("author"))
	withDeleted := strings.EqualFold(strings.TrimSpace(c.Query("with_deleted")), "true")

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
		BookMasjidID       uuid.UUID  `json:"book_masjid_id"        gorm:"column:book_masjid_id"`
		BookTitle          string     `json:"book_title"            gorm:"column:book_title"`
		BookAuthor         *string    `json:"book_author,omitempty" gorm:"column:book_author"`
		BookDesc           *string    `json:"book_desc,omitempty"   gorm:"column:book_desc"`
		BookSlug           *string    `json:"book_slug,omitempty"   gorm:"column:book_slug"`
		BookImageURL       *string    `json:"book_image_url,omitempty"        gorm:"column:book_image_url"`
		BookImageObjectKey *string    `json:"book_image_object_key,omitempty" gorm:"column:book_image_object_key"`
		BookCreatedAt      time.Time  `json:"book_created_at"       gorm:"column:book_created_at"`
		BookUpdatedAt      time.Time  `json:"book_updated_at"       gorm:"column:book_updated_at"`
		BookDeletedAt      *time.Time `json:"-"                     gorm:"column:book_deleted_at"`
		BookIsDeleted      bool       `json:"book_is_deleted"       gorm:"-"`
	}

	base := h.DB.Table("books AS b").
		Where("b.book_masjid_id = ?", masjidID)

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
	if author != "" {
		base = base.Where("b.book_author ILIKE ?", author)
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
			b.book_masjid_id,
			b.book_title,
			b.book_author,
			b.book_desc,
			b.book_slug,
			b.book_image_url,
			b.book_image_object_key,
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

	// ===== Response =====
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, rows, meta)
}
