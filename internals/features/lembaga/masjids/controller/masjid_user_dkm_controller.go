// file: internals/features/masjids/masjids/controller/masjid_controller.go
package controller

import (
	"strconv"
	"strings"

	dto "masjidku_backend/internals/features/lembaga/masjids/dto"
	model "masjidku_backend/internals/features/lembaga/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =========================
// Helpers lokal ringan
// =========================

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

// =========================
// CreateMasjidDKM
// =========================

// CreateMasjidDKM — versi baru (tanpa media/sosial/Maps)
func (mc *MasjidController) CreateMasjidDKM(c *fiber.Ctx) error {
	// 1) Auth
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// 2) Terima multipart atau JSON sederhana (toleran)
	isMultipart := strings.Contains(c.Get("Content-Type"), "multipart/form-data")
	if !isMultipart && !strings.Contains(c.Get("Content-Type"), "application/json") {
		// fallback ke multipart agar kompatibel
		if mf, _ := c.MultipartForm(); mf == nil {
			return helper.JsonError(c, fiber.StatusUnsupportedMediaType, "Gunakan multipart/form-data atau application/json")
		}
	}

	// 3) Ambil field inti
	//   - pakai FormValue agar kompatibel baik multipart maupun JSON (Fiber mengisi untuk keduanya)
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

	// Optional: Yayasan & Plan
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

	// Optional verif input; default pending
	verifStatus := strings.TrimSpace(c.FormValue("masjid_verification_status"))
	if verifStatus == "" {
		verifStatus = "pending"
	}
	verifNotes := c.FormValue("masjid_verification_notes")

	// Koordinat
	var latPtr, longPtr *float64
	if v := strings.TrimSpace(c.FormValue("masjid_latitude")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			latPtr = &f
		}
	}
	if v := strings.TrimSpace(c.FormValue("masjid_longitude")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			longPtr = &f
		}
	}

	// 4) Transaksi
	var respDTO dto.MasjidResponse
	txErr := mc.DB.Transaction(func(tx *gorm.DB) error {
		// a) Slug unik
		slug, err := helper.EnsureUniqueSlug(tx, baseSlug, "masjids", "masjid_slug")
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat slug unik")
		}

		newID := uuid.New()

		// b) Domain pointer (kosong → NULL), case-insensitive
		domainPtr := strPtrOrNil(domain, true)

		// c) Simpan masjid (tanpa media/sosial/Maps)
		newMasjid := model.MasjidModel{
			MasjidID:                 newID,
			MasjidYayasanID:          yayasanID,
			MasjidCurrentPlanID:      planIDPtr,

			MasjidName:      name,
			MasjidBioShort:  strPtrOrNil(bio, false),
			MasjidLocation:  strPtrOrNil(location, false),
			MasjidLatitude:  latPtr,
			MasjidLongitude: longPtr,

			MasjidDomain: domainPtr,
			MasjidSlug:   slug,

			MasjidIsActive:           true,
			MasjidVerificationStatus: model.VerificationStatus(verifStatus),
			MasjidVerificationNotes:  strPtrOrNil(verifNotes, false),

			MasjidIsIslamicSchool: isIslamicSchool,
		}
		if err := tx.Create(&newMasjid).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan masjid")
		}

		// d1) (opsional) pastikan creator minimal punya global 'user' — best-effort
		if err := helperAuth.EnsureGlobalRole(tx, userID, "user", &userID); err != nil {
			// tidak fatal
		}

		// d2) (WAJIB) Grant peran scoped 'dkm' via user_roles (atomic)
		if err := helperAuth.GrantScopedRoleDKM(tx, userID, newMasjid.MasjidID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal grant peran DKM")
		}

		// e) Build response DTO
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
