package controller

import (
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	userdto "masjidku_backend/internals/features/users/user/dto"
	"masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"
)

type AdminUserController struct {
	DB *gorm.DB
}

func NewAdminUserController(db *gorm.DB) *AdminUserController { return &AdminUserController{DB: db} }

func parseBool(q string) bool {
	switch strings.ToLower(strings.TrimSpace(q)) {
	case "1", "true", "yes", "y", "on":
		return true
	}
	return false
}

// ==============================
// READ (ADMIN)
// ==============================

// GET /api/a/users
// ?with_deleted=1  → include soft-deleted (Unscoped) dan expose deleted_at
func (ac *AdminUserController) GetUsers(c *fiber.Ctx) error {
	withDeleted := parseBool(c.Query("with_deleted"))

	var users []model.UserModel
	tx := ac.DB.Order("created_at DESC")
	if withDeleted {
		tx = tx.Unscoped()
	}
	if err := tx.Find(&users).Error; err != nil {
		log.Println("[ERROR] Failed to fetch users:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve users")
	}

	if withDeleted {
		resp := make([]userdto.UserResponseWithDeletedAt, 0, len(users))
		for i := range users {
			if r := userdto.FromModelWithDeletedAt(&users[i]); r != nil {
				resp = append(resp, *r)
			}
		}
		return helper.JsonOK(c, "Users fetched successfully", fiber.Map{
			"total": len(resp),
			"users": resp,
		})
	}

	resp := userdto.FromModelList(users)
	return helper.JsonOK(c, "Users fetched successfully", fiber.Map{
		"total": len(resp),
		"users": resp,
	})
}

// GET /api/a/users/search?q=namaOrEmail
// ?with_deleted=1 → include soft-deleted + expose deleted_at
func (ac *AdminUserController) SearchUsers(c *fiber.Ctx) error {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak boleh kosong")
	}
	withDeleted := parseBool(c.Query("with_deleted"))

	q := "%" + query + "%"
	var users []model.UserModel

	tx := ac.DB.Where("user_name ILIKE ? OR email ILIKE ? OR full_name ILIKE ?", q, q, q).
		Order("created_at DESC")
	if withDeleted {
		tx = tx.Unscoped()
	}
	if err := tx.Find(&users).Error; err != nil {
		log.Println("[ERROR] SearchUsers gagal:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mencari pengguna")
	}

	if withDeleted {
		resp := make([]userdto.UserResponseWithDeletedAt, 0, len(users))
		for i := range users {
			if r := userdto.FromModelWithDeletedAt(&users[i]); r != nil {
				resp = append(resp, *r)
			}
		}
		return helper.JsonOK(c, "Hasil pencarian user", fiber.Map{
			"total": len(resp),
			"users": resp,
		})
	}

	resp := userdto.FromModelList(users)
	return helper.JsonOK(c, "Hasil pencarian user", fiber.Map{
		"total": len(resp),
		"users": resp,
	})
}

// GET /api/a/users/:id
// ?with_deleted=1 → boleh ambil yang soft-deleted + expose deleted_at
func (ac *AdminUserController) GetUserByID(c *fiber.Ctx) error {
	id := c.Params("id")
	uid, err := uuid.Parse(id)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid UUID format")
	}
	withDeleted := parseBool(c.Query("with_deleted"))

	var user model.UserModel
	tx := ac.DB
	if withDeleted {
		tx = tx.Unscoped()
	}
	if err := tx.First(&user, "id = ?", uid).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "User not found")
	}

	if withDeleted {
		return helper.JsonOK(c, "User fetched successfully", userdto.FromModelWithDeletedAt(&user))
	}
	return helper.JsonOK(c, "User fetched successfully", userdto.FromModel(&user))
}

// ==============================
// CREATE (ADMIN)
// ==============================

// POST /api/a/users  (accept single object ataupun array)
func (ac *AdminUserController) CreateUser(c *fiber.Ctx) error {
	v := validator.New()

	// coba parse sebagai array DTO
	var manyReq []userdto.CreateUserRequest
	if err := c.BodyParser(&manyReq); err == nil && len(manyReq) > 0 {
		users := make([]model.UserModel, 0, len(manyReq))
		for i := range manyReq {
			manyReq[i].Normalize()
			if err := v.Struct(&manyReq[i]); err != nil {
				return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
			}
			m := manyReq[i].ToModel()

			// TODO: hash password di sini kalau diperlukan

			if err := m.Validate(); err != nil {
				return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
			}
			users = append(users, *m)
		}
		if err := ac.DB.Create(&users).Error; err != nil {
			log.Println("[ERROR] Failed to create multiple users:", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create multiple users")
		}
		return helper.JsonCreated(c, "Users created successfully", userdto.FromModelList(users))
	}

	// fallback: single DTO
	var oneReq userdto.CreateUserRequest
	if err := c.BodyParser(&oneReq); err != nil {
		log.Println("[ERROR] Invalid input format:", err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid input format")
	}
	oneReq.Normalize()
	if err := v.Struct(&oneReq); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}
	u := oneReq.ToModel()

	// TODO: hash password di sini kalau diperlukan

	if err := u.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusUnprocessableEntity, err.Error())
	}
	if err := ac.DB.Create(u).Error; err != nil {
		log.Println("[ERROR] Failed to create user:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to create user")
	}
	return helper.JsonCreated(c, "User created successfully", userdto.FromModel(u))
}

// ==============================
// DELETE (soft), RESTORE, FORCE DELETE (ADMIN)
// ==============================

// DELETE /api/a/users/:id  (soft delete)
func (ac *AdminUserController) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid UUID format")
	}
	tx := ac.DB.Delete(&model.UserModel{}, "id = ?", id)
	if tx.Error != nil {
		log.Println("[ERROR] Failed to delete user:", tx.Error)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to delete user")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "User not found")
	}
	return helper.JsonDeleted(c, "User deleted successfully", fiber.Map{"id": id})
}

// GET /api/a/users/deleted
func (ac *AdminUserController) GetDeletedUsers(c *fiber.Ctx) error {
	var users []model.UserModel
	if err := ac.DB.Unscoped().
		Where("deleted_at IS NOT NULL").
		Order("deleted_at DESC").
		Find(&users).Error; err != nil {
		log.Println("[ERROR] Failed to fetch deleted users:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve deleted users")
	}
	resp := make([]userdto.UserResponseWithDeletedAt, 0, len(users))
	for i := range users {
		if r := userdto.FromModelWithDeletedAt(&users[i]); r != nil {
			resp = append(resp, *r)
		}
	}
	return helper.JsonOK(c, "Deleted users fetched successfully", fiber.Map{
		"total": len(resp),
		"users": resp,
	})
}

// POST /api/a/users/:id/restore
func (ac *AdminUserController) RestoreUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid UUID format")
	}
	if err := ac.DB.Unscoped().
		Model(&model.UserModel{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error; err != nil {
		log.Println("[ERROR] Failed to restore user:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to restore user")
	}
	return helper.JsonUpdated(c, "User restored successfully", fiber.Map{"id": id})
}

// DELETE /api/a/users/:id/force
func (ac *AdminUserController) ForceDeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Invalid UUID format")
	}
	tx := ac.DB.Unscoped().Delete(&model.UserModel{}, "id = ?", id)
	if tx.Error != nil {
		log.Println("[ERROR] Failed to force delete user:", tx.Error)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to force delete user")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "User not found")
	}
	return helper.JsonDeleted(c, "User permanently deleted", fiber.Map{"id": id})
}
