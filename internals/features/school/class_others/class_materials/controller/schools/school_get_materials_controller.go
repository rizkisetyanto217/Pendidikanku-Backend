package controller

import (
	"madinahsalam_backend/internals/features/school/class_others/class_materials/dto"
	materialModel "madinahsalam_backend/internals/features/school/class_others/class_materials/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

/* =======================================================
   List School Materials
   GET /.../school-materials
   - Akses: member school (student/teacher/dkm/admin/bendahara)
======================================================= */

func (ctl *SchoolMaterialController) ListSchoolMaterials(c *fiber.Ctx) error {
	var q listQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid query params")
	}

	// Resolve school dari token/context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// err dari resolver biasanya sudah JsonError
		return err
	}

	// Guard: minimal member school (student / teacher / dkm / admin / bendahara)
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return err
	}

	// Base query tenant-scoped
	dbBase := ctl.DB.WithContext(c.Context()).
		Model(&materialModel.SchoolMaterialModel{}).
		Where("school_material_school_id = ?", schoolID)

	// Soft delete filter
	if q.WithDeleted == nil || !*q.WithDeleted {
		dbBase = dbBase.Where("school_material_deleted = FALSE")
	}

	// Filter tambahan
	if q.ClassSubjectID != nil {
		dbBase = dbBase.Where("school_material_class_subject_id = ?", *q.ClassSubjectID)
	}
	if q.IsPublished != nil {
		dbBase = dbBase.Where("school_material_is_published = ?", *q.IsPublished)
	}
	if q.IsActive != nil {
		dbBase = dbBase.Where("school_material_is_active = ?", *q.IsActive)
	}

	// Hitung total dulu
	var total int64
	if err := dbBase.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to count school materials")
	}

	// Paging
	paging := helper.ResolvePaging(c, 20, 100) // default 20, max 100

	var materials []*materialModel.SchoolMaterialModel
	if err := dbBase.
		Order("school_material_meeting_number NULLS LAST, school_material_default_order NULLS LAST, school_material_created_at ASC").
		Limit(paging.Limit).
		Offset(paging.Offset).
		Find(&materials).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to list school materials")
	}

	resp := dto.NewSchoolMaterialResponseList(materials)
	pagination := helper.BuildPaginationFromOffset(total, paging.Offset, paging.Limit)

	return helper.JsonList(c, "ok", resp, pagination)
}
