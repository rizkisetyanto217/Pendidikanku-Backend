// internals/features/lembaga/classes/subjects/books/controller/class_subject_book_list_controller.go
package controller

import (
	"log"
	"strings"
	"time"

	csbDTO "masjidku_backend/internals/features/school/academics/books/dto"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
=========================================================
LIST (simple)
GET /admin/:masjid_id/class-subject-books
Query:
  - id / ids         : UUID atau comma-separated UUIDs
  - class_subject_id : UUID
  - book_id          : UUID
  - is_active        : bool
  - with_deleted     : bool
  - q                : cari di slug relasi & judul snapshot buku
  - sort             : created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
  - sort_by/order    : created_at|updated_at + asc|desc
  - limit/per_page, page/offset
  - include          : "book" (opsional; info dasar buku)

=========================================================
*/
func (h *ClassSubjectBookController) List(c *fiber.Ctx) error {
	// 🔐 Masjid context + DKM/Admin
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// Parse include
	includes := parseIncludeSet(strings.TrimSpace(c.Query("include")))

	// Pagination & sorting
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)
	if legacy := strings.ToLower(strings.TrimSpace(c.Query("sort"))); legacy != "" {
		switch legacy {
		case "created_at_asc":
			p.SortBy, p.SortOrder = "created_at", "asc"
		case "created_at_desc":
			p.SortBy, p.SortOrder = "created_at", "desc"
		case "updated_at_asc":
			p.SortBy, p.SortOrder = "updated_at", "asc"
		case "updated_at_desc":
			p.SortBy, p.SortOrder = "updated_at", "desc"
		}
	}
	allowedSort := map[string]string{
		"created_at": "csb.class_subject_book_created_at",
		"updated_at": "csb.class_subject_book_updated_at",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	/* ========== BASE QUERY (tenant-safe) ========== */
	qBase := h.DB.WithContext(c.Context()).
		Table("class_subject_books AS csb").
		Where("csb.class_subject_book_masjid_id = ?", masjidID)

	// Soft-delete filter default
	withDeleted := strings.EqualFold(strings.TrimSpace(c.Query("with_deleted")), "true")
	if !withDeleted {
		qBase = qBase.Where("csb.class_subject_book_deleted_at IS NULL")
	}

	/* ========== id / ids filter (strict) ========== */
	if err := applyIDsFilter(c, qBase); err != nil {
		return err
	}

	/* ========== FILTERS ========== */
	// class_subject_id
	if v := strings.TrimSpace(c.Query("class_subject_id")); v != "" {
		if id, er := uuid.Parse(v); er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_subject_id tidak valid")
		} else {
			qBase = qBase.Where("csb.class_subject_book_class_subject_id = ?", id)
		}
	}
	// book_id
	if v := strings.TrimSpace(c.Query("book_id")); v != "" {
		if id, er := uuid.Parse(v); er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "book_id tidak valid")
		} else {
			qBase = qBase.Where("csb.class_subject_book_book_id = ?", id)
		}
	}
	// is_active
	if v := strings.TrimSpace(c.Query("is_active")); v != "" {
		switch strings.ToLower(v) {
		case "true", "1":
			qBase = qBase.Where("csb.class_subject_book_is_active = TRUE")
		case "false", "0":
			qBase = qBase.Where("csb.class_subject_book_is_active = FALSE")
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "is_active tidak valid")
		}
	}
	// q: cari di slug & judul snapshot
	if v := strings.TrimSpace(c.Query("q")); v != "" {
		like := "%" + v + "%"
		qBase = qBase.Where(`
			(csb.class_subject_book_slug ILIKE ? OR
			 csb.class_subject_book_book_title_snapshot ILIKE ?)`,
			like, like,
		)
	}

	/* ========== OPTIONAL JOIN: book (include=book) ========== */
	if includes["book"] {
		qBase = qBase.Joins(`
			LEFT JOIN books AS b
			  ON b.book_id = csb.class_subject_book_book_id
			 AND b.book_masjid_id = csb.class_subject_book_masjid_id
		`)
	}

	/* ========== TOTAL DISTINCT ========== */
	var total int64
	if err := qBase.
		Session(&gorm.Session{}).
		Distinct("csb.class_subject_book_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	/* ========== SELECT ========== */
	selectCols := []string{
		"csb.class_subject_book_id",
		"csb.class_subject_book_masjid_id",
		"csb.class_subject_book_class_subject_id",
		"csb.class_subject_book_book_id",
		"csb.class_subject_book_slug",
		"csb.class_subject_book_is_active",
		"csb.class_subject_book_desc",
		"csb.class_subject_book_created_at",
		"csb.class_subject_book_updated_at",
		"csb.class_subject_book_deleted_at",

		// snapshots dari books (selalu ada di csb)
		"csb.class_subject_book_book_title_snapshot",
		"csb.class_subject_book_book_author_snapshot",
		"csb.class_subject_book_book_slug_snapshot",
		"csb.class_subject_book_book_publisher_snapshot",
		"csb.class_subject_book_book_publication_year_snapshot",
		"csb.class_subject_book_book_image_url_snapshot",
	}

	if includes["book"] {
		selectCols = append(selectCols,
			"b.book_id AS book_id",
			"b.book_masjid_id AS book_masjid_id",
			"b.book_title AS book_title",
			"b.book_author AS book_author",
			"b.book_slug AS book_slug",
			"b.book_image_url AS book_image_url",
			"b.book_publisher AS book_publisher",
			"b.book_publication_year AS book_publication_year",
		)
	}

	/* ========== SCAN ========== */
	type row struct {
		// csb
		ID             uuid.UUID  `gorm:"column:class_subject_book_id"`
		MasjidID       uuid.UUID  `gorm:"column:class_subject_book_masjid_id"`
		ClassSubjectID uuid.UUID  `gorm:"column:class_subject_book_class_subject_id"`
		BookID         uuid.UUID  `gorm:"column:class_subject_book_book_id"`
		Slug           *string    `gorm:"column:class_subject_book_slug"`
		IsActive       bool       `gorm:"column:class_subject_book_is_active"`
		Desc           *string    `gorm:"column:class_subject_book_desc"`
		CreatedAt      time.Time  `gorm:"column:class_subject_book_created_at"`
		UpdatedAt      time.Time  `gorm:"column:class_subject_book_updated_at"`
		DeletedAt      *time.Time `gorm:"column:class_subject_book_deleted_at"`

		// snapshots
		BookTitleSnap     *string `gorm:"column:class_subject_book_book_title_snapshot"`
		BookAuthorSnap    *string `gorm:"column:class_subject_book_book_author_snapshot"`
		BookSlugSnap      *string `gorm:"column:class_subject_book_book_slug_snapshot"`
		BookPublisherSnap *string `gorm:"column:class_subject_book_book_publisher_snapshot"`
		BookYearSnap      *int16  `gorm:"column:class_subject_book_book_publication_year_snapshot"`
		BookImageURLSnap  *string `gorm:"column:class_subject_book_book_image_url_snapshot"`

		// include book (opsional)
		BID        *uuid.UUID `gorm:"column:book_id"`
		BMasjidID  *uuid.UUID `gorm:"column:book_masjid_id"`
		BTitle     *string    `gorm:"column:book_title"`
		BAuthor    *string    `gorm:"column:book_author"`
		BSlug      *string    `gorm:"column:book_slug"`
		BImg       *string    `gorm:"column:book_image_url"`
		BPublisher *string    `gorm:"column:book_publisher"`
		BYear      *int16     `gorm:"column:book_publication_year"`
	}

	var rows []row
	if err := qBase.
		Select(strings.Join(selectCols, ",")).
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	/* ========== MAP → DTO ========== */
	items := make([]csbDTO.ClassSubjectBookResponse, 0, len(rows))
	for _, r := range rows {
		resp := csbDTO.ClassSubjectBookResponse{
			ClassSubjectBookID:             r.ID,
			ClassSubjectBookMasjidID:       r.MasjidID,
			ClassSubjectBookClassSubjectID: r.ClassSubjectID,
			ClassSubjectBookBookID:         r.BookID,
			ClassSubjectBookSlug:           r.Slug,
			ClassSubjectBookIsActive:       r.IsActive,
			ClassSubjectBookDesc:           r.Desc,
			ClassSubjectBookCreatedAt:      r.CreatedAt,
			ClassSubjectBookUpdatedAt:      r.UpdatedAt,
			ClassSubjectBookDeletedAt:      r.DeletedAt,

			// snapshots
			ClassSubjectBookBookTitleSnapshot:           r.BookTitleSnap,
			ClassSubjectBookBookAuthorSnapshot:          r.BookAuthorSnap,
			ClassSubjectBookBookSlugSnapshot:            r.BookSlugSnap,
			ClassSubjectBookBookPublisherSnapshot:       r.BookPublisherSnap,
			ClassSubjectBookBookPublicationYearSnapshot: r.BookYearSnap,
			ClassSubjectBookBookImageURLSnapshot:        r.BookImageURLSnap,
		}

		if includes["book"] && r.BID != nil {
			resp.Book = &csbDTO.BookLite{
				BookID:        *r.BID,
				BookMasjidID:  derefUUID(r.BMasjidID),
				BookTitle:     derefString(r.BTitle),
				BookAuthor:    r.BAuthor,
				BookSlug:      r.BSlug,
				BookImageURL:  r.BImg,
				BookPublisher: r.BPublisher,
				BookYear:      r.BYear,
			}
		}

		items = append(items, resp)
	}

	/* ========== RESPON ========== */
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}

/* ================= Helpers (local) ================= */

func parseIncludeSet(s string) map[string]bool {
	out := map[string]bool{}
	if s == "" {
		return out
	}
	for _, p := range strings.Split(s, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out[p] = true
		}
	}
	return out
}

func applyIDsFilter(c *fiber.Ctx, q *gorm.DB) error {
	rawID := strings.TrimSpace(c.Query("id"))
	rawIDs := strings.TrimSpace(c.Query("ids"))
	if rawID == "" && rawIDs == "" {
		return nil
	}
	parts := make([]string, 0, 1)
	if rawID != "" {
		parts = append(parts, rawID)
	}
	if rawIDs != "" {
		for _, s := range strings.Split(rawIDs, ",") {
			if ss := strings.TrimSpace(s); ss != "" {
				parts = append(parts, ss)
			}
		}
	}
	seen := make(map[uuid.UUID]struct{}, len(parts))
	ids := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		u, err := uuid.Parse(p)
		if err != nil {
			log.Printf("[CSB.List] id/ids INVALID → %q", p)
			return helper.JsonError(c, fiber.StatusBadRequest, "id/ids tidak valid (harus UUID, dipisah koma)")
		}
		if _, ok := seen[u]; !ok {
			seen[u] = struct{}{}
			ids = append(ids, u)
		}
	}
	if len(ids) == 0 {
		q.Where("1=0") // no results
	} else {
		q.Where("csb.class_subject_book_id IN ?", ids)
	}
	return nil
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
func derefUUID(p *uuid.UUID) uuid.UUID {
	if p == nil {
		return uuid.Nil
	}
	return *p
}
