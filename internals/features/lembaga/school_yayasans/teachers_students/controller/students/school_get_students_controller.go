// file: internals/features/school/students/controller/school_student_list_controller.go
package controller

import (
	"strings"
	"time"

	dto "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/dto"
	model "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	classSectionDTO "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	csstDTO "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	userProfileDTO "madinahsalam_backend/internals/features/users/users/dto"
	userProfileModel "madinahsalam_backend/internals/features/users/users/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/a/school-students
// GET /api/u/school-students
func (h *SchoolStudentController) List(c *fiber.Ctx) error {
	// 0) Pastikan DB di locals (dipakai helper lain)
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}

	// 1) Ambil school_id dari TOKEN
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// 2) Enforce role DKM/Admin
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}

	// 3) Mode: all / compact (mode hanya ngatur shape dasar, bukan include)
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode"))) // "", "all", "compact"

	// ==============================
	// INCLUDE FLAGS (berlaku utk semua mode)
	// ==============================
	var (
		wantSections bool
		wantCSST     bool
		wantProfile  bool
	)

	// a) nested=class-sections,csst  (legacy / alias)
	nested := strings.ToLower(strings.TrimSpace(c.Query("nested")))
	if nested != "" {
		for _, part := range strings.Split(nested, ",") {
			p := strings.TrimSpace(part)
			switch p {
			case "sections", "class-sections", "class_sections", "classsection":
				wantSections = true
			case "csst", "class-section-subject-teachers", "class_section_subject_teachers", "subject-teachers", "subject_teachers":
				wantCSST = true
			}
		}
	}

	// b) include=user_profiles,csst,class_sections (baru, berlaku utk compact & full)
	includeRaw := strings.ToLower(strings.TrimSpace(c.Query("include")))
	if includeRaw != "" {
		for _, part := range strings.Split(includeRaw, ",") {
			p := strings.TrimSpace(part)
			switch p {
			case "user_profile", "user_profiles", "profile", "profiles", "user":
				wantProfile = true
			case "sections", "class-sections", "class_sections", "classsection":
				wantSections = true
			case "csst", "class-section-subject-teachers", "class_section_subject_teachers", "subject-teachers", "subject_teachers":
				wantCSST = true
			}
		}
	}

	// ==============================
	// Sorting + Pagination
	// ==============================
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)
	allowedSort := map[string]string{
		"created_at": "school_student_created_at",
		"updated_at": "school_student_updated_at",
		"code":       "school_student_code",
		"status":     "school_student_status",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid sort_by")
	}
	orderClause = strings.TrimPrefix(orderClause, "ORDER BY ")

	// ==============================
	// Filters
	// ==============================
	search := strings.TrimSpace(c.Query("search"))
	var (
		userProfIDStr = strings.TrimSpace(c.Query("user_profile_id"))
		idStr         = strings.TrimSpace(c.Query("id"))
		createdGe     = strings.TrimSpace(c.Query("created_ge"))
		createdLe     = strings.TrimSpace(c.Query("created_le"))
	)

	var (
		userProfileID uuid.UUID
		rowID         uuid.UUID
	)
	if userProfIDStr != "" {
		v, err := uuid.Parse(userProfIDStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "user_profile_id invalid")
		}
		userProfileID = v
	}
	if idStr != "" {
		v, err := uuid.Parse(idStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id invalid")
		}
		rowID = v
	}

	// status_in ?
	statusIn := getMultiQuery(c, "status_in")
	normStatus := make([]string, 0, len(statusIn))
	for _, s := range statusIn {
		s = strings.ToLower(strings.TrimSpace(s))
		switch model.SchoolStudentStatus(s) {
		case model.SchoolStudentActive, model.SchoolStudentInactive, model.SchoolStudentAlumni:
			normStatus = append(normStatus, s)
		}
	}

	q := h.DB.Model(&model.SchoolStudentModel{}).
		Where("school_student_school_id = ?", schoolID)

	if rowID != uuid.Nil {
		q = q.Where("school_student_id = ?", rowID)
	}
	if userProfileID != uuid.Nil {
		q = q.Where("school_student_user_profile_id = ?", userProfileID)
	}
	if len(normStatus) > 0 {
		q = q.Where("school_student_status IN ?", normStatus)
	}

	// created_at range
	const layout = time.RFC3339
	if createdGe != "" {
		t, err := time.Parse(layout, createdGe)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "created_ge invalid (use RFC3339)")
		}
		q = q.Where("school_student_created_at >= ?", t)
	}
	if createdLe != "" {
		t, err := time.Parse(layout, createdLe)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "created_le invalid (use RFC3339)")
		}
		q = q.Where("school_student_created_at <= ?", t)
	}

	// search code/note/name
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		q = q.Where(`
			LOWER(COALESCE(school_student_code, '')) LIKE ? OR
			LOWER(COALESCE(school_student_note, '')) LIKE ? OR
			LOWER(COALESCE(school_student_user_profile_name_cache, '')) LIKE ?
		`, like, like, like)
	}

	// 5) Count
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 6) Fetch rows
	var rows []model.SchoolStudentModel
	if err := q.Order(orderClause).Offset(p.Offset()).Limit(p.Limit()).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// pagination info
	pg := helper.BuildPaginationFromOffset(total, p.Offset(), p.Limit())

	// Kalau tidak ada data → langsung pulang (include apa pun juga akan kosong)
	if len(rows) == 0 {
		if mode == "compact" {
			empty := []dto.SchoolStudentCompact{}
			return helper.JsonListWithInclude(c, "ok", empty, nil, pg)
		}
		empty := []dto.SchoolStudentResp{}
		return helper.JsonListWithInclude(c, "ok", empty, nil, pg)
	}

	// ======================================
	// Common: studentIDs
	// ======================================
	studentIDs := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		studentIDs = append(studentIDs, rows[i].SchoolStudentID)
	}

	// ======================================
	// Early exit: kalau tidak minta include apa pun
	// (tetap pakai JsonListWithInclude supaya ada "include": {})
	// ======================================
	if !wantProfile && !wantSections && !wantCSST {
		if mode == "compact" {
			comp := make([]dto.SchoolStudentCompact, 0, len(rows))
			for i := range rows {
				comp = append(comp, dto.ToSchoolStudentCompact(c, &rows[i]))
			}
			return helper.JsonListWithInclude(c, "ok", comp, nil, pg)
		}

		baseResp := make([]dto.SchoolStudentResp, 0, len(rows))
		for i := range rows {
			baseResp = append(baseResp, dto.FromModel(c, &rows[i]))
		}
		return helper.JsonListWithInclude(c, "ok", baseResp, nil, pg)
	}

	// ======================================
	// INCLUDE: USER PROFILE (FULL DTO)
	// ======================================
	profileMap := make(map[uuid.UUID]userProfileDTO.UsersProfileDTO)

	if wantProfile {
		// kumpulkan profile_id
		profileIDsSet := make(map[uuid.UUID]struct{}, len(rows))
		for i := range rows {
			if rows[i].SchoolStudentUserProfileID != uuid.Nil {
				profileIDsSet[rows[i].SchoolStudentUserProfileID] = struct{}{}
			}
		}
		profileIDs := make([]uuid.UUID, 0, len(profileIDsSet))
		for id := range profileIDsSet {
			profileIDs = append(profileIDs, id)
		}

		if len(profileIDs) > 0 {
			var profRows []userProfileModel.UserProfileModel
			if err := h.DB.
				Where("user_profile_id IN ?", profileIDs).
				Where("user_profile_deleted_at IS NULL").
				Find(&profRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
			}

			for _, pr := range profRows {
				dtoProfile := userProfileDTO.ToUsersProfileDTO(pr)
				profileMap[pr.UserProfileID] = dtoProfile
			}
		}
	}

	// ======================================
	// INCLUDE: CLASS SECTIONS (compact, via student_class_sections)
	// ======================================
	sectionMapCompactGlobal := make(map[uuid.UUID]classSectionDTO.ClassSectionCompactResponse)

	if wantSections && len(studentIDs) > 0 {
		type studentSectionRow struct {
			SchoolStudentID uuid.UUID `gorm:"column:student_class_section_school_student_id"`
			ClassSectionID  uuid.UUID `gorm:"column:student_class_section_section_id"`
		}

		var ssRows []studentSectionRow
		if err := h.DB.
			Table("student_class_sections").
			Select("student_class_section_school_student_id, student_class_section_section_id").
			Where("student_class_section_school_id = ?", schoolID).
			Where("student_class_section_deleted_at IS NULL").
			Where("student_class_section_school_student_id IN ?", studentIDs).
			Find(&ssRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil relasi section siswa: "+err.Error())
		}

		// unique section IDs
		sectionIDSet := make(map[uuid.UUID]struct{})
		for _, r := range ssRows {
			sectionIDSet[r.ClassSectionID] = struct{}{}
		}
		sectionIDs := make([]uuid.UUID, 0, len(sectionIDSet))
		for id := range sectionIDSet {
			sectionIDs = append(sectionIDs, id)
		}

		if len(sectionIDs) > 0 {
			var secRows []classSectionModel.ClassSectionModel
			if err := h.DB.
				Where("class_section_school_id = ? AND class_section_deleted_at IS NULL", schoolID).
				Where("class_section_id IN ?", sectionIDs).
				Find(&secRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil data class_section: "+err.Error())
			}

			for i := range secRows {
				cmp := classSectionDTO.FromModelClassSectionToCompact(&secRows[i])
				sectionMapCompactGlobal[secRows[i].ClassSectionID] = cmp
			}
		}
	}

	// ======================================
	// INCLUDE: CSST (compact)
	// ======================================
	csstMapCompactGlobal := make(map[uuid.UUID]csstDTO.ClassSectionSubjectTeacherCompactResponse)

	if wantCSST && len(studentIDs) > 0 {
		type studentCSSTRow struct {
			SchoolStudentID uuid.UUID `gorm:"column:student_csst_student_id"`
			CSSTID          uuid.UUID `gorm:"column:student_csst_csst_id"`
		}

		var scRows []studentCSSTRow
		if err := h.DB.
			Table("student_class_section_subject_teachers").
			Select("student_csst_student_id, student_csst_csst_id").
			Where("student_csst_school_id = ?", schoolID).
			Where("student_csst_deleted_at IS NULL").
			Where("student_csst_student_id IN ?", studentIDs).
			Find(&scRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil relasi CSST siswa: "+err.Error())
		}

		csstIDSet := make(map[uuid.UUID]struct{})
		for _, r := range scRows {
			csstIDSet[r.CSSTID] = struct{}{}
		}
		csstIDs := make([]uuid.UUID, 0, len(csstIDSet))
		for id := range csstIDSet {
			csstIDs = append(csstIDs, id)
		}

		if len(csstIDs) > 0 {
			var csstRows []csstModel.ClassSectionSubjectTeacherModel
			if err := h.DB.
				Where("class_section_subject_teacher_school_id = ? AND class_section_subject_teacher_deleted_at IS NULL", schoolID).
				Where("class_section_subject_teacher_id IN ?", csstIDs).
				Find(&csstRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil data CSST: "+err.Error())
			}

			for i := range csstRows {
				cmp := csstDTO.FromClassSectionSubjectTeacherModelCompact(csstRows[i])
				csstMapCompactGlobal[csstRows[i].ClassSectionSubjectTeacherID] = cmp
			}
		}
	}

	// ======================================
	// BUILD INCLUDE PAYLOAD (top-level)
	// ======================================
	includePayload := fiber.Map{}

	if wantProfile && len(profileMap) > 0 {
		profiles := make([]userProfileDTO.UsersProfileDTO, 0, len(profileMap))
		for _, v := range profileMap {
			profiles = append(profiles, v)
		}
		includePayload["user_profiles"] = profiles
	}

	if wantSections && len(sectionMapCompactGlobal) > 0 {
		sections := make([]classSectionDTO.ClassSectionCompactResponse, 0, len(sectionMapCompactGlobal))
		for _, v := range sectionMapCompactGlobal {
			sections = append(sections, v)
		}
		includePayload["class_sections"] = sections
	}

	if wantCSST && len(csstMapCompactGlobal) > 0 {
		cssts := make([]csstDTO.ClassSectionSubjectTeacherCompactResponse, 0, len(csstMapCompactGlobal))
		for _, v := range csstMapCompactGlobal {
			cssts = append(cssts, v)
		}
		includePayload["class_section_subject_teachers"] = cssts
	}

	// ======================================
	// BUILD OUTPUT DATA (TANPA nested user_profile / sections / csst)
	// ======================================
	if mode == "compact" {
		comp := make([]dto.SchoolStudentCompact, 0, len(rows))
		for i := range rows {
			comp = append(comp, dto.ToSchoolStudentCompact(c, &rows[i]))
		}
		return helper.JsonListWithInclude(c, "ok", comp, includePayload, pg)
	}

	baseResp := make([]dto.SchoolStudentResp, 0, len(rows))
	for i := range rows {
		baseResp = append(baseResp, dto.FromModel(c, &rows[i]))
	}
	return helper.JsonListWithInclude(c, "ok", baseResp, includePayload, pg)
}
