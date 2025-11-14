package controller

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	dto "schoolku_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "schoolku_backend/internals/features/school/submissions_assesments/submissions/model"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
)

// GET /submissions/list (LIST — member; student hanya lihat miliknya)
func (ctrl *SubmissionController) List(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// 1) Resolve school context (slug/id)
	mc, err := helperAuth.ResolveSchoolContext(c)
	if err != nil {
		return err
	}

	var mid uuid.UUID
	if mc.ID != uuid.Nil {
		mid = mc.ID
	} else if s := strings.TrimSpace(mc.Slug); s != "" {
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return fiber.NewError(fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		mid = id
	} else {
		return helperAuth.ErrSchoolContextMissing
	}

	// 2) Authorize minimal member school
	if err := helperAuth.EnsureMemberSchool(c, mid); err != nil {
		return err
	}

	// 3) Base query: semua submission milik school ini
	tx := ctrl.DB.WithContext(c.Context()).
		Model(&model.Submission{}).
		Where(`
			submission_school_id = ?
			AND submission_deleted_at IS NULL
		`, mid)

	// 4) Role flags
	isStudent := helperAuth.IsStudent(c)
	isTeacher := helperAuth.IsTeacher(c)
	isDKM := helperAuth.IsDKM(c)

	// Student hanya boleh akses submission miliknya
	if isStudent && !isTeacher && !isDKM {
		if sid, _ := helperAuth.GetSchoolStudentIDForSchool(c, mid); sid != uuid.Nil {
			tx = tx.Where("submission_student_id = ?", sid)
		} else {
			// Student tapi tidak punya relasi school_student -> kosongkan list
			paging := helper.ResolvePaging(c, 20, 100)
			pagination := helper.BuildPaginationFromPage(0, paging.Page, paging.PerPage)
			return helper.JsonList(c, "OK", []any{}, pagination)
		}
	}

	// 5) Optional filters
	// ?assessment_id=<uuid>
	if s := strings.TrimSpace(c.Query("assessment_id")); s != "" {
		if aid, er := uuid.Parse(s); er == nil && aid != uuid.Nil {
			tx = tx.Where("submission_assessment_id = ?", aid)
		}
	}

	// ?student_id=<uuid> (hanya untuk DKM/Teacher)
	if !isStudent {
		if s := strings.TrimSpace(c.Query("student_id")); s != "" {
			if sid, er := uuid.Parse(s); er == nil && sid != uuid.Nil {
				tx = tx.Where("submission_student_id = ?", sid)
			}
		}
	}

	// 6) Pagination (pakai helper)
	paging := helper.ResolvePaging(c, 20, 100)

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.Submission
	if err := tx.
		Order("submission_created_at DESC").
		Offset(paging.Offset).
		Limit(paging.Limit).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// 7) Mapping ke DTO
	items := make([]any, 0, len(rows))
	for i := range rows {
		items = append(items, dto.FromModel(&rows[i]))
	}

	// 8) Build pagination full (TotalPages, HasNext, HasPrev, dsb)
	pagination := helper.BuildPaginationFromPage(total, paging.Page, paging.PerPage)

	return helper.JsonList(c, "OK", items, pagination)
}
