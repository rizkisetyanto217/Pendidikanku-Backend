// internals/features/lembaga/class_subjects/controller/class_subject_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	booksModel "masjidku_backend/internals/features/lembaga/class_books/model"
	csDTO "masjidku_backend/internals/features/lembaga/class_lessons/dto"
	csModel "masjidku_backend/internals/features/lembaga/class_lessons/model"

	// booksModel "masjidku_backend/internals/features/lembaga/class_books/model"
	helper "masjidku_backend/internals/helpers"
)

type ClassSubjectController struct {
	DB *gorm.DB
}

/* ======================= Helpers ======================= */

// func intPtr(v int) *int { return &v }

func wantIncludeBooks(c *fiber.Ctx) bool {
	inc := strings.ToLower(strings.TrimSpace(c.Query("include")))
	if inc == "" {
		return false
	}
	for _, p := range strings.Split(inc, ",") {
		if strings.TrimSpace(p) == "books" || strings.TrimSpace(p) == "book" {
			return true
		}
	}
	return false
}

/* =========================================================
   CREATE
   POST /admin/class-subjects
   ========================================================= */
func (h *ClassSubjectController) Create(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	var req csDTO.CreateClassSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant
	req.MasjidID = masjidID

	// Normalisasi ringan
	if req.AcademicYear != nil {
		ay := strings.TrimSpace(*req.AcademicYear)
		req.AcademicYear = &ay
	}
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		req.Desc = &d
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Cek duplikasi kombinasi (soft delete aware)
		var cnt int64
		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where(`
				class_subjects_masjid_id = ?
				AND class_subjects_class_id = ?
				AND class_subjects_subject_id = ?
				AND COALESCE(class_subjects_academic_year,'') = COALESCE(?, '')
				AND class_subjects_deleted_at IS NULL
			`, req.MasjidID, req.ClassID, req.SubjectID, req.AcademicYear).
			Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+tahun ajaran sudah terdaftar")
		}

		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+tahun ajaran sudah terdaftar")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat class subject")
		}
		c.Locals("created_class_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("created_class_subject").(csModel.ClassSubjectModel)
	return helper.JsonCreated(c, "Class subject berhasil dibuat", csDTO.FromClassSubjectModel(m))
}


//    ========================================================= */
// GET /admin/class-subjects/:id?with_deleted=&include=books,teachers&teacher_user_id=&section_id=&only_active=
func (h *ClassSubjectController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid") }

	withDeleted := strings.EqualFold(c.Query("with_deleted"), "true")
	includeParam := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeBooks := strings.Contains(includeParam, "books")

	// --- parse filter guru opsional
	var teacherID *uuid.UUID
	if v := strings.TrimSpace(c.Query("teacher_user_id")); v != "" {
		tid, err := uuid.Parse(v); if err != nil { return fiber.NewError(fiber.StatusBadRequest, "teacher_user_id tidak valid") }
		teacherID = &tid
	}
	var sectionID *uuid.UUID
	if v := strings.TrimSpace(c.Query("section_id")); v != "" {
		sid, err := uuid.Parse(v); if err != nil { return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid") }
		sectionID = &sid
	}
	onlyActive := !strings.EqualFold(c.Query("only_active"), "false")
	includeTeachers := strings.Contains(includeParam, "teachers") || teacherID != nil || sectionID != nil

	// --- ambil class_subject
	var m csModel.ClassSubjectModel
	if err := h.DB.First(&m, "class_subjects_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan") }
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassSubjectsMasjidID != masjidID { return fiber.NewError(fiber.StatusForbidden, "Akses ditolak") }
	if !withDeleted && m.ClassSubjectsDeletedAt != nil { return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan") }

	base := csDTO.FromClassSubjectModel(m)
	out := fiber.Map{
		"class_subjects_id":                base.ID,
		"class_subjects_masjid_id":         base.MasjidID,
		"class_subjects_class_id":          base.ClassID,
		"class_subjects_subject_id":        base.SubjectID,
		"class_subjects_order_index":       base.OrderIndex,
		"class_subjects_hours_per_week":    base.HoursPerWeek,
		"class_subjects_min_passing_score": base.MinScore,
		"class_subjects_weight_on_report":  base.Weight,
		"class_subjects_is_core":           base.IsCore,
		"class_subjects_academic_year":     base.AcademicYear,
		"class_subjects_desc":              base.Desc,
		"class_subjects_is_active":         base.IsActive,
		"class_subjects_created_at":        base.CreatedAt,
		"class_subjects_updated_at":        base.UpdatedAt,
		"class_subjects_deleted_at":        base.DeletedAt,
	}

	// ===== include=books =====
	if includeBooks {
		var links []booksModel.ClassSubjectBookModel
		if err := h.DB.
			Where(`
				class_subject_books_masjid_id = ?
				AND class_subject_books_class_subject_id = ?
				AND class_subject_books_deleted_at IS NULL
			`, masjidID, m.ClassSubjectsID).
			Find(&links).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil relasi buku")
		}

		bookIDs := make([]uuid.UUID, 0, len(links))
		seen := map[uuid.UUID]struct{}{}
		for _, l := range links {
			if _, ok := seen[l.ClassSubjectBooksBookID]; !ok {
				seen[l.ClassSubjectBooksBookID] = struct{}{}
				bookIDs = append(bookIDs, l.ClassSubjectBooksBookID)
			}
		}
		bookByID := map[uuid.UUID]booksModel.BooksModel{}
		if len(bookIDs) > 0 {
			var books []booksModel.BooksModel
			if err := h.DB.
				Where("books_masjid_id = ? AND books_deleted_at IS NULL", masjidID).
				Where("books_id IN ?", bookIDs).
				Find(&books).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data buku")
			}
			for _, b := range books { bookByID[b.BooksID] = b }
		}
		out["class_subject_books"] = csDTO.NewClassSubjectWithBooksResponse(m, links, bookByID).ClassSubjectBooks
	}

	// ===== include=teachers =====
	// ===== include=teachers =====
	if includeTeachers {
    type CSST = csModel.ClassSectionSubjectTeacherModel

    // Subquery: semua section milik class ini & belum dihapus
    sub := h.DB.
        Table("class_sections").
        Select("class_sections_id").
        Where("class_sections_class_id = ? AND class_sections_deleted_at IS NULL", m.ClassSubjectsClassID)

    q := h.DB.Model(&CSST{}).
        Where(`
            class_section_subject_teachers_masjid_id = ?
            AND class_section_subject_teachers_subject_id = ?
            AND class_section_subject_teachers_deleted_at IS NULL
        `, masjidID, m.ClassSubjectsSubjectID)

    // Batasi ke section milik class ini, kecuali user tegas minta section tertentu
    if sectionID != nil {
        q = q.Where("class_section_subject_teachers_section_id = ?", *sectionID)
    } else {
        q = q.Where("class_section_subject_teachers_section_id IN (?)", sub)
    }

    if onlyActive {
        q = q.Where("class_section_subject_teachers_is_active = TRUE")
    }
    if teacherID != nil {
        q = q.Where("class_section_subject_teachers_teacher_user_id = ?", *teacherID)
    }

    var rows []CSST
    if err := q.Find(&rows).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data guru")
    }

    teachers := make([]fiber.Map, 0, len(rows))
    for _, r := range rows {
        teachers = append(teachers, fiber.Map{
            "class_section_subject_teachers_id":             r.ClassSectionSubjectTeachersID,
            "class_section_subject_teachers_section_id":      r.ClassSectionSubjectTeacherModelSectionID,
            "class_section_subject_teachers_teacher_user_id": r.ClassSectionSubjectTeacherModelTeacherUserID,
            "class_section_subject_teachers_is_active":       r.ClassSectionSubjectTeacherModelIsActive,
            "class_section_subject_teachers_created_at":      r.ClassSectionSubjectTeacherModelCreatedAt,
            "class_section_subject_teachers_updated_at":      r.ClassSectionSubjectTeacherModelUpdatedAt,
        })
    }
    out["class_section_subject_teachers"] = teachers
}


	return helper.JsonOK(c, "Detail class subject", out)
}



/* =========================================================
   LIST
   GET /admin/class-subjects?q=&is_active=&order_by=&sort=&limit=&offset=&with_deleted=&include=books
   ========================================================= */
func (h *ClassSubjectController) List(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	includeBooks := wantIncludeBooks(c)

	var q csDTO.ListClassSubjectQuery
	// default pagination
	q.Limit, q.Offset = intPtr(20), intPtr(0)
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}
	// guard pagination
	if q.Limit == nil || *q.Limit <= 0 || *q.Limit > 200 {
		q.Limit = intPtr(20)
	}
	if q.Offset == nil || *q.Offset < 0 {
		q.Offset = intPtr(0)
	}

	tx := h.DB.Model(&csModel.ClassSubjectModel{}).
		Where("class_subjects_masjid_id = ?", masjidID)
	// exclude soft-deleted by default
	if q.WithDeleted == nil || !*q.WithDeleted {
		tx = tx.Where("class_subjects_deleted_at IS NULL")
	}
	if q.IsActive != nil {
		tx = tx.Where("class_subjects_is_active = ?", *q.IsActive)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		kw := "%" + strings.ToLower(strings.TrimSpace(*q.Q)) + "%"
		tx = tx.Where("LOWER(COALESCE(class_subjects_academic_year,'')) LIKE ?", kw)
	}
	// order
	orderBy := "class_subjects_created_at"
	if q.OrderBy != nil {
		switch strings.ToLower(*q.OrderBy) {
		case "order_index":
			orderBy = "class_subjects_order_index"
		case "created_at":
			orderBy = "class_subjects_created_at"
		case "updated_at":
			orderBy = "class_subjects_updated_at"
		}
	}
	sort := "ASC"
	if q.Sort != nil && strings.ToLower(*q.Sort) == "desc" {
		sort = "DESC"
	}

	// total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// data
	var rows []csModel.ClassSubjectModel
	if err := tx.
		Select(`
			class_subjects_id,
			class_subjects_masjid_id,
			class_subjects_class_id,
			class_subjects_subject_id,
			class_subjects_order_index,
			class_subjects_hours_per_week,
			class_subjects_min_passing_score,
			class_subjects_weight_on_report,
			class_subjects_is_core,
			class_subjects_academic_year,
			class_subjects_desc,
			class_subjects_is_active,
			class_subjects_created_at,
			class_subjects_updated_at,
			class_subjects_deleted_at
		`).
		Order(orderBy + " " + sort).
		Limit(*q.Limit).
		Offset(*q.Offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Tanpa include buku → response biasa
	if !includeBooks {
		return helper.JsonList(
			c,
			csDTO.FromClassSubjectModels(rows),
			csDTO.Pagination{Limit: *q.Limit, Offset: *q.Offset, Total: int(total)},
		)
	}

	// ==== include=books untuk list ====
	// ambil semua links untuk csIDs yang ditampilkan
	csIDs := make([]uuid.UUID, 0, len(rows))
	for _, m := range rows {
		csIDs = append(csIDs, m.ClassSubjectsID)
	}

	linksByCS := map[uuid.UUID][]booksModel.ClassSubjectBookModel{}
	bookIDsSet := map[uuid.UUID]struct{}{}

	if len(csIDs) > 0 {
		var links []booksModel.ClassSubjectBookModel
		if err := h.DB.
			Where("class_subject_books_masjid_id = ? AND class_subject_books_deleted_at IS NULL", masjidID).
			Where("class_subject_books_class_subject_id IN ?", csIDs).
			Find(&links).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil relasi buku")
		}
		for _, l := range links {
			linksByCS[l.ClassSubjectBooksClassSubjectID] = append(linksByCS[l.ClassSubjectBooksClassSubjectID], l)
			bookIDsSet[l.ClassSubjectBooksBookID] = struct{}{}
		}
	}

	// ambil semua books yang dibutuhkan
	bookByID := map[uuid.UUID]booksModel.BooksModel{}
	if len(bookIDsSet) > 0 {
		bookIDs := make([]uuid.UUID, 0, len(bookIDsSet))
		for id := range bookIDsSet {
			bookIDs = append(bookIDs, id)
		}

		var books []booksModel.BooksModel
		if err := h.DB.
			Where("books_masjid_id = ? AND books_deleted_at IS NULL", masjidID).
			Where("books_id IN ?", bookIDs).
			Find(&books).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data buku")
		}
		for _, b := range books {
			bookByID[b.BooksID] = b
		}
	}

	// susun response nested
	items := make([]csDTO.ClassSubjectWithBooksResponse, 0, len(rows))
	for _, m := range rows {
		links := linksByCS[m.ClassSubjectsID]
		items = append(items, csDTO.NewClassSubjectWithBooksResponse(m, links, bookByID))
	}

	return helper.JsonList(
		c,
		items,
		csDTO.Pagination{Limit: *q.Limit, Offset: *q.Offset, Total: int(total)},
	)
}

/* =========================================================
   UPDATE (partial)
   PUT /admin/class-subjects/:id
   ========================================================= */
func (h *ClassSubjectController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req csDTO.UpdateClassSubjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant
	req.MasjidID = &masjidID

	// Normalisasi ringan
	if req.AcademicYear != nil {
		ay := strings.TrimSpace(*req.AcademicYear)
		req.AcademicYear = &ay
	}
	if req.Desc != nil {
		d := strings.TrimSpace(*req.Desc)
		req.Desc = &d
	}

	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var m csModel.ClassSubjectModel
		if err := tx.First(&m, "class_subjects_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah data milik masjid lain")
		}
		if m.ClassSubjectsDeletedAt != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
		}

		// Cek duplikat jika kombinasi berubah
		shouldCheckDup := false
		newClassID := m.ClassSubjectsClassID
		newSubjectID := m.ClassSubjectsSubjectID
		var newAcademicYear *string = m.ClassSubjectsAcademicYear

		if req.ClassID != nil && *req.ClassID != m.ClassSubjectsClassID {
			shouldCheckDup = true
			newClassID = *req.ClassID
		}
		if req.SubjectID != nil && *req.SubjectID != m.ClassSubjectsSubjectID {
			shouldCheckDup = true
			newSubjectID = *req.SubjectID
		}
		if req.AcademicYear != nil {
			ay := strings.TrimSpace(*req.AcademicYear)
			curr := ""
			if m.ClassSubjectsAcademicYear != nil {
				curr = strings.TrimSpace(*m.ClassSubjectsAcademicYear)
			}
			if ay != curr {
				shouldCheckDup = true
			}
			if ay == "" {
				newAcademicYear = nil
			} else {
				newAcademicYear = &ay
			}
		}

		if shouldCheckDup {
			var cnt int64
			if err := tx.Model(&csModel.ClassSubjectModel{}).
				Where(`
					class_subjects_masjid_id = ?
					AND class_subjects_class_id = ?
					AND class_subjects_subject_id = ?
					AND COALESCE(class_subjects_academic_year,'') = COALESCE(?, '')
					AND class_subjects_id <> ?
					AND class_subjects_deleted_at IS NULL
				`, masjidID, newClassID, newSubjectID, newAcademicYear, m.ClassSubjectsID).
				Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi class subject")
			}
			if cnt > 0 {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+tahun ajaran sudah terdaftar")
			}
		}

		// Apply & update timestamp
		req.Apply(&m)
		now := time.Now()
		m.ClassSubjectsUpdatedAt = &now

		patch := map[string]interface{}{
			"class_subjects_masjid_id":         m.ClassSubjectsMasjidID,
			"class_subjects_class_id":          m.ClassSubjectsClassID,
			"class_subjects_subject_id":        m.ClassSubjectsSubjectID,
			"class_subjects_order_index":       m.ClassSubjectsOrderIndex,
			"class_subjects_hours_per_week":    m.ClassSubjectsHoursPerWeek,
			"class_subjects_min_passing_score": m.ClassSubjectsMinPassingScore,
			"class_subjects_weight_on_report":  m.ClassSubjectsWeightOnReport,
			"class_subjects_is_core":           m.ClassSubjectsIsCore,
			"class_subjects_academic_year":     m.ClassSubjectsAcademicYear,
			"class_subjects_desc":              m.ClassSubjectsDesc,
			"class_subjects_is_active":         m.ClassSubjectsIsActive,
			"class_subjects_updated_at":        m.ClassSubjectsUpdatedAt,
		}

		if err := tx.Model(&csModel.ClassSubjectModel{}).
			Where("class_subjects_id = ?", m.ClassSubjectsID).
			Updates(patch).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Kombinasi kelas+subject+tahun ajaran sudah terdaftar")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
		}

		c.Locals("updated_class_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("updated_class_subject").(csModel.ClassSubjectModel)
	return helper.JsonUpdated(c, "Class subject berhasil diperbarui", csDTO.FromClassSubjectModel(m))
}

/* =========================================================
   DELETE
   DELETE /admin/class-subjects/:id?force=true
   ========================================================= */
func (h *ClassSubjectController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}

	// Hanya admin yang boleh hard delete
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	isAdmin := adminMasjidID != uuid.Nil && adminMasjidID == masjidID
	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var m csModel.ClassSubjectModel
		if err := tx.First(&m, "class_subjects_id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if m.ClassSubjectsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus data milik masjid lain")
		}

		if force {
			if err := tx.Delete(&csModel.ClassSubjectModel{}, "class_subjects_id = ?", id).Error; err != nil {
				msg := strings.ToLower(err.Error())
				if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
					return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		} else {
			if m.ClassSubjectsDeletedAt != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Data sudah dihapus")
			}
			now := time.Now()
			if err := tx.Model(&csModel.ClassSubjectModel{}).
				Where("class_subjects_id = ?", id).
				Updates(map[string]interface{}{
					"class_subjects_deleted_at": &now,
					"class_subjects_updated_at": &now,
				}).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
			}
		}

		c.Locals("deleted_class_subject", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("deleted_class_subject").(csModel.ClassSubjectModel)
	return helper.JsonDeleted(c, "Class subject berhasil dihapus", csDTO.FromClassSubjectModel(m))
}
