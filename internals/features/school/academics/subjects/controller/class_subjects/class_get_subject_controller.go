// file: internals/features/lembaga/classes/subjects/main/controller/class_subject_list_controller.go
package controller

import (
	"errors"
	"strings"

	bookModel "madinahsalam_backend/internals/features/school/academics/books/model"
	classSubjectDTO "madinahsalam_backend/internals/features/school/academics/subjects/dto"
	csModel "madinahsalam_backend/internals/features/school/academics/subjects/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =========================
// small helpers
// =========================

// nested param bisa:
// - "1", "true", "yes" → all (ikut include)
// - "class_subject_books"
// - "books"
// - "class_subject_books,books"
func parseNested(raw string) (nestedAny, nestedCSB, nestedBooks bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}

	parts := strings.Split(raw, ",")
	for _, p := range parts {
		t := strings.ToLower(strings.TrimSpace(p))
		switch t {
		case "1", "true", "yes", "all":
			nestedAny = true
		case "class_subject_books", "csb":
			nestedCSB = true
		case "books", "book":
			nestedBooks = true
		}
	}

	return
}

// struct untuk data + nested di dalam "data"
type classSubjectWithNested struct {
	classSubjectDTO.ClassSubjectResponse
	ClassSubjectBooks []bookModel.ClassSubjectBookModel `json:"class_subject_books,omitempty"`
	Books             []bookModel.BookModel             `json:"books,omitempty"`
}

func (h *ClassSubjectController) List(c *fiber.Ctx) error {
	// Kalau helper lain butuh DB di Locals
	c.Locals("DB", h.DB)

	// =====================================================
	// 1) Tentukan schoolID (token → context)
	// =====================================================

	var schoolID uuid.UUID

	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		mc, err := helperAuth.ResolveSchoolContext(c)
		if err != nil {
			return err
		}

		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id
		default:
			return helperAuth.ErrSchoolContextMissing
		}
	}

	// ===== Parse query DTO (toleran) =====
	var q classSubjectDTO.ListClassSubjectQuery
	_ = c.QueryParser(&q)

	// ===== Param include=... (boleh comma-separated) =====
	includeRaw := c.Query("include", "")
	includeCSB := false
	includeBooks := false

	if strings.TrimSpace(includeRaw) != "" {
		for _, part := range strings.Split(includeRaw, ",") {
			token := strings.TrimSpace(strings.ToLower(part))
			switch token {
			case "class_subject_books":
				includeCSB = true
			case "books", "book":
				includeBooks = true
			}
		}
	}

	// ===== Param nested=... =====
	nestedRaw := c.Query("nested", "")
	nestedAny, nestedCSB, nestedBooks := parseNested(nestedRaw)

	// Kalau nested=1/true/all dan belum spesifik CSB/books,
	// anggap "nested semua include yang aktif".
	if nestedAny && !nestedCSB && !nestedBooks {
		nestedCSB = includeCSB
		nestedBooks = includeBooks
	}

	// Kalau user minta nested spesifik, otomatis aktifkan include sumber datanya
	if nestedCSB {
		includeCSB = true
	}
	if nestedBooks {
		includeBooks = true
	}

	// ===== Paging =====
	p := helper.ResolvePaging(c, 20, 200)
	limit, offset := p.Limit, p.Offset

	if q.Limit != nil {
		if *q.Limit < 1 {
			limit = 1
		} else if *q.Limit > 200 {
			limit = 200
		} else {
			limit = *q.Limit
		}
	}
	if q.Offset != nil && *q.Offset >= 0 {
		offset = *q.Offset
	}

	// ===== Base query (tenant-safe) =====
	tx := h.DB.Model(&csModel.ClassSubjectModel{}).
		Where("class_subject_school_id = ?", schoolID)

	// ===== Soft delete =====
	withDeleted := q.WithDeleted != nil && *q.WithDeleted
	if !withDeleted {
		tx = tx.Where("class_subject_deleted_at IS NULL")
	}

	// ===== Filters =====
	if q.IsActive != nil {
		tx = tx.Where("class_subject_is_active = ?", *q.IsActive)
	}
	if q.ClassParentID != nil {
		tx = tx.Where("class_subject_class_parent_id = ?", *q.ClassParentID)
	}
	if q.SubjectID != nil {
		tx = tx.Where("class_subject_subject_id = ?", *q.SubjectID)
	}

	// 🔍 Full-text sederhana: q → desc + subject_name_cache
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where(`
			LOWER(COALESCE(class_subject_desc, '')) LIKE ? OR
			LOWER(COALESCE(class_subject_subject_name_cache, '')) LIKE ?
		`, kw, kw)
	}

	// 🔍 Filter spesifik by subject name: ?name=
	if q.Name != nil && strings.TrimSpace(*q.Name) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Name)) + "%"
		tx = tx.Where("LOWER(COALESCE(class_subject_subject_name_cache,'')) LIKE ?", kw)
	}

	// ===== Sorting =====
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by", "created_at")))
	order := strings.ToLower(strings.TrimSpace(c.Query("order", "asc")))

	if q.Sort != nil {
		switch strings.ToLower(strings.TrimSpace(*q.Sort)) {
		case "order_index_asc":
			sortBy, order = "order_index", "asc"
		case "order_index_desc":
			sortBy, order = "order_index", "desc"
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
		order = "asc"
	}

	colMap := map[string]string{
		"order_index": "class_subject_order_index",
		"created_at":  "class_subject_created_at",
		"updated_at":  "class_subject_updated_at",
	}
	col, ok := colMap[sortBy]
	if !ok {
		col = colMap["created_at"]
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// ===== Count =====
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// ===== Data class_subject =====
	var rows []csModel.ClassSubjectModel
	if err := tx.
		Order(orderExpr).
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ===== Pagination meta =====
	pg := helper.BuildPaginationFromOffset(total, offset, limit)

	// Primary data: DTO biasa
	data := classSubjectDTO.FromClassSubjectModels(rows)

	// Kalau benar-benar nggak ada include & nggak ada nested → simple list
	if !includeCSB && !includeBooks {
		return helper.JsonList(c, "ok", data, pg)
	}

	// Kalau nggak ada rows tapi include diminta:
	if len(rows) == 0 {
		// kalau nested diminta, FOKUS ke nested → data kosong tanpa include
		if nestedCSB || nestedBooks {
			return helper.JsonList(c, "ok", []classSubjectWithNested{}, pg)
		}

		// else: include kosong (flat mode)
		include := fiber.Map{}
		if includeCSB {
			include["class_subject_books"] = []bookModel.ClassSubjectBookModel{}
		}
		if includeBooks {
			include["books"] = []bookModel.BookModel{}
		}
		return helper.JsonListWithInclude(c, "ok", data, include, pg)
	}

	// =====================================================
	// Ambil relasi (CSB & Books)
	// =====================================================

	// Kumpulkan semua class_subject_id
	classSubjectIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		classSubjectIDs = append(classSubjectIDs, r.ClassSubjectID)
	}

	// a) link class_subject_books
	var links []bookModel.ClassSubjectBookModel
	if includeCSB || includeBooks {
		if err := h.DB.
			Where(
				"class_subject_book_school_id = ? "+
					"AND class_subject_book_deleted_at IS NULL "+
					"AND class_subject_book_class_subject_id IN ?",
				schoolID, classSubjectIDs,
			).
			Find(&links).Error; err != nil {

			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data buku mapel")
		}
	}

	// b) books
	var books []bookModel.BookModel
	bookByID := make(map[uuid.UUID]bookModel.BookModel)

	if includeBooks && len(links) > 0 {
		bookIDsSet := make(map[uuid.UUID]struct{})
		for _, l := range links {
			bookIDsSet[l.ClassSubjectBookBookID] = struct{}{}
		}

		bookIDs := make([]uuid.UUID, 0, len(bookIDsSet))
		for id := range bookIDsSet {
			bookIDs = append(bookIDs, id)
		}

		if len(bookIDs) > 0 {
			if err := h.DB.
				Where(
					"book_school_id = ? "+
						"AND book_deleted_at IS NULL "+
						"AND book_id IN ?",
					schoolID, bookIDs,
				).
				Find(&books).Error; err != nil {

				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data buku")
			}

			for _, b := range books {
				bookByID[b.BookID] = b
			}
		}
	}

	// =====================================================
	// Build nested maps (kalau diminta)
	// =====================================================

	var nestedCSBMap map[string][]bookModel.ClassSubjectBookModel
	if nestedCSB {
		nestedCSBMap = make(map[string][]bookModel.ClassSubjectBookModel)
		for _, l := range links {
			csid := l.ClassSubjectBookClassSubjectID.String()
			nestedCSBMap[csid] = append(nestedCSBMap[csid], l)
		}
	}

	var nestedBooksMap map[string][]bookModel.BookModel
	if nestedBooks {
		nestedBooksMap = make(map[string][]bookModel.BookModel)
		for _, l := range links {
			b, ok := bookByID[l.ClassSubjectBookBookID]
			if !ok {
				continue
			}
			csid := l.ClassSubjectBookClassSubjectID.String()

			// hindari duplikat buku di satu class_subject
			existing := nestedBooksMap[csid]
			dup := false
			for _, ex := range existing {
				if ex.BookID == b.BookID {
					dup = true
					break
				}
			}
			if !dup {
				nestedBooksMap[csid] = append(existing, b)
			}
		}
	}

	// =====================================================
	// MODE 1: NESTED → data[] sudah berisi relasi, TANPA include
	// =====================================================
	if nestedCSB || nestedBooks {
		outNested := make([]classSubjectWithNested, 0, len(data))
		for _, d := range data {
			item := classSubjectWithNested{
				ClassSubjectResponse: d,
			}
			csid := d.ID.String()

			if nestedCSB && nestedCSBMap != nil {
				if list, ok := nestedCSBMap[csid]; ok {
					item.ClassSubjectBooks = list
				}
			}
			if nestedBooks && nestedBooksMap != nil {
				if list, ok := nestedBooksMap[csid]; ok {
					item.Books = list
				}
			}

			outNested = append(outNested, item)
		}

		// 🔑 Sesuai permintaan: kalau sudah nested → JANGAN kirim include lagi
		return helper.JsonList(c, "ok", outNested, pg)
	}

	// =====================================================
	// MODE 2: FLAT + INCLUDE
	// =====================================================

	include := fiber.Map{}
	if includeCSB {
		include["class_subject_books"] = links
	}
	if includeBooks {
		include["books"] = books
	}

	return helper.JsonListWithInclude(c, "ok", data, include, pg)
}
