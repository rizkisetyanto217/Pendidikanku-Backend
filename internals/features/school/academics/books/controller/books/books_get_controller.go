// file: internals/features/lembaga/class_books/controller/books_controller.go
package controller

import (
	"errors"
	"strings"

	bookdto "madinahsalam_backend/internals/features/school/academics/books/dto"
	bookModel "madinahsalam_backend/internals/features/school/academics/books/model"
	classSubjectDTO "madinahsalam_backend/internals/features/school/academics/subjects/dto"
	classSubjectModel "madinahsalam_backend/internals/features/school/academics/subjects/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BooksInclude struct {
	ClassSubjects []classSubjectDTO.ClassSubjectCompactResponse `json:"class_subjects,omitempty"`
}

func (h *BooksController) List(c *fiber.Ctx) error {
	c.Locals("DB", h.DB)

	var schoolID uuid.UUID

	// ===== School context (token-aware, PUBLIC) =====
	if id, err := helperAuth.GetActiveSchoolID(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		mc, err2 := helperAuth.ResolveSchoolContext(c)
		if err2 != nil {
			return err2
		}

		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID
		case strings.TrimSpace(mc.Slug) != "":
			s := strings.TrimSpace(mc.Slug)
			if id2, errParse := uuid.Parse(s); errParse == nil {
				schoolID = id2
			} else {
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
			return helperAuth.ErrSchoolContextMissing
		}
	}

	// ===== Query params dasar =====
	q := strings.TrimSpace(c.Query("q"))
	author := strings.TrimSpace(c.Query("author"))
	name := strings.TrimSpace(c.Query("name"))
	withDeleted := strings.EqualFold(strings.TrimSpace(c.Query("with_deleted")), "true")

	// mode: compact | full (default: full)
	modeParam := strings.TrimSpace(c.Query("mode"))
	if modeParam == "" {
		modeParam = strings.TrimSpace(c.Query("view_mode"))
	}
	mode := strings.ToLower(modeParam)
	isCompact := mode == "compact"

	// nested flags: ?nested=class_subjects
	nestedParam := strings.TrimSpace(c.Query("nested"))
	nestedClassSubjects := false
	if nestedParam != "" {
		for _, part := range strings.Split(nestedParam, ",") {
			p := strings.ToLower(strings.TrimSpace(part))
			if p == "class_subjects" || p == "class_subject_books" || p == "csb" {
				nestedClassSubjects = true
			}
		}
	}

	// include flags: ?include=class_subjects
	includeParam := strings.TrimSpace(c.Query("include"))
	includeClassSubjects := false
	if includeParam != "" {
		for _, part := range strings.Split(includeParam, ",") {
			p := strings.ToLower(strings.TrimSpace(part))
			if p == "class_subjects" {
				includeClassSubjects = true
			}
		}
	}

	// ===== Pagination & sorting =====
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)

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
		p.SortOrder = strings.ToLower(v)
	}

	allowedSort := map[string]string{
		"created_at": "book_created_at",
		"title":      "book_title",
		"author":     "book_author",
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
		return helper.JsonError(c, fiber.StatusBadRequest, "id berisi UUID tidak valid")
	}
	if len(idFilter) == 0 {
		if tmp, e2 := parseIDsCSV(c.Query("book_id")); e2 != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "book_id berisi UUID tidak valid")
		} else {
			idFilter = tmp
		}
	}

	// ===== Query dasar (pakai BookModel) =====
	base := h.DB.Model(&bookModel.BookModel{}).
		Where("book_school_id = ?", schoolID)

	if !withDeleted {
		base = base.Where("book_deleted_at IS NULL")
	}
	if len(idFilter) > 0 {
		base = base.Where("book_id IN ?", idFilter)
		p.Page = 1
		p.PerPage = len(idFilter)
	}
	if q != "" {
		needle := "%" + q + "%"
		base = base.Where(
			h.DB.Where("book_title ILIKE ?", needle).
				Or("book_author ILIKE ?", needle).
				Or("book_desc ILIKE ?", needle),
		)
	}
	if author != "" {
		needle := "%" + author + "%"
		base = base.Where("book_author ILIKE ?", needle)
	}
	if name != "" {
		needle := "%" + name + "%"
		base = base.Where("book_title ILIKE ?", needle)
	}

	// ===== Count total =====
	var total int64
	if err := base.Session(&gorm.Session{}).
		Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// ===== Ambil data halaman =====
	var books []bookModel.BookModel
	if err := base.
		Order(orderExpr).
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&books).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data buku")
	}

	pg := helper.BuildPaginationFromOffset(total, p.Offset(), p.Limit())

	if len(books) == 0 {
		if isCompact {
			empty := []bookdto.BookCompact{}
			return helper.JsonList(c, "ok", empty, pg)
		}
		emptyFull := []bookdto.BookResponse{}
		return helper.JsonList(c, "ok", emptyFull, pg)
	}

	// ===== Ambil class_subject_books + detail class_subject (untuk nested/include) =====
	var (
		csbByBookID               map[uuid.UUID][]bookdto.BookClassSubjectItem
		includeClassSubjectsSlice []classSubjectDTO.ClassSubjectCompactResponse
	)

	if (nestedClassSubjects || includeClassSubjects) && len(books) > 0 {
		// 1) Kumpulkan semua book_id di halaman ini
		bookIDs := make([]uuid.UUID, 0, len(books))
		for _, b := range books {
			bookIDs = append(bookIDs, b.BookID)
		}

		// 2) Ambil pivot class_subject_books (per buku)
		type csbRow struct {
			ClassSubjectBookID             uuid.UUID `gorm:"column:class_subject_book_id"`
			ClassSubjectBookBookID         uuid.UUID `gorm:"column:class_subject_book_book_id"`
			ClassSubjectBookClassSubjectID uuid.UUID `gorm:"column:class_subject_book_class_subject_id"`
			ClassSubjectBookIsPrimary      bool      `gorm:"column:class_subject_book_is_primary"`
			ClassSubjectBookIsRequired     bool      `gorm:"column:class_subject_book_is_required"`
			ClassSubjectBookOrder          *int      `gorm:"column:class_subject_book_order"`
		}

		var csbRows []csbRow
		if err := h.DB.Table("class_subject_books AS csb").
			Where("csb.class_subject_book_school_id = ?", schoolID).
			Where("csb.class_subject_book_is_active = TRUE").
			Where("csb.class_subject_book_deleted_at IS NULL").
			Where("csb.class_subject_book_book_id IN ?", bookIDs).
			Select(`
				csb.class_subject_book_id,
				csb.class_subject_book_book_id,
				csb.class_subject_book_class_subject_id,
				csb.class_subject_book_is_primary,
				csb.class_subject_book_is_required,
				csb.class_subject_book_order
			`).
			Scan(&csbRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil relasi class_subject_books")
		}

		if len(csbRows) > 0 {
			// 3) Kumpulkan semua class_subject_id dari pivot
			classSubjectIDsSet := make(map[uuid.UUID]struct{}, len(csbRows))
			for _, r := range csbRows {
				classSubjectIDsSet[r.ClassSubjectBookClassSubjectID] = struct{}{}
			}
			classSubjectIDs := make([]uuid.UUID, 0, len(classSubjectIDsSet))
			for id := range classSubjectIDsSet {
				classSubjectIDs = append(classSubjectIDs, id)
			}

			// 4) Ambil ClassSubjectModel untuk semua ID tersebut
			var csModels []classSubjectModel.ClassSubjectModel
			if err := h.DB.
				Where("class_subject_school_id = ?", schoolID).
				Where("class_subject_deleted_at IS NULL").
				Where("class_subject_id IN ?", classSubjectIDs).
				Find(&csModels).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data class_subjects")
			}

			// 5) Map class_subject_id -> DTO compact
			csByID := make(map[uuid.UUID]classSubjectDTO.ClassSubjectCompactResponse, len(csModels))
			for _, m := range csModels {
				csByID[m.ClassSubjectID] = classSubjectDTO.FromClassSubjectModelToCompact(m)
			}

			// 6) Siapkan slice untuk include.class_subjects
			includeClassSubjectsSlice = make([]classSubjectDTO.ClassSubjectCompactResponse, 0, len(csByID))
			for _, v := range csByID {
				includeClassSubjectsSlice = append(includeClassSubjectsSlice, v)
			}

			// 7) Susun per book_id (nested)
			csbByBookID = make(map[uuid.UUID][]bookdto.BookClassSubjectItem, len(bookIDs))
			for _, r := range csbRows {
				csCompact, ok := csByID[r.ClassSubjectBookClassSubjectID]
				if !ok {
					continue
				}

				item := bookdto.BookClassSubjectItem{
					ClassSubjectBookID:         r.ClassSubjectBookID,
					ClassSubjectBookIsPrimary:  r.ClassSubjectBookIsPrimary,
					ClassSubjectBookIsRequired: r.ClassSubjectBookIsRequired,
					ClassSubjectBookOrder:      r.ClassSubjectBookOrder,
					ClassSubject:               csCompact,
				}
				csbByBookID[r.ClassSubjectBookBookID] = append(csbByBookID[r.ClassSubjectBookBookID], item)
			}
		}
	}

	// ===== mode compact vs full =====
	if isCompact {
		out := make([]bookdto.BookCompact, 0, len(books))
		for i := range books {
			// 🔹 PAKAI VERSI TIMEZONE-AWARE
			item := bookdto.ToBookCompactWithSchoolTime(c, &books[i])

			if nestedClassSubjects && csbByBookID != nil {
				if items, ok := csbByBookID[books[i].BookID]; ok {
					item.ClassSubjectBooks = items
				}
			}

			out = append(out, item)
		}

		if !includeClassSubjects || len(includeClassSubjectsSlice) == 0 {
			return helper.JsonList(c, "ok", out, pg)
		}

		return helper.JsonListWithInclude(
			c,
			"ok",
			out,
			BooksInclude{
				ClassSubjects: includeClassSubjectsSlice,
			},
			pg,
		)
	}

	// ===== mode full: pakai BookResponse DTO =====
	full := make([]bookdto.BookResponse, 0, len(books))
	for i := range books {
		// 🔹 PAKAI VERSI TIMEZONE-AWARE
		item := bookdto.ToBookResponseWithSchoolTime(c, &books[i])

		if nestedClassSubjects && csbByBookID != nil {
			if items, ok := csbByBookID[books[i].BookID]; ok {
				item.ClassSubjectBooks = items
			}
		}

		full = append(full, item)
	}

	if !includeClassSubjects || len(includeClassSubjectsSlice) == 0 {
		return helper.JsonList(c, "ok", full, pg)
	}

	return helper.JsonListWithInclude(
		c,
		"ok",
		full,
		BooksInclude{
			ClassSubjects: includeClassSubjectsSlice,
		},
		pg,
	)
}
