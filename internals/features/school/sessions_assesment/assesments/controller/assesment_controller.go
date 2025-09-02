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

	dto "masjidku_backend/internals/features/school/sessions_assesment/assesments/dto"
	model "masjidku_backend/internals/features/school/sessions_assesment/assesments/model"
	helper "masjidku_backend/internals/helpers"
)

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

/* =========================
   Helpers
   ========================= */

func toResponse(m *model.AssessmentModel) dto.AssessmentResponse {
	return dto.AssessmentResponse{
		ID:                             m.ID,
		MasjidID:                       m.MasjidID,
		ClassSectionID:                 m.ClassSectionID,
		ClassSubjectsID:                m.ClassSubjectsID,
		ClassSectionSubjectTeacherID:   m.ClassSectionSubjectTeacherID,
		TypeID:                         m.TypeID,
		Title:                          m.Title,
		Description:                    m.Description,
		StartAt:                        m.StartAt,
		DueAt:                          m.DueAt,
		MaxScore:                       m.MaxScore,
		IsPublished:                    m.IsPublished,
		AllowSubmission:                m.AllowSubmission,
		CreatedByTeacherID:             m.CreatedByTeacherID,
		CreatedAt:                      m.CreatedAt,
		UpdatedAt:                      m.UpdatedAt,
	}
}


// Optional: pastikan created_by_teacher_id (jika ada) memang milik masjid
func (ctl *AssessmentController) assertTeacherBelongsToMasjid(masjidID uuid.UUID, teacherID *uuid.UUID) error {
	if teacherID == nil || *teacherID == uuid.Nil {
		return nil
	}
	var n int64
	err := ctl.DB.Table("masjid_teachers").
		Where("masjid_teachers_id = ? AND masjid_teacher_masjid_id = ? AND masjid_teachers_deleted_at IS NULL", *teacherID, masjidID).
		Count(&n).Error
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("created_by_teacher_id bukan milik masjid ini")
	}
	return nil
}

/* =========================
   Handlers
   ========================= */

// GET /assessments
// Query (opsional): type_id, section_id, subject_id, csst_id, is_published, q, limit, offset, sort_by, sort_dir
func (ctl *AssessmentController) List(c *fiber.Ctx) error {
	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, http.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}

	var (
		typeIDStr    = strings.TrimSpace(c.Query("type_id"))
		sectionIDStr = strings.TrimSpace(c.Query("section_id"))
		subjectIDStr = strings.TrimSpace(c.Query("subject_id"))
		csstIDStr    = strings.TrimSpace(c.Query("csst_id"))
		qStr         = strings.TrimSpace(c.Query("q"))
		isPubStr     = strings.TrimSpace(c.Query("is_published"))
		limit        = atoiOr(20, c.Query("limit"))
		offset       = atoiOr(0, c.Query("offset"))
		sortBy       = strings.TrimSpace(c.Query("sort_by"))
		sortDir      = strings.TrimSpace(c.Query("sort_dir"))
	)

	var typeID, sectionID, subjectID, csstID *uuid.UUID
	if typeIDStr != "" {
		if u, err := uuid.Parse(typeIDStr); err == nil {
			typeID = &u
		} else {
			return helper.Error(c, http.StatusBadRequest, "type_id tidak valid")
		}
	}
	if sectionIDStr != "" {
		if u, err := uuid.Parse(sectionIDStr); err == nil {
			sectionID = &u
		} else {
			return helper.Error(c, http.StatusBadRequest, "section_id tidak valid")
		}
	}
	if subjectIDStr != "" {
		if u, err := uuid.Parse(subjectIDStr); err == nil {
			subjectID = &u
		} else {
			return helper.Error(c, http.StatusBadRequest, "subject_id tidak valid")
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
		sbPtr = &sortBy // title|created_at|start_at|due_at
	}
	if sortDir != "" {
		sdPtr = &sortDir // asc|desc
	}

	qry := ctl.DB.Model(&model.AssessmentModel{}).
		Where("assessments_masjid_id = ?", masjidIDFromToken)

	if typeID != nil {
		qry = qry.Where("assessments_type_id = ?", *typeID)
	}
	if sectionID != nil {
		qry = qry.Where("assessments_class_section_id = ?", *sectionID)
	}
	if subjectID != nil {
		qry = qry.Where("assessments_class_subjects_id = ?", *subjectID)
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
		return helper.Error(c, http.StatusBadRequest, "ID tidak valid")
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
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
	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
	if masjidIDFromToken == uuid.Nil {
		return helper.Error(c, http.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	req.MasjidID = masjidIDFromToken

	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	// Validasi opsional: teacher creator harus milik masjid
	if err := ctl.assertTeacherBelongsToMasjid(req.MasjidID, req.CreatedByTeacherID); err != nil {
		return helper.Error(c, http.StatusBadRequest, err.Error())
	}

	now := time.Now()
	row := model.AssessmentModel{
		ID:                           uuid.Nil, // biarkan DB generate
		MasjidID:                     req.MasjidID,
		ClassSectionID:               req.ClassSectionID,
		ClassSubjectsID:              req.ClassSubjectsID,
		ClassSectionSubjectTeacherID: req.ClassSectionSubjectTeacherID,
		TypeID:           req.TypeID,
		Title:            strings.TrimSpace(req.Title),
		Description:      nil,
		StartAt:          req.StartAt,
		DueAt:            req.DueAt,
		MaxScore:         100,
		IsPublished:      true,
		AllowSubmission:  true,
		CreatedByTeacherID: req.CreatedByTeacherID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if req.Description != nil {
		d := strings.TrimSpace(*req.Description)
		row.Description = &d
	}
	if req.MaxScore >= 0 { // validator sudah cek 0..100
		row.MaxScore = req.MaxScore
	}
	if req.IsPublished != nil {
		row.IsPublished = *req.IsPublished
	}
	if req.AllowSubmission != nil {
		row.AllowSubmission = *req.AllowSubmission
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
		return helper.Error(c, http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.PatchAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, http.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return helper.ValidationError(c, err)
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
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

	if req.CreatedByTeacherID != nil {
		if err := ctl.assertTeacherBelongsToMasjid(masjidIDFromToken, req.CreatedByTeacherID); err != nil {
			return helper.Error(c, http.StatusBadRequest, err.Error())
		}
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["assessments_title"] = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		updates["assessments_description"] = strings.TrimSpace(*req.Description)
	}
	if req.StartAt != nil {
		updates["assessments_start_at"] = *req.StartAt
	}
	if req.DueAt != nil {
		updates["assessments_due_at"] = *req.DueAt
	}
	if req.MaxScore != nil {
		updates["assessments_max_score"] = *req.MaxScore
	}
	if req.IsPublished != nil {
		updates["assessments_is_published"] = *req.IsPublished
	}
	if req.AllowSubmission != nil {
		updates["assessments_allow_submission"] = *req.AllowSubmission
	}
	if req.TypeID != nil {
		updates["assessments_type_id"] = *req.TypeID
	}
	if req.ClassSectionID != nil {
		updates["assessments_class_section_id"] = *req.ClassSectionID
	}
	if req.ClassSubjectsID != nil {
		updates["assessments_class_subjects_id"] = *req.ClassSubjectsID
	}
	if req.ClassSectionSubjectTeacherID != nil {
		updates["assessments_class_section_subject_teacher_id"] = *req.ClassSectionSubjectTeacherID
	}
	if req.CreatedByTeacherID != nil {
		updates["assessments_created_by_teacher_id"] = *req.CreatedByTeacherID
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
	if err := ctl.DB.Where("assessments_id = ?", id).First(&after).Error; err != nil {
		return helper.Error(c, http.StatusInternalServerError, err.Error())
	}

	return helper.Success(c, "Assessment berhasil diperbarui", toResponse(&after))
}

// DELETE /assessments/:id (soft)
func (ctl *AssessmentController) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.Error(c, http.StatusBadRequest, "ID tidak valid")
	}

	masjidIDFromToken, _ := helper.GetMasjidIDFromToken(c)
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
