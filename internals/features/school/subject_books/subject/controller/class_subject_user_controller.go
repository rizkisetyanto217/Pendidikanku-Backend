// internals/features/lembaga/classes/subjects/main/controller/class_subject_list_controller.go
package controller

import (
	"strings"

	booksModel "masjidku_backend/internals/features/school/subject_books/books/model"
	csDTO "masjidku_backend/internals/features/school/subject_books/subject/dto"
	csModel "masjidku_backend/internals/features/school/subject_books/subject/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
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
   ========================================================= */
func (h *ClassSubjectController) List(c *fiber.Ctx) error {
	// ===== Tenancy: union semua klaim
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
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

	// ===== Base query
	tx := h.DB.Model(&csModel.ClassSubjectModel{}).
		Where("class_subjects_masjid_id IN ?", masjidIDs)

	// ===== Soft delete (default exclude)
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_subjects_deleted_at IS NULL")
	}

	// ===== Filter by id / ids (strict UUID)
	if ids, ok, errResp := uuidListFromQuery(c, "id", "ids"); errResp != nil {
		return errResp
	} else if ok {
		tx = tx.Where("class_subjects_id IN ?", ids)
	}

	// ===== Filter aktif
	if q.IsActive != nil {
		tx = tx.Where("class_subjects_is_active = ?", *q.IsActive)
	}

	// ===== Filter term
	// term_id single
	if termID, ok, errResp := uuidFromQuery(c, "term_id", "term_id tidak valid"); errResp != nil {
		return errResp
	} else if ok {
		tx = tx.Where("class_subjects_term_id = ?", *termID)
	}
	// term_ids multi
	if tids, ok, errResp := uuidListFromQuery(c, "term_ids"); errResp != nil {
		return errResp
	} else if ok {
		tx = tx.Where("class_subjects_term_id IN ?", tids)
	}
	// term_id_isnull
	if v := strings.TrimSpace(c.Query("term_id_isnull")); v != "" {
		if c.QueryBool("term_id_isnull") {
			tx = tx.Where("class_subjects_term_id IS NULL")
		}
	}

	// ===== Search di desc
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("LOWER(COALESCE(class_subjects_desc,'')) LIKE ?", kw)
	}

	// ===== Sorting whitelist
	orderBy := "class_subjects_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(strings.TrimSpace(*q.OrderBy)) {
		case "order_index":
			orderBy = "class_subjects_order_index"
		case "created_at":
			orderBy = "class_subjects_created_at"
		case "updated_at":
			orderBy = "class_subjects_updated_at"
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
			class_subjects_id,
			class_subjects_masjid_id,
			class_subjects_class_id,
			class_subjects_subject_id,
			class_subjects_term_id,
			class_subjects_order_index,
			class_subjects_hours_per_week,
			class_subjects_min_passing_score,
			class_subjects_weight_on_report,
			class_subjects_is_core,
			class_subjects_desc,
			class_subjects_is_active,
			class_subjects_created_at,
			class_subjects_updated_at,
			class_subjects_deleted_at
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

	// ===== include=books (ambil links & buku dalam batch, tenant-safe)
	// Kumpulkan CS IDs
	csIDs := make([]uuid.UUID, 0, len(rows))
	for _, m := range rows {
		csIDs = append(csIDs, m.ClassSubjectsID)
	}

	linksByCS := map[uuid.UUID][]booksModel.ClassSubjectBookModel{}
	bookIDsSet := map[uuid.UUID]struct{}{}

	if len(csIDs) > 0 {
		var links []booksModel.ClassSubjectBookModel
		if err := h.DB.
			Where("class_subject_books_deleted_at IS NULL").
			Where("class_subject_books_masjid_id IN ?", masjidIDs).
			Where("class_subject_books_class_subject_id IN ?", csIDs).
			Find(&links).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil relasi buku")
		}
		for _, l := range links {
			linksByCS[l.ClassSubjectBooksClassSubjectID] = append(linksByCS[l.ClassSubjectBooksClassSubjectID], l)
			bookIDsSet[l.ClassSubjectBooksBookID] = struct{}{}
		}
	}

	bookByID := map[uuid.UUID]booksModel.BooksModel{}
	if len(bookIDsSet) > 0 {
		bookIDs := make([]uuid.UUID, 0, len(bookIDsSet))
		for id := range bookIDsSet {
			bookIDs = append(bookIDs, id)
		}
		var books []booksModel.BooksModel
		if err := h.DB.
			Where("books_deleted_at IS NULL").
			Where("books_masjid_id IN ?", masjidIDs).
			Where("books_id IN ?", bookIDs).
			Find(&books).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data buku")
		}
		for _, b := range books {
			bookByID[b.BooksID] = b
		}
	}

	// DTO with books
	items := make([]csDTO.ClassSubjectWithBooksResponse, 0, len(rows))
	for _, m := range rows {
		links := linksByCS[m.ClassSubjectsID]
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

