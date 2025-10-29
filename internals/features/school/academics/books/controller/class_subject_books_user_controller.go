// file: internals/features/lembaga/classes/subjects/books/controller/class_subject_book_list_controller.go
package controller

import (
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
LIST (simple, pakai DTO, tanpa join)
GET /admin/:masjid_id/class-subject-books

Query:
  - id / ids         : UUID atau comma-separated UUIDs
  - class_subject_id : UUID
  - book_id          : UUID
  - is_active        : bool
  - with_deleted     : bool
  - q                : cari di slug relasi, judul buku snapshot, nama/slug subject snapshot
  - sort (legacy)    : created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
  - sort_by/order    : created_at|updated_at + asc|desc
  - limit/per_page, page/offset

=========================================================
*/
func (h *ClassSubjectBookController) List(c *fiber.Ctx) error {
	// 🔐 Masjid scope + DKM/Admin
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// ===== Parse query ke DTO =====
	var q csbDTO.ListClassSubjectBookQuery
	_ = c.QueryParser(&q) // toleran: bila gagal tetap lanjut

	// Pagination & sorting (default created_at desc)
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)
	// Legacy sort (kompatibilitas)
	if s := strings.ToLower(strings.TrimSpace(c.Query("sort"))); s != "" {
		switch s {
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

	// ===== Base query (tenant-safe, no join) =====
	qBase := h.DB.WithContext(c.Context()).
		Table("class_subject_books AS csb").
		Where("csb.class_subject_book_masjid_id = ?", masjidID)

	// Soft delete
	withDeleted := (q.WithDeleted != nil && *q.WithDeleted) ||
		strings.EqualFold(strings.TrimSpace(c.Query("with_deleted")), "true")
	if !withDeleted {
		qBase = qBase.Where("csb.class_subject_book_deleted_at IS NULL")
	}

	// ===== Filters =====
	// id / ids
	var e error
	if qBase, e = applyIDsFilter(c, qBase); e != nil {
		return e
	}
	// class_subject_id
	if q.ClassSubjectID != nil {
		qBase = qBase.Where("csb.class_subject_book_class_subject_id = ?", *q.ClassSubjectID)
	} else if v := strings.TrimSpace(c.Query("class_subject_id")); v != "" {
		if id, er := uuid.Parse(v); er == nil {
			qBase = qBase.Where("csb.class_subject_book_class_subject_id = ?", id)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_subject_id tidak valid")
		}
	}
	// book_id
	if q.BookID != nil {
		qBase = qBase.Where("csb.class_subject_book_book_id = ?", *q.BookID)
	} else if v := strings.TrimSpace(c.Query("book_id")); v != "" {
		if id, er := uuid.Parse(v); er == nil {
			qBase = qBase.Where("csb.class_subject_book_book_id = ?", id)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "book_id tidak valid")
		}
	}
	// is_active
	if q.IsActive != nil {
		if *q.IsActive {
			qBase = qBase.Where("csb.class_subject_book_is_active = TRUE")
		} else {
			qBase = qBase.Where("csb.class_subject_book_is_active = FALSE")
		}
	} else if v := strings.ToLower(strings.TrimSpace(c.Query("is_active"))); v != "" {
		switch v {
		case "true", "1":
			qBase = qBase.Where("csb.class_subject_book_is_active = TRUE")
		case "false", "0":
			qBase = qBase.Where("csb.class_subject_book_is_active = FALSE")
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "is_active tidak valid")
		}
	}

	// q: cari di slug relasi, judul buku snapshot, nama & slug subject snapshot
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		like := "%" + strings.TrimSpace(*q.Q) + "%"
		qBase = qBase.Where(`
			(csb.class_subject_book_slug ILIKE ? OR
			 csb.class_subject_book_book_title_snapshot ILIKE ? OR
			 csb.class_subject_book_subject_name_snapshot ILIKE ? OR
			 csb.class_subject_book_subject_slug_snapshot ILIKE ?)`,
			like, like, like, like)
	}

	// ===== Hitung total distinct =====
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("csb.class_subject_book_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Select & scan ke ROW lalu map → DTO =====
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

		// snapshots BOOK (inline di csb)
		"csb.class_subject_book_book_title_snapshot",
		"csb.class_subject_book_book_author_snapshot",
		"csb.class_subject_book_book_slug_snapshot",
		"csb.class_subject_book_book_publisher_snapshot",
		"csb.class_subject_book_book_publication_year_snapshot",
		"csb.class_subject_book_book_image_url_snapshot",

		// snapshots SUBJECT (inline di csb)
		"csb.class_subject_book_subject_id_snapshot",
		"csb.class_subject_book_subject_code_snapshot",
		"csb.class_subject_book_subject_name_snapshot",
		"csb.class_subject_book_subject_slug_snapshot",
	}

	type row struct {
		ClassSubjectBookID             uuid.UUID  `gorm:"column:class_subject_book_id"`
		ClassSubjectBookMasjidID       uuid.UUID  `gorm:"column:class_subject_book_masjid_id"`
		ClassSubjectBookClassSubjectID uuid.UUID  `gorm:"column:class_subject_book_class_subject_id"`
		ClassSubjectBookBookID         uuid.UUID  `gorm:"column:class_subject_book_book_id"`
		ClassSubjectBookSlug           *string    `gorm:"column:class_subject_book_slug"`
		ClassSubjectBookIsActive       bool       `gorm:"column:class_subject_book_is_active"`
		ClassSubjectBookDesc           *string    `gorm:"column:class_subject_book_desc"`
		ClassSubjectBookCreatedAt      time.Time  `gorm:"column:class_subject_book_created_at"`
		ClassSubjectBookUpdatedAt      time.Time  `gorm:"column:class_subject_book_updated_at"`
		ClassSubjectBookDeletedAt      *time.Time `gorm:"column:class_subject_book_deleted_at"`

		// BOOK snapshots
		ClassSubjectBookBookTitleSnapshot           *string `gorm:"column:class_subject_book_book_title_snapshot"`
		ClassSubjectBookBookAuthorSnapshot          *string `gorm:"column:class_subject_book_book_author_snapshot"`
		ClassSubjectBookBookSlugSnapshot            *string `gorm:"column:class_subject_book_book_slug_snapshot"`
		ClassSubjectBookBookPublisherSnapshot       *string `gorm:"column:class_subject_book_book_publisher_snapshot"`
		ClassSubjectBookBookPublicationYearSnapshot *int16  `gorm:"column:class_subject_book_book_publication_year_snapshot"`
		ClassSubjectBookBookImageURLSnapshot        *string `gorm:"column:class_subject_book_book_image_url_snapshot"`

		// SUBJECT snapshots
		ClassSubjectBookSubjectIDSnapshot   *uuid.UUID `gorm:"column:class_subject_book_subject_id_snapshot"`
		ClassSubjectBookSubjectCodeSnapshot *string    `gorm:"column:class_subject_book_subject_code_snapshot"`
		ClassSubjectBookSubjectNameSnapshot *string    `gorm:"column:class_subject_book_subject_name_snapshot"`
		ClassSubjectBookSubjectSlugSnapshot *string    `gorm:"column:class_subject_book_subject_slug_snapshot"`
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

	// Map → DTO
	items := make([]csbDTO.ClassSubjectBookResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, csbDTO.ClassSubjectBookResponse{
			ClassSubjectBookID:             r.ClassSubjectBookID,
			ClassSubjectBookMasjidID:       r.ClassSubjectBookMasjidID,
			ClassSubjectBookClassSubjectID: r.ClassSubjectBookClassSubjectID,
			ClassSubjectBookBookID:         r.ClassSubjectBookBookID,
			ClassSubjectBookSlug:           r.ClassSubjectBookSlug,
			ClassSubjectBookIsActive:       r.ClassSubjectBookIsActive,
			ClassSubjectBookDesc:           r.ClassSubjectBookDesc,
			ClassSubjectBookCreatedAt:      r.ClassSubjectBookCreatedAt,
			ClassSubjectBookUpdatedAt:      r.ClassSubjectBookUpdatedAt,
			ClassSubjectBookDeletedAt:      r.ClassSubjectBookDeletedAt,

			// BOOK snapshots
			ClassSubjectBookBookTitleSnapshot:           r.ClassSubjectBookBookTitleSnapshot,
			ClassSubjectBookBookAuthorSnapshot:          r.ClassSubjectBookBookAuthorSnapshot,
			ClassSubjectBookBookSlugSnapshot:            r.ClassSubjectBookBookSlugSnapshot,
			ClassSubjectBookBookPublisherSnapshot:       r.ClassSubjectBookBookPublisherSnapshot,
			ClassSubjectBookBookPublicationYearSnapshot: r.ClassSubjectBookBookPublicationYearSnapshot,
			ClassSubjectBookBookImageURLSnapshot:        r.ClassSubjectBookBookImageURLSnapshot,

			// SUBJECT snapshots
			ClassSubjectBookSubjectIDSnapshot:   r.ClassSubjectBookSubjectIDSnapshot,
			ClassSubjectBookSubjectCodeSnapshot: r.ClassSubjectBookSubjectCodeSnapshot,
			ClassSubjectBookSubjectNameSnapshot: r.ClassSubjectBookSubjectNameSnapshot,
			ClassSubjectBookSubjectSlugSnapshot: r.ClassSubjectBookSubjectSlugSnapshot,
		})
	}

	// ===== Response (pakai helper meta standar) =====
	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, items, meta)
}

/* ================= Helpers (local) ================= */

// kembalikan *gorm.DB yang sudah difilter + error
func applyIDsFilter(c *fiber.Ctx, q *gorm.DB) (*gorm.DB, error) {
	rawID := strings.TrimSpace(c.Query("id"))
	rawIDs := strings.TrimSpace(c.Query("ids"))
	if rawID == "" && rawIDs == "" {
		return q, nil
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
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "id/ids tidak valid (UUID, pisah koma)")
		}
		if _, ok := seen[u]; !ok {
			seen[u] = struct{}{}
			ids = append(ids, u)
		}
	}
	if len(ids) == 0 {
		return q.Where("1=0"), nil
	}
	return q.Where("csb.class_subject_book_id IN ?", ids), nil
}
