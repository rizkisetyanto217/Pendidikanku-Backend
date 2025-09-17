// internals/features/users/user/controller/users_profile_controller.go
package controller

import (
	"context"
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	profileDTO "masjidku_backend/internals/features/users/user/dto"
	profileModel "masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================
// UsersProfileController (versi pakai OSSService helper terbaru)
// =============================================================

type UsersProfileController struct {
	DB  *gorm.DB
	OSS *helperOSS.OSSService
}

// Sediakan dua cara inisialisasi:
// 1) inject OSS yg sudah dibuat di main
func NewUsersProfileController(db *gorm.DB, oss *helperOSS.OSSService) *UsersProfileController {
	return &UsersProfileController{DB: db, OSS: oss}
}

// 2) atau build dari ENV langsung (opsional)
func NewUsersProfileControllerFromEnv(db *gorm.DB) *UsersProfileController {
	oss, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil {
		log.Printf("[WARN] OSS init gagal: %v", err)
	}
	return &UsersProfileController{DB: db, OSS: oss}
}

func httpErr(c *fiber.Ctx, err error) error {
	if fe, ok := err.(*fiber.Error); ok {
		return helper.JsonError(c, fe.Code, fe.Message)
	}
	// default: unauthorized agar tetap backward-compatible dgn versi lama
	return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
}

/* =========================
   GET: All profiles (DTO)
   ========================= */
func (upc *UsersProfileController) GetProfiles(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all user_profile")
	var profiles []profileModel.UserProfileModel
	if err := upc.DB.WithContext(c.Context()).Find(&profiles).Error; err != nil {
		log.Println("[ERROR] Failed to fetch user_profile:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch profiles")
	}
	return helper.JsonOK(c, "Profiles fetched", profileDTO.ToUsersProfileDTOs(profiles))
}

/* =========================
   GET: My profile (DTO)
   ========================= */
func (upc *UsersProfileController) GetProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return httpErr(c, err)
	}
	log.Println("[INFO] Fetching user_profile with user_profile_user_id:", userID)

	var profile profileModel.UserProfileModel
	if err := upc.DB.WithContext(c.Context()).
		Where("user_profile_user_id = ?", userID).
		First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "User profile not found")
		}
		log.Println("[ERROR] DB error:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch user profile")
	}
	return helper.JsonOK(c, "Profile fetched", profileDTO.ToUsersProfileDTO(profile))
}

/* =========================
   POST /profiles (Create only)
   ========================= */
func (upc *UsersProfileController) CreateProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil { return httpErr(c, err) }

	var in profileDTO.CreateUsersProfileRequest
	if err := c.BodyParser(&in); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request format")
	}

	if in.UsersProfilePhoneNumber != nil {
		v := strings.TrimSpace(*in.UsersProfilePhoneNumber)
		v = strings.ReplaceAll(v, " ", "")
		in.UsersProfilePhoneNumber = &v
	}

	row := in.ToModel(userID)
	db := upc.DB.WithContext(c.Context())
	if err := db.Create(&row).Error; err != nil {
		le := strings.ToLower(err.Error())
		if strings.Contains(le, "duplicate key") || strings.Contains(le, "unique constraint") {
			return helper.JsonError(c, fiber.StatusConflict, "User profile already exists")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create user profile")
	}
	return helper.JsonCreated(c, "User profile created", profileDTO.ToUsersProfileDTO(row))
}

/* =========================
   PATCH /profiles (partial update, no file)
   ========================= */
func (upc *UsersProfileController) UpdateProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil { return httpErr(c, err) }
	log.Println("[INFO] Patching user_profile with user_profile_user_id:", userID)

	var profile profileModel.UserProfileModel
	if err := upc.DB.WithContext(c.Context()).
		Where("user_profile_user_id = ?", userID).
		First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "User profile not found")
		}
		log.Println("[ERROR] DB error:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch user profile")
	}

	ct := strings.TrimSpace(c.Get(fiber.HeaderContentType))
	isJSON := strings.HasPrefix(ct, fiber.MIMEApplicationJSON)

	var in profileDTO.UpdateUsersProfileRequest
	if isJSON {
		if err := c.BodyParser(&in); err != nil {
			log.Println("[ERROR] Invalid request body:", err)
			return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request format")
		}
	} else {
		setPtr := func(dst **string, name string) {
			if v := c.FormValue(name); v != "" {
				val := v
				*dst = &val
			}
		}
		setPtr(&in.UserProfileSlug, "user_profile_slug")
		setPtr(&in.UsersProfileDonationName, "users_profile_donation_name")
		setPtr(&in.UsersProfileDateOfBirth, "users_profile_date_of_birth")
		setPtr(&in.UserProfilePlaceOfBirth, "user_profile_place_of_birth")
		setPtr(&in.UsersProfileGender, "users_profile_gender")
		setPtr(&in.UsersProfileLocation, "users_profile_location")
		setPtr(&in.UsersProfileCity, "users_profile_city")
		setPtr(&in.UsersProfilePhoneNumber, "users_profile_phone_number")
		setPtr(&in.UsersProfileBio, "users_profile_bio")
		setPtr(&in.UsersProfileBiographyLong, "users_profile_biography_long")
		setPtr(&in.UsersProfileExperience, "users_profile_experience")
		setPtr(&in.UsersProfileCertifications, "users_profile_certifications")
		setPtr(&in.UsersProfileInstagramURL, "users_profile_instagram_url")
		setPtr(&in.UsersProfileWhatsappURL, "users_profile_whatsapp_url")
		setPtr(&in.UsersProfileLinkedinURL, "users_profile_linkedin_url")
		setPtr(&in.UsersProfileGithubURL, "users_profile_github_url")
		setPtr(&in.UsersProfileYoutubeURL, "users_profile_youtube_url")
		setPtr(&in.UserProfileTelegramUsername, "user_profile_telegram_username")
		if v := c.FormValue("users_profile_is_public_profile"); v != "" {
			val := strings.ToLower(strings.TrimSpace(v))
			in.UsersProfileIsPublicProfile = parseBoolPtr(val)
		}
		if v := c.FormValue("users_profile_is_verified"); v != "" {
			val := strings.ToLower(strings.TrimSpace(v))
			in.UsersProfileIsVerified = parseBoolPtr(val)
		}
		setPtr(&in.UsersProfileVerifiedAt, "users_profile_verified_at")
		if v := c.FormValue("users_profile_verified_by"); v != "" {
			vv := strings.TrimSpace(v)
			in.UsersProfileVerifiedBy = parseUUIDPtr(vv)
		}
	}

	if in.UsersProfilePhoneNumber != nil {
		v := strings.TrimSpace(*in.UsersProfilePhoneNumber)
		v = strings.ReplaceAll(v, " ", "")
		in.UsersProfilePhoneNumber = &v
	}

	updateMap, mapErr := in.ToUpdateMap()
	if mapErr != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, mapErr.Error())
	}
	if len(updateMap) == 0 {
		return helper.JsonOK(c, "No changes", profileDTO.ToUsersProfileDTO(profile))
	}

	if err := upc.DB.WithContext(c.Context()).
		Model(&profileModel.UserProfileModel{}).
		Where("user_profile_user_id = ?", userID).
		Updates(updateMap).Error; err != nil {
		log.Println("[ERROR] Failed to patch user_profile:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update user profile")
	}

	if err := upc.DB.WithContext(c.Context()).
		Where("user_profile_user_id = ?", userID).
		First(&profile).Error; err != nil {
		log.Println("[WARN] Refresh after patch failed:", err)
	}
	return helper.JsonUpdated(c, "User profile updated", profileDTO.ToUsersProfileDTO(profile))
}

/* =========================
   DELETE: Soft delete
   ========================= */
func (upc *UsersProfileController) DeleteProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil { return httpErr(c, err) }
	log.Println("[INFO] Soft deleting user_profile with user_profile_user_id:", userID)

	var profile profileModel.UserProfileModel
	if err := upc.DB.WithContext(c.Context()).
		Where("user_profile_user_id = ?", userID).
		First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "User profile not found")
		}
		log.Println("[ERROR] DB error:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch user profile")
	}

	if err := upc.DB.WithContext(c.Context()).Delete(&profile).Error; err != nil {
		log.Println("[ERROR] Failed to delete user_profile:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete user profile")
	}
	return helper.JsonDeleted(c, "User profile deleted", nil)
}

// =============================================================
//  Upload avatar/cover memakai OSSService helper terbaru
// =============================================================

// POST /api/a/users/profile/avatar  (multipart: image)
func (upc *UsersProfileController) UploadAvatar(c *fiber.Ctx) error {
	if upc.OSS == nil {
		return helper.JsonError(c, fiber.StatusServiceUnavailable, "OSS belum dikonfigurasi")
	}

	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil { return httpErr(c, err) }

	masjidID, err := getMasjidIDFromCtx(c)
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, err.Error()) }

	fh, err := getImageFormFile(c)
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, err.Error()) }

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	url, err := helperOSS.UploadImageToOSS(ctx, upc.OSS, masjidID, "user-avatar", fh)
	if err != nil { return httpErr(c, err) }

	// Optional: simpan ke kolom avatar_url kalau ada
	if err := upc.DB.WithContext(c.Context()).
		Model(&profileModel.UserProfileModel{}).
		Where("user_profile_user_id = ?", userID).
		Update("user_profile_avatar_url", url).Error; err != nil {
		log.Println("[WARN] avatar_url update skipped/failed:", err)
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"url": url})
}

// POST /api/a/users/profile/cover  (multipart: image)
func (upc *UsersProfileController) UploadCover(c *fiber.Ctx) error {
	if upc.OSS == nil {
		return helper.JsonError(c, fiber.StatusServiceUnavailable, "OSS belum dikonfigurasi")
	}

	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil { return httpErr(c, err) }

	masjidID, err := getMasjidIDFromCtx(c)
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, err.Error()) }

	fh, err := getImageFormFile(c)
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, err.Error()) }

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	url, err := helperOSS.UploadImageToOSS(ctx, upc.OSS, masjidID, "user-cover", fh)
	if err != nil { return httpErr(c, err) }

	if err := upc.DB.WithContext(c.Context()).
		Model(&profileModel.UserProfileModel{}).
		Where("user_profile_user_id = ?", userID).
		Update("user_profile_cover_url", url).Error; err != nil {
		log.Println("[WARN] cover_url update skipped/failed:", err)
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"url": url})
}

// Helper: ambil masjid_id dari header/form/token
func getMasjidIDFromCtx(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) Header umum
	if v := strings.TrimSpace(c.Get("X-Masjid-Id")); v != "" {
		if id, err := uuid.Parse(v); err == nil { return id, nil }
	}
	// 2) Form value fallback
	if v := strings.TrimSpace(c.FormValue("masjid_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil { return id, nil }
	}
	// 3) Dari token
	if id, err := helperAuth.GetMasjidIDFromToken(c); err == nil && id != uuid.Nil {
		return id, nil
	}
	return uuid.Nil, errors.New("masjid_id tidak ditemukan pada header/form/token")
}

// Helper: cari file dari beberapa nama field umum
func getImageFormFile(c *fiber.Ctx) (*multipart.FileHeader, error) {
	names := []string{"image", "file", "photo", "picture", "avatar", "cover"}
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil {
			return fh, nil
		}
	}
	return nil, errors.New("Gambar tidak ditemukan (field: image/file/photo/picture/avatar/cover)")
}

func parseBoolPtr(s string) *bool {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "y", "on":
		b := true
		return &b
	case "false", "0", "no", "n", "off":
		b := false
		return &b
	default:
		return nil
	}
}

func parseUUIDPtr(s string) *uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}
