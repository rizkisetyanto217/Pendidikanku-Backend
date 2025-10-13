// file: internals/features/school/assessments/controller/assessment_controller.go
package controller

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/submissions_assesments/assesments/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/assesments/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/*
========================================================

	Controller

========================================================
*/
type AssessmentController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewAssessmentController(db *gorm.DB) *AssessmentController {
	return &AssessmentController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* ========================================================
   Helpers
======================================================== */

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(c.Params(name)))
}

// validasi guru milik masjid
func (ctl *AssessmentController) assertTeacherBelongsToMasjid(
	ctx context.Context,
	masjidID uuid.UUID,
	teacherID *uuid.UUID,
) error {
	if teacherID == nil || *teacherID == uuid.Nil {
		return nil
	}
	var n int64
	if err := ctl.DB.WithContext(ctx).
		Table("masjid_teachers").
		Where(`
			masjid_teacher_id = ?
			AND masjid_teacher_masjid_id = ?
			AND masjid_teacher_deleted_at IS NULL
		`, *teacherID, masjidID).
		Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return errors.New("assessment_created_by_teacher_id bukan milik masjid ini")
	}
	return nil
}

// Resolver akses: DKM/Admin via helper, atau Teacher pada masjid tsb.
// Penting: pastikan caller sudah set c.Locals("DB", ctl.DB) sebelum memanggil.
func resolveMasjidForDKMOrTeacher(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) Ambil masjid context (path/header/cookie/query/host/token)
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return uuid.Nil, err
	}

	// 2) Coba jalur DKM/Admin (helper sudah handle slug→id internal)
	if id, er := helperAuth.EnsureMasjidAccessDKM(c, mc); er == nil && id != uuid.Nil {
		return id, nil
	}

	// 3) Fallback: izinkan GURU pada masjid ini
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetMasjidIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return uuid.Nil, fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
		}
		masjidID = id
	} else {
		return uuid.Nil, helperAuth.ErrMasjidContextMissing
	}

	// Guru valid jika token guru terikat ke masjid ini
	if helperAuth.IsTeacherInMasjid(c, masjidID) {
		return masjidID, nil
	}

	return uuid.Nil, helperAuth.ErrMasjidContextForbidden
}

/* ===============================
   Handlers
=============================== */

// POST /assessments
func (ctl *AssessmentController) Create(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	var req dto.CreateAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()

	// 🔒 resolve & authorize (DKM/Admin atau Teacher masjid)
	mid, err := resolveMasjidForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Enforce tenant dari context (anti cross-tenant injection)
	req.AssessmentMasjidID = mid

	// Validasi DTO
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Validasi creator teacher (opsional)
	if err := ctl.assertTeacherBelongsToMasjid(c.Context(), mid, req.AssessmentCreatedByTeacherID); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Validasi waktu
	if req.AssessmentStartAt != nil && req.AssessmentDueAt != nil &&
		req.AssessmentDueAt.Before(*req.AssessmentStartAt) {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at harus setelah atau sama dengan assessment_start_at")
	}

	// Build model dari DTO
	row := req.ToModel()

	// Simpan
	if err := ctl.DB.WithContext(c.Context()).Create(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat assessment")
	}

	return helper.JsonCreated(c, "Assessment berhasil dibuat", dto.FromModelAssesment(row))
}

// PATCH /assessments/:id
func (ctl *AssessmentController) Patch(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
	}

	var req dto.PatchAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔒 resolve & authorize
	mid, err := resolveMasjidForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var existing model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			assessment_id = ?
			AND assessment_masjid_id = ?
			AND assessment_deleted_at IS NULL
		`, id, mid).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// validasi guru bila diubah
	if req.AssessmentCreatedByTeacherID != nil {
		if err := ctl.assertTeacherBelongsToMasjid(c.Context(), mid, req.AssessmentCreatedByTeacherID); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// validasi waktu (kombinasi)
	switch {
	case req.AssessmentStartAt != nil && req.AssessmentDueAt != nil:
		if req.AssessmentDueAt.Before(*req.AssessmentStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at harus setelah atau sama dengan assessment_start_at")
		}
	case req.AssessmentStartAt != nil && req.AssessmentDueAt == nil:
		if existing.AssessmentDueAt != nil && existing.AssessmentDueAt.Before(*req.AssessmentStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Tanggal due saat ini lebih awal dari start baru")
		}
	case req.AssessmentStartAt == nil && req.AssessmentDueAt != nil:
		if existing.AssessmentStartAt != nil && req.AssessmentDueAt.Before(*existing.AssessmentStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessment_due_at tidak boleh sebelum assessment_start_at")
		}
	}

	// Terapkan PATCH via DTO.Apply
	req.Apply(&existing)
	existing.AssessmentUpdatedAt = time.Now()

	if err := ctl.DB.WithContext(c.Context()).Save(&existing).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui assessment")
	}

	return helper.JsonUpdated(c, "Assessment berhasil diperbarui", dto.FromModelAssesment(existing))
}

// DELETE /assessments/:id (soft delete)
func (ctl *AssessmentController) Delete(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context
	c.Locals("DB", ctl.DB)

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_id tidak valid")
	}

	// 🔒 resolve & authorize
	mid, err := resolveMasjidForDKMOrTeacher(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var row model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			assessment_id = ?
			AND assessment_masjid_id = ?
			AND assessment_deleted_at IS NULL
		`, id, mid).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus assessment")
	}

	return helper.JsonDeleted(c, "Assessment dihapus", fiber.Map{
		"assessment_id": id,
	})
}

/* ========================================================
   Helpers (local)
======================================================== */

func atoiOr(def int, s string) int {
	if s == "" {
		return def
	}
	n := 0
	sign := 1
	for i := 0; i < len(s); i++ {
		if i == 0 && s[i] == '-' {
			sign = -1
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return def
		}
		n = n*10 + int(s[i]-'0')
	}
	n *= sign
	if n <= 0 {
		return def
	}
	return n
}

func eqTrue(s string) bool {
	v := strings.TrimSpace(strings.ToLower(s))
	return v == "1" || v == "true"
}
