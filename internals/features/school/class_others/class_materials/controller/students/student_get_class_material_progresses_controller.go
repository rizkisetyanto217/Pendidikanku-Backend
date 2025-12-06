package controller

import (
	"madinahsalam_backend/internals/features/school/class_others/class_materials/dto"
	"madinahsalam_backend/internals/features/school/class_others/class_materials/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =======================================================
   List progress materi milik murid yang login
   GET /.../student/class-material-progress
   - Akses: hanya STUDENT di school yang sama
======================================================= */

func (ctl *StudentClassMaterialProgressController) ListMyClassMaterialProgress(c *fiber.Ctx) error {
	var q dto.StudentClassMaterialProgressListQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query params")
	}

	// Resolve school dari context/token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// resolver sudah mengirim JsonError
		return err
	}

	// Ambil student_id dari token + validasi memang murid di school tsb
	studentID, err := helperAuth.ResolveStudentIDFromContext(c, schoolID)
	if err != nil {
		return err
	}

	dbBase := ctl.DB.WithContext(c.Context()).
		Model(&model.StudentClassMaterialProgressModel{}).
		Where("student_class_material_progress_school_id = ?", schoolID).
		Where("student_class_material_progress_student_id = ?", studentID)

	// Filter by SCSST
	if q.StudentClassMaterialProgressSCSSTID != nil {
		dbBase = dbBase.Where("student_class_material_progress_scsst_id = ?", *q.StudentClassMaterialProgressSCSSTID)
	}

	// Filter by class_material
	if q.StudentClassMaterialProgressClassMaterialID != nil {
		dbBase = dbBase.Where("student_class_material_progress_class_material_id = ?", *q.StudentClassMaterialProgressClassMaterialID)
	}

	// Filter by status
	if q.StudentClassMaterialProgressStatus != nil && *q.StudentClassMaterialProgressStatus != "" {
		dbBase = dbBase.Where("student_class_material_progress_status = ?", *q.StudentClassMaterialProgressStatus)
	}

	// Hitung total
	var total int64
	if err := dbBase.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to count material progress")
	}

	// Paging
	paging := helper.ResolvePaging(c, 20, 100)

	var progresses []*model.StudentClassMaterialProgressModel
	if err := dbBase.
		Order("student_class_material_progress_created_at ASC").
		Limit(paging.Limit).
		Offset(paging.Offset).
		Find(&progresses).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to list material progress")
	}

	resp := dto.NewStudentClassMaterialProgressResponseList(progresses)
	pagination := helper.BuildPaginationFromOffset(total, paging.Offset, paging.Limit)

	return helper.JsonList(c, "ok", resp, pagination)
}

/* =======================================================
   Get progress 1 materi milik murid yang login
   GET /.../student/class-material-progress/by-material/:class_material_id
   - Akses: hanya STUDENT di school yang sama
   - Dipakai misal saat buka halaman detail materi:
     fetch progress 1 row (kalau belum ada → 404 / not found)
======================================================= */

func (ctl *StudentClassMaterialProgressController) GetMyClassMaterialProgressByMaterial(c *fiber.Ctx) error {
	classMaterialIDStr := c.Params("class_material_id")
	if classMaterialIDStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "class_material_id is required")
	}

	classMaterialID, err := uuid.Parse(classMaterialIDStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid class_material_id")
	}

	// Resolve school dari context/token
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Ambil student_id dari token
	studentID, err := helperAuth.ResolveStudentIDFromContext(c, schoolID)
	if err != nil {
		return err
	}

	var progress model.StudentClassMaterialProgressModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("student_class_material_progress_school_id = ?", schoolID).
		Where("student_class_material_progress_student_id = ?", studentID).
		Where("student_class_material_progress_class_material_id = ?", classMaterialID).
		First(&progress).Error; err != nil {

		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "material progress not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to get material progress")
	}

	resp := dto.NewStudentClassMaterialProgressResponse(&progress)
	return helper.JsonOK(c, "ok", resp)
}
