// file: internals/features/school/students/controller/school_student_list_controller.go
package controller

import (
	"strings"
	"time"

	dto "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/dto"
	model "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

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

	// search code/note
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		q = q.Where(`
			LOWER(COALESCE(school_student_code, '')) LIKE ? OR
			LOWER(COALESCE(school_student_note, '')) LIKE ?
		`, like, like)
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

	// pagination info
	pg := helper.BuildPaginationFromOffset(total, p.Offset(), p.Limit())

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
		FullNameSnapshot  *string   `json:"full_name_snapshot,omitempty"`
		AvatarURL         *string   `json:"avatar_url,omitempty"`
		WhatsappURL       *string   `json:"whatsapp_url,omitempty"`
		ParentName        *string   `json:"parent_name,omitempty"`
		ParentWhatsappURL *string   `json:"parent_whatsapp_url,omitempty"`
		Gender            *string   `json:"gender,omitempty"` // NEW
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
			FullNameSnapshot  *string   `gorm:"column:user_profile_full_name_snapshot"`
			AvatarURL         *string   `gorm:"column:user_profile_avatar_url"`
			WhatsappURL       *string   `gorm:"column:user_profile_whatsapp_url"`
			ParentName        *string   `gorm:"column:user_profile_parent_name"`
			ParentWhatsappURL *string   `gorm:"column:user_profile_parent_whatsapp_url"`
			Gender            *string   `gorm:"column:user_profile_gender"` // NEW
		}

		if err := h.DB.
			Table("user_profiles").
			Select(`
				user_profile_id,
				user_profile_full_name_snapshot,
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
				FullNameSnapshot:  pr.FullNameSnapshot,
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
