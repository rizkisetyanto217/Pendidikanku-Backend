// internals/features/lembaga/classes/subjects/books/controller/class_subject_book_list_controller.go
package controller

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	csbDTO "masjidku_backend/internals/features/school/subject_books/books/dto"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
   LIST
   GET /admin/class-subject-books
   Query:
     - id / ids         : UUID atau comma-separated UUIDs (filter by-id)
     - q                : cari di desc
     - class_subject_id : UUID
     - class_id         : UUID (via class_sections -> class_id)
     - section_id       : UUID
     - subject_id       : UUID (via class_subjects)
     - teacher_id       : UUID (filter via CSST)
     - is_active        : bool
     - with_deleted     : bool
     - sort             : created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
     - limit (<=200), offset
     - include          : CSV → book,section,book_urls,book_cover,book_url_primary
   ========================================================= */
func (h *ClassSubjectBookController) List(c *fiber.Ctx) error {
	// 📝 Log request & param kunci
	log.Printf(
		"[CSB.List] route=%s method=%s url=%s id=%q ids=%q include=%q",
		func() string { if r := c.Route(); r != nil { return r.Path }; return "?" }(),
		c.Method(),
		c.OriginalURL(),
		strings.TrimSpace(c.Query("id")),
		strings.TrimSpace(c.Query("ids")),
		strings.TrimSpace(c.Query("include")),
	)

	/* ========== TENANT (semua klaim) ========== */
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	/* ========== PARSE QUERY SEDERHANA ========== */
	var q csbDTO.ListClassSubjectBookQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// Parse includes
	includes := parseIncludeSet(strings.TrimSpace(c.Query("include")))

	// Pagination guard
	limit := 20
	offset := 0
	if v := intFromPtr(q.Limit); v > 0 && v <= 200 { limit = v }
	if v := intFromPtr(q.Offset); v >= 0 { offset = v }

	/* ========== BASE QUERY ========== */
	qBase := h.DB.Table("class_subject_books AS csb").
		Where("csb.class_subject_books_masjid_id IN ?", masjidIDs)

	// Soft-delete aware (default exclude)
	if !(q.WithDeleted != nil && *q.WithDeleted) {
		qBase = qBase.Where("csb.class_subject_books_deleted_at IS NULL")
	}

	/* ========== STRICT VALIDATION: id/ids ========== */
	rawID := strings.TrimSpace(c.Query("id"))
	rawIDs := strings.TrimSpace(c.Query("ids"))
	if rawID != "" || rawIDs != "" {
		parts := make([]string, 0, 1)
		if rawID != "" { parts = append(parts, rawID) }
		if rawIDs != "" {
			for _, s := range strings.Split(rawIDs, ",") {
				if ss := strings.TrimSpace(s); ss != "" { parts = append(parts, ss) }
			}
		}
		seen := make(map[uuid.UUID]struct{}, len(parts))
		ids := make([]uuid.UUID, 0, len(parts))
		for _, p := range parts {
			u, err := uuid.Parse(p)
			if err != nil {
				log.Printf("[CSB.List] id/ids INVALID → 400 (bad=%q all=%v)", p, parts)
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "id/ids tidak valid (harus UUID, comma-separated)",
				})
			}
			if _, ok := seen[u]; !ok {
				seen[u] = struct{}{}
				ids = append(ids, u)
			}
		}
		if len(ids) == 0 {
			qBase = qBase.Where("1=0")
		} else {
			qBase = qBase.Where("csb.class_subject_books_id IN ?", ids)
		}
	}

	/* ========== FILTERS ========== */
	// Flag JOIN hanya jika butuh (untuk filter/include)
	needJoinCS := false
	needJoinSec := false
	needJoinCSST := false
	needJoinBooks := false

	if q.ClassSubjectID != nil {
		qBase = qBase.Where("csb.class_subject_books_class_subject_id = ?", *q.ClassSubjectID)
		needJoinCS = true
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

	// section_id
	if secID, ok, errResp := uuidFromQuery(c, "section_id", "section_id tidak valid"); errResp != nil {
		return errResp
	} else if ok {
		needJoinCS = true
		needJoinSec = true
		qBase = qBase.Where("sec.class_sections_id = ?", *secID)
	}
	// class_id
	if classID, ok, errResp := uuidFromQuery(c, "class_id", "class_id tidak valid"); errResp != nil {
		return errResp
	} else if ok {
		needJoinCS = true
		needJoinSec = true
		qBase = qBase.Where("sec.class_sections_class_id = ?", *classID)
	}
	// subject_id
	if subID, ok, errResp := uuidFromQuery(c, "subject_id", "subject_id tidak valid"); errResp != nil {
		return errResp
	} else if ok {
		needJoinCS = true
		qBase = qBase.Where("cs.class_subjects_subject_id = ?", *subID)
	}
	// teacher_id (via CSST)
	if teacherID, ok, errResp := uuidFromQuery(c, "teacher_id", "teacher_id tidak valid"); errResp != nil {
		return errResp
	} else if ok {
		needJoinCS = true
		needJoinSec = true
		needJoinCSST = true
		qBase = qBase.Where("csst.class_section_subject_teachers_teacher_id = ?", *teacherID)
	}

	/* ========== OPTIONAL JOINS (untuk include / filter) ========== */
	// CS join
	if needJoinCS || includes["section"] || needJoinSec || needJoinCSST {
		qBase = qBase.Joins(`
			LEFT JOIN class_subjects AS cs
			  ON cs.class_subjects_id = csb.class_subject_books_class_subject_id
		`)
	}
	// SEC join
	if needJoinSec || includes["section"] || needJoinCSST {
		qBase = qBase.Joins(`
			LEFT JOIN class_sections AS sec
			  ON sec.class_sections_class_id = cs.class_subjects_class_id
			 AND sec.class_sections_deleted_at IS NULL
		`)
	}
	// CSST join
	if needJoinCSST {
		qBase = qBase.Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_section_id = sec.class_sections_id
			 AND csst.class_section_subject_teachers_class_subjects_id = cs.class_subjects_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`)
	}
	// Books join
	if includes["book"] || includes["book_urls"] || includes["book_cover"] || includes["book_url_primary"] {
		needJoinBooks = true
		qBase = qBase.Joins(`
			LEFT JOIN books AS b
			  ON b.books_id = csb.class_subject_books_book_id
		`)
	}

	/* ========== TOTAL DISTINCT ========== */
	var total int64
	if err := qBase.Session(&gorm.Session{}).
		Distinct("csb.class_subject_books_id").
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	/* ========== SORTING WHITELIST ========== */
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

	/* ========== SELECT builder + LATERAL opsional ========== */
	selectCols := []string{
		"csb.class_subject_books_id",
		"csb.class_subject_books_masjid_id",
		"csb.class_subject_books_class_subject_id",
		"csb.class_subject_books_book_id",
		"csb.class_subject_books_is_active",
		"csb.class_subject_books_desc",
		"csb.class_subject_books_created_at",
		"csb.class_subject_books_updated_at",
		"csb.class_subject_books_deleted_at",
	}

	// Include book fields
	if includes["book"] {
		selectCols = append(selectCols,
			"b.books_id",
			"b.books_masjid_id",
			"b.books_title",
			"b.books_author",
			"b.books_slug",
		)
	}

	// LATERAL: URL primary
	if needJoinBooks && includes["book_url_primary"] {
		qBase = qBase.Joins(`
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
		selectCols = append(selectCols, "bu.book_url_href AS books_url")
	}

	// LATERAL: cover
	if needJoinBooks && includes["book_cover"] {
		qBase = qBase.Joins(`
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
		selectCols = append(selectCols, "bu_cover.book_url_href AS books_image_url")
	}

	// LATERAL: SEMUA URL (JSON array)
	if needJoinBooks && includes["book_urls"] {
		qBase = qBase.Joins(`
			LEFT JOIN LATERAL (
				SELECT COALESCE(
					json_agg(
						json_build_object(
							'book_url_id', bu_all.book_url_id,
							'book_url_masjid_id', bu_all.book_url_masjid_id,
							'book_url_book_id', bu_all.book_url_book_id,
							'book_url_label', bu_all.book_url_label,
							'book_url_type', bu_all.book_url_type,
							'book_url_href', bu_all.book_url_href,
							'book_url_trash_url', bu_all.book_url_trash_url,
							'book_url_delete_pending_until', bu_all.book_url_delete_pending_until,
							'book_url_created_at', bu_all.book_url_created_at,
							'book_url_updated_at', bu_all.book_url_updated_at,
							'book_url_deleted_at', bu_all.book_url_deleted_at
						)
						ORDER BY bu_all.book_url_created_at DESC
					),
					'[]'::json
				) AS book_urls_json
				FROM book_urls bu_all
				WHERE bu_all.book_url_book_id = b.books_id
				  AND bu_all.book_url_deleted_at IS NULL
			) bu_all ON TRUE
		`)
		selectCols = append(selectCols, "bu_all.book_urls_json AS book_urls_json")
	}

	/* ========== SCAN ========== */
	type row struct {
		// csb
		ID             uuid.UUID  `gorm:"column:class_subject_books_id"`
		MasjidID       uuid.UUID  `gorm:"column:class_subject_books_masjid_id"`
		ClassSubjectID uuid.UUID  `gorm:"column:class_subject_books_class_subject_id"`
		BookID         uuid.UUID  `gorm:"column:class_subject_books_book_id"`
		IsActive       bool       `gorm:"column:class_subject_books_is_active"`
		Desc           *string    `gorm:"column:class_subject_books_desc"`
		CreatedAt      time.Time  `gorm:"column:class_subject_books_created_at"`
		UpdatedAt      *time.Time `gorm:"column:class_subject_books_updated_at"`
		DeletedAt      *time.Time `gorm:"column:class_subject_books_deleted_at"`
		// book (opsional)
		BID       *uuid.UUID `gorm:"column:books_id"`
		BMasjidID *uuid.UUID `gorm:"column:books_masjid_id"`
		BTitle    *string    `gorm:"column:books_title"`
		BAuthor   *string    `gorm:"column:books_author"`
		BURL      *string    `gorm:"column:books_url"`        // primary
		BImageURL *string    `gorm:"column:books_image_url"`  // cover
		BSlug     *string    `gorm:"column:books_slug"`
		// urls (opsional, JSON)
		BookURLsJSON *string `gorm:"column:book_urls_json"`
		// section (opsional)
		SecID       *uuid.UUID `gorm:"column:class_sections_id"`
		SecName     *string    `gorm:"column:class_sections_name"`
		SecSlug     *string    `gorm:"column:class_sections_slug"`
		SecCode     *string    `gorm:"column:class_sections_code"`
		SecCapacity *int       `gorm:"column:class_sections_capacity"`
		SecActive   *bool      `gorm:"column:class_sections_is_active"`
	}

	var rows []row
	if err := qBase.
		Select(strings.Join(selectCols, ",")).
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	/* ========== MAP KE DTO ========== */
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

		// book only if included and available
		if includes["book"] && r.BID != nil {
			resp.Book = &csbDTO.BookLite{
				BooksID:       *r.BID,
				BooksMasjidID: derefUUID(r.BMasjidID),
				BooksTitle:    derefString(r.BTitle),
				BooksAuthor:   r.BAuthor,
				BooksURL:      r.BURL,      // jika include book_url_primary
				BooksImageURL: r.BImageURL, // jika include book_cover
				BooksSlug:     r.BSlug,
			}

			// Inject semua URL jika diminta
			if includes["book_urls"] && r.BookURLsJSON != nil {
				var urls []csbDTO.BookURLLite
				if err := json.Unmarshal([]byte(*r.BookURLsJSON), &urls); err != nil {
					log.Printf("[CSB.List] WARN: gagal unmarshal book_urls_json: %v", err)
				} else {
					resp.Book.BookURLs = urls
				}
			}
		}

		// section only if included and available
		if includes["section"] && r.SecID != nil {
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

	/* ========== RESPONSE LIST ========== */
	return helper.JsonList(c, items, csbDTO.Pagination{
		Limit:  limit,
		Offset: offset,
		Total:  int(total),
	})
}

/* ================= Helpers (local) ================= */

// parseIncludeSet: "a,b,c" -> map["a"]=true, ...
func parseIncludeSet(s string) map[string]bool {
	out := map[string]bool{}
	if s == "" { return out }
	for _, p := range strings.Split(s, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" { continue }
		out[p] = true
	}
	return out
}

func intFromPtr(p *int) int {
	if p == nil { return 0 }
	return *p
}

func uuidFromQuery(c *fiber.Ctx, key string, badMsg string) (*uuid.UUID, bool, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" { return nil, false, nil }
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, false, helper.JsonError(c, fiber.StatusBadRequest, badMsg)
	}
	return &id, true, nil
}

func derefString(p *string) string {
	if p == nil { return "" }
	return *p
}
func derefUUID(p *uuid.UUID) uuid.UUID {
	if p == nil { return uuid.Nil }
	return *p
}
func derefBool(p *bool) bool {
	if p == nil { return false }
	return *p
}
