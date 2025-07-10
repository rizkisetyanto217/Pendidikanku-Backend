package model

import (
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	// "gorm.io/gorm"
)

// Validator instance
var validate = validator.New()

// UserModel merepresentasikan tabel users di database
type UserModel struct {
	ID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserName string    `gorm:"size:50;not null" json:"user_name" validate:"required,min=3,max=50"`
	Email    string    `gorm:"size:255;unique;not null" json:"email" validate:"required,email"`
	Password string    `gorm:"not null" json:"password" validate:"required,min=8"`
	GoogleID *string   `gorm:"size:255;unique" json:"google_id,omitempty"`
	Role     string    `gorm:"type:varchar(20);not null;default:'user'" json:"-"`
	SecurityQuestion string    `gorm:"not null" json:"security_question"`
	SecurityAnswer   string    `gorm:"size:255;not null" json:"security_answer"`
	IsActive         bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName memastikan nama tabel sesuai dengan skema database
func (UserModel) TableName() string {
	return "users"
}

// SetDefaultValues memastikan nilai default sebelum validasi
func (u *UserModel) SetDefaultValues() {
	if u.Role == "" {
		u.Role = "user"
	}
}

// Validate memeriksa apakah input sesuai aturan yang telah didefinisikan
func (u *UserModel) Validate() error {
	// Set default role sebelum validasi
	u.SetDefaultValues()

	err := validate.Struct(u)
	if err != nil {
		return formatValidationError(err)
	}

	return nil
}

// formatValidationError mengubah error validasi menjadi format yang lebih jelas
func formatValidationError(err error) error {
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		errorMessages := make(map[string]string)
		for _, fieldErr := range validationErrs {
			switch fieldErr.Tag() {
			case "required":
				errorMessages[fieldErr.Field()] = fieldErr.Field() + " wajib diisi."
			case "email":
				errorMessages[fieldErr.Field()] = "Format email tidak valid."
			case "min":
				errorMessages[fieldErr.Field()] = fieldErr.Field() + " harus minimal " + fieldErr.Param() + " karakter."
			case "max":
				errorMessages[fieldErr.Field()] = fieldErr.Field() + " harus kurang dari " + fieldErr.Param() + " karakter."
			case "oneof":
				errorMessages[fieldErr.Field()] = fieldErr.Field() + " harus salah satu dari " + fieldErr.Param() + "."
			default:
				errorMessages[fieldErr.Field()] = "Format tidak valid."
			}
		}
		return errors.New(formatErrorMessage(errorMessages))
	}

	return err
}

// formatErrorMessage mengubah map error menjadi string JSON-like
func formatErrorMessage(errors map[string]string) string {
	var msg string
	for field, errorMsg := range errors {
		msg += field + ": " + errorMsg + "\n"
	}
	return msg
}
