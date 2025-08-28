// file: internals/features/masjids/masjids/dto/masjid_dto.go
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
   - Domain dikirim string: "" => NULL
========================================================= */

type MasjidRequest struct {
	// Relasi (opsional)
	MasjidYayasanID *uuid.UUID `json:"masjid_yayasan_id,omitempty"`

	// Identitas & lokasi
	MasjidName     string   `json:"masjid_name"`
	MasjidBioShort string   `json:"masjid_bio_short"`
	MasjidLocation string   `json:"masjid_location"`
	MasjidLatitude *float64 `json:"masjid_latitude,omitempty"`
	MasjidLongitude *float64 `json:"masjid_longitude,omitempty"`

	// Media (default + main + background)
	MasjidImageURL     string `json:"masjid_image_url"`
	MasjidImageMainURL string `json:"masjid_image_main_url"`
	MasjidImageBgURL   string `json:"masjid_image_bg_url"`

	// Maps & domain & slug
	MasjidGoogleMapsURL string `json:"masjid_google_maps_url"`
	MasjidDomain        string `json:"masjid_domain"` // "" => NULL
	MasjidSlug          string `json:"masjid_slug"`

	// Aktivasi & Verifikasi (writable)
	MasjidIsActive           bool   `json:"masjid_is_active"`
	MasjidVerificationStatus string `json:"masjid_verification_status"` // 'pending' | 'approved' | 'rejected'
	MasjidVerificationNotes  string `json:"masjid_verification_notes"`

	// Paket aktif (opsional)
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	// Flag sekolah/pesantren
	MasjidIsIslamicSchool bool `json:"masjid_is_islamic_school"`

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
	MasjidID        string   `json:"masjid_id"`
	MasjidYayasanID *uuid.UUID `json:"masjid_yayasan_id,omitempty"`

	MasjidName      string   `json:"masjid_name"`
	MasjidBioShort  string   `json:"masjid_bio_short"`
	MasjidDomain    string   `json:"masjid_domain"`
	MasjidLocation  string   `json:"masjid_location"`
	MasjidLatitude  *float64 `json:"masjid_latitude,omitempty"`
	MasjidLongitude *float64 `json:"masjid_longitude,omitempty"`

	// Media (default)
	MasjidImageURL string `json:"masjid_image_url"`
	// Trash info (default)
	MasjidImageTrashURL           *string    `json:"masjid_image_trash_url,omitempty"`
	MasjidImageDeletePendingUntil *time.Time `json:"masjid_image_delete_pending_until,omitempty"`

	// Media (main) + trash info
	MasjidImageMainURL               string     `json:"masjid_image_main_url"`
	MasjidImageMainTrashURL          *string    `json:"masjid_image_main_trash_url,omitempty"`
	MasjidImageMainDeletePendingUntil *time.Time `json:"masjid_image_main_delete_pending_until,omitempty"`

	// Media (background) + trash info
	MasjidImageBgURL               string     `json:"masjid_image_bg_url"`
	MasjidImageBgTrashURL          *string    `json:"masjid_image_bg_trash_url,omitempty"`
	MasjidImageBgDeletePendingUntil *time.Time `json:"masjid_image_bg_delete_pending_until,omitempty"`

	// Maps & slug
	MasjidGoogleMapsURL string `json:"masjid_google_maps_url"`
	MasjidSlug          string `json:"masjid_slug"`

	// Verifikasi (read-only hasil trigger)
	MasjidIsActive           bool       `json:"masjid_is_active"`
	MasjidIsVerified         bool       `json:"masjid_is_verified"`
	MasjidVerificationStatus string     `json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time `json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  string     `json:"masjid_verification_notes"`

	// Relasi plan
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	// Flag sekolah/pesantren
	MasjidIsIslamicSchool bool `json:"masjid_is_islamic_school"`

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
	// Relasi
	MasjidYayasanID *uuid.UUID `json:"masjid_yayasan_id"`

	// Identitas & lokasi
	MasjidName     *string  `json:"masjid_name"`
	MasjidBioShort *string  `json:"masjid_bio_short"`
	MasjidLocation *string  `json:"masjid_location"`
	MasjidLatitude *float64 `json:"masjid_latitude"`
	MasjidLongitude *float64 `json:"masjid_longitude"`

	// Media (default + main + background)
	MasjidImageURL     *string `json:"masjid_image_url"`       // trigger DB handle trash
	MasjidImageMainURL *string `json:"masjid_image_main_url"`  // trigger DB handle trash
	MasjidImageBgURL   *string `json:"masjid_image_bg_url"`    // trigger DB handle trash

	// Maps & domain & slug
	MasjidGoogleMapsURL *string `json:"masjid_google_maps_url"`
	MasjidDomain        *string `json:"masjid_domain"` // "" => NULL
	MasjidSlug          *string `json:"masjid_slug"`

	// Aktivasi & Verifikasi
	MasjidIsActive           *bool   `json:"masjid_is_active"`
	MasjidVerificationStatus *string `json:"masjid_verification_status"` // trigger set flags
	MasjidVerificationNotes  *string `json:"masjid_verification_notes"`

	// Paket aktif
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id"`

	// Flag sekolah/pesantren
	MasjidIsIslamicSchool *bool `json:"masjid_is_islamic_school"`

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
	return MasjidResponse{
		MasjidID:        m.MasjidID.String(),
		MasjidYayasanID: m.MasjidYayasanID,

		MasjidName:      m.MasjidName,
		MasjidBioShort:  valOrEmpty(m.MasjidBioShort),
		MasjidDomain:    valOrEmpty(m.MasjidDomain),
		MasjidLocation:  valOrEmpty(m.MasjidLocation),
		MasjidLatitude:  m.MasjidLatitude,
		MasjidLongitude: m.MasjidLongitude,

		// default image + trash
		MasjidImageURL:                valOrEmpty(m.MasjidImageURL),
		MasjidImageTrashURL:           m.MasjidImageTrashURL,
		MasjidImageDeletePendingUntil: m.MasjidImageDeletePendingUntil,

		// main image + trash
		MasjidImageMainURL:               valOrEmpty(m.MasjidImageMainURL),
		MasjidImageMainTrashURL:          m.MasjidImageMainTrashURL,
		MasjidImageMainDeletePendingUntil: m.MasjidImageMainDeletePendingUntil,

		// background image + trash
		MasjidImageBgURL:               valOrEmpty(m.MasjidImageBgURL),
		MasjidImageBgTrashURL:          m.MasjidImageBgTrashURL,
		MasjidImageBgDeletePendingUntil: m.MasjidImageBgDeletePendingUntil,

		MasjidGoogleMapsURL: valOrEmpty(m.MasjidGoogleMapsURL),
		MasjidSlug:          m.MasjidSlug,

		MasjidIsActive:           m.MasjidIsActive,
		MasjidIsVerified:         m.MasjidIsVerified,
		MasjidVerificationStatus: string(m.MasjidVerificationStatus),
		MasjidVerifiedAt:         m.MasjidVerifiedAt,
		MasjidVerificationNotes:  valOrEmpty(m.MasjidVerificationNotes),

		MasjidCurrentPlanID:   m.MasjidCurrentPlanID,
		MasjidIsIslamicSchool: m.MasjidIsIslamicSchool,

		MasjidInstagramURL:           valOrEmpty(m.MasjidInstagramURL),
		MasjidWhatsappURL:            valOrEmpty(m.MasjidWhatsappURL),
		MasjidYoutubeURL:             valOrEmpty(m.MasjidYoutubeURL),
		MasjidFacebookURL:            valOrEmpty(m.MasjidFacebookURL),
		MasjidTiktokURL:              valOrEmpty(m.MasjidTiktokURL),
		MasjidWhatsappGroupIkhwanURL: valOrEmpty(m.MasjidWhatsappGroupIkhwanURL),
		MasjidWhatsappGroupAkhwatURL: valOrEmpty(m.MasjidWhatsappGroupAkhwatURL),

		MasjidCreatedAt: m.MasjidCreatedAt,
		MasjidUpdatedAt: m.MasjidUpdatedAt,
	}
}

// ToModelMasjid: buat instance model dari request (untuk INSERT)
func ToModelMasjid(in *MasjidRequest, id uuid.UUID) *model.MasjidModel {
	return &model.MasjidModel{
		MasjidID:          id,
		MasjidYayasanID:   in.MasjidYayasanID,

		MasjidName:        in.MasjidName,
		MasjidBioShort:    normalizeOptionalStringToPtr(in.MasjidBioShort),
		MasjidLocation:    normalizeOptionalStringToPtr(in.MasjidLocation),
		MasjidLatitude:    in.MasjidLatitude,
		MasjidLongitude:   in.MasjidLongitude,

		MasjidImageURL:     normalizeOptionalStringToPtr(in.MasjidImageURL),
		MasjidImageMainURL: normalizeOptionalStringToPtr(in.MasjidImageMainURL),
		MasjidImageBgURL:   normalizeOptionalStringToPtr(in.MasjidImageBgURL),

		MasjidGoogleMapsURL: normalizeOptionalStringToPtr(in.MasjidGoogleMapsURL),
		MasjidDomain:        normalizeOptionalStringToPtr(in.MasjidDomain),
		MasjidSlug:          in.MasjidSlug,

		// Flags/verify — is_verified & verified_at TIDAK di-set manual
		MasjidIsActive:           in.MasjidIsActive,
		MasjidVerificationStatus: model.VerificationStatus(in.MasjidVerificationStatus),
		MasjidVerificationNotes:  normalizeOptionalStringToPtr(in.MasjidVerificationNotes),

		MasjidCurrentPlanID:   in.MasjidCurrentPlanID,
		MasjidIsIslamicSchool: in.MasjidIsIslamicSchool,

		// Sosial
		MasjidInstagramURL:           normalizeOptionalStringToPtr(in.MasjidInstagramURL),
		MasjidWhatsappURL:            normalizeOptionalStringToPtr(in.MasjidWhatsappURL),
		MasjidYoutubeURL:             normalizeOptionalStringToPtr(in.MasjidYoutubeURL),
		MasjidFacebookURL:            normalizeOptionalStringToPtr(in.MasjidFacebookURL),
		MasjidTiktokURL:              normalizeOptionalStringToPtr(in.MasjidTiktokURL),
		MasjidWhatsappGroupIkhwanURL: normalizeOptionalStringToPtr(in.MasjidWhatsappGroupIkhwanURL),
		MasjidWhatsappGroupAkhwatURL: normalizeOptionalStringToPtr(in.MasjidWhatsappGroupAkhwatURL),
	}
}

/* =========================================================
   APPLY UPDATE — patch model dari MasjidUpdateRequest
   (gunakan sebelum uc.DB.Save/Updates)
========================================================= */

func ApplyMasjidUpdate(m *model.MasjidModel, u *MasjidUpdateRequest) {
	// Relasi
	if u.MasjidYayasanID != nil {
		m.MasjidYayasanID = u.MasjidYayasanID
	}

	// Identitas & lokasi
	if u.MasjidName != nil {
		m.MasjidName = *u.MasjidName
	}
	if u.MasjidBioShort != nil {
		m.MasjidBioShort = normalizeOptionalStringToPtr(*u.MasjidBioShort)
	}
	if u.MasjidLocation != nil {
		m.MasjidLocation = normalizeOptionalStringToPtr(*u.MasjidLocation)
	}
	if u.MasjidLatitude != nil {
		m.MasjidLatitude = u.MasjidLatitude
	}
	if u.MasjidLongitude != nil {
		m.MasjidLongitude = u.MasjidLongitude
	}

	// Media (default + main + background) — trigger DB urus trash/due
	if u.MasjidImageURL != nil {
		m.MasjidImageURL = normalizeOptionalStringToPtr(*u.MasjidImageURL)
	}
	if u.MasjidImageMainURL != nil {
		m.MasjidImageMainURL = normalizeOptionalStringToPtr(*u.MasjidImageMainURL)
	}
	if u.MasjidImageBgURL != nil {
		m.MasjidImageBgURL = normalizeOptionalStringToPtr(*u.MasjidImageBgURL)
	}

	// Maps & domain & slug
	if u.MasjidGoogleMapsURL != nil {
		m.MasjidGoogleMapsURL = normalizeOptionalStringToPtr(*u.MasjidGoogleMapsURL)
	}
	if u.MasjidDomain != nil {
		m.MasjidDomain = normalizeOptionalStringToPtr(*u.MasjidDomain)
	}
	if u.MasjidSlug != nil {
		m.MasjidSlug = *u.MasjidSlug
	}

	// Aktivasi & verifikasi
	if u.MasjidIsActive != nil {
		m.MasjidIsActive = *u.MasjidIsActive
	}
	if u.MasjidVerificationStatus != nil {
		m.MasjidVerificationStatus = model.VerificationStatus(*u.MasjidVerificationStatus)
	}
	if u.MasjidVerificationNotes != nil {
		m.MasjidVerificationNotes = normalizeOptionalStringToPtr(*u.MasjidVerificationNotes)
	}
	if u.MasjidCurrentPlanID != nil {
		m.MasjidCurrentPlanID = u.MasjidCurrentPlanID
	}

	// Flag sekolah/pesantren
	if u.MasjidIsIslamicSchool != nil {
		m.MasjidIsIslamicSchool = *u.MasjidIsIslamicSchool
	}

	// Sosial
	if u.MasjidInstagramURL != nil {
		m.MasjidInstagramURL = normalizeOptionalStringToPtr(*u.MasjidInstagramURL)
	}
	if u.MasjidWhatsappURL != nil {
		m.MasjidWhatsappURL = normalizeOptionalStringToPtr(*u.MasjidWhatsappURL)
	}
	if u.MasjidYoutubeURL != nil {
		m.MasjidYoutubeURL = normalizeOptionalStringToPtr(*u.MasjidYoutubeURL)
	}
	if u.MasjidFacebookURL != nil {
		m.MasjidFacebookURL = normalizeOptionalStringToPtr(*u.MasjidFacebookURL)
	}
	if u.MasjidTiktokURL != nil {
		m.MasjidTiktokURL = normalizeOptionalStringToPtr(*u.MasjidTiktokURL)
	}
	if u.MasjidWhatsappGroupIkhwanURL != nil {
		m.MasjidWhatsappGroupIkhwanURL = normalizeOptionalStringToPtr(*u.MasjidWhatsappGroupIkhwanURL)
	}
	if u.MasjidWhatsappGroupAkhwatURL != nil {
		m.MasjidWhatsappGroupAkhwatURL = normalizeOptionalStringToPtr(*u.MasjidWhatsappGroupAkhwatURL)
	}
}

/* =========================================================
   HELPERS
========================================================= */

// "" atau whitespace → nil, selain itu trim (dan untuk domain: lower)
func normalizeOptionalStringToPtr(s string) *string {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return nil
	}
	// khusus domain: lower-case
	return strPtr(trim)
}

// util respon: kembalikan "" jika nil
func valOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strPtr(s string) *string { return &s }
