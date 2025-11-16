// file: internals/features/school/classes/class_enrollments/controller/list.go
package controller

import (
	"strings"

	dto "schoolku_backend/internals/features/school/classes/classes/dto"
	emodel "schoolku_backend/internals/features/school/classes/classes/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
)

// Skenario 2: endpoint khusus murid → hanya melihat enrollments miliknya sendiri
//
// GET /api/u/:school_id/my/class-enrollments
// ?status_in=...
// ?applied_from=...
// ?applied_to=...
// ?order_by=...
// ?sort=...
// ?limit=...
// ?offset=...
// ?view=compact|full
func (ctl *StudentClassEnrollmentController) ListMy(c *fiber.Ctx) error {
	// ========== tenant ==========
	schoolID, err := helperAuth.ParseSchoolIDFromPath(c)
	if err != nil {
		return err
	}

	// Hanya murid dari school ini yang diizinkan
	if err := helperAuth.EnsureStudentSchool(c, schoolID); err != nil {
		return err
	}

	// Ambil daftar student_id dari token / context
	studentIDs, err := helperAuth.GetSchoolStudentIDsFromToken(c)
	if err != nil || len(studentIDs) == 0 {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Konteks murid tidak ditemukan")
	}

	// Untuk endpoint "my", kita pakai satu ID (mis. yang pertama)
	studentID := studentIDs[0]

	// ========== query ==========
	var q dto.ListStudentClassEnrollmentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query")
	}

	// FORCE: hanya enrollment milik murid ini
	q.StudentID = &studentID

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
		Where("student_class_enrollments_school_id = ?", schoolID).
		Where("student_class_enrollments_school_student_id = ?", studentID).
		Where("student_class_enrollments_deleted_at IS NULL") // untuk murid, biasanya hanya alive

	// ========== filters tambahan ==========
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
	tx := base

	// murid juga bisa pakai compact view biar ringan
	if view == "compact" || view == "summary" {
		tx = tx.Select([]string{
			"student_class_enrollments_id",
			"student_class_enrollments_status",
			"student_class_enrollments_total_due_idr",

			"student_class_enrollments_school_student_id",
			"student_class_enrollments_student_name_snapshot",
			"student_class_enrollments_class_id",
			"student_class_enrollments_class_name_snapshot",

			"student_class_enrollments_term_id",
			"student_class_enrollments_term_name_snapshot",
			"student_class_enrollments_term_academic_year_snapshot",
			"student_class_enrollments_term_angkatan_snapshot",

			"student_class_enrollments_payment_snapshot",
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

	if view == "compact" || view == "summary" {
		compact := dto.FromModelsCompact(rows)
		return helper.JsonList(c, "ok", compact, pagination)
	}

	resp := dto.FromModels(rows)
	// Untuk murid biasanya nggak perlu enrich extras heavy, tapi kalau mau boleh:
	// enrichEnrollmentExtras(c.Context(), ctl.DB, schoolID, resp)

	return helper.JsonList(c, "ok", resp, pagination)
}
