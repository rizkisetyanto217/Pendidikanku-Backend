// file: internals/features/masjids/masjids/dto/masjid_dto.go
package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"masjidku_backend/internals/features/lembaga/masjids/model"
)

/* =========================================================
   REQUEST DTO — CREATE / UPDATE (writable fields only)
   Catatan:
   - is_verified & verified_at TIDAK diterima dari client
     (diset otomatis lewat trigger saat verification_status berubah)
   - masjid_domain: "" => NULL, disimpan lower-case
========================================================= */

type MasjidRequest struct {
	// Relasi (opsional)
	MasjidYayasanID   *uuid.UUID `json:"masjid_yayasan_id,omitempty"`
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	// Identitas & lokasi
	MasjidName       string   `json:"masjid_name"`
	MasjidBioShort   string   `json:"masjid_bio_short"`
	MasjidLocation   string   `json:"masjid_location"`
	MasjidLatitude   *float64 `json:"masjid_latitude,omitempty"`
	MasjidLongitude  *float64 `json:"masjid_longitude,omitempty"`

	// Domain & slug
	MasjidDomain string `json:"masjid_domain"` // "" => NULL (lower-case)
	MasjidSlug   string `json:"masjid_slug"`

	// Aktivasi & Verifikasi (writable)
	MasjidIsActive           bool   `json:"masjid_is_active"`
	MasjidVerificationStatus string `json:"masjid_verification_status"` // 'pending' | 'approved' | 'rejected'
	MasjidVerificationNotes  string `json:"masjid_verification_notes"`

	// Flag sekolah/pesantren
	MasjidIsIslamicSchool bool `json:"masjid_is_islamic_school"`
}

/* =========================================================
   RESPONSE DTO — lengkap untuk client (sesuai kolom SQL)
========================================================= */

type MasjidResponse struct {
	MasjidID           string     `json:"masjid_id"`
	MasjidYayasanID    *uuid.UUID `json:"masjid_yayasan_id,omitempty"`
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	MasjidName       string   `json:"masjid_name"`
	MasjidBioShort   string   `json:"masjid_bio_short"`
	MasjidDomain     string   `json:"masjid_domain"`
	MasjidSlug       string   `json:"masjid_slug"`
	MasjidLocation   string   `json:"masjid_location"`
	MasjidLatitude   *float64 `json:"masjid_latitude,omitempty"`
	MasjidLongitude  *float64 `json:"masjid_longitude,omitempty"`

	// Verifikasi (read-only hasil trigger)
	MasjidIsActive           bool       `json:"masjid_is_active"`
	MasjidIsVerified         bool       `json:"masjid_is_verified"`
	MasjidVerificationStatus string     `json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time `json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  string     `json:"masjid_verification_notes"`

	// Flag sekolah/pesantren
	MasjidIsIslamicSchool bool `json:"masjid_is_islamic_school"`

	// Audit
	MasjidCreatedAt time.Time `json:"masjid_created_at"`
	MasjidUpdatedAt time.Time `json:"masjid_updated_at"`
}

/* =========================================================
   PARTIAL UPDATE DTO — pointer semua writable fields
========================================================= */

type MasjidUpdateRequest struct {
	// Relasi
	MasjidYayasanID     *uuid.UUID `json:"masjid_yayasan_id"`
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id"`

	// Identitas & lokasi
	MasjidName       *string  `json:"masjid_name"`
	MasjidBioShort   *string  `json:"masjid_bio_short"`
	MasjidLocation   *string  `json:"masjid_location"`
	MasjidLatitude   *float64 `json:"masjid_latitude"`
	MasjidLongitude  *float64 `json:"masjid_longitude"`

	// Domain & slug
	MasjidDomain *string `json:"masjid_domain"` // "" => NULL, lower-case
	MasjidSlug   *string `json:"masjid_slug"`

	// Aktivasi & Verifikasi
	MasjidIsActive           *bool   `json:"masjid_is_active"`
	MasjidVerificationStatus *string `json:"masjid_verification_status"` // trigger set flags
	MasjidVerificationNotes  *string `json:"masjid_verification_notes"`

	// Flag sekolah/pesantren
	MasjidIsIslamicSchool *bool `json:"masjid_is_islamic_school"`
}

/* =========================================================
   KONVERSI MODEL <-> DTO
========================================================= */

func FromModelMasjid(m *model.MasjidModel) MasjidResponse {
	return MasjidResponse{
		MasjidID:           m.MasjidID.String(),
		MasjidYayasanID:    m.MasjidYayasanID,
		MasjidCurrentPlanID: m.MasjidCurrentPlanID,

		MasjidName:      m.MasjidName,
		MasjidBioShort:  valOrEmpty(m.MasjidBioShort),
		MasjidDomain:    valOrEmpty(m.MasjidDomain),
		MasjidSlug:      m.MasjidSlug,
		MasjidLocation:  valOrEmpty(m.MasjidLocation),
		MasjidLatitude:  m.MasjidLatitude,
		MasjidLongitude: m.MasjidLongitude,

		MasjidIsActive:           m.MasjidIsActive,
		MasjidIsVerified:         m.MasjidIsVerified,
		MasjidVerificationStatus: string(m.MasjidVerificationStatus),
		MasjidVerifiedAt:         m.MasjidVerifiedAt,
		MasjidVerificationNotes:  valOrEmpty(m.MasjidVerificationNotes),

		MasjidIsIslamicSchool: m.MasjidIsIslamicSchool,

		MasjidCreatedAt: m.MasjidCreatedAt,
		MasjidUpdatedAt: m.MasjidUpdatedAt,
	}
}

// ToModelMasjid: buat instance model dari request (untuk INSERT)
func ToModelMasjid(in *MasjidRequest, id uuid.UUID) *model.MasjidModel {
	return &model.MasjidModel{
		MasjidID:           id,
		MasjidYayasanID:    in.MasjidYayasanID,
		MasjidCurrentPlanID: in.MasjidCurrentPlanID,

		MasjidName:      in.MasjidName,
		MasjidBioShort:  normalizeOptionalStringToPtr(in.MasjidBioShort),
		MasjidLocation:  normalizeOptionalStringToPtr(in.MasjidLocation),
		MasjidLatitude:  in.MasjidLatitude,
		MasjidLongitude: in.MasjidLongitude,

		MasjidDomain: normalizeDomainToPtr(in.MasjidDomain),
		MasjidSlug:   in.MasjidSlug,

		// Flags/verify — is_verified & verified_at TIDAK di-set manual
		MasjidIsActive:           in.MasjidIsActive,
		MasjidVerificationStatus: model.VerificationStatus(in.MasjidVerificationStatus),
		MasjidVerificationNotes:  normalizeOptionalStringToPtr(in.MasjidVerificationNotes),

		MasjidIsIslamicSchool: in.MasjidIsIslamicSchool,
	}
}

/* =========================================================
   APPLY UPDATE — patch model dari MasjidUpdateRequest
   (gunakan sebelum db.Save / db.Updates)
========================================================= */

func ApplyMasjidUpdate(m *model.MasjidModel, u *MasjidUpdateRequest) {
	// Relasi
	if u.MasjidYayasanID != nil {
		m.MasjidYayasanID = u.MasjidYayasanID
	}
	if u.MasjidCurrentPlanID != nil {
		m.MasjidCurrentPlanID = u.MasjidCurrentPlanID
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

	// Domain & slug
	if u.MasjidDomain != nil {
		m.MasjidDomain = normalizeDomainToPtr(*u.MasjidDomain)
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

	// Flag sekolah/pesantren
	if u.MasjidIsIslamicSchool != nil {
		m.MasjidIsIslamicSchool = *u.MasjidIsIslamicSchool
	}
}

/* =========================================================
   HELPERS
========================================================= */

// "" atau whitespace → nil, selain itu trim
func normalizeOptionalStringToPtr(s string) *string {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return nil
	}
	return strPtr(trim)
}

// Domain: "" -> nil; non-empty -> lower(trim)
func normalizeDomainToPtr(s string) *string {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return nil
	}
	lower := strings.ToLower(trim)
	return &lower
}

// util respon: kembalikan "" jika nil
func valOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strPtr(s string) *string { return &s }
