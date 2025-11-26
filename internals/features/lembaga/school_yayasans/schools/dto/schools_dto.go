// file: internals/features/school/dto/school_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	"madinahsalam_backend/internals/features/lembaga/school_yayasans/schools/model"

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

type SchoolCreateReq struct {
	SchoolYayasanID     *uuid.UUID `json:"school_yayasan_id,omitempty"`
	SchoolCurrentPlanID *uuid.UUID `json:"school_current_plan_id,omitempty"`

	SchoolName     string `json:"school_name"`
	SchoolBioShort string `json:"school_bio_short"`
	SchoolLocation string `json:"school_location"`
	SchoolCity     string `json:"school_city"`

	SchoolDomain string `json:"school_domain"`
	SchoolSlug   string `json:"school_slug"`

	SchoolIsActive           bool   `json:"school_is_active"`
	SchoolVerificationStatus string `json:"school_verification_status"`
	SchoolVerificationNotes  string `json:"school_verification_notes"`

	SchoolContactPersonName  string `json:"school_contact_person_name"`
	SchoolContactPersonPhone string `json:"school_contact_person_phone"`

	SchoolIsIslamicSchool bool   `json:"school_is_islamic_school"`
	SchoolTenantProfile   string `json:"school_tenant_profile"`

	SchoolLevels []string `json:"school_levels"`

	// Media (seed awal; *_old dikelola sistem)
	SchoolIconURL             string `json:"school_icon_url"`
	SchoolIconObjectKey       string `json:"school_icon_object_key"`
	SchoolLogoURL             string `json:"school_logo_url"`
	SchoolLogoObjectKey       string `json:"school_logo_object_key"`
	SchoolBackgroundURL       string `json:"school_background_url"`
	SchoolBackgroundObjectKey string `json:"school_background_object_key"`

	// (Opsional) set kode undangan guru secara langsung (plaintext)
	SchoolTeacherCodePlain string `json:"school_teacher_code_plain"`
}

type SchoolUpdateReq struct {
	SchoolYayasanID     *uuid.UUID `json:"school_yayasan_id"      form:"school_yayasan_id"`
	SchoolCurrentPlanID *uuid.UUID `json:"school_current_plan_id" form:"school_current_plan_id"`

	SchoolName     *string `json:"school_name"      form:"school_name"`
	SchoolBioShort *string `json:"school_bio_short" form:"school_bio_short"`
	SchoolLocation *string `json:"school_location"  form:"school_location"`
	SchoolCity     *string `json:"school_city"      form:"school_city"`

	SchoolDomain *string `json:"school_domain" form:"school_domain"`
	SchoolSlug   *string `json:"school_slug"   form:"school_slug"`

	SchoolIsActive           *bool   `json:"school_is_active"           form:"school_is_active"`
	SchoolVerificationStatus *string `json:"school_verification_status" form:"school_verification_status"`
	SchoolVerificationNotes  *string `json:"school_verification_notes"  form:"school_verification_notes"`

	SchoolContactPersonName  *string `json:"school_contact_person_name"  form:"school_contact_person_name"`
	SchoolContactPersonPhone *string `json:"school_contact_person_phone" form:"school_contact_person_phone"`

	SchoolIsIslamicSchool *bool   `json:"school_is_islamic_school" form:"school_is_islamic_school"`
	SchoolTenantProfile   *string `json:"school_tenant_profile"    form:"school_tenant_profile"`

	SchoolLevels *[]string `json:"school_levels" form:"school_levels"`

	// Media current (PATCH via JSON)
	SchoolIconURL             *string `json:"school_icon_url"              form:"school_icon_url"`
	SchoolIconObjectKey       *string `json:"school_icon_object_key"       form:"school_icon_object_key"`
	SchoolLogoURL             *string `json:"school_logo_url"              form:"school_logo_url"`
	SchoolLogoObjectKey       *string `json:"school_logo_object_key"       form:"school_logo_object_key"`
	SchoolBackgroundURL       *string `json:"school_background_url"        form:"school_background_url"`
	SchoolBackgroundObjectKey *string `json:"school_background_object_key" form:"school_background_object_key"`

	// Rotate/set ulang kode undangan guru (plaintext)
	SchoolTeacherCodePlain *string `json:"school_teacher_code_plain" form:"school_teacher_code_plain"`

	// Clear → NULL/empty eksplisit pada kolom tertentu
	Clear []string `json:"__clear,omitempty" form:"__clear"`
}

/* ===================== RESPONSE ===================== */

type SchoolResp struct {
	SchoolID            string     `json:"school_id"`
	SchoolYayasanID     *uuid.UUID `json:"school_yayasan_id,omitempty"`
	SchoolCurrentPlanID *uuid.UUID `json:"school_current_plan_id,omitempty"`

	SchoolName     string `json:"school_name"`
	SchoolBioShort string `json:"school_bio_short"`
	SchoolDomain   string `json:"school_domain"`
	SchoolSlug     string `json:"school_slug"`
	SchoolLocation string `json:"school_location"`
	SchoolCity     string `json:"school_city"`

	// 🔢 nomor running sekolah (global, dari DB)
	SchoolNumber int64 `json:"school_number"`

	SchoolIsActive           bool       `json:"school_is_active"`
	SchoolIsVerified         bool       `json:"school_is_verified"`
	SchoolVerificationStatus string     `json:"school_verification_status"`
	SchoolVerifiedAt         *time.Time `json:"school_verified_at,omitempty"`
	SchoolVerificationNotes  string     `json:"school_verification_notes"`

	SchoolContactPersonName  string `json:"school_contact_person_name"`
	SchoolContactPersonPhone string `json:"school_contact_person_phone"`

	SchoolIsIslamicSchool bool   `json:"school_is_islamic_school"`
	SchoolTenantProfile   string `json:"school_tenant_profile"`

	SchoolLevels []string `json:"school_levels"`

	// Teacher code (tidak expose hash)
	SchoolHasTeacherCode   bool       `json:"school_has_teacher_code"`
	SchoolTeacherCodeSetAt *time.Time `json:"school_teacher_code_set_at,omitempty"`

	// ICON
	SchoolIconURL                string     `json:"school_icon_url"`
	SchoolIconObjectKey          string     `json:"school_icon_object_key"`
	SchoolIconURLOld             string     `json:"school_icon_url_old"`
	SchoolIconObjectKeyOld       string     `json:"school_icon_object_key_old"`
	SchoolIconDeletePendingUntil *time.Time `json:"school_icon_delete_pending_until,omitempty"`

	// LOGO
	SchoolLogoURL                string     `json:"school_logo_url"`
	SchoolLogoObjectKey          string     `json:"school_logo_object_key"`
	SchoolLogoURLOld             string     `json:"school_logo_url_old"`
	SchoolLogoObjectKeyOld       string     `json:"school_logo_object_key_old"`
	SchoolLogoDeletePendingUntil *time.Time `json:"school_logo_delete_pending_until,omitempty"`

	// BACKGROUND
	SchoolBackgroundURL                string     `json:"school_background_url"`
	SchoolBackgroundObjectKey          string     `json:"school_background_object_key"`
	SchoolBackgroundURLOld             string     `json:"school_background_url_old"`
	SchoolBackgroundObjectKeyOld       string     `json:"school_background_object_key_old"`
	SchoolBackgroundDeletePendingUntil *time.Time `json:"school_background_delete_pending_until,omitempty"`

	SchoolCreatedAt      time.Time  `json:"school_created_at"`
	SchoolUpdatedAt      time.Time  `json:"school_updated_at"`
	SchoolLastActivityAt *time.Time `json:"school_last_activity_at,omitempty"`
}

/* ===================== CONVERTERS ===================== */
func FromModel(m *model.SchoolModel) SchoolResp {
	levels := levelsFromJSON(m.SchoolLevels)

	return SchoolResp{
		SchoolID:            m.SchoolID.String(),
		SchoolYayasanID:     m.SchoolYayasanID,
		SchoolCurrentPlanID: m.SchoolCurrentPlanID,

		SchoolName:     m.SchoolName,
		SchoolBioShort: valOrEmpty(m.SchoolBioShort),
		SchoolDomain:   valOrEmpty(m.SchoolDomain),
		SchoolSlug:     m.SchoolSlug,
		SchoolLocation: valOrEmpty(m.SchoolLocation),
		SchoolCity:     valOrEmpty(m.SchoolCity),

		// 🔢 nomor sekolah dari DB
		SchoolNumber: m.SchoolNumber,

		SchoolIsActive:           m.SchoolIsActive,
		SchoolIsVerified:         m.SchoolIsVerified,
		SchoolVerificationStatus: string(m.SchoolVerificationStatus),
		SchoolVerifiedAt:         m.SchoolVerifiedAt,
		SchoolVerificationNotes:  valOrEmpty(m.SchoolVerificationNotes),

		SchoolContactPersonName:  valOrEmpty(m.SchoolContactPersonName),
		SchoolContactPersonPhone: valOrEmpty(m.SchoolContactPersonPhone),

		SchoolIsIslamicSchool: m.SchoolIsIslamicSchool,
		SchoolTenantProfile:   string(m.SchoolTenantProfile),
		SchoolLevels:          levels,

		SchoolHasTeacherCode:   len(m.SchoolTeacherCodeHash) > 0,
		SchoolTeacherCodeSetAt: m.SchoolTeacherCodeSetAt,

		// ICON
		SchoolIconURL:                valOrEmpty(m.SchoolIconURL),
		SchoolIconObjectKey:          valOrEmpty(m.SchoolIconObjectKey),
		SchoolIconURLOld:             valOrEmpty(m.SchoolIconURLOld),
		SchoolIconObjectKeyOld:       valOrEmpty(m.SchoolIconObjectKeyOld),
		SchoolIconDeletePendingUntil: m.SchoolIconDeletePendingUntil,

		// LOGO
		SchoolLogoURL:                valOrEmpty(m.SchoolLogoURL),
		SchoolLogoObjectKey:          valOrEmpty(m.SchoolLogoObjectKey),
		SchoolLogoURLOld:             valOrEmpty(m.SchoolLogoURLOld),
		SchoolLogoObjectKeyOld:       valOrEmpty(m.SchoolLogoObjectKeyOld),
		SchoolLogoDeletePendingUntil: m.SchoolLogoDeletePendingUntil,

		// BACKGROUND
		SchoolBackgroundURL:                valOrEmpty(m.SchoolBackgroundURL),
		SchoolBackgroundObjectKey:          valOrEmpty(m.SchoolBackgroundObjectKey),
		SchoolBackgroundURLOld:             valOrEmpty(m.SchoolBackgroundURLOld),
		SchoolBackgroundObjectKeyOld:       valOrEmpty(m.SchoolBackgroundObjectKeyOld),
		SchoolBackgroundDeletePendingUntil: m.SchoolBackgroundDeletePendingUntil,

		SchoolCreatedAt:      m.SchoolCreatedAt,
		SchoolUpdatedAt:      m.SchoolUpdatedAt,
		SchoolLastActivityAt: m.SchoolLastActivityAt,
	}
}

func ToModel(in *SchoolCreateReq, id uuid.UUID) *model.SchoolModel {
	out := &model.SchoolModel{
		SchoolID:            id,
		SchoolYayasanID:     in.SchoolYayasanID,
		SchoolCurrentPlanID: in.SchoolCurrentPlanID,

		SchoolName:     strings.TrimSpace(in.SchoolName),
		SchoolBioShort: optStrPtr(in.SchoolBioShort),
		SchoolLocation: optStrPtr(in.SchoolLocation),
		SchoolCity:     optStrPtr(in.SchoolCity),

		SchoolDomain: normDomainPtr(in.SchoolDomain),
		SchoolSlug:   strings.TrimSpace(in.SchoolSlug),

		SchoolIsActive:           in.SchoolIsActive,
		SchoolVerificationStatus: model.VerificationStatus(normVerification(in.SchoolVerificationStatus)),
		SchoolVerificationNotes:  optStrPtr(in.SchoolVerificationNotes),

		SchoolContactPersonName:  optStrPtr(in.SchoolContactPersonName),
		SchoolContactPersonPhone: optStrPtr(in.SchoolContactPersonPhone),

		SchoolIsIslamicSchool: in.SchoolIsIslamicSchool,
		SchoolTenantProfile:   model.TenantProfile(normTenantProfile(in.SchoolTenantProfile)),

		SchoolIconURL:             optStrPtr(in.SchoolIconURL),
		SchoolIconObjectKey:       optStrPtr(in.SchoolIconObjectKey),
		SchoolLogoURL:             optStrPtr(in.SchoolLogoURL),
		SchoolLogoObjectKey:       optStrPtr(in.SchoolLogoObjectKey),
		SchoolBackgroundURL:       optStrPtr(in.SchoolBackgroundURL),
		SchoolBackgroundObjectKey: optStrPtr(in.SchoolBackgroundObjectKey),
	}

	// Levels
	if len(in.SchoolLevels) > 0 {
		out.SchoolLevels = levelsToJSON(in.SchoolLevels)
	}

	// Teacher invite code (plaintext) → hash akan diisi di service (bukan di DTO)
	// Di sini cukup tandai SetAt agar service bisa mengisi kalau diperlukan.
	if strings.TrimSpace(in.SchoolTeacherCodePlain) != "" {
		now := time.Now()
		out.SchoolTeacherCodeSetAt = &now
	}

	return out
}

func ApplyUpdate(m *model.SchoolModel, u *SchoolUpdateReq) {
	// Relasi
	if u.SchoolYayasanID != nil {
		m.SchoolYayasanID = u.SchoolYayasanID
	}
	if u.SchoolCurrentPlanID != nil {
		m.SchoolCurrentPlanID = u.SchoolCurrentPlanID
	}

	// Identitas & lokasi
	if u.SchoolName != nil {
		m.SchoolName = strings.TrimSpace(*u.SchoolName)
	}
	if u.SchoolBioShort != nil {
		m.SchoolBioShort = optStrPtr(strings.TrimSpace(*u.SchoolBioShort))
	}
	if u.SchoolLocation != nil {
		m.SchoolLocation = optStrPtr(strings.TrimSpace(*u.SchoolLocation))
	}
	if u.SchoolCity != nil {
		m.SchoolCity = optStrPtr(strings.TrimSpace(*u.SchoolCity))
	}

	// Domain & slug
	if u.SchoolDomain != nil {
		m.SchoolDomain = normDomainPtr(*u.SchoolDomain)
	}
	if u.SchoolSlug != nil {
		m.SchoolSlug = strings.TrimSpace(*u.SchoolSlug)
	}

	// Aktivasi & verifikasi
	if u.SchoolIsActive != nil {
		m.SchoolIsActive = *u.SchoolIsActive
	}
	if u.SchoolVerificationStatus != nil {
		m.SchoolVerificationStatus = model.VerificationStatus(normVerification(*u.SchoolVerificationStatus))
	}
	if u.SchoolVerificationNotes != nil {
		m.SchoolVerificationNotes = optStrPtr(strings.TrimSpace(*u.SchoolVerificationNotes))
	}

	// Kontak
	if u.SchoolContactPersonName != nil {
		m.SchoolContactPersonName = optStrPtr(strings.TrimSpace(*u.SchoolContactPersonName))
	}
	if u.SchoolContactPersonPhone != nil {
		m.SchoolContactPersonPhone = optStrPtr(strings.TrimSpace(*u.SchoolContactPersonPhone))
	}

	// Flag & profil
	if u.SchoolIsIslamicSchool != nil {
		m.SchoolIsIslamicSchool = *u.SchoolIsIslamicSchool
	}
	if u.SchoolTenantProfile != nil {
		m.SchoolTenantProfile = model.TenantProfile(normTenantProfile(*u.SchoolTenantProfile))
	}

	// Levels
	if u.SchoolLevels != nil {
		m.SchoolLevels = levelsToJSON(*u.SchoolLevels)
	}

	// Media current (PATCH)
	if u.SchoolIconURL != nil {
		m.SchoolIconURL = optStrPtr(strings.TrimSpace(*u.SchoolIconURL))
	}
	if u.SchoolIconObjectKey != nil {
		m.SchoolIconObjectKey = optStrPtr(strings.TrimSpace(*u.SchoolIconObjectKey))
	}
	if u.SchoolLogoURL != nil {
		m.SchoolLogoURL = optStrPtr(strings.TrimSpace(*u.SchoolLogoURL))
	}
	if u.SchoolLogoObjectKey != nil {
		m.SchoolLogoObjectKey = optStrPtr(strings.TrimSpace(*u.SchoolLogoObjectKey))
	}
	if u.SchoolBackgroundURL != nil {
		m.SchoolBackgroundURL = optStrPtr(strings.TrimSpace(*u.SchoolBackgroundURL))
	}
	if u.SchoolBackgroundObjectKey != nil {
		m.SchoolBackgroundObjectKey = optStrPtr(strings.TrimSpace(*u.SchoolBackgroundObjectKey))
	}

	// Teacher code: set/rotate via plaintext (hashing dikerjakan di service)
	if u.SchoolTeacherCodePlain != nil && strings.TrimSpace(*u.SchoolTeacherCodePlain) != "" {
		now := time.Now()
		m.SchoolTeacherCodeSetAt = &now
		// m.SchoolTeacherCodeHash akan diisi oleh service setelah hashing
	}

	// Clear → NULL/empty eksplisit
	for _, col := range u.Clear {
		switch strings.TrimSpace(strings.ToLower(col)) {
		case "school_domain":
			m.SchoolDomain = nil
		case "school_bio_short":
			m.SchoolBioShort = nil
		case "school_location":
			m.SchoolLocation = nil
		case "school_city":
			m.SchoolCity = nil
		case "school_contact_person_name":
			m.SchoolContactPersonName = nil
		case "school_contact_person_phone":
			m.SchoolContactPersonPhone = nil
		case "school_levels":
			m.SchoolLevels = nil // datatypes.JSON(nil) ⇒ NULL
		case "school_icon_url":
			m.SchoolIconURL = nil
		case "school_icon_object_key":
			m.SchoolIconObjectKey = nil
		case "school_logo_url":
			m.SchoolLogoURL = nil
		case "school_logo_object_key":
			m.SchoolLogoObjectKey = nil
		case "school_background_url":
			m.SchoolBackgroundURL = nil
		case "school_background_object_key":
			m.SchoolBackgroundObjectKey = nil
		case "school_teacher_code":
			m.SchoolTeacherCodeHash = nil
			m.SchoolTeacherCodeSetAt = nil
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
