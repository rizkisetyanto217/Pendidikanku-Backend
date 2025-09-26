// file: internals/features/lembaga/classes/subjects/main/controller/class_subject_list_controller.go
package controller

import (
	"errors"
	"strings"

	booksModel "masjidku_backend/internals/features/school/academics/books/model"
	csDTO "masjidku_backend/internals/features/school/academics/subject/dto"
	csModel "masjidku_backend/internals/features/school/academics/subject/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
=========================================================

	LIST
	GET /admin/class-subjects
	Query:
	  - q                       : cari pada desc (ILIKE)
	  - is_active               : bool
	  - term_id                 : UUID (single)
	  - term_ids                : comma-separated UUIDs (multi)
	  - term_id_isnull          : bool (filter yang tanpa term / NULL)
	  - id / ids                : filter by ID (single / multi)
	  - with_deleted            : bool (default false)
	  - order_by                : order_index|created_at|updated_at (default: created_at)
	  - sort                    : asc|desc (default: asc)
	  - include                 : books (include detail buku)
	  - limit (1..200), offset
	=========================================================
*/
func (h *ClassSubjectController) List(c *fiber.Ctx) error {
	// ===== Masjid context (PUBLIC): no role check =====
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err // fiber.Error dari resolver
	}
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal resolve masjid dari slug")
		}
		masjidID = id
	} else {
		return helperAuth.ErrMasjidContextMissing
	}

	// ===== Include flags
	includeBooks := wantInclude(c, "books")

	// ===== Parse & guard pagination
	var q csDTO.ListClassSubjectQuery
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	limit, offset := 20, 0
	if q.Limit != nil && *q.Limit > 0 && *q.Limit <= 200 {
		limit = *q.Limit
	}
	if q.Offset != nil && *q.Offset >= 0 {
		offset = *q.Offset
	}

	// ===== Base query (single-tenant via context) =====
	tx := h.DB.Model(&csModel.ClassSubjectModel{}).
		Where("class_subject_masjid_id = ?", masjidID)

	// ===== Soft delete (default exclude)
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_subject_deleted_at IS NULL")
	}

	// ===== Filter by id / ids (strict UUID)
	if ids, ok, errResp := uuidListFromQueryClassSubject(c, "id", "ids"); errResp != nil {
		return errResp
	} else if ok {
		tx = tx.Where("class_subject_id IN ?", ids)
	}

	// ===== Filter aktif
	if q.IsActive != nil {
		tx = tx.Where("class_subject_is_active = ?", *q.IsActive)
	}

	// ===== Filter term
	if termID, ok, errResp := uuidFromQuery(c, "term_id", "term_id tidak valid"); errResp != nil {
		return errResp
	} else if ok {
		tx = tx.Where("class_subject_term_id = ?", *termID)
	}
	if tids, ok, errResp := uuidListFromQueryClassSubject(c, "term_ids"); errResp != nil {
		return errResp
	} else if ok {
		tx = tx.Where("class_subject_term_id IN ?", tids)
	}
	if v := strings.TrimSpace(c.Query("term_id_isnull")); v != "" {
		if c.QueryBool("term_id_isnull") {
			tx = tx.Where("class_subject_term_id IS NULL")
		}
	}

	// ===== Search di desc
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("LOWER(COALESCE(class_subject_desc,'')) LIKE ?", kw)
	}

	// ===== Sorting whitelist (kolom singular)
	orderBy := "class_subject_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(strings.TrimSpace(*q.OrderBy)) {
		case "order_index":
			orderBy = "class_subject_order_index"
		case "created_at":
			orderBy = "class_subject_created_at"
		case "updated_at":
			orderBy = "class_subject_updated_at"
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(strings.TrimSpace(*q.Sort)) == "desc" {
		sort = "DESC"
	}

	// ===== Total (sebelum limit/offset)
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Data
	var rows []csModel.ClassSubjectModel
	if err := tx.
		Select(`
			class_subject_id,
			class_subject_masjid_id,
			class_subject_class_id,
			class_subject_subject_id,
			class_subject_order_index,
			class_subject_hours_per_week,
			class_subject_min_passing_score,
			class_subject_weight_on_report,
			class_subject_is_core,
			class_subject_desc,
			class_subject_is_active,
			class_subject_created_at,
			class_subject_updated_at,
			class_subject_deleted_at
		`).
		Order(orderBy + " " + sort).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ===== Tanpa include=books → langsung kirim
	if !includeBooks {
		return helper.JsonList(
			c,
			csDTO.FromClassSubjectModels(rows),
			csDTO.Pagination{Limit: limit, Offset: offset, Total: int(total)},
		)
	}

	// ===== include=books (tenant-safe & single masjid)
	csIDs := make([]uuid.UUID, 0, len(rows))
	for _, m := range rows {
		csIDs = append(csIDs, m.ClassSubjectID)
	}

	linksByCS := map[uuid.UUID][]booksModel.ClassSubjectBookModel{}
	bookIDsSet := map[uuid.UUID]struct{}{}

	if len(csIDs) > 0 {
		var links []booksModel.ClassSubjectBookModel
		if err := h.DB.
			Where("class_subject_book_deleted_at IS NULL").
			Where("class_subject_book_masjid_id = ?", masjidID).
			Where("class_subject_book_class_subject_id IN ?", csIDs).
			Find(&links).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil relasi buku")
		}
		for _, l := range links {
			linksByCS[l.ClassSubjectBookClassSubjectID] = append(linksByCS[l.ClassSubjectBookClassSubjectID], l)
			bookIDsSet[l.ClassSubjectBookBookID] = struct{}{}
		}
	}

	bookByID := map[uuid.UUID]booksModel.BookModel{}
	if len(bookIDsSet) > 0 {
		bookIDs := make([]uuid.UUID, 0, len(bookIDsSet))
		for id := range bookIDsSet {
			bookIDs = append(bookIDs, id)
		}
		var books []booksModel.BookModel
		if err := h.DB.
			Where("book_deleted_at IS NULL").
			Where("book_masjid_id = ?", masjidID).
			Where("book_id IN ?", bookIDs).
			Find(&books).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data buku")
		}
		for _, b := range books {
			bookByID[b.BookID] = b
		}
	}

	items := make([]csDTO.ClassSubjectWithBooksResponse, 0, len(rows))
	for _, m := range rows {
		links := linksByCS[m.ClassSubjectID]
		items = append(items, csDTO.NewClassSubjectWithBooksResponse(m, links, bookByID))
	}

	return helper.JsonList(
		c,
		items,
		csDTO.Pagination{Limit: limit, Offset: offset, Total: int(total)},
	)
}

/* ================= Helpers ================= */

func wantInclude(c *fiber.Ctx, key string) bool {
	inc := strings.ToLower(strings.TrimSpace(c.Query("include")))
	if inc == "all" {
		return true
	}
	for _, p := range strings.Split(inc, ",") {
		if strings.TrimSpace(p) == key || strings.TrimSpace(p) == key+"s" {
			return true
		}
	}
	return false
}

// uuidFromQuery: baca UUID single dari query param; balikan (val, ok, errResp)
func uuidFromQuery(c *fiber.Ctx, key string, badMsg string) (*uuid.UUID, bool, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, false, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, false, fiber.NewError(fiber.StatusBadRequest, badMsg)
	}
	return &id, true, nil
}

// // uuidListFromQuery: baca list UUID dari satu/lebih keys (mis. "id","ids")
// // Return: (ids, foundAny, errResp)
func uuidListFromQueryClassSubject(c *fiber.Ctx, keys ...string) ([]uuid.UUID, bool, error) {
	seen := map[uuid.UUID]struct{}{}
	for _, k := range keys {
		raw := strings.TrimSpace(c.Query(k))
		if raw == "" {
			continue
		}
		parts := strings.Split(raw, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := uuid.Parse(p)
			if err != nil {
				return nil, false, fiber.NewError(fiber.StatusBadRequest, k+" berisi UUID tidak valid")
			}
			seen[id] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil, false, nil
	}
	out := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out, true, nil
}
