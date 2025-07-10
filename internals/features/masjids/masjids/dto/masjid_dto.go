package dto

import (
	"masjidku_backend/internals/features/masjids/masjids/model"
	"time"

	"github.com/google/uuid"
)

// 📝 Request DTO untuk CREATE / UPDATE
type MasjidRequest struct {
	MasjidName         string  `json:"masjid_name"`
	MasjidBioShort     string  `json:"masjid_bio_short"`
	MasjidLocation     string  `json:"masjid_location"`
	MasjidLatitude     float64 `json:"masjid_latitude"`
	MasjidLongitude    float64 `json:"masjid_longitude"`
	MasjidImageURL     string  `json:"masjid_image_url"`
	MasjidSlug         string  `json:"masjid_slug"`
	MasjidIsVerified   bool    `json:"masjid_is_verified"`
	MasjidInstagramURL string  `json:"masjid_instagram_url"`
	MasjidWhatsappURL  string  `json:"masjid_whatsapp_url"`
	MasjidYoutubeURL   string  `json:"masjid_youtube_url"`
}

// 📤 Response DTO untuk client
type MasjidResponse struct {
	MasjidID           string    `json:"masjid_id"` // UUID as string
	MasjidName         string    `json:"masjid_name"`
	MasjidBioShort     string    `json:"masjid_bio_short"`
	MasjidLocation     string    `json:"masjid_location"`
	MasjidLatitude     float64   `json:"masjid_latitude"`
	MasjidLongitude    float64   `json:"masjid_longitude"`
	MasjidImageURL     string    `json:"masjid_image_url"`
	MasjidSlug         string    `json:"masjid_slug"`
	MasjidIsVerified   bool      `json:"masjid_is_verified"`
	MasjidInstagramURL string    `json:"masjid_instagram_url"`
	MasjidWhatsappURL  string    `json:"masjid_whatsapp_url"`
	MasjidYoutubeURL   string    `json:"masjid_youtube_url"`
	MasjidCreatedAt    time.Time `json:"masjid_created_at"`
	MasjidUpdatedAt    time.Time `json:"masjid_updated_at"`
}

// 🔁 Konversi dari Model ke Response DTO
func FromModelMasjid(m *model.MasjidModel) MasjidResponse {
	return MasjidResponse{
		MasjidID:           m.MasjidID.String(),
		MasjidName:         m.MasjidName,
		MasjidBioShort:     m.MasjidBioShort,
		MasjidLocation:     m.MasjidLocation,
		MasjidLatitude:     m.MasjidLatitude,
		MasjidLongitude:    m.MasjidLongitude,
		MasjidImageURL:     m.MasjidImageURL,
		MasjidSlug:         m.MasjidSlug,
		MasjidIsVerified:   m.MasjidIsVerified,
		MasjidInstagramURL: m.MasjidInstagramURL,
		MasjidWhatsappURL:  m.MasjidWhatsappURL,
		MasjidYoutubeURL:   m.MasjidYoutubeURL,
		MasjidCreatedAt:    m.MasjidCreatedAt,
		MasjidUpdatedAt:    m.MasjidUpdatedAt,
	}
}

// 🔁 Konversi dari Request DTO ke Model (untuk insert/update)
func ToModelMasjid(input *MasjidRequest, masjidID uuid.UUID) *model.MasjidModel {
	return &model.MasjidModel{
		MasjidID:           masjidID,
		MasjidName:         input.MasjidName,
		MasjidBioShort:     input.MasjidBioShort,
		MasjidLocation:     input.MasjidLocation,
		MasjidLatitude:     input.MasjidLatitude,
		MasjidLongitude:    input.MasjidLongitude,
		MasjidImageURL:     input.MasjidImageURL,
		MasjidSlug:         input.MasjidSlug,
		MasjidIsVerified:   input.MasjidIsVerified,
		MasjidInstagramURL: input.MasjidInstagramURL,
		MasjidWhatsappURL:  input.MasjidWhatsappURL,
		MasjidYoutubeURL:   input.MasjidYoutubeURL,
	}
}
