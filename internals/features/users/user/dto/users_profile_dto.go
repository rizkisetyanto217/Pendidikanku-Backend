package dto

import (
	"errors"
	"strings"
	"time"

	profilemodel "masjidku_backend/internals/features/users/user/model"

	"github.com/google/uuid"
)

/* ===========================
   Response DTO
   =========================== */

type UsersProfileDTO struct {
	ID                      uuid.UUID  `json:"id"`
	UserID                  uuid.UUID  `json:"user_id"`
	DonationName            string     `json:"donation_name"`
	PhotoURL                *string    `json:"photo_url,omitempty"`
	PhotoTrashURL           *string    `json:"photo_trash_url,omitempty"`
	PhotoDeletePendingUntil *time.Time `json:"photo_delete_pending_until,omitempty"`
	DateOfBirth             *time.Time `json:"date_of_birth,omitempty"`
	Gender                  *string    `json:"gender,omitempty"` // "male" / "female"
	Location                *string    `json:"location,omitempty"`
	Occupation              *string    `json:"occupation,omitempty"`
	PhoneNumber             *string    `json:"phone_number,omitempty"`
	Bio                     *string    `json:"bio,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// Model → DTO
func ToUsersProfileDTO(m profilemodel.UsersProfileModel) UsersProfileDTO {
	var genderStr *string
	if m.Gender != nil {
		g := string(*m.Gender)
		genderStr = &g
	}

	var deletedAtPtr *time.Time
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		deletedAtPtr = &t
	}

	return UsersProfileDTO{
		ID:                      m.ID,
		UserID:                  m.UserID,
		DonationName:            m.DonationName,
		PhotoURL:                m.PhotoURL,
		PhotoTrashURL:           m.PhotoTrashURL,
		PhotoDeletePendingUntil: m.PhotoDeletePendingUntil,
		DateOfBirth:             m.DateOfBirth,
		Gender:                  genderStr,
		Location:                m.Location,
		Occupation:              m.Occupation,
		PhoneNumber:             m.PhoneNumber,
		Bio:                     m.Bio,
		CreatedAt:               m.CreatedAt,
		UpdatedAt:               m.UpdatedAt,
		DeletedAt:               deletedAtPtr,
	}
}

func ToUsersProfileDTOs(list []profilemodel.UsersProfileModel) []UsersProfileDTO {
	out := make([]UsersProfileDTO, 0, len(list))
	for _, it := range list {
		out = append(out, ToUsersProfileDTO(it))
	}
	return out
}

/* ===========================
   Request DTOs
   =========================== */

// Create: semua optional (partial allowed); date_of_birth dikirim "YYYY-MM-DD"
type CreateUsersProfileRequest struct {
	DonationName            string  `json:"donation_name" validate:"omitempty,max=50"`
	PhotoURL                *string `json:"photo_url,omitempty" validate:"omitempty,url,max=255"`
	DateOfBirth             *string `json:"date_of_birth,omitempty" validate:"omitempty"` // "2006-01-02"
	Gender                  *string `json:"gender,omitempty" validate:"omitempty,oneof=male female"`
	Location                *string `json:"location,omitempty" validate:"omitempty,max=100"`
	Occupation              *string `json:"occupation,omitempty" validate:"omitempty,max=50"`
	PhoneNumber             *string `json:"phone_number,omitempty" validate:"omitempty,max=20"`
	Bio                     *string `json:"bio,omitempty" validate:"omitempty,max=300"`
	PhotoTrashURL           *string `json:"photo_trash_url,omitempty" validate:"omitempty,url"`
	PhotoDeletePendingUntil *string `json:"photo_delete_pending_until,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// Update: semua pointer agar bisa partial update; date_of_birth "YYYY-MM-DD"
type UpdateUsersProfileRequest struct {
	DonationName            *string `json:"donation_name" validate:"omitempty,max=50"`
	PhotoURL                *string `json:"photo_url" validate:"omitempty,url,max=255"`
	PhotoTrashURL           *string `json:"photo_trash_url" validate:"omitempty,url"`
	PhotoDeletePendingUntil *string `json:"photo_delete_pending_until" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	DateOfBirth             *string `json:"date_of_birth" validate:"omitempty"` // "2006-01-02"
	Gender                  *string `json:"gender" validate:"omitempty,oneof=male female"`
	Location                *string `json:"location" validate:"omitempty,max=100"`
	Occupation              *string `json:"occupation" validate:"omitempty,max=50"`
	PhoneNumber             *string `json:"phone_number" validate:"omitempty,max=20"`
	Bio                     *string `json:"bio" validate:"omitempty,max=300"`
}

/* ===========================
   Converters / Appliers
   =========================== */

// Create → Model (UserID wajib dari context)
func (r CreateUsersProfileRequest) ToModel(userID uuid.UUID) profilemodel.UsersProfileModel {
	m := profilemodel.UsersProfileModel{
		UserID:                  userID,
		DonationName:            strings.TrimSpace(r.DonationName),
		PhotoURL:                trimPtr(r.PhotoURL),
		Location:                trimPtr(r.Location),
		Occupation:              trimPtr(r.Occupation),
		PhoneNumber:             trimPtr(r.PhoneNumber),
		Bio:                     trimPtr(r.Bio),
		PhotoTrashURL:           trimPtr(r.PhotoTrashURL),
		PhotoDeletePendingUntil: parseRFC3339Ptr(r.PhotoDeletePendingUntil),
	}

	// date_of_birth
	if r.DateOfBirth != nil && strings.TrimSpace(*r.DateOfBirth) != "" {
		if dob, err := time.Parse("2006-01-02", strings.TrimSpace(*r.DateOfBirth)); err == nil {
			m.DateOfBirth = &dob
		}
	}

	// gender
	if g := parseGenderPtr(r.Gender); g != nil {
		m.Gender = g
	}

	return m
}

// ToUpdateMap: hasilkan map kolom→nilai yang benar-benar diubah.
func (in *UpdateUsersProfileRequest) ToUpdateMap() (map[string]interface{}, error) {
	m := map[string]interface{}{}

	if in.DonationName != nil {
		m["donation_name"] = strings.TrimSpace(*in.DonationName)
	}
	if in.PhotoURL != nil {
		m["photo_url"] = strings.TrimSpace(*in.PhotoURL)
	}
	if in.PhotoTrashURL != nil {
		m["photo_trash_url"] = strings.TrimSpace(*in.PhotoTrashURL)
	}
	if in.PhotoDeletePendingUntil != nil {
		if strings.TrimSpace(*in.PhotoDeletePendingUntil) == "" {
			m["photo_delete_pending_until"] = nil
		} else {
			ts, err := time.Parse(time.RFC3339, strings.TrimSpace(*in.PhotoDeletePendingUntil))
			if err != nil {
				return nil, errors.New("photo_delete_pending_until must be RFC3339 timestamp")
			}
			m["photo_delete_pending_until"] = ts
		}
	}
	if in.DateOfBirth != nil {
		if strings.TrimSpace(*in.DateOfBirth) == "" {
			m["date_of_birth"] = nil
		} else {
			dob, err := time.Parse("2006-01-02", strings.TrimSpace(*in.DateOfBirth))
			if err != nil {
				return nil, errors.New("date_of_birth must be YYYY-MM-DD")
			}
			m["date_of_birth"] = dob
		}
	}
	if in.Gender != nil {
		g := strings.ToLower(strings.TrimSpace(*in.Gender))
		if g != string(profilemodel.Male) && g != string(profilemodel.Female) {
			return nil, errors.New("gender must be 'male' or 'female'")
		}
		m["gender"] = g
	}
	if in.Location != nil {
		m["location"] = strings.TrimSpace(*in.Location)
	}
	if in.Occupation != nil {
		m["occupation"] = strings.TrimSpace(*in.Occupation)
	}
	if in.PhoneNumber != nil {
		m["phone_number"] = strings.TrimSpace(*in.PhoneNumber)
	}
	if in.Bio != nil {
		m["bio"] = strings.TrimSpace(*in.Bio)
	}
	return m, nil
}

/* ===========================
   Helpers
   =========================== */

func parseGenderPtr(s *string) *profilemodel.Gender {
	if s == nil {
		return nil
	}
	val := strings.ToLower(strings.TrimSpace(*s))
	switch val {
	case string(profilemodel.Male), string(profilemodel.Female):
		g := profilemodel.Gender(val)
		return &g
	default:
		return nil
	}
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func parseRFC3339Ptr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	if ts, err := time.Parse(time.RFC3339, t); err == nil {
		return &ts
	}
	return nil
}
