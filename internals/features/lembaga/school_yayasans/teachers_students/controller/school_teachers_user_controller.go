// file: internals/features/lembaga/school_yayasans/teachers_students/controller/school_teacher_list_controller.go
package controller

import (
	"strconv"
	"strings"

	yDTO "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/dto"
	yModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

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
		"created_at": "school_teacher_created_at",
		"updated_at": "school_teacher_updated_at",
	}
	col, ok := colMap[sortBy]
	if !ok {
		col = colMap["created_at"]
	}
	orderExpr := col + " " + strings.ToUpper(order)

	// 3) Filters
	idStr := strings.TrimSpace(c.Query("id"))

	// ⬇️ di sini kita tambahkan semua alias untuk FK user_teacher:
	userTeacherIDStr := strings.TrimSpace(
		c.Query("user_teacher_id",
			c.Query("user_id",
				c.Query("teacher_id",
					c.Query("school_teacher_user_teacher_id")))),
	)

	userProfileIDStr := strings.TrimSpace(c.Query("user_profile_id")) // tetap

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
		Model(&yModel.SchoolTeacherModel{}).
		Where("school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL", schoolID)

	// ⬇️ filter by PK teacher
	if rowID != uuid.Nil {
		tx = tx.Where("school_teacher_id = ?", rowID)
	}
	// ⬇️ filter by FK user_teacher (school_teacher_user_teacher_id)
	if userTeacherID != uuid.Nil {
		tx = tx.Where("school_teacher_user_teacher_id = ?", userTeacherID)
	}
	// ⬇️ filter by user_profile_id (via join ke user_teachers & user_profiles)
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
		tx = tx.Where(`(school_teacher_notes ILIKE ? 
			OR school_teacher_code ILIKE ? 
			OR school_teacher_slug ILIKE ?
			OR school_teacher_user_teacher_name_snapshot ILIKE ?)`, pat, pat, pat, pat)
	}

	// 4) Count + data
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []yModel.SchoolTeacherModel
	if err := tx.
		Order(orderExpr).
		Limit(p.Limit).
		Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 5) include flags
	inc := strings.ToLower(strings.TrimSpace(c.Query("include")))
	wantTeacher := false
	wantUser := false
	wantProfile := false // ⬅️ NEW

	if inc != "" {
		for _, part := range strings.Split(inc, ",") {
			switch strings.TrimSpace(part) {
			case "teacher", "teachers", "user-teachers":
				wantTeacher = true
			case "user", "users":
				wantUser = true
			case "user-profile", "profile", "profiles":
				wantProfile = true
			case "all":
				wantTeacher = true
				wantUser = true
				wantProfile = true
			}
		}
	}

	// 6) base responses + kumpulkan IDs
	base := make([]*yDTO.SchoolTeacher, 0, len(rows))
	teacherIDsSet := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		base = append(base, yDTO.NewSchoolTeacherResponse(&rows[i]))
		if rows[i].SchoolTeacherUserTeacherID != uuid.Nil {
			teacherIDsSet[rows[i].SchoolTeacherUserTeacherID] = struct{}{}
		}
	}

	// ==== include: teacher (user_teachers) ====
	type TeacherLite struct {
		ID         uuid.UUID `json:"id"`
		UserID     uuid.UUID `json:"user_id"`
		Name       string    `json:"name"`
		Whatsapp   *string   `json:"whatsapp_url,omitempty"`
		AvatarURL  *string   `json:"avatar_url,omitempty"`
		TitlePref  *string   `json:"title_prefix,omitempty"`
		TitleSuf   *string   `json:"title_suffix,omitempty"`
		IsActive   bool      `json:"is_active"`
		IsVerified bool      `json:"is_verified"`
	}
	teacherIDs := make([]uuid.UUID, 0, len(teacherIDsSet))
	for id := range teacherIDsSet {
		teacherIDs = append(teacherIDs, id)
	}

	teacherMap := make(map[uuid.UUID]TeacherLite, len(teacherIDs))
	userIDsSet := make(map[uuid.UUID]struct{}, len(teacherIDs))

	if wantTeacher || wantUser || wantProfile {
		if len(teacherIDs) > 0 {
			var trows []struct {
				UserTeacherID        uuid.UUID `gorm:"column:user_teacher_id"`
				UserTeacherUserID    uuid.UUID `gorm:"column:user_teacher_user_id"`
				UserTeacherName      string    `gorm:"column:user_teacher_name"`
				UserTeacherWhatsapp  *string   `gorm:"column:user_teacher_whatsapp_url"`
				UserTeacherAvatarURL *string   `gorm:"column:user_teacher_avatar_url"`
				TitlePrefix          *string   `gorm:"column:user_teacher_title_prefix"`
				TitleSuffix          *string   `gorm:"column:user_teacher_title_suffix"`
				IsActive             bool      `gorm:"column:user_teacher_is_active"`
				IsVerified           bool      `gorm:"column:user_teacher_is_verified"`
			}
			if err := ctrl.DB.Table("user_teachers").
				Select(`user_teacher_id, user_teacher_user_id, user_teacher_name, user_teacher_whatsapp_url,
						user_teacher_avatar_url, user_teacher_title_prefix, user_teacher_title_suffix,
						user_teacher_is_active, user_teacher_is_verified`).
				Where("user_teacher_id IN ?", teacherIDs).
				Where("user_teacher_deleted_at IS NULL").
				Find(&trows).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
			}
			for _, t := range trows {
				teacherMap[t.UserTeacherID] = TeacherLite{
					ID:         t.UserTeacherID,
					UserID:     t.UserTeacherUserID,
					Name:       t.UserTeacherName,
					Whatsapp:   t.UserTeacherWhatsapp,
					AvatarURL:  t.UserTeacherAvatarURL,
					TitlePref:  t.TitlePrefix,
					TitleSuf:   t.TitleSuffix,
					IsActive:   t.IsActive,
					IsVerified: t.IsVerified,
				}
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
		if err := ctrl.DB.Table("users").
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

	// ==== include: user_profile (user_profiles) ====
	type UserProfileLite struct {
		ID                uuid.UUID `json:"id"`
		UserID            uuid.UUID `json:"user_id"`
		FullNameSnapshot  *string   `json:"full_name_snapshot,omitempty"`
		AvatarURL         *string   `json:"avatar_url,omitempty"`
		WhatsappURL       *string   `json:"whatsapp_url,omitempty"`
		ParentName        *string   `json:"parent_name,omitempty"`
		ParentWhatsappURL *string   `json:"parent_whatsapp_url,omitempty"`
	}

	profileMap := make(map[uuid.UUID]UserProfileLite, len(userIDs)) // key: user_id
	if wantProfile && len(userIDs) > 0 {
		var prows []struct {
			ID                uuid.UUID `gorm:"column:user_profile_id"`
			UserID            uuid.UUID `gorm:"column:user_profile_user_id"`
			FullNameSnapshot  *string   `gorm:"column:user_profile_full_name_snapshot"`
			AvatarURL         *string   `gorm:"column:user_profile_avatar_url"`
			WhatsappURL       *string   `gorm:"column:user_profile_whatsapp_url"`
			ParentName        *string   `gorm:"column:user_profile_parent_name"`
			ParentWhatsappURL *string   `gorm:"column:user_profile_parent_whatsapp_url"`
		}
		if err := ctrl.DB.Table("user_profiles").
			Select(`
				user_profile_id,
				user_profile_user_id,
				user_profile_full_name_snapshot,
				user_profile_avatar_url,
				user_profile_whatsapp_url,
				user_profile_parent_name,
				user_profile_parent_whatsapp_url
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
				FullNameSnapshot:  pr.FullNameSnapshot,
				AvatarURL:         pr.AvatarURL,
				WhatsappURL:       pr.WhatsappURL,
				ParentName:        pr.ParentName,
				ParentWhatsappURL: pr.ParentWhatsappURL,
			}
		}
	}

	// 7) Susun output
	type Item struct {
		*yDTO.SchoolTeacher `json:",inline"`
		Teacher             *TeacherLite     `json:"user_teacher,omitempty"`
		User                *UserLite        `json:"user,omitempty"`
		UserProfile         *UserProfileLite `json:"user_profile,omitempty"`
	}

	out := make([]Item, 0, len(base))
	for i := range rows {
		var t *TeacherLite
		if wantTeacher {
			if v, ok := teacherMap[rows[i].SchoolTeacherUserTeacherID]; ok {
				tmp := v
				t = &tmp
			}
		}
		var u *UserLite
		if wantUser && t != nil {
			if v, ok := userMap[t.UserID]; ok {
				tmp := v
				u = &tmp
			}
		}
		var up *UserProfileLite
		if wantProfile && t != nil {
			if v, ok := profileMap[t.UserID]; ok {
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
