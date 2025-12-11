// internals/features/lembaga/teachers/dto/school_teacher_dto.go
package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	teacherModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	helperDbTime "madinahsalam_backend/internals/helpers/dbtime"
)

/* ========================
   📦 Item JSON (DTO view)
   ======================== */

type DTOTeacherSectionItem struct {
	ClassSectionID             uuid.UUID `json:"class_section_id"`
	ClassSectionRole           string    `json:"class_section_role"` // "homeroom" | "teacher" | "assistant"
	From                       *string   `json:"from,omitempty"`     // "YYYY-MM-DD"
	To                         *string   `json:"to,omitempty"`       // "YYYY-MM-DD"
	ClassSectionName           *string   `json:"class_section_name,omitempty"`
	ClassSectionSlug           *string   `json:"class_section_slug,omitempty"`
	ClassSectionImageURL       *string   `json:"class_section_image_url,omitempty"`
	ClassSectionImageObjectKey *string   `json:"class_section_image_object_key,omitempty"`
}

type DTOTeacherCSSTItem struct {
	CSSTID           uuid.UUID  `json:"csst_id"`
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
	SchoolTeacherUserTeacherFullNameCache    *string `json:"school_teacher_user_teacher_full_name_cache,omitempty"`
	SchoolTeacherUserTeacherAvatarURLCache   *string `json:"school_teacher_user_teacher_avatar_url_cache,omitempty"`
	SchoolTeacherUserTeacherWhatsappURLCache *string `json:"school_teacher_user_teacher_whatsapp_url_cache,omitempty"`
	SchoolTeacherUserTeacherTitlePrefixCache *string `json:"school_teacher_user_teacher_title_prefix_cache,omitempty"`
	SchoolTeacherUserTeacherTitleSuffixCache *string `json:"school_teacher_user_teacher_title_suffix_cache,omitempty"`
	SchoolTeacherUserTeacherGenderCache      *string `json:"school_teacher_user_teacher_gender_cache,omitempty"`

	// Nested (opsional, diisi dari service lain kalau mau)
	SchoolTeacherSections []DTOTeacherSectionItem `json:"school_teacher_sections"`
	SchoolTeacherCSST     []DTOTeacherCSSTItem    `json:"school_teacher_csst"`

	// Audit
	SchoolTeacherCreatedAt time.Time  `json:"school_teacher_created_at"`
	SchoolTeacherUpdatedAt time.Time  `json:"school_teacher_updated_at"`
	SchoolTeacherDeletedAt *time.Time `json:"school_teacher_deleted_at,omitempty"`
}

// ========================
// 📦 Compact DTO (untuk embed di tempat lain)
// ========================
type SchoolTeacherCompact struct {
	SchoolTeacherID            string `json:"school_teacher_id"`
	SchoolTeacherUserTeacherID string `json:"school_teacher_user_teacher_id"`

	// Identitas/kepegawaian ringkas
	SchoolTeacherCode       *string `json:"school_teacher_code,omitempty"`
	SchoolTeacherEmployment *string `json:"school_teacher_employment,omitempty"` // enum as string
	SchoolTeacherIsActive   bool    `json:"school_teacher_is_active"`

	// Periode kerja ringkas
	SchoolTeacherJoinedAt *time.Time `json:"school_teacher_joined_at,omitempty"`

	// Snapshot dari user_teachers
	SchoolTeacherUserTeacherFullNameCache    *string `json:"school_teacher_user_teacher_full_name_cache,omitempty"`
	SchoolTeacherUserTeacherAvatarURLCache   *string `json:"school_teacher_user_teacher_avatar_url_cache,omitempty"`
	SchoolTeacherUserTeacherWhatsappURLCache *string `json:"school_teacher_user_teacher_whatsapp_url_cache,omitempty"`
	SchoolTeacherUserTeacherTitlePrefixCache *string `json:"school_teacher_user_teacher_title_prefix_cache,omitempty"`
	SchoolTeacherUserTeacherTitleSuffixCache *string `json:"school_teacher_user_teacher_title_suffix_cache,omitempty"`
	SchoolTeacherUserTeacherGenderCache      *string `json:"school_teacher_user_teacher_gender_cache,omitempty"`
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

	SchoolTeacherJoinedAt   *string `json:"school_teacher_joined_at,omitempty"`   // YYYY-MM-DD
	SchoolTeacherLeftAt     *string `json:"school_teacher_left_at,omitempty"`     // YYYY-MM-DD
	SchoolTeacherIsVerified *bool   `json:"school_teacher_is_verified,omitempty"` // flag
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
   🧠 Time helpers (DB → school TZ)
   ======================== */

func normalizeTimePtr(c *fiber.Ctx, t *time.Time) *time.Time {
	return helperDbTime.ToSchoolTimePtr(c, t)
}

/* ========================
   🔁 Converters (Model -> DTO)
   ======================== */

func NewSchoolTeacherResponse(c *fiber.Ctx, m *teacherModel.SchoolTeacherModel) *SchoolTeacher {
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
		t := helperDbTime.ToSchoolTime(c, m.SchoolTeacherDeletedAt.Time)
		delAt = &t
	}

	return &SchoolTeacher{
		SchoolTeacherID:            m.SchoolTeacherID.String(),
		SchoolTeacherSchoolID:      m.SchoolTeacherSchoolID.String(),
		SchoolTeacherUserTeacherID: m.SchoolTeacherUserTeacherID.String(),

		SchoolTeacherCode:       m.SchoolTeacherCode,
		SchoolTeacherSlug:       m.SchoolTeacherSlug,
		SchoolTeacherEmployment: emp,
		SchoolTeacherIsActive:   m.SchoolTeacherIsActive,

		SchoolTeacherJoinedAt: normalizeTimePtr(c, m.SchoolTeacherJoinedAt),
		SchoolTeacherLeftAt:   normalizeTimePtr(c, m.SchoolTeacherLeftAt),

		SchoolTeacherIsVerified: m.SchoolTeacherIsVerified,
		SchoolTeacherVerifiedAt: normalizeTimePtr(c, m.SchoolTeacherVerifiedAt),

		SchoolTeacherIsPublic: m.SchoolTeacherIsPublic,
		SchoolTeacherNotes:    m.SchoolTeacherNotes,

		SchoolTeacherUserTeacherFullNameCache:    m.SchoolTeacherUserTeacherFullNameCache,
		SchoolTeacherUserTeacherAvatarURLCache:   m.SchoolTeacherUserTeacherAvatarURLCache,
		SchoolTeacherUserTeacherWhatsappURLCache: m.SchoolTeacherUserTeacherWhatsappURLCache,
		SchoolTeacherUserTeacherTitlePrefixCache: m.SchoolTeacherUserTeacherTitlePrefixCache,
		SchoolTeacherUserTeacherTitleSuffixCache: m.SchoolTeacherUserTeacherTitleSuffixCache,
		SchoolTeacherUserTeacherGenderCache:      m.SchoolTeacherUserTeacherGenderCache,

		// Nested default: kosong, bisa diisi manual di service/controller
		SchoolTeacherSections: []DTOTeacherSectionItem{},
		SchoolTeacherCSST:     []DTOTeacherCSSTItem{},

		SchoolTeacherCreatedAt: helperDbTime.ToSchoolTime(c, m.SchoolTeacherCreatedAt),
		SchoolTeacherUpdatedAt: helperDbTime.ToSchoolTime(c, m.SchoolTeacherUpdatedAt),
		SchoolTeacherDeletedAt: delAt,
	}
}

func NewSchoolTeacherResponses(c *fiber.Ctx, items []teacherModel.SchoolTeacherModel) []*SchoolTeacher {
	out := make([]*SchoolTeacher, 0, len(items))
	for i := range items {
		out = append(out, NewSchoolTeacherResponse(c, &items[i]))
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

func parseEmploymentPtr(s *string) (*teacherModel.TeacherEmployment, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.ToLower(strings.TrimSpace(*s))
	if v == "" {
		return nil, nil
	}
	switch v {
	case "tetap", "kontrak", "paruh_waktu", "magang", "honorer", "relawan", "tamu":
		st := teacherModel.TeacherEmployment(v)
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

func (r CreateSchoolTeacherRequest) ToModel(schoolID string) (*teacherModel.SchoolTeacherModel, error) {
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

	return &teacherModel.SchoolTeacherModel{
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
	}, nil
}

func (r UpdateSchoolTeacherRequest) ApplyToModel(m *teacherModel.SchoolTeacherModel) error {
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

/* ========================
   📦 Compact (model -> DTO)
   ======================== */

func NewSchoolTeacherCompact(c *fiber.Ctx, m *teacherModel.SchoolTeacherModel) *SchoolTeacherCompact {
	if m == nil {
		return nil
	}

	var emp *string
	if m.SchoolTeacherEmployment != nil {
		s := string(*m.SchoolTeacherEmployment)
		emp = &s
	}

	return &SchoolTeacherCompact{
		SchoolTeacherID:            m.SchoolTeacherID.String(),
		SchoolTeacherUserTeacherID: m.SchoolTeacherUserTeacherID.String(),

		SchoolTeacherCode:       m.SchoolTeacherCode,
		SchoolTeacherEmployment: emp,
		SchoolTeacherIsActive:   m.SchoolTeacherIsActive,

		SchoolTeacherJoinedAt: helperDbTime.ToSchoolTimePtr(c, m.SchoolTeacherJoinedAt),

		SchoolTeacherUserTeacherFullNameCache:    m.SchoolTeacherUserTeacherFullNameCache,
		SchoolTeacherUserTeacherAvatarURLCache:   m.SchoolTeacherUserTeacherAvatarURLCache,
		SchoolTeacherUserTeacherWhatsappURLCache: m.SchoolTeacherUserTeacherWhatsappURLCache,
		SchoolTeacherUserTeacherTitlePrefixCache: m.SchoolTeacherUserTeacherTitlePrefixCache,
		SchoolTeacherUserTeacherTitleSuffixCache: m.SchoolTeacherUserTeacherTitleSuffixCache,
		SchoolTeacherUserTeacherGenderCache:      m.SchoolTeacherUserTeacherGenderCache,
	}
}

func NewSchoolTeacherCompacts(c *fiber.Ctx, items []teacherModel.SchoolTeacherModel) []*SchoolTeacherCompact {
	out := make([]*SchoolTeacherCompact, 0, len(items))
	for i := range items {
		out = append(out, NewSchoolTeacherCompact(c, &items[i]))
	}
	return out
}
