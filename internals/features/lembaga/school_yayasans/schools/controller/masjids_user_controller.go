package controller

import (
	"log"

	helper "schoolku_backend/internals/helpers"

	schoolDto "schoolku_backend/internals/features/lembaga/school_yayasans/schools/dto"
	schoolModel "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// 🟢 GET ALL MASJIDS
func (mc *SchoolController) GetAllSchools(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all schools")

	var schools []schoolModel.SchoolModel
	if err := mc.DB.Find(&schools).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch schools: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school")
	}

	log.Printf("[SUCCESS] Retrieved %d schools\n", len(schools))

	// 🔁 Transform ke DTO response
	resp := make([]schoolDto.SchoolResp, 0, len(schools))
	for i := range schools {
		resp = append(resp, schoolDto.FromModel(&schools[i]))
	}

	return helper.JsonList(c, resp, fiber.Map{
		"total": len(resp),
	})
}

// 🟢 GET VERIFIED MASJIDS
func (mc *SchoolController) GetAllVerifiedSchools(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all verified schools")

	var schools []schoolModel.SchoolModel
	if err := mc.DB.Where("school_is_verified = ?", true).Find(&schools).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch verified schools: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school terverifikasi")
	}

	log.Printf("[SUCCESS] Retrieved %d verified schools\n", len(schools))

	// 🔁 Transform ke DTO response
	resp := make([]schoolDto.SchoolResp, 0, len(schools))
	for i := range schools {
		resp = append(resp, schoolDto.FromModel(&schools[i]))
	}

	return helper.JsonList(c, resp, fiber.Map{
		"total": len(resp),
	})
}

// 🟢 GET VERIFIED MASJID BY ID
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

	return helper.JsonOK(c, "Data school terverifikasi berhasil diambil", schoolDto.FromModel(&m))
}

// 🟢 GET MASJID BY SLUG
func (mc *SchoolController) GetSchoolBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	log.Printf("[INFO] Fetching school with slug: %s\n", slug)

	var m schoolModel.SchoolModel
	if err := mc.DB.Where("school_slug = ?", slug).First(&m).Error; err != nil {
		log.Printf("[ERROR] School with slug %s not found\n", slug)
		return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
	}

	log.Printf("[SUCCESS] Retrieved school: %s\n", m.SchoolName)

	return helper.JsonOK(c, "Data school berhasil diambil", schoolDto.FromModel(&m))
}
