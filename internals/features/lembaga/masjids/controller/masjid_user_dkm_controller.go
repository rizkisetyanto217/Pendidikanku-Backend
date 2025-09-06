// file: internals/features/masjids/masjids/controller/masjid_controller.go
package controller

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	dto "masjidku_backend/internals/features/lembaga/masjids/dto"
	model "masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =======================================================
// Helpers lokal
// =======================================================

func strPtrOrNil(s string, lower bool) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	if lower {
		l := strings.ToLower(t)
		return &l
	}
	return &t
}

func boolFromForm(v string) bool {
	return v == "true" || v == "1" || strings.ToLower(v) == "yes"
}

// parse masjid_levels dari multipart/JSON form
func parseLevelsFromRequest(c *fiber.Ctx) []string {
	levels := make([]string, 0, 8)

	// 1) JSON string / CSV single field
	if raw := strings.TrimSpace(c.FormValue("masjid_levels")); raw != "" {
		var arr []string
		if err := json.Unmarshal([]byte(raw), &arr); err == nil {
			levels = append(levels, arr...)
		} else {
			parts := strings.Split(raw, ",")
			levels = append(levels, parts...)
		}
	}
	// 2) Multipart array: masjid_levels[]
	if mf, _ := c.MultipartForm(); mf != nil {
		if vals, ok := mf.Value["masjid_levels[]"]; ok {
			levels = append(levels, vals...)
		}
		// 3) Repeated keys: masjid_levels=...&masjid_levels=...
		if vals, ok := mf.Value["masjid_levels"]; ok && len(vals) > 1 {
			levels = append(levels, vals...)
		}
	}

	// Normalisasi (trim + dedup)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(levels))
	for _, v := range levels {
		t := strings.TrimSpace(v)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		t := strings.TrimSpace(r)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func clampInt(vs string, def, min, max int) int {
	if vs == "" {
		return def
	}
	v, err := strconv.Atoi(vs)
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}


// =======================================================
// POST /api/a/masjids (CreateMasjidDKM)
// =======================================================

// CreateMasjidDKM — versi inti (tanpa media/sosial/Maps)
func (mc *MasjidController) CreateMasjidDKM(c *fiber.Ctx) error {
	// Auth
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// Terima multipart / JSON
	isMultipart := strings.Contains(c.Get("Content-Type"), "multipart/form-data")
	if !isMultipart && !strings.Contains(c.Get("Content-Type"), "application/json") {
		if mf, _ := c.MultipartForm(); mf == nil {
			return helper.JsonError(c, fiber.StatusUnsupportedMediaType, "Gunakan multipart/form-data atau application/json")
		}
	}

	// Fields inti
	name := strings.TrimSpace(c.FormValue("masjid_name"))
	if name == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama masjid wajib diisi")
	}
	baseSlug := helper.GenerateSlug(name)
	if baseSlug == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Nama masjid tidak valid untuk slug")
	}

	bio := c.FormValue("masjid_bio_short")
	location := c.FormValue("masjid_location")
	domain := c.FormValue("masjid_domain")
	isIslamicSchool := boolFromForm(c.FormValue("masjid_is_islamic_school"))

	// Relasi opsional
	var yayasanID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("masjid_yayasan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			yayasanID = &id
		}
	}
	var planIDPtr *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("masjid_current_plan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			planIDPtr = &id
		}
	}

	// Verifikasi
	verifStatus := strings.TrimSpace(c.FormValue("masjid_verification_status"))
	if verifStatus == "" {
		verifStatus = "pending"
	}
	verifNotes := c.FormValue("masjid_verification_notes")

	// Levels (tags)
	reqLevels := parseLevelsFromRequest(c)

	// Transaksi
	var respDTO dto.MasjidResponse
	txErr := mc.DB.Transaction(func(tx *gorm.DB) error {
		// Slug unik
		slug, err := helper.EnsureUniqueSlug(tx, baseSlug, "masjids", "masjid_slug")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik")
		}

		newID := uuid.New()
		domainPtr := strPtrOrNil(domain, true)

		// Build record
		newMasjid := model.MasjidModel{
			MasjidID:            newID,
			MasjidYayasanID:     yayasanID,
			MasjidCurrentPlanID: planIDPtr,

			MasjidName:     name,
			MasjidBioShort: strPtrOrNil(bio, false),
			MasjidLocation: strPtrOrNil(location, false),

			MasjidDomain: domainPtr,
			MasjidSlug:   slug,

			MasjidIsActive:           true,
			MasjidVerificationStatus: model.VerificationStatus(verifStatus),
			MasjidVerificationNotes:  strPtrOrNil(verifNotes, false),

			MasjidIsIslamicSchool: isIslamicSchool,
		}
		// Set levels JSONB
		if len(reqLevels) > 0 {
			_ = newMasjid.SetLevels(reqLevels)
		} else {
			_ = newMasjid.SetLevels([]string{})
		}
		// Sinkron flag verifikasi
		syncVerificationFlags(&newMasjid)

		// Simpan
		if err := tx.Create(&newMasjid).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan masjid")
		}

		// Role best-effort
		_ = helperAuth.EnsureGlobalRole(tx, userID, "user", &userID)
		// Grant DKM wajib
		if err := helperAuth.GrantScopedRoleDKM(tx, userID, newMasjid.MasjidID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal grant peran DKM")
		}

		respDTO = dto.FromModelMasjid(&newMasjid)
		return nil
	})
	if txErr != nil {
		if fe, ok := txErr.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Transaksi gagal")
	}

	return helper.JsonCreated(c, "Masjid berhasil dibuat", respDTO)
}

// =======================================================
// GET /api/masjids  (list + filter)
// =======================================================

func (mc *MasjidController) GetMasjids(c *fiber.Ctx) error {
	tx := mc.DB.Model(&model.MasjidModel{})

	// q: ILIKE (match trigram index)
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		qq := "%" + q + "%"
		tx = tx.Where("(masjid_name ILIKE ? OR masjid_location ILIKE ? OR masjid_bio_short ILIKE ?)", qq, qq, qq)
	}

	// flags
	if v := strings.TrimSpace(c.Query("verified")); v != "" {
		tx = tx.Where("masjid_is_verified = ?", v == "true" || v == "1")
	}
	if v := strings.TrimSpace(c.Query("active")); v != "" {
		tx = tx.Where("masjid_is_active = ?", v == "true" || v == "1")
	}
	if v := strings.TrimSpace(c.Query("is_islamic_school")); v != "" {
		tx = tx.Where("masjid_is_islamic_school = ?", v == "true" || v == "1")
	}

	// relations
	if s := strings.TrimSpace(c.Query("yayasan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			tx = tx.Where("masjid_yayasan_id = ?", id)
		}
	}
	if s := strings.TrimSpace(c.Query("plan_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			tx = tx.Where("masjid_current_plan_id = ?", id)
		}
	}

	// levels_any => OR of "masjid_levels ? ?"
	if s := strings.TrimSpace(c.Query("levels_any")); s != "" {
		parts := splitCSV(s)
		if len(parts) > 0 {
			orSQL := make([]string, 0, len(parts))
			args := make([]interface{}, 0, len(parts))
			for _, p := range parts {
				orSQL = append(orSQL, "masjid_levels ? ?")
				args = append(args, p)
			}
			tx = tx.Where("("+strings.Join(orSQL, " OR ")+")", args...)
		}
	}
	// levels_all => masjid_levels @> '["a","b"]'::jsonb
	if s := strings.TrimSpace(c.Query("levels_all")); s != "" {
		parts := splitCSV(s)
		if len(parts) > 0 {
			jb, _ := json.Marshal(parts)
			tx = tx.Where("masjid_levels @> ?::jsonb", string(jb))
		}
	}

	limit := clampInt(c.Query("limit"), 20, 1, 100)
	offset := clampInt(c.Query("offset"), 0, 0, 100000)
	tx = tx.Limit(limit).Offset(offset).Order("masjid_created_at DESC")

	var rows []model.MasjidModel
	if err := tx.Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid")
	}

	out := make([]dto.MasjidResponse, 0, len(rows))
	for i := range rows {
		out = append(out, dto.FromModelMasjid(&rows[i]))
	}
	return helper.JsonOK(c, "OK", fiber.Map{
		"items":  out,
		"count":  len(out),
		"limit":  limit,
		"offset": offset,
	})
}

// =======================================================
// GET /api/masjids/:id_or_slug (detail 1 masjid)
// =======================================================

func (mc *MasjidController) GetMasjid(c *fiber.Ctx) error {
	key := strings.TrimSpace(c.Params("id_or_slug"))
	if key == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter kosong")
	}

	var row model.MasjidModel
	var err error
	if id, parseErr := uuid.Parse(key); parseErr == nil {
		err = mc.DB.First(&row, "masjid_id = ?", id).Error
	} else {
		err = mc.DB.First(&row, "masjid_slug = ?", key).Error
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data masjid")
	}
	return helper.JsonOK(c, "OK", dto.FromModelMasjid(&row))
}

// =======================================================
// GET /api/masjids/:id/profile  (profil + primary files)
// =======================================================

func (mc *MasjidController) GetMasjidProfile(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// 1) Ambil profil
	var mp model.MasjidProfileModel
	if err := mc.DB.First(&mp, "masjid_profile_masjid_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Profil masjid belum tersedia")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil profil")
	}

	// 2) Ambil primary files via VIEW (join sekali jalan)
	type primaryURLRow struct {
		Type               string     `gorm:"column:type" json:"type"`
		FileURL            string     `gorm:"column:file_url" json:"file_url"`
		TrashURL           *string    `gorm:"column:trash_url" json:"trash_url,omitempty"`
		DeletePendingUntil *time.Time `gorm:"column:delete_pending_until" json:"delete_pending_until,omitempty"`
		CreatedAt          time.Time  `gorm:"column:created_at" json:"created_at"`
		UpdatedAt          time.Time  `gorm:"column:updated_at" json:"updated_at"`
	}
	var prim []primaryURLRow
	if err := mc.DB.
		Table("masjids_profiles mp").
		Select("v.type, v.file_url, v.trash_url, v.delete_pending_until, v.created_at, v.updated_at").
		Joins("LEFT JOIN v_masjid_primary_urls v ON v.masjid_id = mp.masjid_profile_masjid_id").
		Where("mp.masjid_profile_masjid_id = ?", id).
		Where("v.type IS NOT NULL").
		Scan(&prim).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil primary files")
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"profile":       dto.FromModelMasjidProfile(&mp),
		"primary_files": prim,
	})
}

// =======================================================
// GET /api/masjids/:id/urls (primary + gallery)
// =======================================================

func (mc *MasjidController) GetMasjidURLs(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Primary files (VIEW)
	type primaryURLRow struct {
		Type               string     `gorm:"column:type" json:"type"`
		FileURL            string     `gorm:"column:file_url" json:"file_url"`
		TrashURL           *string    `gorm:"column:trash_url" json:"trash_url,omitempty"`
		DeletePendingUntil *time.Time `gorm:"column:delete_pending_until" json:"delete_pending_until,omitempty"`
		CreatedAt          time.Time  `gorm:"column:created_at" json:"created_at"`
		UpdatedAt          time.Time  `gorm:"column:updated_at" json:"updated_at"`
	}
	var primary []primaryURLRow
	if err := mc.DB.
		Table("masjids_profiles mp").
		Select("v.type, v.file_url, v.trash_url, v.delete_pending_until, v.created_at, v.updated_at").
		Joins("LEFT JOIN v_masjid_primary_urls v ON v.masjid_id = mp.masjid_profile_masjid_id").
		Where("mp.masjid_profile_masjid_id = ?", id).
		Where("v.type IS NOT NULL").
		Scan(&primary).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil primary files")
	}

	// Gallery (non-singleton) dari tabel masjid_urls
	type galleryURLRow struct {
		MasjidURLID uuid.UUID `gorm:"column:masjid_url_id" json:"masjid_url_id"`
		Type        string    `gorm:"column:masjid_url_type" json:"type"`
		FileURL     string    `gorm:"column:masjid_url_file_url" json:"file_url"`
		CreatedAt   time.Time `gorm:"column:masjid_url_created_at" json:"created_at"`
		UpdatedAt   time.Time `gorm:"column:masjid_url_updated_at" json:"updated_at"`
		IsPrimary   bool      `gorm:"column:masjid_url_is_primary" json:"is_primary"`
	}
	var gallery []galleryURLRow
	if err := mc.DB.
		Table("masjid_urls u").
		Select("u.masjid_url_id, u.masjid_url_type, u.masjid_url_file_url, u.masjid_url_created_at, u.masjid_url_updated_at, u.masjid_url_is_primary").
		Where("u.masjid_url_masjid_id = ? AND u.masjid_url_deleted_at IS NULL AND u.masjid_url_is_active = TRUE", id).
		Where("u.masjid_url_type = ?", "gallery").
		Order("u.masjid_url_created_at DESC").
		Scan(&gallery).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil gallery")
	}

	return helper.JsonOK(c, "OK", fiber.Map{
		"primary_files": primary,
		"gallery":       gallery,
	})
}
