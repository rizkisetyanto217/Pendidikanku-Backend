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

	csstDTO "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	classSectionDTO "madinahsalam_backend/internals/features/school/classes/class_sections/dto"
	classSectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"

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

	// =========================
	// Siapkan IDs
	// =========================

	// PK school_teacher (untuk class_sections & csst)
	teacherIDs := make([]uuid.UUID, 0, len(rows))
	// FK user_teacher (untuk user_teachers sidecar & turunan)
	userTeacherIDsSet := make(map[uuid.UUID]struct{}, len(rows))

	for i := range rows {
		teacherIDs = append(teacherIDs, rows[i].SchoolTeacherID)
		if rows[i].SchoolTeacherUserTeacherID != uuid.Nil {
			userTeacherIDsSet[rows[i].SchoolTeacherUserTeacherID] = struct{}{}
		}
	}

	// =========================
	// MODE COMPACT vs FULL → bentuk DATA (school_teachers saja)
	// =========================
	var data interface{}

	if isCompact {
		// Compact: DTO compact guru
		compacts := teacherDTO.NewSchoolTeacherCompacts(rows)
		data = compacts
	} else {
		// Full: DTO full guru (tanpa nesting user_teacher/user/user_profile)
		base := make([]*teacherDTO.SchoolTeacher, 0, len(rows))
		for i := range rows {
			base = append(base, teacherDTO.NewSchoolTeacherResponse(&rows[i]))
		}
		data = base
	}

	// =========================================
	// PARSE include=... SEKALI SAJA
	// =========================================

	incRaw := strings.ToLower(strings.TrimSpace(c.Query("include")))

	// sidecar flags
	wantUserTeachers := false
	wantUsers := false
	wantUserProfiles := false
	wantWithSections := false
	wantWithCSST := false

	if incRaw != "" {
		for _, part := range strings.Split(incRaw, ",") {
			p := strings.TrimSpace(part)
			switch p {
			// user_teachers (sidecar)
			case "teacher", "teachers", "user_teacher", "user_teachers":
				wantUserTeachers = true
			// users (sidecar)
			case "user", "users":
				wantUsers = true
			// user_profiles (sidecar)
			case "user-profile", "profile", "profiles", "user_profiles":
				wantUserProfiles = true
			// class_sections (sidecar)
			case "sections", "class-sections", "class_sections", "classsection":
				wantWithSections = true
			// csst (sidecar)
			case "csst", "class-section-subject-teachers", "class_section_subject_teachers", "subject-teachers", "subject_teachers":
				wantWithCSST = true
			case "all":
				wantUserTeachers = true
				wantUsers = true
				wantUserProfiles = true
				wantWithSections = true
				wantWithCSST = true
			}
		}
	}

	// =========================================
	// SIAPKAN INCLUDE PAYLOAD
	// =========================================

	type UserLite struct {
		ID       uuid.UUID `json:"id"`
		UserName string    `json:"user_name"`
		FullName *string   `json:"full_name,omitempty"`
		Email    string    `json:"email"`
		IsActive bool      `json:"is_active"`
	}

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

	type IncludePayload struct {
		UserTeachers                []userTeacherModel.UserTeacherModel                 `json:"user_teachers,omitempty"`
		Users                       []UserLite                                          `json:"users,omitempty"`
		UserProfiles                []UserProfileLite                                   `json:"user_profiles,omitempty"`
		ClassSections               []classSectionDTO.ClassSectionCompactResponse       `json:"class_sections,omitempty"`
		ClassSectionSubjectTeachers []csstDTO.ClassSectionSubjectTeacherCompactResponse `json:"class_section_subject_teachers,omitempty"`
	}

	includePayload := IncludePayload{}

	// ==================================================
	// SIDE CAR: user_teachers (+ cascade users & profiles)
	// ==================================================
	userTeacherIDs := make([]uuid.UUID, 0, len(userTeacherIDsSet))
	for id := range userTeacherIDsSet {
		userTeacherIDs = append(userTeacherIDs, id)
	}

	userIDsSet := make(map[uuid.UUID]struct{}, len(userTeacherIDs))

	if (wantUserTeachers || wantUsers || wantUserProfiles) && len(userTeacherIDs) > 0 {
		// user_teachers
		if wantUserTeachers || wantUsers || wantUserProfiles {
			var trows []userTeacherModel.UserTeacherModel
			if err := ctrl.DB.WithContext(c.Context()).
				Table("user_teachers").
				Where("user_teacher_id IN ?", userTeacherIDs).
				Where("user_teacher_deleted_at IS NULL").
				Find(&trows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
			}

			if wantUserTeachers {
				includePayload.UserTeachers = trows
			}

			// kumpulkan user_id untuk users & user_profiles
			if wantUsers || wantUserProfiles {
				for _, t := range trows {
					if t.UserTeacherUserID != uuid.Nil {
						userIDsSet[t.UserTeacherUserID] = struct{}{}
					}
				}
			}
		}
	}

	// SIDE CAR: users
	userIDs := make([]uuid.UUID, 0, len(userIDsSet))
	for id := range userIDsSet {
		userIDs = append(userIDs, id)
	}

	if wantUsers && len(userIDs) > 0 {
		var urows []UserLite
		if err := ctrl.DB.WithContext(c.Context()).
			Table("users").
			Select("id, user_name, full_name, email, is_active").
			Where("id IN ?", userIDs).
			Where("deleted_at IS NULL").
			Find(&urows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		includePayload.Users = urows
	}

	// SIDE CAR: user_profiles
	if wantUserProfiles && len(userIDs) > 0 {
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
		outProfiles := make([]UserProfileLite, 0, len(prows))
		for _, pr := range prows {
			outProfiles = append(outProfiles, UserProfileLite{
				ID:                pr.ID,
				UserID:            pr.UserID,
				FullNameCache:     pr.FullNameCache,
				AvatarURL:         pr.AvatarURL,
				WhatsappURL:       pr.WhatsappURL,
				ParentName:        pr.ParentName,
				ParentWhatsappURL: pr.ParentWhatsappURL,
				GenderSnapshot:    pr.GenderSnapshot,
			})
		}
		includePayload.UserProfiles = outProfiles
	}

	// =========================================
	// SIDE CAR: CLASS SECTIONS (compact)
	// =========================================
	if wantWithSections && len(teacherIDs) > 0 {
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

		compacts := make([]classSectionDTO.ClassSectionCompactResponse, 0, len(secRows))
		for i := range secRows {
			cs := &secRows[i]
			compacts = append(compacts, classSectionDTO.FromModelClassSectionToCompact(cs))
		}
		includePayload.ClassSections = compacts
	}

	// =========================================
	// SIDE CAR: CSST (compact)
	// =========================================
	// SIDE CAR: CSST (compact)
	if wantWithCSST && len(teacherIDs) > 0 {
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

		includePayload.ClassSectionSubjectTeachers =
			csstDTO.FromClassSectionSubjectTeacherModelsCompact(csstRows)
	}

	// 8) Pagination + response
	pg := helper.BuildPaginationFromPage(total, p.Page, p.PerPage)

	// Kalau ada minimal satu include flag yang diminta, kirim via JsonListWithInclude
	if wantUserTeachers || wantUsers || wantUserProfiles || wantWithSections || wantWithCSST {
		return helper.JsonListWithInclude(c, "ok", data, includePayload, pg)
	}

	// Tidak ada include yang diminta → plain list
	return helper.JsonList(c, "ok", data, pg)

}
