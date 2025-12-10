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
// ?school_student_id=me|<uuid,uuid2,...>
// ?section_id=<uuid,uuid2,...>        // alias lama
// ?class_section_id=<uuid,uuid2,...>  // alias baru
// ?status=active|inactive|completed   // status enrolment siswa
// ?q=...
// ?include=class_sections,csst
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

	// kalau minta CSST, otomatis butuh class_sections juga
	if includeCSST {
		includeClassSections = true
	}
	// kalau view=class_sections, otomatis butuh class_sections
	if viewClassSections {
		includeClassSections = true
	}

	// ----------------- RESOLVE school_student_id -----------------
	rawSchoolStudent := strings.TrimSpace(c.Query("school_student_id"))

	var schoolStudentIDs []uuid.UUID

	if rawSchoolStudent == "" {
		// kalau kosong:
		// - staff  → boleh lihat semua (tanpa filter student)
		// - non-staff → auto "me"
		if !isStaff {
			rawSchoolStudent = "me"
		}
	}

	if rawSchoolStudent == "me" {
		// ==== MODE "ME" → resolve dari user_id ====
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
		// ==== MODE FILTER EXPLICIT UUID LIST ====
		ids, err := parseUUIDList(rawSchoolStudent)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_student_id tidak valid: "+err.Error())
		}
		schoolStudentIDs = ids

		// kalau bukan staff, pastikan id-id ini memang milik dia
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
				return helper.JsonError(c, fiber.StatusForbidden, "Beberapa school_student_id bukan milik Anda / beda tenant")
			}
		}
	}

	// ----------------- FILTER SECTION & STATUS & SEARCH -----------------
	var (
		secIDs     []uuid.UUID
		status     string // status enrolment (student_class_section_status_enum)
		searchTerm = strings.TrimSpace(c.Query("q"))
	)

	// 🔹 section_id (lama)
	if raw := strings.TrimSpace(c.Query("section_id")); raw != "" {
		ids, e := parseUUIDList(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "section_id tidak valid: "+e.Error())
		}
		secIDs = append(secIDs, ids...)
	}

	// 🔹 class_section_id (alias baru)
	if raw := strings.TrimSpace(c.Query("class_section_id")); raw != "" {
		ids, e := parseUUIDList(raw)
		if e != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_section_id tidak valid: "+e.Error())
		}
		secIDs = append(secIDs, ids...)
	}

	// (opsional) hilangkan duplikat secIDs
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
		// biarkan apa adanya (active/inactive/completed), validasi hard di DB
		status = s
	}

	// pagination (masih pakai getPageSize kamu)
	page, size := getPageSize(c)
	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}

	// BASE QUERY ke tabel student_class_sections
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

	// pagination style standar
	pagination := helper.BuildPaginationFromOffset(total, offset, size)

	// =====================================================================
	//  MODE VIEW=COMPACT
	// =====================================================================
	if viewCompact {
		out := dto.FromModelsStudentClassSectionCompact(rows)
		// include selalu ada, minimal {}
		return helper.JsonListWithInclude(c, "OK", out, nil, pagination)
	}

	// =====================================================================
	//  MODE TANPA INCLUDE PARAM (tidak butuh class_sections/csst)
	// =====================================================================
	if !includeClassSections && !includeCSST {
		out := make([]dto.StudentClassSectionResp, 0, len(rows))
		for i := range rows {
			out = append(out, dto.FromModel(&rows[i]))
		}
		return helper.JsonListWithInclude(c, "OK", out, nil, pagination)
	}

	// =====================================================================
	//  MODE include=class_sections / csst → FULL + nested
	// =====================================================================

	// 1) Kumpulkan section_id unik dari hasil query
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

	// 3) Query class_sections
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
			// copy ke variabel baru supaya pointer stabil
			csCopy := compact
			classSectionMap[csModel.ClassSectionID] = &csCopy
		}
	}

	// 4) Query CSST & kumpulkan ke slice flat (pakai DTO compact bawaan CSST)
	var csstList []csstDto.ClassSectionSubjectTeacherCompactResponse

	if includeCSST && len(secIDs) > 0 {
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

		// Reuse mapper compact dari paket CSST DTO
		csstList = csstDto.FromClassSectionSubjectTeacherModelsCompact(csstRows)
	}

	// siapkan includePayload (selalu ada di response)
	includePayload := fiber.Map{}

	// kalau diminta includeClassSections → flatten list ke include.class_sections
	if includeClassSections {
		classSectionList := make([]dto.ClassSectionCompactResponse, 0, len(classSectionMap))
		for _, cs := range classSectionMap {
			if cs == nil {
				continue
			}
			classSectionList = append(classSectionList, *cs)
		}
		includePayload["class_sections"] = classSectionList
	}

	// kalau diminta includeCSST → pakai csstList flat (compact)
	if includeCSST {
		includePayload["csst"] = csstList
	}

	// 5) MODE view=class_sections → balikin hanya daftar class_section (+ include di atas)
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

	// 6) MODE default nested: per-enrollment + nested class_section (+ csst)
	type StudentClassSectionWithClassSectionResp struct {
		dto.StudentClassSectionResp
		ClassSection *dto.ClassSectionCompactResponse `json:"class_section,omitempty"`
	}

	out := make([]StudentClassSectionWithClassSectionResp, 0, len(rows))
	for i := range rows {
		base := dto.FromModel(&rows[i])

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
