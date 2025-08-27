package controller

import (
	"log"
	"strings"
	"time"

	uModel "masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ===== Request DTOs (lebih aman dari langsung pakai model) ===== */

type createProfileReq struct {
	DonationName string     `json:"donation_name"`
	FatherName   string     `json:"father_name"`
	MotherName   string     `json:"mother_name"`
	DateOfBirth  *string    `json:"date_of_birth"` // "2006-01-02"
	Gender       *string    `json:"gender"`        // "male" | "female"
	PhoneNumber  string     `json:"phone_number"`
	Bio          string     `json:"bio"`
	Location     string     `json:"location"`
	Occupation   string     `json:"occupation"`
}

type updateProfileReq struct {
	DonationName *string `json:"donation_name"`
	FatherName   *string `json:"father_name"`
	MotherName   *string `json:"mother_name"`
	DateOfBirth  *string `json:"date_of_birth"` // "2006-01-02" | ""
	Gender       *string `json:"gender"`        // "male" | "female" | ""
	PhoneNumber  *string `json:"phone_number"`
	Bio          *string `json:"bio"`
	Location     *string `json:"location"`
	Occupation   *string `json:"occupation"`
}

func parseDOB(s *string) *time.Time {
	if s == nil {
		return nil
	}
	ss := strings.TrimSpace(*s)
	if ss == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", ss)
	if err != nil {
		return nil
	}
	return &t
}

func parseGender(s *string) *uModel.Gender {
	if s == nil {
		return nil
	}
	val := uModel.Gender(strings.ToLower(strings.TrimSpace(*s)))
	if val != uModel.Male && val != uModel.Female {
		return nil
	}
	return &val
}

/* ================= Controller ================= */

type UsersProfileController struct {
	DB *gorm.DB
}

func NewUsersProfileController(db *gorm.DB) *UsersProfileController {
	return &UsersProfileController{DB: db}
}

func (upc *UsersProfileController) GetProfiles(c *fiber.Ctx) error {
	log.Println("[INFO] Fetching all user profiles")
	var profiles []uModel.UsersProfileModel
	if err := upc.DB.Find(&profiles).Error; err != nil {
		log.Println("[ERROR] Failed to fetch user profiles:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to fetch user profiles")
	}
	return helper.Success(c, "User profiles fetched successfully", profiles)
}

func (upc *UsersProfileController) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	log.Println("[INFO] Fetching user profile with user_id:", userID)

	var profile uModel.UsersProfileModel
	if err := upc.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.Error(c, fiber.StatusNotFound, "User profile not found")
		}
		log.Println("[ERROR] DB error:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to fetch user profile")
	}
	return helper.Success(c, "User profile fetched successfully", profile)
}

func (upc *UsersProfileController) CreateProfile(c *fiber.Ctx) error {
	log.Println("[INFO] Creating or updating user profile")

	uid := c.Locals("user_id")
	if uid == nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Unauthorized: no user_id")
	}
	userID, ok := uid.(uuid.UUID)
	if !ok {
		return helper.Error(c, fiber.StatusUnauthorized, "Invalid user_id type")
	}

	var in createProfileReq
	if err := c.BodyParser(&in); err != nil {
		log.Println("[ERROR] Invalid request body:", err)
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request format")
	}

	// Upsert by user_id
	var existing uModel.UsersProfileModel
	err := upc.DB.Where("user_id = ?", userID).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Println("[ERROR] DB error:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "DB error")
	}

	// Build model fields
	dob := parseDOB(in.DateOfBirth)
	g := parseGender(in.Gender)

	if err == nil {
		// update
		patch := map[string]any{
			"donation_name": in.DonationName,
			"father_name":   in.FatherName,
			"mother_name":   in.MotherName,
			"phone_number":  in.PhoneNumber,
			"bio":           in.Bio,
			"location":      in.Location,
			"occupation":    in.Occupation,
			"updated_at":    time.Now(),
		}
		if dob != nil {
			patch["date_of_birth"] = *dob
		}
		if g != nil {
			patch["gender"] = *g
		}
		if err := upc.DB.Model(&existing).Updates(patch).Error; err != nil {
			log.Println("[ERROR] Failed to update user profile:", err)
			return helper.Error(c, fiber.StatusInternalServerError, "Failed to update user profile")
		}
		// re-read to return the latest
		var latest uModel.UsersProfileModel
		_ = upc.DB.Where("user_id = ?", userID).First(&latest).Error
		return helper.Success(c, "User profile updated successfully", latest)
	}

	// create
	newRow := uModel.UsersProfileModel{
		UserID:       userID,
		DonationName: in.DonationName,
		FatherName:   in.FatherName,
		MotherName:   in.MotherName,
		PhoneNumber:  in.PhoneNumber,
		Bio:          in.Bio,
		Location:     in.Location,
		Occupation:   in.Occupation,
	}
	if dob != nil {
		newRow.DateOfBirth = dob
	}
	if g != nil {
		newRow.Gender = g
	}

	if err := upc.DB.Create(&newRow).Error; err != nil {
		log.Println("[ERROR] Failed to create user profile:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to create user profile")
	}
	return helper.SuccessWithCode(c, fiber.StatusCreated, "User profile created successfully", newRow)
}

func (upc *UsersProfileController) UpdateProfile(c *fiber.Ctx) error {
	uid := c.Locals("user_id")
	if uid == nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Unauthorized - user_id missing")
	}
	userID, ok := uid.(uuid.UUID)
	if !ok {
		return helper.Error(c, fiber.StatusUnauthorized, "Invalid user_id type")
	}

	log.Println("[INFO] Updating user profile with user_id:", userID)

	var profile uModel.UsersProfileModel
	if err := upc.DB.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.Error(c, fiber.StatusNotFound, "User profile not found")
		}
		log.Println("[ERROR] DB error:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to fetch user profile")
	}

	var in updateProfileReq
	if err := c.BodyParser(&in); err != nil {
		log.Println("[ERROR] Invalid request body:", err)
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request format")
	}

	patch := map[string]any{
		"updated_at": time.Now(),
	}

	// only set provided fields (including empty string to clear)
	if in.DonationName != nil {
		patch["donation_name"] = *in.DonationName
	}
	if in.FatherName != nil {
		patch["father_name"] = *in.FatherName
	}
	if in.MotherName != nil {
		patch["mother_name"] = *in.MotherName
	}
	if in.PhoneNumber != nil {
		patch["phone_number"] = *in.PhoneNumber
	}
	if in.Bio != nil {
		patch["bio"] = *in.Bio
	}
	if in.Location != nil {
		patch["location"] = *in.Location
	}
	if in.Occupation != nil {
		patch["occupation"] = *in.Occupation
	}
	if in.DateOfBirth != nil { // allow clear or set
		if t := parseDOB(in.DateOfBirth); t != nil {
			patch["date_of_birth"] = *t
		} else {
			patch["date_of_birth"] = nil
		}
	}
	if in.Gender != nil { // allow clear or set
		if g := parseGender(in.Gender); g != nil {
			patch["gender"] = *g
		} else {
			patch["gender"] = nil
		}
	}

	if err := upc.DB.Model(&profile).Updates(patch).Error; err != nil {
		log.Println("[ERROR] Failed to update user profile:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to update user profile")
	}

	// return the latest row
	var latest uModel.UsersProfileModel
	_ = upc.DB.Where("user_id = ?", userID).First(&latest).Error
	return helper.Success(c, "User profile updated successfully", latest)
}

func (upc *UsersProfileController) DeleteProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	log.Println("[INFO] Deleting user profile with user_id:", userID)

	var profile uModel.UsersProfileModel
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
