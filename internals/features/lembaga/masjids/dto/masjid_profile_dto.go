package dto

import (
	"masjidku_backend/internals/features/lembaga/masjids/model"
	"time"

	"github.com/google/uuid"
)

type MasjidProfileRequest struct {
	MasjidProfileDescription    string `json:"masjid_profile_description"`
	MasjidProfileFoundedYear    int    `json:"masjid_profile_founded_year"`
	MasjidProfileMasjidID       string `json:"masjid_profile_masjid_id"` // UUID string
	MasjidProfileLogoURL        string `json:"masjid_profile_logo_url"`
	MasjidProfileStampURL       string `json:"masjid_profile_stamp_url"`
	MasjidProfileTTDKetuaDKMURL string `json:"masjid_profile_ttd_ketua_dkm_url"`
}

type MasjidProfileResponse struct {
	MasjidProfileID             string    `json:"masjid_profile_id"`
	MasjidProfileDescription    string    `json:"masjid_profile_description"`
	MasjidProfileFoundedYear    int       `json:"masjid_profile_founded_year"`
	MasjidProfileMasjidID       string    `json:"masjid_profile_masjid_id"`
	MasjidProfileLogoURL        string    `json:"masjid_profile_logo_url"`
	MasjidProfileStampURL       string    `json:"masjid_profile_stamp_url"`
	MasjidProfileTTDKetuaDKMURL string    `json:"masjid_profile_ttd_ketua_dkm_url"`
	MasjidProfileCreatedAt      time.Time `json:"masjid_profile_created_at"`
}

// 🔁 Konversi dari Model ke DTO Response
func FromModelMasjidProfile(profile *model.MasjidProfileModel) MasjidProfileResponse {
	return MasjidProfileResponse{
		MasjidProfileID:             profile.MasjidProfileID.String(),
		MasjidProfileDescription:    profile.MasjidProfileDescription,
		MasjidProfileFoundedYear:    profile.MasjidProfileFoundedYear,
		MasjidProfileMasjidID:       profile.MasjidProfileMasjidID.String(),
		MasjidProfileLogoURL:        profile.MasjidProfileLogoURL,
		MasjidProfileStampURL:       profile.MasjidProfileStampURL,
		MasjidProfileTTDKetuaDKMURL: profile.MasjidProfileTTDKetuaDKMURL,
		MasjidProfileCreatedAt:      profile.MasjidProfileCreatedAt,
	}
}

// 🔁 Konversi dari DTO Request ke Model
func ToModelMasjidProfile(input *MasjidProfileRequest) *model.MasjidProfileModel {
	parsedUUID, _ := uuid.Parse(input.MasjidProfileMasjidID)
	return &model.MasjidProfileModel{
		MasjidProfileDescription:    input.MasjidProfileDescription,
		MasjidProfileFoundedYear:    input.MasjidProfileFoundedYear,
		MasjidProfileMasjidID:       parsedUUID,
		MasjidProfileLogoURL:        input.MasjidProfileLogoURL,
		MasjidProfileStampURL:       input.MasjidProfileStampURL,
		MasjidProfileTTDKetuaDKMURL: input.MasjidProfileTTDKetuaDKMURL,
	}
}
