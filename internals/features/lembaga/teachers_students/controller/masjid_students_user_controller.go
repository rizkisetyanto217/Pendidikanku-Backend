package controller

import (
	dto "masjidku_backend/internals/features/lembaga/teachers_students/dto"
	model "masjidku_backend/internals/features/lembaga/teachers_students/model"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GET /api/a/masjid-students
// Query: page|per_page|limit, search, status_in (multi), masjid_id, user_id, created_ge, created_le, sort_by, sort(order)
// GET /api/a/masjid-students
// Query: page|per_page|limit, search, status_in (multi), masjid_id, user_id, id,
//        created_ge, created_le, sort_by, sort(order), include=user
func (h *MasjidStudentController) List(c *fiber.Ctx) error {
	// Pagination & Sorting via helper
	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)

	// Whitelist sort key -> kolom DB
	allowedSort := map[string]string{
		"created_at": "masjid_student_created_at",
		"updated_at": "masjid_student_updated_at",
		"code":       "masjid_student_code",
		"status":     "masjid_student_status",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid sort_by")
	}
	orderClause = strings.TrimPrefix(orderClause, "ORDER BY ")

	// Filters
	search := strings.TrimSpace(c.Query("search"))
	var (
		masjidIDStr = strings.TrimSpace(c.Query("masjid_id"))
		userIDStr   = strings.TrimSpace(c.Query("user_id"))
		idStr       = strings.TrimSpace(c.Query("id")) // NEW: filter by masjid_student_id
		createdGe   = strings.TrimSpace(c.Query("created_ge"))
		createdLe   = strings.TrimSpace(c.Query("created_le"))
	)

	var (
		masjidID uuid.UUID
		userID   uuid.UUID
		rowID    uuid.UUID
	)
	if masjidIDStr != "" {
		if v, err := uuid.Parse(masjidIDStr); err == nil {
			masjidID = v
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "masjid_id invalid")
		}
	}
	if userIDStr != "" {
		if v, err := uuid.Parse(userIDStr); err == nil {
			userID = v
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "user_id invalid")
		}
	}
	if idStr != "" {
		if v, err := uuid.Parse(idStr); err == nil {
			rowID = v
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "id invalid")
		}
	}

	// status_in (multi value safe di Fiber v2 + fallback)
	statusIn := getMultiQuery(c, "status_in")
	normStatus := make([]string, 0, len(statusIn))
	for _, s := range statusIn {
		s = strings.ToLower(strings.TrimSpace(s))
		switch s {
		case model.MasjidStudentStatusActive,
			model.MasjidStudentStatusInactive,
			model.MasjidStudentStatusAlumni:
			normStatus = append(normStatus, s)
		}
	}

	q := h.DB.Model(&model.MasjidStudentModel{})

	// (Opsional) Enforce MasjidContext dari Locals
	// if v := c.Locals("masjid_id"); v != nil {
	// 	if ctxMasjidID, ok := v.(uuid.UUID); ok && ctxMasjidID != uuid.Nil {
	// 		q = q.Where("masjid_student_masjid_id = ?", ctxMasjidID)
	// 	}
	// }

	if rowID != uuid.Nil {
		q = q.Where("masjid_student_id = ?", rowID)
	}
	if masjidID != uuid.Nil {
		q = q.Where("masjid_student_masjid_id = ?", masjidID)
	}
	if userID != uuid.Nil {
		q = q.Where("masjid_student_user_id = ?", userID)
	}
	if len(normStatus) > 0 {
		q = q.Where("masjid_student_status IN ?", normStatus)
	}

	// created_at range (RFC3339)
	const layout = time.RFC3339
	if createdGe != "" {
		if t, err := time.Parse(layout, createdGe); err == nil {
			q = q.Where("masjid_student_created_at >= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "created_ge invalid (use RFC3339)")
		}
	}
	if createdLe != "" {
		if t, err := time.Parse(layout, createdLe); err == nil {
			q = q.Where("masjid_student_created_at <= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "created_le invalid (use RFC3339)")
		}
	}

	// search in code or note (case-insensitive)  ← fixed COALESCE typo
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		q = q.Where(`
			LOWER(COALESCE(masjid_student_code, '')) LIKE ? OR
			LOWER(COALESCE(masjid_student_note, '')) LIKE ?
		`, like, like)
	}

	// count total
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// data
	var rows []model.MasjidStudentModel
	if err := q.Order(orderClause).Offset(p.Offset()).Limit(p.Limit()).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// =========================
	// include=user (bulk fetch users, no N+1)
	// =========================
	include := strings.ToLower(strings.TrimSpace(c.Query("include")))
	wantUser := false
	if include != "" {
		for _, part := range strings.Split(include, ",") {
			if strings.TrimSpace(part) == "user" {
				wantUser = true
				break
			}
		}
	}

	// Response dasar
	baseResp := make([]dto.MasjidStudentResp, 0, len(rows))
	for i := range rows {
		baseResp = append(baseResp, dto.FromModel(&rows[i]))
	}

	// Kalau tidak minta user → return biasa
	if !wantUser {
		meta := helper.BuildMeta(total, p)
		return helper.JsonList(c, baseResp, meta)
	}

	// Siapkan struktur response dengan user
	type UserLite struct {
		ID       uuid.UUID `json:"id"`
		UserName string    `json:"user_name"`
		FullName *string   `json:"full_name,omitempty"`
		Email    string    `json:"email"`
		IsActive bool      `json:"is_active"`
	}
	type MasjidStudentWithUserResp struct {
		dto.MasjidStudentResp `json:",inline"`
		User                  *UserLite `json:"user,omitempty"`
	}

	// Kumpulkan user_ids unik dari rows (field model: MasjidStudentUserID)
	userIDsSet := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		if rows[i].MasjidStudentUserID != uuid.Nil {
			userIDsSet[rows[i].MasjidStudentUserID] = struct{}{}
		}
	}
	userIDs := make([]uuid.UUID, 0, len(userIDsSet))
	for id := range userIDsSet {
		userIDs = append(userIDs, id)
	}

	// Ambil users dalam 1 query
	// Ambil users dalam 1 query (no S1016)
	userMap := make(map[uuid.UUID]UserLite, len(userIDs))
	if len(userIDs) > 0 {
		var urows []UserLite
		if err := h.DB.
			Table("users").
			Select("id, user_name, full_name, email, is_active").
			Where("id IN ?", userIDs).
			Where("deleted_at IS NULL").
			Find(&urows).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		for _, u := range urows {
			userMap[u.ID] = u // langsung assign; tidak pakai struct literal
		}
	}


	// Gabungkan
	out := make([]MasjidStudentWithUserResp, 0, len(rows))
	for i := range rows {
		base := baseResp[i]
		var u *UserLite
		if val, ok := userMap[rows[i].MasjidStudentUserID]; ok {
			tmp := val
			u = &tmp
		}
		out = append(out, MasjidStudentWithUserResp{
			MasjidStudentResp: base,
			User:              u,
		})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}
