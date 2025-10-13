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

// UserModel merepresentasikan tabel "users"
type UserModel struct {
	ID              uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserName        string         `gorm:"type:varchar(50);not null;index" json:"user_name" validate:"required,min=3,max=50"`
	FullName        *string        `gorm:"type:varchar(100)" json:"full_name,omitempty" validate:"omitempty,min=3,max=100"`
	Email           string         `gorm:"type:citext;unique;not null" json:"email" validate:"required,email"`
	Password        *string        `gorm:"type:varchar(250)" json:"password,omitempty" validate:"omitempty,min=8"`
	GoogleID        *string        `gorm:"type:varchar(255);unique" json:"google_id,omitempty" validate:"omitempty,max=255"`
	IsActive        bool           `gorm:"not null;default:true" json:"is_active"`
	EmailVerifiedAt *time.Time     `json:"email_verified_at,omitempty"`

	CreatedAt time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName memastikan nama tabel sesuai skema database
func (UserModel) TableName() string { return "users" }

// BeforeSave: normalisasi ringan (trim spasi)
func (u *UserModel) BeforeSave(tx *gorm.DB) error {
	u.Email = strings.TrimSpace(u.Email)
	u.UserName = strings.TrimSpace(u.UserName)
	if u.FullName != nil {
		f := strings.TrimSpace(*u.FullName)
		u.FullName = &f
	}
	if u.Password != nil {
		p := strings.TrimSpace(*u.Password)
		u.Password = &p
	}
	if u.GoogleID != nil {
		g := strings.TrimSpace(*u.GoogleID)
		u.GoogleID = &g
	}
	return nil
}

// Validate memeriksa apakah input sesuai aturan
func (u *UserModel) Validate() error {
	if err := validate.Struct(u); err != nil {
		return formatValidationError(err)
	}
	return nil
}

// ---- helpers ----------------------------------------------------

func formatValidationError(err error) error {
	if validationErrs, ok := err.(validator.ValidationErrors); ok {
		errorMessages := make(map[string]string)
		for _, fe := range validationErrs {
			switch fe.Tag() {
			case "required":
				errorMessages[fe.Field()] = fe.Field() + " wajib diisi."
			case "email":
				errorMessages[fe.Field()] = "Format email tidak valid."
			case "min":
				errorMessages[fe.Field()] = fe.Field() + " harus minimal " + fe.Param() + " karakter."
			case "max":
				errorMessages[fe.Field()] = fe.Field() + " harus kurang dari " + fe.Param() + " karakter."
			default:
				errorMessages[fe.Field()] = "Format tidak valid."
			}
		}
		return errors.New(formatErrorMessage(errorMessages))
	}
	return err
}

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
