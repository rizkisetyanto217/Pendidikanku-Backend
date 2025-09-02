// file: internals/features/users/model/user_model.go
package model

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Validator instance
var validate = validator.New()

// UserModel merepresentasikan tabel "users" (tanpa kolom role)
type UserModel struct {
	ID               uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserName         string         `gorm:"size:50;not null;index" json:"user_name" validate:"required,min=3,max=50"`
	FullName         *string        `gorm:"size:100" json:"full_name,omitempty" validate:"omitempty,min=3,max=100"`
	Email            string         `gorm:"type:citext;unique;not null" json:"email" validate:"required,email"`
	Password         string         `gorm:"not null" json:"password" validate:"required,min=8"`
	GoogleID         *string        `gorm:"size:255;unique" json:"google_id,omitempty"`
	SecurityQuestion string         `gorm:"not null" json:"security_question" validate:"required"`
	SecurityAnswer   string         `gorm:"size:255;not null" json:"security_answer" validate:"required"`
	IsActive         bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt        time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName memastikan nama tabel sesuai skema database
func (UserModel) TableName() string { return "users" }

// BeforeSave: normalisasi ringan (optional, aman untuk citext)
func (u *UserModel) BeforeSave(tx *gorm.DB) error {
	u.Email = strings.TrimSpace(u.Email)
	u.UserName = strings.TrimSpace(u.UserName)
	if u.FullName != nil {
		f := strings.TrimSpace(*u.FullName)
		u.FullName = &f
	}
	return nil
}

// Validate memeriksa apakah input sesuai aturan yang telah didefinisikan
func (u *UserModel) Validate() error {
	// Tidak ada default role lagi; role diambil dari tabel lain (user_roles)
	if err := validate.Struct(u); err != nil {
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

// formatErrorMessage mengubah map error menjadi string multi-baris
func formatErrorMessage(errors map[string]string) string {
	var b strings.Builder
	for field, msg := range errors {
		b.WriteString(field)
		b.WriteString(": ")
		b.WriteString(msg)
		b.WriteString("\n")
	}
	return b.String()
}
