package controller

import (
	"errors"
	ucsDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GET /admin/class-sections
// GET /admin/class-sections
func (ctrl *ClassSectionController) ListClassSections(c *fiber.Ctx) error {
	// 🔁 izinkan akses berdasarkan semua klaim masjid (teacher/DKM/admin/student)
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	// parse query (punya default)
	var q ucsDTO.ListClassSectionQuery
	q.Limit = 20
	q.Offset = 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_deleted_at IS NULL").
		Where("class_sections_masjid_id IN ?", masjidIDs)

	// ---- Filters ----
	if q.ActiveOnly != nil {
		tx = tx.Where("class_sections_is_active = ?", *q.ActiveOnly)
	}
	if q.ClassID != nil {
		tx = tx.Where("class_sections_class_id = ?", *q.ClassID)
	}
	if q.TeacherID != nil {
		tx = tx.Where("class_sections_teacher_id = ?", *q.TeacherID)
	}
	// NEW: filter by class room
	if q.RoomID != nil {
		tx = tx.Where("class_sections_class_room_id = ?", *q.RoomID)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
		tx = tx.Where("(LOWER(class_sections_name) LIKE ? OR LOWER(class_sections_code) LIKE ? OR LOWER(class_sections_slug) LIKE ?)", s, s, s)
	}

	// ---- Sorting ----
	sortVal := ""
	if q.Sort != nil {
		sortVal = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sortVal {
	case "name_asc":
		tx = tx.Order("class_sections_name ASC")
	case "name_desc":
		tx = tx.Order("class_sections_name DESC")
	case "created_at_asc":
		tx = tx.Order("class_sections_created_at ASC")
	default:
		tx = tx.Order("class_sections_created_at DESC")
	}

	// ---- Pagination ----
	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	// ---- Fetch sections ----
	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ---- Kumpulkan teacher IDs unik dari section ----
	teacherSet := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		if rows[i].ClassSectionsTeacherID != nil {
			teacherSet[*rows[i].ClassSectionsTeacherID] = struct{}{}
		}
	}
	teacherIDs := make([]uuid.UUID, 0, len(teacherSet))
	for id := range teacherSet {
		teacherIDs = append(teacherIDs, id)
	}

	// ---- Petakan class_sections_teacher_id -> users.id (tangani 2 skema) ----
	teacherToUser := make(map[uuid.UUID]uuid.UUID) // key: teacher_id dari section; val: users.id
	if len(teacherIDs) > 0 {
		type mtRow struct {
			ID     uuid.UUID `gorm:"column:masjid_teacher_id"`
			UserID uuid.UUID `gorm:"column:masjid_teacher_user_id"`
		}
		var mts []mtRow
		if err := ctrl.DB.
			Table("masjid_teachers").
			Select("masjid_teacher_id, masjid_teacher_user_id").
			Where("masjid_teacher_id IN ?", teacherIDs).
			Find(&mts).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data relasi pengajar")
		}
		found := make(map[uuid.UUID]struct{}, len(mts))
		for _, r := range mts {
			teacherToUser[r.ID] = r.UserID
			found[r.ID] = struct{}{}
		}
		for _, tid := range teacherIDs {
			if _, ok := found[tid]; !ok {
				teacherToUser[tid] = tid // fallback: skema lama users.id langsung
			}
		}
	}

	// ---- Ambil data users (guru) berdasarkan users.id ----
	userIDsSet := make(map[uuid.UUID]struct{}, len(teacherToUser))
	for _, uid := range teacherToUser {
		userIDsSet[uid] = struct{}{}
	}
	userIDs := make([]uuid.UUID, 0, len(userIDsSet))
	for uid := range userIDsSet {
		userIDs = append(userIDs, uid)
	}

	userMap := map[uuid.UUID]ucsDTO.UserLite{} // key: users.id
	if len(userIDs) > 0 {
		type userRow struct {
			ID       uuid.UUID `gorm:"column:id"`
			UserName string    `gorm:"column:user_name"`
			FullName *string   `gorm:"column:full_name"`
			Email    string    `gorm:"column:email"`
			IsActive bool      `gorm:"column:is_active"`
		}
		var urs []userRow
		if err := ctrl.DB.
			Table("users").
			Select("id, user_name, full_name, email, is_active").
			Where("id IN ?", userIDs).
			Find(&urs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data guru")
		}
		for _, u := range urs {
			full := ""
			if u.FullName != nil {
				full = *u.FullName
			}
			userMap[u.ID] = ucsDTO.UserLite{
				ID:       u.ID,
				UserName: u.UserName,
				FullName: full,
				Email:    u.Email,
				IsActive: u.IsActive,
			}
		}
	}

	// ---- Build response + embed teacher ----
	out := make([]*ucsDTO.ClassSectionResponse, 0, len(rows))
	for i := range rows {
		var t *ucsDTO.UserLite
		teacherName := ""

		if rows[i].ClassSectionsTeacherID != nil {
			if uid, ok := teacherToUser[*rows[i].ClassSectionsTeacherID]; ok {
				if ul, ok := userMap[uid]; ok {
					uCopy := ul
					t = &uCopy
					if ul.FullName != "" {
						teacherName = ul.FullName
					} else {
						teacherName = ul.UserName
					}
				}
			}
		}

		resp := ucsDTO.NewClassSectionResponse(&rows[i], teacherName)
		resp.Teacher = t
		out = append(out, resp)
	}

	return helper.JsonOK(c, "OK", out)
}

// GET /admin/class-sections/:id/books
func (ctrl *ClassSectionController) ListBooksBySection(c *fiber.Ctx) error {
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c) // 🔁 multi-tenant read
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	type row struct {
		BooksID             uuid.UUID  `json:"books_id"`
		BooksTitle          string     `json:"books_title"`
		BooksAuthor         *string    `json:"books_author,omitempty"`
		BooksDesc           *string    `json:"books_desc,omitempty"`
		BooksImageURL       *string    `json:"books_image_url,omitempty"`
		BooksURL            *string    `json:"books_url,omitempty"`
		BooksSlug           *string    `json:"books_slug,omitempty"`
		ClassSubjectsID     uuid.UUID  `json:"class_subjects_id"`
		SubjectsID          *uuid.UUID `json:"subjects_id,omitempty"`
		ClassSubjectBooksID uuid.UUID  `json:"class_subject_books_id"`
		TeacherUserID       *uuid.UUID `json:"teacher_user_id,omitempty"`
	}

	var out []row
	q := ctrl.DB.
		Table("class_sections AS sect").
		Select(`
			DISTINCT b.books_id, b.books_title, b.books_author, b.books_desc, b.books_image_url, b.books_url, b.books_slug,
			cs.class_subjects_id,
			cs.class_subjects_subject_id AS subjects_id,
			csb.class_subject_books_id,
			sst.class_section_subject_teachers_teacher_user_id AS teacher_user_id`).
		Joins(`JOIN class_subjects AS cs
				ON cs.class_subjects_class_id = sect.class_sections_class_id
				AND cs.class_subjects_deleted_at IS NULL`).
		Joins(`JOIN class_subject_books AS csb
				ON csb.class_subject_books_class_subject_id = cs.class_subjects_id
				AND csb.class_subject_books_deleted_at IS NULL
				AND csb.class_subject_books_is_active = TRUE
				AND csb.class_subject_books_masjid_id IN ?`, masjidIDs).
		Joins(`JOIN books AS b
				ON b.books_id = csb.class_subject_books_book_id
				AND b.books_deleted_at IS NULL
				AND b.books_masjid_id IN ?`, masjidIDs).
		Joins(`LEFT JOIN class_section_subject_teachers AS sst
				ON sst.class_section_subject_teachers_section_id = sect.class_sections_id
				AND sst.class_section_subject_teachers_subject_id = cs.class_subjects_subject_id
				AND sst.class_section_subject_teachers_deleted_at IS NULL
				AND sst.class_section_subject_teachers_is_active = TRUE`).
		Where(`sect.class_sections_id = ?
				AND sect.class_sections_deleted_at IS NULL
				AND sect.class_sections_masjid_id IN ?`,
			sectionID, masjidIDs)

	if err := q.Scan(&out).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil buku")
	}
	return helper.JsonOK(c, "OK", out)
}


// GET /admin/class-sections/search
func (ctrl *ClassSectionController) SearchClassSections(c *fiber.Ctx) error {
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c) // 🔁 multi-tenant read
	if err != nil {
		return err
	}

	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		return fiber.NewError(fiber.StatusBadRequest, "Parameter q minimal 2 karakter")
	}

	limit := 10
	offset := 0
	if v := c.QueryInt("limit"); v > 0 && v <= 50 {
		limit = v
	}
	if v := c.QueryInt("offset"); v >= 0 {
		offset = v
	}

	var activeOnly *bool
	if v := strings.TrimSpace(c.Query("active_only")); v != "" {
		b := c.QueryBool("active_only")
		activeOnly = &b
	}

	var classID *uuid.UUID
	if s := strings.TrimSpace(c.Query("class_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			classID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "class_id tidak valid")
		}
	}

	var teacherID *uuid.UUID
	if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			teacherID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
	}

	enrichTeacher := true
	if s := strings.TrimSpace(c.Query("enrich_teacher")); s != "" {
		enrichTeacher = c.QueryBool("enrich_teacher")
	}

	// ---- Base query (tenant + not deleted) ----
	tx := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_deleted_at IS NULL").
		Where("class_sections_masjid_id IN ?", masjidIDs)

	// ---- Filters ----
	if activeOnly != nil {
		tx = tx.Where("class_sections_is_active = ?", *activeOnly)
	}
	if classID != nil {
		tx = tx.Where("class_sections_class_id = ?", *classID)
	}
	if teacherID != nil {
		tx = tx.Where("class_sections_teacher_id = ?", *teacherID)
	}

	// ---- Search ----
	s := "%" + strings.ToLower(q) + "%"
	tx = tx.Where(`(LOWER(class_sections_name) LIKE ?
	               OR LOWER(class_sections_code) LIKE ?
	               OR LOWER(class_sections_slug) LIKE ?)`, s, s, s)

	// ---- Sort + Pagination ----
	tx = tx.Order("class_sections_name ASC").
		Limit(limit).
		Offset(offset)

	// ---- Fetch sections ----
	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mencari data")
	}

	if !enrichTeacher || len(rows) == 0 {
		out := make([]*ucsDTO.ClassSectionResponse, 0, len(rows))
		for i := range rows {
			out = append(out, ucsDTO.NewClassSectionResponse(&rows[i], ""))
		}
		return helper.JsonOK(c, "OK", out)
	}

	// ---- Enrichment guru (pakai skema ganda seperti ListClassSections) ----
	teacherSet := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		if rows[i].ClassSectionsTeacherID != nil {
			teacherSet[*rows[i].ClassSectionsTeacherID] = struct{}{}
		}
	}
	teacherIDs := make([]uuid.UUID, 0, len(teacherSet))
	for id := range teacherSet {
		teacherIDs = append(teacherIDs, id)
	}

	// map teacherId(section) -> users.id
	teacherToUser := make(map[uuid.UUID]uuid.UUID)
	if len(teacherIDs) > 0 {
		type mtRow struct {
			ID     uuid.UUID `gorm:"column:masjid_teacher_id"`
			UserID uuid.UUID `gorm:"column:masjid_teacher_user_id"`
		}
		var mts []mtRow
		if err := ctrl.DB.
			Table("masjid_teachers").
			Select("masjid_teacher_id, masjid_teacher_user_id").
			Where("masjid_teacher_id IN ?", teacherIDs).
			Find(&mts).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data relasi pengajar")
		}
		found := make(map[uuid.UUID]struct{}, len(mts))
		for _, r := range mts {
			teacherToUser[r.ID] = r.UserID
			found[r.ID] = struct{}{}
		}
		for _, tid := range teacherIDs {
			if _, ok := found[tid]; !ok {
				teacherToUser[tid] = tid // fallback skema lama
			}
		}
	}

	// ambil users
	userIDsSet := make(map[uuid.UUID]struct{}, len(teacherToUser))
	for _, uid := range teacherToUser {
		userIDsSet[uid] = struct{}{}
	}
	userIDs := make([]uuid.UUID, 0, len(userIDsSet))
	for uid := range userIDsSet {
		userIDs = append(userIDs, uid)
	}

	userMap := map[uuid.UUID]ucsDTO.UserLite{}
	if len(userIDs) > 0 {
		type userRow struct {
			ID       uuid.UUID `gorm:"column:id"`
			UserName string    `gorm:"column:user_name"`
			FullName *string   `gorm:"column:full_name"`
			Email    string    `gorm:"column:email"`
			IsActive bool      `gorm:"column:is_active"`
		}
		var urs []userRow
		if err := ctrl.DB.
			Table("users").
			Select("id, user_name, full_name, email, is_active").
			Where("id IN ?", userIDs).
			Find(&urs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data guru")
		}
		for _, u := range urs {
			full := ""
			if u.FullName != nil {
				full = *u.FullName
			}
			userMap[u.ID] = ucsDTO.UserLite{
				ID:       u.ID,
				UserName: u.UserName,
				FullName: full,
				Email:    u.Email,
				IsActive: u.IsActive,
			}
		}
	}

	// ---- Build response
	out := make([]*ucsDTO.ClassSectionResponse, 0, len(rows))
	for i := range rows {
		var t *ucsDTO.UserLite
		teacherName := ""

		if rows[i].ClassSectionsTeacherID != nil {
			if uid, ok := teacherToUser[*rows[i].ClassSectionsTeacherID]; ok {
				if ul, ok := userMap[uid]; ok {
					uCopy := ul
					t = &uCopy
					if ul.FullName != "" {
						teacherName = ul.FullName
					} else {
						teacherName = ul.UserName
					}
				}
			}
		}

		resp := ucsDTO.NewClassSectionResponse(&rows[i], teacherName)
		resp.Teacher = t
		out = append(out, resp)
	}

	return helper.JsonOK(c, "OK", out)
}



// GET /admin/class-sections/slug/:slug
// Mengambil data section berdasarkan slug dan memastikan data milik masjid yang valid
func (ctrl *ClassSectionController) GetClassSectionBySlug(c *fiber.Ctx) error {
	// Ambil masjid ID dari token user
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// Ambil slug dari URL params dan normalisasi
	slug := helper.GenerateSlug(c.Params("slug"))

	// Ambil data section milik masjid ini, slug case-insensitive, dan belum terhapus
	var m secModel.ClassSectionModel
	if err := ctrl.DB.
		Where("class_sections_masjid_id = ? AND lower(class_sections_slug) = lower(?) AND class_sections_deleted_at IS NULL",
			masjidID, slug).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Tidak perlu cek tenant lagi karena sudah difilter di query

	// Kembalikan response yang sudah diformat
	return helper.JsonOK(c, "OK", ucsDTO.NewClassSectionResponse(&m, ""))
}
