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

/* ========================================================
   Controller
======================================================== */
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

// helpers kecil lokal
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




// toResponse memetakan model -> DTO respons
func toResponse(m *model.AssessmentModel) dto.AssessmentResponse {
	var deletedAt *time.Time
	if m.AssessmentsDeletedAt.Valid {
		t := m.AssessmentsDeletedAt.Time
		deletedAt = &t
	}

	return dto.AssessmentResponse{
		AssessmentsID:                           m.AssessmentsID,
		AssessmentsMasjidID:                     m.AssessmentsMasjidID,
		AssessmentsClassSectionSubjectTeacherID: m.AssessmentsClassSectionSubjectTeacherID,
		AssessmentsTypeID:                       m.AssessmentsTypeID,

		AssessmentsTitle:       m.AssessmentsTitle,
		AssessmentsDescription: m.AssessmentsDescription,

		AssessmentsStartAt: m.AssessmentsStartAt,
		AssessmentsDueAt:   m.AssessmentsDueAt,

		AssessmentsMaxScore: m.AssessmentsMaxScore,

		AssessmentsIsPublished:     m.AssessmentsIsPublished,
		AssessmentsAllowSubmission: m.AssessmentsAllowSubmission,

		AssessmentsCreatedByTeacherID: m.AssessmentsCreatedByTeacherID,

		AssessmentsCreatedAt: m.AssessmentsCreatedAt,
		AssessmentsUpdatedAt: m.AssessmentsUpdatedAt,
		AssessmentsDeletedAt: deletedAt,
	}
}

// Validasi opsional: created_by_teacher_id (jika ada) memang milik masjid
func (ctl *AssessmentController) assertTeacherBelongsToMasjid(
    ctx context.Context,
    masjidID uuid.UUID,
    teacherID *uuid.UUID,
) error {
    if teacherID == nil || *teacherID == uuid.Nil {
        return nil
    }
    var n int64
    // Pastikan nama kolom sesuai skema mu:
    // umumnya: masjid_teacher_id, masjid_teacher_masjid_id, masjid_teacher_deleted_at
    if err := ctl.DB.WithContext(ctx).
        Table("masjid_teachers").
        Where("masjid_teacher_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teacher_deleted_at IS NULL",
            *teacherID, masjidID).
        Count(&n).Error; err != nil {
        return err
    }
    if n == 0 {
        return errors.New("assessments_created_by_teacher_id bukan milik masjid ini")
    }
    return nil
}




// POST /assessments
func (ctl *AssessmentController) Create(c *fiber.Ctx) error {
	var req dto.CreateAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, mid); err != nil {
		return err
	}

	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// validasi creator (opsional)
	if err := ctl.assertTeacherBelongsToMasjid(c.Context(), req.AssessmentsMasjidID, req.AssessmentsCreatedByTeacherID); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// validasi waktu: due >= start (kalau keduanya ada)
	if req.AssessmentsStartAt != nil && req.AssessmentsDueAt != nil &&
		req.AssessmentsDueAt.Before(*req.AssessmentsStartAt) {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessments_due_at harus setelah atau sama dengan assessments_start_at")
	}

	now := time.Now()

	row := model.AssessmentModel{
		// ❌ JANGAN set AssessmentsID ke uuid.Nil, biarkan DB atau app yang generate
		// ✅ Opsi A (biar DB generate): pastikan kolom punya DEFAULT gen_random_uuid() dan tag gorm default
		// ✅ Opsi B (app-generate): aktifkan baris di bawah ini dan pastikan kolom tidak override
		AssessmentsID: uuid.New(),

		AssessmentsMasjidID:                     req.AssessmentsMasjidID,
		AssessmentsClassSectionSubjectTeacherID: req.AssessmentsClassSectionSubjectTeacherID,
		AssessmentsTypeID:                       req.AssessmentsTypeID,

		AssessmentsTitle:       strings.TrimSpace(req.AssessmentsTitle),
		AssessmentsDescription: nil,

		AssessmentsMaxScore:        100,
		AssessmentsIsPublished:     true,
		AssessmentsAllowSubmission: true,

		AssessmentsCreatedByTeacherID: req.AssessmentsCreatedByTeacherID,

		AssessmentsCreatedAt: now,
		AssessmentsUpdatedAt: now,
	}

	// optional fields
	if req.AssessmentsDescription != nil {
		if d := strings.TrimSpace(*req.AssessmentsDescription); d != "" {
			row.AssessmentsDescription = &d
		}
	}
	if req.AssessmentsMaxScore != nil {
		row.AssessmentsMaxScore = *req.AssessmentsMaxScore
	}
	if req.AssessmentsIsPublished != nil {
		row.AssessmentsIsPublished = *req.AssessmentsIsPublished
	}
	if req.AssessmentsAllowSubmission != nil {
		row.AssessmentsAllowSubmission = *req.AssessmentsAllowSubmission
	}
	// ⬇️ pointer → value
	if req.AssessmentsStartAt != nil {
		row.AssessmentsStartAt = req.AssessmentsStartAt
	}
	if req.AssessmentsDueAt != nil {
		row.AssessmentsDueAt = req.AssessmentsDueAt
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat assessment")
	}

	// Safety: kalau masih nol (misal default DB tidak terset), fallback generate di app (opsional)
	if row.AssessmentsID == uuid.Nil {
		row.AssessmentsID = uuid.New()
		_ = ctl.DB.WithContext(c.Context()).
			Model(&model.AssessmentModel{}).
			Where("assessments_created_at = ? AND assessments_title = ? AND assessments_masjid_id = ?",
				row.AssessmentsCreatedAt, row.AssessmentsTitle, row.AssessmentsMasjidID).
			Update("assessments_id", row.AssessmentsID).Error
	}

	return helper.JsonCreated(c, "Assessment berhasil dibuat", toResponse(&row))
}


// PATCH /assessments/:id (partial)
// PATCH /assessments/:id (partial)
func (ctl *AssessmentController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessments_id tidak valid")
	}

	var req dto.PatchAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	
	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, mid); err != nil {
		return err
	}

	var existing model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, mid).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Validasi guru jika diubah
	if req.AssessmentsCreatedByTeacherID != nil {
		if err := ctl.assertTeacherBelongsToMasjid(
			c.Context(),
			mid,
			req.AssessmentsCreatedByTeacherID,
		); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// ===== Validasi waktu bila diubah =====
	switch {
	case req.AssessmentsStartAt != nil && req.AssessmentsDueAt != nil:
		// Keduanya diubah → due >= start
		if req.AssessmentsDueAt.Before(*req.AssessmentsStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessments_due_at harus setelah atau sama dengan assessments_start_at")
		}
	case req.AssessmentsStartAt != nil && req.AssessmentsDueAt == nil:
		// Start diubah saja → pastikan due existing (jika ada) tidak sebelum start baru
		if existing.AssessmentsDueAt != nil && existing.AssessmentsDueAt.Before(*req.AssessmentsStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "Tanggal due saat ini lebih awal dari start baru")
		}
	case req.AssessmentsStartAt == nil && req.AssessmentsDueAt != nil:
		// Due diubah saja → pastikan due baru tidak sebelum start existing
		if existing.AssessmentsStartAt != nil && req.AssessmentsDueAt.Before(*existing.AssessmentsStartAt) {
			return helper.JsonError(c, fiber.StatusBadRequest, "assessments_due_at tidak boleh sebelum assessments_start_at")
		}
	}

	// ===== Build updates =====
	updates := map[string]interface{}{}

	if req.AssessmentsTitle != nil {
		updates["assessments_title"] = strings.TrimSpace(*req.AssessmentsTitle)
	}
	if req.AssessmentsDescription != nil {
		updates["assessments_description"] = strings.TrimSpace(*req.AssessmentsDescription)
	}
	if req.AssessmentsStartAt != nil {
		// model pakai *time.Time, tapi gorm bisa terima value juga
		updates["assessments_start_at"] = *req.AssessmentsStartAt
	}
	if req.AssessmentsDueAt != nil {
		updates["assessments_due_at"] = *req.AssessmentsDueAt
	}
	if req.AssessmentsMaxScore != nil {
		updates["assessments_max_score"] = *req.AssessmentsMaxScore
	}
	if req.AssessmentsIsPublished != nil {
		updates["assessments_is_published"] = *req.AssessmentsIsPublished
	}
	if req.AssessmentsAllowSubmission != nil {
		updates["assessments_allow_submission"] = *req.AssessmentsAllowSubmission
	}
	if req.AssessmentsTypeID != nil {
		updates["assessments_type_id"] = *req.AssessmentsTypeID
	}
	if req.AssessmentsClassSectionSubjectTeacherID != nil {
		updates["assessments_class_section_subject_teacher_id"] = *req.AssessmentsClassSectionSubjectTeacherID
	}
	if req.AssessmentsCreatedByTeacherID != nil {
		updates["assessments_created_by_teacher_id"] = *req.AssessmentsCreatedByTeacherID
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", toResponse(&existing))
	}
	updates["assessments_updated_at"] = time.Now()

	if err := ctl.DB.WithContext(c.Context()).
		Model(&model.AssessmentModel{}).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, mid).
		Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui assessment")
	}

	var after model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, mid).
		First(&after).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memuat ulang assessment")
	}

	return helper.JsonUpdated(c, "Assessment berhasil diperbarui", toResponse(&after))
}

// DELETE /assessments/:id (soft delete)
func (ctl *AssessmentController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessments_id tidak valid")
	}

	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, mid); err != nil {
		return err
	}

	var row model.AssessmentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, mid).
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
		"assessments_id": id,
	})
}
