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

// Skenario 2: endpoint khusus murid → hanya melihat enrollments miliknya sendiri
//
// GET /api/u/:school_id/my/class-enrollments
// ?status_in=...
// ?applied_from=...
// ?applied_to=...
// ?academic_term_id=...
// ?term_id=...
// ?academic_year=...
// ?angkatan=...
// ?q=...
// ?order_by=...
// ?sort=...
// ?limit=...
// ?offset=...
// ?view=compact|full|summary
// file: internals/features/school/classes/class_enrollments/controller/list.go
func (ctl *StudentClassEnrollmentController) ListMe(c *fiber.Ctx) error {
	// ========== tenant dari TOKEN (bukan dari path) ==========
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err // helper sudah balikin JsonError
	}

	// Hanya murid dari school ini yang diizinkan
	if err := helperAuth.EnsureStudentSchool(c, schoolID); err != nil {
		return err
	}

	// 🔹 Ambil student_id dari token (bukan user_id)
	studentID, err := helperAuth.GetPrimarySchoolStudentID(c)
	if err != nil || studentID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Konteks murid tidak ditemukan")
	}

	// ========== query (DTO) ==========
	var q dto.ListStudentClassEnrollmentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query")
	}

	// FORCE: hanya enrollment milik murid ini
	q.StudentID = &studentID

	// status_in (comma-separated → slice) → override ke q.StatusIn
	if raw := strings.TrimSpace(c.Query("status_in")); raw != "" {
		sts, er := parseStatusInParam(raw)
		if er != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, er.Error())
		}
		q.StatusIn = sts
	}

	// view mode
	view := strings.ToLower(strings.TrimSpace(c.Query("view"))) // "", "compact", "full", "summary"

	// paging (masih pakai page/per_page helper)
	pg := helper.ResolvePaging(c, 20, 200)

	// ========== base query ==========
	base := ctl.DB.WithContext(c.Context()).
		Model(&emodel.StudentClassEnrollmentModel{}).
		Where("student_class_enrollments_school_id = ?", schoolID).
		Where("student_class_enrollments_school_student_id = ?", studentID).
		Where("student_class_enrollments_deleted_at IS NULL") // untuk murid: hanya alive

	// ========== filters tambahan dari DTO ==========
	// status_in
	if len(q.StatusIn) > 0 {
		base = base.Where("student_class_enrollments_status IN ?", q.StatusIn)
	}

	// applied_from / applied_to
	if q.AppliedFrom != nil {
		base = base.Where("student_class_enrollments_applied_at >= ?", *q.AppliedFrom)
	}
	if q.AppliedTo != nil {
		base = base.Where("student_class_enrollments_applied_at <= ?", *q.AppliedTo)
	}

	// ===== TERM FILTERS (academic_term_id / term_id / academic_year / angkatan) =====
	var termID *uuid.UUID
	if q.AcademicTermID != nil && *q.AcademicTermID != uuid.Nil {
		termID = q.AcademicTermID
	} else if q.TermID != nil && *q.TermID != uuid.Nil {
		termID = q.TermID
	}
	if termID != nil {
		base = base.Where("student_class_enrollments_term_id = ?", *termID)
	}

	if strings.TrimSpace(q.AcademicYear) != "" {
		base = base.Where(
			"student_class_enrollments_term_academic_year_snapshot = ?",
			strings.TrimSpace(q.AcademicYear),
		)
	}
	if q.Angkatan != nil {
		base = base.Where("student_class_enrollments_term_angkatan_snapshot = ?", *q.Angkatan)
	}

	// ===== Q search (nama siswa / nama kelas / nama term) =====
	if strings.TrimSpace(q.Q) != "" {
		pat := "%" + strings.TrimSpace(q.Q) + "%"
		base = base.Where(`
			student_class_enrollments_student_name_snapshot ILIKE ?
			OR student_class_enrollments_class_name_snapshot ILIKE ?
			OR COALESCE(student_class_enrollments_term_name_snapshot, '') ILIKE ?
		`, pat, pat, pat)
	}

	// ========== count ==========
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to count")
	}

	// ========== data ==========
	tx := base

	// murid juga bisa pakai compact/summary view biar ringan
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
	return helper.JsonList(c, "ok", resp, pagination)
}
