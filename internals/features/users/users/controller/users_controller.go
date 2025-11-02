package controller

import (
	"errors"
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	userdto "schoolku_backend/internals/features/users/users/dto"
	"schoolku_backend/internals/features/users/users/model"
	helper "schoolku_backend/internals/helpers"
)

type AdminUserController struct {
	DB *gorm.DB
}

func NewAdminUserController(db *gorm.DB) *AdminUserController { return &AdminUserController{DB: db} }

// GET /api/a/users
// Query:
//   q=namaOrEmail (opsional; jika diisi → filter/search)
//   with_deleted=1 (opsional; include soft-deleted + expose deleted_at)

func (ac *AdminUserController) ListUsers(c *fiber.Ctx) error {
	// flags
	withDeleted := strings.EqualFold(c.Query("with_deleted"), "1") ||
		strings.EqualFold(c.Query("with_deleted"), "true")

	// DETAIL via ?id=...
	if id := strings.TrimSpace(c.Query("id")); id != "" {
		var u model.UserModel
		tx := ac.DB
		if withDeleted {
			tx = tx.Unscoped()
		}
		if err := tx.First(&u, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return helper.JsonError(c, fiber.StatusNotFound, "User tidak ditemukan")
			}
			log.Println("[ERROR] GetUserByID:", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil user")
		}

		if withDeleted {
			if r := userdto.FromModelWithDeletedAt(&u); r != nil {
				return helper.JsonOK(c, "User fetched successfully", r)
			}
		}
		return helper.JsonOK(c, "User fetched successfully", userdto.FromModel(&u))
	}

	// LIST / SEARCH via ?q=
	q := strings.TrimSpace(c.Query("q"))
	var users []model.UserModel

	tx := ac.DB.Order("created_at DESC")
	if withDeleted {
		tx = tx.Unscoped()
	}
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("user_name ILIKE ? OR email ILIKE ? OR full_name ILIKE ?", like, like, like)
	}

	if err := tx.Find(&users).Error; err != nil {
		log.Println("[ERROR] ListUsers:", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data pengguna")
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
