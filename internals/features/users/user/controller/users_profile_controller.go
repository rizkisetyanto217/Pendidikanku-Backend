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

	// Jika multipart + ada avatar, upload sekalian dan isi kolom avatar (tanpa old karena belum ada)
	if isMultipart {
		masjidID, merr := getMasjidIDFromCtx(c)
		if merr == nil {
			if fh, err := getImageFormFile(c); err == nil && fh != nil {
				svc := upc.OSS
				if svc == nil {
					tmp, err := helperOSS.NewOSSServiceFromEnv("")
					if err != nil {
						return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
					}
					svc = tmp
				}
				ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
				defer cancel()

				url, upErr := helperOSS.UploadImageToOSS(ctx, svc, masjidID, "user-avatar", fh)
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
	var in profileDTO.UpdateUsersProfileRequest
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

	// ==== handle avatar (optional) via multipart ====
	if isMultipart {
		// OSS service
		svc := upc.OSS
		if svc == nil {
			tmp, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "OSS belum terkonfigurasi")
			}
			svc = tmp
		}

		// masjid scope wajib saat upload
		masjidID, merr := getMasjidIDFromCtx(c)
		if merr == nil {
			if fh, err := getImageFormFile(c); err == nil && fh != nil {
				ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
				defer cancel()

				url, upErr := helperOSS.UploadImageToOSS(ctx, svc, masjidID, "user-avatar", fh)
				if upErr != nil {
					return httpErr(c, upErr)
				}
				key, kerr := helperOSS.KeyFromPublicURL(url)
				if kerr != nil {
					return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (avatar)")
				}

				// 2-slot; due ditentukan oleh helper OSS (satu pintu)
				if before.UserProfileAvatarURL != nil && *before.UserProfileAvatarURL != "" {
					due := now.Add(helperOSS.GetRetentionDuration()) // ← ambil dari helperOSS
					updateMap["user_profile_avatar_url_old"] = before.UserProfileAvatarURL
					updateMap["user_profile_avatar_object_key_old"] = before.UserProfileAvatarObjectKey
					updateMap["user_profile_avatar_delete_pending_until"] = &due
				}

				updateMap["user_profile_avatar_url"] = url
				updateMap["user_profile_avatar_object_key"] = key
			}
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
