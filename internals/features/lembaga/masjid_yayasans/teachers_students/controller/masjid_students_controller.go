// internals/features/lembaga/teachers_students/controller/masjid_student_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/dto"
	model "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model"
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

// ambil multi-value query (?key=a&key=b atau ?key=a,b)
func getMultiQuery(c *fiber.Ctx, key string) []string {
	out := make([]string, 0, 2)

	if qa := c.Context().QueryArgs(); qa != nil {
		raw := qa.PeekMulti(key)
		for _, b := range raw {
			out = append(out, string(b))
		}
	}
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

	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()
	if err := h.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "created", dto.FromModel(m))
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

// POST /api/a/masjid-students/:id/restore
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
