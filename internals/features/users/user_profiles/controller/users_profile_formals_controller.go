// file: internals/features/users/profile/controller/users_profile_formal_controller.go
package controller

import (
	"errors"
	"log"
	"net/http"

	"madinahsalam_backend/internals/features/users/user_profiles/dto"
	"madinahsalam_backend/internals/features/users/user_profiles/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UsersProfileFormalController struct {
	DB *gorm.DB
}

func NewUsersProfileFormalController(db *gorm.DB) *UsersProfileFormalController {
	return &UsersProfileFormalController{DB: db}
}

/* ===========================================================
 * Auth: GET /api/a/users-profile-formal (punya sendiri)
 * =========================================================== */
func (ctl *UsersProfileFormalController) GetMine(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, "Unauthorized")
	}

	var m model.UsersProfileFormalModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_id = ?", userID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "Formal profile not found")
		}
		log.Println("[ERROR] DB:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "DB error")
	}

	return helper.JsonOK(c, "Success get profile formal", dto.NewUsersProfileFormalResponse(&m))
}

/* ===========================================================
 * Auth: PUT /api/a/users-profile-formal (upsert idempotent)
 * =========================================================== */
func (ctl *UsersProfileFormalController) UpsertMine(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, "Unauthorized")
	}

	body, err := dto.BindUpdate(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}

	var m model.UsersProfileFormalModel
	tx := ctl.DB.WithContext(c.Context())

	if err := tx.Where("user_id = ?", userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			m = model.UsersProfileFormalModel{UserID: userID}
			body.ApplyToModelPartial(&m)

			if err := tx.Create(&m).Error; err != nil {
				log.Println("[ERROR] Create:", err)
				return helper.JsonError(c, http.StatusInternalServerError, "Failed to create")
			}
			return helper.JsonCreated(c, "Formal profile created", dto.NewUsersProfileFormalResponse(&m))
		}
		log.Println("[ERROR] DB First:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "DB error")
	}

	body.ApplyToModelPartial(&m)
	if err := tx.Save(&m).Error; err != nil {
		log.Println("[ERROR] Save:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "Failed to update")
	}
	return helper.JsonUpdated(c, "Formal profile updated", dto.NewUsersProfileFormalResponse(&m))
}

/* ===========================================================
 * Auth: PATCH /api/a/users-profile-formal (partial update)
 * =========================================================== */
func (ctl *UsersProfileFormalController) PatchMine(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, "Unauthorized")
	}

	var m model.UsersProfileFormalModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_id = ?", userID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			m = model.UsersProfileFormalModel{UserID: userID}
			body, berr := dto.BindUpdate(c)
			if berr != nil {
				return helper.JsonError(c, http.StatusBadRequest, berr.Error())
			}
			body.ApplyToModelPartial(&m)

			if cerr := ctl.DB.WithContext(c.Context()).Create(&m).Error; cerr != nil {
				log.Println("[ERROR] Create:", cerr)
				return helper.JsonError(c, http.StatusInternalServerError, "Failed to create")
			}
			return helper.JsonCreated(c, "Formal profile created", dto.NewUsersProfileFormalResponse(&m))
		}
		log.Println("[ERROR] DB First:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "DB error")
	}

	body, err := dto.BindUpdate(c)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, err.Error())
	}
	body.ApplyToModelPartial(&m)

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		log.Println("[ERROR] Save:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "Failed to update")
	}
	return helper.JsonUpdated(c, "Formal profile updated", dto.NewUsersProfileFormalResponse(&m))
}

/* ===========================================================
 * Auth: DELETE /api/a/users-profile-formal (soft delete own)
 * =========================================================== */
func (ctl *UsersProfileFormalController) DeleteMine(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, http.StatusUnauthorized, "Unauthorized")
	}

	if err := ctl.DB.WithContext(c.Context()).
		Where("user_id = ?", userID).
		Delete(&model.UsersProfileFormalModel{}).Error; err != nil {
		log.Println("[ERROR] Delete:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "Failed to delete")
	}

	return helper.JsonDeleted(c, "Formal profile deleted", nil)
}

/* ===========================================================
 * ADMIN OPSIONAL
 * =========================================================== */

func (ctl *UsersProfileFormalController) AdminGetByUserID(c *fiber.Ctx) error {
	userIDStr := c.Params("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid user_id")
	}

	var m model.UsersProfileFormalModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_id = ?", userID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "not found")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "DB error")
	}

	return helper.JsonOK(c, "Success get formal profile by user_id", dto.NewUsersProfileFormalResponse(&m))
}

func (ctl *UsersProfileFormalController) AdminDeleteByUserID(c *fiber.Ctx) error {
	userIDStr := c.Params("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "invalid user_id")
	}

	if err := ctl.DB.WithContext(c.Context()).
		Where("user_id = ?", userID).
		Delete(&model.UsersProfileFormalModel{}).Error; err != nil {
		log.Println("[ERROR] AdminDelete:", err)
		return helper.JsonError(c, http.StatusInternalServerError, "Failed to delete")
	}
	return helper.JsonDeleted(c, "Formal profile deleted by admin", nil)
}
