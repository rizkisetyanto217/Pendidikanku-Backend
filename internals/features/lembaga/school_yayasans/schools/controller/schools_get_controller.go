// file: internals/features/lembaga/school_yayasans/schools/controller/school_controller.go
package controller

import (
	"log"
	"strings"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	schoolDto "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/dto"
	schoolModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
🟢 GET SCHOOLS — super fleksibel + default ambil school dari token

Bisa handle:

  - List + paging:
    GET /api/u/schools?page=1&per_page=20&q=madinah

  - Filter by verified:
    GET /api/u/schools?verified_only=1

  - Filter by id(s):
    GET /api/u/schools?id=<uuid>
    GET /api/u/schools?ids=<uuid1>,<uuid2>

  - Filter by slug:
    GET /api/u/schools?slug=madinah-salam

  - Filter by tenant profile:
    GET /api/u/schools?tenant_profile=school_basic

  - Filter by default attendance mode:
    GET /api/u/schools?attendance_mode=teacher_only|student_only|both

  - Filter by city:
    GET /api/u/schools?city=bekasi

  - Detail by path param:
    GET /api/u/schools/:id
    GET /api/u/schools/slug/:slug

  - Paksa single mode:
    GET /api/u/schools?mode=single&id=<uuid>

  - Default (nggak kirim apa-apa):
    GET /api/u/schools
    → otomatis pakai school_id dari token (via ResolveSchoolIDFromContext), single mode

  - Include profile:
    GET /api/u/schools?include=profile
    GET /api/u/schools/:id?include=profile
*/
func (mc *SchoolController) GetSchools(c *fiber.Ctx) error {
	log.Println("[INFO] [GetSchools] called")

	// ==== path params (opsional) ====
	pathID := strings.TrimSpace(c.Params("id"))
	pathSlug := strings.TrimSpace(c.Params("slug"))

	// ==== query params ====
	q := strings.TrimSpace(c.Query("q"))                        // search by name (ILIKE)
	id := strings.TrimSpace(c.Query("id"))                      // single id (query)
	idsParam := strings.TrimSpace(c.Query("ids"))               // multiple ids
	slug := strings.TrimSpace(c.Query("slug"))                  // slug (query)
	mode := strings.TrimSpace(c.Query("mode"))                  // "single" / "list"
	verifiedOnly := strings.TrimSpace(c.Query("verified_only")) // "1"/"true"/...

	tenantProfile := strings.TrimSpace(c.Query("tenant_profile"))   // filter by tenant_profile
	attendanceMode := strings.TrimSpace(c.Query("attendance_mode")) // filter by default attendance mode
	cityFilter := strings.TrimSpace(c.Query("city"))                // filter by city (exact lower)

	includeParam := strings.TrimSpace(c.Query("include")) // "profile", "profile,xxx"

	// path param override query param
	if pathID != "" {
		id = pathID
	}
	if pathSlug != "" {
		slug = pathSlug
	}

	// parse include (profile)
	wantProfile := false
	if includeParam != "" {
		for _, part := range strings.Split(includeParam, ",") {
			if strings.TrimSpace(strings.ToLower(part)) == "profile" {
				wantProfile = true
				break
			}
		}
	}

	// ==== tentukan single / list mode (sementara) ====
	singleMode := false
	if mode == "single" || mode == "detail" {
		singleMode = true
	}
	if pathID != "" || pathSlug != "" {
		singleMode = true
	}

	log.Printf("[INFO] [GetSchools] raw params: q=%q id=%q ids=%q slug=%q verified_only=%q mode=%q include=%q tenant_profile=%q attendance_mode=%q city=%q singleMode(init)=%v\n",
		q, id, idsParam, slug, verifiedOnly, mode, includeParam, tenantProfile, attendanceMode, cityFilter, singleMode)

	const colID = "school_id"
	const colName = "school_name"

	dbq := mc.DB.Model(&schoolModel.SchoolModel{})

	// ==== include: profile (Preload relasi) ====
	if wantProfile {
		log.Println("[INFO] [GetSchools] include=profile → Preload(SchoolProfile)")
		dbq = dbq.Preload("SchoolProfile")
	}

	// ==== filter verified_only ====
	if verifiedOnly != "" {
		val := strings.ToLower(verifiedOnly)
		if val == "1" || val == "true" || val == "yes" {
			dbq = dbq.Where("school_is_verified = ?", true)
		} else if val == "0" || val == "false" || val == "no" {
			// kalau mau khusus unverified-only, bisa aktifkan:
			// dbq = dbq.Where("school_is_verified = ?", false)
		}
	}

	// ==== filter single id (query/path) ====
	if id != "" {
		if _, err := uuid.Parse(id); err != nil {
			log.Printf("[WARN] [GetSchools] invalid id UUID: %v\n", err)
			return helper.JsonError(c, fiber.StatusBadRequest, "Parameter id tidak valid (harus UUID)")
		}
		dbq = dbq.Where(colID+" = ?", id)
	}

	// ==== filter multiple ids ====
	if idsParam != "" {
		raw := strings.Split(idsParam, ",")
		ids := make([]string, 0, len(raw))
		for _, s := range raw {
			v := strings.TrimSpace(s)
			if v == "" {
				continue
			}
			if _, err := uuid.Parse(v); err != nil {
				log.Printf("[WARN] [GetSchools] invalid UUID in ids: %v\n", err)
				return helper.JsonError(c, fiber.StatusBadRequest, "Parameter ids mengandung UUID tidak valid")
			}
			ids = append(ids, v)
		}
		if len(ids) > 0 {
			dbq = dbq.Where(colID+" IN ?", ids)
		}
	}

	// ==== filter slug ====
	if slug != "" {
		dbq = dbq.Where("school_slug = ?", slug)
	}

	// ==== filter by name (ILIKE) ====
	if q != "" {
		dbq = dbq.Where(colName+" ILIKE ?", "%"+q+"%")
	}

	// ==== filter by tenant_profile (opsional) ====
	if tenantProfile != "" {
		dbq = dbq.Where("school_tenant_profile = ?", strings.ToLower(tenantProfile))
	}

	// ==== filter by default attendance mode (opsional) ====
	if attendanceMode != "" {
		dbq = dbq.Where("school_default_attendance_entry_mode = ?", strings.ToLower(attendanceMode))
	}

	// ==== filter by city (opsional, exact lower) ====
	if cityFilter != "" {
		dbq = dbq.Where("LOWER(school_city) = ?", strings.ToLower(cityFilter))
	}

	// ==== DEFAULT: kalau tidak kirim filter apa-apa → pakai school_id dari context ====
	noExplicitFilter := (q == "" && id == "" && idsParam == "" && slug == "" && verifiedOnly == "" &&
		tenantProfile == "" && attendanceMode == "" && cityFilter == "")
	if noExplicitFilter {
		// 🔥 pakai helper yang kamu kasih: ResolveSchoolIDFromContext
		schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
		if err != nil {
			// catatan: ResolveSchoolIDFromContext sudah boleh return helper.JsonError(...)
			// jadi di sini cukup return err apa adanya
			log.Printf("[ERROR] [GetSchools] no filters + ResolveSchoolIDFromContext error: %v\n", err)
			return err
		}

		if schoolID == uuid.Nil {
			log.Println("[WARN] [GetSchools] no filters + ResolveSchoolIDFromContext returned Nil UUID")
			return helper.JsonError(c, fiber.StatusBadRequest, "User tidak memiliki school aktif di token")
		}

		log.Printf("[INFO] [GetSchools] no filters → default to school_id from context: %s\n", schoolID)
		dbq = dbq.Where(colID+" = ?", schoolID)

		// default mode jadi single (karena konteks 1 sekolah aktif)
		singleMode = true
	}

	// stabilkan order
	dbq = dbq.Order(colName + " ASC")

	// ==== SINGLE MODE ====
	if singleMode {
		var m schoolModel.SchoolModel
		if err := dbq.First(&m).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Printf("[WARN] [GetSchools] single mode - school not found\n")
				return helper.JsonError(c, fiber.StatusNotFound, "School tidak ditemukan")
			}
			log.Printf("[ERROR] [GetSchools] single mode - query failed: %v\n", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school")
		}

		log.Printf(
			"[SUCCESS] [GetSchools] single mode - got school: %s (%s), include_profile=%v\n",
			m.SchoolName, m.SchoolID, wantProfile,
		)

		// DTO FromModel sudah kirim semua field baru (attendance mode, timezone, settings, dsb)
		return helper.JsonOK(c, "ok", schoolDto.FromModel(&m))
	}

	// ==== LIST MODE (paginated) ====
	paging := helper.ResolvePaging(c, 20, 100)

	// total count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		log.Printf("[ERROR] [GetSchools] count failed: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data school")
	}

	var schools []schoolModel.SchoolModel
	if err := dbq.
		Limit(paging.Limit).
		Offset(paging.Offset).
		Find(&schools).Error; err != nil {
		log.Printf("[ERROR] [GetSchools] list query failed: %v\n", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data school")
	}

	log.Printf("[SUCCESS] [GetSchools] list mode - got %d schools (page=%d per_page=%d total=%d include_profile=%v)\n",
		len(schools), paging.Page, paging.PerPage, total, wantProfile)

	resp := make([]schoolDto.SchoolResp, 0, len(schools))
	for i := range schools {
		resp = append(resp, schoolDto.FromModel(&schools[i]))
	}

	pg := helper.BuildPaginationFromOffset(total, paging.Offset, paging.Limit)
	return helper.JsonList(c, "ok", resp, pg)
}
