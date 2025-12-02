// file: internals/features/school/materials/controller/school_material_controller.go
package controller

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	dto "madinahsalam_backend/internals/features/school/classes/class_materials/dto"
	materialModel "madinahsalam_backend/internals/features/school/classes/class_materials/model"
)

/* =======================================================
   Query params (list)
======================================================= */

type listQuery struct {
	ClassSubjectID *uuid.UUID `query:"class_subject_id"`
	IsPublished    *bool      `query:"is_published"`
	IsActive       *bool      `query:"is_active"`
	WithDeleted    *bool      `query:"with_deleted"`
}

/* =======================================================
   Controller struct
======================================================= */

type SchoolMaterialController struct {
	DB *gorm.DB
}

func NewSchoolMaterialController(db *gorm.DB) *SchoolMaterialController {
	return &SchoolMaterialController{DB: db}
}

/* =======================================================
   Helper: get user_id (school_id pakai resolver baru)
======================================================= */

func getUserIDFromContext(c *fiber.Ctx) *uuid.UUID {
	// pakai helperAuth.GetUserIDFromToken (ini yang ada di helper.go kamu)
	uid, err := helperAuth.GetUserIDFromToken(c)
	if err != nil || uid == uuid.Nil {
		return nil
	}
	return &uid
}



/* =======================================================
   Get Detail School Material
   GET /.../school-materials/:id
   - Akses: member school
======================================================= */

func (ctl *SchoolMaterialController) GetSchoolMaterialDetail(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "missing id")
	}

	materialID, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Pastikan user memang member school
	if err := helperAuth.EnsureMemberSchool(c, schoolID); err != nil {
		return err
	}

	var material materialModel.SchoolMaterialModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("school_material_id = ? AND school_material_school_id = ?", materialID, schoolID).
		Where("school_material_deleted = FALSE").
		First(&material).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "school material not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to get school material")
	}

	resp := dto.NewSchoolMaterialResponse(&material)
	return helper.JsonOK(c, "ok", resp)
}

/* =======================================================
   Create School Material
   POST /.../school-materials
   - Akses: hanya DKM / Teacher / Admin (via ResolveSchoolForDKMOrTeacher)
======================================================= */

func (ctl *SchoolMaterialController) CreateSchoolMaterial(c *fiber.Ctx) error {
	var body dto.SchoolMaterialCreateRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	// Guard + resolve school untuk DKM/Teacher/Admin
	schoolID, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		return err
	}

	userID := getUserIDFromContext(c)

	// Validasi minimal
	if body.SchoolMaterialTitle == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_material_title is required")
	}
	if body.SchoolMaterialType == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "school_material_type is required")
	}

	model := body.ToModel(schoolID, userID)

	if err := ctl.DB.WithContext(c.Context()).
		Create(model).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to create school material")
	}

	resp := dto.NewSchoolMaterialResponse(model)
	return helper.JsonCreated(c, "created", resp)
}

/* =======================================================
   Update School Material
   PATCH /.../school-materials/:id
   - Akses: hanya DKM / Teacher / Admin
======================================================= */

func (ctl *SchoolMaterialController) UpdateSchoolMaterial(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "missing id")
	}

	materialID, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	// Guard + resolve school untuk DKM/Teacher/Admin
	schoolID, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		return err
	}

	var body dto.SchoolMaterialUpdateRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	var material materialModel.SchoolMaterialModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("school_material_id = ? AND school_material_school_id = ?", materialID, schoolID).
		Where("school_material_deleted = FALSE").
		First(&material).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "school material not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to get school material")
	}

	body.ApplyToModel(&material)

	if err := ctl.DB.WithContext(c.Context()).
		Save(&material).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to update school material")
	}

	resp := dto.NewSchoolMaterialResponse(&material)
	return helper.JsonUpdated(c, "updated", resp)
}

/* =======================================================
   Delete (Soft Delete) School Material
   DELETE /.../school-materials/:id
   - Akses: hanya DKM / Teacher / Admin
======================================================= */

func (ctl *SchoolMaterialController) DeleteSchoolMaterial(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if idStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "missing id")
	}

	materialID, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid id")
	}

	// Guard + resolve school untuk DKM/Teacher/Admin
	schoolID, err := helperAuth.ResolveSchoolForDKMOrTeacher(c)
	if err != nil {
		return err
	}

	var material materialModel.SchoolMaterialModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("school_material_id = ? AND school_material_school_id = ?", materialID, schoolID).
		Where("school_material_deleted = FALSE").
		First(&material).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "school material not found")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to get school material")
	}

	now := time.Now()
	material.SchoolMaterialDeleted = true
	material.SchoolMaterialDeletedAt = &now

	if err := ctl.DB.WithContext(c.Context()).
		Save(&material).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to delete school material")
	}

	return helper.JsonDeleted(c, "deleted", fiber.Map{
		"school_material_id":    material.SchoolMaterialID,
		"school_material_title": material.SchoolMaterialTitle,
	})
}
