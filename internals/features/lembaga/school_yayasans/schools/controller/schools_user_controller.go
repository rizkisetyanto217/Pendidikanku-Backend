// file: internals/features/lembaga/school_yayasans/schools/controller/school_controller.go
package controller

import (
	"log"

	helper "schoolku_backend/internals/helpers"

	schoolDto "schoolku_backend/internals/features/lembaga/school_yayasans/schools/dto"
	schoolModel "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// 🟢 GET ALL SCHOOLS (tanpa paging param → seluruh data 1 halaman)
func (mc *SchoolController) GetAllSchools(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all schools")

	var schools []schoolModel.SchoolModel
	if err := mc.DB.Find(&schools).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch schools: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school")
	}

	log.Printf("[SUCCESS] Retrieved %d schools\n", len(schools))

	resp := make([]schoolDto.SchoolResp, 0, len(schools))
	for i := range schools {
		resp = append(resp, schoolDto.FromModel(&schools[i]))
	}

	// pagination default: seluruh data dalam satu halaman
	total := len(resp)
	pg := helper.Pagination{
		Page:       1,
		PerPage:    total, // biarkan 0 jika memang kosong; helper akan tetap aman
		Total:      int64(total),
		TotalPages: 1,
		HasNext:    false,
		HasPrev:    false,
	}
	return helper.JsonList(c, "ok", resp, pg)
}

// 🟢 GET VERIFIED SCHOOLS (tanpa paging param → seluruh data 1 halaman)
func (mc *SchoolController) GetAllVerifiedSchools(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all verified schools")

	var schools []schoolModel.SchoolModel
	if err := mc.DB.Where("school_is_verified = ?", true).Find(&schools).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch verified schools: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school terverifikasi")
	}

	log.Printf("[SUCCESS] Retrieved %d verified schools\n", len(schools))

	resp := make([]schoolDto.SchoolResp, 0, len(schools))
	for i := range schools {
		resp = append(resp, schoolDto.FromModel(&schools[i]))
	}

	total := len(resp)
	pg := helper.Pagination{
		Page:       1,
		PerPage:    total,
		Total:      int64(total),
		TotalPages: 1,
		HasNext:    false,
		HasPrev:    false,
	}
	return helper.JsonList(c, "ok", resp, pg)
}

// 🟢 GET VERIFIED SCHOOL BY ID (single resource)
func (mc *SchoolController) GetVerifiedSchoolByID(c *fiber.Ctx) error {
	id := c.Params("id")
	log.Printf("[INFO] Fetching verified school with ID: %s\n", id)

	schoolUUID, err := uuid.Parse(id)
	if err != nil {
		log.Printf("[ERROR] Invalid UUID format: %v\n", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Format ID tidak valid")
	}

	var m schoolModel.SchoolModel
	if err := mc.DB.
		Where("school_id = ? AND school_is_verified = ?", schoolUUID, true).
		First(&m).Error; err != nil {
		log.Printf("[ERROR] Verified school with ID %s not found\n", id)
		return helper.JsonError(c, fiber.StatusNotFound, "School terverifikasi tidak ditemukan")
	}

	log.Printf("[SUCCESS] Retrieved verified school: %s\n", m.SchoolName)
	return helper.JsonOK(c, "ok", schoolDto.FromModel(&m))
}

// 🟢 GET SCHOOL BY SLUG (single resource)
func (mc *SchoolController) GetSchoolBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	log.Printf("[INFO] Fetching school with slug: %s\n", slug)

	var m schoolModel.SchoolModel
	if err := mc.DB.Where("school_slug = ?", slug).First(&m).Error; err != nil {
		log.Printf("[ERROR] School with slug %s not found\n", slug)
		return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
	}

	log.Printf("[SUCCESS] Retrieved school: %s\n", m.SchoolName)
	return helper.JsonOK(c, "ok", schoolDto.FromModel(&m))
}
