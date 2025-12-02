package controller

import (
	"log"
	"madinahsalam_backend/internals/features/school/classes/class_materials/dto"
	"madinahsalam_backend/internals/features/school/classes/class_materials/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
)

/* =========================================================
   Handlers (Teacher / Staff)
========================================================= */

// GET /api/t/csst/:csst_id/materials
// akses: masih longgar (selama punya school_id di token dan anggota school)
func (h *ClassMaterialsController) List(c *fiber.Ctx) error {
	// ambil school_id dari token (gaya lama)
	schoolID, err := helperAuth.GetSchoolIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// csst_id dari path
	csstParam := c.Params("csst_id")
	csstID, err := uuid.Parse(csstParam)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid csst_id")
	}

	// paging (?page= & ?per_page=/limit=)
	paging := helper.ResolvePaging(c, 20, 100)

	var (
		total  int64
		models []model.ClassMaterialsModel
	)

	// hitung total
	if err := h.DB.
		Model(&model.ClassMaterialsModel{}).
		Where("class_material_school_id = ? AND class_material_csst_id = ? AND NOT class_material_deleted",
			schoolID, csstID).
		Count(&total).Error; err != nil {

		log.Printf("[TeacherListByCSST] count error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch materials")
	}

	// ambil data dengan limit+offset
	if err := h.DB.
		Where("class_material_school_id = ? AND class_material_csst_id = ? AND NOT class_material_deleted",
			schoolID, csstID).
		Order("class_material_meeting_number NULLS LAST, class_material_order NULLS LAST, class_material_created_at ASC").
		Limit(paging.Limit).
		Offset(paging.Offset).
		Find(&models).Error; err != nil {

		log.Printf("[TeacherListByCSST] query error: %v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to fetch materials")
	}

	resp := make([]*dto.ClassMaterialResponseDTO, 0, len(models))
	for i := range models {
		resp = append(resp, dto.FromModel(&models[i]))
	}

	pagination := helper.BuildPaginationFromOffset(total, paging.Offset, paging.Limit)
	return helper.JsonList(c, "ok", resp, pagination)
}
