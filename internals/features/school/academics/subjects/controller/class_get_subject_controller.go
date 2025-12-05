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

func (h *ClassSubjectController) List(c *fiber.Ctx) error {
	// Kalau helper lain butuh DB di Locals
	c.Locals("DB", h.DB)

	// =====================================================
	// 1) Tentukan schoolID:
	//    - Prioritas: dari token (GetSchoolIDFromTokenPreferTeacher)
	//    - Fallback: dari ResolveSchoolContext (id / slug)
	// =====================================================

	var schoolID uuid.UUID

	// 1. Coba dulu dari token (kalau user login & token punya school)
	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		// 2. Kalau tidak ada / gagal dari token → pakai konteks umum (path/header/query/host)
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
			// bener-bener nggak dapat apapun
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

	// ===== Paging (jsonresponse helper; dukung page/per_page & limit/offset) =====
	// default per_page = 20, max = 200 sesuai kebutuhan endpoint ini
	p := helper.ResolvePaging(c, 20, 200)
	limit, offset := p.Limit, p.Offset

	// Override dari DTO kalau ada (dan masih dalam batas wajar)
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

	// ===== Soft delete (default exclude) =====
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

	// (opsional) Kalau di DTO kamu nanti ada field khusus untuk nama mapel, misalnya:
	//   SubjectName *string `query:"subject_name"`
	// bisa tambahin blok terpisah seperti ini:
	/*
		if q.SubjectName != nil && strings.TrimSpace(*q.SubjectName) != "" {
			kw := "%" + strings.ToLower(strings.TrimSpace(*q.SubjectName)) + "%"
			tx = tx.Where("LOWER(COALESCE(class_subject_subject_name_cache,'')) LIKE ?", kw)
		}
	*/

	// ===== Sorting whitelist =====
	// Dukungan:
	//   - DTO.Sort (enum): order_index_asc|order_index_desc|created_at_asc|created_at_desc|updated_at_asc|updated_at_desc
	//   - Query: sort_by=order_index|created_at|updated_at + order=asc|desc
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by", "created_at")))
	order := strings.ToLower(strings.TrimSpace(c.Query("order", "asc")))

	// DTO.Sort (enum) override kalau diisi
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

	// ===== Count sebelum limit/offset =====
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

	// ===== Pagination meta (jsonresponse) =====
	pg := helper.BuildPaginationFromOffset(total, offset, limit)

	// =====================================================
	// 2A) TANPA include apapun → simple list
	// =====================================================
	if !includeCSB && !includeBooks {
		if len(rows) == 0 {
			return helper.JsonList(c, "ok", []classSubjectDTO.ClassSubjectResponse{}, pg)
		}
		out := classSubjectDTO.FromClassSubjectModels(rows)
		return helper.JsonList(c, "ok", out, pg)
	}

	// =====================================================
	// 2B) Ada include → pakai jsonresponse + include
	// =====================================================

	// Primary data tetap basic: ClassSubjectResponse
	data := classSubjectDTO.FromClassSubjectModels(rows)

	// Kalau tidak ada class_subject sama sekali, balikin include kosong sesuai yang diminta
	if len(rows) == 0 {
		include := fiber.Map{}
		if includeCSB {
			include["class_subject_books"] = []bookModel.ClassSubjectBookModel{}
		}
		if includeBooks {
			include["books"] = []bookModel.BookModel{}
		}
		return helper.JsonListWithInclude(c, "ok", data, include, pg)
	}

	// Kumpulkan semua class_subject_id dari class_subjects
	classSubjectIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		classSubjectIDs = append(classSubjectIDs, r.ClassSubjectID)
	}

	// a) Ambil semua link class_subject_books (kalau butuh CSB atau books)
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

	// b) Ambil semua books yang dipakai (kalau diminta)
	var books []bookModel.BookModel
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
		}
	}

	// =====================================================
	// 3) Build include payload (hanya yang diminta)
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
