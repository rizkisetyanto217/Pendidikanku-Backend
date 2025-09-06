package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/lembaga/teachers_students/dto"
	model "masjidku_backend/internals/features/lembaga/teachers_students/model"
	helper "masjidku_backend/internals/helpers"
)

/* =========================
   Controller & Constructor
   ========================= */

type MasjidStudentController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func New(db *gorm.DB, v *validator.Validate) *MasjidStudentController {
	return &MasjidStudentController{DB: db, Validate: v}
}

/* =========================
   Helpers
   ========================= */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	idStr := strings.TrimSpace(c.Params(name))
	if idStr == "" {
		return uuid.Nil, errors.New(name + " is required")
	}
	u, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, errors.New(name + " is invalid uuid")
	}
	return u, nil
}

func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}


// di file controller yg sama (atau taruh di helper internal kalian)
// helper lokal: ambil multi-value query utk Fiber v2 (+ fallback comma-separated)
func getMultiQuery(c *fiber.Ctx, key string) []string {
	out := make([]string, 0, 2)

	// Fiber v2: multi via QueryArgs().PeekMulti
	if qa := c.Context().QueryArgs(); qa != nil {
		raw := qa.PeekMulti(key) // [][]byte
		for _, b := range raw {
			out = append(out, string(b))
		}
	}

	// Fallback: single value atau comma-separated (?key=a,b)
	if len(out) == 0 {
		if s := strings.TrimSpace(c.Query(key)); s != "" {
			if strings.Contains(s, ",") {
				for _, part := range strings.Split(s, ",") {
					if v := strings.TrimSpace(part); v != "" {
						out = append(out, v)
					}
				}
			} else {
				out = append(out, s)
			}
		}
	}
	return out
}


/* =========================
   Routes Handlers
   ========================= */

// POST /api/a/masjid-students
func (h *MasjidStudentController) Create(c *fiber.Ctx) error {
	var req dto.MasjidStudentCreateReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}
	req.Normalize()

	// (opsional) enforce MasjidContext dari Locals
	// if v := c.Locals("masjid_id"); v != nil {
	// 	if id, ok := v.(uuid.UUID); ok && id != uuid.Nil {
	// 		req.MasjidStudentMasjidID = id
	// 	}
	// }
	// if v := c.Locals("user_id"); v != nil && req.MasjidStudentUserID == uuid.Nil {
	// 	if id, ok := v.(uuid.UUID); ok && id != uuid.Nil {
	// 		req.MasjidStudentUserID = id
	// 	}
	// }

	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "created", dto.FromModel(m))
}

// GET /api/a/masjid-students/:id
func (h *MasjidStudentController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.MasjidStudentModel
	if err := h.DB.First(&m, "masjid_student_id = ?", id).Error; err != nil {
		if isNotFound(err) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "ok", dto.FromModel(&m))
}


// GET /api/a/masjid-students
// Query: page|per_page|limit, search, status_in (multi), masjid_id, user_id, created_ge, created_le, sort_by, sort(order)
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
		createdGe   = strings.TrimSpace(c.Query("created_ge"))
		createdLe   = strings.TrimSpace(c.Query("created_le"))
	)

	var (
		masjidID uuid.UUID
		userID   uuid.UUID
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

	// search in code or note (case-insensitive)
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		q = q.Where(`
			LOWER(COALESCE(masjid_student_code, '')) LIKE ? OR
			LOWER(COALESC E(masjid_student_note, '')) LIKE ?
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

	resp := make([]dto.MasjidStudentResp, 0, len(rows))
	for i := range rows {
		resp = append(resp, dto.FromModel(&rows[i]))
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, resp, meta)
}

// PUT /api/a/masjid-students/:id
func (h *MasjidStudentController) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.MasjidStudentModel
	if err := h.DB.First(&m, "masjid_student_id = ?", id).Error; err != nil {
		if isNotFound(err) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var req dto.MasjidStudentUpdateReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	req.Apply(&m)

	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "updated", dto.FromModel(&m))
}

// PATCH /api/a/masjid-students/:id
func (h *MasjidStudentController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.MasjidStudentModel
	if err := h.DB.First(&m, "masjid_student_id = ?", id).Error; err != nil {
		if isNotFound(err) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var req dto.MasjidStudentPatchReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	req.Apply(&m)

	if err := h.DB.Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "patched", dto.FromModel(&m))
}

// DELETE /api/a/masjid-students/:id  (soft delete)
func (h *MasjidStudentController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.Delete(&model.MasjidStudentModel{}, "masjid_student_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "deleted", fiber.Map{"masjid_student_id": id})
}

// POST /api/a/masjid-students/:id/restore (optional)
func (h *MasjidStudentController) Restore(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.MasjidStudentModel
	if err := h.DB.Unscoped().First(&m, "masjid_student_id = ?", id).Error; err != nil {
		if isNotFound(err) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// clear deleted_at
	if err := h.DB.Model(&m).Update("masjid_student_deleted_at", nil).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "restored", dto.FromModel(&m))
}
