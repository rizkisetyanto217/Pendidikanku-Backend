// file: internals/features/lembaga/classes/subjects/books/controller/class_subject_book_list_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	csbDTO "schoolku_backend/internals/features/school/academics/books/dto"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
=========================================================
LIST (simple, pakai DTO, tanpa join)

Contoh route (admin / DKM-only):

  - GET /api/a/:school_id/class-subject-books/list
  - GET /api/a/m/:school_slug/class-subject-books/list
  - atau versi token-scope-only kalau dipasang di group yang pakai UseSchoolScope

Resolver school:

1) Kalau ada token & active_school → pakai school dari token.
2) Kalau tidak ada / gagal → pakai ResolveSchoolContext (ID atau slug).
3) Kalau tetap tidak ada → ErrSchoolContextMissing.

Query:
  - id / ids         : UUID atau comma-separated UUIDs
  - class_subject_id : UUID
  - book_id          : UUID
  - is_active        : bool
  - is_primary       : bool
  - is_required      : bool
  - with_deleted     : bool
  - q                : cari di slug relasi, judul buku snapshot, nama/slug subject snapshot
  - sort (legacy)    : created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
  - sort_by/order    : created_at|updated_at + asc|desc
  - limit/per_page, page/offset

=========================================================
*/
func (h *ClassSubjectBookController) List(c *fiber.Ctx) error {
	// DB ke locals supaya helper yang butuh DB via context tetap jalan
	c.Locals("DB", h.DB)

	// ===== Resolve school_id (token-aware + fallback slug/ID) =====
	var schoolID uuid.UUID

	// 1) Coba dari token dulu (active_school)
	if id, err := helperAuth.GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		// 2) Fallback: ResolveSchoolContext (PUBLIC-style, pakai ID / slug di path)
		mc, err2 := helperAuth.ResolveSchoolContext(c)
		if err2 != nil {
			// bisa ErrSchoolContextMissing atau fiber.Error lain
			return err2
		}

		switch {
		case mc.ID != uuid.Nil:
			// Sudah dapat ID langsung
			schoolID = mc.ID

		case strings.TrimSpace(mc.Slug) != "":
			// mc.Slug bisa berisi UUID atau slug beneran
			s := strings.TrimSpace(mc.Slug)
			if id2, errParse := uuid.Parse(s); errParse == nil {
				// Ternyata UUID → pakai langsung
				schoolID = id2
			} else {
				// Beneran slug → resolve dari DB
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
			// Tidak ada ID, tidak ada slug → context kurang
			return helperAuth.ErrSchoolContextMissing
		}
	}

	// 🔐 DKM/Admin only untuk school ini
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	// ===== Parse query ke DTO (toleran) =====
	var q csbDTO.ListClassSubjectBookQuery
	_ = c.QueryParser(&q)

	// ===== Paging (jsonresponse helper) =====
	p := helper.ResolvePaging(c, 20, 100) // default 20, max 100

	// ===== Sorting whitelist (manual) =====
	// Legacy 'sort' tetap didukung
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by", "created_at")))
	order := strings.ToLower(strings.TrimSpace(c.Query("order", "desc")))
	if s := strings.ToLower(strings.TrimSpace(c.Query("sort"))); s != "" {
		switch s {
		case "created_at_asc":
			sortBy, order = "created_at", "asc"
		case "created_at_desc":
			sortBy, order = "created_at", "desc"
		case "updated_at_asc":
			sortBy, order = "updated_at", "asc"
		case "updated_at_desc":
			sortBy, order = "updated_at", "desc"
		}
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}
	colMap := map[string]string{
		"created_at": "csb.class_subject_book_created_at",
		"updated_at": "csb.class_subject_book_updated_at",
	}
	col, ok := colMap[sortBy]
	if !ok {
		col = colMap["created_at"]
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// ===== Base query (tenant-safe, no join) =====
	qBase := h.DB.WithContext(c.Context()).
		Table("class_subject_books AS csb").
		Where("csb.class_subject_book_school_id = ?", schoolID)

	// Soft delete
	withDeleted := (q.WithDeleted != nil && *q.WithDeleted) ||
		strings.EqualFold(strings.TrimSpace(c.Query("with_deleted")), "true")
	if !withDeleted {
		qBase = qBase.Where("csb.class_subject_book_deleted_at IS NULL")
	}

	// ===== Filters =====
	// id / ids
	var err error
	if qBase, err = applyIDsFilter(c, qBase); err != nil {
		return err
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

	// is_primary
	if q.IsPrimary != nil {
		if *q.IsPrimary {
			qBase = qBase.Where("csb.class_subject_book_is_primary = TRUE")
		} else {
			qBase = qBase.Where("csb.class_subject_book_is_primary = FALSE")
		}
	} else if v := strings.ToLower(strings.TrimSpace(c.Query("is_primary"))); v != "" {
		switch v {
		case "true", "1":
			qBase = qBase.Where("csb.class_subject_book_is_primary = TRUE")
		case "false", "0":
			qBase = qBase.Where("csb.class_subject_book_is_primary = FALSE")
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "is_primary tidak valid")
		}
	}

	// is_required
	if q.IsRequired != nil {
		if *q.IsRequired {
			qBase = qBase.Where("csb.class_subject_book_is_required = TRUE")
		} else {
			qBase = qBase.Where("csb.class_subject_book_is_required = FALSE")
		}
	} else if v := strings.ToLower(strings.TrimSpace(c.Query("is_required"))); v != "" {
		switch v {
		case "true", "1":
			qBase = qBase.Where("csb.class_subject_book_is_required = TRUE")
		case "false", "0":
			qBase = qBase.Where("csb.class_subject_book_is_required = FALSE")
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "is_required tidak valid")
		}
	}

	// q: cari di slug relasi & snapshots
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		needle := "%" + strings.TrimSpace(*q.Q) + "%"
		qBase = qBase.Where(`
			(csb.class_subject_book_slug ILIKE ? OR
			 csb.class_subject_book_book_title_snapshot ILIKE ? OR
			 csb.class_subject_book_subject_name_snapshot ILIKE ? OR
			 csb.class_subject_book_subject_slug_snapshot ILIKE ?)`,
			needle, needle, needle, needle)
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
		"csb.class_subject_book_school_id",
		"csb.class_subject_book_class_subject_id",
		"csb.class_subject_book_book_id",
		"csb.class_subject_book_slug",
		"csb.class_subject_book_is_primary",
		"csb.class_subject_book_is_required",
		"csb.class_subject_book_order",
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
		"csb.class_subject_book_subject_id",
		"csb.class_subject_book_subject_code_snapshot",
		"csb.class_subject_book_subject_name_snapshot",
		"csb.class_subject_book_subject_slug_snapshot",
	}

	type row struct {
		ClassSubjectBookID             uuid.UUID  `gorm:"column:class_subject_book_id"`
		ClassSubjectBookSchoolID       uuid.UUID  `gorm:"column:class_subject_book_school_id"`
		ClassSubjectBookClassSubjectID uuid.UUID  `gorm:"column:class_subject_book_class_subject_id"`
		ClassSubjectBookBookID         uuid.UUID  `gorm:"column:class_subject_book_book_id"`
		ClassSubjectBookSlug           *string    `gorm:"column:class_subject_book_slug"`
		ClassSubjectBookIsPrimary      bool       `gorm:"column:class_subject_book_is_primary"`
		ClassSubjectBookIsRequired     bool       `gorm:"column:class_subject_book_is_required"`
		ClassSubjectBookOrder          *int       `gorm:"column:class_subject_book_order"`
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
		ClassSubjectBookSubjectID           *uuid.UUID `gorm:"column:class_subject_book_subject_id"`
		ClassSubjectBookSubjectCodeSnapshot *string    `gorm:"column:class_subject_book_subject_code_snapshot"`
		ClassSubjectBookSubjectNameSnapshot *string    `gorm:"column:class_subject_book_subject_name_snapshot"`
		ClassSubjectBookSubjectSlugSnapshot *string    `gorm:"column:class_subject_book_subject_slug_snapshot"`
	}

	var rows []row
	if err := qBase.
		Select(strings.Join(selectCols, ",")).
		Order(orderExpr).
		Limit(p.Limit).
		Offset(p.Offset).
		Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Map → DTO
	items := make([]csbDTO.ClassSubjectBookResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, csbDTO.ClassSubjectBookResponse{
			ClassSubjectBookID:             r.ClassSubjectBookID,
			ClassSubjectBookSchoolID:       r.ClassSubjectBookSchoolID,
			ClassSubjectBookClassSubjectID: r.ClassSubjectBookClassSubjectID,
			ClassSubjectBookBookID:         r.ClassSubjectBookBookID,

			ClassSubjectBookSlug:       r.ClassSubjectBookSlug,
			ClassSubjectBookIsPrimary:  r.ClassSubjectBookIsPrimary,
			ClassSubjectBookIsRequired: r.ClassSubjectBookIsRequired,
			ClassSubjectBookOrder:      r.ClassSubjectBookOrder,
			ClassSubjectBookIsActive:   r.ClassSubjectBookIsActive,
			ClassSubjectBookDesc:       r.ClassSubjectBookDesc,

			ClassSubjectBookBookTitleSnapshot:           r.ClassSubjectBookBookTitleSnapshot,
			ClassSubjectBookBookAuthorSnapshot:          r.ClassSubjectBookBookAuthorSnapshot,
			ClassSubjectBookBookSlugSnapshot:            r.ClassSubjectBookBookSlugSnapshot,
			ClassSubjectBookBookPublisherSnapshot:       r.ClassSubjectBookBookPublisherSnapshot,
			ClassSubjectBookBookPublicationYearSnapshot: r.ClassSubjectBookBookPublicationYearSnapshot,
			ClassSubjectBookBookImageURLSnapshot:        r.ClassSubjectBookBookImageURLSnapshot,

			ClassSubjectBookSubjectID:           r.ClassSubjectBookSubjectID,
			ClassSubjectBookSubjectCodeSnapshot: r.ClassSubjectBookSubjectCodeSnapshot,
			ClassSubjectBookSubjectNameSnapshot: r.ClassSubjectBookSubjectNameSnapshot,
			ClassSubjectBookSubjectSlugSnapshot: r.ClassSubjectBookSubjectSlugSnapshot,

			ClassSubjectBookCreatedAt: r.ClassSubjectBookCreatedAt,
			ClassSubjectBookUpdatedAt: r.ClassSubjectBookUpdatedAt,
			ClassSubjectBookDeletedAt: r.ClassSubjectBookDeletedAt,
		})
	}

	// ===== Pagination meta (jsonresponse helper) =====
	pg := helper.BuildPaginationFromOffset(total, p.Offset, p.Limit)

	// ===== Response (JsonList standar) =====
	return helper.JsonList(c, "ok", items, pg)
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
