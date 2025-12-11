// file: internals/features/school/materials/controller/class_materials_controller.go
package controller

import (
	"errors"
	"log"
	"strings"

	"madinahsalam_backend/internals/features/school/class_others/class_materials/dto"
	"madinahsalam_backend/internals/features/school/class_others/class_materials/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	dbtime "madinahsalam_backend/internals/helpers/dbtime"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
   Controller struct & constructor
========================================================= */

type ClassMaterialsController struct {
	DB *gorm.DB
}

func NewClassMaterialsController(db *gorm.DB) *ClassMaterialsController {
	return &ClassMaterialsController{DB: db}
}

/* =========================================================
   Local validation helpers (tanpa helper.ValidateStruct)
========================================================= */

var allowedTypes = map[string]bool{
	"article":    true,
	"doc":        true,
	"ppt":        true,
	"pdf":        true,
	"image":      true,
	"youtube":    true,
	"video_file": true,
	"link":       true,
	"embed":      true,
}

func validateClassMaterialCreateDTO(req *dto.ClassMaterialCreateRequestDTO) map[string][]string {
	errors := map[string][]string{}

	// title required
	if strings.TrimSpace(req.ClassMaterialTitle) == "" {
		errors["class_material_title"] = append(errors["class_material_title"], "class_material_title is required")
	}

	// type required + oneof
	t := strings.TrimSpace(req.ClassMaterialType)
	if t == "" {
		errors["class_material_type"] = append(errors["class_material_type"], "class_material_type is required")
	} else if !allowedTypes[t] {
		errors["class_material_type"] = append(errors["class_material_type"], "invalid class_material_type")
	}

	if len(errors) == 0 {
		return nil
	}
	return errors
}

func validateClassMaterialUpdateDTO(req *dto.ClassMaterialUpdateRequestDTO) map[string][]string {
	errors := map[string][]string{}

	// kalau type diisi, harus valid
	if req.ClassMaterialType != nil {
		t := strings.TrimSpace(*req.ClassMaterialType)
		if t == "" {
			errors["class_material_type"] = append(errors["class_material_type"], "class_material_type cannot be empty")
		} else if !allowedTypes[t] {
			errors["class_material_type"] = append(errors["class_material_type"], "invalid class_material_type")
		}
	}

	if len(errors) == 0 {
		return nil
	}
	return errors
}

// POST /api/t/csst/:csst_id/materials
// 🔐 khusus DKM/Admin sekolah (bukan guru biasa)
func (h *ClassMaterialsController) TeacherCreate(c *fiber.Ctx) error {
	// resolve school dari context + guard DKM/admin
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		// ResolveSchoolIDFromContext sudah return JsonError kalau gagal
		return err
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		// EnsureDKMSchool juga sudah pakai JsonError
		return err
	}

	// created_by: kalau nanti ada helper GetUserIDFromToken, bisa diisi
	var createdBy *uuid.UUID = nil

	csstID, err := uuid.Parse(c.Params("csst_id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid csst_id")
	}

	var req dto.ClassMaterialCreateRequestDTO
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	// validasi lokal
	if ferrs := validateClassMaterialCreateDTO(&req); ferrs != nil {
		return helper.JsonValidationError(c, ferrs)
	}

	// pakai waktu dari DB (dbtime helper)
	now, err := dbtime.GetDBTime(c)
	if err != nil {
		log.Printf("[TeacherCreate] get db time error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to get server time")
	}

	var m model.ClassMaterialsModel
	dto.ApplyCreateDTOToModel(&req, &m, schoolID, csstID, createdBy, now)

	if err := h.DB.Create(&m).Error; err != nil {
		log.Printf("[TeacherCreate] insert error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to create material")
	}

	return helper.JsonCreated(c, "created", dto.FromModel(&m))
}

// PATCH /api/t/csst/:csst_id/materials/:material_id
// 🔐 khusus DKM/Admin sekolah (bukan guru biasa)
func (h *ClassMaterialsController) TeacherUpdate(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	csstID, err := uuid.Parse(c.Params("csst_id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid csst_id")
	}

	materialID, err := uuid.Parse(c.Params("material_id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid material_id")
	}

	var req dto.ClassMaterialUpdateRequestDTO
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	// validasi lokal (khusus field yang diisi)
	if ferrs := validateClassMaterialUpdateDTO(&req); ferrs != nil {
		return helper.JsonValidationError(c, ferrs)
	}

	var m model.ClassMaterialsModel
	if err := h.DB.
		Where("class_material_id = ? AND class_material_school_id = ? AND class_material_csst_id = ? AND NOT class_material_deleted",
			materialID, schoolID, csstID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "material not found")
		}
		log.Printf("[TeacherUpdate] find error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch material")
	}

	// pakai waktu dari DB (dbtime helper)
	now, err := dbtime.GetDBTime(c)
	if err != nil {
		log.Printf("[TeacherUpdate] get db time error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to get server time")
	}

	dto.ApplyUpdateDTOToModel(&req, &m, now)

	if err := h.DB.Save(&m).Error; err != nil {
		log.Printf("[TeacherUpdate] save error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to update material")
	}

	return helper.JsonUpdated(c, "updated", dto.FromModel(&m))
}

// DELETE /api/t/csst/:csst_id/materials/:material_id
// 🔐 khusus DKM/Admin sekolah (bukan guru biasa)
// soft delete: set class_material_deleted = true, deleted_at = now
func (h *ClassMaterialsController) TeacherSoftDelete(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	csstID, err := uuid.Parse(c.Params("csst_id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid csst_id")
	}

	materialID, err := uuid.Parse(c.Params("material_id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid material_id")
	}

	// pakai waktu dari DB (dbtime helper)
	now, err := dbtime.GetDBTime(c)
	if err != nil {
		log.Printf("[TeacherSoftDelete] get db time error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to get server time")
	}

	fields := dto.BuildSoftDeleteFields(now)

	res := h.DB.
		Model(&model.ClassMaterialsModel{}).
		Where("class_material_id = ? AND class_material_school_id = ? AND class_material_csst_id = ? AND NOT class_material_deleted",
			materialID, schoolID, csstID).
		Updates(fields)
	if res.Error != nil {
		log.Printf("[TeacherSoftDelete] update error: %v", res.Error)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to delete material")
	}
	if res.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "material not found")
	}

	return helper.JsonDeleted(c, "deleted", nil)
}
