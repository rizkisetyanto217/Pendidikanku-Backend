package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	profileDTO "masjidku_backend/internals/features/users/user/dto"
	profileModel "masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UsersProfileController struct {
	DB *gorm.DB
}

func NewUsersProfileController(db *gorm.DB) *UsersProfileController {
	return &UsersProfileController{DB: db}
}

func httpErr(c *fiber.Ctx, err error) error {
	if fe, ok := err.(*fiber.Error); ok {
		return helper.Error(c, fe.Code, fe.Message)
	}
	return helper.Error(c, fiber.StatusUnauthorized, err.Error())
}

/* =========================
   GET: All profiles (DTO)
   ========================= */
func (upc *UsersProfileController) GetProfiles(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all user profiles")
	var profiles []profileModel.UsersProfileModel
	if err := upc.DB.Find(&profiles).Error; err != nil {
		log.Println("[ERROR] Failed to fetch user profiles:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to fetch user profiles")
	}
	return helper.Success(c, "User profiles fetched successfully", profileDTO.ToUsersProfileDTOs(profiles))
}

/* =========================
   GET: My profile (DTO)
   ========================= */
func (upc *UsersProfileController) GetProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return httpErr(c, err)
	}
	log.Println("[INFO] Fetching user profile with user_id:", userID)

	var profile profileModel.UsersProfileModel
	if err := upc.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.Error(c, fiber.StatusNotFound, "User profile not found")
		}
		log.Println("[ERROR] DB error:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to fetch user profile")
	}
	return helper.Success(c, "User profile fetched successfully", profileDTO.ToUsersProfileDTO(profile))
}

/* =========================
   POST: Create / Upsert (DTO)
   - Buat jika belum ada; kalau sudah ada, update dari payload create
   ========================= */
// =========================
// POST /profiles (Create only)
// =========================
func (upc *UsersProfileController) CreateProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return httpErr(c, err)
	}

	// Parse body (JSON/form) ke DTO kamu
	var in profileDTO.CreateUsersProfileRequest
	if err := c.BodyParser(&in); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request format")
	}
	// Validasi ringan contoh (silakan sesuaikan DTO/validator kamu)
	if strings.TrimSpace(in.DonationName) == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Donation name is required")
	}
	if in.PhoneNumber != nil {
		v := strings.TrimSpace(*in.PhoneNumber)
		v = strings.ReplaceAll(v, " ", "")
		in.PhoneNumber = &v
	}

	// (opsional) upload avatar multipart field "photo" → WebP → OSS
	var uploadedAvatarURL *string
	if fh, errFile := c.FormFile("photo"); errFile == nil && fh != nil {
		// Batasi ukuran & tipe dasar
		if fh.Size > 5*1024*1024 {
			return helper.Error(c, fiber.StatusRequestEntityTooLarge, "Max file size 5MB")
		}

		svc, errInit := helperOSS.NewOSSServiceFromEnv("")
		if errInit != nil {
			return helper.Error(c, fiber.StatusInternalServerError, "Failed to init file service")
		}

		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()

		dir := fmt.Sprintf("users/%s/avatar", userID.String())
		publicURL, errUp := svc.UploadAsWebP(ctx, fh, dir)
		if errUp != nil {
			if strings.Contains(strings.ToLower(errUp.Error()), "format tidak didukung") {
				return helper.Error(c, fiber.StatusUnsupportedMediaType, "Unsupported image format (use jpg/png/webp)")
			}
			return helper.Error(c, fiber.StatusBadGateway, "Failed to upload avatar")
		}
		uploadedAvatarURL = &publicURL
		in.PhotoURL = uploadedAvatarURL
	}

	// Build model dari DTO
	row := in.ToModel(userID)

	// Gunakan unique index di DB utk cegah duplikasi secara race-safe
	db := upc.DB.WithContext(c.Context())
	if err := db.Create(&row).Error; err != nil {
		// Rollback avatar jika DB gagal
		if uploadedAvatarURL != nil {
			_ = helperOSS.DeleteByPublicURLENV(*uploadedAvatarURL, 10*time.Second)
		}

		// Tangkap duplicate key (unique constraint)
		le := strings.ToLower(err.Error())
		if strings.Contains(le, "duplicate key") || strings.Contains(le, "unique constraint") {
			return helper.Error(c, fiber.StatusConflict, "User profile already exists")
		}
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to create user profile")
	}

	return helper.SuccessWithCode(c, fiber.StatusCreated, "User profile created successfully",
		profileDTO.ToUsersProfileDTO(row))
}



// =========================
// PATCH /profiles (partial update + optional avatar replace)
// =========================
func (upc *UsersProfileController) UpdateProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return httpErr(c, err)
	}
	log.Println("[INFO] Updating user profile with user_id:", userID)

	// --- Ambil profile existing
	var profile profileModel.UsersProfileModel
	if err := upc.DB.WithContext(c.Context()).
		Where("user_id = ?", userID).
		First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.Error(c, fiber.StatusNotFound, "User profile not found")
		}
		log.Println("[ERROR] DB error:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to fetch user profile")
	}

	ct := strings.TrimSpace(c.Get(fiber.HeaderContentType))
	isJSON := strings.HasPrefix(ct, fiber.MIMEApplicationJSON)
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	// --- Kumpulkan perubahan dari body
	var in profileDTO.UpdateUsersProfileRequest
	if isJSON {
		if err := c.BodyParser(&in); err != nil {
			log.Println("[ERROR] Invalid request body:", err)
			return helper.Error(c, fiber.StatusBadRequest, "Invalid request format")
		}
	} else {
		// form-urlencoded / multipart -> isi pointer kalau ada
		setPtr := func(dst **string, name string) {
			if v := c.FormValue(name); v != "" {
				val := v
				*dst = &val
			}
		}
		setPtr(&in.DonationName, "donation_name")
		setPtr(&in.PhotoURL, "photo_url") // opsional: bila mau set URL langsung
		setPtr(&in.PhotoTrashURL, "photo_trash_url")
		setPtr(&in.PhotoDeletePendingUntil, "photo_delete_pending_until")
		setPtr(&in.DateOfBirth, "date_of_birth")
		setPtr(&in.Gender, "gender")
		setPtr(&in.Location, "location")
		setPtr(&in.Occupation, "occupation")
		setPtr(&in.PhoneNumber, "phone_number")
		setPtr(&in.Bio, "bio")
	}

	updateMap, mapErr := in.ToUpdateMap()
	if mapErr != nil {
		return helper.Error(c, fiber.StatusBadRequest, mapErr.Error())
	}

	// =========================
	// Upload avatar (jika ada file) -> pakai helper OSS
	// =========================
	var (
		newAvatarURL string  // url avatar baru
		hadNewFile   bool
	)

	if isMultipart {
		// terima file dengan field "photo" (utama) atau fallback "photo_url"
		fh, errFile := c.FormFile("photo")
		if errFile != nil || fh == nil {
			fh, _ = c.FormFile("photo_url")
		}
		if fh != nil {
			if fh.Size > 5*1024*1024 {
				return helper.Error(c, fiber.StatusRequestEntityTooLarge, "Max file size 5MB")
			}

			ossSvc, errInit := helperOSS.NewOSSServiceFromEnv("") // asumsi sudah ada
			if errInit != nil {
				return helper.Error(c, fiber.StatusInternalServerError, "Failed to init file service")
			}

			ctxUp, cancelUp := context.WithTimeout(c.Context(), 30*time.Second)
			defer cancelUp()

			dir := fmt.Sprintf("users/%s/avatar", userID.String())
			publicURL, errUp := ossSvc.UploadAsWebP(ctxUp, fh, dir)
			if errUp != nil {
				if strings.Contains(strings.ToLower(errUp.Error()), "format tidak didukung") {
					return helper.Error(c, fiber.StatusUnsupportedMediaType, "Unsupported image format (use jpg/png/webp)")
				}
				return helper.Error(c, fiber.StatusBadGateway, "Failed to upload avatar")
			}

			newAvatarURL = publicURL
			hadNewFile = true
			updateMap["photo_url"] = newAvatarURL
		}
	}

	// =========================
	// Transaksi DB (update kolom)
	// =========================
	tx := upc.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		// kalau upload sukses tapi transaksi gagal start -> hapus file baru
		if hadNewFile {
			_ = helperOSS.DeleteByPublicURLENV(newAvatarURL, 10*time.Second)
		}
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to start transaction")
	}

	if len(updateMap) > 0 {
		if err := tx.Model(&profile).Where("user_id = ?", userID).Updates(updateMap).Error; err != nil {
			_ = tx.Rollback().Error
			if hadNewFile {
				_ = helperOSS.DeleteByPublicURLENV(newAvatarURL, 10*time.Second)
			}
			log.Println("[ERROR] Failed to update user profile:", err)
			return helper.Error(c, fiber.StatusInternalServerError, "Failed to update user profile")
		}
	}

	if err := tx.Commit().Error; err != nil {
		if hadNewFile {
			_ = helperOSS.DeleteByPublicURLENV(newAvatarURL, 10*time.Second)
		}
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to commit update")
	}

	// =========================
	// Post-commit: pindahkan foto lama ke spam + set metadata trash
	// (dilakukan setelah commit supaya konsisten)
	// =========================
	if hadNewFile {
		if profile.PhotoURL != nil && strings.TrimSpace(*profile.PhotoURL) != "" {
			oldURL := strings.TrimSpace(*profile.PhotoURL)

			// NOTE: pada titik ini, variable profile masih nilai PRE-UPDATE.
			// kita pakai oldURL dari pre-update untuk dipindah ke spam.
			spamURL, errMove := helperOSS.MoveToSpamByPublicURLENV(oldURL, 15*time.Second)
			if errMove != nil {
				log.Printf("[WARN] Move old avatar to spam failed: %v", errMove)
			} else {
				// tulis metadata trash ke DB (best-effort)
				_ = upc.DB.WithContext(c.Context()).
					Model(&profileModel.UsersProfileModel{}).
					Where("user_id = ?", userID).
					Updates(map[string]interface{}{
						"photo_trash_url":           spamURL,
						"photo_delete_pending_until": time.Now().Add(7 * 24 * time.Hour).UTC(),
					}).Error
			}
		}
	}

	// Refresh respons
	if err := upc.DB.WithContext(c.Context()).
		Where("user_id = ?", userID).First(&profile).Error; err != nil {
		log.Println("[WARN] Refresh after update failed:", err)
	}

	return helper.Success(c, "User profile updated successfully", profileDTO.ToUsersProfileDTO(profile))
}



/* =========================
   DELETE: Soft delete
   ========================= */
func (upc *UsersProfileController) DeleteProfile(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return httpErr(c, err)
	}
	log.Println("[INFO] Deleting user profile with user_id:", userID)

	var profile profileModel.UsersProfileModel
	if err := upc.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.Error(c, fiber.StatusNotFound, "User profile not found")
		}
		log.Println("[ERROR] DB error:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to fetch user profile")
	}

	if err := upc.DB.Delete(&profile).Error; err != nil {
		log.Println("[ERROR] Failed to delete user profile:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to delete user profile")
	}

	return helper.Success(c, "User profile deleted successfully", nil)
}
