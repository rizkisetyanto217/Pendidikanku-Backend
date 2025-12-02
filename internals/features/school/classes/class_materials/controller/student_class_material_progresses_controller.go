// file: internals/features/school/classes/class_materials/controller/student_class_material_progress_controller.go
package controller

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	dto "madinahsalam_backend/internals/features/school/classes/class_materials/dto"
	model "madinahsalam_backend/internals/features/school/classes/class_materials/model"
)

/* =======================================================
   Controller struct
======================================================= */

type StudentClassMaterialProgressController struct {
	DB *gorm.DB
}

func NewStudentClassMaterialProgressController(db *gorm.DB) *StudentClassMaterialProgressController {
	return &StudentClassMaterialProgressController{DB: db}
}


/* =======================================================
   Ping / Upsert progress materi murid
   POST /.../student/class-material-progress/ping
   - Akses: hanya STUDENT di school yang sama
   - Body: StudentClassMaterialProgressPingRequest
======================================================= */

func (ctl *StudentClassMaterialProgressController) PingMyClassMaterialProgress(c *fiber.Ctx) error {
	var body dto.StudentClassMaterialProgressPingRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}

	// Validasi minimal id materi & scsst
	if body.StudentClassMaterialProgressSCSSTID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "student_class_material_progress_scsst_id is required")
	}
	if body.StudentClassMaterialProgressClassMaterialID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "student_class_material_progress_class_material_id is required")
	}

	// Resolve school
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}

	// Resolve student (dan pastikan ini memang murid di school tersebut)
	studentID, err := helperAuth.ResolveStudentIDFromContext(c, schoolID)
	if err != nil {
		return err
	}

	now := time.Now()

	// Upsert berdasarkan (school_id, scsst_id, class_material_id)
	var progress model.StudentClassMaterialProgressModel
	tx := ctl.DB.WithContext(c.Context())

	err = tx.
		Where(
			"student_class_material_progress_school_id = ? AND student_class_material_progress_scsst_id = ? AND student_class_material_progress_class_material_id = ?",
			schoolID,
			body.StudentClassMaterialProgressSCSSTID,
			body.StudentClassMaterialProgressClassMaterialID,
		).
		First(&progress).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to get material progress")
	}

	if err == gorm.ErrRecordNotFound {
		// Belum ada → buat baru
		newModel := body.ToNewModel(schoolID, studentID, now)
		if err := tx.Create(newModel).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "failed to create material progress")
		}
		resp := dto.NewStudentClassMaterialProgressResponse(newModel)
		return helper.JsonOK(c, "ok", resp)
	}

	// Sudah ada → apply ping ke existing
	body.ApplyToModel(&progress, now)

	if err := tx.Save(&progress).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to update material progress")
	}

	resp := dto.NewStudentClassMaterialProgressResponse(&progress)
	return helper.JsonOK(c, "ok", resp)
}
