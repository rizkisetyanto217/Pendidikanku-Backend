package controller

import (
	"masjidku_backend/internals/features/users/auth/service"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type AuthController struct {
	DB *gorm.DB
}

func NewAuthController(db *gorm.DB) *AuthController {
	return &AuthController{DB: db}
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
