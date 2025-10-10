// internals/features/users/user/controller/users_profile_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
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

func (upc *UsersProfileController) ensureOSS() (*helperOSS.OSSService, error) {
	if upc.OSS != nil {
		return upc.OSS, nil
	}
	svc, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil {
		return nil, err
	}
	upc.OSS = svc
	return svc, nil
}

// =============================================================
// UsersProfileController (versi pakai OSSService helper terbaru)
// =============================================================

type UsersProfileController struct {
	DB  *gorm.DB
	OSS *helperOSS.OSSService
}

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
	return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
}

func formJSONOrCSVToSlice(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var arr []string
	if json.Unmarshal([]byte(s), &arr) == nil {
		return profileDTO.CompactStrings(arr)
	}
	parts := strings.Split(s, ",")
	return profileDTO.CompactStrings(parts)
}

/*
=========================

	GET: All profiles (DTO)
	=========================
*/
func (upc *UsersProfileController) GetProfiles(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all user_profile")
	var profiles []profileModel.UserProfileModel
	if err := upc.DB.WithContext(c.Context()).Find(&profiles).Error; err != nil {
		log.Println("[ERROR] Failed to fetch user_profile:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch profiles")
	}
	return helper.JsonOK(c, "Profiles fetched", profileDTO.ToUsersProfileDTOs(profiles))
}

/*
=========================

	GET: My profile (DTO)
	=========================
*/
/*
=========================

	GET: My profile atau by :user_id (DTO)
	=========================
*/
func (upc *UsersProfileController) GetProfile(c *fiber.Ctx) error {
	// 1️⃣ Cek apakah ada param user_id
	userIDStr := strings.TrimSpace(c.Params("user_id"))
	var userID uuid.UUID
	var err error

	if userIDStr != "" {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "user_id tidak valid (harus UUID)")
		}
		log.Println("[INFO] Fetching user_profile by param user_id:", userID)
	} else {
		// fallback ke token kalau param kosong
		userID, err = helperAuth.GetUserIDFromToken(c)
		if err != nil {
			return httpErr(c, err)
		}
		log.Println("[INFO] Fetching user_profile (from token):", userID)
	}

	// 2️⃣ Query ke DB
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

	// 3️⃣ Response DTO
	return helper.JsonOK(c, "Profile fetched", profileDTO.ToUsersProfileDTO(profile))
}

/*
=========================

	POST /profiles (Create) — support JSON atau multipart (payload+avatar)
	=========================
*/
func (upc *UsersProfileController) CreateProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return httpErr(c, err)
	}

	var in profileDTO.CreateUsersProfileRequest

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, "multipart/form-data")

	if isMultipart {
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			if err := c.App().Config().JSONDecoder([]byte(s), &in); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Invalid payload JSON")
			}
		} else if err := c.BodyParser(&in); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Invalid multipart form")
		}
	} else {
		if err := c.BodyParser(&in); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request format")
		}
	}

	row := in.ToModel(userID)

	// Jika multipart + ada avatar, upload sekalian (scope: USER)
	if isMultipart {
		if fh, err := getImageFormFile(c); err == nil && fh != nil {
			svc, err := upc.ensureOSS()
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
			}
			ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
			defer cancel()

			// Pakai userID sebagai scope (tidak butuh masjid)
			url, upErr := helperOSS.UploadImageToOSS(ctx, svc, userID, "user-avatar", fh)
			if upErr != nil {
				return httpErr(c, upErr)
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (avatar)")
			}

			row.UserProfileAvatarURL = &url
			row.UserProfileAvatarObjectKey = &key
		}
	}

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

/*
=========================

	PATCH /profiles — multipart ala Masjid (payload JSON + avatar)
	=========================
*/
func (upc *UsersProfileController) UpdateProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return httpErr(c, err)
	}
	log.Println("[INFO] Patching user_profile with user_profile_user_id:", userID)

	// Ambil row existing
	var before profileModel.UserProfileModel
	if err := upc.DB.WithContext(c.Context()).
		Where("user_profile_user_id = ?", userID).
		First(&before).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "User profile not found")
		}
		log.Println("[ERROR] DB error:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch user profile")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, "multipart/form-data")

	// UPDATE: multipart payload parsing
	// UPDATE: multipart payload parsing
	var in profileDTO.UpdateUsersProfileRequest
	if isMultipart {
		if s := strings.TrimSpace(c.FormValue("payload")); s != "" {
			if err := c.App().Config().JSONDecoder([]byte(s), &in); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Invalid payload JSON")
			}
		} else {
			// Coba bind form-data → struct
			if err := c.BodyParser(&in); err != nil {
				log.Println("[WARN] multipart BodyParser error (will fallback):", err)
				// jangan langsung return 400; lanjut fallback manual di bawah
			}

			// ===== Fallback hydrator untuk field yang sering gagal di multipart =====

			// Arrays: izinkan JSON string, CSV, atau key berulang (users_profile_interests=.. (multi))
			if in.UsersProfileInterests == nil {
				if v := strings.TrimSpace(c.FormValue("users_profile_interests")); v != "" {
					in.UsersProfileInterests = formJSONOrCSVToSlice(v)
				} else if vals := c.FormValue("users_profile_interests[]"); vals != "" {
					in.UsersProfileInterests = formJSONOrCSVToSlice(vals)
				}
			}
			if in.UsersProfileSkills == nil {
				if v := strings.TrimSpace(c.FormValue("users_profile_skills")); v != "" {
					in.UsersProfileSkills = formJSONOrCSVToSlice(v)
				} else if vals := c.FormValue("users_profile_skills[]"); vals != "" {
					in.UsersProfileSkills = formJSONOrCSVToSlice(vals)
				}
			}

			// Booleans
			if in.UsersProfileIsPublicProfile == nil {
				if v := strings.TrimSpace(c.FormValue("users_profile_is_public_profile")); v != "" {
					if b, err := strconv.ParseBool(v); err == nil {
						in.UsersProfileIsPublicProfile = &b
					}
				}
			}
			if in.UsersProfileIsVerified == nil {
				if v := strings.TrimSpace(c.FormValue("users_profile_is_verified")); v != "" {
					if b, err := strconv.ParseBool(v); err == nil {
						in.UsersProfileIsVerified = &b
					}
				}
			}

			// Timestamps / Date (biarkan ToUpdateMap yang validasi format)
			if in.UsersProfileVerifiedAt == nil {
				if v := strings.TrimSpace(c.FormValue("users_profile_verified_at")); v != "" {
					in.UsersProfileVerifiedAt = &v
				}
			}
			if in.UsersProfileDateOfBirth == nil {
				if v := strings.TrimSpace(c.FormValue("users_profile_date_of_birth")); v != "" {
					in.UsersProfileDateOfBirth = &v
				}
			}

			// Strings lain (kalau BodyParser gagal karena perbedaan driver)
			setIfEmpty := func(dst **string, key string) {
				if *dst == nil {
					if v := strings.TrimSpace(c.FormValue(key)); v != "" {
						*dst = &v
					}
				}
			}
			setIfEmpty(&in.UserProfileSlug, "user_profile_slug")
			setIfEmpty(&in.UsersProfileDonationName, "users_profile_donation_name")
			setIfEmpty(&in.UserProfilePlaceOfBirth, "user_profile_place_of_birth")
			setIfEmpty(&in.UsersProfileGender, "users_profile_gender")
			setIfEmpty(&in.UsersProfileLocation, "users_profile_location")
			setIfEmpty(&in.UsersProfileCity, "users_profile_city")
			setIfEmpty(&in.UsersProfileBio, "users_profile_bio")
			setIfEmpty(&in.UsersProfileBiographyLong, "users_profile_biography_long")
			setIfEmpty(&in.UsersProfileExperience, "users_profile_experience")
			setIfEmpty(&in.UsersProfileCertifications, "users_profile_certifications")
			setIfEmpty(&in.UsersProfileInstagramURL, "users_profile_instagram_url")
			setIfEmpty(&in.UsersProfileWhatsappURL, "users_profile_whatsapp_url")
			setIfEmpty(&in.UsersProfileLinkedinURL, "users_profile_linkedin_url")
			setIfEmpty(&in.UsersProfileGithubURL, "users_profile_github_url")
			setIfEmpty(&in.UsersProfileYoutubeURL, "users_profile_youtube_url")
			setIfEmpty(&in.UserProfileTelegramUsername, "user_profile_telegram_username")
			setIfEmpty(&in.UserProfileParentName, "user_profile_parent_name")
			setIfEmpty(&in.UserProfileParentWhatsappURL, "user_profile_parent_whatsapp_url")
			setIfEmpty(&in.UsersProfileEducation, "users_profile_education")
			setIfEmpty(&in.UsersProfileCompany, "users_profile_company")
			setIfEmpty(&in.UsersProfilePosition, "users_profile_position")
		}
	} else {
		if err := c.BodyParser(&in); err != nil {
			log.Println("[ERROR] Invalid request body:", err)
			return helper.JsonError(c, fiber.StatusBadRequest, "Invalid request format")
		}
	}

	updateMap, mapErr := in.ToUpdateMap()
	if mapErr != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, mapErr.Error())
	}

	now := time.Now()
	updateMap["user_profile_updated_at"] = now

	// ==== handle avatar (optional) via multipart, scope: USER ====
	if isMultipart {
		svc, err := upc.ensureOSS()
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
		}

		if fh, err := getImageFormFile(c); err == nil && fh != nil {
			ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
			defer cancel()

			url, upErr := helperOSS.UploadImageToOSS(ctx, svc, userID, "user-avatar", fh)
			if upErr != nil {
				return httpErr(c, upErr)
			}
			key, kerr := helperOSS.KeyFromPublicURL(url)
			if kerr != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (avatar)")
			}

			// 2-slot (old → pending delete)
			if before.UserProfileAvatarURL != nil && *before.UserProfileAvatarURL != "" {
				due := now.Add(helperOSS.GetRetentionDuration())
				updateMap["user_profile_avatar_url_old"] = before.UserProfileAvatarURL
				updateMap["user_profile_avatar_object_key_old"] = before.UserProfileAvatarObjectKey
				updateMap["user_profile_avatar_delete_pending_until"] = &due
			}

			updateMap["user_profile_avatar_url"] = url
			updateMap["user_profile_avatar_object_key"] = key
		}
	}

	if len(updateMap) == 1 { // cuma updated_at
		return helper.JsonOK(c, "No changes", profileDTO.ToUsersProfileDTO(before))
	}

	// Apply ke DB
	if err := upc.DB.WithContext(c.Context()).
		Model(&profileModel.UserProfileModel{}).
		Where("user_profile_user_id = ?", userID).
		Updates(updateMap).Error; err != nil {
		log.Println("[ERROR] Failed to patch user_profile:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update user profile")
	}

	// Refresh
	var after profileModel.UserProfileModel
	if err := upc.DB.WithContext(c.Context()).
		Where("user_profile_user_id = ?", userID).
		First(&after).Error; err != nil {
		log.Println("[WARN] Refresh after patch failed:", err)
		return helper.JsonUpdated(c, "User profile updated (no refresh)", profileDTO.ToUsersProfileDTO(before))
	}
	return helper.JsonUpdated(c, "User profile updated", profileDTO.ToUsersProfileDTO(after))
}

/*
=========================

	DELETE: Soft delete
	=========================
*/
func (upc *UsersProfileController) DeleteProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return httpErr(c, err)
	}
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
//  Upload avatar (legacy) — masih disediakan, tapi PATCH multipart sudah cukup
// =============================================================

// POST /api/a/users/profile/avatar  (multipart: image)
func (upc *UsersProfileController) UploadAvatar(c *fiber.Ctx) error {
	if upc.OSS == nil {
		return helper.JsonError(c, fiber.StatusServiceUnavailable, "OSS belum dikonfigurasi")
	}

	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return httpErr(c, err)
	}

	masjidID, err := getMasjidIDFromCtx(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	fh, err := getImageFormFile(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	url, err := helperOSS.UploadImageToOSS(ctx, upc.OSS, masjidID, "user-avatar", fh)
	if err != nil {
		return httpErr(c, err)
	}

	// Simple set (tanpa 2-slot di endpoint legacy ini)
	if err := upc.DB.WithContext(c.Context()).
		Model(&profileModel.UserProfileModel{}).
		Where("user_profile_user_id = ?", userID).
		Update("user_profile_avatar_url", url).Error; err != nil {
		log.Println("[WARN] avatar_url update skipped/failed:", err)
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"url": url})
}

// =============================================================
// Helpers (scope & file)
// =============================================================

func getMasjidIDFromCtx(c *fiber.Ctx) (uuid.UUID, error) {
	// 1) Header umum
	if v := strings.TrimSpace(c.Get("X-Masjid-Id")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			return id, nil
		}
	}
	// 2) Form value fallback
	if v := strings.TrimSpace(c.FormValue("masjid_id")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			return id, nil
		}
	}
	// 3) Dari token
	if id, err := helperAuth.GetMasjidIDFromToken(c); err == nil && id != uuid.Nil {
		return id, nil
	}
	return uuid.Nil, errors.New("masjid_id tidak ditemukan pada header/form/token")
}

func getImageFormFile(c *fiber.Ctx) (*multipart.FileHeader, error) {
	names := []string{"avatar", "image", "file", "photo", "picture"}
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil {
			return fh, nil
		}
	}
	return nil, errors.New("gambar tidak ditemukan")
}
