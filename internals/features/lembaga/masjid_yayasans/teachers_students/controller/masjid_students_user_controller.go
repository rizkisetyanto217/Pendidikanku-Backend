// file: internals/features/lembaga/masjid_yayasans/teachers_students/controller/masjid_student_list_controller.go
package controller

import (
	"strings"
	"time"

	dto "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/dto"
	model "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/a/masjid-students
// Query:
//
//	page|per_page|limit,
//	search,
//	status_in (multi: active,inactive,alumni),
//	user_profile_id,
//	id,
//	created_ge, created_le (RFC3339),
//	sort_by(created_at|updated_at|code|status), sort(asc|desc),
//	include=user_profile (alias: include=user)
func (h *MasjidStudentController) List(c *fiber.Ctx) error {
	// Pastikan DB ada di Locals untuk helper resolver slug→id
	if c.Locals("DB") == nil {
		c.Locals("DB", h.DB)
	}

	// 1) Resolve & Enforce Masjid Context (DKM)
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	enforcedMasjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// 2) Pagination & Sorting (whitelist)
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)
	allowedSort := map[string]string{
		"created_at": "masjid_student_created_at",
		"updated_at": "masjid_student_updated_at",
		"code":       "masjid_student_code",
		"status":     "masjid_student_status",
		// bisa ditambah: "slug": "masjid_student_slug",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid sort_by")
	}
	orderClause = strings.TrimPrefix(orderClause, "ORDER BY ")

	// 3) Filters
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

	// status_in (multi) → normalisasi ke set yang valid
	statusIn := getMultiQuery(c, "status_in")
	normStatus := make([]string, 0, len(statusIn))
	for _, s := range statusIn {
		s = strings.ToLower(strings.TrimSpace(s))
		switch model.MasjidStudentStatus(s) {
		case model.MasjidStudentActive, model.MasjidStudentInactive, model.MasjidStudentAlumni:
			normStatus = append(normStatus, s)
		}
	}

	q := h.DB.Model(&model.MasjidStudentModel{})

	// tenant-scope
	q = q.Where("masjid_student_masjid_id = ?", enforcedMasjidID)

	if rowID != uuid.Nil {
		q = q.Where("masjid_student_id = ?", rowID)
	}
	if userProfileID != uuid.Nil {
		q = q.Where("masjid_student_user_profile_id = ?", userProfileID)
	}
	if len(normStatus) > 0 {
		q = q.Where("masjid_student_status IN ?", normStatus)
	}

	// created_at range (RFC3339)
	const layout = time.RFC3339
	if createdGe != "" {
		t, err := time.Parse(layout, createdGe)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "created_ge invalid (use RFC3339)")
		}
		q = q.Where("masjid_student_created_at >= ?", t)
	}
	if createdLe != "" {
		t, err := time.Parse(layout, createdLe)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "created_le invalid (use RFC3339)")
		}
		q = q.Where("masjid_student_created_at <= ?", t)
	}

	// Search di code/note (case-insensitive)
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		q = q.Where(`
			LOWER(COALESCE(masjid_student_code, '')) LIKE ? OR
			LOWER(COALESCE(masjid_student_note, '')) LIKE ?
		`, like, like)
		// (opsional) tambah slug/name snapshot bila perlu:
		// OR LOWER(masjid_student_slug) LIKE ?
		// OR LOWER(COALESCE(masjid_student_user_profile_name_snapshot,'')) LIKE ?
	}

	// 4) Count + Fetch
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.MasjidStudentModel
	if err := q.Order(orderClause).Offset(p.Offset()).Limit(p.Limit()).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 5) include=user_profile (alias: include=user)
	include := strings.ToLower(strings.TrimSpace(c.Query("include")))
	wantProfile := false
	if include != "" {
		for _, part := range strings.Split(include, ",") {
			part = strings.TrimSpace(part)
			if part == "user_profile" || part == "user" {
				wantProfile = true
				break
			}
		}
	}

	baseResp := make([]dto.MasjidStudentResp, 0, len(rows))
	for i := range rows {
		baseResp = append(baseResp, dto.FromModel(&rows[i]))
	}

	if !wantProfile {
		meta := helper.BuildMeta(total, p)
		return helper.JsonList(c, baseResp, meta)
	}

	// ---- Join ringan ke users_profile (sinkron DDL teranyar)
	type ProfileLite struct {
		ID                uuid.UUID `json:"id"`
		Name              *string   `json:"name,omitempty"`
		AvatarURL         *string   `json:"avatar_url,omitempty"`
		WhatsappURL       *string   `json:"whatsapp_url,omitempty"`
		ParentName        *string   `json:"parent_name,omitempty"`
		ParentWhatsappURL *string   `json:"parent_whatsapp_url,omitempty"`
	}

	type MasjidStudentWithProfileResp struct {
		dto.MasjidStudentResp `json:",inline"`
		UserProfile           *ProfileLite `json:"user_profile,omitempty"`
	}

	// Kumpulkan profile_ids unik
	profileIDsSet := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		if rows[i].MasjidStudentUserProfileID != uuid.Nil {
			profileIDsSet[rows[i].MasjidStudentUserProfileID] = struct{}{}
		}
	}
	profileIDs := make([]uuid.UUID, 0, len(profileIDsSet))
	for id := range profileIDsSet {
		profileIDs = append(profileIDs, id)
	}

	// Ambil users_profile dalam 1 query (PERHATIKAN nama tabel & kolom!)
	profileMap := make(map[uuid.UUID]ProfileLite, len(profileIDs))
	if len(profileIDs) > 0 {
		var profRows []ProfileLite
		if err := h.DB.
			Table("users_profile").
			Select(`
				users_profile_id                                  AS id,
				user_profile_name                                 AS name,
				user_profile_avatar_url                           AS avatar_url,
				user_profile_whatsapp_url                         AS whatsapp_url,
				user_profile_parent_name                          AS parent_name,
				user_profile_parent_whatsapp_url                  AS parent_whatsapp_url
			`).
			Where("users_profile_id IN ?", profileIDs).
			Where("users_profile_deleted_at IS NULL").
			Find(&profRows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		for _, pr := range profRows {
			profileMap[pr.ID] = pr
		}
	}

	// Merge profile
	out := make([]MasjidStudentWithProfileResp, 0, len(rows))
	for i := range rows {
		base := baseResp[i]
		var up *ProfileLite
		if val, ok := profileMap[rows[i].MasjidStudentUserProfileID]; ok {
			tmp := val
			up = &tmp
		}
		out = append(out, MasjidStudentWithProfileResp{
			MasjidStudentResp: base,
			UserProfile:       up,
		})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}
