package controller

import (
	"errors"
	"strings"
	"time"

	ucsDTO "masjidku_backend/internals/features/school/classes/class_sections/dto"
	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ================= List ================= */

// GET /admin/class-sections
func (ctrl *ClassSectionController) ListClassSections(c *fiber.Ctx) error {
	// =========================
	// Multi-tenant read (semua klaim masjid)
	// =========================
	masjidIDs, err := helperAuth.GetMasjidIDsFromToken(c)
	if err != nil {
		return err
	}

	// =========================
	// Search term (gabungan q & search)
	// =========================
	rawQ := strings.TrimSpace(c.Query("q"))
	rawSearch := strings.TrimSpace(c.Query("search"))
	searchTerm := rawSearch
	if rawQ != "" {
		searchTerm = rawQ
		if len([]rune(searchTerm)) < 2 {
			return fiber.NewError(fiber.StatusBadRequest, "Parameter q minimal 2 karakter")
		}
	}

	// =========================
	// Pagination & sorting (dynamic default)
	// =========================
	defaultSortBy := "created_at"
	defaultSortOrder := "desc"
	if searchTerm != "" {
		defaultSortBy = "name"
		defaultSortOrder = "asc"
	}
	p := helper.ParseFiber(c, defaultSortBy, defaultSortOrder, helper.AdminOpts)

	// Kolom yang diizinkan untuk sort
	allowed := map[string]string{
		"name":       "class_sections_name",
		"created_at": "class_sections_created_at",
	}
	orderClause, _ := helper.Params{SortBy: p.SortBy, SortOrder: p.SortOrder}.SafeOrderClause(allowed, defaultSortBy)
	orderClause = strings.TrimPrefix(orderClause, "ORDER BY ")

	// =========================
	// Parse includes
	// =========================
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeAll := includeStr == "all"
	includes := map[string]bool{}
	for _, part := range strings.Split(includeStr, ",") {
		if s := strings.TrimSpace(part); s != "" {
			includes[s] = true
		}
	}
	wantClass := includeAll || includes["class"] || includes["classes"]
	wantRoom := includeAll || includes["room"] || includes["rooms"]
	wantTeacher := includeAll || includes["teacher"] || includes["teachers"]
	wantSubjects := includeAll || includes["subject"] || includes["subjects"]
	wantBooks := includeAll || includes["book"] || includes["books"]
	if wantBooks {
		wantSubjects = true // books implies subjects
	}
	// UCS includes
	wantUCS := includeAll ||
		includes["user_class_sections"] ||
		includes["ucs"] ||
		includes["placements"] ||
		includes["user_class_sections_all"]
	// Ambil semua riwayat jika:
	// - include=user_class_sections_all, atau
	// - ucs_all=true, atau
	// - include=all
	wantUCSAll := includeAll || includes["user_class_sections_all"] || c.QueryBool("ucs_all")

	// =========================
	// Filters
	// =========================
	var (
		classID, teacherID, roomID *uuid.UUID
		activeOnly                 *bool
		sectionIDs                 []uuid.UUID // filter by id (comma-separated supported)
	)

	if s := strings.TrimSpace(c.Query("id")); s != "" {
		ids, err := parseUUIDList(s)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "id tidak valid: "+err.Error())
		}
		sectionIDs = ids
	}
	if s := strings.TrimSpace(c.Query("class_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			classID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "class_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("teacher_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			teacherID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "teacher_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("room_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			roomID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "room_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.Query("active_only")); s != "" {
		b := c.QueryBool("active_only")
		activeOnly = &b
	}

	// =========================
	// Base query
	// =========================
	tx := ctrl.DB.Model(&secModel.ClassSectionModel{}).
		Where("class_sections_deleted_at IS NULL").
		Where("class_sections_masjid_id IN ?", masjidIDs)

	if len(sectionIDs) > 0 {
		tx = tx.Where("class_sections_id IN ?", sectionIDs)
	}
	if activeOnly != nil {
		tx = tx.Where("class_sections_is_active = ?", *activeOnly)
	}
	if classID != nil {
		tx = tx.Where("class_sections_class_id = ?", *classID)
	}
	if teacherID != nil {
		tx = tx.Where("class_sections_teacher_id = ?", *teacherID)
	}
	if roomID != nil {
		tx = tx.Where("class_sections_class_room_id = ?", *roomID)
	}
	if searchTerm != "" {
		s := "%" + strings.ToLower(searchTerm) + "%"
		tx = tx.Where(`LOWER(class_sections_name) LIKE ?
		               OR LOWER(class_sections_code) LIKE ?
		               OR LOWER(class_sections_slug) LIKE ?`, s, s, s)
	}

	// =========================
	// Total count
	// =========================
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	// =========================
	// Fetch rows
	// =========================
	if orderClause != "" {
		tx = tx.Order(orderClause)
	}
	if !p.All {
		tx = tx.Limit(p.Limit()).Offset(p.Offset())
	}

	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Short-circuit bila kosong
	if len(rows) == 0 {
		meta := helper.BuildMeta(total, p)
		return helper.JsonOK(c, "OK", fiber.Map{
			"data": []*struct{}{},
			"meta": meta,
		})
	}

	// =========================
	// Prefetch TEACHER → users (batched)
	// =========================
	// Gunakan tipe lokal agar tidak tergantung DTO punya UserLite
	type userLite struct {
		ID       uuid.UUID `json:"id"`
		UserName string    `json:"user_name"`
		FullName string    `json:"full_name"`
		Email    string    `json:"email"`
		IsActive bool      `json:"is_active"`
	}

	teacherToUser := make(map[uuid.UUID]uuid.UUID) // masjid_teacher_id -> users.id
	userMap := map[uuid.UUID]userLite{}            // users.id -> user lite
	if wantTeacher {
		tSet := make(map[uuid.UUID]struct{})
		for i := range rows {
			if rows[i].ClassSectionsTeacherID != nil {
				tSet[*rows[i].ClassSectionsTeacherID] = struct{}{}
			}
		}
		if len(tSet) > 0 {
			teacherIDs := make([]uuid.UUID, 0, len(tSet))
			for id := range tSet {
				teacherIDs = append(teacherIDs, id)
			}
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
			// fallback skema lama (kompat)
			for _, tid := range teacherIDs {
				if _, ok := found[tid]; !ok {
					teacherToUser[tid] = tid
				}
			}
			uSet := make(map[uuid.UUID]struct{}, len(teacherToUser))
			for _, uid := range teacherToUser {
				uSet[uid] = struct{}{}
			}
			if len(uSet) > 0 {
				userIDs := make([]uuid.UUID, 0, len(uSet))
				for uid := range uSet {
					userIDs = append(userIDs, uid)
				}
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
					userMap[u.ID] = userLite{
						ID:       u.ID,
						UserName: u.UserName,
						FullName: full,
						Email:    u.Email,
						IsActive: u.IsActive,
					}
				}
			}
		}
	}

	// =========================
	// Prefetch CLASSES (nama dari class_parent)
	// =========================
	type classLite struct {
		ID   uuid.UUID `json:"id"   gorm:"column:id"`
		Name string    `json:"name" gorm:"column:name"`
		Slug string    `json:"slug,omitempty" gorm:"column:slug"`
	}
	classMap := map[uuid.UUID]classLite{}
	if wantClass {
		cSet := map[uuid.UUID]struct{}{}
		for i := range rows {
			if rows[i].ClassSectionsClassID != uuid.Nil {
				cSet[rows[i].ClassSectionsClassID] = struct{}{}
			}
		}
		if len(cSet) > 0 {
			classIDs := make([]uuid.UUID, 0, len(cSet))
			for id := range cSet {
				classIDs = append(classIDs, id)
			}
			var cr []classLite
			if err := ctrl.DB.
				Table("classes AS c").
				Select(`
					c.class_id AS id,
					cp.class_parent_name AS name,
					c.class_slug AS slug
				`).
				Joins(`JOIN public.class_parents AS cp
						ON cp.class_parent_id = c.class_parent_id
						AND cp.class_parent_masjid_id = c.class_masjid_id
						AND cp.class_parent_deleted_at IS NULL`).
				Where("c.class_id IN ? AND c.class_deleted_at IS NULL", classIDs).
				Find(&cr).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data classes")
			}
			for _, r := range cr {
				classMap[r.ID] = r
			}
		}
	}

	// =========================
	// Prefetch ROOMS
	// =========================
	type roomLite struct {
		ID   uuid.UUID `json:"id"   gorm:"column:id"`
		Name string    `json:"name" gorm:"column:name"`
	}
	roomMap := map[uuid.UUID]roomLite{}
	if wantRoom {
		rSet := map[uuid.UUID]struct{}{}
		for i := range rows {
			if rows[i].ClassSectionsClassRoomID != nil && *rows[i].ClassSectionsClassRoomID != uuid.Nil {
				rSet[*rows[i].ClassSectionsClassRoomID] = struct{}{}
			}
		}
		if len(rSet) > 0 {
			roomIDs := make([]uuid.UUID, 0, len(rSet))
			for id := range rSet {
				roomIDs = append(roomIDs, id)
			}
			type rrRow struct {
				ID   uuid.UUID `gorm:"column:id"`
				Name string    `gorm:"column:name"`
			}
			var rowsRR []rrRow
			if err := ctrl.DB.
				Table("class_rooms").
				Select("class_room_id AS id, class_rooms_name AS name").
				Where("class_room_id IN ? AND class_rooms_deleted_at IS NULL", roomIDs).
				Find(&rowsRR).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data rooms")
			}
			for _, r := range rowsRR {
				roomMap[r.ID] = roomLite(r)
			}
		}
	}

	// =========================
	// Prefetch SUBJECTS & BOOKS (tenant-safe, batched)
	// =========================
	type bookLite struct {
		ID     uuid.UUID `json:"id"    gorm:"column:id"`
		Title  string    `json:"title" gorm:"column:title"`
		Author *string   `json:"author,omitempty" gorm:"column:author"`
	}
	type subjectLite struct {
		ClassSubjectID uuid.UUID  `json:"class_subject_id" gorm:"column:class_subject_id"`
		SubjectID      uuid.UUID  `json:"subject_id"       gorm:"column:subject_id"`
		SubjectName    string     `json:"subject_name"     gorm:"column:subject_name"`
		SubjectCode    *string    `json:"subject_code,omitempty" gorm:"column:subject_code"`
		Books          []bookLite `json:"books,omitempty"`
	}

	subjectsByClass := map[uuid.UUID][]subjectLite{} // class_id -> []subjectLite
	if wantSubjects {
		classIDSet := map[uuid.UUID]struct{}{}
		for i := range rows {
			if rows[i].ClassSectionsClassID != uuid.Nil {
				classIDSet[rows[i].ClassSectionsClassID] = struct{}{}
			}
		}
		if len(classIDSet) > 0 {
			classIDs := make([]uuid.UUID, 0, len(classIDSet))
			for id := range classIDSet {
				classIDs = append(classIDs, id)
			}

			// subjects
			var srows []struct {
				ClassID        uuid.UUID `gorm:"column:class_id"`
				ClassSubjectID uuid.UUID `gorm:"column:class_subject_id"`
				SubjectID      uuid.UUID `gorm:"column:subject_id"`
				SubjectName    string    `gorm:"column:subject_name"`
				SubjectCode    *string   `gorm:"column:subject_code"`
			}
			if err := ctrl.DB.
				Table("class_subjects AS cs").
				Select(`
					cs.class_subjects_class_id AS class_id,
					cs.class_subjects_id        AS class_subject_id,
					s.subjects_id               AS subject_id,
					s.subjects_name             AS subject_name,
					s.subjects_code             AS subject_code
				`).
				Joins(`JOIN subjects AS s
						ON s.subjects_id = cs.class_subjects_subject_id
					   AND s.subjects_deleted_at IS NULL`).
				Where("cs.class_subjects_class_id IN ?", classIDs).
				Where("cs.class_subjects_deleted_at IS NULL").
				Where("cs.class_subjects_masjid_id IN ?", masjidIDs).
				Find(&srows).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil subjects")
			}
			type subjIdx struct{ ClassID uuid.UUID; Idx int }
			locator := map[uuid.UUID]subjIdx{}
			for _, sr := range srows {
				sub := subjectLite{
					ClassSubjectID: sr.ClassSubjectID,
					SubjectID:      sr.SubjectID,
					SubjectName:    sr.SubjectName,
					SubjectCode:    sr.SubjectCode,
				}
				subjectsByClass[sr.ClassID] = append(subjectsByClass[sr.ClassID], sub)
				locator[sr.ClassSubjectID] = subjIdx{ClassID: sr.ClassID, Idx: len(subjectsByClass[sr.ClassID]) - 1}
			}

			// books (optional)
			if wantBooks && len(locator) > 0 {
				classSubjectIDs := make([]uuid.UUID, 0, len(locator))
				for csid := range locator {
					classSubjectIDs = append(classSubjectIDs, csid)
				}
				var brows []struct {
					ClassSubjectID uuid.UUID `gorm:"column:class_subject_id"`
					ID             uuid.UUID `gorm:"column:id"`
					Title          string    `gorm:"column:title"`
					Author         *string   `gorm:"column:author"`
				}
				if err := ctrl.DB.
					Table("class_subject_books AS csb").
					Select(`
						csb.class_subject_books_class_subject_id AS class_subject_id,
						b.books_id     AS id,
						b.books_title  AS title,
						b.books_author AS author
					`).
					Joins(`JOIN books AS b
							ON b.books_id = csb.class_subject_books_book_id
						   AND b.books_deleted_at IS NULL`).
					Where("csb.class_subject_books_class_subject_id IN ?", classSubjectIDs).
					Where("csb.class_subject_books_deleted_at IS NULL").
					Where("csb.class_subject_books_is_active = TRUE").
					Where("csb.class_subject_books_masjid_id IN ?", masjidIDs).
					Find(&brows).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil books")
				}
				for _, br := range brows {
					if bucket, ok := locator[br.ClassSubjectID]; ok {
						sl := subjectsByClass[bucket.ClassID]
						sl[bucket.Idx].Books = append(sl[bucket.Idx].Books, bookLite{
							ID:     br.ID,
							Title:  br.Title,
							Author: br.Author,
						})
						subjectsByClass[bucket.ClassID] = sl
					}
				}
			}
		}
	}

	// =========================
	// Prefetch USER_CLASS_SECTIONS
	// =========================
	type userClassSectionLite struct {
		ID           uuid.UUID  `json:"id"             gorm:"column:id"`
		UserClassID  uuid.UUID  `json:"user_class_id"  gorm:"column:user_class_id"`
		SectionID    uuid.UUID  `json:"section_id"     gorm:"column:section_id"`
		AssignedAt   time.Time  `json:"assigned_at"    gorm:"column:assigned_at"`
		UnassignedAt *time.Time `json:"unassigned_at"  gorm:"column:unassigned_at"`
		IsActive     bool       `json:"is_active"      gorm:"column:is_active"`
	}
	ucsBySection := map[uuid.UUID][]userClassSectionLite{}
	if wantUCS {
		secSet := make(map[uuid.UUID]struct{}, len(rows))
		for i := range rows {
			secSet[rows[i].ClassSectionsID] = struct{}{}
		}
		if len(secSet) > 0 {
			secIDs := make([]uuid.UUID, 0, len(secSet))
			for id := range secSet {
				secIDs = append(secIDs, id)
			}
			var urows []userClassSectionLite
			q := ctrl.DB.
				Table("user_class_sections AS ucs").
				Select(`
					ucs.user_class_sections_id            AS id,
					ucs.user_class_sections_user_class_id AS user_class_id,
					ucs.user_class_sections_section_id    AS section_id,
					ucs.user_class_sections_assigned_at   AS assigned_at,
					ucs.user_class_sections_unassigned_at AS unassigned_at,
					(ucs.user_class_sections_unassigned_at IS NULL) AS is_active
				`).
				Where("ucs.user_class_sections_section_id IN ?", secIDs).
				Where("ucs.user_class_sections_masjid_id IN ?", masjidIDs).
				Where("ucs.user_class_sections_deleted_at IS NULL")

			// default: hanya aktif; kalau minta all → hilangkan filter aktif
			if !wantUCSAll {
				q = q.Where("ucs.user_class_sections_unassigned_at IS NULL")
			}

			if err := q.Find(&urows).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil user_class_sections")
			}
			for _, r := range urows {
				ucsBySection[r.SectionID] = append(ucsBySection[r.SectionID], r)
			}
		}
	}

	// =========================
	// Build response
	// =========================
	type sectionWithExpand struct {
		*ucsDTO.ClassSectionResponse `json:",inline"`
		Class             *classLite             `json:"class,omitempty"`
		Room              *roomLite              `json:"room,omitempty"`
		Teacher           *userLite              `json:"teacher,omitempty"`
		Subjects          []subjectLite          `json:"subjects,omitempty"`
		UserClassSections []userClassSectionLite `json:"user_class_sections,omitempty"`
	}

	out := make([]*sectionWithExpand, 0, len(rows))
	for i := range rows {
		var teacherPtr *userLite
		if wantTeacher && rows[i].ClassSectionsTeacherID != nil {
			if uid, ok := teacherToUser[*rows[i].ClassSectionsTeacherID]; ok {
				if ul, ok := userMap[uid]; ok {
					uCopy := ul
					teacherPtr = &uCopy
				}
			}
		}
		base := ucsDTO.FromModelClassSection(&rows[i])

		w := &sectionWithExpand{ClassSectionResponse: &base}
		if wantTeacher {
			w.Teacher = teacherPtr
		}
		if wantClass {
			if cl, ok := classMap[rows[i].ClassSectionsClassID]; ok {
				cCopy := cl
				w.Class = &cCopy
			}
		}
		if wantRoom && rows[i].ClassSectionsClassRoomID != nil {
			if rl, ok := roomMap[*rows[i].ClassSectionsClassRoomID]; ok {
				rCopy := rl
				w.Room = &rCopy
			}
		}
		if wantSubjects && rows[i].ClassSectionsClassID != uuid.Nil {
			if subs, ok := subjectsByClass[rows[i].ClassSectionsClassID]; ok && len(subs) > 0 {
				cp := make([]subjectLite, len(subs))
				copy(cp, subs)
				w.Subjects = cp
			}
		}
		if wantUCS {
			if list, ok := ucsBySection[rows[i].ClassSectionsID]; ok && len(list) > 0 {
				cp := make([]userClassSectionLite, len(list))
				copy(cp, list)
				w.UserClassSections = cp
			}
		}
		out = append(out, w)
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonOK(c, "OK", fiber.Map{
		"data": out,
		"meta": meta,
	})
}

/* ================= Helpers ================= */

// parseUUIDList mem-parse "a,b,c" → []uuid.UUID (dedupe + validasi)
func parseUUIDList(s string) ([]uuid.UUID, error) {
	parts := strings.Split(s, ",")
	seen := make(map[uuid.UUID]struct{}, len(parts))
	out := make([]uuid.UUID, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := uuid.Parse(p)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("daftar kosong")
	}
	return out, nil
}

/* ================= Get by Slug ================= */

// GET /admin/class-sections/slug/:slug
func (ctrl *ClassSectionController) GetClassSectionBySlug(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	slug := helper.GenerateSlug(c.Params("slug"))

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

	return helper.JsonOK(c, "OK", ucsDTO.FromModelClassSection(&m))
}
