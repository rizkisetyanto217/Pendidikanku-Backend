// file: internals/features/school/classes/class_sections/controller/student_class_section_list.go
package controller

import (
	"strings"

	csstDto "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	dto "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

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

// GET /api/u/student-class-sections/list
// ?student=me|<uuid,uuid2,...>
// ?section_id=<uuid,uuid2,...>        // alias lama
// ?class_section_id=<uuid,uuid2,...>  // alias baru
// ?status=active|inactive|completed
// ?q=...
// ?include=class_sections,csst
// ?nested=class_sections,csst
// ?view=compact|full|class_sections
// ?page=1&size=20
func (ctl *StudentClassSectionController) List(c *fiber.Ctx) error {
	// 1) school dari TOKEN
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// 2) cek apakah caller staff (guru/dkm/admin/bendahara)
	isStaff := (helperAuth.EnsureStaffSchool(c, schoolID) == nil)

	// 3) ambil user_id dari token (perlu untuk "me")
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	tx := ctl.DB.WithContext(c.Context())

	// ----------------- PARSE VIEW -----------------
	view := strings.ToLower(strings.TrimSpace(c.Query("view"))) // "", "compact", "full", "class_sections"
	viewCompact := view == "compact"
	viewClassSections := view == "class_sections" || view == "sections" || view == "class_section"

	// ----------------- PARSE INCLUDE -----------------
	includeRaw := strings.TrimSpace(c.Query("include"))
	var includeClassSections bool
	var includeCSST bool

	if includeRaw != "" {
		parts := strings.Split(includeRaw, ",")
		for _, p := range parts {
			switch strings.TrimSpace(p) {
			case "class_sections":
				includeClassSections = true
			case "csst", "cssts", "class_section_subject_teachers":
				includeCSST = true
			}
		}
	}

	// ----------------- PARSE NESTED -----------------
	nestedRaw := strings.TrimSpace(c.Query("nested"))
	var nestedClassSections bool
	var nestedCSST bool

	if nestedRaw != "" {
		parts := strings.Split(nestedRaw, ",")
		for _, p := range parts {
			switch strings.TrimSpace(strings.ToLower(p)) {
			case "class_sections", "sections", "class_section":
				nestedClassSections = true
			case "csst", "cssts", "class_section_subject_teachers":
				nestedCSST = true
			}
		}
	}

	// nested & include nggak saling override,
	// tapi kalau nestedClassSections / viewClassSections → pasti butuh data class_sections
	if viewClassSections || nestedClassSections {
		includeClassSections = true
	}

	// ----------------- RESOLVE student -----------------
	rawSchoolStudent := strings.TrimSpace(c.Query("student"))

	var schoolStudentIDs []uuid.UUID

	if rawSchoolStudent == "" {
		if !isStaff {
			rawSchoolStudent = "me"
		}
	}

	if rawSchoolStudent == "me" {
		// MODE "ME"
		usersProfileID, err := getUsersProfileID(tx, userID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Profil user belum ada. Lengkapi profil terlebih dahulu.")
		}

		msID, err := getOrCreateSchoolStudentWithCaches(c.Context(), tx, schoolID, usersProfileID, nil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mendapatkan status student")
		}
		schoolStudentIDs = []uuid.UUID{msID}

	} else if rawSchoolStudent != "" {
		// MODE daftar UUID explicit
		ids, err := parseUUIDList(rawSchoolStudent)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "student tidak valid: "+err.Error())
		}
		schoolStudentIDs = ids

		if !isStaff && len(ids) > 0 {
			usersProfileID, err := getUsersProfileID(tx, userID)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Profil user belum ada. Lengkapi profil terlebih dahulu.")
			}

			var cnt int64
			if err := tx.Table("school_students").
				Where(`
					school_student_id IN ?
					AND school_student_school_id = ?
					AND school_student_user_profile_id = ?
					AND school_student_deleted_at IS NULL
				`, ids, schoolID, usersProfileID).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal validasi school_student")
			}
			if cnt != int64(len(ids)) {
				return helper.JsonError(c, fiber.StatusForbidden, "Beberapa student bukan milik Anda / beda tenant")
			}
		}
	}

	// ----------------- FILTER SECTION & STATUS & SEARCH -----------------
	var (
		secIDs     []uuid.UUID
		status     string
		searchTerm = strings.TrimSpace(c.Query("q"))
	)

	if raw := strings.TrimSpace(c.Query("section_id")); raw != "" {
		ids, e := parseUUIDList(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "section_id tidak valid: "+e.Error())
		}
		secIDs = append(secIDs, ids...)
	}

	if raw := strings.TrimSpace(c.Query("class_section_id")); raw != "" {
		ids, e := parseUUIDList(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_section_id tidak valid: "+e.Error())
		}
		secIDs = append(secIDs, ids...)
	}

	if len(secIDs) > 1 {
		tmpSet := make(map[uuid.UUID]struct{}, len(secIDs))
		uniq := make([]uuid.UUID, 0, len(secIDs))
		for _, id := range secIDs {
			if _, ok := tmpSet[id]; !ok {
				tmpSet[id] = struct{}{}
				uniq = append(uniq, id)
			}
		}
		secIDs = uniq
	}

	if s := strings.TrimSpace(c.Query("status")); s != "" {
		status = s
	}

	page, size := getPageSize(c)
	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}

	// BASE QUERY ke student_class_sections
	q := tx.Model(&classSectionModel.StudentClassSection{}).
		Where(`
			student_class_section_school_id = ?
			AND student_class_section_deleted_at IS NULL
		`, schoolID)

	if len(schoolStudentIDs) > 0 {
		q = q.Where("student_class_section_school_student_id IN ?", schoolStudentIDs)
	}
	if len(secIDs) > 0 {
		q = q.Where("student_class_section_section_id IN ?", secIDs)
	}
	if status != "" {
		q = q.Where("student_class_section_status = ?", status)
	}
	if searchTerm != "" {
		s := "%" + strings.ToLower(searchTerm) + "%"
		q = q.Where(`
			LOWER(COALESCE(student_class_section_user_profile_name_cache, '')) LIKE ?
			OR LOWER(student_class_section_section_slug_cache) LIKE ?
			OR LOWER(COALESCE(student_class_section_student_code_cache, '')) LIKE ?
		`, s, s, s)
	}

	// COUNT
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// DATA
	var rows []classSectionModel.StudentClassSection
	if err := q.
		Order("student_class_section_created_at DESC").
		Limit(size).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	pagination := helper.BuildPaginationFromOffset(total, offset, size)

	// ================= VIEW=COMPACT =================
	if viewCompact {
		out := dto.FromModelsStudentClassSectionCompactWithSchoolTime(c, rows)
		return helper.JsonListWithInclude(c, "OK", out, nil, pagination)
	}

	// ======= TANPA include & TANPA nested =======
	if !includeClassSections && !includeCSST && !nestedClassSections && !nestedCSST {
		out := make([]dto.StudentClassSectionResp, 0, len(rows))
		for i := range rows {
			out = append(out, dto.FromModelWithSchoolTime(c, &rows[i]))
		}
		return helper.JsonListWithInclude(c, "OK", out, nil, pagination)
	}

	// =====================================================================
	//  PERSIAPAN DATA CLASS_SECTION & CSST (untuk include / nested)
	// =====================================================================

	// 1) kumpulkan section_id dari rows
	secIDSet := make(map[uuid.UUID]struct{})
	for i := range rows {
		secIDSet[rows[i].StudentClassSectionSectionID] = struct{}{}
	}
	secIDs = make([]uuid.UUID, 0, len(secIDSet))
	for id := range secIDSet {
		secIDs = append(secIDs, id)
	}

	// 2) Map section_id → ClassSectionCompactResponse
	classSectionMap := make(map[uuid.UUID]*dto.ClassSectionCompactResponse)

	if includeClassSections || viewClassSections || nestedClassSections {
		if len(secIDs) > 0 {
			var secRows []classSectionModel.ClassSectionModel
			if err := ctl.DB.WithContext(c.Context()).
				Model(&classSectionModel.ClassSectionModel{}).
				Where(`
					class_section_id IN ?
					AND class_section_school_id = ?
					AND class_section_deleted_at IS NULL
				`, secIDs, schoolID).
				Find(&secRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data class sections")
			}

			for i := range secRows {
				csModel := &secRows[i]
				compact := dto.FromModelClassSectionToCompact(csModel)
				csCopy := compact
				classSectionMap[csModel.ClassSectionID] = &csCopy
			}
		}
	}

	// 3) CSST: flat list + map per section
	var csstList []csstDto.ClassSectionSubjectTeacherCompactResponse
	csstBySection := make(map[uuid.UUID][]csstDto.ClassSectionSubjectTeacherCompactResponse)
	needCSST := includeCSST || nestedCSST

	if needCSST && len(secIDs) > 0 {
		var csstRows []csstModel.ClassSectionSubjectTeacherModel
		if err := ctl.DB.WithContext(c.Context()).
			Model(&csstModel.ClassSectionSubjectTeacherModel{}).
			Where(`
				class_section_subject_teacher_school_id = ?
				AND class_section_subject_teacher_deleted_at IS NULL
				AND class_section_subject_teacher_class_section_id IN ?
			`, schoolID, secIDs).
			Find(&csstRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data subject teachers")
		}

		compactList := csstDto.FromClassSectionSubjectTeacherModelsCompact(csstRows)
		csstList = compactList

		for i, row := range csstRows {
			secID := row.ClassSectionSubjectTeacherClassSectionID
			if secID == uuid.Nil {
				continue
			}
			csstBySection[secID] = append(csstBySection[secID], compactList[i])
		}
	}

	// ================= include payload =================
	includePayload := fiber.Map{}
	noNested := !nestedClassSections && !nestedCSST

	if includeClassSections && noNested {
		classSectionList := make([]dto.ClassSectionCompactResponse, 0, len(classSectionMap))
		for _, cs := range classSectionMap {
			if cs == nil {
				continue
			}
			classSectionList = append(classSectionList, *cs)
		}
		includePayload["class_sections"] = classSectionList
	}
	if includeCSST && noNested {
		includePayload["csst"] = csstList
	}

	// =====================================================================
	//  MODE nested=class_sections,csst
	// =====================================================================
	if nestedClassSections {
		type ClassSectionNested struct {
			dto.ClassSectionCompactResponse
			StudentClassSections        []dto.StudentClassSectionResp                       `json:"student_class_sections"`
			ClassSectionSubjectTeachers []csstDto.ClassSectionSubjectTeacherCompactResponse `json:"class_section_subject_teachers,omitempty"`
		}

		nestedMap := make(map[uuid.UUID]*ClassSectionNested)

		// init dari classSectionMap
		for secID, cs := range classSectionMap {
			if cs == nil {
				continue
			}
			nestedMap[secID] = &ClassSectionNested{
				ClassSectionCompactResponse: *cs,
				StudentClassSections:        []dto.StudentClassSectionResp{},
			}
		}

		// isi students per section
		for i := range rows {
			secID := rows[i].StudentClassSectionSectionID
			secNested, ok := nestedMap[secID]
			if !ok {
				continue
			}
			secNested.StudentClassSections = append(
				secNested.StudentClassSections,
				dto.FromModelWithSchoolTime(c, &rows[i]),
			)
		}

		// isi CSST per section (pakai csstBySection)
		if nestedCSST {
			for secID, secNested := range nestedMap {
				if list, ok := csstBySection[secID]; ok && len(list) > 0 {
					secNested.ClassSectionSubjectTeachers = append(secNested.ClassSectionSubjectTeachers, list...)
				}
			}
		}

		out := make([]ClassSectionNested, 0, len(nestedMap))
		for _, v := range nestedMap {
			if v == nil {
				continue
			}
			out = append(out, *v)
		}

		// 🔥 nested → TIDAK kirim include sama sekali
		return helper.JsonListWithInclude(c, "OK", out, nil, pagination)
	}

	// =====================================================================
	//  MODE view=class_sections → cuma list section
	// =====================================================================
	if viewClassSections {
		list := make([]dto.ClassSectionCompactResponse, 0, len(classSectionMap))
		for _, cs := range classSectionMap {
			if cs == nil {
				continue
			}
			list = append(list, *cs)
		}
		return helper.JsonListWithInclude(c, "OK", list, includePayload, pagination)
	}

	// =====================================================================
	//  MODE default: per-enrollment + nested class_section di tiap row
	// =====================================================================
	type StudentClassSectionWithClassSectionResp struct {
		dto.StudentClassSectionResp
		ClassSection *dto.ClassSectionCompactResponse `json:"class_section,omitempty"`
	}

	out := make([]StudentClassSectionWithClassSectionResp, 0, len(rows))
	for i := range rows {
		base := dto.FromModelWithSchoolTime(c, &rows[i])

		var included *dto.ClassSectionCompactResponse
		if cs, ok := classSectionMap[rows[i].StudentClassSectionSectionID]; ok {
			included = cs
		}

		out = append(out, StudentClassSectionWithClassSectionResp{
			StudentClassSectionResp: base,
			ClassSection:            included,
		})
	}

	return helper.JsonListWithInclude(c, "OK", out, includePayload, pagination)
}
