package controller

import (
	yModel "masjidku_backend/internals/features/lembaga/teachers_students/model"
	helper "masjidku_backend/internals/helpers"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	yDTO "masjidku_backend/internals/features/lembaga/teachers_students/dto"

	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* ===================== LIST ===================== */
// GET /api/a/masjid-teachers
// Query: page, per_page|limit, sort_by(created_at|updated_at), order
//        id, user_id, employment, active, verified, public,
//        joined_from, joined_to (YYYY-MM-DD), q (notes/code/slug ILIKE)
//        include=user
func (ctrl *MasjidTeacherController) List(c *fiber.Ctx) error {
	masjidUUID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	p := helper.ParseFiber(c, "created_at", "desc", helper.DefaultOpts)
	allowedSort := map[string]string{
		"created_at": "masjid_teacher_created_at",
		"updated_at": "masjid_teacher_updated_at",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "created_at")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid sort_by")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// filters
	idStr := strings.TrimSpace(c.Query("id"))
	userIDStr := strings.TrimSpace(c.Query("user_id"))
	employment := strings.ToLower(strings.TrimSpace(c.Query("employment")))
	activeStr := strings.TrimSpace(c.Query("active"))
	verifiedStr := strings.TrimSpace(c.Query("verified"))
	publicStr := strings.TrimSpace(c.Query("public"))
	joinedFromStr := strings.TrimSpace(c.Query("joined_from"))
	joinedToStr := strings.TrimSpace(c.Query("joined_to"))
	q := strings.TrimSpace(c.Query("q"))

	var (
		rowID  uuid.UUID
		userID uuid.UUID
	)
	if idStr != "" {
		v, er := uuid.Parse(idStr)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "id invalid")
		}
		rowID = v
	}
	if userIDStr != "" {
		v, er := uuid.Parse(userIDStr)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "user_id invalid")
		}
		userID = v
	}

	tx := ctrl.DB.WithContext(c.Context()).
		Model(&yModel.MasjidTeacherModel{}).
		Where("masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL", masjidUUID)

	if rowID != uuid.Nil {
		tx = tx.Where("masjid_teacher_id = ?", rowID)
	}
	if userID != uuid.Nil {
		tx = tx.Where("masjid_teacher_user_id = ?", userID)
	}
	if employment != "" {
		tx = tx.Where("masjid_teacher_employment = ?", employment)
	}
	if activeStr != "" {
		if b, er := strconv.ParseBool(activeStr); er == nil {
			tx = tx.Where("masjid_teacher_is_active = ?", b)
		}
	}
	if verifiedStr != "" {
		if b, er := strconv.ParseBool(verifiedStr); er == nil {
			tx = tx.Where("masjid_teacher_is_verified = ?", b)
		}
	}
	if publicStr != "" {
		if b, er := strconv.ParseBool(publicStr); er == nil {
			tx = tx.Where("masjid_teacher_is_public = ?", b)
		}
	}
	if joinedFromStr != "" {
		if t, er := parseDateYYYYMMDD(joinedFromStr); er == nil {
			tx = tx.Where("masjid_teacher_joined_at IS NOT NULL AND masjid_teacher_joined_at >= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "joined_from invalid (YYYY-MM-DD)")
		}
	}
	if joinedToStr != "" {
		if t, er := parseDateYYYYMMDD(joinedToStr); er == nil {
			tx = tx.Where("masjid_teacher_joined_at IS NOT NULL AND masjid_teacher_joined_at <= ?", t)
		} else {
			return helper.JsonError(c, fiber.StatusBadRequest, "joined_to invalid (YYYY-MM-DD)")
		}
	}
	if q != "" {
		pat := "%" + q + "%"
		tx = tx.Where(`(masjid_teacher_notes ILIKE ? OR masjid_teacher_code ILIKE ? OR masjid_teacher_slug ILIKE ?)`, pat, pat, pat)
	}

	// count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// data
	var rows []yModel.MasjidTeacherModel
	if err := tx.Order(orderExpr).Limit(p.Limit()).Offset(p.Offset()).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// include=user ?
	wantUser := false
	if inc := strings.ToLower(strings.TrimSpace(c.Query("include"))); inc != "" {
		for _, part := range strings.Split(inc, ",") {
			if strings.TrimSpace(part) == "user" {
				wantUser = true
				break
			}
		}
	}

	// DTO dasar
	base := make([]*yDTO.MasjidTeacherResponse, 0, len(rows))
	userIDsSet := make(map[uuid.UUID]struct{}, len(rows))
	for i := range rows {
		base = append(base, yDTO.NewMasjidTeacherResponse(&rows[i]))
		if wantUser && rows[i].MasjidTeacherUserID != uuid.Nil {
			userIDsSet[rows[i].MasjidTeacherUserID] = struct{}{}
		}
	}

	if !wantUser {
		meta := helper.BuildMeta(total, p)
		return helper.JsonList(c, base, meta)
	}

	// bulk fetch users
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
	if len(userIDs) > 0 {
		var urows []UserLite
		if err := ctrl.DB.
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

	type Item struct {
		*yDTO.MasjidTeacherResponse `json:",inline"`
		User                        *UserLite `json:"user,omitempty"`
	}
	out := make([]Item, 0, len(base))
	for i := range rows {
		var u *UserLite
		if v, ok := userMap[rows[i].MasjidTeacherUserID]; ok {
			tmp := v
			u = &tmp
		}
		out = append(out, Item{MasjidTeacherResponse: base[i], User: u})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}

