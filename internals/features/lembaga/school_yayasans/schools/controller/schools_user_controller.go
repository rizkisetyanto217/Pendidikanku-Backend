// file: internals/features/lembaga/school_yayasans/schools/controller/school_controller.go
package controller

import (
	"log"
	"strings"

	helper "madinahsalam_backend/internals/helpers"

	schoolDto "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/dto"
	schoolModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)


// 🟢 GET ALL SCHOOLS (filter by name & id/ids) + PAGINATION (?page=&per_page=)
func (mc *SchoolController) GetAllSchools(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all schools (paginated)")

	// ==== query params ====
	q := strings.TrimSpace(c.Query("q"))          // search by name (ILIKE)
	id := strings.TrimSpace(c.Query("id"))        // single id
	idsParam := strings.TrimSpace(c.Query("ids")) // multiple ids, comma-separated

	// ==== paging params ====
	// defaultPerPage=20; maxPerPage=100 (silakan sesuaikan)
	paging := helper.ResolvePaging(c, 20, 100)

	// ==== sesuaikan nama kolom sesuai skema DB ====
	const colID = "school_id" // PK tabel (disesuaikan)
	const colName = "school_name"    // kolom nama (ganti "school_name" jika perlu)

	// ==== base query (filter) ====
	dbq := mc.DB.Model(&schoolModel.SchoolModel{})

	// filter single id
	if id != "" {
		if _, err := uuid.Parse(id); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter id tidak valid (harus UUID)")
		}
		dbq = dbq.Where(colID+" = ?", id)
	}

	// filter multiple ids
	if idsParam != "" {
		raw := strings.Split(idsParam, ",")
		ids := make([]string, 0, len(raw))
		for _, s := range raw {
			v := strings.TrimSpace(s)
			if v == "" {
				continue
			}
			if _, err := uuid.Parse(v); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ids mengandung UUID tidak valid")
			}
			ids = append(ids, v)
		}
		if len(ids) > 0 {
			dbq = dbq.Where(colID+" IN ?", ids)
		}
	}

	// filter by name (ILIKE)
	if q != "" {
		dbq = dbq.Where(colName+" ILIKE ?", "%"+q+"%")
	}

	// ==== total count (sebelum limit/offset) ====
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		log.Printf("[ERROR] Count schools failed: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data school")
	}

	// ==== ambil data page ini ====
	// Optional: stabilkan urutan
	dbq = dbq.Order(colName + " ASC")

	var schools []schoolModel.SchoolModel
	if err := dbq.
		Limit(paging.Limit).
		Offset(paging.Offset).
		Find(&schools).Error; err != nil {
		log.Printf("[ERROR] Failed to fetch schools: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school")
	}

	log.Printf("[SUCCESS] Retrieved %d schools (page=%d per_page=%d total=%d)\n",
		len(schools), paging.Page, paging.PerPage, total)

	// ==== mapping DTO ====
	resp := make([]schoolDto.SchoolResp, 0, len(schools))
	for i := range schools {
		resp = append(resp, schoolDto.FromModel(&schools[i]))
	}

	// ==== build pagination dari offset/limit ====
	pg := helper.BuildPaginationFromOffset(total, paging.Offset, paging.Limit)

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
