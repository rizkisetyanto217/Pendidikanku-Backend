// file: internals/features/school/assessments/controller/assessment_controller.go
package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	dto "masjidku_backend/internals/features/school/submissions_assesment/assesments/dto"
	model "masjidku_backend/internals/features/school/submissions_assesment/assesments/model"
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
func (ctl *AssessmentController) assertTeacherBelongsToMasjid(masjidID uuid.UUID, teacherID *uuid.UUID) error {
	if teacherID == nil || *teacherID == uuid.Nil {
		return nil
	}
	var n int64
	// Sesuaikan kolom sesuai DDL masjid_teachers (umumnya PK: masjid_teacher_id)
	// dan kolom tenant: masjid_teachers_masjid_id
	err := ctl.DB.Table("masjid_teachers").
		Where("masjid_teacher_id = ? AND masjid_teachers_masjid_id = ? AND masjid_teachers_deleted_at IS NULL", *teacherID, masjidID).
		Count(&n).Error
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("assessments_created_by_teacher_id bukan milik masjid ini")
	}
	return nil
}

/* ========================================================
   Handlers
======================================================== */

// GET /assessments
// Query (opsional): type_id, csst_id, is_published, q, limit, offset, sort_by, sort_dir
func (ctl *AssessmentController) List(c *fiber.Ctx) error {
	masjidIDFromToken, _ := helperAuth.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, http.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var (
		typeIDStr = strings.TrimSpace(c.Query("type_id"))
		csstIDStr = strings.TrimSpace(c.Query("csst_id"))
		qStr      = strings.TrimSpace(c.Query("q"))
		isPubStr  = strings.TrimSpace(c.Query("is_published"))
		limit     = atoiOr(20, c.Query("limit"))
		offset    = atoiOr(0, c.Query("offset"))
		sortBy    = strings.TrimSpace(c.Query("sort_by"))
		sortDir   = strings.TrimSpace(c.Query("sort_dir"))
	)

	var typeID, csstID *uuid.UUID
	if typeIDStr != "" {
		if u, err := uuid.Parse(typeIDStr); err == nil {
			typeID = &u
		} else {
			return helper.Error(c, http.StatusBadRequest, "type_id tidak valid")
		}
	}
	if csstIDStr != "" {
		if u, err := uuid.Parse(csstIDStr); err == nil {
			csstID = &u
		} else {
			return helper.Error(c, http.StatusBadRequest, "csst_id tidak valid")
		}
	}

	var isPublished *bool
	if isPubStr != "" {
		b := strings.EqualFold(isPubStr, "true") || isPubStr == "1"
		isPublished = &b
	}

	var sbPtr, sdPtr *string
	if sortBy != "" {
		// Disarankan: title|created_at|start_at|due_at
		sbPtr = &sortBy
	}
	if sortDir != "" {
		// asc|desc
		sdPtr = &sortDir
	}

	qry := ctl.DB.Model(&model.AssessmentModel{}).
		Where("assessments_masjid_id = ?", masjidIDFromToken)

	if typeID != nil {
		qry = qry.Where("assessments_type_id = ?", *typeID)
	}
	if csstID != nil {
		qry = qry.Where("assessments_class_section_subject_teacher_id = ?", *csstID)
	}
	if isPublished != nil {
		qry = qry.Where("assessments_is_published = ?", *isPublished)
	}
	if qStr != "" {
		q := "%" + strings.ToLower(qStr) + "%"
		qry = qry.Where("(LOWER(assessments_title) LIKE ? OR LOWER(COALESCE(assessments_description, '')) LIKE ?)", q, q)
	}

	var total int64
	if err := qry.Count(&total).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	var rows []model.AssessmentModel
	if err := qry.
		Order(getSortClause(sbPtr, sdPtr)).
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	out := make([]dto.AssessmentResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}

	return helper.Success(c, "OK", dto.ListAssessmentResponse{
		Data:   out,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GET /assessments/:id
func (ctl *AssessmentController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.Error(c, http.StatusBadRequest, "assessments_id tidak valid")
	}

	masjidIDFromToken, _ := helperAuth.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, http.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var row model.AssessmentModel
	if err := ctl.DB.
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidIDFromToken).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	return helper.Success(c, "OK", toResponse(&row))
}

// POST /assessments
func (ctl *AssessmentController) Create(c *fiber.Ctx) error {
	var req dto.CreateAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, http.StatusBadRequest, "Payload tidak valid")
	}

	// Tenant override dari token
	masjidIDFromToken, _ := helperAuth.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, http.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	req.AssessmentsMasjidID = masjidIDFromToken

	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	// Validasi opsional: teacher creator harus milik masjid
	if err := ctl.assertTeacherBelongsToMasjid(req.AssessmentsMasjidID, req.AssessmentsCreatedByTeacherID); err != nil {
		return helper.Error(c, http.StatusBadRequest, err.Error())
	}

	now := time.Now()

	row := model.AssessmentModel{
		// Biarkan DB generate via DEFAULT gen_random_uuid()
		AssessmentsID:                           uuid.Nil,
		AssessmentsMasjidID:                     req.AssessmentsMasjidID,
		AssessmentsClassSectionSubjectTeacherID: req.AssessmentsClassSectionSubjectTeacherID,
		AssessmentsTypeID:                       req.AssessmentsTypeID,

		AssessmentsTitle:       strings.TrimSpace(req.AssessmentsTitle),
		AssessmentsDescription: nil,

		AssessmentsStartAt: req.AssessmentsStartAt,
		AssessmentsDueAt:   req.AssessmentsDueAt,

		// default DB 100, tapi kalau kamu ingin set manual:
		AssessmentsMaxScore: 100,

		AssessmentsIsPublished:     true,
		AssessmentsAllowSubmission: true,

		AssessmentsCreatedByTeacherID: req.AssessmentsCreatedByTeacherID,

		AssessmentsCreatedAt: now,
		AssessmentsUpdatedAt: now,
	}

	if req.AssessmentsDescription != nil {
		d := strings.TrimSpace(*req.AssessmentsDescription)
		row.AssessmentsDescription = &d
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

	if err := ctl.DB.Create(&row).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}
	return helper.SuccessWithCode(c, http.StatusCreated, "Assessment berhasil dibuat", toResponse(&row))
}

// PATCH /assessments/:id (partial)
func (ctl *AssessmentController) Patch(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.Error(c, http.StatusBadRequest, "assessments_id tidak valid")
	}

	var req dto.PatchAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, http.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	masjidIDFromToken, _ := helperAuth.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, http.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var existing model.AssessmentModel
	if err := ctl.DB.
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidIDFromToken).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	if req.AssessmentsCreatedByTeacherID != nil {
		if err := ctl.assertTeacherBelongsToMasjid(masjidIDFromToken, req.AssessmentsCreatedByTeacherID); err != nil {
			return helper.Error(c, http.StatusBadRequest, err.Error())
		}
	}

	updates := map[string]interface{}{}

	if req.AssessmentsTitle != nil {
		updates["assessments_title"] = strings.TrimSpace(*req.AssessmentsTitle)
	}
	if req.AssessmentsDescription != nil {
		updates["assessments_description"] = strings.TrimSpace(*req.AssessmentsDescription)
	}
	if req.AssessmentsStartAt != nil {
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
		return helper.Success(c, "Tidak ada perubahan", toResponse(&existing))
	}

	updates["assessments_updated_at"] = time.Now()

	if err := ctl.DB.Model(&model.AssessmentModel{}).
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidIDFromToken).
		Updates(updates).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	var after model.AssessmentModel
	if err := ctl.DB.
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidIDFromToken).
		First(&after).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	return helper.Success(c, "Assessment berhasil diperbarui", toResponse(&after))
}

// DELETE /assessments/:id (soft delete)
func (ctl *AssessmentController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.Error(c, http.StatusBadRequest, "assessments_id tidak valid")
	}

	masjidIDFromToken, _ := helperAuth.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, http.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var row model.AssessmentModel
	if err := ctl.DB.
		Where("assessments_id = ? AND assessments_masjid_id = ?", id, masjidIDFromToken).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	if err := ctl.DB.Delete(&row).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	return helper.SuccessWithCode(c, http.StatusNoContent, "Assessment dihapus", nil)
}
