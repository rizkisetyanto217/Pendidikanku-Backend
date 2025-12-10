// file: internals/features/lembaga/school_yayasans/teachers_students/controller/school_teacher_list_controller.go
package controller

import (
	"strconv"
	"strings"

	teacherDTO "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/dto"
	teacherModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	userTeacherModel "madinahsalam_backend/internals/features/users/user_teachers/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	classSectionDTO "madinahsalam_backend/internals/features/school/classes/class_sections/dto"

	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	csstDTO "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (ctrl *SchoolTeacherController) List(c *fiber.Ctx) error {
	// (opsional) kalau ada helper lain yang butuh DB di Locals
	if c.Locals("DB") == nil {
		c.Locals("DB", ctrl.DB)
	}

	// 1) Ambil school_id dari TOKEN (bukan dari path/slug)
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Hanya DKM/Admin yang boleh
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
		return er
	}

	// 2) Paging
	p := helper.ResolvePaging(c, 20, 100)

	// 2b) Sorting whitelist manual
	sortBy := strings.ToLower(strings.TrimSpace(c.Query("sort_by", "created_at")))
	order := strings.ToLower(strings.TrimSpace(c.Query("order", "desc")))
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	colMap := map[string]string{
		"created_at":            "school_teacher_created_at",
		"updated_at":            "school_teacher_updated_at",
		"total_sections":        "school_teacher_total_class_sections",
		"total_sections_active": "school_teacher_total_class_sections_active",
		"total_csst":            "school_teacher_total_class_section_subject_teachers",
		"total_csst_active":     "school_teacher_total_class_section_subject_teachers_active",
	}
	col, ok := colMap[sortBy]
	if !ok {
		col = colMap["created_at"]
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// 🔁 Mode: compact / full (default)
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	isCompact := mode == "compact" || mode == "lite"

	// 3) Filters
	idStr := strings.TrimSpace(c.Query("id"))

	userTeacherIDStr := strings.TrimSpace(
		c.Query("user_teacher_id",
			c.Query("user_id",
				c.Query("teacher_id",
					c.Query("school_teacher_user_teacher_id")))),
	)

	userProfileIDStr := strings.TrimSpace(c.Query("user_profile_id"))

	employment := strings.ToLower(strings.TrimSpace(c.Query("employment")))
	activeStr := strings.TrimSpace(c.Query("active"))
	verifiedStr := strings.TrimSpace(c.Query("verified"))
	publicStr := strings.TrimSpace(c.Query("public"))
	joinedFromStr := strings.TrimSpace(c.Query("joined_from"))
	joinedToStr := strings.TrimSpace(c.Query("joined_to"))
	q := strings.TrimSpace(c.Query("q"))

	var (
		rowID         uuid.UUID
		userTeacherID uuid.UUID
		userProfileID uuid.UUID
	)

	if idStr != "" {
		v, er := uuid.Parse(idStr)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id invalid")
		}
		rowID = v
	}
	if userTeacherIDStr != "" {
		v, er := uuid.Parse(userTeacherIDStr)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "user_teacher_id invalid")
		}
		userTeacherID = v
	}
	if userProfileIDStr != "" {
		v, er := uuid.Parse(userProfileIDStr)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "user_profile_id invalid")
		}
		userProfileID = v
	}

	tx := ctrl.DB.WithContext(c.Context()).
		Model(&teacherModel.SchoolTeacherModel{}).
		Where("school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL", schoolID)

	// filter by PK teacher
	if rowID != uuid.Nil {
		tx = tx.Where("school_teacher_id = ?", rowID)
	}
	// filter by FK user_teacher
	if userTeacherID != uuid.Nil {
		tx = tx.Where("school_teacher_user_teacher_id = ?", userTeacherID)
	}
	// filter by user_profile_id (via join ke user_profiles lewat user_teachers)
	if userProfileID != uuid.Nil {
		tx = tx.Joins(`
			JOIN user_teachers ut
			  ON ut.user_teacher_id = school_teacher_user_teacher_id
			 AND ut.user_teacher_deleted_at IS NULL
			JOIN user_profiles up
			  ON up.user_profile_user_id = ut.user_teacher_user_id
			 AND up.user_profile_deleted_at IS NULL
		`).Where("up.user_profile_id = ?", userProfileID)
	}

	if employment != "" {
		tx = tx.Where("school_teacher_employment = ?", employment)
	}
	if activeStr != "" {
		if b, er := strconv.ParseBool(activeStr); er == nil {
			tx = tx.Where("school_teacher_is_active = ?", b)
		}
	}
	if verifiedStr != "" {
		if b, er := strconv.ParseBool(verifiedStr); er == nil {
			tx = tx.Where("school_teacher_is_verified = ?", b)
		}
	}
	if publicStr != "" {
		if b, er := strconv.ParseBool(publicStr); er == nil {
			tx = tx.Where("school_teacher_is_public = ?", b)
		}
	}
	if joinedFromStr != "" {
		if t, er := parseDateYYYYMMDD(joinedFromStr); er == nil {
			tx = tx.Where("school_teacher_joined_at IS NOT NULL AND school_teacher_joined_at >= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "joined_from invalid (YYYY-MM-DD)")
		}
	}
	if joinedToStr != "" {
		if t, er := parseDateYYYYMMDD(joinedToStr); er == nil {
			tx = tx.Where("school_teacher_joined_at IS NOT NULL AND school_teacher_joined_at <= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "joined_to invalid (YYYY-MM-DD)")
		}
	}
	if q != "" {
		pat := "%" + q + "%"
		tx = tx.Where(`(
			school_teacher_notes ILIKE ? 
			OR school_teacher_code ILIKE ? 
			OR school_teacher_slug ILIKE ?
			OR school_teacher_user_teacher_full_name_cache ILIKE ?
		)`, pat, pat, pat, pat)
	}

	// 4) Count + data
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []teacherModel.SchoolTeacherModel
	if err := tx.
		Order(orderExpr).
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 🔁 Mode COMPACT (+ optional nested)
	if isCompact {
		// 🔥 NEW: nested=sections,csst
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

		// Behaviour lama: kalau tidak minta nested apa pun → langsung pulang
		if !wantSections && !wantCSST {
			compacts := teacherDTO.NewSchoolTeacherCompacts(rows)
			pg := helper.BuildPaginationFromPage(total, p.Page, p.PerPage)
			return helper.JsonList(c, "ok", compacts, pg)
		}

		// 🔥 NEW: compact + nested
		// NOTE: NewSchoolTeacherCompacts mengembalikan []*SchoolTeacherCompact
		type SchoolTeacherCompactWithNested struct {
			*teacherDTO.SchoolTeacherCompact `json:",inline"`                                    // embed pointer
			ClassSections                    []classSectionDTO.ClassSectionCompactResponse       `json:"class_sections,omitempty"`
			ClassSectionSubjectTeachers      []csstDTO.ClassSectionSubjectTeacherCompactResponse `json:"class_section_subject_teachers,omitempty"`
		}

		// base compact
		compacts := teacherDTO.NewSchoolTeacherCompacts(rows) // []*SchoolTeacherCompact

		// siapkan result
		result := make([]SchoolTeacherCompactWithNested, len(compacts))
		for i := range compacts {
			result[i].SchoolTeacherCompact = compacts[i] // ✅ sekarang tipe-nya cocok (*SchoolTeacherCompact)
		}

		// kumpulkan teacher_id
		teacherIDs := make([]uuid.UUID, 0, len(rows))
		idxByTeacher := make(map[uuid.UUID]int, len(rows))
		for i := range rows {
			tid := rows[i].SchoolTeacherID
			teacherIDs = append(teacherIDs, tid)
			idxByTeacher[tid] = i
		}

		// ============================
		// NESTED: CLASS SECTIONS (compact)
		// ============================
		if wantSections && len(teacherIDs) > 0 {
			var secRows []classSectionModel.ClassSectionModel
			if err := ctrl.DB.WithContext(c.Context()).
				Where("class_section_school_id = ? AND class_section_deleted_at IS NULL", schoolID).
				Where(`
					class_section_school_teacher_id IN ?
					OR class_section_assistant_school_teacher_id IN ?
				`, teacherIDs, teacherIDs).
				Find(&secRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil class_sections: "+err.Error())
			}

			for i := range secRows {
				cs := &secRows[i]
				cmp := classSectionDTO.FromModelClassSectionToCompact(cs)

				// homeroom
				if cs.ClassSectionSchoolTeacherID != nil {
					if idx, ok := idxByTeacher[*cs.ClassSectionSchoolTeacherID]; ok {
						result[idx].ClassSections = append(result[idx].ClassSections, cmp)
					}
				}
				// assistant
				if cs.ClassSectionAssistantSchoolTeacherID != nil {
					if idx, ok := idxByTeacher[*cs.ClassSectionAssistantSchoolTeacherID]; ok {
						result[idx].ClassSections = append(result[idx].ClassSections, cmp)
					}
				}
			}
		}

		// ============================
		// NESTED: CSST (compact)
		// ============================
		if wantCSST && len(teacherIDs) > 0 {
			var csstRows []csstModel.ClassSectionSubjectTeacherModel
			if err := ctrl.DB.WithContext(c.Context()).
				Where("class_section_subject_teacher_school_id = ? AND class_section_subject_teacher_deleted_at IS NULL", schoolID).
				Where(`
					class_section_subject_teacher_school_teacher_id IN ?
					OR class_section_subject_teacher_assistant_school_teacher_id IN ?
				`, teacherIDs, teacherIDs).
				Find(&csstRows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "gagal ambil CSST: "+err.Error())
			}

			for i := range csstRows {
				row := &csstRows[i]
				cmp := csstDTO.FromClassSectionSubjectTeacherModelCompact(*row)

				// main teacher
				if row.ClassSectionSubjectTeacherSchoolTeacherID != nil {
					if idx, ok := idxByTeacher[*row.ClassSectionSubjectTeacherSchoolTeacherID]; ok {
						result[idx].ClassSectionSubjectTeachers = append(result[idx].ClassSectionSubjectTeachers, cmp)
					}
				}
				// assistant teacher
				if row.ClassSectionSubjectTeacherAssistantSchoolTeacherID != nil {
					if idx, ok := idxByTeacher[*row.ClassSectionSubjectTeacherAssistantSchoolTeacherID]; ok {
						result[idx].ClassSectionSubjectTeachers = append(result[idx].ClassSectionSubjectTeachers, cmp)
					}
				}
			}
		}

		pg := helper.BuildPaginationFromPage(total, p.Page, p.PerPage)
		return helper.JsonList(c, "ok", result, pg)
	}

	// ========================
	// Mode FULL (default)
	// ========================

	// 5) include flags
	inc := strings.ToLower(strings.TrimSpace(c.Query("include")))
	wantTeacher := false
	wantUser := false
	wantProfile := false

	if inc != "" {
		for _, part := range strings.Split(inc, ",") {
			switch strings.TrimSpace(part) {
			case "teacher", "teachers", "user-teachers", "user_teacher", "user_teachers":
				wantTeacher = true
			case "user", "users":
				wantUser = true
			case "user-profile", "profile", "profiles", "user_profiles":
				wantProfile = true
			case "all":
				wantTeacher = true
				wantUser = true
				wantProfile = true
			}
		}
	}

	// 6) base responses + kumpulkan IDs user_teacher
	base := make([]*teacherDTO.SchoolTeacher, 0, len(rows))
	teacherIDsSet := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		base = append(base, teacherDTO.NewSchoolTeacherResponse(&rows[i]))
		if rows[i].SchoolTeacherUserTeacherID != uuid.Nil {
			teacherIDsSet[rows[i].SchoolTeacherUserTeacherID] = struct{}{}
		}
	}

	// ==== include: teacher (user_teachers) – full model ====
	teacherIDs := make([]uuid.UUID, 0, len(teacherIDsSet))
	for id := range teacherIDsSet {
		teacherIDs = append(teacherIDs, id)
	}

	teacherMap := make(map[uuid.UUID]userTeacherModel.UserTeacherModel, len(teacherIDs))
	userIDsSet := make(map[uuid.UUID]struct{}, len(teacherIDs))

	if wantTeacher || wantUser || wantProfile {
		if len(teacherIDs) > 0 {
			var trows []userTeacherModel.UserTeacherModel
			if err := ctrl.DB.WithContext(c.Context()).
				Table("user_teachers").
				Where("user_teacher_id IN ?", teacherIDs).
				Where("user_teacher_deleted_at IS NULL").
				Find(&trows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
			}
			for _, t := range trows {
				teacherMap[t.UserTeacherID] = t
				if (wantUser || wantProfile) && t.UserTeacherUserID != uuid.Nil {
					userIDsSet[t.UserTeacherUserID] = struct{}{}
				}
			}
		}
	}

	// ==== include: user (users) ====
	type UserLite struct {
		ID       uuid.UUID `json:"id"`
		UserName string    `json:"user_name"`
		FullName *string   `json:"full_name,omitempty"`
		Email    string    `json:"email"`
		IsActive bool      `json:"is_active"`
	}

	userIDs := make([]uuid.UUID, 0, len(userIDsSet))
	for id := range userIDsSet {
		userIDs = append(userIDs, id)
	}
	userMap := make(map[uuid.UUID]UserLite, len(userIDs))
	if wantUser && len(userIDs) > 0 {
		var urows []UserLite
		if err := ctrl.DB.WithContext(c.Context()).
			Table("users").
			Select("id, user_name, full_name, email, is_active").
			Where("id IN ?", userIDs).
			Where("deleted_at IS NULL").
			Find(&urows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		for _, u := range urows {
			userMap[u.ID] = u
		}
	}

	// ==== include: user_profile (user_profiles) – optional ====
	type UserProfileLite struct {
		ID                uuid.UUID `json:"id"`
		UserID            uuid.UUID `json:"user_id"`
		FullNameCache     *string   `json:"full_name_cache,omitempty"`
		AvatarURL         *string   `json:"avatar_url,omitempty"`
		WhatsappURL       *string   `json:"whatsapp_url,omitempty"`
		ParentName        *string   `json:"parent_name,omitempty"`
		ParentWhatsappURL *string   `json:"parent_whatsapp_url,omitempty"`
		GenderSnapshot    *string   `json:"gender_snapshot,omitempty"`
	}

	profileMap := make(map[uuid.UUID]UserProfileLite, len(userIDs))
	if wantProfile && len(userIDs) > 0 {
		var prows []struct {
			ID                uuid.UUID `gorm:"column:user_profile_id"`
			UserID            uuid.UUID `gorm:"column:user_profile_user_id"`
			FullNameCache     *string   `gorm:"column:user_profile_full_name_cache"`
			AvatarURL         *string   `gorm:"column:user_profile_avatar_url"`
			WhatsappURL       *string   `gorm:"column:user_profile_whatsapp_url"`
			ParentName        *string   `gorm:"column:user_profile_parent_name"`
			ParentWhatsappURL *string   `gorm:"column:user_profile_parent_whatsapp_url"`
			GenderSnapshot    *string   `gorm:"column:user_profile_gender_snapshot"`
		}
		if err := ctrl.DB.WithContext(c.Context()).
			Table("user_profiles").
			Select(`
				user_profile_id,
				user_profile_user_id,
				user_profile_full_name_cache,
				user_profile_avatar_url,
				user_profile_whatsapp_url,
				user_profile_parent_name,
				user_profile_parent_whatsapp_url,
				user_profile_gender_snapshot
			`).
			Where("user_profile_user_id IN ?", userIDs).
			Where("user_profile_deleted_at IS NULL").
			Find(&prows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		for _, pr := range prows {
			profileMap[pr.UserID] = UserProfileLite{
				ID:                pr.ID,
				UserID:            pr.UserID,
				FullNameCache:     pr.FullNameCache,
				AvatarURL:         pr.AvatarURL,
				WhatsappURL:       pr.WhatsappURL,
				ParentName:        pr.ParentName,
				ParentWhatsappURL: pr.ParentWhatsappURL,
				GenderSnapshot:    pr.GenderSnapshot,
			}
		}
	}

	// 7) Susun output
	type Item struct {
		*teacherDTO.SchoolTeacher `json:",inline"`
		Teacher                   *userTeacherModel.UserTeacherModel `json:"user_teacher,omitempty"`
		User                      *UserLite                          `json:"user,omitempty"`
		UserProfile               *UserProfileLite                   `json:"user_profile,omitempty"`
	}

	out := make([]Item, 0, len(base))
	for i := range rows {
		var t *userTeacherModel.UserTeacherModel
		if wantTeacher {
			if v, ok := teacherMap[rows[i].SchoolTeacherUserTeacherID]; ok {
				tmp := v
				t = &tmp
			}
		}

		var u *UserLite
		if wantUser && t != nil {
			if v, ok := userMap[t.UserTeacherUserID]; ok {
				tmp := v
				u = &tmp
			}
		}

		var up *UserProfileLite
		if wantProfile && t != nil {
			if v, ok := profileMap[t.UserTeacherUserID]; ok {
				tmp := v
				up = &tmp
			}
		}

		out = append(out, Item{
			SchoolTeacher: base[i],
			Teacher:       t,
			User:          u,
			UserProfile:   up,
		})
	}

	// 8) Pagination meta
	pg := helper.BuildPaginationFromPage(total, p.Page, p.PerPage)
	return helper.JsonList(c, "ok", out, pg)
}
