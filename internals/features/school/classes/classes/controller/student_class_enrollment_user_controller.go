// file: internals/features/school/classes/class_enrollments/controller/list.go
package controller

import (
	"strings"

	dto "schoolku_backend/internals/features/school/classes/classes/dto"
	emodel "schoolku_backend/internals/features/school/classes/classes/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/*
GET /:school_id/class-enrollments
?status_in=a,b,c
?applied_from=...
?applied_to=...
?only_alive=true|false
?order_by=created_at|applied_at|updated_at
?sort=asc|desc
?limit=...
?offset=...
?view=compact|full   <-- NEW (default: full)
*/
func (ctl *StudentClassEnrollmentController) List(c *fiber.Ctx) error {
	// ========== tenant ==========
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}
	if er := helperAuth.EnsureMemberSchool(c, schoolID); er != nil {
		return er
	}

	// ========== query ==========
	var q dto.ListStudentClassEnrollmentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query")
	}
	// status_in (comma-separated → slice)
	if raw := strings.TrimSpace(c.Query("status_in")); raw != "" {
		sts, er := parseStatusInParam(raw)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
		}
		q.StatusIn = sts
	}
	// view mode
	view := strings.ToLower(strings.TrimSpace(c.Query("view"))) // "", "compact", "full"

	// paging
	pg := helper.ResolvePaging(c, 20, 200)

	// ========== base query ==========
	base := ctl.DB.WithContext(c.Context()).
		Model(&emodel.StudentClassEnrollmentModel{}).
		Where("student_class_enrollments_school_id = ?", schoolID)

	// OnlyAlive default: true (filter soft-delete)
	onlyAlive := true
	if q.OnlyAlive != nil {
		onlyAlive = *q.OnlyAlive
	}
	if onlyAlive {
		base = base.Where("student_class_enrollments_deleted_at IS NULL")
	}

	// ========== filters ==========
	if q.StudentID != nil && *q.StudentID != uuid.Nil {
		base = base.Where("student_class_enrollments_school_student_id = ?", *q.StudentID)
	}
	if q.ClassID != nil && *q.ClassID != uuid.Nil {
		base = base.Where("student_class_enrollments_class_id = ?", *q.ClassID)
	}
	if len(q.StatusIn) > 0 {
		base = base.Where("student_class_enrollments_status IN ?", q.StatusIn)
	}
	if q.AppliedFrom != nil {
		base = base.Where("student_class_enrollments_applied_at >= ?", *q.AppliedFrom)
	}
	if q.AppliedTo != nil {
		base = base.Where("student_class_enrollments_applied_at <= ?", *q.AppliedTo)
	}

	// ========== count ==========
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to count")
	}

	// ========== data ==========
	tx := base // copy builder for data fetch

	// optimisasi kolom saat compact
	if view == "compact" || view == "summary" {
		tx = tx.Select([]string{
			"student_class_enrollments_id",
			"student_class_enrollments_status",
			"student_class_enrollments_total_due_idr",
			"student_class_enrollments_payment_snapshot",
			"student_class_enrollments_class_name_snapshot",
			"student_class_enrollments_student_name_snapshot",
			"student_class_enrollments_applied_at",
		})
	}

	var rows []emodel.StudentClassEnrollmentModel
	if err := tx.
		Order(orderClause(q.OrderBy, q.Sort)).
		Offset(pg.Offset).
		Limit(pg.Limit).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch")
	}

	pagination := helper.BuildPaginationFromOffset(total, pg.Offset, pg.Limit)

	// ========== mapping sesuai view ==========
	if view == "compact" || view == "summary" {
		compact := dto.FromModelsCompact(rows)
		return helper.JsonList(c, "ok", compact, pagination)
	}

	// default: full payload
	resp := dto.FromModels(rows)
	// (opsional) enrich convenience fields tambahan (Username, dsb.)
	enrichEnrollmentExtras(c.Context(), ctl.DB, schoolID, resp)

	return helper.JsonList(c, "ok", resp, pagination)
}
