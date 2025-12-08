// file: internals/features/school/classes/class_sections/controller/class_section_list.go
package controller

import (
	"encoding/json"
	"strings"

	csstDTO "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	secDTO "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	secModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =================== parseUUIDList tetap sama =================== */

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
		return nil, fiber.NewError(fiber.StatusBadRequest, "daftar id kosong")
	}
	return out, nil
}

/* =================== ACADEMIC TERM LITE (local di controller) =================== */

// bentuk lite yang dipakai untuk nested & include
type AcademicTermLite struct {
	ID       *uuid.UUID `json:"id,omitempty"`
	Name     *string    `json:"name,omitempty"`
	Slug     *string    `json:"slug,omitempty"`
	Year     *string    `json:"year,omitempty"`
	Angkatan *int       `json:"angkatan,omitempty"`
}

// dari satu ClassSectionModel → AcademicTermLite (pakai cache di model)
func academicTermLiteFromSectionModel(cs *secModel.ClassSectionModel) *AcademicTermLite {
	if cs == nil {
		return nil
	}

	// kalau benar-benar nggak ada jejak term sama sekali, skip
	if cs.ClassSectionAcademicTermID == nil &&
		(cs.ClassSectionAcademicTermNameCache == nil || strings.TrimSpace(*cs.ClassSectionAcademicTermNameCache) == "") &&
		(cs.ClassSectionAcademicTermSlugCache == nil || strings.TrimSpace(*cs.ClassSectionAcademicTermSlugCache) == "") &&
		(cs.ClassSectionAcademicTermAcademicYearCache == nil || strings.TrimSpace(*cs.ClassSectionAcademicTermAcademicYearCache) == "") &&
		cs.ClassSectionAcademicTermAngkatanCache == nil {
		return nil
	}

	return &AcademicTermLite{
		ID:       cs.ClassSectionAcademicTermID,
		Name:     cs.ClassSectionAcademicTermNameCache,
		Slug:     cs.ClassSectionAcademicTermSlugCache,
		Year:     cs.ClassSectionAcademicTermAcademicYearCache,
		Angkatan: cs.ClassSectionAcademicTermAngkatanCache,
	}
}

// dari slice ClassSectionModel → unique list AcademicTermLite (buat include["academic_term"])
func buildAcademicTermInclude(list []secModel.ClassSectionModel) []AcademicTermLite {
	out := make([]AcademicTermLite, 0, len(list))
	seen := make(map[string]struct{}, len(list))

	for i := range list {
		at := academicTermLiteFromSectionModel(&list[i])
		if at == nil {
			continue
		}

		// kunci keunikan: utamakan ID, lalu slug, lalu name
		var key string
		switch {
		case at.ID != nil:
			key = "id:" + at.ID.String()
		case at.Slug != nil && strings.TrimSpace(*at.Slug) != "":
			key = "slug:" + strings.TrimSpace(*at.Slug)
		case at.Name != nil && strings.TrimSpace(*at.Name) != "":
			key = "name:" + strings.TrimSpace(*at.Name)
		default:
			continue
		}

		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, *at)
	}

	return out
}

// GET /api/{a|u}/:school_id/class-sections/list
func (ctrl *ClassSectionController) List(c *fiber.Ctx) error {
	// ---------- School context: token > slug/id ----------
	var schoolID uuid.UUID

	// 1) Coba dari token dulu (student/teacher/dkm/admin/bendahara)
	if sid, errTok := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); errTok == nil && sid != uuid.Nil {
		schoolID = sid
	} else {
		// 2) Fallback ke ResolveSchoolContext (path param / header / dll)
		mc, err := helperAuth.ResolveSchoolContext(c)
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID

		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
			if er != nil {
				if er == gorm.ErrRecordNotFound {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id

		default:
			return helper.JsonError(c, fiber.StatusBadRequest, helperAuth.ErrSchoolContextMissing.Error())
		}
	}

	// ---------- Search term ----------
	rawQ := strings.TrimSpace(c.Query("q"))
	rawSearch := strings.TrimSpace(c.Query("search"))
	searchTerm := rawSearch
	if rawQ != "" {
		searchTerm = rawQ
		if len([]rune(searchTerm)) < 2 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter q minimal 2 karakter")
		}
	}

	// ---------- Paging & sorting ----------
	defaultSortBy := "created_at"
	defaultOrder := "desc"
	if searchTerm != "" {
		defaultSortBy = "name"
		defaultOrder = "asc"
	}
	pg := helper.ResolvePaging(c, 20, 200)

	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by", defaultSortBy)))
	order := strings.ToLower(strings.TrimSpace(c.Query("order", defaultOrder)))
	if order != "asc" && order != "desc" {
		order = defaultOrder
	}
	col := "class_section_created_at"
	switch sortBy {
	case "name":
		col = "class_section_name"
	case "created_at":
		col = "class_section_created_at"
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// ---------- Filters ----------
	// ---------- Filters ----------
	var (
		sectionIDs []uuid.UUID
		classIDs   []uuid.UUID
		parentIDs  []uuid.UUID // 🆕 filter by class_parent
		teacherIDs []uuid.UUID // filter by school_teacher (wali / asisten)
		termIDs    []uuid.UUID // 🆕 filter by academic_term
		roomIDs    []uuid.UUID // 🆕 filter by class_room
		activeOnly *bool
	)

	// filter by section id (existing)
	if s := strings.TrimSpace(c.Query("id")); s != "" {
		ids, e := parseUUIDList(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid: "+e.Error())
		}
		sectionIDs = ids
	}

	// filter by class_id (mendukung multi)
	if s := strings.TrimSpace(c.Query("class_id")); s != "" {
		ids, e := parseUUIDList(s)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_id tidak valid: "+e.Error())
		}
		classIDs = ids
	}

	// 🆕 filter by academic_term_id / term_id (mendukung multi)
	rawTerm := strings.TrimSpace(c.Query("academic_term_id"))
	if rawTerm == "" {
		rawTerm = strings.TrimSpace(c.Query("term_id"))
	}
	if rawTerm != "" {
		ids, e := parseUUIDList(rawTerm)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "academic_term_id/term_id tidak valid: "+e.Error())
		}
		termIDs = ids
	}

	// 🆕 filter by class_room_id / room_id (mendukung multi)
	rawRoom := strings.TrimSpace(c.Query("class_room_id"))
	if rawRoom == "" {
		rawRoom = strings.TrimSpace(c.Query("room_id"))
	}
	if rawRoom != "" {
		ids, e := parseUUIDList(rawRoom)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_room_id/room_id tidak valid: "+e.Error())
		}
		roomIDs = ids
	}

	// filter by school_teacher_id (alias: teacher_id), mendukung multi
	rawTeacher := strings.TrimSpace(c.Query("school_teacher_id"))
	if rawTeacher == "" {
		rawTeacher = strings.TrimSpace(c.Query("teacher_id"))
	}
	if rawTeacher != "" {
		ids, e := parseUUIDList(rawTeacher)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_teacher_id/teacher_id tidak valid: "+e.Error())
		}
		teacherIDs = ids
	}

	// 🆕 filter by class_parent_id / parent_id (mendukung multi) via tabel classes
	rawParent := strings.TrimSpace(c.Query("class_parent_id"))
	if rawParent == "" {
		rawParent = strings.TrimSpace(c.Query("parent_id"))
	}
	if rawParent != "" {
		ids, e := parseUUIDList(rawParent)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_parent_id/parent_id tidak valid: "+e.Error())
		}
		parentIDs = ids
	}

	if s := strings.TrimSpace(c.Query("is_active")); s != "" {
		v := c.QueryBool("is_active")
		activeOnly = &v
	}

	// ------------ INCLUDE / NESTED FLAGS ------------
	// include=csst           -> include["csst"] (global pool, tanpa nested)
	// nested=csst            -> nested per class_section: class_section_subject_teacher (tanpa include.csst)
	// with_csst=1            -> legacy, dianggap nested=csst
	// include=academic_term  -> include["academic_term"] (unique list dari cache)
	// nested=academic_term   -> nested per class_section: class_section_academic_term = {id,name,slug,year,angkatan}
	includeRaw := strings.TrimSpace(c.Query("include"))
	nestedRaw := strings.TrimSpace(c.Query("nested"))

	var includeCSST bool
	var nestedCSST bool

	var includeAcademicTerm bool
	var nestedAcademicTerm bool

	// parse include=...
	if includeRaw != "" {
		parts := strings.Split(includeRaw, ",")
		for _, p := range parts {
			switch strings.TrimSpace(p) {
			case "csst", "cssts", "class_section_subject_teachers":
				includeCSST = true
			case "academic_term", "academic_terms", "term", "terms":
				includeAcademicTerm = true
			}
		}
	}

	// parse nested=...
	if nestedRaw != "" {
		parts := strings.Split(nestedRaw, ",")
		for _, p := range parts {
			switch strings.TrimSpace(p) {
			case "csst", "cssts", "class_section_subject_teachers":
				nestedCSST = true
			case "academic_term", "academic_terms", "term", "terms":
				nestedAcademicTerm = true
			}
		}
	}

	// legacy: with_csst=1 => nested=csst
	if c.QueryBool("with_csst") {
		nestedCSST = true
	}

	// apakah kita perlu query CSST sama sekali?
	queryCSST := includeCSST || nestedCSST

	// lebih eksplisit: include student_class_sections (masih boolean biasa)
	withStudentSections := c.QueryBool("with_student_class_sections")

	// Kalau filter parent dipakai → turunkan ke class_id via tabel classes
	if len(parentIDs) > 0 {
		type classRow struct {
			ID uuid.UUID `gorm:"column:class_id"`
		}

		classQ := ctrl.DB.
			Table("classes").
			Select("class_id").
			Where("class_deleted_at IS NULL").
			Where("class_school_id = ?", schoolID).
			Where("class_class_parent_id IN ?", parentIDs)

		// kalau sebelumnya sudah ada filter class_id → jadikan irisan
		if len(classIDs) > 0 {
			classQ = classQ.Where("class_id IN ?", classIDs)
		}

		var cls []classRow
		if err := classQ.Scan(&cls).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memproses filter class_parent_id")
		}

		// kalau nggak ada class yang match → langsung balikin kosong
		if len(cls) == 0 {
			pagination := helper.BuildPaginationFromOffset(0, pg.Offset, pg.Limit)
			return helper.JsonListWithInclude(c, "ok", []any{}, fiber.Map{}, pagination)
		}

		// override classIDs dengan hasil dari filter parent
		classIDs = make([]uuid.UUID, 0, len(cls))
		for _, r := range cls {
			classIDs = append(classIDs, r.ID)
		}
	}

	// ---------- Query base ----------
	tx := ctrl.DB.
		Model(&secModel.ClassSectionModel{}).
		Where("class_section_deleted_at IS NULL").
		Where("class_section_school_id = ?", schoolID)

	if len(sectionIDs) > 0 {
		tx = tx.Where("class_section_id IN ?", sectionIDs)
	}

	if len(classIDs) > 0 {
		tx = tx.Where("class_section_class_id IN ?", classIDs)
	}

	// 🆕 filter by academic_term
	if len(termIDs) > 0 {
		tx = tx.Where("class_section_academic_term_id IN ?", termIDs)
	}

	// 🆕 filter by class_room
	if len(roomIDs) > 0 {
		tx = tx.Where("class_section_class_room_id IN ?", roomIDs)
	}

	// filter wali / assistant teacher ke dua kolom FK
	if len(teacherIDs) > 0 {
		tx = tx.Where(
			"(class_section_school_teacher_id IN ? OR class_section_assistant_school_teacher_id IN ?)",
			teacherIDs, teacherIDs,
		)
	}

	if activeOnly != nil {
		tx = tx.Where("class_section_is_active = ?", *activeOnly)
	}

	if searchTerm != "" {
		s := "%" + strings.ToLower(searchTerm) + "%"
		tx = tx.Where(`
			LOWER(class_section_name) LIKE ?
			OR LOWER(class_section_code) LIKE ?
			OR LOWER(class_section_slug) LIKE ?`,
			s, s, s)
	}

	// ---------- Total ----------
	var total int64
	if err := tx.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total")
	}

	// ---------- Optional: all=1 ----------
	if c.QueryBool("all") {
		pg.Offset = 0
		pg.Limit = int(total)
	}

	// ---------- Data ----------
	tx = tx.Order(orderExpr).Limit(pg.Limit).Offset(pg.Offset)

	var rows []secModel.ClassSectionModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ---------- Build compact DTO ----------
	compacts := secDTO.FromSectionModelsToCompact(rows)

	idsInPage := make([]uuid.UUID, 0, len(compacts))
	for _, it := range compacts {
		idsInPage = append(idsInPage, it.ClassSectionID)
	}

	pagination := helper.BuildPaginationFromOffset(total, pg.Offset, pg.Limit)

	// includePayload: selalu ada "include": {} di response (bisa kosong)
	includePayload := fiber.Map{}

	// top-level include: academic_term
	if includeAcademicTerm {
		includePayload["academic_term"] = buildAcademicTermInclude(rows)
	}

	// Kalau nggak perlu CSST sama sekali, nggak perlu student sections,
	// dan nggak perlu nested academic_term → langsung balikin compact DTO + include
	if !queryCSST && !withStudentSections && !nestedAcademicTerm {
		return helper.JsonListWithInclude(c, "ok", compacts, includePayload, pagination)
	}

	// =========================================================
	//  INCLUDE: CSST & STUDENT_CLASS_SECTIONS (bisa keduanya)
	//  + nested academic_term
	// =========================================================

	// Target section IDs untuk query relasi
	targetIDs := sectionIDs
	if len(targetIDs) == 0 {
		targetIDs = idsInPage
	}

	// Map: section_id -> *model (dipakai nested academic_term)
	modelBySection := make(map[uuid.UUID]*secModel.ClassSectionModel, len(rows))
	for i := range rows {
		cs := &rows[i]
		modelBySection[cs.ClassSectionID] = cs
	}

	// Base: konversi item ke map + index by section_id
	out := make([]fiber.Map, 0, len(compacts))
	indexBySection := make(map[uuid.UUID]int, len(compacts))

	for i := range compacts {
		b, _ := json.Marshal(compacts[i])

		// ✅ pastikan map-nya selalu non-nil
		m := fiber.Map{}
		_ = json.Unmarshal(b, &m)

		// init field CSST kalau diminta nested
		if nestedCSST {
			m["class_section_subject_teacher"] = []csstDTO.CSSTItemLite{}
			m["class_section_subject_teacher_count"] = 0
			m["class_section_subject_teacher_active_count"] = 0
		}

		// init field student_class_sections kalau diminta
		if withStudentSections {
			m["class_sections_student_class_sections"] = []secDTO.StudentClassSectionResp{}
			m["class_sections_student_class_sections_count"] = 0
			m["class_sections_student_class_sections_active_count"] = 0
		}

		// init field academic_term kalau nested diminta
		if nestedAcademicTerm {
			m["class_section_academic_term"] = nil
		}

		out = append(out, m)
		indexBySection[compacts[i].ClassSectionID] = i
	}

	// ---------- Inject CSST ----------
	if queryCSST && len(targetIDs) > 0 {
		var csstRows []csstModel.ClassSectionSubjectTeacherModel
		csstQ := ctrl.DB.
			Model(&csstModel.ClassSectionSubjectTeacherModel{}).
			Where("class_section_subject_teacher_deleted_at IS NULL").
			Where("class_section_subject_teacher_school_id = ?", schoolID).
			Where("class_section_subject_teacher_class_section_id IN ?", targetIDs)

		if activeOnly != nil {
			csstQ = csstQ.Where("class_section_subject_teacher_is_active = ?", *activeOnly)
		}

		if err := csstQ.Find(&csstRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil CSST")
		}

		// nested=csst → inject per class_section
		if nestedCSST {
			// group by class_section_id
			bySection := make(map[uuid.UUID][]csstModel.ClassSectionSubjectTeacherModel, len(targetIDs))
			for i := range csstRows {
				r := csstRows[i]
				bySection[r.ClassSectionSubjectTeacherClassSectionID] =
					append(bySection[r.ClassSectionSubjectTeacherClassSectionID], r)
			}

			for secID, list := range bySection {
				idx, ok := indexBySection[secID]
				if !ok {
					continue
				}
				lite := csstDTO.CSSTLiteSliceFromModels(list)
				activeCnt := 0
				for _, it := range list {
					if it.ClassSectionSubjectTeacherIsActive {
						activeCnt++
					}
				}
				out[idx]["class_section_subject_teacher"] = lite
				out[idx]["class_section_subject_teacher_count"] = len(lite)
				out[idx]["class_section_subject_teacher_active_count"] = activeCnt
			}
		}

		// include=csst → top-level include.csst (semua csst di page ini)
		if includeCSST {
			if len(csstRows) > 0 {
				includePayload["csst"] = csstDTO.CSSTLiteSliceFromModels(csstRows)
			} else {
				includePayload["csst"] = []csstDTO.CSSTItemLite{}
			}
		}
	}

	// ---------- Inject student_class_sections ----------
	if withStudentSections && len(targetIDs) > 0 {
		var scsRows []secModel.StudentClassSection

		scsQ := ctrl.DB.
			Model(&secModel.StudentClassSection{}).
			Where("student_class_section_deleted_at IS NULL").
			Where("student_class_section_school_id = ?", schoolID).
			Where("student_class_section_section_id IN ?", targetIDs)

		// kalau is_active=true → hanya enrolment status=active
		if activeOnly != nil && *activeOnly {
			scsQ = scsQ.Where("student_class_section_status = ?", secModel.StudentClassSectionActive)
		}

		if err := scsQ.Find(&scsRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil student_class_sections")
		}

		bySection := make(map[uuid.UUID][]secModel.StudentClassSection, len(targetIDs))
		for i := range scsRows {
			r := scsRows[i]
			bySection[r.StudentClassSectionSectionID] =
				append(bySection[r.StudentClassSectionSectionID], r)
		}

		for secID, list := range bySection {
			idx, ok := indexBySection[secID]
			if !ok {
				continue
			}

			dtos := make([]secDTO.StudentClassSectionResp, 0, len(list))
			activeCnt := 0

			for i := range list {
				dtoItem := secDTO.FromModel(&list[i])
				dtos = append(dtos, dtoItem)
				if dtoItem.StudentClassSectionStatus == string(secModel.StudentClassSectionActive) {
					activeCnt++
				}
			}

			out[idx]["class_sections_student_class_sections"] = dtos
			out[idx]["class_sections_student_class_sections_count"] = len(dtos)
			out[idx]["class_sections_student_class_sections_active_count"] = activeCnt
		}
	}

	// ---------- Inject academic_term (nested) ----------
	if nestedAcademicTerm {
		for secID, idx := range indexBySection {
			cs, ok := modelBySection[secID]
			if !ok {
				continue
			}
			if at := academicTermLiteFromSectionModel(cs); at != nil {
				out[idx]["class_section_academic_term"] = at
			}
		}
	}

	return helper.JsonListWithInclude(c, "ok", out, includePayload, pagination)
}
