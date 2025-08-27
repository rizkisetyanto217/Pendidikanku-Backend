package controller

import (
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/users/user/model"
	helper "masjidku_backend/internals/helpers"
)

type UserController struct {
	DB *gorm.DB
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{DB: db}
}

// helper: sanitasi field sensitif sebelum kirim ke client
func sanitizeUser(u *model.UserModel) {
	u.Password = ""
	u.SecurityAnswer = ""
	u.SecurityQuestion = "" // opsional: sembunyikan juga pertanyaan
}

// GET /api/a/users
func (uc *UserController) GetUsers(c *fiber.Ctx) error {
	var users []model.UserModel
	if err := uc.DB.Find(&users).Error; err != nil {
		log.Println("[ERROR] Failed to fetch users:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to retrieve users")
	}

	for i := range users {
		sanitizeUser(&users[i])
	}

	log.Printf("[SUCCESS] Retrieved %d users\n", len(users))
	return helper.Success(c, "Users fetched successfully", fiber.Map{
		"total": len(users),
		"users": users,
	})
}

// GET /api/a/users/search?q=namaOrEmail
func (uc *UserController) SearchUsers(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Query tidak boleh kosong")
	}

	var users []model.UserModel
	if err := uc.DB.
		Where("user_name ILIKE ? OR email ILIKE ? OR full_name ILIKE ?", "%"+query+"%", "%"+query+"%", "%"+query+"%").
		Find(&users).Error; err != nil {
		log.Println("[ERROR] SearchUsers gagal:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mencari pengguna")
	}

	for i := range users {
		sanitizeUser(&users[i])
	}

	log.Printf("[SUCCESS] Ditemukan %d user dengan query '%s'\n", len(users), query)
	return helper.Success(c, "Hasil pencarian user", fiber.Map{
		"total": len(users),
		"users": users,
	})
}

// GET /api/a/users/me — profile user dari JWT
func (uc *UserController) GetUser(c *fiber.Ctx) error {
	userIDRaw := c.Locals("user_id")
	userIDStr, ok := userIDRaw.(string)
	if !ok {
		return helper.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Invalid user ID format")
	}

	var user model.UserModel
	if err := uc.DB.First(&user, "id = ?", userID).Error; err != nil {
		return helper.Error(c, fiber.StatusNotFound, "User not found")
	}

	sanitizeUser(&user)
	return helper.Success(c, "User profile fetched successfully", user)
}

// POST /api/a/users — create single atau multiple users (JSON array)
func (uc *UserController) CreateUser(c *fiber.Ctx) error {
	// coba parse sebagai array dulu
	var many []model.UserModel
	if err := c.BodyParser(&many); err == nil && len(many) > 0 {
		// validasi semua item
		for i := range many {
			if err := many[i].Validate(); err != nil {
				return helper.ErrorWithDetails(c, fiber.StatusBadRequest, "Validation error", err.Error())
			}
			// TODO: hash password sebelum simpan (jika kamu punya helpernya)
			// many[i].Password, _ = helper.HashPassword(many[i].Password)
		}
		if err := uc.DB.Create(&many).Error; err != nil {
			log.Println("[ERROR] Failed to create multiple users:", err)
			return helper.Error(c, fiber.StatusInternalServerError, "Failed to create multiple users")
		}
		for i := range many {
			sanitizeUser(&many[i])
		}
		log.Printf("[SUCCESS] Created %d users\n", len(many))
		return helper.SuccessWithCode(c, fiber.StatusCreated, "Users created successfully", many)
	}

	// kalau bukan array, parse single
	var one model.UserModel
	if err := c.BodyParser(&one); err != nil {
		log.Println("[ERROR] Invalid input format:", err)
		return helper.Error(c, fiber.StatusBadRequest, "Invalid input format")
	}

	if err := one.Validate(); err != nil {
		return helper.ErrorWithDetails(c, fiber.StatusBadRequest, "Validation error", err.Error())
	}

	// TODO: hash password sebelum simpan
	// one.Password, _ = helper.HashPassword(one.Password)

	if err := uc.DB.Create(&one).Error; err != nil {
		log.Println("[ERROR] Failed to create user:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to create user")
	}

	sanitizeUser(&one)
	log.Printf("[SUCCESS] Created single user ID: %v\n", one.ID)
	return helper.SuccessWithCode(c, fiber.StatusCreated, "User created successfully", one)
}

// PUT /api/a/users — update user by JWT (profile update)
// NOTE: ubah password dan security Q/A sebaiknya endpoint terpisah
type UpdateUserInput struct {
	UserName string  `json:"user_name" validate:"required,min=3,max=50"`
	FullName string  `json:"full_name" validate:"required,min=3,max=50"`
	Email    string  `json:"email" validate:"required,email"`
	Role     *string `json:"role,omitempty" validate:"omitempty,oneof=user admin dkm author teacher"`
	IsActive *bool   `json:"is_active,omitempty" validate:"omitempty"`
}

func (uc *UserController) UpdateUser(c *fiber.Ctx) error {
	userIDRaw := c.Locals("user_id")
	if userIDRaw == nil {
		return helper.Error(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	userIDStr, ok := userIDRaw.(string)
	if !ok {
		return helper.Error(c, fiber.StatusInternalServerError, "Invalid user ID in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid UUID format")
	}

	var user model.UserModel
	if err := uc.DB.First(&user, "id = ?", userID).Error; err != nil {
		return helper.Error(c, fiber.StatusNotFound, "User not found")
	}

	var input UpdateUserInput
	if err := c.BodyParser(&input); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid request body")
	}

	v := validator.New()
	if err := v.Struct(&input); err != nil {
		return helper.ErrorWithDetails(c, fiber.StatusBadRequest, "Validation error", err.Error())
	}

	// Update fields yang disetujui
	user.UserName = input.UserName
	user.FullName = input.FullName
	user.Email = input.Email
	if input.Role != nil && *input.Role != "" {
		user.Role = *input.Role
	}
	if input.IsActive != nil {
		user.IsActive = *input.IsActive
	}

	// validasi lagi pakai rule di model (jaga konsistensi)
	if err := user.Validate(); err != nil {
		return helper.ErrorWithDetails(c, fiber.StatusBadRequest, "Validation error", err.Error())
	}

	if err := uc.DB.Save(&user).Error; err != nil {
		log.Println("[ERROR] Failed to update user:", err)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to update user")
	}

	sanitizeUser(&user)
	log.Printf("[SUCCESS] Updated user ID: %v\n", user.ID)
	return helper.Success(c, "User updated successfully", user)
}

// DELETE /api/a/users/:id — hard delete (jika mau soft delete, tambahkan gorm.DeletedAt di model)
func (uc *UserController) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Invalid UUID format")
	}

	tx := uc.DB.Delete(&model.UserModel{}, "id = ?", id)
	if tx.Error != nil {
		log.Println("[ERROR] Failed to delete user:", tx.Error)
		return helper.Error(c, fiber.StatusInternalServerError, "Failed to delete user")
	}
	if tx.RowsAffected == 0 {
		return helper.Error(c, fiber.StatusNotFound, "User not found")
	}

	log.Printf("[SUCCESS] Deleted user ID: %s\n", id)
	return helper.Success(c, "User deleted successfully", fiber.Map{
		"id": id,
	})
}
