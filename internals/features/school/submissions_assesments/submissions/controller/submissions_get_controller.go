// file: internals/features/school/attendance_assesment/submissions/controller/submission_list_controller.go
package controller

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	dto "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

// GET /submissions/list (LIST — member; student hanya lihat miliknya, school via token)
func (ctrl *SubmissionController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// 1) Ambil school dari token (via helper yang sudah ada: parseSchoolIDParam -> GetActiveSchoolID)
	schoolID, err := parseSchoolIDParam(c)
	if err != nil {
		return err
	}

	// 2) Authorize minimal member school
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return err
	}

	// 3) Parse query ke DTO ListSubmissionsQuery
	var q dto.ListSubmissionsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "query params tidak valid")
	}
	if ctrl.Validator != nil {
		if err := ctrl.Validator.Struct(&q); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// SchoolID selalu di-force dari token (abaikan query school_id)
	q.SchoolID = &schoolID

	// 4) Base query: semua submission milik school ini
	tx := ctrl.DB.WithContext(c.Context()).
		Model(&model.SubmissionModel{}).
		Where(`
			submission_school_id = ?
			AND submission_deleted_at IS NULL
		`, schoolID)

	// 5) Role flags
	isStudent := helperAuth.IsStudent(c)
	isTeacher := helperAuth.IsTeacher(c)
	isDKM := helperAuth.IsDKM(c)

	// Student hanya boleh akses submission miliknya
	if isStudent && !isTeacher && !isDKM {
		if sid, _ := helperAuth.GetSchoolStudentIDForSchool(c, schoolID); sid != uuid.Nil {
			q.StudentID = &sid
		} else {
			// Student tapi tidak punya relasi school_student -> kosongkan list
			page := q.Page
			if page <= 0 {
				page = 1
			}
			perPage := q.PerPage
			if perPage <= 0 {
				perPage = 20
			} else if perPage > 200 {
				perPage = 200
			}
			pagination := helper.BuildPaginationFromPage(0, page, perPage)
			return helper.JsonList(c, "OK", []any{}, pagination)
		}
	}

	// 6) Optional filters dari query (DTO)

	// 🔹 Filter by submission_id / id (single UUID) — tambahan di luar DTO
	if s := strings.TrimSpace(c.Query("id")); s != "" {
		if sid, er := uuid.Parse(s); er == nil && sid != uuid.Nil {
			tx = tx.Where("submission_id = ?", sid)
		}
	} else if s := strings.TrimSpace(c.Query("submission_id")); s != "" {
		if sid, er := uuid.Parse(s); er == nil && sid != uuid.Nil {
			tx = tx.Where("submission_id = ?", sid)
		}
	}

	// 🔹 Filter by assessment_id (DTO)
	if q.AssessmentID != nil {
		tx = tx.Where("submission_assessment_id = ?", *q.AssessmentID)
	}

	// 🔹 (opsional) multiple assessment_ids=uuid1,uuid2,... (tambahan)
	if s := strings.TrimSpace(c.Query("assessment_ids")); s != "" {
		parts := strings.Split(s, ",")
		var ids []uuid.UUID
		for _, p := range parts {
			ps := strings.TrimSpace(p)
			if ps == "" {
				continue
			}
			if aid, er := uuid.Parse(ps); er == nil && aid != uuid.Nil {
				ids = append(ids, aid)
			}
		}
		if len(ids) > 0 {
			tx = tx.Where("submission_assessment_id IN ?", ids)
		}
	}

	// 🔹 StudentID (hanya untuk non-student / staff; untuk student sudah di-force di atas)
	if !isStudent && q.StudentID != nil {
		tx = tx.Where("submission_student_id = ?", *q.StudentID)
	}

	// 🔹 Status
	if q.Status != nil {
		tx = tx.Where("submission_status = ?", *q.Status)
	}

	// 🔹 Periode submitted_from / submitted_to
	if q.SubmittedFrom != nil {
		tx = tx.Where("submission_submitted_at >= ?", *q.SubmittedFrom)
	}
	if q.SubmittedTo != nil {
		tx = tx.Where("submission_submitted_at < ?", *q.SubmittedTo)
	}

	// 7) Pagination (pakai Page/PerPage dari DTO dengan clamp)
	page := q.Page
	if page <= 0 {
		page = 1
	}
	perPage := q.PerPage
	if perPage <= 0 {
		perPage = 20
	} else if perPage > 200 {
		perPage = 200
	}
	offset := (page - 1) * perPage

	// 8) Hitung total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 9) Sorting (pakai helper applySort yang sudah didefinisikan di controller lain)
	tx = applySort(tx, q.Sort)

	// 10) Ambil data
	var rows []model.SubmissionModel
	if err := tx.
		Offset(offset).
		Limit(perPage).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 11) Mapping ke DTO (timezone-aware)
	items := make([]any, 0, len(rows))
	for i := range rows {
		items = append(items, dto.FromModelWithCtx(c, &rows[i]))
	}

	// 12) Build pagination full (TotalPages, HasNext, HasPrev, dsb)
	pagination := helper.BuildPaginationFromPage(total, page, perPage)

	return helper.JsonList(c, "OK", items, pagination)
}
