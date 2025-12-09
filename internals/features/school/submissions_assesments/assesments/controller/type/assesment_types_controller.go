// file: internals/features/school/assessments/controller/assessment_type_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/dto"
	assessmentModel "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/model"
	assessSvc "madinahsalam_backend/internals/features/school/submissions_assesments/assesments/service"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/* ========================= Controller ========================= */

type AssessmentTypeController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewAssessmentTypeController(db *gorm.DB) *AssessmentTypeController {
	return &AssessmentTypeController{
		DB:        db,
		Validator: validator.New(),
	}
}

func parseUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(c.Params(name)))
}

/* ========================= Helpers ========================= */

func mapToResponse(m *assessmentModel.AssessmentTypeModel) dto.AssessmentTypeResponse {
	// supaya tetep 1 sumber kebenaran di dto.FromModel
	return dto.FromModel(*m)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "sqlstate 23505") ||
		strings.Contains(s, "duplicate key value") ||
		strings.Contains(s, "unique constraint")
}

/* ========================= Handlers ========================= */

// POST /assessment-types — DKM/Admin untuk school tsb
func (ctl *AssessmentTypeController) Create(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context (kalau ada yang butuh)
	c.Locals("DB", ctl.DB)

	var req dto.CreateAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// DEBUG
	fmt.Printf("DEBUG CreateAssessmentTypeRequest: %+v\n", req)
	req = req.Normalize()

	// 🔒 School context: STRICT dari token active_school
	schoolID, err := helperAuth.GetActiveSchoolID(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "School aktif di token tidak ditemukan")
	}

	// 🔒 RBAC: hanya role DKM/Admin di school ini
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	// Enforce tenant: selalu override dari context, abaikan dari client
	req.AssessmentTypeSchoolID = schoolID

	// Validasi request
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Guard ekstra bobot 0..100
	if req.AssessmentTypeWeightPercent < 0 || req.AssessmentTypeWeightPercent > 100 {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity,
			"assessment_type_weight_percent harus di antara 0 hingga 100")
	}

	now := time.Now()

	// Build model dari DTO (supaya semua quiz settings + late policy ikut keisi dengan default)
	row := req.ToModel()
	row.AssessmentTypeID = uuid.New()
	row.AssessmentTypeCreatedAt = now
	row.AssessmentTypeUpdatedAt = now

	// Validasi agregat aktif ≤ 100
	if row.AssessmentTypeIsActive {
		sum, err := assessSvc.New().SumActiveWeights(
			ctl.DB.WithContext(c.Context()),
			schoolID,
			nil,
		)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total bobot")
		}
		if sum+row.AssessmentTypeWeightPercent > 100.0 {
			remaining := 100.0 - sum
			if remaining < 0 {
				remaining = 0
			}
			return helper.JsonError(c, fiber.StatusUnprocessableEntity,
				fmt.Sprintf("Total bobot melebihi 100. Sisa yang tersedia: %.2f%%", remaining))
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Key sudah dipakai untuk school ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Assessment type dibuat", mapToResponse(&row))
}

// PATCH /assessment-types/:id — DKM/Admin
func (ctl *AssessmentTypeController) Patch(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context (kalau ada yang butuh)
	c.Locals("DB", ctl.DB)

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_type_id tidak valid")
	}

	var req dto.PatchAssessmentTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 🔒 School context: STRICT dari token active_school
	schoolID, err := helperAuth.GetActiveSchoolID(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "School aktif di token tidak ditemukan")
	}

	// 🔒 DKM/Admin check
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	var existing assessmentModel.AssessmentTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessment_type_id = ? AND assessment_type_school_id = ? AND assessment_type_deleted_at IS NULL", id, schoolID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Hitung nilai akhir utk validasi agregat
	finalActive := existing.AssessmentTypeIsActive
	if req.AssessmentTypeIsActive != nil {
		finalActive = *req.AssessmentTypeIsActive
	}
	finalWeight := existing.AssessmentTypeWeightPercent
	if req.AssessmentTypeWeightPercent != nil {
		if *req.AssessmentTypeWeightPercent < 0 || *req.AssessmentTypeWeightPercent > 100 {
			return helper.JsonError(c, fiber.StatusUnprocessableEntity,
				"assessment_type_weight_percent harus di antara 0 hingga 100")
		}
		finalWeight = *req.AssessmentTypeWeightPercent
	}

	// Validasi agregat (aktif) ≤ 100 — exclude row ini
	if finalActive {
		sum, err := assessSvc.New().SumActiveWeights(
			ctl.DB.WithContext(c.Context()),
			schoolID,
			&existing.AssessmentTypeID,
		)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung total bobot")
		}
		if sum+finalWeight > 100.0 {
			remaining := 100.0 - sum
			if remaining < 0 {
				remaining = 0
			}
			return helper.JsonError(c, fiber.StatusUnprocessableEntity,
				fmt.Sprintf("Total bobot melebihi 100. Sisa yang tersedia: %.2f%%", remaining))
		}
	}

	updates := map[string]any{}
	if req.AssessmentTypeName != nil {
		updates["assessment_type_name"] = strings.TrimSpace(*req.AssessmentTypeName)
	}
	if req.AssessmentTypeWeightPercent != nil {
		updates["assessment_type_weight_percent"] = *req.AssessmentTypeWeightPercent
	}
	if req.AssessmentTypeIsActive != nil {
		updates["assessment_type_is_active"] = *req.AssessmentTypeIsActive
	}
	if req.AssessmentTypeIsGraded != nil {
		updates["assessment_type_is_graded"] = *req.AssessmentTypeIsGraded
	}

	// Ubah kategori besar assessment (training / daily_exam / exam)
	if req.AssessmentTypeCategory != nil {
		cat := strings.ToLower(strings.TrimSpace(*req.AssessmentTypeCategory))
		if cat != "" {
			updates["assessment_type"] = cat
		}
	}

	// ===== quiz settings =====
	if req.AssessmentTypeShuffleQuestions != nil {
		updates["assessment_type_shuffle_questions"] = *req.AssessmentTypeShuffleQuestions
	}
	if req.AssessmentTypeShuffleOptions != nil {
		updates["assessment_type_shuffle_options"] = *req.AssessmentTypeShuffleOptions
	}
	if req.AssessmentTypeShowCorrectAfterSubmit != nil {
		updates["assessment_type_show_correct_after_submit"] = *req.AssessmentTypeShowCorrectAfterSubmit
	}
	if req.AssessmentTypeStrictMode != nil {
		updates["assessment_type_strict_mode"] = *req.AssessmentTypeStrictMode
	}
	if req.AssessmentTypeTimePerQuestionSec != nil {
		// Catatan: dengan desain ini kita belum bisa clear ke NULL (tanpa batas) via PATCH
		// kecuali pakai pola khusus (mis. payload khusus untuk "clear").
		updates["assessment_type_time_per_question_sec"] = *req.AssessmentTypeTimePerQuestionSec
	}
	if req.AssessmentTypeAttemptsAllowed != nil {
		updates["assessment_type_attempts_allowed"] = *req.AssessmentTypeAttemptsAllowed
	}
	if req.AssessmentTypeRequireLogin != nil {
		updates["assessment_type_require_login"] = *req.AssessmentTypeRequireLogin
	}

	// ===== late policy & scoring =====
	if req.AssessmentTypeAllowLateSubmission != nil {
		updates["assessment_type_allow_late_submission"] = *req.AssessmentTypeAllowLateSubmission
	}
	if req.AssessmentTypeLatePenaltyPercent != nil {
		updates["assessment_type_late_penalty_percent"] = *req.AssessmentTypeLatePenaltyPercent
	}
	if req.AssessmentTypePassingScorePercent != nil {
		updates["assessment_type_passing_score_percent"] = *req.AssessmentTypePassingScorePercent
	}
	if req.AssessmentTypeScoreAggregationMode != nil {
		mode := strings.ToLower(strings.TrimSpace(*req.AssessmentTypeScoreAggregationMode))
		if mode != "" {
			updates["assessment_type_score_aggregation_mode"] = mode
		}
	}
	if req.AssessmentTypeShowScoreAfterSubmit != nil {
		updates["assessment_type_show_score_after_submit"] = *req.AssessmentTypeShowScoreAfterSubmit
	}
	if req.AssessmentTypeShowCorrectAfterClosed != nil {
		updates["assessment_type_show_correct_after_closed"] = *req.AssessmentTypeShowCorrectAfterClosed
	}
	if req.AssessmentTypeAllowReviewBeforeSubmit != nil {
		updates["assessment_type_allow_review_before_submit"] = *req.AssessmentTypeAllowReviewBeforeSubmit
	}
	if req.AssessmentTypeRequireCompleteAttempt != nil {
		updates["assessment_type_require_complete_attempt"] = *req.AssessmentTypeRequireCompleteAttempt
	}
	if req.AssessmentTypeShowDetailsAfterAllAttempts != nil {
		updates["assessment_type_show_details_after_all_attempts"] = *req.AssessmentTypeShowDetailsAfterAllAttempts
	}

	if len(updates) == 0 {
		// Tidak ada perubahan: balikin data existing saja
		return helper.JsonOK(c, "OK", mapToResponse(&existing))
	}
	updates["assessment_type_updated_at"] = time.Now()

	// Update assessment_type
	if err := ctl.DB.WithContext(c.Context()).
		Model(&assessmentModel.AssessmentTypeModel{}).
		Where("assessment_type_id = ? AND assessment_type_school_id = ?", id, schoolID).
		Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Key sudah dipakai untuk school ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Ambil data terbaru
	var after assessmentModel.AssessmentTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessment_type_id = ? AND assessment_type_school_id = ?", id, schoolID).
		First(&after).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ⬇ Sinkronkan seluruh snapshot scalar di assessments
	//    (graded, late_policy, passing_score, category)
	if err := assessSvc.New().SyncAssessmentTypeSnapshot(
		ctl.DB.WithContext(c.Context()),
		schoolID,
		&after,
	); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyinkronkan snapshot tipe pada assessment")
	}

	return helper.JsonUpdated(c, "Assessment type diperbarui", mapToResponse(&after))
}

// DELETE /assessment-types/:id — DKM/Admin
func (ctl *AssessmentTypeController) Delete(c *fiber.Ctx) error {
	// Pastikan helper slug→id bisa akses DB dari context (kalau ada yang butuh)
	c.Locals("DB", ctl.DB)

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "assessment_type_id tidak valid")
	}

	// 🔒 School context: STRICT dari token active_school
	schoolID, err := helperAuth.GetActiveSchoolID(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "School aktif di token tidak ditemukan")
	}

	// 🔒 DKM/Admin check
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	// 🔒 GUARD: cek apakah assessment type masih dipakai di assessments
	var usedCount int64
	if err := ctl.DB.WithContext(c.Context()).
		Model(&assessmentModel.AssessmentModel{}).
		Where(`
			assessment_school_id = ?
			AND assessment_type_id = ?
			AND assessment_deleted_at IS NULL
		`, schoolID, id).
		Count(&usedCount).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek relasi assessment")
	}

	if usedCount > 0 {
		return helper.JsonError(
			c,
			fiber.StatusBadRequest,
			"Tidak dapat menghapus tipe penilaian karena masih digunakan oleh beberapa assessment.",
		)
	}

	// Kalau sudah dipastikan tidak dipakai, baru ambil row dan hapus
	var row assessmentModel.AssessmentTypeModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("assessment_type_id = ? AND assessment_type_school_id = ?", id, schoolID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.WithContext(c.Context()).Delete(&row).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Assessment type dihapus", fiber.Map{
		"assessment_type_id": id,
	})
}
