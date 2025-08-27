package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"masjidku_backend/internals/features/masjids/masjids/model"
)

/* =========================================================
   REQUEST DTO — CREATE / UPDATE (writable fields only)
   Catatan:
   - is_verified & verified_at TIDAK diterima dari client
     (diset otomatis lewat trigger saat verification_status berubah)
========================================================= */

type MasjidRequest struct {
	MasjidName          string   `json:"masjid_name"`
	MasjidBioShort      string   `json:"masjid_bio_short"`
	MasjidLocation      string   `json:"masjid_location"`
	MasjidLatitude      *float64 `json:"masjid_latitude,omitempty"`
	MasjidLongitude     *float64 `json:"masjid_longitude,omitempty"`
	MasjidDomain        string   `json:"masjid_domain"` // kosongkan untuk null
	MasjidImageURL      string   `json:"masjid_image_url"`
	MasjidGoogleMapsURL string   `json:"masjid_google_maps_url"`
	MasjidSlug          string   `json:"masjid_slug"`

	// Aktivasi & Verifikasi (writable)
	MasjidIsActive           bool       `json:"masjid_is_active"`
	MasjidVerificationStatus string     `json:"masjid_verification_status"` // 'pending' | 'approved' | 'rejected'
	MasjidVerificationNotes  string     `json:"masjid_verification_notes"`
	MasjidCurrentPlanID      *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	// Sosial
	MasjidInstagramURL           string `json:"masjid_instagram_url"`
	MasjidWhatsappURL            string `json:"masjid_whatsapp_url"`
	MasjidYoutubeURL             string `json:"masjid_youtube_url"`
	MasjidFacebookURL            string `json:"masjid_facebook_url"`
	MasjidTiktokURL              string `json:"masjid_tiktok_url"`
	MasjidWhatsappGroupIkhwanURL string `json:"masjid_whatsapp_group_ikhwan_url"`
	MasjidWhatsappGroupAkhwatURL string `json:"masjid_whatsapp_group_akhwat_url"`
}

/* =========================================================
   RESPONSE DTO — lengkap untuk client
   Termasuk kolom trash image + due date & flags verifikasi
========================================================= */

type MasjidResponse struct {
	MasjidID          string   `json:"masjid_id"`
	MasjidName        string   `json:"masjid_name"`
	MasjidBioShort    string   `json:"masjid_bio_short"`
	MasjidDomain      string   `json:"masjid_domain"`
	MasjidLocation    string   `json:"masjid_location"`
	MasjidLatitude    *float64 `json:"masjid_latitude,omitempty"`
	MasjidLongitude   *float64 `json:"masjid_longitude,omitempty"`
	MasjidImageURL    string   `json:"masjid_image_url"`
	MasjidGoogleMapsURL string `json:"masjid_google_maps_url"`
	MasjidSlug        string   `json:"masjid_slug"`

	// Image trash & GC info
	MasjidImageTrashURL          *string    `json:"masjid_image_trash_url,omitempty"`
	MasjidImageDeletePendingUntil *time.Time `json:"masjid_image_delete_pending_until,omitempty"`

	// Verifikasi (read-only hasil trigger)
	MasjidIsActive           bool       `json:"masjid_is_active"`
	MasjidIsVerified         bool       `json:"masjid_is_verified"`
	MasjidVerificationStatus string     `json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time `json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  string     `json:"masjid_verification_notes"`

	// Relasi plan
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	// Sosial
	MasjidInstagramURL           string `json:"masjid_instagram_url"`
	MasjidWhatsappURL            string `json:"masjid_whatsapp_url"`
	MasjidYoutubeURL             string `json:"masjid_youtube_url"`
	MasjidFacebookURL            string `json:"masjid_facebook_url"`
	MasjidTiktokURL              string `json:"masjid_tiktok_url"`
	MasjidWhatsappGroupIkhwanURL string `json:"masjid_whatsapp_group_ikhwan_url"`
	MasjidWhatsappGroupAkhwatURL string `json:"masjid_whatsapp_group_akhwat_url"`

	// Audit
	MasjidCreatedAt time.Time `json:"masjid_created_at"`
	MasjidUpdatedAt time.Time `json:"masjid_updated_at"`
}

/* =========================================================
   PARTIAL UPDATE DTO — pointer semua writable fields
========================================================= */

type MasjidUpdateRequest struct {
	MasjidName          *string   `json:"masjid_name"`
	MasjidBioShort      *string   `json:"masjid_bio_short"`
	MasjidLocation      *string   `json:"masjid_location"`
	MasjidLatitude      *float64  `json:"masjid_latitude"`
	MasjidLongitude     *float64  `json:"masjid_longitude"`
	MasjidDomain        *string   `json:"masjid_domain"`          // "" => null-kan
	MasjidImageURL      *string   `json:"masjid_image_url"`       // ubah → aktifkan trash logic
	MasjidGoogleMapsURL *string   `json:"masjid_google_maps_url"`
	MasjidSlug          *string   `json:"masjid_slug"`

	// Aktivasi & Verifikasi
	MasjidIsActive           *bool      `json:"masjid_is_active"`
	MasjidVerificationStatus *string    `json:"masjid_verification_status"` // trigger akan set is_verified/verified_at
	MasjidVerificationNotes  *string    `json:"masjid_verification_notes"`
	MasjidCurrentPlanID      *uuid.UUID `json:"masjid_current_plan_id"`

	// Sosial
	MasjidInstagramURL           *string `json:"masjid_instagram_url"`
	MasjidWhatsappURL            *string `json:"masjid_whatsapp_url"`
	MasjidYoutubeURL             *string `json:"masjid_youtube_url"`
	MasjidFacebookURL            *string `json:"masjid_facebook_url"`
	MasjidTiktokURL              *string `json:"masjid_tiktok_url"`
	MasjidWhatsappGroupIkhwanURL *string `json:"masjid_whatsapp_group_ikhwan_url"`
	MasjidWhatsappGroupAkhwatURL *string `json:"masjid_whatsapp_group_akhwat_url"`
}

/* =========================================================
   KONVERSI MODEL <-> DTO
========================================================= */

func FromModelMasjid(m *model.MasjidModel) MasjidResponse {
	var domain string
	if m.MasjidDomain != nil {
		domain = *m.MasjidDomain
	}

	return MasjidResponse{
		MasjidID:                      m.MasjidID.String(),
		MasjidName:                    m.MasjidName,
		MasjidBioShort:                m.MasjidBioShort,
		MasjidDomain:                  domain,
		MasjidLocation:                m.MasjidLocation,
		MasjidLatitude:                m.MasjidLatitude,
		MasjidLongitude:               m.MasjidLongitude,
		MasjidImageURL:                m.MasjidImageURL,
		MasjidGoogleMapsURL:           m.MasjidGoogleMapsURL,
		MasjidSlug:                    m.MasjidSlug,

		MasjidImageTrashURL:           m.MasjidImageTrashURL,
		MasjidImageDeletePendingUntil: m.MasjidImageDeletePendingUntil,

		MasjidIsActive:           m.MasjidIsActive,
		MasjidIsVerified:         m.MasjidIsVerified,
		MasjidVerificationStatus: m.MasjidVerificationStatus,
		MasjidVerifiedAt:         m.MasjidVerifiedAt,
		MasjidVerificationNotes:  m.MasjidVerificationNotes,

		MasjidCurrentPlanID: m.MasjidCurrentPlanID,

		MasjidInstagramURL:           m.MasjidInstagramURL,
		MasjidWhatsappURL:            m.MasjidWhatsappURL,
		MasjidYoutubeURL:             m.MasjidYoutubeURL,
		MasjidFacebookURL:            m.MasjidFacebookURL,
		MasjidTiktokURL:              m.MasjidTiktokURL,
		MasjidWhatsappGroupIkhwanURL: m.MasjidWhatsappGroupIkhwanURL,
		MasjidWhatsappGroupAkhwatURL: m.MasjidWhatsappGroupAkhwatURL,

		MasjidCreatedAt: m.MasjidCreatedAt,
		MasjidUpdatedAt: m.MasjidUpdatedAt,
	}
}

// ToModelMasjid: buat instance model dari request (untuk INSERT)
func ToModelMasjid(in *MasjidRequest, id uuid.UUID) *model.MasjidModel {
	domainPtr := normalizeOptionalStringToPtr(in.MasjidDomain)

	return &model.MasjidModel{
		MasjidID:                 id,
		MasjidName:               in.MasjidName,
		MasjidBioShort:           in.MasjidBioShort,
		MasjidLocation:           in.MasjidLocation,
		MasjidLatitude:           in.MasjidLatitude,
		MasjidLongitude:          in.MasjidLongitude,
		MasjidDomain:             domainPtr,
		MasjidImageURL:           in.MasjidImageURL,
		MasjidGoogleMapsURL:      in.MasjidGoogleMapsURL,
		MasjidSlug:               in.MasjidSlug,

		// Flags/verify — is_verified & verified_at TIDAK di-set manual
		MasjidIsActive:           in.MasjidIsActive,
		MasjidVerificationStatus: in.MasjidVerificationStatus,
		MasjidVerificationNotes:  in.MasjidVerificationNotes,

		MasjidCurrentPlanID: in.MasjidCurrentPlanID,

		// Sosial
		MasjidInstagramURL:           in.MasjidInstagramURL,
		MasjidWhatsappURL:            in.MasjidWhatsappURL,
		MasjidYoutubeURL:             in.MasjidYoutubeURL,
		MasjidFacebookURL:            in.MasjidFacebookURL,
		MasjidTiktokURL:              in.MasjidTiktokURL,
		MasjidWhatsappGroupIkhwanURL: in.MasjidWhatsappGroupIkhwanURL,
		MasjidWhatsappGroupAkhwatURL: in.MasjidWhatsappGroupAkhwatURL,
	}
}

/* =========================================================
   APPLY UPDATE — patch model dari MasjidUpdateRequest
   (gunakan sebelum uc.DB.Save/Updates)
========================================================= */

func ApplyMasjidUpdate(m *model.MasjidModel, u *MasjidUpdateRequest) {
	if u.MasjidName != nil {
		m.MasjidName = *u.MasjidName
	}
	if u.MasjidBioShort != nil {
		m.MasjidBioShort = *u.MasjidBioShort
	}
	if u.MasjidLocation != nil {
		m.MasjidLocation = *u.MasjidLocation
	}
	if u.MasjidLatitude != nil {
		m.MasjidLatitude = u.MasjidLatitude
	}
	if u.MasjidLongitude != nil {
		m.MasjidLongitude = u.MasjidLongitude
	}
	if u.MasjidDomain != nil {
		m.MasjidDomain = normalizeOptionalStringToPtr(*u.MasjidDomain)
	}
	if u.MasjidImageURL != nil {
		// Mengganti image_url akan di-handle trigger: trash+due 30 hari.
		m.MasjidImageURL = *u.MasjidImageURL
	}
	if u.MasjidGoogleMapsURL != nil {
		m.MasjidGoogleMapsURL = *u.MasjidGoogleMapsURL
	}
	if u.MasjidSlug != nil {
		m.MasjidSlug = *u.MasjidSlug
	}

	if u.MasjidIsActive != nil {
		m.MasjidIsActive = *u.MasjidIsActive
	}
	if u.MasjidVerificationStatus != nil {
		m.MasjidVerificationStatus = *u.MasjidVerificationStatus
		// is_verified & verified_at akan diset otomatis oleh trigger di DB
	}
	if u.MasjidVerificationNotes != nil {
		m.MasjidVerificationNotes = *u.MasjidVerificationNotes
	}
	if u.MasjidCurrentPlanID != nil {
		m.MasjidCurrentPlanID = u.MasjidCurrentPlanID
	}

	// Sosial
	if u.MasjidInstagramURL != nil {
		m.MasjidInstagramURL = *u.MasjidInstagramURL
	}
	if u.MasjidWhatsappURL != nil {
		m.MasjidWhatsappURL = *u.MasjidWhatsappURL
	}
	if u.MasjidYoutubeURL != nil {
		m.MasjidYoutubeURL = *u.MasjidYoutubeURL
	}
	if u.MasjidFacebookURL != nil {
		m.MasjidFacebookURL = *u.MasjidFacebookURL
	}
	if u.MasjidTiktokURL != nil {
		m.MasjidTiktokURL = *u.MasjidTiktokURL
	}
	if u.MasjidWhatsappGroupIkhwanURL != nil {
		m.MasjidWhatsappGroupIkhwanURL = *u.MasjidWhatsappGroupIkhwanURL
	}
	if u.MasjidWhatsappGroupAkhwatURL != nil {
		m.MasjidWhatsappGroupAkhwatURL = *u.MasjidWhatsappGroupAkhwatURL
	}
}

/* =========================================================
   HELPERS
========================================================= */

// "" atau whitespace → nil, selain itu lower-case + trim dikembalikan *string
func normalizeOptionalStringToPtr(s string) *string {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return nil
	}
	// domain case-insensitive → simpan lower
	l := strings.ToLower(trim)
	return &l
}
