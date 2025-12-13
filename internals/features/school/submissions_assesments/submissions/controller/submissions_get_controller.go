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

/* =========================
   include parser
========================= */

func hasInclude(c *fiber.Ctx, key string) bool {
	raw := strings.ToLower(strings.TrimSpace(c.Query("include")))
	if raw == "" {
		return false
	}
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		if strings.TrimSpace(p) == key {
			return true
		}
	}
	return false
}

// GET /submissions/list (LIST — member; student hanya lihat miliknya, school via token)
func (ctrl *SubmissionController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// 1) Ambil school dari token
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

	// SchoolID selalu di-force dari token
	q.SchoolID = &schoolID

	// include flags
	includeURLs := hasInclude(c, "submission_urls")

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

	// 6) Optional filters dari query

	// Filter by submission_id / id (single UUID)
	if s := strings.TrimSpace(c.Query("id")); s != "" {
		if sid, er := uuid.Parse(s); er == nil && sid != uuid.Nil {
			tx = tx.Where("submission_id = ?", sid)
		}
	} else if s := strings.TrimSpace(c.Query("submission_id")); s != "" {
		if sid, er := uuid.Parse(s); er == nil && sid != uuid.Nil {
			tx = tx.Where("submission_id = ?", sid)
		}
	}

	// Filter by assessment_id
	if q.AssessmentID != nil {
		tx = tx.Where("submission_assessment_id = ?", *q.AssessmentID)
	}

	// multiple assessment_ids=uuid1,uuid2,...
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

	// StudentID (staff only)
	if !isStudent && q.StudentID != nil {
		tx = tx.Where("submission_student_id = ?", *q.StudentID)
	}

	// Status
	if q.Status != nil {
		tx = tx.Where("submission_status = ?", *q.Status)
	}

	// Periode submitted_from / submitted_to
	if q.SubmittedFrom != nil {
		tx = tx.Where("submission_submitted_at >= ?", *q.SubmittedFrom)
	}
	if q.SubmittedTo != nil {
		tx = tx.Where("submission_submitted_at < ?", *q.SubmittedTo)
	}

	// 7) Pagination
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

	// 8) Total count
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 9) Sorting
	tx = applySort(tx, q.Sort)

	// 10) Ambil data
	var rows []model.SubmissionModel
	if err := tx.
		Offset(offset).
		Limit(perPage).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 11) Optional include=submission_urls (compact, batch)
	urlBySubmission := map[uuid.UUID][]dto.SubmissionURLDocCompact{}
	if includeURLs && len(rows) > 0 {
		ids := make([]uuid.UUID, 0, len(rows))
		for i := range rows {
			ids = append(ids, rows[i].SubmissionID)
		}

		var urls []model.SubmissionURLModel
		if err := ctrl.DB.WithContext(c.Context()).
			Model(&model.SubmissionURLModel{}).
			Where(`
				submission_url_school_id = ?
				AND submission_url_submission_id IN ?
				AND submission_url_deleted_at IS NULL
			`, schoolID, ids).
			Order("submission_url_is_primary DESC, submission_url_order ASC, submission_url_created_at ASC").
			Find(&urls).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}

		for i := range urls {
			sid := urls[i].SubmissionURLSubmissionID
			urlBySubmission[sid] = append(urlBySubmission[sid], dto.FromModelSubmissionURLDocCompact(urls[i]))
		}
	}

	// 12) Mapping ke DTO (timezone-aware) + inject include
	items := make([]any, 0, len(rows))
	for i := range rows {
		resp := dto.FromModelWithCtx(c, &rows[i])
		if includeURLs {
			resp.SubmissionURLs = urlBySubmission[rows[i].SubmissionID]
		}
		items = append(items, resp)
	}

	// 13) Pagination meta
	pagination := helper.BuildPaginationFromPage(total, page, perPage)

	return helper.JsonList(c, "OK", items, pagination)
}
