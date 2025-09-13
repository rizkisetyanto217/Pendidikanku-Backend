// file: internals/features/school/attendance_assesment/submissions/controller/submission_controller.go
package controller

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/submissions/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

type SubmissionController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewSubmissionController(db *gorm.DB) *SubmissionController {
	return &SubmissionController{
		DB:        db,
		Validator: validator.New(),
	}
}

func clampPage(n int) int {
	if n <= 0 { return 1 }
	return n
}
func clampPerPage(n int) int {
	if n <= 0 { return 20 }
	if n > 200 { return 200 }
	return n
}

func applyFilters(q *gorm.DB, f *dto.ListSubmissionsQuery) *gorm.DB {
	if f == nil { return q }
	if f.MasjidID != nil {
		q = q.Where("submissions_masjid_id = ?", *f.MasjidID)
	}
	if f.AssessmentID != nil {
		q = q.Where("submissions_assessment_id = ?", *f.AssessmentID)
	}
	if f.StudentID != nil {
		q = q.Where("submissions_student_id = ?", *f.StudentID)
	}
	if f.Status != nil {
		q = q.Where("submissions_status = ?", *f.Status)
	}
	if f.SubmittedFrom != nil {
		q = q.Where("submissions_submitted_at >= ?", *f.SubmittedFrom)
	}
	if f.SubmittedTo != nil {
		q = q.Where("submissions_submitted_at < ?", *f.SubmittedTo)
	}
	return q
}

func applySort(q *gorm.DB, sort string) *gorm.DB {
	switch strings.TrimSpace(sort) {
	case "created_at":
		return q.Order("submissions_created_at ASC")
	case "desc_created_at", "":
		return q.Order("submissions_created_at DESC")
	case "submitted_at":
		return q.Order("submissions_submitted_at ASC NULLS LAST")
	case "desc_submitted_at":
		return q.Order("submissions_submitted_at DESC NULLS LAST")
	case "score":
		return q.Order("submissions_score ASC NULLS LAST")
	case "desc_score":
		return q.Order("submissions_score DESC NULLS LAST")
	default:
		return q.Order("submissions_created_at DESC")
	}
}

/* =========================
   Handlers
========================= */

// POST / (STUDENT ONLY)
func (ctrl *SubmissionController) Create(c *fiber.Ctx) error {
	var body dto.CreateSubmissionRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Ambil masjid aktif & pastikan caller adalah STUDENT di masjid tsb
	mid, err := helperAuth.GetActiveMasjidIDFromToken(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid aktif tidak ditemukan di token")
	}
	if err := helperAuth.EnsureStudentMasjid(c, mid); err != nil {
		return err
	}

	// Ambil student_id milik caller pada masjid tsb
	sid, err := helperAuth.GetMasjidStudentIDForMasjid(c, mid)
	if err != nil || sid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya siswa terdaftar yang diizinkan membuat submission")
	}

	// Paksa tenant & identitas student dari token
	body.SubmissionMasjidID = mid
	body.SubmissionStudentID = sid

	// Validasi payload setelah override
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Default status
	status := model.SubmissionStatusSubmitted
	if body.SubmissionStatus != nil {
		status = *body.SubmissionStatus
	}

	sub := &model.Submission{
		SubmissionMasjidID:     body.SubmissionMasjidID,
		SubmissionAssessmentID: body.SubmissionAssessmentID,
		SubmissionStudentID:    body.SubmissionStudentID,
		SubmissionText:         body.SubmissionText,
		SubmissionStatus:       status,
		SubmissionSubmittedAt:  body.SubmissionSubmittedAt,
		SubmissionIsLate:       body.SubmissionIsLate,
	}

	// Auto submitted_at bila perlu
	if (sub.SubmissionStatus == model.SubmissionStatusSubmitted || sub.SubmissionStatus == model.SubmissionStatusResubmitted) &&
		sub.SubmissionSubmittedAt == nil {
		now := time.Now()
		sub.SubmissionSubmittedAt = &now
	}

	if err := ctrl.DB.WithContext(c.Context()).Create(sub).Error; err != nil {
		le := strings.ToLower(err.Error())
		if strings.Contains(le, "duplicate key") || strings.Contains(le, "unique constraint") {
			return helper.JsonError(c, fiber.StatusConflict, "Submission untuk assessment & student ini sudah ada")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonCreated(c, "Submission berhasil dibuat", dto.FromModel(sub))
}



// PATCH /:id (WRITE — DKM/Teacher/Admin)
func (ctrl *SubmissionController) Patch(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, mid); err != nil {
		return err
	}

	var sub model.Submission
	if err := ctrl.DB.WithContext(c.Context()).
		First(&sub, "submissions_id = ? AND submissions_masjid_id = ?", id, mid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Submission tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var body dto.PatchSubmissionRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Validasi enum / range
	if body.SubmissionStatus != nil && body.SubmissionStatus.ShouldUpdate() && !body.SubmissionStatus.IsNull() {
		switch *body.SubmissionStatus.Value {
		case model.SubmissionStatusDraft, model.SubmissionStatusSubmitted, model.SubmissionStatusResubmitted,
			model.SubmissionStatusGraded, model.SubmissionStatusReturned:
		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "submissions_status invalid")
		}
	}
	if body.SubmissionScore != nil && body.SubmissionScore.ShouldUpdate() && !body.SubmissionScore.IsNull() {
		if *body.SubmissionScore.Value < 0 || *body.SubmissionScore.Value > 100 {
			return helper.JsonError(c, fiber.StatusBadRequest, "submissions_score harus 0..100")
		}
	}

	updates := body.ToUpdates()

	// Auto submitted_at jika berubah ke submitted/resubmitted
	if v, ok := updates["submissions_status"]; ok {
		if st, ok2 := v.(model.SubmissionStatus); ok2 {
			if (st == model.SubmissionStatusSubmitted || st == model.SubmissionStatusResubmitted) &&
				updates["submissions_submitted_at"] == nil && sub.SubmissionSubmittedAt == nil {
				updates["submissions_submitted_at"] = time.Now()
			}
		}
	}

	if len(updates) > 0 {
		if err := ctrl.DB.WithContext(c.Context()).Model(&sub).Updates(updates).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	if err := ctrl.DB.WithContext(c.Context()).
		First(&sub, "submissions_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Submission diperbarui", dto.FromModel(&sub))
}

// PATCH /:id/grade (WRITE — DKM/Teacher/Admin)
func (ctrl *SubmissionController) Grade(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, mid); err != nil {
		return err
	}

	var sub model.Submission
	if err := ctrl.DB.WithContext(c.Context()).
		First(&sub, "submissions_id = ? AND submissions_masjid_id = ?", id, mid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Submission tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var body dto.GradeSubmissionRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctrl.Validator.Struct(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}
	if body.SubmissionScore != nil && body.SubmissionScore.ShouldUpdate() && !body.SubmissionScore.IsNull() {
		if *body.SubmissionScore.Value < 0 || *body.SubmissionScore.Value > 100 {
			return helper.JsonError(c, fiber.StatusBadRequest, "submissions_score harus 0..100")
		}
	}

	updates := body.ToUpdates()

	// Auto status graded saat ada field grading
	if _, ok := updates["submissions_score"]; ok {
		updates["submissions_status"] = model.SubmissionStatusGraded
	} else if _, ok := updates["submissions_feedback"]; ok {
		updates["submissions_status"] = model.SubmissionStatusGraded
	} else if _, ok := updates["submissions_graded_by_teacher_id"]; ok {
		updates["submissions_status"] = model.SubmissionStatusGraded
	} else if _, ok := updates["submissions_graded_at"]; ok {
		updates["submissions_status"] = model.SubmissionStatusGraded
	}

	// graded_at default now bila ada perubahan grading tanpa graded_at
	if updates["submissions_graded_at"] == nil &&
		(updates["submissions_score"] != nil || updates["submissions_feedback"] != nil || updates["submissions_graded_by_teacher_id"] != nil) {
		updates["submissions_graded_at"] = time.Now()
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "OK", dto.FromModel(&sub))
	}

	if err := ctrl.DB.WithContext(c.Context()).Model(&sub).Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctrl.DB.WithContext(c.Context()).
		First(&sub, "submissions_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonUpdated(c, "Submission dinilai", dto.FromModel(&sub))
}

// DELETE /:id (WRITE — DKM/Teacher/Admin)
func (ctrl *SubmissionController) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	mid, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || mid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, mid); err != nil {
		return err
	}

	var sub model.Submission
	if err := ctrl.DB.WithContext(c.Context()).
		Select("submissions_id").
		First(&sub, "submissions_id = ? AND submissions_masjid_id = ?", id, mid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Submission tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	if err := ctrl.DB.WithContext(c.Context()).Delete(&sub).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Submission dihapus", fiber.Map{
		"submissions_id": id,
	})
}