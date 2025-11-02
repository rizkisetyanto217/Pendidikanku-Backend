// internals/features/lembaga/teachers_students/controller/school_student_controller.go
package controller

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "schoolku_backend/internals/features/lembaga/school_yayasans/teachers_students/dto"
	model "schoolku_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	snapshotUserProfile "schoolku_backend/internals/features/users/users/snapshot"
	helper "schoolku_backend/internals/helpers"
)

/* =========================
   Controller & Constructor
   ========================= */

type SchoolStudentController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func New(db *gorm.DB, v *validator.Validate) *SchoolStudentController {
	return &SchoolStudentController{DB: db, Validate: v}
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

// POST /api/a/school-students
// POST /api/a/school-students
func (h *SchoolStudentController) Create(c *fiber.Ctx) error {
	var req dto.SchoolStudentCreateReq
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}
	req.Normalize()

	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	m := req.ToModel()

	// ===== Snapshot user_profile (by profile_id) =====
	if m.SchoolStudentUserProfileID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "user_profile_id is required")
	}
	snap, err := snapshotUserProfile.BuildUserProfileSnapshotByProfileID(c.Context(), h.DB, m.SchoolStudentUserProfileID)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return helper.JsonError(c, fiber.StatusBadRequest, "user profile tidak ditemukan / sudah dihapus")
		default:
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil snapshot user profile")
		}
	}
	// Isi kolom snapshot di model (hanya yang tersedia di model kamu)
	if snap != nil {
		// Name (string wajib di struct snapshot → pastikan non-empty)
		if strings.TrimSpace(snap.Name) != "" {
			m.SchoolStudentUserProfileNameSnapshot = &snap.Name
		}
		m.SchoolStudentUserProfileAvatarURLSnapshot = snap.AvatarURL
		m.SchoolStudentUserProfileWhatsappURLSnapshot = snap.WhatsappURL
		m.SchoolStudentUserProfileParentNameSnapshot = snap.ParentName
		m.SchoolStudentUserProfileParentWhatsappURLSnapshot = snap.ParentWhatsappURL
		// snap.Slug / snap.DonationName / snap.Location tersedia, tapi model kamu
		// tidak punya kolom-kolom itu—abaikan saja.
	}

	// ===== Insert =====
	if err := h.DB.Create(m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "created", dto.FromModel(m))
}

// PUT /api/a/school-students/:id
func (h *SchoolStudentController) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.SchoolStudentModel
	if err := h.DB.First(&m, "school_student_id = ?", id).Error; err != nil {
		if isNotFound(err) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var req dto.SchoolStudentUpdateReq
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

// PATCH /api/a/school-students/:id
func (h *SchoolStudentController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.SchoolStudentModel
	if err := h.DB.First(&m, "school_student_id = ?", id).Error; err != nil {
		if isNotFound(err) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var req dto.SchoolStudentPatchReq
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

// DELETE /api/a/school-students/:id  (soft delete)
func (h *SchoolStudentController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.DB.Delete(&model.SchoolStudentModel{}, "school_student_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "deleted", fiber.Map{"school_student_id": id})
}

// POST /api/a/school-students/:id/restore
func (h *SchoolStudentController) Restore(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m model.SchoolStudentModel
	if err := h.DB.Unscoped().First(&m, "school_student_id = ?", id).Error; err != nil {
		if isNotFound(err) {
			return helper.JsonError(c, fiber.StatusNotFound, "not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// clear deleted_at
	if err := h.DB.Model(&m).Update("school_student_deleted_at", nil).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "restored", dto.FromModel(&m))
}
