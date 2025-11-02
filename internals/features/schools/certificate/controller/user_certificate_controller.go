package controllers

import (
	"time"

	helper "schoolku_backend/internals/helpers"

	"schoolku_backend/internals/features/schools/certificate/dto"
	userCertModel "schoolku_backend/internals/features/schools/certificate/model"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type UserCertificateController struct {
	DB *gorm.DB
}

func NewUserCertificateController(db *gorm.DB) *UserCertificateController {
	return &UserCertificateController{DB: db}
}

// ✅ CREATE
func (ctrl *UserCertificateController) Create(c *fiber.Ctx) error {
	var body dto.CreateUserCertificateDTO
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid body")
	}

	newCert := userCertModel.UserCertificateModel{
		UserCertUserID:        body.UserCertUserID,
		UserCertCertificateID: body.UserCertCertificateID,
		UserCertScore:         body.UserCertScore,
		UserCertSlugURL:       body.UserCertSlugURL,
		UserCertIsUpToDate:    true,
		UserCertIssuedAt:      time.Now(),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := ctrl.DB.Create(&newCert).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create user certificate")
	}

	return helper.JsonCreated(c, "User certificate created", newCert)
}

// ✅ GET ALL (tanpa pagination; bisa ditambah jika perlu)
func (ctrl *UserCertificateController) GetAll(c *fiber.Ctx) error {
	var certs []userCertModel.UserCertificateModel
	if err := ctrl.DB.Find(&certs).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to fetch data")
	}
	return helper.JsonList(c, certs, nil)
}

// ✅ GET BY ID
func (ctrl *UserCertificateController) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")

	var cert userCertModel.UserCertificateModel
	if err := ctrl.DB.First(&cert, "user_cert_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "User certificate not found")
	}

	return helper.JsonOK(c, "OK", cert)
}

// ✅ UPDATE (partial)
func (ctrl *UserCertificateController) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var cert userCertModel.UserCertificateModel
	if err := ctrl.DB.First(&cert, "user_cert_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "User certificate not found")
	}

	var body dto.UpdateUserCertificateDTO
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid body")
	}

	if body.UserCertScore != nil {
		cert.UserCertScore = body.UserCertScore
	}
	if body.UserCertSlugURL != nil {
		cert.UserCertSlugURL = *body.UserCertSlugURL
	}
	if body.UserCertIsUpToDate != nil {
		cert.UserCertIsUpToDate = *body.UserCertIsUpToDate
	}
	cert.UpdatedAt = time.Now()

	if err := ctrl.DB.Save(&cert).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to update user certificate")
	}

	return helper.JsonUpdated(c, "User certificate updated", cert)
}

// ✅ DELETE
func (ctrl *UserCertificateController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")

	// optional: cek eksistensi agar error lebih jelas
	var exists int64
	if err := ctrl.DB.Model(&userCertModel.UserCertificateModel{}).
		Where("user_cert_id = ?", id).
		Count(&exists).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete user certificate")
	}
	if exists == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "User certificate not found")
	}

	if err := ctrl.DB.Delete(&userCertModel.UserCertificateModel{}, "user_cert_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete user certificate")
	}

	return helper.JsonDeleted(c, "User certificate deleted", fiber.Map{"id": id})
}
