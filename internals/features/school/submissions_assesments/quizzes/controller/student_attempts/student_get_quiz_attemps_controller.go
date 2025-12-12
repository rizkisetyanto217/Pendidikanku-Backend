package controller

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	qdto "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/dto"
	qmodel "madinahsalam_backend/internals/features/school/submissions_assesments/quizzes/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

// GET /student-quiz-attempts?quiz_id=&student_id=&status=&active_only=true&school_id=&all=1
func (ctl *StudentQuizAttemptsController) List(c *fiber.Ctx) error {
	quizIDStr := strings.TrimSpace(c.Query("quiz_id"))
	studentIDStr := strings.TrimSpace(c.Query("student_id"))
	statusStr := strings.TrimSpace(c.Query("status"))
	activeOnly := strings.EqualFold(strings.TrimSpace(c.Query("active_only")), "true")
	schoolIDStr := strings.TrimSpace(c.Query("school_id"))
	all := parseBool(c.Query("all"))

	q := ctl.DB.WithContext(c.Context()).Model(&qmodel.StudentQuizAttemptModel{})

	// ===== Role-based scoping =====
	if helperAuth.IsStudent(c) {
		// Student: lock ke school aktif + student_id sendiri
		mid, err := helperAuth.GetActiveSchoolID(c)
		if err != nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
		}
		if err := helperAuth.EnsureStudentSchool(c, mid); err != nil {
			return err
		}
		sid, err := helperAuth.GetSchoolStudentIDForSchool(c, mid)
		if err != nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
		}
		q = q.Where("student_quiz_attempt_school_id = ? AND student_quiz_attempt_student_id = ?", mid, sid)
	} else {
		// Admin/DKM/Teacher (Owner juga diizinkan)
		var mid uuid.UUID
		var err error
		if schoolIDStr != "" {
			mid, err = uuid.Parse(schoolIDStr)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "school_id tidak valid")
			}
			if err := helperAuth.EnsureDKMOrTeacherSchool(c, mid); err != nil && !helperAuth.IsOwner(c) {
				if fe, ok := err.(*fiber.Error); ok {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return err
			}
		} else {
			mid, err = helperAuth.GetSchoolIDFromTokenPreferTeacher(c)
			if err != nil {
				return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
			}
			if err := helperAuth.EnsureDKMOrTeacherSchool(c, mid); err != nil && !helperAuth.IsOwner(c) {
				if fe, ok := err.(*fiber.Error); ok {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return err
			}
		}
		q = q.Where("student_quiz_attempt_school_id = ?", mid)

		// teacher/dkm boleh filter student_id tertentu
		if studentIDStr != "" {
			studentID, err := uuid.Parse(studentIDStr)
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "student_id tidak valid")
			}
			q = q.Where("student_quiz_attempt_student_id = ?", studentID)
		}
	}

	// ===== Filters =====
	if quizIDStr != "" {
		quizID, err := uuid.Parse(quizIDStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "quiz_id tidak valid")
		}
		q = q.Where("student_quiz_attempt_quiz_id = ?", quizID)
	}

	if statusStr != "" {
		st := qmodel.StudentQuizAttemptStatus(statusStr)
		if !validAttemptStatus(st) {
			return helper.JsonError(c, fiber.StatusBadRequest, "status tidak valid (in_progress|submitted|finished|abandoned)")
		}
		q = q.Where("student_quiz_attempt_status = ?", st)
	}

	if activeOnly {
		q = q.Where("student_quiz_attempt_status IN (?)",
			[]string{
				string(qmodel.StudentQuizAttemptInProgress),
				string(qmodel.StudentQuizAttemptSubmitted),
			})
	}

	// ===== Count total (sebelum limit/offset) =====
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// ===== Paging & Sort =====
	pg := helper.ResolvePaging(c, 20, 100) // default per_page=20, max=100
	q = q.Order("student_quiz_attempt_started_at DESC")
	if !all {
		q = q.Offset(pg.Offset).Limit(pg.Limit)
	}

	// ===== Fetch =====
	var rows []qmodel.StudentQuizAttemptModel
	if err := q.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// ===== Build pagination object =====
	var pagination helper.Pagination
	if all {
		per := int(total)
		if per == 0 {
			per = 1
		}
		pagination = helper.BuildPaginationFromPage(total, 1, per)
	} else {
		pagination = helper.BuildPaginationFromPage(total, pg.Page, pg.PerPage)
	}

	// ===== JSON =====
	return helper.JsonList(c, "OK", qdto.FromModelsStudentQuizAttemptsWithCtx(c, rows), pagination)
}
