// file: internals/features/masjid/dto/masjid_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	"masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* ===================== ENUMS mirror ===================== */

type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

type TenantProfile string

const (
	TenantTeacherSolo       TenantProfile = "teacher_solo"
	TenantTeacherPlusSchool TenantProfile = "teacher_plus_school"
	TenantSchoolBasic       TenantProfile = "school_basic"
	TenantSchoolComplex     TenantProfile = "school_complex"
)

/* ===================== REQUESTS ===================== */

type MasjidCreateReq struct {
	MasjidYayasanID     *uuid.UUID `json:"masjid_yayasan_id,omitempty"`
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	MasjidName     string `json:"masjid_name"`
	MasjidBioShort string `json:"masjid_bio_short"`
	MasjidLocation string `json:"masjid_location"`
	MasjidCity     string `json:"masjid_city"`

	MasjidDomain string `json:"masjid_domain"`
	MasjidSlug   string `json:"masjid_slug"`

	MasjidIsActive           bool   `json:"masjid_is_active"`
	MasjidVerificationStatus string `json:"masjid_verification_status"`
	MasjidVerificationNotes  string `json:"masjid_verification_notes"`

	MasjidContactPersonName  string `json:"masjid_contact_person_name"`
	MasjidContactPersonPhone string `json:"masjid_contact_person_phone"`

	MasjidIsIslamicSchool bool   `json:"masjid_is_islamic_school"`
	MasjidTenantProfile   string `json:"masjid_tenant_profile"`

	MasjidLevels []string `json:"masjid_levels"`

	// Media (seed awal; *_old dikelola sistem)
	MasjidIconURL             string `json:"masjid_icon_url"`
	MasjidIconObjectKey       string `json:"masjid_icon_object_key"`
	MasjidLogoURL             string `json:"masjid_logo_url"`
	MasjidLogoObjectKey       string `json:"masjid_logo_object_key"`
	MasjidBackgroundURL       string `json:"masjid_background_url"`
	MasjidBackgroundObjectKey string `json:"masjid_background_object_key"`

	// (Opsional) set kode undangan guru secara langsung (plaintext)
	MasjidTeacherCodePlain string `json:"masjid_teacher_code_plain"`
}

type MasjidUpdateReq struct {
	MasjidYayasanID     *uuid.UUID `json:"masjid_yayasan_id"      form:"masjid_yayasan_id"`
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id" form:"masjid_current_plan_id"`

	MasjidName     *string `json:"masjid_name"      form:"masjid_name"`
	MasjidBioShort *string `json:"masjid_bio_short" form:"masjid_bio_short"`
	MasjidLocation *string `json:"masjid_location"  form:"masjid_location"`
	MasjidCity     *string `json:"masjid_city"      form:"masjid_city"`

	MasjidDomain *string `json:"masjid_domain" form:"masjid_domain"`
	MasjidSlug   *string `json:"masjid_slug"   form:"masjid_slug"`

	MasjidIsActive           *bool   `json:"masjid_is_active"           form:"masjid_is_active"`
	MasjidVerificationStatus *string `json:"masjid_verification_status" form:"masjid_verification_status"`
	MasjidVerificationNotes  *string `json:"masjid_verification_notes"  form:"masjid_verification_notes"`

	MasjidContactPersonName  *string `json:"masjid_contact_person_name"  form:"masjid_contact_person_name"`
	MasjidContactPersonPhone *string `json:"masjid_contact_person_phone" form:"masjid_contact_person_phone"`

	MasjidIsIslamicSchool *bool   `json:"masjid_is_islamic_school" form:"masjid_is_islamic_school"`
	MasjidTenantProfile   *string `json:"masjid_tenant_profile"    form:"masjid_tenant_profile"`

	MasjidLevels *[]string `json:"masjid_levels" form:"masjid_levels"`

	// Media current (PATCH via JSON)
	MasjidIconURL             *string `json:"masjid_icon_url"              form:"masjid_icon_url"`
	MasjidIconObjectKey       *string `json:"masjid_icon_object_key"       form:"masjid_icon_object_key"`
	MasjidLogoURL             *string `json:"masjid_logo_url"              form:"masjid_logo_url"`
	MasjidLogoObjectKey       *string `json:"masjid_logo_object_key"       form:"masjid_logo_object_key"`
	MasjidBackgroundURL       *string `json:"masjid_background_url"        form:"masjid_background_url"`
	MasjidBackgroundObjectKey *string `json:"masjid_background_object_key" form:"masjid_background_object_key"`

	// Rotate/set ulang kode undangan guru (plaintext)
	MasjidTeacherCodePlain *string `json:"masjid_teacher_code_plain" form:"masjid_teacher_code_plain"`

	// Clear → NULL/empty eksplisit pada kolom tertentu
	Clear []string `json:"__clear,omitempty" form:"__clear"`
}

/* ===================== RESPONSE ===================== */

type MasjidResp struct {
	MasjidID            string     `json:"masjid_id"`
	MasjidYayasanID     *uuid.UUID `json:"masjid_yayasan_id,omitempty"`
	MasjidCurrentPlanID *uuid.UUID `json:"masjid_current_plan_id,omitempty"`

	MasjidName     string `json:"masjid_name"`
	MasjidBioShort string `json:"masjid_bio_short"`
	MasjidDomain   string `json:"masjid_domain"`
	MasjidSlug     string `json:"masjid_slug"`
	MasjidLocation string `json:"masjid_location"`
	MasjidCity     string `json:"masjid_city"`

	MasjidIsActive           bool       `json:"masjid_is_active"`
	MasjidIsVerified         bool       `json:"masjid_is_verified"`
	MasjidVerificationStatus string     `json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time `json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  string     `json:"masjid_verification_notes"`

	MasjidContactPersonName  string `json:"masjid_contact_person_name"`
	MasjidContactPersonPhone string `json:"masjid_contact_person_phone"`

	MasjidIsIslamicSchool bool   `json:"masjid_is_islamic_school"`
	MasjidTenantProfile   string `json:"masjid_tenant_profile"`

	MasjidLevels []string `json:"masjid_levels"`

	// Teacher code (tidak expose hash)
	MasjidHasTeacherCode   bool       `json:"masjid_has_teacher_code"`
	MasjidTeacherCodeSetAt *time.Time `json:"masjid_teacher_code_set_at,omitempty"`

	// ICON
	MasjidIconURL                string     `json:"masjid_icon_url"`
	MasjidIconObjectKey          string     `json:"masjid_icon_object_key"`
	MasjidIconURLOld             string     `json:"masjid_icon_url_old"`
	MasjidIconObjectKeyOld       string     `json:"masjid_icon_object_key_old"`
	MasjidIconDeletePendingUntil *time.Time `json:"masjid_icon_delete_pending_until,omitempty"`

	// LOGO
	MasjidLogoURL                string     `json:"masjid_logo_url"`
	MasjidLogoObjectKey          string     `json:"masjid_logo_object_key"`
	MasjidLogoURLOld             string     `json:"masjid_logo_url_old"`
	MasjidLogoObjectKeyOld       string     `json:"masjid_logo_object_key_old"`
	MasjidLogoDeletePendingUntil *time.Time `json:"masjid_logo_delete_pending_until,omitempty"`

	// BACKGROUND
	MasjidBackgroundURL                string     `json:"masjid_background_url"`
	MasjidBackgroundObjectKey          string     `json:"masjid_background_object_key"`
	MasjidBackgroundURLOld             string     `json:"masjid_background_url_old"`
	MasjidBackgroundObjectKeyOld       string     `json:"masjid_background_object_key_old"`
	MasjidBackgroundDeletePendingUntil *time.Time `json:"masjid_background_delete_pending_until,omitempty"`

	MasjidCreatedAt      time.Time  `json:"masjid_created_at"`
	MasjidUpdatedAt      time.Time  `json:"masjid_updated_at"`
	MasjidLastActivityAt *time.Time `json:"masjid_last_activity_at,omitempty"`
}

/* ===================== CONVERTERS ===================== */

func FromModel(m *model.MasjidModel) MasjidResp {
	levels := levelsFromJSON(m.MasjidLevels)

	return MasjidResp{
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

		MasjidHasTeacherCode:   len(m.MasjidTeacherCodeHash) > 0,
		MasjidTeacherCodeSetAt: m.MasjidTeacherCodeSetAt,

		// ICON
		MasjidIconURL:                valOrEmpty(m.MasjidIconURL),
		MasjidIconObjectKey:          valOrEmpty(m.MasjidIconObjectKey),
		MasjidIconURLOld:             valOrEmpty(m.MasjidIconURLOld),
		MasjidIconObjectKeyOld:       valOrEmpty(m.MasjidIconObjectKeyOld),
		MasjidIconDeletePendingUntil: m.MasjidIconDeletePendingUntil,

		// LOGO
		MasjidLogoURL:                valOrEmpty(m.MasjidLogoURL),
		MasjidLogoObjectKey:          valOrEmpty(m.MasjidLogoObjectKey),
		MasjidLogoURLOld:             valOrEmpty(m.MasjidLogoURLOld),
		MasjidLogoObjectKeyOld:       valOrEmpty(m.MasjidLogoObjectKeyOld),
		MasjidLogoDeletePendingUntil: m.MasjidLogoDeletePendingUntil,

		// BACKGROUND
		MasjidBackgroundURL:                valOrEmpty(m.MasjidBackgroundURL),
		MasjidBackgroundObjectKey:          valOrEmpty(m.MasjidBackgroundObjectKey),
		MasjidBackgroundURLOld:             valOrEmpty(m.MasjidBackgroundURLOld),
		MasjidBackgroundObjectKeyOld:       valOrEmpty(m.MasjidBackgroundObjectKeyOld),
		MasjidBackgroundDeletePendingUntil: m.MasjidBackgroundDeletePendingUntil,

		MasjidCreatedAt:      m.MasjidCreatedAt,
		MasjidUpdatedAt:      m.MasjidUpdatedAt,
		MasjidLastActivityAt: m.MasjidLastActivityAt,
	}
}

func ToModel(in *MasjidCreateReq, id uuid.UUID) *model.MasjidModel {
	out := &model.MasjidModel{
		MasjidID:            id,
		MasjidYayasanID:     in.MasjidYayasanID,
		MasjidCurrentPlanID: in.MasjidCurrentPlanID,

		MasjidName:     strings.TrimSpace(in.MasjidName),
		MasjidBioShort: optStrPtr(in.MasjidBioShort),
		MasjidLocation: optStrPtr(in.MasjidLocation),
		MasjidCity:     optStrPtr(in.MasjidCity),

		MasjidDomain: normDomainPtr(in.MasjidDomain),
		MasjidSlug:   strings.TrimSpace(in.MasjidSlug),

		MasjidIsActive:           in.MasjidIsActive,
		MasjidVerificationStatus: model.VerificationStatus(normVerification(in.MasjidVerificationStatus)),
		MasjidVerificationNotes:  optStrPtr(in.MasjidVerificationNotes),

		MasjidContactPersonName:  optStrPtr(in.MasjidContactPersonName),
		MasjidContactPersonPhone: optStrPtr(in.MasjidContactPersonPhone),

		MasjidIsIslamicSchool: in.MasjidIsIslamicSchool,
		MasjidTenantProfile:   model.TenantProfile(normTenantProfile(in.MasjidTenantProfile)),

		MasjidIconURL:             optStrPtr(in.MasjidIconURL),
		MasjidIconObjectKey:       optStrPtr(in.MasjidIconObjectKey),
		MasjidLogoURL:             optStrPtr(in.MasjidLogoURL),
		MasjidLogoObjectKey:       optStrPtr(in.MasjidLogoObjectKey),
		MasjidBackgroundURL:       optStrPtr(in.MasjidBackgroundURL),
		MasjidBackgroundObjectKey: optStrPtr(in.MasjidBackgroundObjectKey),
	}

	// Levels
	if len(in.MasjidLevels) > 0 {
		out.MasjidLevels = levelsToJSON(in.MasjidLevels)
	}

	// Teacher invite code (plaintext) → hash akan diisi di service (bukan di DTO)
	// Di sini cukup tandai SetAt agar service bisa mengisi kalau diperlukan.
	if strings.TrimSpace(in.MasjidTeacherCodePlain) != "" {
		now := time.Now()
		out.MasjidTeacherCodeSetAt = &now
	}

	return out
}

func ApplyUpdate(m *model.MasjidModel, u *MasjidUpdateReq) {
	// Relasi
	if u.MasjidYayasanID != nil {
		m.MasjidYayasanID = u.MasjidYayasanID
	}
	if u.MasjidCurrentPlanID != nil {
		m.MasjidCurrentPlanID = u.MasjidCurrentPlanID
	}

	// Identitas & lokasi
	if u.MasjidName != nil {
		m.MasjidName = strings.TrimSpace(*u.MasjidName)
	}
	if u.MasjidBioShort != nil {
		m.MasjidBioShort = optStrPtr(strings.TrimSpace(*u.MasjidBioShort))
	}
	if u.MasjidLocation != nil {
		m.MasjidLocation = optStrPtr(strings.TrimSpace(*u.MasjidLocation))
	}
	if u.MasjidCity != nil {
		m.MasjidCity = optStrPtr(strings.TrimSpace(*u.MasjidCity))
	}

	// Domain & slug
	if u.MasjidDomain != nil {
		m.MasjidDomain = normDomainPtr(*u.MasjidDomain)
	}
	if u.MasjidSlug != nil {
		m.MasjidSlug = strings.TrimSpace(*u.MasjidSlug)
	}

	// Aktivasi & verifikasi
	if u.MasjidIsActive != nil {
		m.MasjidIsActive = *u.MasjidIsActive
	}
	if u.MasjidVerificationStatus != nil {
		m.MasjidVerificationStatus = model.VerificationStatus(normVerification(*u.MasjidVerificationStatus))
	}
	if u.MasjidVerificationNotes != nil {
		m.MasjidVerificationNotes = optStrPtr(strings.TrimSpace(*u.MasjidVerificationNotes))
	}

	// Kontak
	if u.MasjidContactPersonName != nil {
		m.MasjidContactPersonName = optStrPtr(strings.TrimSpace(*u.MasjidContactPersonName))
	}
	if u.MasjidContactPersonPhone != nil {
		m.MasjidContactPersonPhone = optStrPtr(strings.TrimSpace(*u.MasjidContactPersonPhone))
	}

	// Flag & profil
	if u.MasjidIsIslamicSchool != nil {
		m.MasjidIsIslamicSchool = *u.MasjidIsIslamicSchool
	}
	if u.MasjidTenantProfile != nil {
		m.MasjidTenantProfile = model.TenantProfile(normTenantProfile(*u.MasjidTenantProfile))
	}

	// Levels
	if u.MasjidLevels != nil {
		m.MasjidLevels = levelsToJSON(*u.MasjidLevels)
	}

	// Media current (PATCH)
	if u.MasjidIconURL != nil {
		m.MasjidIconURL = optStrPtr(strings.TrimSpace(*u.MasjidIconURL))
	}
	if u.MasjidIconObjectKey != nil {
		m.MasjidIconObjectKey = optStrPtr(strings.TrimSpace(*u.MasjidIconObjectKey))
	}
	if u.MasjidLogoURL != nil {
		m.MasjidLogoURL = optStrPtr(strings.TrimSpace(*u.MasjidLogoURL))
	}
	if u.MasjidLogoObjectKey != nil {
		m.MasjidLogoObjectKey = optStrPtr(strings.TrimSpace(*u.MasjidLogoObjectKey))
	}
	if u.MasjidBackgroundURL != nil {
		m.MasjidBackgroundURL = optStrPtr(strings.TrimSpace(*u.MasjidBackgroundURL))
	}
	if u.MasjidBackgroundObjectKey != nil {
		m.MasjidBackgroundObjectKey = optStrPtr(strings.TrimSpace(*u.MasjidBackgroundObjectKey))
	}

	// Teacher code: set/rotate via plaintext (hashing dikerjakan di service)
	if u.MasjidTeacherCodePlain != nil && strings.TrimSpace(*u.MasjidTeacherCodePlain) != "" {
		now := time.Now()
		m.MasjidTeacherCodeSetAt = &now
		// m.MasjidTeacherCodeHash akan diisi oleh service setelah hashing
	}

	// Clear → NULL/empty eksplisit
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
			m.MasjidLevels = nil // datatypes.JSON(nil) ⇒ NULL
		case "masjid_icon_url":
			m.MasjidIconURL = nil
		case "masjid_icon_object_key":
			m.MasjidIconObjectKey = nil
		case "masjid_logo_url":
			m.MasjidLogoURL = nil
		case "masjid_logo_object_key":
			m.MasjidLogoObjectKey = nil
		case "masjid_background_url":
			m.MasjidBackgroundURL = nil
		case "masjid_background_object_key":
			m.MasjidBackgroundObjectKey = nil
		case "masjid_teacher_code":
			m.MasjidTeacherCodeHash = nil
			m.MasjidTeacherCodeSetAt = nil
		}
	}
}

/* ===================== Helpers ===================== */

func optStrPtr(s string) *string {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return nil
	}
	return &trim
}

func normDomainPtr(s string) *string {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return nil
	}
	lower := strings.ToLower(trim)
	return &lower
}

func valOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func normVerification(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "approved":
		return "approved"
	case "rejected":
		return "rejected"
	default:
		return "pending"
	}
}

func normTenantProfile(s string) string {
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

func levelsFromJSON(js datatypes.JSON) []string {
	if len(js) == 0 {
		return []string{}
	}
	var out []string
	_ = json.Unmarshal(js, &out)
	return out
}

func levelsToJSON(src []string) datatypes.JSON {
	if len(src) == 0 {
		return nil // akan tersimpan sebagai NULL
	}
	b, _ := json.Marshal(src)
	return datatypes.JSON(b)
}
