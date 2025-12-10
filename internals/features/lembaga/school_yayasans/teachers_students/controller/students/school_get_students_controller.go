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

	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	csstDTO "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/a/school-students
// GET /api/a/school-students
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

	// --- mode: all / compact ---
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode"))) // "", "all", "compact"

	// 🔥 NEW: nested=class-sections,csst
	nested := strings.ToLower(strings.TrimSpace(c.Query("nested")))
	wantSections := false
	wantCSST := false
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

	// 3) Sorting + Pagination
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

	// 4) Filters
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

	// =========================
	// MODE: COMPACT
	// =========================
	if mode == "compact" {
		// Kalau nggak minta nested apa-apa → tetap behaviour lama
		if !wantSections && !wantCSST {
			comp := make([]dto.SchoolStudentCompact, 0, len(rows))
			for i := range rows {
				comp = append(comp, dto.ToSchoolStudentCompact(&rows[i]))
			}
			// compact tidak butuh join user_profile lagi: sudah pakai cache
			return helper.JsonList(c, "ok", comp, pg)
		}

		// 🔥 NEW: compact + nested
		type SchoolStudentCompactWithNested struct {
			dto.SchoolStudentCompact   `json:",inline"`
			ClassSections              []classSectionDTO.ClassSectionCompactResponse       `json:"class_sections,omitempty"`
			ClassSectionSubjectTeaches []csstDTO.ClassSectionSubjectTeacherCompactResponse `json:"class_section_subject_teachers,omitempty"`
		}

		// base compact dulu
		baseComp := make([]dto.SchoolStudentCompact, 0, len(rows))
		studentIDs := make([]uuid.UUID, 0, len(rows))
		for i := range rows {
			baseComp = append(baseComp, dto.ToSchoolStudentCompact(&rows[i]))
			studentIDs = append(studentIDs, rows[i].SchoolStudentID)
		}

		// Map studentID -> index
		idxByStudent := make(map[uuid.UUID]int, len(studentIDs))
		for i, id := range studentIDs {
			idxByStudent[id] = i
		}

		// Prepare result slice
		result := make([]SchoolStudentCompactWithNested, len(baseComp))
		for i := range baseComp {
			result[i].SchoolStudentCompact = baseComp[i]
		}

		// ============================
		// NESTED: CLASS SECTIONS (compact)
		// ============================
		if wantSections && len(studentIDs) > 0 {
			// Sesuaikan nama tabel & kolom join dengan schema-mu ya bang.
			// Di sini diasumsikan ada table: class_section_students
			// dengan kolom:
			//   class_section_student_school_id
			//   class_section_student_school_student_id
			//   class_section_student_class_section_id
			//   class_section_student_deleted_at
			type studentSectionRow struct {
				SchoolStudentID uuid.UUID `gorm:"column:class_section_student_school_student_id"`
				ClassSectionID  uuid.UUID `gorm:"column:class_section_student_class_section_id"`
			}

			var ssRows []studentSectionRow
			if err := h.DB.
				Table("class_section_students").
				Select("class_section_student_school_student_id, class_section_student_class_section_id").
				Where("class_section_student_school_id = ?", schoolID).
				Where("class_section_student_deleted_at IS NULL").
				Where("class_section_student_school_student_id IN ?", studentIDs).
				Find(&ssRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil relasi section siswa: "+err.Error())
			}

			// Kumpulkan unique section IDs
			sectionIDSet := make(map[uuid.UUID]struct{})
			for _, r := range ssRows {
				sectionIDSet[r.ClassSectionID] = struct{}{}
			}
			sectionIDs := make([]uuid.UUID, 0, len(sectionIDSet))
			for id := range sectionIDSet {
				sectionIDs = append(sectionIDs, id)
			}

			// Load class_sections penuh → mapping ke compact
			sectionMapCompact := make(map[uuid.UUID]classSectionDTO.ClassSectionCompactResponse, len(sectionIDs))
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
					sectionMapCompact[secRows[i].ClassSectionID] = cmp
				}
			}

			// Distribusi ke tiap siswa
			for _, r := range ssRows {
				sec, ok := sectionMapCompact[r.ClassSectionID]
				if !ok {
					continue
				}
				if idx, ok := idxByStudent[r.SchoolStudentID]; ok {
					result[idx].ClassSections = append(result[idx].ClassSections, sec)
				}
			}
		}

		// ============================
		// NESTED: CSST (compact)
		// ============================
		if wantCSST && len(studentIDs) > 0 {
			// Lagi-lagi, sesuaikan nama tabel join:
			// diasumsikan ada table: class_section_subject_teacher_students
			// dengan kolom:
			//   csst_student_school_id
			//   csst_student_school_student_id
			//   csst_student_csst_id
			//   csst_student_deleted_at
			type studentCSSTRow struct {
				SchoolStudentID uuid.UUID `gorm:"column:csst_student_school_student_id"`
				CSSTID          uuid.UUID `gorm:"column:csst_student_csst_id"`
			}

			var scRows []studentCSSTRow
			if err := h.DB.
				Table("class_section_subject_teacher_students").
				Select("csst_student_school_student_id, csst_student_csst_id").
				Where("csst_student_school_id = ?", schoolID).
				Where("csst_student_deleted_at IS NULL").
				Where("csst_student_school_student_id IN ?", studentIDs).
				Find(&scRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil relasi CSST siswa: "+err.Error())
			}

			// Kumpulkan unique CSST IDs
			csstIDSet := make(map[uuid.UUID]struct{})
			for _, r := range scRows {
				csstIDSet[r.CSSTID] = struct{}{}
			}
			csstIDs := make([]uuid.UUID, 0, len(csstIDSet))
			for id := range csstIDSet {
				csstIDs = append(csstIDs, id)
			}

			// Load CSST models → compact
			csstMapCompact := make(map[uuid.UUID]csstDTO.ClassSectionSubjectTeacherCompactResponse, len(csstIDs))
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
					csstMapCompact[csstRows[i].ClassSectionSubjectTeacherID] = cmp
				}
			}

			// Distribusi ke tiap siswa
			for _, r := range scRows {
				cmp, ok := csstMapCompact[r.CSSTID]
				if !ok {
					continue
				}
				if idx, ok := idxByStudent[r.SchoolStudentID]; ok {
					result[idx].ClassSectionSubjectTeaches = append(result[idx].ClassSectionSubjectTeaches, cmp)
				}
			}
		}

		return helper.JsonList(c, "ok", result, pg)
	}

	// =========================
	// MODE: ALL (default)
	// =========================

	// include=user_profile ?
	include := strings.ToLower(strings.TrimSpace(c.Query("include")))
	wantProfile := false
	if include != "" {
		for _, part := range strings.Split(include, ",") {
			part = strings.TrimSpace(part)
			switch part {
			case "user-profile", "user-profiles", "profile", "profiles", "user":
				wantProfile = true
			}
		}
	}

	// base response
	baseResp := make([]dto.SchoolStudentResp, 0, len(rows))
	for i := range rows {
		baseResp = append(baseResp, dto.FromModel(&rows[i]))
	}
	if !wantProfile {
		return helper.JsonList(c, "ok", baseResp, pg)
	}

	// --------------------------------------------------------------------
	// JOIN USER PROFILE (LITE)
	// --------------------------------------------------------------------

	type ProfileLite struct {
		ID                uuid.UUID `json:"id"`
		FullNameCache     *string   `json:"full_name_cache,omitempty"`
		AvatarURL         *string   `json:"avatar_url,omitempty"`
		WhatsappURL       *string   `json:"whatsapp_url,omitempty"`
		ParentName        *string   `json:"parent_name,omitempty"`
		ParentWhatsappURL *string   `json:"parent_whatsapp_url,omitempty"`
		Gender            *string   `json:"gender,omitempty"`
	}

	type SchoolStudentWithProfileResp struct {
		dto.SchoolStudentResp `json:",inline"`
		UserProfile           *ProfileLite `json:"user_profile,omitempty"`
	}

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

	// fetch user_profiles
	profileMap := make(map[uuid.UUID]ProfileLite, len(profileIDs))
	if len(profileIDs) > 0 {
		var profRows []struct {
			ID                uuid.UUID `gorm:"column:user_profile_id"`
			FullNameCache     *string   `gorm:"column:user_profile_full_name_cache"`
			AvatarURL         *string   `gorm:"column:user_profile_avatar_url"`
			WhatsappURL       *string   `gorm:"column:user_profile_whatsapp_url"`
			ParentName        *string   `gorm:"column:user_profile_parent_name"`
			ParentWhatsappURL *string   `gorm:"column:user_profile_parent_whatsapp_url"`
			Gender            *string   `gorm:"column:user_profile_gender"`
		}

		if err := h.DB.
			Table("user_profiles").
			Select(`
				user_profile_id,
				user_profile_full_name_cache,
				user_profile_avatar_url,
				user_profile_whatsapp_url,
				user_profile_parent_name,
				user_profile_parent_whatsapp_url,
				user_profile_gender
			`).
			Where("user_profile_id IN ?", profileIDs).
			Where("user_profile_deleted_at IS NULL").
			Find(&profRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		for _, pr := range profRows {
			profileMap[pr.ID] = ProfileLite{
				ID:                pr.ID,
				FullNameCache:     pr.FullNameCache,
				AvatarURL:         pr.AvatarURL,
				WhatsappURL:       pr.WhatsappURL,
				ParentName:        pr.ParentName,
				ParentWhatsappURL: pr.ParentWhatsappURL,
				Gender:            pr.Gender,
			}
		}
	}

	// merge final
	out := make([]SchoolStudentWithProfileResp, 0, len(rows))
	for i := range rows {
		base := baseResp[i]
		var up *ProfileLite
		if val, ok := profileMap[rows[i].SchoolStudentUserProfileID]; ok {
			tmp := val
			up = &tmp
		}
		out = append(out, SchoolStudentWithProfileResp{
			SchoolStudentResp: base,
			UserProfile:       up,
		})
	}

	return helper.JsonList(c, "ok", out, pg)
}
