// file: internals/features/lembaga/classes/user_classes/main/controller/user_my_class_controller.go
package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "schoolku_backend/internals/features/school/classes/classes/dto"
	model "schoolku_backend/internals/features/school/classes/classes/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

// =====================================================
// GET /api/u/:school_id/classes/slug/:slug   (PUBLIC)
// =====================================================
func (ctrl *ClassController) GetClassBySlug(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB
	c.Locals("DB", ctrl.DB)

	// 1) Resolve school context secara publik
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err // helper sudah kasih 400 kalau konteks tak ada
	}

	var schoolID uuid.UUID
	if mc.ID != uuid.Nil {
		schoolID = mc.ID
	} else {
		id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
		if er != nil {
			if er == gorm.ErrRecordNotFound {
				return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
		}
		schoolID = id
	}

	// 2) Normalisasi slug kelas
	slug := helper.Slugify(c.Params("slug"), 120)

	// 3) Ambil data kelas (public)
	var m model.ClassModel
	if err := ctrl.DB.
		Where(`
			class_school_id = ?
			AND lower(class_slug) = lower(?)
			AND class_deleted_at IS NULL
		`, schoolID, slug).
		First(&m).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "Data diterima", dto.FromModel(&m))
}

// =====================================================
// GET /api/u/:school_id/classes/list        (PUBLIC)
// =====================================================
func (ctrl *ClassController) ListClasses(c *fiber.Ctx) error {
	// DB ke Locals agar helper bisa dipakai
	c.Locals("DB", ctrl.DB)

	// --- util: deteksi table parent (class_parents / class_parent) ---
	detectParentTable := func(db *gorm.DB) string {
		var reg *string
		db.Raw(`SELECT to_regclass('class_parents')::text`).Scan(&reg)
		if reg != nil && *reg != "" {
			return "class_parents"
		}
		reg = nil
		db.Raw(`SELECT to_regclass('class_parent')::text`).Scan(&reg)
		if reg != nil && *reg != "" {
			return "class_parent"
		}
		var name string
		row := db.Raw(`
			SELECT c.relname
			FROM pg_catalog.pg_class c
			JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
			WHERE c.relkind = 'r'
			  AND c.relname IN ('class_parents','class_parent')
			LIMIT 1
		`).Row()
		if err := row.Scan(&name); err == nil && name != "" {
			return name
		}
		reg = nil
		db.Raw(`SELECT to_regclass('public.class_parents')::text`).Scan(&reg)
		if reg != nil && *reg != "" {
			return "class_parents"
		}
		reg = nil
		db.Raw(`SELECT to_regclass('public.class_parent')::text`).Scan(&reg)
		if reg != nil && *reg != "" {
			return "class_parent"
		}
		return ""
	}

	// LEFT JOIN parent pakai kolom baru di tabel classes
	addParentJoin := func(tx *gorm.DB, aliasClass, parentTbl string) (*gorm.DB, bool) {
		if parentTbl == "" {
			return tx, false
		}
		j := fmt.Sprintf(`LEFT JOIN %s AS p
			ON p.class_parent_id = %s.class_class_parent_id
		   AND p.class_parent_deleted_at IS NULL`, parentTbl, aliasClass)
		return tx.Joins(j), true
	}

	// --- resolve tenant scope publik ---
	getTenantSchoolIDs := func() ([]uuid.UUID, error) {
		if mc, er := helperAuth.ResolveSchoolContext(c); er == nil {
			if mc.ID != uuid.Nil {
				return []uuid.UUID{mc.ID}, nil
			}
			if strings.TrimSpace(mc.Slug) != "" {
				id, er2 := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
				if er2 != nil {
					if er2 == gorm.ErrRecordNotFound {
						return nil, fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
					}
					return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal resolve school dari slug")
				}
				return []uuid.UUID{id}, nil
			}
		}
		if ids, err := helperAuth.GetSchoolIDsFromToken(c); err == nil && len(ids) > 0 {
			return ids, nil
		}

		var out []uuid.UUID
		if rawIDs := strings.TrimSpace(c.Query("school_id")); rawIDs != "" {
			for _, part := range strings.Split(rawIDs, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				id, e := uuid.Parse(part)
				if e != nil {
					return nil, fiber.NewError(fiber.StatusBadRequest, "school_id tidak valid")
				}
				out = append(out, id)
			}
		}
		if len(out) == 0 {
			if rawSlugs := strings.TrimSpace(c.Query("school_slug")); rawSlugs != "" {
				slugs := make([]string, 0, 4)
				for _, s := range strings.Split(rawSlugs, ",") {
					if s = strings.TrimSpace(s); s != "" {
						slugs = append(slugs, s)
					}
				}
				if len(slugs) > 0 {
					var ids []uuid.UUID
					if err := ctrl.DB.
						Table("schools").
						Select("school_id").
						Where("school_deleted_at IS NULL").
						Where("school_slug IN ?", slugs).
						Scan(&ids).Error; err != nil {
						return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca school_slug: "+err.Error())
					}
					out = append(out, ids...)
				}
			}
		}
		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "School context tidak ditemukan. Sertakan :school_id pada path atau header X-Active-School-ID / ?school_id / ?school_slug.")
		}
		return out, nil
	}

	// filter umum untuk alias class table
	applyCommonFilters := func(tx *gorm.DB, aliasClass string, q dto.ListClassQuery) *gorm.DB {
		tx = tx.Where(aliasClass + ".class_deleted_at IS NULL")

		// 🔁 pakai field baru dari DTO: ClassParentID & ClassTermID
		if q.ClassParentID != nil {
			tx = tx.Where(aliasClass+".class_class_parent_id = ?", *q.ClassParentID)
		}
		if q.ClassTermID != nil {
			tx = tx.Where(aliasClass+".class_academic_term_id = ?", *q.ClassTermID)
		}

		if raw := strings.TrimSpace(c.Query("id")); raw != "" {
			var ids []uuid.UUID
			for _, part := range strings.Split(raw, ",") {
				if part = strings.TrimSpace(part); part == "" {
					continue
				}
				id, err := uuid.Parse(part)
				if err != nil {
					return tx.Where("1=0")
				}
				ids = append(ids, id)
			}
			if len(ids) > 0 {
				tx = tx.Where(aliasClass+".class_id IN ?", ids)
			}
		} else if raw := strings.TrimSpace(c.Query("class_id")); raw != "" {
			var ids []uuid.UUID
			for _, part := range strings.Split(raw, ",") {
				if part = strings.TrimSpace(part); part == "" {
					continue
				}
				id, err := uuid.Parse(part)
				if err != nil {
					return tx.Where("1=0")
				}
				ids = append(ids, id)
			}
			if len(ids) > 0 {
				tx = tx.Where(aliasClass+".class_id IN ?", ids)
			}
		}
		if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
			tx = tx.Where(aliasClass+".class_status = ?", strings.ToLower(strings.TrimSpace(*q.Status)))
		}
		if q.DeliveryMode != nil && strings.TrimSpace(*q.DeliveryMode) != "" {
			tx = tx.Where("LOWER("+aliasClass+".class_delivery_mode) = LOWER(?)", strings.TrimSpace(*q.DeliveryMode))
		}
		if q.Slug != nil && strings.TrimSpace(*q.Slug) != "" {
			tx = tx.Where("LOWER("+aliasClass+".class_slug) = LOWER(?)", strings.TrimSpace(*q.Slug))
		}
		if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
			s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
			tx = tx.Where(`LOWER(COALESCE(`+aliasClass+`.class_notes,'')) LIKE ? OR LOWER(`+aliasClass+`.class_slug) LIKE ?`, s, s)
		}
		if q.StartGe != nil {
			tx = tx.Where(aliasClass+".class_start_date >= ?", *q.StartGe)
		}
		if q.StartLe != nil {
			tx = tx.Where(aliasClass+".class_start_date <= ?", *q.StartLe)
		}
		if q.RegOpenGe != nil {
			tx = tx.Where(aliasClass+".class_registration_opens_at >= ?", *q.RegOpenGe)
		}
		if q.RegCloseLe != nil {
			tx = tx.Where(aliasClass+".class_registration_closes_at <= ?", *q.RegCloseLe)
		}
		if q.CompletedGe != nil {
			tx = tx.Where(aliasClass+".class_completed_at >= ?", *q.CompletedGe)
		}
		if q.CompletedLe != nil {
			tx = tx.Where(aliasClass+".class_completed_at <= ?", *q.CompletedLe)
		}
		return tx
	}

	// 1) Tenant scope (PUBLIC)
	schoolIDs, err := getTenantSchoolIDs()
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 2) Parse filter + includes
	var q dto.ListClassQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	q.Normalize()

	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeAll := includeStr == "all"
	includes := map[string]bool{}
	for _, part := range strings.Split(includeStr, ",") {
		if p := strings.TrimSpace(part); p != "" {
			includes[p] = true
		}
	}

	wantSubjects := includeAll || includes["subject"] || includes["subjects"]
	wantSections := includeAll || includes["class_sections"]

	sectionsOnlyActive := strings.EqualFold(strings.TrimSpace(c.Query("sections_active")), "true")

	searchQ := strings.ToLower(strings.TrimSpace(c.Query("q")))
	if searchQ != "" {
		wantSubjects = true
	}
	like := "%" + searchQ + "%"
	parentName := strings.ToLower(strings.TrimSpace(c.Query("parent_name")))
	parentLike := "%" + parentName + "%"

	// 3) Pagination & Sorting
	pg := helper.ResolvePaging(c, 20, 200) // default 20, max 200
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by")))
	order := strings.ToLower(strings.TrimSpace(c.Query("order")))
	if order != "asc" && order != "desc" {
		if v := strings.ToLower(strings.TrimSpace(c.Query("sort"))); v != "" {
			switch v {
			case "created_at_asc":
				sortBy, order = "created_at", "asc"
			case "created_at_desc":
				sortBy, order = "created_at", "desc"
			case "slug_asc":
				sortBy, order = "slug", "asc"
			case "slug_desc":
				sortBy, order = "slug", "desc"
			default:
				order = "desc"
			}
		} else {
			order = "desc"
		}
	}

	// 4) Query: search / non-search
	parentTbl := detectParentTable(ctrl.DB)

	var classIDs []uuid.UUID
	var total int64

	if searchQ != "" {
		// SEARCH MODE: JOIN subjects via class_subjects by PARENT (baru)
		filter := ctrl.DB.Table("classes AS c").
			Where("c.class_school_id IN ?", schoolIDs).
			Joins(`
				LEFT JOIN class_subjects AS cs
				  ON  cs.class_subject_class_parent_id = c.class_class_parent_id
				  AND cs.class_subject_school_id = c.class_school_id
				  AND cs.class_subject_is_active = TRUE
				  AND cs.class_subject_deleted_at IS NULL
			`).
			Joins(`LEFT JOIN subjects AS s ON s.subject_id = cs.class_subject_subject_id`)

		filter = applyCommonFilters(filter, "c", q)
		var hasParent bool
		filter, hasParent = addParentJoin(filter, "c", parentTbl)

		if hasParent {
			filter = filter.Where(`
				LOWER(c.class_slug) LIKE ? OR
				LOWER(COALESCE(c.class_notes,'')) LIKE ? OR
				LOWER(s.subject_name) LIKE ? OR
				LOWER(COALESCE(p.class_parent_name,'')) LIKE ?
			`, like, like, like, like)
		} else {
			filter = filter.Where(`
				LOWER(c.class_slug) LIKE ? OR
				LOWER(COALESCE(c.class_notes,'')) LIKE ? OR
				LOWER(s.subject_name) LIKE ?
			`, like, like, like)
		}

		if hasParent && parentName != "" {
			filter = filter.Where(`LOWER(COALESCE(p.class_parent_name,'')) LIKE ?`, parentLike)
		}

		if err := filter.Session(&gorm.Session{}).
			Distinct("c.class_id").
			Count(&total).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total")
		}

		type idRow struct {
			ClassID   uuid.UUID `gorm:"column:class_id"`
			ClassSlug string    `gorm:"column:class_slug"`
		}
		var idRows []idRow

		// Sorting di mode search: pakai slug asc (stabil)
		orderExpr := "LOWER(c.class_slug) ASC"

		if err := filter.
			Select("c.class_id, c.class_slug").
			Group("c.class_id, c.class_slug").
			Order(orderExpr).
			Limit(pg.Limit).
			Offset(pg.Offset).
			Scan(&idRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar kelas")
		}
		if len(idRows) == 0 {
			pagination := helper.BuildPaginationFromOffset(total, pg.Offset, pg.Limit)
			return helper.JsonList(c, "ok", []any{}, pagination)
		}
		classIDs = make([]uuid.UUID, 0, len(idRows))
		for _, r := range idRows {
			classIDs = append(classIDs, r.ClassID)
		}

	} else {
		tx := ctrl.DB.Model(&model.ClassModel{}).
			Where("class_school_id IN ?", schoolIDs)

		tx = applyCommonFilters(tx, "classes", q)
		var hasParent bool
		tx, hasParent = addParentJoin(tx, "classes", parentTbl)

		if hasParent && parentName != "" {
			tx = tx.Where(`LOWER(COALESCE(p.class_parent_name,'')) LIKE ?`, parentLike)
		}

		if err := tx.Count(&total).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
		}

		// Sorting whitelist (non-search)
		switch sortBy {
		case "slug":
			tx = tx.Order("LOWER(class_slug) " + strings.ToUpper(order)).Order("class_created_at DESC")
		case "start_date":
			tx = tx.Order("class_start_date " + strings.ToUpper(order)).Order("class_created_at DESC")
		case "status":
			tx = tx.Order("class_status " + strings.ToUpper(order)).Order("class_created_at DESC")
		case "delivery_mode":
			tx = tx.Order("LOWER(class_delivery_mode) " + strings.ToUpper(order)).Order("class_created_at DESC")
		case "reg_open":
			tx = tx.Order("class_registration_opens_at " + strings.ToUpper(order)).Order("class_created_at DESC")
		case "reg_close":
			tx = tx.Order("class_registration_closes_at " + strings.ToUpper(order)).Order("class_created_at DESC")
		case "created_at":
			fallthrough
		default:
			tx = tx.Order("class_created_at " + strings.ToUpper(order))
		}

		type idOnly struct {
			ClassID uuid.UUID `gorm:"column:class_id"`
		}
		var idRows []idOnly
		if err := tx.Select("class_id").Limit(pg.Limit).Offset(pg.Offset).Scan(&idRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if len(idRows) == 0 {
			pagination := helper.BuildPaginationFromOffset(total, pg.Offset, pg.Limit)
			return helper.JsonList(c, "ok", []any{}, pagination)
		}
		classIDs = make([]uuid.UUID, 0, len(idRows))
		for _, r := range idRows {
			classIDs = append(classIDs, r.ClassID)
		}
	}

	// 5) Ambil detail rows
	var rows []model.ClassModel
	if err := ctrl.DB.
		Where("class_id IN ?", classIDs).
		Where("class_deleted_at IS NULL").
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil detail kelas")
	}

	// 6) Prefetch term & parent
	type termLite struct {
		ID           uuid.UUID  `json:"academic_terms_id"`
		Name         string     `json:"academic_terms_name"`
		AcademicYear string     `json:"academic_terms_academic_year"`
		StartDate    *time.Time `json:"academic_terms_start_date,omitempty"`
		EndDate      *time.Time `json:"academic_terms_end_date,omitempty"`
		IsActive     bool       `json:"academic_terms_is_active"`
		Angkatan     *int       `json:"academic_terms_angkatan,omitempty"`
	}
	type parentLite struct {
		ID        uuid.UUID  `json:"class_parent_id"                   gorm:"column:class_parent_id"`
		Name      string     `json:"class_parent_name"                 gorm:"column:class_parent_name"`
		Code      *string    `json:"class_parent_code,omitempty"       gorm:"column:class_parent_code"`
		Level     *int16     `json:"class_parent_level,omitempty"      gorm:"column:class_parent_level"`
		ImageURL  *string    `json:"class_parent_image_url,omitempty"  gorm:"column:class_parent_image_url"`
		IsActive  bool       `json:"class_parent_is_active"            gorm:"column:class_parent_is_active"`
		CreatedAt *time.Time `json:"class_parent_created_at,omitempty" gorm:"column:class_parent_created_at"`
	}

	termMap := map[uuid.UUID]termLite{}
	parentMap := map[uuid.UUID]parentLite{}

	if len(rows) > 0 {
		tSet := map[uuid.UUID]struct{}{}
		pSet := map[uuid.UUID]struct{}{}
		for i := range rows {
			if rows[i].ClassAcademicTermID != nil {
				tSet[*rows[i].ClassAcademicTermID] = struct{}{}
			}
			if rows[i].ClassClassParentID != uuid.Nil {
				pSet[rows[i].ClassClassParentID] = struct{}{}
			}
		}
		if len(tSet) > 0 {
			ids := make([]uuid.UUID, 0, len(tSet))
			for id := range tSet {
				ids = append(ids, id)
			}
			type tr struct {
				ID           uuid.UUID  `gorm:"column:academic_term_id"`
				Name         string     `gorm:"column:academic_term_name"`
				AcademicYear string     `gorm:"column:academic_term_academic_year"`
				Start        *time.Time `gorm:"column:academic_term_start_date"`
				End          *time.Time `gorm:"column:academic_term_end_date"`
				IsActive     bool       `gorm:"column:academic_term_is_active"`
				Angkatan     *int       `gorm:"column:academic_term_angkatan"`
			}
			var ts []tr
			if err := ctrl.DB.
				Table("academic_terms").
				Select(`
					academic_term_id,
					academic_term_name,
					academic_term_academic_year,
					academic_term_start_date,
					academic_term_end_date,
					academic_term_is_active,
					academic_term_angkatan
				`).
				Where("academic_term_id IN ? AND academic_term_deleted_at IS NULL", ids).
				Find(&ts).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil academic_terms")
			}
			for _, r := range ts {
				termMap[r.ID] = termLite{
					ID: r.ID, Name: r.Name, AcademicYear: r.AcademicYear,
					StartDate: r.Start, EndDate: r.End, IsActive: r.IsActive, Angkatan: r.Angkatan,
				}
			}
		}
		if len(pSet) > 0 {
			ids := make([]uuid.UUID, 0, len(pSet))
			for id := range pSet {
				ids = append(ids, id)
			}
			var ps []parentLite
			parentTbl := detectParentTable(ctrl.DB)
			if parentTbl != "" {
				if err := ctrl.DB.
					Table(parentTbl).
					Select(`class_parent_id, class_parent_name, class_parent_code, class_parent_level,
							class_parent_image_url, class_parent_is_active, class_parent_created_at`).
					Where("class_parent_id IN ? AND class_parent_deleted_at IS NULL", ids).
					Find(&ps).Error; err != nil {
					return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil class_parent")
				}
				for _, r := range ps {
					parentMap[r.ID] = r
				}
			}
		}
	}

	// 7) Prefetch subjects (opsional) — via class_parent (bukan class_id)
	type SubjectLite struct {
		SubjectID      uuid.UUID `json:"subject_id"       gorm:"column:subject_id"`
		SubjectName    string    `json:"subject_name"     gorm:"column:subject_name"`
		ClassSubjectID uuid.UUID `json:"class_subject_id" gorm:"column:class_subject_id"`
	}
	subjectsMap := map[uuid.UUID][]SubjectLite{} // key: class_id

	if wantSubjects && len(rows) > 0 {
		// kumpulkan parent_id dari kelas pada halaman ini
		parentSet := map[uuid.UUID]struct{}{}
		classToParent := make(map[uuid.UUID]uuid.UUID, len(rows))
		for i := range rows {
			p := rows[i].ClassClassParentID
			classToParent[rows[i].ClassID] = p
			parentSet[p] = struct{}{}
		}

		parentIDs := make([]uuid.UUID, 0, len(parentSet))
		for id := range parentSet {
			parentIDs = append(parentIDs, id)
		}

		type subjRow struct {
			ParentID       uuid.UUID `gorm:"column:parent_id"`
			SubjectID      uuid.UUID `gorm:"column:subject_id"`
			SubjectName    string    `gorm:"column:subject_name"`
			ClassSubjectID uuid.UUID `gorm:"column:class_subject_id"`
		}
		var sjRows []subjRow

		if len(parentIDs) > 0 {
			if err := ctrl.DB.
				Table("class_subjects AS cs").
				Joins(`JOIN subjects AS s ON s.subject_id = cs.class_subject_subject_id`).
				Where(`
					cs.class_subject_school_id IN ?
					AND cs.class_subject_is_active = TRUE
					AND cs.class_subject_deleted_at IS NULL
				`, schoolIDs).
				Where("cs.class_subject_class_parent_id IN ?", parentIDs).
				Select(`
					cs.class_subject_class_parent_id AS parent_id,
					s.subject_id,
					s.subject_name,
					cs.class_subject_id
				`).
				Order("s.subject_name ASC").
				Scan(&sjRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil subject kelas")
			}
		}

		parentSubjects := make(map[uuid.UUID][]SubjectLite, len(parentIDs))
		for _, r := range sjRows {
			parentSubjects[r.ParentID] = append(parentSubjects[r.ParentID], SubjectLite{
				SubjectID:      r.SubjectID,
				SubjectName:    r.SubjectName,
				ClassSubjectID: r.ClassSubjectID,
			})
		}

		for classID, parentID := range classToParent {
			subjectsMap[classID] = parentSubjects[parentID]
		}
	}

	// 7b) Prefetch class sections (opsional)
	type SectionLite struct {
		ClassSectionID      uuid.UUID `json:"class_section_id" gorm:"column:class_section_id"`
		ClassSectionClassID uuid.UUID `json:"class_section_class_id" gorm:"column:class_section_class_id"`
		Slug                string    `json:"class_section_slug" gorm:"column:class_section_slug"`
		Name                string    `json:"class_section_name" gorm:"column:class_section_name"`
		Code                *string   `json:"class_section_code,omitempty" gorm:"column:class_section_code"`
		Capacity            *int      `json:"class_section_capacity,omitempty" gorm:"column:class_section_capacity"`
		TotalStudents       int       `json:"class_section_total_students" gorm:"column:class_section_total_students"`
		IsActive            bool      `json:"class_section_is_active" gorm:"column:class_section_is_active"`

		// kolom snapshot (nama kolom sesuai model baru)
		TeacherNameSnap      *string `json:"class_section_teacher_name_snap,omitempty" gorm:"column:class_section_school_teacher_name_snapshot"`
		AssistantTeacherName *string `json:"class_section_assistant_teacher_name_snap,omitempty" gorm:"column:class_section_assistant_school_teacher_name_snapshot"`
		RoomNameSnap         *string `json:"class_section_room_name_snap,omitempty" gorm:"column:class_section_class_room_name_snapshot"`

		// term id section
		TermID *uuid.UUID `json:"class_section_term_id,omitempty" gorm:"column:class_section_academic_term_id"`

		// counter kolom yang benar
		CSSTCount       int `json:"class_sections_subject_teachers_count" gorm:"column:class_section_subject_teachers_count"`
		CSSTActiveCount int `json:"class_sections_subject_teachers_active_count" gorm:"column:class_section_subject_teachers_active_count"`
	}
	sectionsMap := map[uuid.UUID][]SectionLite{}

	if wantSections && len(rows) > 0 {
		classIDs2 := make([]uuid.UUID, 0, len(rows))
		for i := range rows {
			classIDs2 = append(classIDs2, rows[i].ClassID)
		}

		txSec := ctrl.DB.
			Table("class_sections").
			Select(`
				class_section_id,
				class_section_school_id,
				class_section_class_id,
				class_section_slug,
				class_section_name,
				class_section_code,
				class_section_capacity,
				class_section_total_students,
				class_section_is_active,
				class_section_school_teacher_name_snapshot,
				class_section_assistant_school_teacher_name_snapshot,
				class_section_class_room_name_snapshot,
				class_section_academic_term_id,
				class_section_subject_teachers_count,
				class_section_subject_teachers_active_count
			`).
			Where("class_section_deleted_at IS NULL").
			Where("class_section_school_id IN ?", schoolIDs).
			Where("class_section_class_id IN ?", classIDs2)

		if sectionsOnlyActive {
			txSec = txSec.Where("class_section_is_active = TRUE")
		}

		txSec = txSec.Order("LOWER(class_section_name) ASC, class_section_created_at DESC")

		var secRows []SectionLite
		if err := txSec.Scan(&secRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil class sections")
		}
		for _, s := range secRows {
			sectionsMap[s.ClassSectionClassID] = append(sectionsMap[s.ClassSectionClassID], s)
		}
	}

	// 8) Compose output (pertahankan urutan page)
	type classWithExpand struct {
		*dto.ClassResponse `json:",inline"`
		AcademicTerms      *termLite     `json:"academic_terms,omitempty"`
		ClassParents       *parentLite   `json:"class_parents,omitempty"`
		Subjects           []SubjectLite `json:"subjects,omitempty"`
		ClassSections      []SectionLite `json:"class_sections,omitempty"`
	}

	rowByID := make(map[uuid.UUID]*model.ClassModel, len(rows))
	for i := range rows {
		rowByID[rows[i].ClassID] = &rows[i]
	}

	out := make([]*classWithExpand, 0, len(classIDs))
	for _, id := range classIDs {
		r := rowByID[id]
		if r == nil {
			continue
		}
		base := dto.FromModel(r)
		item := &classWithExpand{ClassResponse: &base}

		if r.ClassAcademicTermID != nil {
			if t, ok := termMap[*r.ClassAcademicTermID]; ok {
				tCopy := t
				item.AcademicTerms = &tCopy
			}
		}
		if pLite, ok := parentMap[r.ClassClassParentID]; ok {
			pCopy := pLite
			item.ClassParents = &pCopy
		}
		if wantSubjects {
			item.Subjects = subjectsMap[r.ClassID]
		}
		if wantSections {
			item.ClassSections = sectionsMap[r.ClassID]
		}
		out = append(out, item)
	}

	// ✅ Pagination
	pagination := helper.BuildPaginationFromOffset(total, pg.Offset, pg.Limit)
	return helper.JsonList(c, "ok", out, pagination)
}
