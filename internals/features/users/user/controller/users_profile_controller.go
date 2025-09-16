package controller

import (
	"context"
	"errors"
	"log"
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
// UsersProfileController (versi pakai BlobService)
// =============================================================

type UsersProfileController struct {
	DB   *gorm.DB
	Blob helperOSS.BlobService
}

func NewUsersProfileController(db *gorm.DB, blob helperOSS.BlobService) *UsersProfileController {
	return &UsersProfileController{DB: db, Blob: blob}
}

func httpErr(c *fiber.Ctx, err error) error {
	if fe, ok := err.(*fiber.Error); ok {
		return helper.JsonError(c, fe.Code, fe.Message)
	}
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
   PATCH /profiles (partial update)
   - Mendukung JSON dan form-urlencoded/multipart TANPA file
   - Upload avatar/cover gunakan endpoint terpisah (lihat bawah)
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
//  Upload avatar/cover memakai BlobService
//  (endpoint terpisah agar concern update teks vs file jelas)
// =============================================================

// POST /api/a/users/profile/avatar  (multipart: image)
// Header/Retrieval masjid_id: ambil dari header X-Masjid-Id atau form masjid_id (fallback)
func (upc *UsersProfileController) UploadAvatar(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil { return httpErr(c, err) }

	masjidID, err := getMasjidIDFromCtx(c)
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, err.Error()) }

	fh, err := helperOSS.TryGetImageFile(c)
	if err != nil || fh == nil { return helper.JsonError(c, fiber.StatusBadRequest, "Gambar tidak ditemukan (field: image/file/photo/picture)") }

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	url, err := upc.Blob.UploadImage(ctx, masjidID, "user-avatar", fh)
	if err != nil { return httpErr(c, err) }

	// Optional: simpan ke kolom avatar_url kalau ada (tanpa hard dependency)
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
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil { return httpErr(c, err) }

	masjidID, err := getMasjidIDFromCtx(c)
	if err != nil { return helper.JsonError(c, fiber.StatusBadRequest, err.Error()) }

	fh, err := helperOSS.TryGetImageFile(c)
	if err != nil || fh == nil { return helper.JsonError(c, fiber.StatusBadRequest, "Gambar tidak ditemukan (field: image/file/photo/picture)") }

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	url, err := upc.Blob.UploadImage(ctx, masjidID, "user-cover", fh)
	if err != nil { return httpErr(c, err) }

	if err := upc.DB.WithContext(c.Context()).
		Model(&profileModel.UserProfileModel{}).
		Where("user_profile_user_id = ?", userID).
		Update("user_profile_cover_url", url).Error; err != nil {
		log.Println("[WARN] cover_url update skipped/failed:", err)
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"url": url})
}

// Helper: cari masjid_id dari header atau form
func getMasjidIDFromCtx(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) Header umum yang sering kamu pakai
	if v := strings.TrimSpace(c.Get("X-Masjid-Id")); v != "" {
		if id, err := uuid.Parse(v); err == nil { return id, nil }
	}
	// 2) Form value fallback
	if v := strings.TrimSpace(c.FormValue("masjid_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil { return id, nil }
	}
	// 3) (Opsional) dari token jika ada helper-nya
	if id, err := helperAuth.GetMasjidIDFromToken(c); err == nil && id != uuid.Nil {
		return id, nil
	}
	return uuid.Nil, errors.New("masjid_id tidak ditemukan pada header/form/token")
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
