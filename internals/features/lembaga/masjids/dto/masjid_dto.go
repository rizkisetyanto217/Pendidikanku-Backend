package dto

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/* ===================== ENUMS ===================== */

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

/* ===================== MODEL: Masjid ===================== */

type Masjid struct {
	MasjidID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:masjid_id" json:"masjid_id"`

	MasjidYayasanID     *uuid.UUID `gorm:"type:uuid;column:masjid_yayasan_id" json:"masjid_yayasan_id,omitempty"`
	MasjidCurrentPlanID *uuid.UUID `gorm:"type:uuid;column:masjid_current_plan_id" json:"masjid_current_plan_id,omitempty"`

	MasjidName     string  `gorm:"type:varchar(100);not null;column:masjid_name" json:"masjid_name"`
	MasjidBioShort *string `gorm:"type:text;column:masjid_bio_short" json:"masjid_bio_short,omitempty"`
	MasjidLocation *string `gorm:"type:text;column:masjid_location" json:"masjid_location,omitempty"`
	MasjidCity     *string `gorm:"type:varchar(80);column:masjid_city" json:"masjid_city,omitempty"`

	MasjidDomain *string `gorm:"type:varchar(50);column:masjid_domain" json:"masjid_domain,omitempty"`
	MasjidSlug   string  `gorm:"type:varchar(100);uniqueIndex;not null;column:masjid_slug" json:"masjid_slug"`

	MasjidIsActive           bool               `gorm:"type:boolean;not null;default:true;column:masjid_is_active" json:"masjid_is_active"`
	MasjidIsVerified         bool               `gorm:"type:boolean;not null;default:false;column:masjid_is_verified" json:"masjid_is_verified"`
	MasjidVerificationStatus VerificationStatus `gorm:"type:verification_status_enum;not null;default:'pending';column:masjid_verification_status" json:"masjid_verification_status"`
	MasjidVerifiedAt         *time.Time         `gorm:"type:timestamptz;column:masjid_verified_at" json:"masjid_verified_at,omitempty"`
	MasjidVerificationNotes  *string            `gorm:"type:text;column:masjid_verification_notes" json:"masjid_verification_notes,omitempty"`

	MasjidContactPersonName  *string `gorm:"type:varchar(100);column:masjid_contact_person_name" json:"masjid_contact_person_name,omitempty"`
	MasjidContactPersonPhone *string `gorm:"type:varchar(30);column:masjid_contact_person_phone" json:"masjid_contact_person_phone,omitempty"`

	MasjidIsIslamicSchool bool          `gorm:"type:boolean;not null;default:false;column:masjid_is_islamic_school" json:"masjid_is_islamic_school"`
	MasjidTenantProfile   TenantProfile `gorm:"type:tenant_profile_enum;not null;default:'teacher_solo';column:masjid_tenant_profile" json:"masjid_tenant_profile"`

	MasjidLevels *datatypes.JSON `gorm:"type:jsonb;column:masjid_levels" json:"masjid_levels,omitempty"`

	// ICON (2-slot)
	MasjidIconURL                *string    `gorm:"type:text;column:masjid_icon_url" json:"masjid_icon_url,omitempty"`
	MasjidIconObjectKey          *string    `gorm:"type:text;column:masjid_icon_object_key" json:"masjid_icon_object_key,omitempty"`
	MasjidIconURLOld             *string    `gorm:"type:text;column:masjid_icon_url_old" json:"masjid_icon_url_old,omitempty"`
	MasjidIconObjectKeyOld       *string    `gorm:"type:text;column:masjid_icon_object_key_old" json:"masjid_icon_object_key_old,omitempty"`
	MasjidIconDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:masjid_icon_delete_pending_until" json:"masjid_icon_delete_pending_until,omitempty"`

	// LOGO (2-slot)
	MasjidLogoURL                *string    `gorm:"type:text;column:masjid_logo_url" json:"masjid_logo_url,omitempty"`
	MasjidLogoObjectKey          *string    `gorm:"type:text;column:masjid_logo_object_key" json:"masjid_logo_object_key,omitempty"`
	MasjidLogoURLOld             *string    `gorm:"type:text;column:masjid_logo_url_old" json:"masjid_logo_url_old,omitempty"`
	MasjidLogoObjectKeyOld       *string    `gorm:"type:text;column:masjid_logo_object_key_old" json:"masjid_logo_object_key_old,omitempty"`
	MasjidLogoDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:masjid_logo_delete_pending_until" json:"masjid_logo_delete_pending_until,omitempty"`

	// BACKGROUND (2-slot)
	MasjidBackgroundURL                *string    `gorm:"type:text;column:masjid_background_url" json:"masjid_background_url,omitempty"`
	MasjidBackgroundObjectKey          *string    `gorm:"type:text;column:masjid_background_object_key" json:"masjid_background_object_key,omitempty"`
	MasjidBackgroundURLOld             *string    `gorm:"type:text;column:masjid_background_url_old" json:"masjid_background_url_old,omitempty"`
	MasjidBackgroundObjectKeyOld       *string    `gorm:"type:text;column:masjid_background_object_key_old" json:"masjid_background_object_key_old,omitempty"`
	MasjidBackgroundDeletePendingUntil *time.Time `gorm:"type:timestamptz;column:masjid_background_delete_pending_until" json:"masjid_background_delete_pending_until,omitempty"`

	MasjidCreatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now();autoCreateTime;column:masjid_created_at" json:"masjid_created_at"`
	MasjidUpdatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now();autoUpdateTime;column:masjid_updated_at" json:"masjid_updated_at"`
	MasjidLastActivityAt *time.Time     `gorm:"type:timestamptz;column:masjid_last_activity_at" json:"masjid_last_activity_at,omitempty"`
	MasjidDeletedAt      gorm.DeletedAt `gorm:"index;column:masjid_deleted_at" json:"masjid_deleted_at,omitempty"`
}

func (Masjid) TableName() string { return "masjids" }

// JSONB helpers
func (m *Masjid) GetLevels() []string {
	if m == nil || m.MasjidLevels == nil || len(*m.MasjidLevels) == 0 {
		return []string{}
	}
	var out []string
	_ = json.Unmarshal(*m.MasjidLevels, &out)
	return out
}
func (m *Masjid) SetLevels(levels []string) {
	if len(levels) == 0 {
		m.MasjidLevels = nil
		return
	}
	if b, err := json.Marshal(levels); err == nil {
		js := datatypes.JSON(b)
		m.MasjidLevels = &js
	}
}

/* ===================== MODEL: MasjidProfile (pakai DTO) ===================== */

type MasjidProfile struct {
	MasjidProfileMasjidID               uuid.UUID `gorm:"column:masjid_profile_masjid_id;primaryKey"`
	MasjidProfileDescription            *string   `gorm:"column:masjid_profile_description"`
	MasjidProfileFoundedYear            *int      `gorm:"column:masjid_profile_founded_year"`
	MasjidProfileAddress                *string   `gorm:"column:masjid_profile_address"`
	MasjidProfileContactPhone           *string   `gorm:"column:masjid_profile_contact_phone"`
	MasjidProfileContactEmail           *string   `gorm:"column:masjid_profile_contact_email"`
	MasjidProfileGoogleMapsURL          *string   `gorm:"column:masjid_profile_google_maps_url"`
	MasjidProfileInstagramURL           *string   `gorm:"column:masjid_profile_instagram_url"`
	MasjidProfileWhatsappURL            *string   `gorm:"column:masjid_profile_whatsapp_url"`
	MasjidProfileYoutubeURL             *string   `gorm:"column:masjid_profile_youtube_url"`
	MasjidProfileFacebookURL            *string   `gorm:"column:masjid_profile_facebook_url"`
	MasjidProfileTiktokURL              *string   `gorm:"column:masjid_profile_tiktok_url"`
	MasjidProfileWhatsappGroupIkhwanURL *string   `gorm:"column:masjid_profile_whatsapp_group_ikhwan_url"`
	MasjidProfileWhatsappGroupAkhwatURL *string   `gorm:"column:masjid_profile_whatsapp_group_akhwat_url"`
	MasjidProfileWebsiteURL             *string   `gorm:"column:masjid_profile_website_url"`

	MasjidProfileSchoolNPSN            *string    `gorm:"column:masjid_profile_school_npsn"`
	MasjidProfileSchoolNSS             *string    `gorm:"column:masjid_profile_school_nss"`
	MasjidProfileSchoolAccreditation   *string    `gorm:"column:masjid_profile_school_accreditation"`
	MasjidProfileSchoolPrincipalUserID *uuid.UUID `gorm:"column:masjid_profile_school_principal_user_id"`
	MasjidProfileSchoolPhone           *string    `gorm:"column:masjid_profile_school_phone"`
	MasjidProfileSchoolEmail           *string    `gorm:"column:masjid_profile_school_email"`
	MasjidProfileSchoolAddress         *string    `gorm:"column:masjid_profile_school_address"`
	MasjidProfileSchoolStudentCapacity *int       `gorm:"column:masjid_profile_school_student_capacity"`
	MasjidProfileSchoolIsBoarding      bool       `gorm:"column:masjid_profile_school_is_boarding"`
}

func (MasjidProfile) TableName() string { return "masjid_profiles" }

// Builder dari payload → model
func ToModelMasjidProfile(p *MasjidProfilePayload, masjidID uuid.UUID) *MasjidProfile {
	if p == nil {
		return nil
	}
	ptr := func(s string) *string {
		ss := strings.TrimSpace(s)
		if ss == "" {
			return nil
		}
		return &ss
	}
	out := &MasjidProfile{
		MasjidProfileMasjidID:               masjidID,
		MasjidProfileDescription:            ptr(p.Description),
		MasjidProfileFoundedYear:            p.FoundedYear,
		MasjidProfileAddress:                ptr(p.Address),
		MasjidProfileContactPhone:           ptr(p.ContactPhone),
		MasjidProfileContactEmail:           ptr(p.ContactEmail),
		MasjidProfileGoogleMapsURL:          ptr(p.GoogleMapsURL),
		MasjidProfileInstagramURL:           ptr(p.InstagramURL),
		MasjidProfileWhatsappURL:            ptr(p.WhatsappURL),
		MasjidProfileYoutubeURL:             ptr(p.YoutubeURL),
		MasjidProfileFacebookURL:            ptr(p.FacebookURL),
		MasjidProfileTiktokURL:              ptr(p.TiktokURL),
		MasjidProfileWhatsappGroupIkhwanURL: ptr(p.WhatsappGroupIkhwanURL),
		MasjidProfileWhatsappGroupAkhwatURL: ptr(p.WhatsappGroupAkhwatURL),
		MasjidProfileWebsiteURL:             ptr(p.WebsiteURL),

		MasjidProfileSchoolNPSN:            ptr(p.SchoolNPSN),
		MasjidProfileSchoolNSS:             ptr(p.SchoolNSS),
		MasjidProfileSchoolAccreditation:   ptr(p.SchoolAccreditation),
		MasjidProfileSchoolPrincipalUserID: p.SchoolPrincipalUserID,
		MasjidProfileSchoolPhone:           ptr(p.SchoolPhone),
		MasjidProfileSchoolEmail:           ptr(p.SchoolEmail),
		MasjidProfileSchoolAddress:         ptr(p.SchoolAddress),
		MasjidProfileSchoolStudentCapacity: p.SchoolStudentCapacity,
	}
	if p.SchoolIsBoarding != nil {
		out.MasjidProfileSchoolIsBoarding = *p.SchoolIsBoarding
	}
	return out
}

/* ===================== DTO: Requests/Responses ===================== */

type MasjidRequest struct {
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

	// Media current (opsional seed awal)
	MasjidIconURL             string `json:"masjid_icon_url"`
	MasjidIconObjectKey       string `json:"masjid_icon_object_key"`
	MasjidLogoURL             string `json:"masjid_logo_url"`
	MasjidLogoObjectKey       string `json:"masjid_logo_object_key"`
	MasjidBackgroundURL       string `json:"masjid_background_url"`
	MasjidBackgroundObjectKey string `json:"masjid_background_object_key"`
}

type MasjidProfilePayload struct {
	Description string `json:"description"`
	FoundedYear *int   `json:"founded_year"`

	Address      string `json:"address"`
	ContactPhone string `json:"contact_phone"`
	ContactEmail string `json:"contact_email"`

	GoogleMapsURL          string `json:"google_maps_url"`
	InstagramURL           string `json:"instagram_url"`
	WhatsappURL            string `json:"whatsapp_url"`
	YoutubeURL             string `json:"youtube_url"`
	FacebookURL            string `json:"facebook_url"`
	TiktokURL              string `json:"tiktok_url"`
	WhatsappGroupIkhwanURL string `json:"whatsapp_group_ikhwan_url"`
	WhatsappGroupAkhwatURL string `json:"whatsapp_group_akhwat_url"`
	WebsiteURL             string `json:"website_url"`

	SchoolNPSN            string     `json:"school_npsn"`
	SchoolNSS             string     `json:"school_nss"`
	SchoolAccreditation   string     `json:"school_accreditation"`
	SchoolPrincipalUserID *uuid.UUID `json:"school_principal_user_id"`
	SchoolPhone           string     `json:"school_phone"`
	SchoolEmail           string     `json:"school_email"`
	SchoolAddress         string     `json:"school_address"`
	SchoolStudentCapacity *int       `json:"school_student_capacity"`
	SchoolIsBoarding      *bool      `json:"school_is_boarding"`
}

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

	// ICON (current + shadow)
	MasjidIconURL                string     `json:"masjid_icon_url"`
	MasjidIconObjectKey          string     `json:"masjid_icon_object_key"`
	MasjidIconURLOld             string     `json:"masjid_icon_url_old"`
	MasjidIconObjectKeyOld       string     `json:"masjid_icon_object_key_old"`
	MasjidIconDeletePendingUntil *time.Time `json:"masjid_icon_delete_pending_until,omitempty"`

	// LOGO (current + shadow)
	MasjidLogoURL                string     `json:"masjid_logo_url"`
	MasjidLogoObjectKey          string     `json:"masjid_logo_object_key"`
	MasjidLogoURLOld             string     `json:"masjid_logo_url_old"`
	MasjidLogoObjectKeyOld       string     `json:"masjid_logo_object_key_old"`
	MasjidLogoDeletePendingUntil *time.Time `json:"masjid_logo_delete_pending_until,omitempty"`

	// BACKGROUND (current + shadow)
	MasjidBackgroundURL                string     `json:"masjid_background_url"`
	MasjidBackgroundObjectKey          string     `json:"masjid_background_object_key"`
	MasjidBackgroundURLOld             string     `json:"masjid_background_url_old"`
	MasjidBackgroundObjectKeyOld       string     `json:"masjid_background_object_key_old"`
	MasjidBackgroundDeletePendingUntil *time.Time `json:"masjid_background_delete_pending_until,omitempty"`

	MasjidCreatedAt      time.Time  `json:"masjid_created_at"`
	MasjidUpdatedAt      time.Time  `json:"masjid_updated_at"`
	MasjidLastActivityAt *time.Time `json:"masjid_last_activity_at,omitempty"`
}

type MasjidUpdateRequest struct {
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

	// Media current (PATCH by JSON)
	MasjidIconURL             *string `json:"masjid_icon_url"              form:"masjid_icon_url"`
	MasjidIconObjectKey       *string `json:"masjid_icon_object_key"       form:"masjid_icon_object_key"`
	MasjidLogoURL             *string `json:"masjid_logo_url"              form:"masjid_logo_url"`
	MasjidLogoObjectKey       *string `json:"masjid_logo_object_key"       form:"masjid_logo_object_key"`
	MasjidBackgroundURL       *string `json:"masjid_background_url"        form:"masjid_background_url"`
	MasjidBackgroundObjectKey *string `json:"masjid_background_object_key" form:"masjid_background_object_key"`

	Clear []string `json:"__clear,omitempty" form:"__clear"`
}

/* ===================== Converters ===================== */

func FromModelMasjid(m *Masjid) MasjidResponse {
	levels := m.GetLevels()
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

func ToModelMasjid(in *MasjidRequest, id uuid.UUID) *Masjid {
	m := &Masjid{
		MasjidID:            id,
		MasjidYayasanID:     in.MasjidYayasanID,
		MasjidCurrentPlanID: in.MasjidCurrentPlanID,

		MasjidName:     strings.TrimSpace(in.MasjidName),
		MasjidBioShort: normalizeOptionalStringToPtr(in.MasjidBioShort),
		MasjidLocation: normalizeOptionalStringToPtr(in.MasjidLocation),
		MasjidCity:     normalizeOptionalStringToPtr(in.MasjidCity),

		MasjidDomain: normalizeDomainToPtr(in.MasjidDomain),
		MasjidSlug:   strings.TrimSpace(in.MasjidSlug),

		MasjidIsActive:           in.MasjidIsActive,
		MasjidVerificationStatus: VerificationStatus(normalizeVerification(in.MasjidVerificationStatus)),
		MasjidVerificationNotes:  normalizeOptionalStringToPtr(in.MasjidVerificationNotes),

		MasjidContactPersonName:  normalizeOptionalStringToPtr(in.MasjidContactPersonName),
		MasjidContactPersonPhone: normalizeOptionalStringToPtr(in.MasjidContactPersonPhone),

		MasjidIsIslamicSchool: in.MasjidIsIslamicSchool,
		MasjidTenantProfile:   TenantProfile(normalizeTenantProfile(in.MasjidTenantProfile)),

		// Media current (opsional)
		MasjidIconURL:             normalizeOptionalStringToPtr(in.MasjidIconURL),
		MasjidIconObjectKey:       normalizeOptionalStringToPtr(in.MasjidIconObjectKey),
		MasjidLogoURL:             normalizeOptionalStringToPtr(in.MasjidLogoURL),
		MasjidLogoObjectKey:       normalizeOptionalStringToPtr(in.MasjidLogoObjectKey),
		MasjidBackgroundURL:       normalizeOptionalStringToPtr(in.MasjidBackgroundURL),
		MasjidBackgroundObjectKey: normalizeOptionalStringToPtr(in.MasjidBackgroundObjectKey),
	}

	// Levels
	if len(in.MasjidLevels) > 0 {
		if b, err := json.Marshal(in.MasjidLevels); err == nil {
			val := datatypes.JSON(b)
			m.MasjidLevels = &val
		}
	}
	return m
}

func ApplyMasjidUpdate(m *Masjid, u *MasjidUpdateRequest) {
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
		m.MasjidVerificationStatus = VerificationStatus(normalizeVerification(*u.MasjidVerificationStatus))
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

	// Flag & profil
	if u.MasjidIsIslamicSchool != nil {
		m.MasjidIsIslamicSchool = *u.MasjidIsIslamicSchool
	}
	if u.MasjidTenantProfile != nil {
		m.MasjidTenantProfile = TenantProfile(normalizeTenantProfile(*u.MasjidTenantProfile))
	}

	// Levels
	if u.MasjidLevels != nil {
		if b, err := json.Marshal(*u.MasjidLevels); err == nil {
			val := datatypes.JSON(b)
			m.MasjidLevels = &val
		}
	}

	// Media current (handled here untuk PATCH JSON)
	if u.MasjidIconURL != nil {
		m.MasjidIconURL = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidIconURL))
	}
	if u.MasjidIconObjectKey != nil {
		m.MasjidIconObjectKey = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidIconObjectKey))
	}
	if u.MasjidLogoURL != nil {
		m.MasjidLogoURL = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidLogoURL))
	}
	if u.MasjidLogoObjectKey != nil {
		m.MasjidLogoObjectKey = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidLogoObjectKey))
	}
	if u.MasjidBackgroundURL != nil {
		m.MasjidBackgroundURL = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidBackgroundURL))
	}
	if u.MasjidBackgroundObjectKey != nil {
		m.MasjidBackgroundObjectKey = normalizeOptionalStringToPtr(strings.TrimSpace(*u.MasjidBackgroundObjectKey))
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
		}
	}
}

/* ===================== Helpers ===================== */

func normalizeOptionalStringToPtr(s string) *string {
	trim := strings.TrimSpace(s)
	if trim == "" {
		return nil
	}
	return &trim
}
func normalizeDomainToPtr(s string) *string {
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
