package controller

import (
	masjidAdminModel "masjidku_backend/internals/features/masjids/masjid_admins_teachers/model"
	"masjidku_backend/internals/features/users/auth/service"
	models "masjidku_backend/internals/features/users/user/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthController struct {
	DB *gorm.DB
}

func NewAuthController(db *gorm.DB) *AuthController {
	return &AuthController{DB: db}
}

func (ac *AuthController) Me(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid user ID in context")
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid UUID format")
	}

	var user models.UserModel
	if err := ac.DB.First(&user, "id = ?", userUUID).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	user.Password = ""

	// Ambil daftar masjid_id jika user adalah DKM
	var masjidIDs []string
	if user.Role == "dkm" {
		var adminMasjids []masjidAdminModel.MasjidAdminModel
		if err := ac.DB.
			Where("masjid_admins_user_id = ? AND masjid_admins_is_active = true", user.ID).
			Find(&adminMasjids).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data masjid admin")
		}

		for _, m := range adminMasjids {
			masjidIDs = append(masjidIDs, m.MasjidID.String())
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user": fiber.Map{
			"id":                user.ID,
			"user_name":         user.UserName,
			"email":             user.Email,
			"role":              user.Role,
			"masjid_admin_ids":  masjidIDs, // ✅ Tambahkan ini
			// tambahkan field lain sesuai kebutuhan
		},
	})
}


func (ac *AuthController) UpdateUserName(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid user ID in context")
	}

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid UUID format")
	}

	var req struct {
		UserName string `json:"user_name" validate:"required,min=3,max=50"`
	}

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := ac.DB.Model(&models.UserModel{}).
		Where("id = ?", userUUID).
		Update("user_name", req.UserName).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update user name")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Username berhasil diperbarui",
	})
}

func (ac *AuthController) Register(c *fiber.Ctx) error {
	return service.Register(ac.DB, c)
}

func (ac *AuthController) Login(c *fiber.Ctx) error {
	return service.Login(ac.DB, c)
}

func (ac *AuthController) LoginGoogle(c *fiber.Ctx) error {
	return service.LoginGoogle(ac.DB, c)
}

func (ac *AuthController) Logout(c *fiber.Ctx) error {
	return service.Logout(ac.DB, c)
}

func (pc *AuthController) ChangePassword(c *fiber.Ctx) error {
	return service.ChangePassword(pc.DB, c)
}

func (rc *AuthController) RefreshToken(c *fiber.Ctx) error {
	return service.RefreshToken(rc.DB, c)
}

func (ac *AuthController) ResetPassword(c *fiber.Ctx) error {
	return service.ResetPassword(ac.DB, c)
}

func (ac *AuthController) CheckSecurityAnswer(c *fiber.Ctx) error {
	return service.CheckSecurityAnswer(ac.DB, c)
}
