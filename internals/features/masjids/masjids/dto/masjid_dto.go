package dto

import (
	"masjidku_backend/internals/features/masjids/masjids/model"
	"strings"
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
	MasjidDomain 		string `json:"masjid_domain"`
	MasjidImageURL     string  `json:"masjid_image_url"`
	MasjidGoogleMapsURL string `json:"masjid_google_maps_url"`
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
	MasjidDomain 		string `json:"masjid_domain"`
	MasjidLocation     string    `json:"masjid_location"`
	MasjidLatitude     float64   `json:"masjid_latitude"`
	MasjidLongitude    float64   `json:"masjid_longitude"`
	MasjidImageURL     string    `json:"masjid_image_url"`
	MasjidGoogleMapsURL string `json:"masjid_google_maps_url"`
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
	var domain string
	if m.MasjidDomain != nil {
		domain = *m.MasjidDomain
	}

	return MasjidResponse{
		MasjidID:            m.MasjidID.String(),
		MasjidName:          m.MasjidName,
		MasjidBioShort:      m.MasjidBioShort,
		MasjidLocation:      m.MasjidLocation,
		MasjidDomain:        domain, // handle nil pointer
		MasjidLatitude:      m.MasjidLatitude,
		MasjidLongitude:     m.MasjidLongitude,
		MasjidImageURL:      m.MasjidImageURL,
		MasjidGoogleMapsURL: m.MasjidGoogleMapsURL,
		MasjidSlug:          m.MasjidSlug,
		MasjidIsVerified:    m.MasjidIsVerified,
		MasjidInstagramURL:  m.MasjidInstagramURL,
		MasjidWhatsappURL:   m.MasjidWhatsappURL,
		MasjidYoutubeURL:    m.MasjidYoutubeURL,
		MasjidCreatedAt:     m.MasjidCreatedAt,
		MasjidUpdatedAt:     m.MasjidUpdatedAt,
	}
}

// 🔁 Konversi dari Request DTO ke Model (untuk insert/update)
func ToModelMasjid(input *MasjidRequest, masjidID uuid.UUID) *model.MasjidModel {
	var domainPtr *string
	if trimmed := strings.TrimSpace(input.MasjidDomain); trimmed != "" {
		domainPtr = &trimmed
	}

	return &model.MasjidModel{
		MasjidID:            masjidID,
		MasjidName:          input.MasjidName,
		MasjidBioShort:      input.MasjidBioShort,
		MasjidLocation:      input.MasjidLocation,
		MasjidDomain:        domainPtr, // 🛠️ pointer only if not empty
		MasjidLatitude:      input.MasjidLatitude,
		MasjidLongitude:     input.MasjidLongitude,
		MasjidImageURL:      input.MasjidImageURL,
		MasjidSlug:          input.MasjidSlug,
		MasjidGoogleMapsURL: input.MasjidGoogleMapsURL,
		MasjidIsVerified:    input.MasjidIsVerified,
		MasjidInstagramURL:  input.MasjidInstagramURL,
		MasjidWhatsappURL:   input.MasjidWhatsappURL,
		MasjidYoutubeURL:    input.MasjidYoutubeURL,
	}
}
