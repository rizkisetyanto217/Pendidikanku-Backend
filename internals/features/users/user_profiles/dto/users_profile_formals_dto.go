// file: internals/features/users/profile/dto/users_profile_formal_dto.go
package dto

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	model "schoolku_backend/internals/features/users/user_profiles/model"
)

/* =========================
 * Validator instance
 * ========================= */
var validate = validator.New()

/* =========================
 * Helpers
 * ========================= */

// NormalizePhone: hilangkan spasi, -, (, ) dan ubah prefix 0 -> +62 (opsional)
func NormalizePhone(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	r := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")
	s = r.Replace(s)

	// Contoh kebijakan normalisasi sederhana:
	if strings.HasPrefix(s, "0") {
		s = "+62" + s[1:]
	}
	return s
}

/* =========================
 * Request DTO
 * ========================= */

// POST (Create) — jika kamu izinkan create terpisah dari otomatisasi
type CreateUsersProfileFormalDTO struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`

	FatherName    *string `json:"father_name" validate:"omitempty,max=50"`
	FatherPhone   *string `json:"father_phone" validate:"omitempty,max=20"`
	MotherName    *string `json:"mother_name" validate:"omitempty,max=50"`
	MotherPhone   *string `json:"mother_phone" validate:"omitempty,max=20"`
	Guardian      *string `json:"guardian" validate:"omitempty,max=50"`
	GuardianPhone *string `json:"guardian_phone" validate:"omitempty,max=20"`
}

func (d *CreateUsersProfileFormalDTO) Sanitize() {
	if d.FatherPhone != nil {
		v := NormalizePhone(*d.FatherPhone)
		d.FatherPhone = &v
	}
	if d.MotherPhone != nil {
		v := NormalizePhone(*d.MotherPhone)
		d.MotherPhone = &v
	}
	if d.GuardianPhone != nil {
		v := NormalizePhone(*d.GuardianPhone)
		d.GuardianPhone = &v
	}
}

func (d *CreateUsersProfileFormalDTO) Validate() error {
	return validate.Struct(d)
}

func (d *CreateUsersProfileFormalDTO) ToModel() *model.UsersProfileFormalModel {
	return &model.UsersProfileFormalModel{
		UserID:        d.UserID,
		FatherName:    d.FatherName,
		FatherPhone:   d.FatherPhone,
		MotherName:    d.MotherName,
		MotherPhone:   d.MotherPhone,
		Guardian:      d.Guardian,
		GuardianPhone: d.GuardianPhone,
	}
}

// PATCH (Partial Update)
type UpdateUsersProfileFormalDTO struct {
	FatherName    *string `json:"father_name" validate:"omitempty,max=50"`
	FatherPhone   *string `json:"father_phone" validate:"omitempty,max=20"`
	MotherName    *string `json:"mother_name" validate:"omitempty,max=50"`
	MotherPhone   *string `json:"mother_phone" validate:"omitempty,max=20"`
	Guardian      *string `json:"guardian" validate:"omitempty,max=50"`
	GuardianPhone *string `json:"guardian_phone" validate:"omitempty,max=20"`
}

func (d *UpdateUsersProfileFormalDTO) Sanitize() {
	if d.FatherPhone != nil {
		v := NormalizePhone(*d.FatherPhone)
		d.FatherPhone = &v
	}
	if d.MotherPhone != nil {
		v := NormalizePhone(*d.MotherPhone)
		d.MotherPhone = &v
	}
	if d.GuardianPhone != nil {
		v := NormalizePhone(*d.GuardianPhone)
		d.GuardianPhone = &v
	}
}

func (d *UpdateUsersProfileFormalDTO) Validate() error {
	return validate.Struct(d)
}

// ApplyToModelPartial: hanya timpa field yang != nil
func (d *UpdateUsersProfileFormalDTO) ApplyToModelPartial(m *model.UsersProfileFormalModel) {
	if d.FatherName != nil {
		m.FatherName = d.FatherName
	}
	if d.FatherPhone != nil {
		m.FatherPhone = d.FatherPhone
	}
	if d.MotherName != nil {
		m.MotherName = d.MotherName
	}
	if d.MotherPhone != nil {
		m.MotherPhone = d.MotherPhone
	}
	if d.Guardian != nil {
		m.Guardian = d.Guardian
	}
	if d.GuardianPhone != nil {
		m.GuardianPhone = d.GuardianPhone
	}
}

/* =========================
 * Response DTO
 * ========================= */

type UsersProfileFormalResponse struct {
	ID uuid.UUID `json:"id"`

	UserID uuid.UUID `json:"user_id"`

	FatherName    *string `json:"father_name"`
	FatherPhone   *string `json:"father_phone"`
	MotherName    *string `json:"mother_name"`
	MotherPhone   *string `json:"mother_phone"`
	Guardian      *string `json:"guardian"`
	GuardianPhone *string `json:"guardian_phone"`

	CreatedAt string  `json:"created_at"`
	UpdatedAt *string `json:"updated_at,omitempty"`
	DeletedAt *string `json:"deleted_at,omitempty"`
}

func NewUsersProfileFormalResponse(m *model.UsersProfileFormalModel) UsersProfileFormalResponse {
	var updatedAt, deletedAt *string
	if m.UpdatedAt != nil {
		s := m.UpdatedAt.UTC().Format(timeRFC3339)
		updatedAt = &s
	}
	if m.DeletedAt.Valid {
		s := m.DeletedAt.Time.UTC().Format(timeRFC3339)
		deletedAt = &s
	}
	return UsersProfileFormalResponse{
		ID:            m.ID,
		UserID:        m.UserID,
		FatherName:    m.FatherName,
		FatherPhone:   m.FatherPhone,
		MotherName:    m.MotherName,
		MotherPhone:   m.MotherPhone,
		Guardian:      m.Guardian,
		GuardianPhone: m.GuardianPhone,
		CreatedAt:     m.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt:     updatedAt,
		DeletedAt:     deletedAt,
	}
}

/* =========================
 * Common time format
 * ========================= */

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

// BindCreate: helper ambil JSON, sanitize, validate
func BindCreate(c *fiber.Ctx) (*CreateUsersProfileFormalDTO, error) {
	var body CreateUsersProfileFormalDTO
	if err := c.BodyParser(&body); err != nil {
		return nil, err
	}
	body.Sanitize()
	if err := body.Validate(); err != nil {
		return nil, err
	}
	return &body, nil
}

// BindUpdate: helper ambil JSON, sanitize, validate
func BindUpdate(c *fiber.Ctx) (*UpdateUsersProfileFormalDTO, error) {
	var body UpdateUsersProfileFormalDTO
	if err := c.BodyParser(&body); err != nil {
		return nil, err
	}
	body.Sanitize()
	if err := body.Validate(); err != nil {
		return nil, err
	}
	return &body, nil
}
