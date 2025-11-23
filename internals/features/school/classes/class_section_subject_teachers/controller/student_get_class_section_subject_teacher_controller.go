package controller

import (
	dto "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	model "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	"strings"

	"github.com/gofiber/fiber/v2"
)

/* =========================================================
   LIST
   ========================================================= */

// GET /api/a/student-csst
func (ctl *StudentCSSTController) List(c *fiber.Ctx) error {
	// 1) School context dari token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// 2) Guard: staff (teacher/dkm/admin/bendahara)
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	// 3) Parse query
	var q dto.StudentCSSTListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "query params tidak valid")
	}

	// default paging
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 200 {
		q.PageSize = 20
	}

	// 4) Base query
	tx := ctl.DB.WithContext(c.Context()).
		Model(&model.StudentClassSectionSubjectTeacher{}).
		Where("student_class_section_subject_teacher_school_id = ?", schoolID)

	// Filters
	if q.StudentID != nil {
		tx = tx.Where("student_class_section_subject_teacher_student_id = ?", *q.StudentID)
	}
	if q.CSSTID != nil {
		tx = tx.Where("student_class_section_subject_teacher_csst_id = ?", *q.CSSTID)
	}
	if q.IsActive != nil {
		tx = tx.Where("student_class_section_subject_teacher_is_active = ?", *q.IsActive)
	}
	if !q.IncludeDeleted {
		tx = tx.Where("student_class_section_subject_teacher_deleted_at IS NULL")
	}

	// Pencarian ringan (sementara by slug)
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		like := "%" + strings.TrimSpace(*q.Q) + "%"
		tx = tx.Where("student_class_section_subject_teacher_slug ILIKE ?", like)
	}

	// Sorting
	sortBy := strings.ToLower(strings.TrimSpace(q.SortBy))
	order := strings.ToLower(strings.TrimSpace(q.Order))
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	switch sortBy {
	case dto.StudentCSSTSortUpdatedAt:
		tx = tx.Order("student_class_section_subject_teacher_updated_at " + order)
	case dto.StudentCSSTSortStudent:
		tx = tx.Order("student_class_section_subject_teacher_student_id " + order)
	case dto.StudentCSSTSortCSST:
		tx = tx.Order("student_class_section_subject_teacher_csst_id " + order)
	default:
		tx = tx.Order("student_class_section_subject_teacher_created_at " + order)
	}

	// 5) Hitung total
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal menghitung total")
	}

	if total == 0 {
		resp := dto.StudentCSSTListResponse{
			Data: []dto.StudentCSSTItem{},
			Meta: dto.PageMeta{
				Total:       0,
				Page:        q.Page,
				PageSize:    q.PageSize,
				TotalPages:  0,
				HasNext:     false,
				HasPrevious: false,
			},
		}
		return helper.JsonOK(c, "ok", resp)
	}

	// 6) Ambil page
	var rows []model.StudentClassSectionSubjectTeacher
	if err := tx.
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	items := make([]dto.StudentCSSTItem, 0, len(rows))
	for i := range rows {
		items = append(items, toStudentCSSTItem(&rows[i]))
	}

	totalPages := int((total + int64(q.PageSize) - 1) / int64(q.PageSize))
	resp := dto.StudentCSSTListResponse{
		Data: items,
		Meta: dto.PageMeta{
			Total:       total,
			Page:        q.Page,
			PageSize:    q.PageSize,
			TotalPages:  totalPages,
			HasNext:     q.Page < totalPages,
			HasPrevious: q.Page > 1,
		},
	}
	return helper.JsonOK(c, "ok", resp)
}
