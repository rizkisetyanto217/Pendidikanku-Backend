// internals/features/lembaga/classes/user_classes/main/controller/user_my_class_controller.go
package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/school/classes/classes/dto"
	"masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

// Struct ringan utk expand
type termLite struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	AcademicYear string     `json:"academic_year"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	IsActive     bool       `json:"is_active"`
	Angkatan     *int       `json:"angkatan,omitempty"`
}

type parentLite struct {
	ID        uuid.UUID  `json:"id"                     gorm:"column:class_parent_id"`
	Name      string     `json:"name"                   gorm:"column:class_parent_name"`
	Code      *string    `json:"code,omitempty"         gorm:"column:class_parent_code"`
	Level     *int16     `json:"level,omitempty"        gorm:"column:class_parent_level"`
	ImageURL  *string    `json:"image_url,omitempty"    gorm:"column:class_parent_image_url"`
	IsActive  bool       `json:"is_active"              gorm:"column:class_parent_is_active"`
	CreatedAt *time.Time `json:"created_at,omitempty"   gorm:"column:class_parent_created_at"`
}

// GET /admin/classes/slug/:slug
func (ctrl *ClassController) GetClassBySlug(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	// slugify ringan (lower + trim + safe)
	slug := helper.GenerateSlug(c.Params("slug"))

	var m model.ClassModel
	if err := ctrl.DB.
		Where(`
			class_masjid_id = ?
			AND lower(class_slug) = lower(?)
			AND class_deleted_at IS NULL
			AND class_delete_pending_until IS NULL
		`, masjidID, slug).
		First(&m).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "Data diterima", dto.FromModel(&m))
}



// GET /admin/classes  (public-friendly; auth optional utk list)
// GANTI fungsi ListClasses dengan versi ini
// GET /admin/classes  (public-friendly; auth optional utk list)
// Refactor total: gabungan list + search (q), dukung filter parent_name, include term/parent/subjects
// GET /admin/classes  (public-friendly; auth optional utk list)
// Refactor total: gabungan list + search (q), filter parent_name via JOIN dinamis (class_parents/class_parent)
func (ctrl *ClassController) ListClasses(c *fiber.Ctx) error {

		// Deteksi tabel parent yang tersedia: "class_parents" atau "class_parent"
		detectParentTable := func(db *gorm.DB) string {
		// 1) coba to_regclass tanpa schema (mengikuti search_path)
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

			// 2) fallback: ambil satu relname dari pg_class (scan ke string langsung)
			var name string
			row := db.Raw(`
				SELECT c.relname
				FROM pg_catalog.pg_class c
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE c.relkind = 'r'
				AND c.relname IN ('class_parents', 'class_parent')
				LIMIT 1
			`).Row()
			if err := row.Scan(&name); err == nil && name != "" {
				return name
			}

			// 3) fallback terakhir: coba qualified public (untuk env lama)
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

	// Tambahkan LEFT JOIN ke parent table (jika ada) dengan alias p
	addParentJoin := func(tx *gorm.DB, aliasClass, parentTbl string) (*gorm.DB, bool) {
		if parentTbl == "" {
			return tx, false
		}
		j := fmt.Sprintf(`LEFT JOIN %s AS p
			ON p.class_parent_id = %s.class_parent_id
		   AND p.class_parent_deleted_at IS NULL`, parentTbl, aliasClass)
		return tx.Joins(j), true
	}

	// Ambil tenant scope dari token atau dari query (?masjid_id / ?masjid_slug)
	getTenantMasjidIDs := func() ([]uuid.UUID, error) {
		if ids, err := helperAuth.GetMasjidIDsFromToken(c); err == nil && len(ids) > 0 {
			return ids, nil
		}
		var out []uuid.UUID

		// ?masjid_id=uuid,uuid,...
		if rawIDs := strings.TrimSpace(c.Query("masjid_id")); rawIDs != "" {
			for _, part := range strings.Split(rawIDs, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				id, e := uuid.Parse(part)
				if e != nil {
					return nil, fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak valid")
				}
				out = append(out, id)
			}
		}

		// ?masjid_slug=slug,slug,...
		if len(out) == 0 {
			if rawSlugs := strings.TrimSpace(c.Query("masjid_slug")); rawSlugs != "" {
				slugs := make([]string, 0, 4)
				for _, s := range strings.Split(rawSlugs, ",") {
					if s = strings.TrimSpace(s); s != "" {
						slugs = append(slugs, s)
					}
				}
				if len(slugs) > 0 {
					var ids []uuid.UUID
					if err := ctrl.DB.
						Table("masjids").
						Select("masjid_id").
						Where("masjid_deleted_at IS NULL").
						Where("masjid_slug IN ?", slugs).
						Scan(&ids).Error; err != nil {
						return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca masjid_slug: "+err.Error())
					}
					out = append(out, ids...)
				}
			}
		}

		if len(out) == 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Tanpa auth, wajib kirim masjid_id atau masjid_slug")
		}
		return out, nil
	}

	// Terapkan filter umum (tanpa q) ke query "classes" dengan alias tertentu: "c" atau "classes"
	applyCommonFilters := func(tx *gorm.DB, aliasClass string, q dto.ListClassQuery) *gorm.DB {
		tx = tx.
			Where(aliasClass+".class_deleted_at IS NULL").
			Where(aliasClass+".class_delete_pending_until IS NULL")

		if q.ParentID != nil {
			tx = tx.Where(aliasClass+".class_parent_id = ?", *q.ParentID)
		}
		if q.TermID != nil {
			tx = tx.Where(aliasClass+".class_term_id = ?", *q.TermID)
		}
		if q.IsOpen != nil {
			tx = tx.Where(aliasClass+".class_is_open = ?", *q.IsOpen)
		}
		if raw := strings.TrimSpace(c.Query("id")); raw != "" {
			var ids []uuid.UUID
			for _, part := range strings.Split(raw, ",") {
				if part = strings.TrimSpace(part); part == "" {
					continue
				}
				id, err := uuid.Parse(part)
				if err != nil { return tx.Where("1=0") }
				ids = append(ids, id)
			}
			if len(ids) > 0 { tx = tx.Where(aliasClass+".class_id IN ?", ids) }
		} else if raw := strings.TrimSpace(c.Query("class_id")); raw != "" {
			var ids []uuid.UUID
			for _, part := range strings.Split(raw, ",") {
				if part = strings.TrimSpace(part); part == "" { continue }
				id, err := uuid.Parse(part)
				if err != nil { return tx.Where("1=0") }
				ids = append(ids, id)
			}
			if len(ids) > 0 { tx = tx.Where(aliasClass+".class_id IN ?", ids) }
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
		// legacy search (khusus non-search mode)
		if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
			s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
			tx = tx.Where(`LOWER(COALESCE(`+aliasClass+`.class_notes,'')) LIKE ? OR LOWER(`+aliasClass+`.class_slug) LIKE ?`, s, s)
		}
		if q.StartGe != nil { tx = tx.Where(aliasClass+".class_start_date >= ?", *q.StartGe) }
		if q.StartLe != nil { tx = tx.Where(aliasClass+".class_start_date <= ?", *q.StartLe) }
		if q.RegOpenGe != nil { tx = tx.Where(aliasClass+".class_registration_opens_at >= ?", *q.RegOpenGe) }
		if q.RegCloseLe != nil { tx = tx.Where(aliasClass+".class_registration_closes_at <= ?", *q.RegCloseLe) }
		if q.CompletedGe != nil { tx = tx.Where(aliasClass+".class_completed_at >= ?", *q.CompletedGe) }
		if q.CompletedLe != nil { tx = tx.Where(aliasClass+".class_completed_at <= ?", *q.CompletedLe) }

		return tx
	}

	/* =============================
	   1) Tenant scope
	============================= */
	masjidIDs, err := getTenantMasjidIDs()
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	/* =============================
	   2) Parse filter DTO & includes
	============================= */
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
	wantTerm := includeAll || includes["term"] || includes["terms"]
	wantParent := includeAll || includes["parent"] || includes["parents"]
	wantSubjects := includeAll || includes["subject"] || includes["subjects"]

	// Pencarian gabungan; saat q != "", auto include subjects
	searchQ := strings.ToLower(strings.TrimSpace(c.Query("q")))
	if searchQ != "" { wantSubjects = true }
	like := "%" + searchQ + "%"

	// Filter tambahan: parent_name
	parentName := strings.ToLower(strings.TrimSpace(c.Query("parent_name")))
	parentLike := "%" + parentName + "%"

	/* =============================
	   3) Pagination & sorting
	============================= */
	rawQuery := string(c.Request().URI().QueryString())
	httpReq, _ := http.NewRequest("GET", "http://local?"+rawQuery, nil)
	pg := helper.ParseWith(httpReq, "created_at", "desc", helper.AdminOpts)

	/* =============================
	   4) Querying: MODE A (search) vs MODE B (non-search)
	============================= */
	parentTbl := detectParentTable(ctrl.DB)

	var classIDs []uuid.UUID
	var total int64

	if searchQ != "" {
		// ----- MODE A: Search -----
		filter := ctrl.DB.Table("classes AS c").
			Where("c.class_masjid_id IN ?", masjidIDs).
			Joins(`
				LEFT JOIN class_subjects AS cs
				  ON cs.class_subjects_class_id = c.class_id
				 AND cs.class_subjects_masjid_id = c.class_masjid_id
				 AND cs.class_subjects_is_active = TRUE
				 AND cs.class_subjects_deleted_at IS NULL
			`).
			Joins(`LEFT JOIN subjects AS s ON s.subjects_id = cs.class_subjects_subject_id`)

		filter = applyCommonFilters(filter, "c", q)
		var hasParent bool
		filter, hasParent = addParentJoin(filter, "c", parentTbl)

		// match q ke slug/notes/subjects/parent.name (kalau ada parent table)
		if hasParent {
			filter = filter.Where(`
				LOWER(c.class_slug) LIKE ? OR
				LOWER(COALESCE(c.class_notes,'')) LIKE ? OR
				LOWER(s.subjects_name) LIKE ? OR
				LOWER(COALESCE(p.class_parent_name,'')) LIKE ?
			`, like, like, like, like)
		} else {
			filter = filter.Where(`
				LOWER(c.class_slug) LIKE ? OR
				LOWER(COALESCE(c.class_notes,'')) LIKE ? OR
				LOWER(s.subjects_name) LIKE ?
			`, like, like, like)
		}

		// filter parent_name spesifik jika ada dan tabel parent tersedia
		if hasParent && parentName != "" {
			filter = filter.Where(`LOWER(COALESCE(p.class_parent_name,'')) LIKE ?`, parentLike)
		}

		// total distinct
		if err := filter.Session(&gorm.Session{}).
			Distinct("c.class_id").
			Count(&total).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total")
		}

		// page of IDs (urut slug ASC agar stabil)
		type idRow struct {
			ClassID   uuid.UUID `gorm:"column:class_id"`
			ClassSlug string    `gorm:"column:class_slug"`
		}
		var idRows []idRow
		if err := filter.
			Select("c.class_id, c.class_slug").
			Group("c.class_id, c.class_slug").
			Order("LOWER(c.class_slug) ASC").
			Limit(pg.Limit()).Offset(pg.Offset()).
			Scan(&idRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil daftar kelas")
		}
		if len(idRows) == 0 {
			return helper.JsonList(c, []any{}, helper.BuildMeta(total, pg))
		}
		classIDs = make([]uuid.UUID, 0, len(idRows))
		for _, r := range idRows {
			classIDs = append(classIDs, r.ClassID)
		}

	} else {
		// ----- MODE B: Non-search -----
		tx := ctrl.DB.Model(&model.ClassModel{}).
			Where("class_masjid_id IN ?", masjidIDs)

		tx = applyCommonFilters(tx, "classes", q)
		var hasParent bool
		tx, hasParent = addParentJoin(tx, "classes", parentTbl)

		// filter parent_name bila ada parent table
		if hasParent && parentName != "" {
			tx = tx.Where(`LOWER(COALESCE(p.class_parent_name,'')) LIKE ?`, parentLike)
		}

		// total
		if err := tx.Count(&total).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total data")
		}

		// sorting
		sortBy := strings.ToLower(strings.TrimSpace(pg.SortBy))
		order := strings.ToLower(strings.TrimSpace(pg.SortOrder))
		if order != "asc" && order != "desc" { order = "desc" }
		switch sortBy {
		case "slug":
			tx = tx.Order("LOWER(class_slug) " + strings.ToUpper(order)).Order("class_created_at DESC")
		case "start_date":
			tx = tx.Order("class_start_date " + strings.ToUpper(order)).Order("class_created_at DESC")
		case "is_open":
			tx = tx.Order("class_is_open " + strings.ToUpper(order)).Order("class_created_at DESC")
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

		// page of IDs
		type idOnly struct{ ClassID uuid.UUID `gorm:"column:class_id"` }
		var idRows []idOnly
		if err := tx.Select("class_id").Limit(pg.Limit()).Offset(pg.Offset()).Scan(&idRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if len(idRows) == 0 {
		 return helper.JsonList(c, []any{}, helper.BuildMeta(total, pg))
		}
		classIDs = make([]uuid.UUID, 0, len(idRows))
		for _, r := range idRows { classIDs = append(classIDs, r.ClassID) }
	}

	/* =============================
	   5) Ambil detail rows untuk classIDs
	============================= */
	var rows []model.ClassModel
	if err := ctrl.DB.
		Where("class_id IN ?", classIDs).
		Where("class_deleted_at IS NULL").
		Where("class_delete_pending_until IS NULL").
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil detail kelas")
	}

	/* =============================
	   6) Prefetch term & parent (opsional)
	============================= */
	termMap := map[uuid.UUID]termLite{}
	parentMap := map[uuid.UUID]parentLite{}

	if (wantTerm || wantParent) && len(rows) > 0 {
		tSet := map[uuid.UUID]struct{}{}
		pSet := map[uuid.UUID]struct{}{}
		for i := range rows {
			if wantTerm && rows[i].ClassTermID != nil { tSet[*rows[i].ClassTermID] = struct{}{} }
			if wantParent && rows[i].ClassParentID != uuid.Nil { pSet[rows[i].ClassParentID] = struct{}{} }
		}

		// term
		if wantTerm && len(tSet) > 0 {
			ids := make([]uuid.UUID, 0, len(tSet))
			for id := range tSet { ids = append(ids, id) }
			type tr struct {
				ID           uuid.UUID  `gorm:"column:academic_terms_id"`
				Name         string     `gorm:"column:academic_terms_name"`
				AcademicYear string     `gorm:"column:academic_terms_academic_year"`
				Start        *time.Time `gorm:"column:academic_terms_start_date"`
				End          *time.Time `gorm:"column:academic_terms_end_date"`
				IsActive     bool       `gorm:"column:academic_terms_is_active"`
				Angkatan     *int       `gorm:"column:academic_terms_angkatan"`
			}
			var ts []tr
			if err := ctrl.DB.
				Table("academic_terms").
				Select(`academic_terms_id, academic_terms_name, academic_terms_academic_year,
				        academic_terms_start_date, academic_terms_end_date,
				        academic_terms_is_active, academic_terms_angkatan`).
				Where("academic_terms_id IN ? AND academic_terms_deleted_at IS NULL", ids).
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

		// parent (pakai tabel yang terdeteksi saja; tidak ada fallback subquery)
		if wantParent && len(pSet) > 0 && parentTbl != "" {
			ids := make([]uuid.UUID, 0, len(pSet))
			for id := range pSet { ids = append(ids, id) }

			var ps []parentLite
			if err := ctrl.DB.
				Table(parentTbl).
				Select(`class_parent_id, class_parent_name, class_parent_code, class_parent_level,
						class_parent_image_url, class_parent_is_active, class_parent_created_at`).
				Where("class_parent_id IN ? AND class_parent_deleted_at IS NULL", ids).
				Find(&ps).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil class_parent")
			}
			for _, r := range ps { parentMap[r.ID] = r }
		}
	}

	/* =============================
	   7) Prefetch subjects (opsional / auto saat q)
	============================= */
	type SubjectLite struct {
		SubjectsID      uuid.UUID `json:"subjects_id"`
		SubjectsName    string    `json:"subjects_name"`
		ClassSubjectsID uuid.UUID `json:"class_subjects_id"`
	}
	subjectsMap := map[uuid.UUID][]SubjectLite{}
	if wantSubjects && len(rows) > 0 {
		classIDs2 := make([]uuid.UUID, 0, len(rows))
		for i := range rows { classIDs2 = append(classIDs2, rows[i].ClassID) }

		type subjRow struct {
			ClassID         uuid.UUID `gorm:"column:class_id"`
			SubjectsID      uuid.UUID `gorm:"column:subjects_id"`
			SubjectsName    string    `gorm:"column:subjects_name"`
			ClassSubjectsID uuid.UUID `gorm:"column:class_subjects_id"`
		}
		var sjRows []subjRow
		if err := ctrl.DB.Table("class_subjects AS cs").
			Joins(`JOIN subjects AS s ON s.subjects_id = cs.class_subjects_subject_id`).
			Where(`
				cs.class_subjects_masjid_id IN ?
				AND cs.class_subjects_is_active = TRUE
				AND cs.class_subjects_deleted_at IS NULL
			`, masjidIDs).
			Where("cs.class_subjects_class_id IN ?", classIDs2).
			Select(`cs.class_subjects_class_id AS class_id, s.subjects_id, s.subjects_name, cs.class_subjects_id`).
			Order("s.subjects_name ASC").
			Scan(&sjRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil subject kelas")
		}
		for _, r := range sjRows {
			subjectsMap[r.ClassID] = append(subjectsMap[r.ClassID], SubjectLite{
				SubjectsID:      r.SubjectsID,
				SubjectsName:    r.SubjectsName,
				ClassSubjectsID: r.ClassSubjectsID,
			})
		}
	}

	/* =============================
	   8) Compose output (pertahankan urutan page)
	============================= */
	type classWithExpand struct {
		*dto.ClassResponse `json:",inline"`
		Term               *termLite     `json:"term,omitempty"`
		Parent             *parentLite   `json:"parent,omitempty"`
		Subjects           []SubjectLite `json:"subjects,omitempty"`
	}

	rowByID := make(map[uuid.UUID]*model.ClassModel, len(rows))
	for i := range rows { rowByID[rows[i].ClassID] = &rows[i] }

	out := make([]*classWithExpand, 0, len(classIDs))
	for _, id := range classIDs {
		r := rowByID[id]
		if r == nil { continue }
		base := dto.FromModel(r)
		item := &classWithExpand{ClassResponse: &base}

		if wantTerm && r.ClassTermID != nil {
			if t, ok := termMap[*r.ClassTermID]; ok { tCopy := t; item.Term = &tCopy }
		}
		if wantParent {
			if pLite, ok := parentMap[r.ClassParentID]; ok { pCopy := pLite; item.Parent = &pCopy }
		}
		if wantSubjects { item.Subjects = subjectsMap[r.ClassID] }

		out = append(out, item)
	}

	meta := helper.BuildMeta(total, pg)
	return helper.JsonList(c, out, meta)
}
