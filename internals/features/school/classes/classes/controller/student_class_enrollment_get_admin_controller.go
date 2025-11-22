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

func (ctl *StudentClassEnrollmentController) List(c *fiber.Ctx) error {
	// ========== tenant dari TOKEN (bukan dari path) ==========
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err // helper sudah balikin JsonError
	}

	// ❗ hanya DKM/Admin (boleh tambah bendahara kalau mau)
	if er := helperAuth.EnsureDKMSchool(c, schoolID); er != nil {
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
	view := strings.ToLower(strings.TrimSpace(c.Query("view"))) // "", "compact", "summary", "full"

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

	// ===== TERM FILTERS =====
	// Prioritas: academic_term_id (baru), kalau kosong pakai term_id (legacy)
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

	// TODO: kalau mau search Q (nama siswa/kelas/term) bisa ditambah di sini

	// ========== count ==========
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to count")
	}

	// ========== data ==========
	tx := base

	// optimisasi kolom saat compact/summary
	if view == "compact" || view == "summary" {
		tx = tx.Select([]string{
			// id & status & nominal
			"student_class_enrollments_id",
			"student_class_enrollments_status",
			"student_class_enrollments_total_due_idr",

			// convenience (mirror snapshot & ids)
			"student_class_enrollments_school_student_id",
			"student_class_enrollments_student_name_snapshot",
			"student_class_enrollments_class_id",
			"student_class_enrollments_class_name_snapshot",

			// term (denormalized, optional)
			"student_class_enrollments_term_id",
			"student_class_enrollments_term_name_snapshot",
			"student_class_enrollments_term_academic_year_snapshot",
			"student_class_enrollments_term_angkatan_snapshot",

			// payment snapshot
			"student_class_enrollments_payment_snapshot",

			// jejak penting
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

	// (opsional) enrich convenience fields tambahan (Username, dll.)
	enrichEnrollmentExtras(c.Context(), ctl.DB, schoolID, resp)

	return helper.JsonList(c, "ok", resp, pagination)
}
