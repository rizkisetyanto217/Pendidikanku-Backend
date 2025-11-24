// internals/features/lembaga/teachers/dto/school_teacher_dto.go
package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	yModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* ========================
   📦 Item JSON (DTO view)
   ======================== */

type DTOTeacherSectionItem struct {
	ClassSectionID             uuid.UUID `json:"class_section_id"`
	Role                       string    `json:"role"` // "homeroom" | "teacher" | "assistant"
	IsActive                   bool      `json:"is_active"`
	From                       *string   `json:"from,omitempty"` // "YYYY-MM-DD"
	To                         *string   `json:"to,omitempty"`   // "YYYY-MM-DD"
	ClassSectionName           *string   `json:"class_section_name,omitempty"`
	ClassSectionSlug           *string   `json:"class_section_slug,omitempty"`
	ClassSectionImageURL       *string   `json:"class_section_image_url,omitempty"`
	ClassSectionImageObjectKey *string   `json:"class_section_image_object_key,omitempty"`
}

type DTOTeacherCSSTItem struct {
	CSSTID           uuid.UUID  `json:"csst_id"`
	IsActive         bool       `json:"is_active"`
	From             *string    `json:"from,omitempty"`
	To               *string    `json:"to,omitempty"`
	SubjectName      *string    `json:"subject_name,omitempty"`
	SubjectSlug      *string    `json:"subject_slug,omitempty"`
	ClassSectionID   *uuid.UUID `json:"class_section_id,omitempty"`
	ClassSectionName *string    `json:"class_section_name,omitempty"`
	ClassSectionSlug *string    `json:"class_section_slug,omitempty"`
}

/* ========================
   📦 DTO Full Mirror (Entity Snapshot)
   ======================== */

type SchoolTeacher struct {
	SchoolTeacherID            string `json:"school_teacher_id"`
	SchoolTeacherSchoolID      string `json:"school_teacher_school_id"`
	SchoolTeacherUserTeacherID string `json:"school_teacher_user_teacher_id"`

	// Identitas/kepegawaian
	SchoolTeacherCode       *string `json:"school_teacher_code,omitempty"`
	SchoolTeacherSlug       *string `json:"school_teacher_slug,omitempty"`
	SchoolTeacherEmployment *string `json:"school_teacher_employment,omitempty"` // enum as string
	SchoolTeacherIsActive   bool    `json:"school_teacher_is_active"`

	// Periode kerja
	SchoolTeacherJoinedAt *time.Time `json:"school_teacher_joined_at,omitempty"`
	SchoolTeacherLeftAt   *time.Time `json:"school_teacher_left_at,omitempty"`

	// Verifikasi internal
	SchoolTeacherIsVerified bool       `json:"school_teacher_is_verified"`
	SchoolTeacherVerifiedAt *time.Time `json:"school_teacher_verified_at,omitempty"`

	// Visibilitas & catatan
	SchoolTeacherIsPublic bool    `json:"school_teacher_is_public"`
	SchoolTeacherNotes    *string `json:"school_teacher_notes,omitempty"`

	// Snapshot dari user_teachers
	SchoolTeacherUserTeacherNameSnapshot        *string `json:"school_teacher_user_teacher_name_snapshot,omitempty"`
	SchoolTeacherUserTeacherAvatarURLSnapshot   *string `json:"school_teacher_user_teacher_avatar_url_snapshot,omitempty"`
	SchoolTeacherUserTeacherWhatsappURLSnapshot *string `json:"school_teacher_user_teacher_whatsapp_url_snapshot,omitempty"`
	SchoolTeacherUserTeacherTitlePrefixSnapshot *string `json:"school_teacher_user_teacher_title_prefix_snapshot,omitempty"`
	SchoolTeacherUserTeacherTitleSuffixSnapshot *string `json:"school_teacher_user_teacher_title_suffix_snapshot,omitempty"`

	// MASJID SNAPSHOT (sinkron dgn model terbaru)
	SchoolTeacherSchoolNameSnapshot          *string `json:"school_teacher_school_name_snapshot,omitempty"`
	SchoolTeacherSchoolSlugSnapshot          *string `json:"school_teacher_school_slug_snapshot,omitempty"`
	SchoolTeacherSchoolLogoURLSnapshot       *string `json:"school_teacher_school_logo_url_snapshot,omitempty"`
	SchoolTeacherSchoolIconURLSnapshot       *string `json:"school_teacher_school_icon_url_snapshot,omitempty"`       // ⟵ BARU
	SchoolTeacherSchoolBackgroundURLSnapshot *string `json:"school_teacher_school_background_url_snapshot,omitempty"` // ⟵ BARU

	// JSONB: sections & csst
	SchoolTeacherSections []DTOTeacherSectionItem `json:"school_teacher_sections"`
	SchoolTeacherCSST     []DTOTeacherCSSTItem    `json:"school_teacher_csst"`

	// Audit
	SchoolTeacherCreatedAt time.Time  `json:"school_teacher_created_at"`
	SchoolTeacherUpdatedAt time.Time  `json:"school_teacher_updated_at"`
	SchoolTeacherDeletedAt *time.Time `json:"school_teacher_deleted_at,omitempty"`
}

/* ========================
   📥 Create Request DTO
   ======================== */

type CreateSchoolTeacherRequest struct {
	// Scope/relasi
	SchoolTeacherUserTeacherID *string `json:"school_teacher_user_teacher_id,omitempty" validate:"omitempty,uuid"`

	// Identitas/kepegawaian
	SchoolTeacherCode       *string `json:"school_teacher_code,omitempty"`
	SchoolTeacherSlug       *string `json:"school_teacher_slug,omitempty"`
	SchoolTeacherEmployment *string `json:"school_teacher_employment,omitempty"` // enum

	// Flags
	SchoolTeacherIsActive *bool `json:"school_teacher_is_active,omitempty"`
	SchoolTeacherIsPublic *bool `json:"school_teacher_is_public,omitempty"`

	// Waktu
	SchoolTeacherJoinedAt   *string `json:"school_teacher_joined_at,omitempty"`   // YYYY-MM-DD
	SchoolTeacherLeftAt     *string `json:"school_teacher_left_at,omitempty"`     // YYYY-MM-DD
	SchoolTeacherIsVerified *bool   `json:"school_teacher_is_verified,omitempty"` // default: false
	SchoolTeacherVerifiedAt *string `json:"school_teacher_verified_at,omitempty"` // RFC3339

	// Notes
	SchoolTeacherNotes *string `json:"school_teacher_notes,omitempty"`
}

/* ========================
   ✏️ Update Request DTO (pointer = tri-state)
   ======================== */

type UpdateSchoolTeacherRequest struct {
	SchoolTeacherUserTeacherID *string `json:"school_teacher_user_teacher_id,omitempty" validate:"omitempty,uuid"`

	SchoolTeacherCode       *string `json:"school_teacher_code,omitempty"`
	SchoolTeacherSlug       *string `json:"school_teacher_slug,omitempty"`
	SchoolTeacherEmployment *string `json:"school_teacher_employment,omitempty"`

	SchoolTeacherIsActive *bool `json:"school_teacher_is_active,omitempty"`
	SchoolTeacherIsPublic *bool `json:"school_teacher_is_public,omitempty"`

	SchoolTeacherJoinedAt   *string `json:"school_teacher_joined_at,omitempty"` // YYYY-MM-DD
	SchoolTeacherLeftAt     *string `json:"school_teacher_left_at,omitempty"`   // YYYY-MM-DD
	SchoolTeacherIsVerified *bool   `json:"school_teacher_is_verified,omitempty"`
	SchoolTeacherVerifiedAt *string `json:"school_teacher_verified_at,omitempty"` // RFC3339

	SchoolTeacherNotes *string `json:"school_teacher_notes,omitempty"`
}

/* ========================
   🔤 Normalizers
   ======================== */

func toNilIfEmpty(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func toLowerTrimmedOrNil(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.ToLower(strings.TrimSpace(*p))
	if s == "" {
		return nil
	}
	return &s
}

func (r *CreateSchoolTeacherRequest) Normalize() {
	r.SchoolTeacherUserTeacherID = toNilIfEmpty(r.SchoolTeacherUserTeacherID)

	r.SchoolTeacherCode = toNilIfEmpty(r.SchoolTeacherCode)
	r.SchoolTeacherSlug = toLowerTrimmedOrNil(r.SchoolTeacherSlug) // slug → lower
	r.SchoolTeacherEmployment = toLowerTrimmedOrNil(r.SchoolTeacherEmployment)

	r.SchoolTeacherJoinedAt = toNilIfEmpty(r.SchoolTeacherJoinedAt)
	r.SchoolTeacherLeftAt = toNilIfEmpty(r.SchoolTeacherLeftAt)
	r.SchoolTeacherVerifiedAt = toNilIfEmpty(r.SchoolTeacherVerifiedAt)

	r.SchoolTeacherNotes = toNilIfEmpty(r.SchoolTeacherNotes)
}

func (r *UpdateSchoolTeacherRequest) Normalize() {
	r.SchoolTeacherUserTeacherID = toNilIfEmpty(r.SchoolTeacherUserTeacherID)

	r.SchoolTeacherCode = toNilIfEmpty(r.SchoolTeacherCode)
	r.SchoolTeacherSlug = toLowerTrimmedOrNil(r.SchoolTeacherSlug)
	r.SchoolTeacherEmployment = toLowerTrimmedOrNil(r.SchoolTeacherEmployment)

	r.SchoolTeacherJoinedAt = toNilIfEmpty(r.SchoolTeacherJoinedAt)
	r.SchoolTeacherLeftAt = toNilIfEmpty(r.SchoolTeacherLeftAt)
	r.SchoolTeacherVerifiedAt = toNilIfEmpty(r.SchoolTeacherVerifiedAt)

	r.SchoolTeacherNotes = toNilIfEmpty(r.SchoolTeacherNotes)
}

/* ========================
   🔁 Converters (Model -> DTO)
   ======================== */

func NewSchoolTeacherResponse(m *yModel.SchoolTeacherModel) *SchoolTeacher {
	if m == nil {
		return nil
	}

	var emp *string
	if m.SchoolTeacherEmployment != nil {
		s := string(*m.SchoolTeacherEmployment)
		emp = &s
	}

	var delAt *time.Time
	if m.SchoolTeacherDeletedAt.Valid {
		delAt = &m.SchoolTeacherDeletedAt.Time
	}

	// Unmarshal JSONB (sections & csst)
	var sections []DTOTeacherSectionItem
	if len(m.SchoolTeacherSections) > 0 {
		_ = json.Unmarshal(m.SchoolTeacherSections, &sections)
	}
	if sections == nil {
		sections = []DTOTeacherSectionItem{}
	}

	var cssts []DTOTeacherCSSTItem
	if len(m.SchoolTeacherCSST) > 0 {
		_ = json.Unmarshal(m.SchoolTeacherCSST, &cssts)
	}
	if cssts == nil {
		cssts = []DTOTeacherCSSTItem{}
	}

	return &SchoolTeacher{
		SchoolTeacherID:            m.SchoolTeacherID.String(),
		SchoolTeacherSchoolID:      m.SchoolTeacherSchoolID.String(),
		SchoolTeacherUserTeacherID: m.SchoolTeacherUserTeacherID.String(),

		SchoolTeacherCode:       m.SchoolTeacherCode,
		SchoolTeacherSlug:       m.SchoolTeacherSlug,
		SchoolTeacherEmployment: emp,
		SchoolTeacherIsActive:   m.SchoolTeacherIsActive,

		SchoolTeacherJoinedAt: m.SchoolTeacherJoinedAt,
		SchoolTeacherLeftAt:   m.SchoolTeacherLeftAt,

		SchoolTeacherIsVerified: m.SchoolTeacherIsVerified,
		SchoolTeacherVerifiedAt: m.SchoolTeacherVerifiedAt,

		SchoolTeacherIsPublic: m.SchoolTeacherIsPublic,
		SchoolTeacherNotes:    m.SchoolTeacherNotes,

		SchoolTeacherUserTeacherNameSnapshot:        m.SchoolTeacherUserTeacherNameSnapshot,
		SchoolTeacherUserTeacherAvatarURLSnapshot:   m.SchoolTeacherUserTeacherAvatarURLSnapshot,
		SchoolTeacherUserTeacherWhatsappURLSnapshot: m.SchoolTeacherUserTeacherWhatsappURLSnapshot,
		SchoolTeacherUserTeacherTitlePrefixSnapshot: m.SchoolTeacherUserTeacherTitlePrefixSnapshot,
		SchoolTeacherUserTeacherTitleSuffixSnapshot: m.SchoolTeacherUserTeacherTitleSuffixSnapshot,

		// School snapshots (termasuk 2 field BARU)
		SchoolTeacherSchoolNameSnapshot:          m.SchoolTeacherSchoolNameSnapshot,
		SchoolTeacherSchoolSlugSnapshot:          m.SchoolTeacherSchoolSlugSnapshot,
		SchoolTeacherSchoolLogoURLSnapshot:       m.SchoolTeacherSchoolLogoURLSnapshot,
		SchoolTeacherSchoolIconURLSnapshot:       m.SchoolTeacherSchoolIconURLSnapshot,       // ⟵ BARU
		SchoolTeacherSchoolBackgroundURLSnapshot: m.SchoolTeacherSchoolBackgroundURLSnapshot, // ⟵ BARU

		// JSONB
		SchoolTeacherSections: sections,
		SchoolTeacherCSST:     cssts,

		// Audit
		SchoolTeacherCreatedAt: m.SchoolTeacherCreatedAt,
		SchoolTeacherUpdatedAt: m.SchoolTeacherUpdatedAt,
		SchoolTeacherDeletedAt: delAt,
	}
}

func NewSchoolTeacherResponses(items []yModel.SchoolTeacherModel) []*SchoolTeacher {
	out := make([]*SchoolTeacher, 0, len(items))
	for i := range items {
		out = append(out, NewSchoolTeacherResponse(&items[i]))
	}
	return out
}

/* ========================
   🛠️ Helpers (parse & validate)
   ======================== */

func parseDateYYYYMMDD(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	ss := strings.TrimSpace(*s)
	if ss == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", ss, time.UTC)
	if err != nil {
		return nil, fmt.Errorf("format tanggal harus YYYY-MM-DD")
	}
	return &t, nil
}

func parseRFC3339(ts *string) (*time.Time, error) {
	if ts == nil {
		return nil, nil
	}
	ss := strings.TrimSpace(*ts)
	if ss == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, ss)
	if err != nil {
		return nil, fmt.Errorf("format waktu harus RFC3339")
	}
	return &t, nil
}

func parseEmploymentPtr(s *string) (*yModel.TeacherEmployment, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.ToLower(strings.TrimSpace(*s))
	if v == "" {
		return nil, nil
	}
	switch v {
	case "tetap", "kontrak", "paruh_waktu", "magang", "honorer", "relawan", "tamu":
		st := yModel.TeacherEmployment(v)
		return &st, nil
	default:
		return nil, fmt.Errorf("employment harus salah satu dari: tetap, kontrak, paruh_waktu, magang, honorer, relawan, tamu")
	}
}

func uuidFrom(s string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(s))
}

/* ========================
   🧱 Mapping ke Model
   ======================== */

func (r CreateSchoolTeacherRequest) ToModel(schoolID string) (*yModel.SchoolTeacherModel, error) {
	rc := r
	rc.Normalize()

	if strings.TrimSpace(schoolID) == "" {
		return nil, fmt.Errorf("school_id wajib")
	}
	if rc.SchoolTeacherUserTeacherID == nil || strings.TrimSpace(*rc.SchoolTeacherUserTeacherID) == "" {
		return nil, fmt.Errorf("user_teacher_id wajib")
	}

	mzID, err := uuidFrom(schoolID)
	if err != nil {
		return nil, fmt.Errorf("school_id tidak valid")
	}
	utID, err := uuidFrom(*rc.SchoolTeacherUserTeacherID)
	if err != nil {
		return nil, fmt.Errorf("user_teacher_id tidak valid")
	}

	emp, err := parseEmploymentPtr(rc.SchoolTeacherEmployment)
	if err != nil {
		return nil, err
	}
	joinedAt, err := parseDateYYYYMMDD(rc.SchoolTeacherJoinedAt)
	if err != nil {
		return nil, err
	}
	leftAt, err := parseDateYYYYMMDD(rc.SchoolTeacherLeftAt)
	if err != nil {
		return nil, err
	}
	if joinedAt != nil && leftAt != nil && leftAt.Before(*joinedAt) {
		return nil, fmt.Errorf("left_at harus >= joined_at")
	}
	verifiedAt, err := parseRFC3339(rc.SchoolTeacherVerifiedAt)
	if err != nil {
		return nil, err
	}

	isActive := true
	if rc.SchoolTeacherIsActive != nil {
		isActive = *rc.SchoolTeacherIsActive
	}
	isPublic := true
	if rc.SchoolTeacherIsPublic != nil {
		isPublic = *rc.SchoolTeacherIsPublic
	}
	isVerified := false
	if rc.SchoolTeacherIsVerified != nil {
		isVerified = *rc.SchoolTeacherIsVerified
	}

	// Penting: inisialisasi JSONB agar tidak NULL pada insert
	emptyArr := datatypes.JSON([]byte("[]"))

	return &yModel.SchoolTeacherModel{
		SchoolTeacherSchoolID:      mzID,
		SchoolTeacherUserTeacherID: utID,

		SchoolTeacherCode:       rc.SchoolTeacherCode,
		SchoolTeacherSlug:       rc.SchoolTeacherSlug,
		SchoolTeacherEmployment: emp,

		SchoolTeacherIsActive: isActive,
		SchoolTeacherJoinedAt: joinedAt,
		SchoolTeacherLeftAt:   leftAt,

		SchoolTeacherIsVerified: isVerified,
		SchoolTeacherVerifiedAt: verifiedAt,

		SchoolTeacherIsPublic: isPublic,
		SchoolTeacherNotes:    rc.SchoolTeacherNotes,

		// JSONB defaults
		SchoolTeacherSections: emptyArr,
		SchoolTeacherCSST:     emptyArr,
	}, nil
}

func (r UpdateSchoolTeacherRequest) ApplyToModel(m *yModel.SchoolTeacherModel) error {
	ru := r
	ru.Normalize()

	// scope/user_teacher
	if ru.SchoolTeacherUserTeacherID != nil {
		id, err := uuidFrom(*ru.SchoolTeacherUserTeacherID)
		if err != nil {
			return fmt.Errorf("user_teacher_id tidak valid")
		}
		m.SchoolTeacherUserTeacherID = id
	}

	// identitas
	if ru.SchoolTeacherCode != nil {
		m.SchoolTeacherCode = ru.SchoolTeacherCode
	}
	if ru.SchoolTeacherSlug != nil {
		m.SchoolTeacherSlug = ru.SchoolTeacherSlug
	}
	if ru.SchoolTeacherEmployment != nil {
		emp, err := parseEmploymentPtr(ru.SchoolTeacherEmployment)
		if err != nil {
			return err
		}
		m.SchoolTeacherEmployment = emp
	}

	// flags
	if ru.SchoolTeacherIsActive != nil {
		m.SchoolTeacherIsActive = *ru.SchoolTeacherIsActive
	}
	if ru.SchoolTeacherIsPublic != nil {
		m.SchoolTeacherIsPublic = *ru.SchoolTeacherIsPublic
	}
	if ru.SchoolTeacherIsVerified != nil {
		m.SchoolTeacherIsVerified = *ru.SchoolTeacherIsVerified
	}

	// tanggal
	if ru.SchoolTeacherJoinedAt != nil {
		t, err := parseDateYYYYMMDD(ru.SchoolTeacherJoinedAt)
		if err != nil {
			return err
		}
		m.SchoolTeacherJoinedAt = t
	}
	if ru.SchoolTeacherLeftAt != nil {
		t, err := parseDateYYYYMMDD(ru.SchoolTeacherLeftAt)
		if err != nil {
			return err
		}
		m.SchoolTeacherLeftAt = t
	}
	if m.SchoolTeacherJoinedAt != nil && m.SchoolTeacherLeftAt != nil &&
		m.SchoolTeacherLeftAt.Before(*m.SchoolTeacherJoinedAt) {
		return fmt.Errorf("left_at harus >= joined_at")
	}

	if ru.SchoolTeacherVerifiedAt != nil {
		t, err := parseRFC3339(ru.SchoolTeacherVerifiedAt)
		if err != nil {
			return err
		}
		m.SchoolTeacherVerifiedAt = t
	}

	// notes
	if ru.SchoolTeacherNotes != nil {
		m.SchoolTeacherNotes = ru.SchoolTeacherNotes
	}

	return nil
}
