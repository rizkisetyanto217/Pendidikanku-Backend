package snapsvc

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =========================================================
   USER PROFILE SNAPSHOT  (city → location)
========================================================= */

type UserProfileSnapshot struct {
	ID                uuid.UUID `json:"id"` // user_profile_user_id
	Name              string    `json:"name"`
	AvatarURL         *string   `json:"avatar_url,omitempty"`
	WhatsappURL       *string   `json:"whatsapp_url,omitempty"`
	ParentName        *string   `json:"parent_name,omitempty"`
	ParentWhatsappURL *string   `json:"parent_whatsapp_url,omitempty"`
	Slug              *string   `json:"slug,omitempty"`
	DonationName      *string   `json:"donation_name,omitempty"`
	Location          *string   `json:"location,omitempty"` // ← pakai user_profile_location
	Gender            *string   `json:"gender,omitempty"`   // ← NEW: user_profile_gender
}

// BuildUserProfileSnapshotByUserID membuat snapshot berdasar user_profile_user_id
func BuildUserProfileSnapshotByUserID(
	ctx context.Context,
	tx *gorm.DB,
	userID uuid.UUID,
) (*UserProfileSnapshot, error) {
	var row struct {
		ProfileID         uuid.UUID
		UserID            uuid.UUID
		FullNameSnapshot  *string
		DonationName      *string
		Slug              *string
		AvatarURL         *string
		WhatsappURL       *string
		ParentName        *string
		ParentWhatsappURL *string
		Location          *string
		Gender            *string
	}

	if err := tx.WithContext(ctx).Raw(`
		SELECT
			user_profile_id                  AS profile_id,
			user_profile_user_id             AS user_id,
			user_profile_full_name_snapshot  AS full_name_snapshot,
			user_profile_donation_name       AS donation_name,
			user_profile_slug                AS slug,
			user_profile_avatar_url          AS avatar_url,
			user_profile_whatsapp_url        AS whatsapp_url,
			user_profile_parent_name         AS parent_name,
			user_profile_parent_whatsapp_url AS parent_whatsapp_url,
			user_profile_location            AS location,
			user_profile_gender              AS gender
		FROM user_profiles
		WHERE user_profile_user_id = ?
		  AND user_profile_deleted_at IS NULL
		LIMIT 1
	`, userID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ProfileID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	nz := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	name := ""
	if s := nz(row.FullNameSnapshot); s != nil {
		name = *s
	} else if s := nz(row.DonationName); s != nil {
		name = *s
	} else if s := nz(row.Slug); s != nil {
		name = *s
	}

	return &UserProfileSnapshot{
		ID:                row.UserID,
		Name:              name,
		AvatarURL:         nz(row.AvatarURL),
		WhatsappURL:       nz(row.WhatsappURL),
		ParentName:        nz(row.ParentName),
		ParentWhatsappURL: nz(row.ParentWhatsappURL),
		Slug:              nz(row.Slug),
		DonationName:      nz(row.DonationName),
		Location:          nz(row.Location),
		Gender:            nz(row.Gender),
	}, nil
}

// BuildUserProfileSnapshotByProfileID membuat snapshot berdasar user_profile_id
func BuildUserProfileSnapshotByProfileID(
	ctx context.Context,
	tx *gorm.DB,
	profileID uuid.UUID,
) (*UserProfileSnapshot, error) {
	var row struct {
		ProfileID         uuid.UUID
		UserID            uuid.UUID
		FullNameSnapshot  *string
		DonationName      *string
		Slug              *string
		AvatarURL         *string
		WhatsappURL       *string
		ParentName        *string
		ParentWhatsappURL *string
		Location          *string
		Gender            *string
	}

	if err := tx.WithContext(ctx).Raw(`
		SELECT
			user_profile_id                  AS profile_id,
			user_profile_user_id             AS user_id,
			user_profile_full_name_snapshot  AS full_name_snapshot,
			user_profile_donation_name       AS donation_name,
			user_profile_slug                AS slug,
			user_profile_avatar_url          AS avatar_url,
			user_profile_whatsapp_url        AS whatsapp_url,
			user_profile_parent_name         AS parent_name,
			user_profile_parent_whatsapp_url AS parent_whatsapp_url,
			user_profile_location            AS location,
			user_profile_gender              AS gender
		FROM user_profiles
		WHERE user_profile_id = ?
		  AND user_profile_deleted_at IS NULL
		LIMIT 1
	`, profileID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ProfileID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	nz := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	name := ""
	if s := nz(row.FullNameSnapshot); s != nil {
		name = *s
	} else if s := nz(row.DonationName); s != nil {
		name = *s
	} else if s := nz(row.Slug); s != nil {
		name = *s
	}

	return &UserProfileSnapshot{
		ID:                row.UserID,
		Name:              name,
		AvatarURL:         nz(row.AvatarURL),
		WhatsappURL:       nz(row.WhatsappURL),
		ParentName:        nz(row.ParentName),
		ParentWhatsappURL: nz(row.ParentWhatsappURL),
		Slug:              nz(row.Slug),
		DonationName:      nz(row.DonationName),
		Location:          nz(row.Location),
		Gender:            nz(row.Gender),
	}, nil
}
