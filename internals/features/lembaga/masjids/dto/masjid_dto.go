// file: internals/features/masjids/masjids/dto/masjid_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"masjidku_backend/internals/features/lembaga/masjids/model"
)

/* =========================================================
   REQUEST DTO — CREATE (writable fields only)
   Catatan:
   - is_verified & verified_at TIDAK diterima dari client
   - masjid_domain: "" => NULL, disimpan lower-case
   - masjid_levels: optional (tags), contoh: ["kursus","ilmu_quran"]
   - masjid_tenant_profile: "teacher_solo" | "teacher_plus_school" | "school_basic" | "school_complex"
========================================================= */

type MasjidRequest struct {
	// Relasi (opsional)
	MasjidYayasanID     *uuid.UUID `json:"masjid_yayasan_id,omitempty"`
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	// Identitas & lokasi ringkas
	MasjidName     string `json:"masjid_name"`
	MasjidBioShort string `json:"masjid_bio_short"`
	MasjidLocation string `json:"masjid_location"`
	MasjidCity     string `json:"masjid_city"`

	// Domain & slug
	MasjidDomain string `json:"masjid_domain"` // "" => NULL (lower-case)
	MasjidSlug   string `json:"masjid_slug"`

	// Aktivasi & Verifikasi
	MasjidIsActive           bool   `json:"masjid_is_active"`
	MasjidVerificationStatus string `json:"masjid_verification_status"` // 'pending' | 'approved' | 'rejected'
	MasjidVerificationNotes  string `json:"masjid_verification_notes"`

	// Kontak
	MasjidContactPersonName  string `json:"masjid_contact_person_name"`
	MasjidContactPersonPhone string `json:"masjid_contact_person_phone"`

	// Flag & profil tenant
	MasjidIsIslamicSchool bool   `json:"masjid_is_islamic_school"`
	MasjidTenantProfile   string `json:"masjid_tenant_profile"`

	// Levels (tags)
	MasjidLevels []string `json:"masjid_levels"`
}

/* =========================================================
   RESPONSE DTO — lengkap untuk client (sesuai kolom SQL)
========================================================= */

type MasjidResponse struct {
	MasjidID            string     `json:"masjid_id"`
	MasjidYayasanID     *uuid.UUID `json:"masjid_yayasan_id,omitempty"`
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	MasjidName     string `json:"masjid_name"`
	MasjidBioShort string `json:"masjid_bio_short"`
	MasjidDomain   string `json:"masjid_domain"`
	MasjidSlug     string `json:"masjid_slug"`
	MasjidLocation string `json:"masjid_location"`
	MasjidCity     string `json:"masjid_city"`

	// Verifikasi (read-only dari server)
	MasjidIsActive           bool       `json:"masjid_is_active"`
	MasjidIsVerified         bool       `json:"masjid_is_verified"`
	MasjidVerificationStatus string     `json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time `json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  string     `json:"masjid_verification_notes"`

	// Kontak
	MasjidContactPersonName  string `json:"masjid_contact_person_name"`
	MasjidContactPersonPhone string `json:"masjid_contact_person_phone"`

	// Flag & profil tenant
	MasjidIsIslamicSchool bool   `json:"masjid_is_islamic_school"`
	MasjidTenantProfile   string `json:"masjid_tenant_profile"`

	// Levels (tags)
	MasjidLevels []string `json:"masjid_levels"`

	// Audit
	MasjidCreatedAt     time.Time  `json:"masjid_created_at"`
	MasjidUpdatedAt     time.Time  `json:"masjid_updated_at"`
	MasjidLastActivityAt *time.Time `json:"masjid_last_activity_at,omitempty"`
}

/* =========================================================
   PARTIAL UPDATE DTO — pointer semua writable fields
   Catatan:
   - MasjidLevels pakai pointer ke slice; nil = tidak diubah,
     &[]{} = set jadi array kosong.
   - Clear[] untuk set kolom tertentu menjadi NULL eksplisit.
========================================================= */

type MasjidUpdateRequest struct {
	// Relasi
	MasjidYayasanID     *uuid.UUID `json:"masjid_yayasan_id"`
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id"`

	// Identitas & lokasi ringkas
	MasjidName     *string `json:"masjid_name"`
	MasjidBioShort *string `json:"masjid_bio_short"`
	MasjidLocation *string `json:"masjid_location"`
	MasjidCity     *string `json:"masjid_city"`

	// Domain & slug
	MasjidDomain *string `json:"masjid_domain"` // "" => NULL, lower-case
	MasjidSlug   *string `json:"masjid_slug"`

	// Aktivasi & verifikasi
	MasjidIsActive           *bool   `json:"masjid_is_active"`
	MasjidVerificationStatus *string `json:"masjid_verification_status"`
	MasjidVerificationNotes  *string `json:"masjid_verification_notes"`

	// Kontak
	MasjidContactPersonName  *string `json:"masjid_contact_person_name"`
	MasjidContactPersonPhone *string `json:"masjid_contact_person_phone"`

	// Flag & profil tenant
	MasjidIsIslamicSchool *bool   `json:"masjid_is_islamic_school"`
	MasjidTenantProfile   *string `json:"masjid_tenant_profile"`

	// Levels (tags)
	MasjidLevels *[]string `json:"masjid_levels"`

	// Clear → set NULL eksplisit
	Clear []string `json:"__clear,omitempty" validate:"omitempty,dive,oneof=masjid_domain masjid_bio_short masjid_location masjid_city masjid_contact_person_name masjid_contact_person_phone masjid_levels"`
}

/* =========================================================
   KONVERSI MODEL <-> DTO
========================================================= */

func FromModelMasjid(m *model.MasjidModel) MasjidResponse {
	levels, _ := m.GetLevels() // kalau error, biarkan jadi [] kosong
	return MasjidResponse{
		MasjidID:            m.MasjidID.String(),
		MasjidYayasanID:     m.MasjidYayasanID,
		MasjidCurrentPlanID: m.MasjidCurrentPlanID,

		MasjidName:     m.MasjidName,
		MasjidBioShort: valOrEmpty(m.MasjidBioShort),
		MasjidDomain:   valOrEmpty(m.MasjidDomain),
		MasjidSlug:     m.MasjidSlug,
		MasjidLocation: valOrEmpty(m.MasjidLocation),
		MasjidCity:     valOrEmpty(m.MasjidCity),

		MasjidIsActive:           m.MasjidIsActive,
		MasjidIsVerified:         m.MasjidIsVerified,
		MasjidVerificationStatus: string(m.MasjidVerificationStatus),
		MasjidVerifiedAt:         m.MasjidVerifiedAt,
		MasjidVerificationNotes:  valOrEmpty(m.MasjidVerificationNotes),

		MasjidContactPersonName:  valOrEmpty(m.MasjidContactPersonName),
		MasjidContactPersonPhone: valOrEmpty(m.MasjidContactPersonPhone),

		MasjidIsIslamicSchool: m.MasjidIsIslamicSchool,
		MasjidTenantProfile:   string(m.MasjidTenantProfile),
		MasjidLevels:          levels,

		MasjidCreatedAt:      m.MasjidCreatedAt,
		MasjidUpdatedAt:      m.MasjidUpdatedAt,
		MasjidLastActivityAt: m.MasjidLastActivityAt,
	}
}

// ToModelMasjid: buat instance model dari request (untuk INSERT)
func ToModelMasjid(in *MasjidRequest, id uuid.UUID) *model.MasjidModel {
	m := &model.MasjidModel{
		MasjidID:            id,
		MasjidYayasanID:     in.MasjidYayasanID,
		MasjidCurrentPlanID: in.MasjidCurrentPlanID,

		MasjidName:     in.MasjidName,
		MasjidBioShort: normalizeOptionalStringToPtr(in.MasjidBioShort),
		MasjidLocation: normalizeOptionalStringToPtr(in.MasjidLocation),
		MasjidCity:     normalizeOptionalStringToPtr(in.MasjidCity),

		MasjidDomain: normalizeDomainToPtr(in.MasjidDomain),
		MasjidSlug:   in.MasjidSlug,

		MasjidIsActive:           in.MasjidIsActive,
		MasjidVerificationStatus: model.VerificationStatus(normalizeVerification(in.MasjidVerificationStatus)),
		MasjidVerificationNotes:  normalizeOptionalStringToPtr(in.MasjidVerificationNotes),

		MasjidContactPersonName:  normalizeOptionalStringToPtr(in.MasjidContactPersonName),
		MasjidContactPersonPhone: normalizeOptionalStringToPtr(in.MasjidContactPersonPhone),

		MasjidIsIslamicSchool: in.MasjidIsIslamicSchool,
		MasjidTenantProfile:   model.TenantProfile(normalizeTenantProfile(in.MasjidTenantProfile)),
	}

	// Set levels (JSONB) → pointer agar bisa NULL
	if len(in.MasjidLevels) > 0 {
		if b, err := json.Marshal(in.MasjidLevels); err == nil {
			val := datatypes.JSON(b)
			m.MasjidLevels = &val
		}
	}
	return m
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

	// Identitas & lokasi ringkas
	if u.MasjidName != nil {
		m.MasjidName = strings.TrimSpace(*u.MasjidName)
	}
	if u.MasjidBioShort != nil {
		m.MasjidBioShort = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidBioShort))
	}
	if u.MasjidLocation != nil {
		m.MasjidLocation = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidLocation))
	}
	if u.MasjidCity != nil {
		m.MasjidCity = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidCity))
	}

	// Domain & slug
	if u.MasjidDomain != nil {
		m.MasjidDomain = normalizeDomainToPtr(*u.MasjidDomain)
	}
	if u.MasjidSlug != nil {
		m.MasjidSlug = strings.TrimSpace(*u.MasjidSlug)
	}

	// Aktivasi & verifikasi
	if u.MasjidIsActive != nil {
		m.MasjidIsActive = *u.MasjidIsActive
	}
	if u.MasjidVerificationStatus != nil {
		m.MasjidVerificationStatus = model.VerificationStatus(normalizeVerification(*u.MasjidVerificationStatus))
	}
	if u.MasjidVerificationNotes != nil {
		m.MasjidVerificationNotes = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidVerificationNotes))
	}

	// Kontak
	if u.MasjidContactPersonName != nil {
		m.MasjidContactPersonName = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidContactPersonName))
	}
	if u.MasjidContactPersonPhone != nil {
		m.MasjidContactPersonPhone = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidContactPersonPhone))
	}

	// Flag & profil tenant
	if u.MasjidIsIslamicSchool != nil {
		m.MasjidIsIslamicSchool = *u.MasjidIsIslamicSchool
	}
	if u.MasjidTenantProfile != nil {
		m.MasjidTenantProfile = model.TenantProfile(normalizeTenantProfile(*u.MasjidTenantProfile))
	}

	// Levels (tags)
	if u.MasjidLevels != nil {
		// &[]{} → set jadi array kosong (bukan NULL)
		if b, err := json.Marshal(*u.MasjidLevels); err == nil {
			val := datatypes.JSON(b)
			m.MasjidLevels = &val
		}
	}

	// Clear → NULL eksplisit
	for _, col := range u.Clear {
		switch strings.TrimSpace(strings.ToLower(col)) {
		case "masjid_domain":
			m.MasjidDomain = nil
		case "masjid_bio_short":
			m.MasjidBioShort = nil
		case "masjid_location":
			m.MasjidLocation = nil
		case "masjid_city":
			m.MasjidCity = nil
		case "masjid_contact_person_name":
			m.MasjidContactPersonName = nil
		case "masjid_contact_person_phone":
			m.MasjidContactPersonPhone = nil
		case "masjid_levels":
			m.MasjidLevels = nil
		}
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
	return &trim
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

func normalizeVerification(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "approved":
		return "approved"
	case "rejected":
		return "rejected"
	default:
		return "pending"
	}
}

func normalizeTenantProfile(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "teacher_plus_school":
		return "teacher_plus_school"
	case "school_basic":
		return "school_basic"
	case "school_complex":
		return "school_complex"
	default:
		return "teacher_solo"
	}
}
