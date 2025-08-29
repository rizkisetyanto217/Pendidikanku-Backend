// internals/features/lembaga/classes/sections/main/controller/class_section_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "masjidku_backend/internals/helpers"

	"masjidku_backend/internals/features/lembaga/stats/lembaga_stats/service"
	ucsDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	classModel "masjidku_backend/internals/features/school/classes/classes/model"
)

type ClassSectionController struct {
	DB *gorm.DB
}

func NewClassSectionController(db *gorm.DB) *ClassSectionController {
	return &ClassSectionController{DB: db}
}

var validate = validator.New()

/* ================= Handlers (ADMIN) ================= */

// GET /admin/class-sections/:id
func (ctrl *ClassSectionController) GetClassSectionByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m secModel.ClassSectionModel
	if err := ctrl.DB.First(&m, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassSectionsMasjidID == nil || *m.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses section milik masjid lain")
	}
	return helper.JsonOK(c, "OK", ucsDTO.NewClassSectionResponse(&m))
}

// GET /admin/class-sections/:id/participants
// Mengambil peserta yang TERDAFTAR (masih assigned) pada section tertentu.
// - Filter tenant by masjid
// - Hanya baris dengan user_class_sections_unassigned_at IS NULL
// - Enrich: user_classes.status/started_at/ended_at, users, users_profile
func (ctrl *ClassSectionController) ListRegisteredParticipants(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Paging opsional
	limit := 50
	offset := 0
	if v := c.QueryInt("limit"); v > 0 && v <= 200 {
		limit = v
	}
	if v := c.QueryInt("offset"); v >= 0 {
		offset = v
	}

	// Ambil user_class_sections yg masih aktif (belum di-unassign)
	var rows []secModel.UserClassSectionsModel
	if err := ctrl.DB.
		Model(&secModel.UserClassSectionsModel{}).
		Where("user_class_sections_masjid_id = ?", masjidID).
		Where("user_class_sections_section_id = ?", sectionID).
		Where("user_class_sections_unassigned_at IS NULL").
		Order("user_class_sections_assigned_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil peserta")
	}
	if len(rows) == 0 {
		return helper.JsonOK(c, "OK", []*ucsDTO.UserClassSectionResponse{})
	}

	/* ===== Enrichment (mirip ListUserClassSections) ===== */

	// 1) Kumpulkan user_class_id unik
	ucSet := make(map[uuid.UUID]struct{}, len(rows))
	userClassIDs := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		id := rows[i].UserClassSectionsUserClassID
		if _, ok := ucSet[id]; !ok {
			ucSet[id] = struct{}{}
			userClassIDs = append(userClassIDs, id)
		}
	}

	// 2) Ambil mapping user_class -> (user_id, status, started_at)
	type ucMeta struct {
		UserClassID uuid.UUID  `gorm:"column:user_classes_id"`
		UserID      uuid.UUID  `gorm:"column:user_classes_user_id"`
		Status      string     `gorm:"column:user_classes_status"`
		StartedAt   *time.Time `gorm:"column:user_classes_started_at"`
		// EndedAt sudah tidak ada
	}

	ucMetaByID := make(map[uuid.UUID]ucMeta, len(userClassIDs))
	userIDByUC := make(map[uuid.UUID]uuid.UUID, len(userClassIDs))

	{
		var ucRows []ucMeta
		if err := ctrl.DB.
			Table("user_classes").
			Select("user_classes_id, user_classes_user_id, user_classes_status, user_classes_started_at").
			Where("user_classes_id IN ?", userClassIDs).
			Find(&ucRows).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data enrolment")
		}
		for _, r := range ucRows {
			ucMetaByID[r.UserClassID] = r
			userIDByUC[r.UserClassID] = r.UserID
		}
	}


	// 3) Kumpulkan user_id unik
	uSet := make(map[uuid.UUID]struct{}, len(userClassIDs))
	userIDs := make([]uuid.UUID, 0, len(userClassIDs))
	for _, uc := range userClassIDs {
		if uid, ok := userIDByUC[uc]; ok {
			if _, seen := uSet[uid]; !seen {
				uSet[uid] = struct{}{}
				userIDs = append(userIDs, uid)
			}
		}
	}

	// 4) Ambil users -> map[user_id]UcsUser
	userMap := make(map[uuid.UUID]ucsDTO.UcsUser, len(userIDs))
	if len(userIDs) > 0 {
		var urs []struct {
			ID       uuid.UUID `gorm:"column:id"`
			UserName string    `gorm:"column:user_name"`
			Email    string    `gorm:"column:email"`
			IsActive bool      `gorm:"column:is_active"`
		}
		if err := ctrl.DB.
			Table("users").
			Select("id, user_name, email, is_active").
			Where("id IN ?", userIDs).
			Find(&urs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data user")
		}
		for _, u := range urs {
			userMap[u.ID] = ucsDTO.UcsUser{
				ID:       u.ID,
				UserName: u.UserName,
				Email:    u.Email,
				IsActive: u.IsActive,
			}
		}
	}

	// 5) Ambil users_profile -> map[user_id]UcsUserProfile
	profileMap := make(map[uuid.UUID]ucsDTO.UcsUserProfile, len(userIDs))
	if len(userIDs) > 0 {
		var prs []struct {
			UserID       uuid.UUID  `gorm:"column:user_id"`
			DonationName string     `gorm:"column:donation_name"`
			FullName     string     `gorm:"column:full_name"`
			FatherName   string     `gorm:"column:father_name"`
			MotherName   string     `gorm:"column:mother_name"`
			DateOfBirth  *time.Time `gorm:"column:date_of_birth"`
			Gender       *string    `gorm:"column:gender"`
			PhoneNumber  string     `gorm:"column:phone_number"`
			Bio          string     `gorm:"column:bio"`
			Location     string     `gorm:"column:location"`
			Occupation   string     `gorm:"column:occupation"`
		}
		if err := ctrl.DB.
			Table("users_profile").
			Select(`user_id, donation_name, full_name, father_name, mother_name,
			        date_of_birth, gender, phone_number, bio, location, occupation`).
			Where("user_id IN ?", userIDs).
			Where("deleted_at IS NULL").
			Find(&prs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data profile")
		}
		for _, p := range prs {
			profileMap[p.UserID] = ucsDTO.UcsUserProfile{
				UserID:       p.UserID,
				DonationName: p.DonationName,
				FullName:     p.FullName,
				FatherName:   p.FatherName,
				MotherName:   p.MotherName,
				DateOfBirth:  p.DateOfBirth,
				Gender:       p.Gender,
				PhoneNumber:  p.PhoneNumber,
				Bio:          p.Bio,
				Location:     p.Location,
				Occupation:   p.Occupation,
			}
		}
	}

	// 6) Build response
	resp := make([]*ucsDTO.UserClassSectionResponse, 0, len(rows))
	for i := range rows {
		r := ucsDTO.NewUserClassSectionResponse(&rows[i])

		ucID := rows[i].UserClassSectionsUserClassID
		if meta, ok := ucMetaByID[ucID]; ok {
			r.UserClassesStatus = meta.Status
		}
		if uid, ok := userIDByUC[ucID]; ok {
			if u, ok := userMap[uid]; ok {
				uCopy := u
				r.User = &uCopy
			}
			if p, ok := profileMap[uid]; ok {
				pCopy := p
				r.Profile = &pCopy
			}
		}
		resp = append(resp, r)
	}

	return helper.JsonOK(c, "OK", resp)
}



// GET /admin/class-sections
func (ctrl *ClassSectionController) ListClassSections(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q ucsDTO.ListClassSectionQuery
	q.Limit = 20
	q.Offset = 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_deleted_at IS NULL").
		Where("class_sections_masjid_id = ?", masjidID)

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

	// ---- Kumpulkan teacher IDs unik ----
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

	// ---- Ambil data users (guru) ----
	userMap := map[uuid.UUID]ucsDTO.UserLite{}
	if len(teacherIDs) > 0 {
		// pakai model users kamu; minimal kolom id, user_name, email, is_active
		type userRow struct {
			ID       uuid.UUID `gorm:"column:id"`
			UserName string    `gorm:"column:user_name"`
			Email    string    `gorm:"column:email"`
			IsActive bool      `gorm:"column:is_active"`
		}
		var urs []userRow
		if err := ctrl.DB.
			Table("users").
			Select("id, user_name, email, is_active").
			Where("id IN ?", teacherIDs).
			Find(&urs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data guru")
		}
		for _, u := range urs {
			userMap[u.ID] = ucsDTO.UserLite{
				ID:       u.ID,
				UserName: u.UserName,
				Email:    u.Email,
				IsActive: u.IsActive,
			}
		}
	}

	// ---- Bangun response + embed teacher ----
	out := make([]*ucsDTO.ClassSectionResponse, 0, len(rows))
	for i := range rows {
		var t *ucsDTO.UserLite
		if rows[i].ClassSectionsTeacherID != nil {
			if ul, ok := userMap[*rows[i].ClassSectionsTeacherID]; ok {
				uCopy := ul
				t = &uCopy
			}
		}
		resp := ucsDTO.NewClassSectionResponse(&rows[i])
		resp.Teacher = t
		out = append(out, resp)
	}

	return helper.JsonOK(c, "OK", out)
}


// GET /admin/class-sections/:id/books
func (ctrl *ClassSectionController) ListBooksBySection(c *fiber.Ctx) error {
    masjidID, err := helper.GetMasjidIDFromToken(c)
    if err != nil { return err }

    sectionID, err := uuid.Parse(c.Params("id"))
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
        TeacherUserID       *uuid.UUID `json:"teacher_user_id,omitempty"` // enrichment
    }

    var out []row
    q := ctrl.DB.
        Table("class_sections AS sect").
        // ❗️SST tidak jadi syarat; hanya LEFT JOIN untuk info guru
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
                AND csb.class_subject_books_masjid_id = ?`, masjidID).
        Joins(`JOIN books AS b
                 ON b.books_id = csb.class_subject_books_book_id
                AND b.books_deleted_at IS NULL
                AND b.books_masjid_id = ?`, masjidID).
        Joins(`LEFT JOIN class_section_subject_teachers AS sst
                 ON sst.class_section_subject_teachers_section_id = sect.class_sections_id
                AND sst.class_section_subject_teachers_subject_id = cs.class_subjects_subject_id
                AND sst.class_section_subject_teachers_deleted_at IS NULL
                AND sst.class_section_subject_teachers_is_active = TRUE`).
        Where(`sect.class_sections_id = ?
               AND sect.class_sections_deleted_at IS NULL
               AND sect.class_sections_masjid_id = ?`,
            sectionID, masjidID)

    if err := q.Scan(&out).Error; err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil buku")
    }
    // Kalau tidak ada baris, out = nil (bisa kamu ubah ke [] jika mau)
    return helper.JsonOK(c, "OK", out)
}


// GET /admin/class-sections/search
// Query:
//   - q                : string (wajib, min 2 huruf)
//   - limit            : int (default 10, max 50)
//   - offset           : int (default 0)
//   - active_only      : bool (opsional)
//   - class_id         : uuid (opsional)
//   - teacher_id       : uuid (opsional)
//   - enrich_teacher   : bool (default true) -> embed data guru seperti ListClassSections
func (ctrl *ClassSectionController) SearchClassSections(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
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
		Where("class_sections_masjid_id = ?", masjidID)

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

	// Tanpa enrichment guru
	if !enrichTeacher || len(rows) == 0 {
		out := make([]*ucsDTO.ClassSectionResponse, 0, len(rows))
		for i := range rows {
			out = append(out, ucsDTO.NewClassSectionResponse(&rows[i]))
		}
		return helper.JsonOK(c, "OK", out)
	}

	// ---- Enrichment guru (mirip ListClassSections) ----
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

	userMap := map[uuid.UUID]ucsDTO.UserLite{}
	if len(teacherIDs) > 0 {
		type userRow struct {
			ID       uuid.UUID `gorm:"column:id"`
			UserName string    `gorm:"column:user_name"`
			Email    string    `gorm:"column:email"`
			IsActive bool      `gorm:"column:is_active"`
		}
		var urs []userRow
		if err := ctrl.DB.
			Table("users").
			Select("id, user_name, email, is_active").
			Where("id IN ?", teacherIDs).
			Find(&urs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data guru")
		}
		for _, u := range urs {
			userMap[u.ID] = ucsDTO.UserLite{
				ID:       u.ID,
				UserName: u.UserName,
				Email:    u.Email,
				IsActive: u.IsActive,
			}
		}
	}

	out := make([]*ucsDTO.ClassSectionResponse, 0, len(rows))
	for i := range rows {
		var t *ucsDTO.UserLite
		if rows[i].ClassSectionsTeacherID != nil {
			if ul, ok := userMap[*rows[i].ClassSectionsTeacherID]; ok {
				uCopy := ul
				t = &uCopy
			}
		}
		resp := ucsDTO.NewClassSectionResponse(&rows[i])
		resp.Teacher = t
		out = append(out, resp)
	}

	return helper.JsonOK(c, "OK", out)
}


// POST /admin/class-sections
func (ctrl *ClassSectionController) CreateClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req ucsDTO.CreateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// force tenant
	req.ClassSectionsMasjidID = &masjidID

	// === AUTO SLUG ===
	if strings.TrimSpace(req.ClassSectionsSlug) == "" {
		req.ClassSectionsSlug = helper.GenerateSlug(req.ClassSectionsName)
	} else {
		req.ClassSectionsSlug = helper.GenerateSlug(req.ClassSectionsSlug)
	}
	if req.ClassSectionsSlug == "" {
		req.ClassSectionsSlug = "section-" + uuid.NewString()[:8]
	}

	// Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Mapping ke model
	m := req.ToModel()

	// === TRANSACTION START ===
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Cek unik slug (lock ringan biar anti-race)
	if err := tx.
		Clauses(clause.Locking{Strength: "SHARE"}).
		Where("class_sections_slug = ? AND class_sections_deleted_at IS NULL", m.ClassSectionsSlug).
		First(&secModel.ClassSectionModel{}).Error; err == nil {
		// ada row ⇒ slug sudah dipakai
		tx.Rollback()
		return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// error lain (bukan "tidak ditemukan")
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Simpan section
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat section")
	}

	// === Update lembaga_stats: +1 jika section AKTIF ===
	// Asumsi field boolean di model: m.ClassSectionsIsActive
	if m.ClassSectionsIsActive {
		statsSvc := service.NewLembagaStatsService()
		if err := statsSvc.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := statsSvc.IncActiveSections(tx, masjidID, +1); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// === TRANSACTION END ===

	return helper.JsonCreated(c, "Section berhasil dibuat", ucsDTO.NewClassSectionResponse(m))
}


// PUT /admin/class-sections/:id
func (ctrl *ClassSectionController) UpdateClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// Parse & normalisasi payload lebih dulu
	var req ucsDTO.UpdateClassSectionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Normalisasi slug bila dikirim; atau auto dari name bila name dikirim
	if req.ClassSectionsSlug != nil {
		s := helper.GenerateSlug(*req.ClassSectionsSlug)
		if s == "" {
			s = "section-" + uuid.NewString()[:8]
		}
		req.ClassSectionsSlug = &s
	} else if req.ClassSectionsName != nil {
		s := helper.GenerateSlug(*req.ClassSectionsName)
		if s == "" {
			s = "section-" + uuid.NewString()[:8]
		}
		req.ClassSectionsSlug = &s
	}

	// Cegah ganti tenant dari luar
	req.ClassSectionsMasjidID = &masjidID

	// Validasi payload
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ===== TRANSACTION START =====
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Ambil existing + LOCK (hindari race)
	var existing secModel.ClassSectionModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&existing, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Guard tenant
	if existing.ClassSectionsMasjidID == nil || *existing.ClassSectionsMasjidID != masjidID {
		tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah section milik masjid lain")
	}
	
	// Jika class_id diganti, validasi class milik tenant
	if req.ClassSectionsClassID != nil {
		var cls classModel.ClassModel
		if err := tx.
			Select("class_id, class_masjid_id").
			First(&cls, "class_id = ? AND class_deleted_at IS NULL", *req.ClassSectionsClassID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Class tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi class")
		}

		// cls.ClassMasjidID adalah uuid.UUID (bukan *uuid.UUID), jadi langsung compare
		if cls.ClassMasjidID != masjidID {
			tx.Rollback()
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh memindahkan section ke class milik masjid lain")
		}
	}

	// Cek unik slug (exclude current)
	if req.ClassSectionsSlug != nil && *req.ClassSectionsSlug != existing.ClassSectionsSlug {
		var cnt int64
		if err := tx.Model(&secModel.ClassSectionModel{}).
			Where("class_sections_slug = ? AND class_sections_id <> ? AND class_sections_deleted_at IS NULL",
				*req.ClassSectionsSlug, existing.ClassSectionsID).
			Count(&cnt).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		} else if cnt > 0 {
			tx.Rollback()
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
		}
	}

	// Cek unik (class_id, name) exclude current
	targetClassID := existing.ClassSectionsClassID
	if req.ClassSectionsClassID != nil {
		targetClassID = *req.ClassSectionsClassID
	}
	targetName := existing.ClassSectionsName
	if req.ClassSectionsName != nil {
		targetName = *req.ClassSectionsName
	}
	{
		var cnt int64
		if err := tx.Model(&secModel.ClassSectionModel{}).
			Where(`class_sections_class_id = ?
			       AND class_sections_name = ?
			       AND class_sections_id <> ?
			       AND class_sections_deleted_at IS NULL`,
				targetClassID, targetName, existing.ClassSectionsID).
			Count(&cnt).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		} else if cnt > 0 {
			tx.Rollback()
			return fiber.NewError(fiber.StatusConflict, "Nama section sudah dipakai pada class ini")
		}
	}

	// Hitung perubahan status aktif
	wasActive := existing.ClassSectionsIsActive
	newActive := wasActive
	if req.ClassSectionsIsActive != nil {
		newActive = *req.ClassSectionsIsActive
	}

	// Apply & save
	req.ApplyToModel(&existing)
	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", existing.ClassSectionsID).
		Updates(&existing).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui section")
	}

	// Sinkronkan lembaga_stats jika status aktif berubah
	if wasActive != newActive {
		stats := service.NewLembagaStatsService()
		// pastikan baris stats ada (idempotent)
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		delta := -1
		if newActive {
			delta = +1
		}
		if err := stats.IncActiveSections(tx, masjidID, delta); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// ===== TRANSACTION END =====

	return helper.JsonUpdated(c, "Section berhasil diperbarui", ucsDTO.NewClassSectionResponse(&existing))
}





// DELETE /admin/class-sections/:id  (soft delete)
func (ctrl *ClassSectionController) SoftDeleteClassSection(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	sectionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Lock row agar anti race & pastikan belum soft-deleted
	var m secModel.ClassSectionModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&m, "class_sections_id = ? AND class_sections_deleted_at IS NULL", sectionID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Guard tenant
	if m.ClassSectionsMasjidID == nil || *m.ClassSectionsMasjidID != masjidID {
		tx.Rollback()
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus section milik masjid lain")
	}

	// simpan status aktif sebelum delete
	wasActive := m.ClassSectionsIsActive

	// Soft delete
	now := time.Now()
	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_id = ?", m.ClassSectionsID).
		Updates(map[string]any{
			"class_sections_deleted_at": now,
			"class_sections_is_active":  false,
			"class_sections_updated_at": now,
		}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus section")
	}

	// Decrement stats hanya jika sebelumnya aktif
	if wasActive {
		stats := service.NewLembagaStatsService()
		// pastikan baris stats ada
		if err := stats.EnsureForMasjid(tx, masjidID); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal inisialisasi lembaga_stats: "+err.Error())
		}
		if err := stats.IncActiveSections(tx, masjidID, -1); err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal update lembaga_stats: "+err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Section berhasil dihapus", fiber.Map{
		"class_sections_id": m.ClassSectionsID,
	})
}
