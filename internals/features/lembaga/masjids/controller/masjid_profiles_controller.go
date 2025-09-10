// file: internals/features/lembaga/masjids/controller/masjid_profile_controller.go
package controller

import (
	"errors"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	d "masjidku_backend/internals/features/lembaga/masjids/dto"
	m "masjidku_backend/internals/features/lembaga/masjids/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

/* =======================================================
   Controller & Constructor
   ======================================================= */

type MasjidProfileController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewMasjidProfileController(db *gorm.DB, v *validator.Validate) *MasjidProfileController {
	return &MasjidProfileController{DB: db, Validate: v}
}

/* =======================================================
   Helpers
   ======================================================= */

func isUniqueViolation(err error) bool {
	// Postgres unique_violation code = 23505
	return err != nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}

/* =======================================================
   Handlers
   ======================================================= */

// Create (Admin-only). Satu masjid cuma boleh punya 1 profile.
// POST /
func (ctl *MasjidProfileController) Create(c *fiber.Ctx) error {
	if !helperAuth.IsDKM(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: admin saja")
	}

	var req d.MasjidProfileCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(&req); err != nil {
			// Gantikan ValidationError → 422 Unprocessable
			return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
		}
	}

	// (Opsional) Override masjid_id dari token kalau ingin strict tenant
	if tokenMasjidID, err := helperAuth.GetMasjidIDFromToken(c); err == nil && tokenMasjidID != uuid.Nil {
		req.MasjidProfileMasjidID = tokenMasjidID.String()
	}

	model := d.ToModelMasjidProfileCreate(&req)

	// Pastikan belum ada profile utk masjid ini (UNIQUE)
	var count int64
	if err := ctl.DB.Model(&m.MasjidProfileModel{}).
		Where("masjid_profile_masjid_id = ?", model.MasjidProfileMasjidID).
		Count(&count).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}
	if count > 0 {
		return helper.JsonError(c, fiber.StatusConflict, "Profil untuk masjid ini sudah ada")
	}

	if err := ctl.DB.Create(model).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Duplikasi NPSN/NSS/masjid_id")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat profil: "+err.Error())
	}

	resp := d.FromModelMasjidProfile(model)
	return helper.JsonCreated(c, "Profil masjid berhasil dibuat", resp)
}

// GET /:id
func (ctl *MasjidProfileController) GetByID(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var p m.MasjidProfileModel
	if err := ctl.DB.First(&p, "masjid_profile_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}
	return helper.JsonOK(c, "OK", d.FromModelMasjidProfile(&p))
}

// GET /by-masjid/:masjid_id
func (ctl *MasjidProfileController) GetByMasjidID(c *fiber.Ctx) error {
	masjidID, err := parseUUIDParam(c, "masjid_id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var p m.MasjidProfileModel
	if err := ctl.DB.First(&p, "masjid_profile_masjid_id = ?", masjidID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil untuk masjid ini belum ada")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}
	return helper.JsonOK(c, "OK", d.FromModelMasjidProfile(&p))
}

// GET / (list + filter + pagination)
func (ctl *MasjidProfileController) List(c *fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 1000 {
		limit = 20
	}
	offset := (page - 1) * limit

	dbq := ctl.DB.Model(&m.MasjidProfileModel{}).Where("masjid_profile_deleted_at IS NULL")

	// Full-text search (tsvector)
	if q != "" {
		dbq = dbq.Where("masjid_profile_search @@ plainto_tsquery('simple', ?)", q)
	}

	// Filters
	if acc := strings.TrimSpace(c.Query("accreditation")); acc != "" {
		dbq = dbq.Where("masjid_profile_school_accreditation = ?", acc)
	}
	if ib := strings.TrimSpace(c.Query("is_boarding")); ib != "" {
		switch strings.ToLower(ib) {
		case "true", "1", "yes", "y":
			dbq = dbq.Where("masjid_profile_school_is_boarding = TRUE")
		case "false", "0", "no", "n":
			dbq = dbq.Where("masjid_profile_school_is_boarding = FALSE")
		}
	}
	if fyMin := strings.TrimSpace(c.Query("founded_year_min")); fyMin != "" {
		if v, err := strconv.Atoi(fyMin); err == nil {
			dbq = dbq.Where("masjid_profile_founded_year >= ?", v)
		}
	}
	if fyMax := strings.TrimSpace(c.Query("founded_year_max")); fyMax != "" {
		if v, err := strconv.Atoi(fyMax); err == nil {
			dbq = dbq.Where("masjid_profile_founded_year <= ?", v)
		}
	}

	// Count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// Data
	var rows []m.MasjidProfileModel
	if err := dbq.
		Order("masjid_profile_created_at DESC").
		Offset(offset).Limit(limit).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	items := make([]d.MasjidProfileResponse, 0, len(rows))
	for i := range rows {
		items = append(items, d.FromModelMasjidProfile(&rows[i]))
	}

	// Pakai JsonList: data & pagination dipisah
	return helper.JsonList(c, items, fiber.Map{
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": int(math.Ceil(float64(total) / float64(limit))),
	})
}

// GET /nearest?lat=..&lon=..&limit=..
func (ctl *MasjidProfileController) Nearest(c *fiber.Ctx) error {
	latStr := strings.TrimSpace(c.Query("lat"))
	lonStr := strings.TrimSpace(c.Query("lon"))
	limitStr := strings.TrimSpace(c.Query("limit", "10"))

	if latStr == "" || lonStr == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "lat & lon wajib")
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "lat tidak valid")
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "lon tidak valid")
	}
	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 200 {
		limit = 10
	}

	type rowWithDist struct {
		m.MasjidProfileModel
		Distance float64 `gorm:"column:distance"`
	}

	var rows []rowWithDist

	// earth_distance return meter (butuh extension earthdistance + cube)
	sql := `
	SELECT mp.*,
		earth_distance(
			ll_to_earth(?, ?),
			ll_to_earth(mp.masjid_profile_latitude::float8, mp.masjid_profile_longitude::float8)
		) as distance
	FROM masjids_profiles mp
	WHERE mp.masjid_profile_latitude IS NOT NULL
	AND mp.masjid_profile_longitude IS NOT NULL
	AND mp.masjid_profile_deleted_at IS NULL
	ORDER BY distance ASC
	LIMIT ?;
	`
	if err := ctl.DB.Raw(sql, lat, lon, limit).Scan(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	type item struct {
		d.MasjidProfileResponse
		Distance float64 `json:"distance_meters"`
	}

	out := make([]item, 0, len(rows))
	for i := range rows {
		resp := d.FromModelMasjidProfile(&rows[i].MasjidProfileModel)
		out = append(out, item{MasjidProfileResponse: resp, Distance: rows[i].Distance})
	}

	return helper.JsonOK(c, "OK", fiber.Map{"items": out})
}

// PATCH /:id
func (ctl *MasjidProfileController) Update(c *fiber.Ctx) error {
	if !helperAuth.IsOwner(c) && !helperAuth.IsDKM(c) {
		// ganti sesuai kebijakan aksesmu; minimal admin
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak")
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var req d.MasjidProfileUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}

	var p m.MasjidProfileModel
	if err := ctl.DB.First(&p, "masjid_profile_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error: "+err.Error())
	}

	// Terapkan patch ke struct in-memory
	d.ApplyPatchToModel(&p, &req)

	// Update selective fields (hindari overwrite kolom generated/read-only)
	if err := ctl.DB.Model(&p).
		Omit(
			"masjid_profile_id",
			"masjid_profile_masjid_id",
			"masjid_profile_search",
			"masjid_profile_created_at",
		).
		Select("*").
		Updates(&p).Error; err != nil {

		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Duplikasi NPSN/NSS")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update: "+err.Error())
	}

	// Reload to get updated_at
	if err := ctl.DB.First(&p, "masjid_profile_id = ?", id).Error; err != nil {
		log.Println("[WARN] reload after update:", err)
	}

	return helper.JsonUpdated(c, "Profil masjid berhasil diperbarui", d.FromModelMasjidProfile(&p))
}

// DELETE (soft)
// DELETE /:id
func (ctl *MasjidProfileController) Delete(c *fiber.Ctx) error {
	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Akses ditolak: admin saja")
	}

	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Soft delete pakai timestamp ke kolom deleted_at
	if err := ctl.DB.
		Model(&m.MasjidProfileModel{}).
		Where("masjid_profile_id = ?", id).
		Update("masjid_profile_deleted_at", time.Now()).
		Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus: "+err.Error())
	}

	return helper.JsonDeleted(c, "Profil masjid dihapus (soft delete)", fiber.Map{"deleted": true})
}

/* =======================================================
   Notes
   =======================================================

- Pastikan extensions:
  CREATE EXTENSION IF NOT EXISTS earthdistance;
  CREATE EXTENSION IF NOT EXISTS cube;

- Index di DDL (GIN tsvector, GIST ll_to_earth) akan mempercepat query /search & nearest.

- Tenant-safe (opsional):
  * Create: override masjid_id dari token (sudah disiapkan).
  * Get/Update/Delete: validasi p.MasjidProfileMasjidID == tokenMasjidID untuk non-admin.

- Jika ingin update super-aman (map only changed fields), ganti blok Updates dengan map dari field non-nil pada req.
*/
