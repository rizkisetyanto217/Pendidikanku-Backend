package controller

import (
	"strconv"
	"strings"

	yDTO "schoolku_backend/internals/features/lembaga/school_yayasans/teachers_students/dto"
	yModel "schoolku_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* ===================== LIST ===================== */
// GET /api/u/schools/:school_id/school-teachers/list
// GET /api/u/m/:school_slug/school-teachers/list
// Query: page, per_page|limit, sort_by(created_at|updated_at), order
//        id, user_teacher_id|user_id, employment, active, verified, public,
//        joined_from, joined_to (YYYY-MM-DD), q, include=teacher|user|all
func (ctrl *SchoolTeacherController) List(c *fiber.Ctx) error {
	// DB resolver (untuk slug → id)
	if c.Locals("DB") == nil {
		c.Locals("DB", ctrl.DB)
	}

	// 1) Context & akses
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}
	schoolID, err := helperAuth.EnsureSchoolAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// 2) Pagination & sorting
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)
	allowedSort := map[string]string{
		"created_at": "school_teacher_created_at",
		"updated_at": "school_teacher_updated_at",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid sort_by")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// 3) Filters
	idStr := strings.TrimSpace(c.Query("id"))
	userTeacherIDStr := strings.TrimSpace(c.Query("user_teacher_id", c.Query("user_id")))
	employment := strings.ToLower(strings.TrimSpace(c.Query("employment")))
	activeStr := strings.TrimSpace(c.Query("active"))
	verifiedStr := strings.TrimSpace(c.Query("verified"))
	publicStr := strings.TrimSpace(c.Query("public"))
	joinedFromStr := strings.TrimSpace(c.Query("joined_from"))
	joinedToStr := strings.TrimSpace(c.Query("joined_to"))
	q := strings.TrimSpace(c.Query("q"))

	var rowID, userTeacherID uuid.UUID
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

	tx := ctrl.DB.WithContext(c.Context()).
		Model(&yModel.SchoolTeacherModel{}).
		Where("school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL", schoolID)

	if rowID != uuid.Nil {
		tx = tx.Where("school_teacher_id = ?", rowID)
	}
	if userTeacherID != uuid.Nil {
		tx = tx.Where("school_teacher_user_teacher_id = ?", userTeacherID)
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
	if err := tx.Order(orderExpr).Limit(p.Limit()).Offset(p.Offset()).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 5) include flags
	inc := strings.ToLower(strings.TrimSpace(c.Query("include")))
	wantTeacher := false
	wantUser := false
	if inc != "" {
		for _, part := range strings.Split(inc, ",") {
			switch strings.TrimSpace(part) {
			case "teacher", "teachers":
				wantTeacher = true
			case "user", "users":
				wantUser = true
			case "all":
				wantTeacher = true
				wantUser = true
			}
		}
	}

	// 6) base responses + kumpulkan IDs
	base := make([]*yDTO.SchoolTeacher, 0, len(rows))
	teacherIDsSet := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		base = append(base, yDTO.NewSchoolTeacherResponse(&rows[i])) // returns *dto.SchoolTeacher
		if rows[i].SchoolTeacherUserTeacherID != uuid.Nil {
			teacherIDsSet[rows[i].SchoolTeacherUserTeacherID] = struct{}{}
		}
	}

	// Jika tidak minta include apa pun, return cepat
	if !wantTeacher && !wantUser {
		return helper.JsonList(c, base, helper.BuildMeta(total, p))
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

	if wantTeacher || wantUser {
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
				if wantUser && t.UserTeacherUserID != uuid.Nil {
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

	// 7) Susun output
	type Item struct {
		*yDTO.SchoolTeacher `json:",inline"`
		Teacher             *TeacherLite `json:"teacher,omitempty"`
		User                *UserLite    `json:"user,omitempty"`
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

		out = append(out, Item{
			SchoolTeacher: base[i], // sudah berisi snapshots + sections/csst dari DTO
			Teacher:       t,
			User:          u,
		})
	}

	return helper.JsonList(c, out, helper.BuildMeta(total, p))
}
