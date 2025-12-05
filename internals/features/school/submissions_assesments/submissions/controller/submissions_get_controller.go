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

	// 3) Base query: semua submission milik school ini
	tx := ctrl.DB.WithContext(c.Context()).
		Model(&model.SubmissionModel{}).
		Where(`
			submission_school_id = ?
			AND submission_deleted_at IS NULL
		`, schoolID)

	// 4) Role flags
	isStudent := helperAuth.IsStudent(c)
	isTeacher := helperAuth.IsTeacher(c)
	isDKM := helperAuth.IsDKM(c)

	// Student hanya boleh akses submission miliknya
	if isStudent && !isTeacher && !isDKM {
		if sid, _ := helperAuth.GetSchoolStudentIDForSchool(c, schoolID); sid != uuid.Nil {
			tx = tx.Where("submission_student_id = ?", sid)
		} else {
			// Student tapi tidak punya relasi school_student -> kosongkan list
			paging := helper.ResolvePaging(c, 20, 100)
			pagination := helper.BuildPaginationFromPage(0, paging.Page, paging.PerPage)
			return helper.JsonList(c, "OK", []any{}, pagination)
		}
	}

	// 5) Optional filters

	// 🔹 Filter by submission_id / id (single UUID)
	if s := strings.TrimSpace(c.Query("id")); s != "" {
		if sid, er := uuid.Parse(s); er == nil && sid != uuid.Nil {
			tx = tx.Where("submission_id = ?", sid)
		}
	} else if s := strings.TrimSpace(c.Query("submission_id")); s != "" {
		if sid, er := uuid.Parse(s); er == nil && sid != uuid.Nil {
			tx = tx.Where("submission_id = ?", sid)
		}
	}

	// 🔹 Filter by assessment_id (single UUID)
	if s := strings.TrimSpace(c.Query("assessment_id")); s != "" {
		if aid, er := uuid.Parse(s); er == nil && aid != uuid.Nil {
			tx = tx.Where("submission_assessment_id = ?", aid)
		}
	}

	// 🔹 (opsional) multiple assessment_ids=uuid1,uuid2,...
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

	// ?student_id=<uuid> (hanya untuk non-student / staff)
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

	var rows []model.SubmissionModel
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
