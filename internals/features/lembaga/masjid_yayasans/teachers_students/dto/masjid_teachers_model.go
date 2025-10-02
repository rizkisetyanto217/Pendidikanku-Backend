// internals/features/lembaga/teachers/dto/masjid_teacher_dto.go
package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	yModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model"

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

type MasjidTeacher struct {
	MasjidTeacherID            string `json:"masjid_teacher_id"`
	MasjidTeacherMasjidID      string `json:"masjid_teacher_masjid_id"`
	MasjidTeacherUserTeacherID string `json:"masjid_teacher_user_teacher_id"`

	// Identitas/kepegawaian
	MasjidTeacherCode       *string `json:"masjid_teacher_code,omitempty"`
	MasjidTeacherSlug       *string `json:"masjid_teacher_slug,omitempty"`
	MasjidTeacherEmployment *string `json:"masjid_teacher_employment,omitempty"` // enum as string
	MasjidTeacherIsActive   bool    `json:"masjid_teacher_is_active"`

	// Periode kerja
	MasjidTeacherJoinedAt *time.Time `json:"masjid_teacher_joined_at,omitempty"`
	MasjidTeacherLeftAt   *time.Time `json:"masjid_teacher_left_at,omitempty"`

	// Verifikasi internal
	MasjidTeacherIsVerified bool       `json:"masjid_teacher_is_verified"`
	MasjidTeacherVerifiedAt *time.Time `json:"masjid_teacher_verified_at,omitempty"`

	// Visibilitas & catatan
	MasjidTeacherIsPublic bool    `json:"masjid_teacher_is_public"`
	MasjidTeacherNotes    *string `json:"masjid_teacher_notes,omitempty"`

	// Snapshot dari user_teachers
	MasjidTeacherUserTeacherNameSnapshot        *string `json:"masjid_teacher_user_teacher_name_snapshot,omitempty"`
	MasjidTeacherUserTeacherAvatarURLSnapshot   *string `json:"masjid_teacher_user_teacher_avatar_url_snapshot,omitempty"`
	MasjidTeacherUserTeacherWhatsappURLSnapshot *string `json:"masjid_teacher_user_teacher_whatsapp_url_snapshot,omitempty"`
	MasjidTeacherUserTeacherTitlePrefixSnapshot *string `json:"masjid_teacher_user_teacher_title_prefix_snapshot,omitempty"`
	MasjidTeacherUserTeacherTitleSuffixSnapshot *string `json:"masjid_teacher_user_teacher_title_suffix_snapshot,omitempty"`

	// MASJID SNAPSHOT (sinkron dgn model terbaru)
	MasjidTeacherMasjidNameSnapshot          *string `json:"masjid_teacher_masjid_name_snapshot,omitempty"`
	MasjidTeacherMasjidSlugSnapshot          *string `json:"masjid_teacher_masjid_slug_snapshot,omitempty"`
	MasjidTeacherMasjidLogoURLSnapshot       *string `json:"masjid_teacher_masjid_logo_url_snapshot,omitempty"`
	MasjidTeacherMasjidIconURLSnapshot       *string `json:"masjid_teacher_masjid_icon_url_snapshot,omitempty"`       // ⟵ BARU
	MasjidTeacherMasjidBackgroundURLSnapshot *string `json:"masjid_teacher_masjid_background_url_snapshot,omitempty"` // ⟵ BARU

	// JSONB: sections & csst
	MasjidTeacherSections []DTOTeacherSectionItem `json:"masjid_teacher_sections"`
	MasjidTeacherCSST     []DTOTeacherCSSTItem    `json:"masjid_teacher_csst"`

	// Audit
	MasjidTeacherCreatedAt time.Time  `json:"masjid_teacher_created_at"`
	MasjidTeacherUpdatedAt time.Time  `json:"masjid_teacher_updated_at"`
	MasjidTeacherDeletedAt *time.Time `json:"masjid_teacher_deleted_at,omitempty"`
}

/* ========================
   📥 Create Request DTO
   ======================== */

type CreateMasjidTeacherRequest struct {
	// Scope/relasi
	MasjidTeacherUserTeacherID *string `json:"masjid_teacher_user_teacher_id,omitempty" validate:"omitempty,uuid"`

	// Identitas/kepegawaian
	MasjidTeacherCode       *string `json:"masjid_teacher_code,omitempty"`
	MasjidTeacherSlug       *string `json:"masjid_teacher_slug,omitempty"`
	MasjidTeacherEmployment *string `json:"masjid_teacher_employment,omitempty"` // enum

	// Flags
	MasjidTeacherIsActive *bool `json:"masjid_teacher_is_active,omitempty"`
	MasjidTeacherIsPublic *bool `json:"masjid_teacher_is_public,omitempty"`

	// Waktu
	MasjidTeacherJoinedAt   *string `json:"masjid_teacher_joined_at,omitempty"`   // YYYY-MM-DD
	MasjidTeacherLeftAt     *string `json:"masjid_teacher_left_at,omitempty"`     // YYYY-MM-DD
	MasjidTeacherIsVerified *bool   `json:"masjid_teacher_is_verified,omitempty"` // default: false
	MasjidTeacherVerifiedAt *string `json:"masjid_teacher_verified_at,omitempty"` // RFC3339

	// Notes
	MasjidTeacherNotes *string `json:"masjid_teacher_notes,omitempty"`
}

/* ========================
   ✏️ Update Request DTO (pointer = tri-state)
   ======================== */

type UpdateMasjidTeacherRequest struct {
	MasjidTeacherUserTeacherID *string `json:"masjid_teacher_user_teacher_id,omitempty" validate:"omitempty,uuid"`

	MasjidTeacherCode       *string `json:"masjid_teacher_code,omitempty"`
	MasjidTeacherSlug       *string `json:"masjid_teacher_slug,omitempty"`
	MasjidTeacherEmployment *string `json:"masjid_teacher_employment,omitempty"`

	MasjidTeacherIsActive *bool `json:"masjid_teacher_is_active,omitempty"`
	MasjidTeacherIsPublic *bool `json:"masjid_teacher_is_public,omitempty"`

	MasjidTeacherJoinedAt   *string `json:"masjid_teacher_joined_at,omitempty"` // YYYY-MM-DD
	MasjidTeacherLeftAt     *string `json:"masjid_teacher_left_at,omitempty"`   // YYYY-MM-DD
	MasjidTeacherIsVerified *bool   `json:"masjid_teacher_is_verified,omitempty"`
	MasjidTeacherVerifiedAt *string `json:"masjid_teacher_verified_at,omitempty"` // RFC3339

	MasjidTeacherNotes *string `json:"masjid_teacher_notes,omitempty"`
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

func (r *CreateMasjidTeacherRequest) Normalize() {
	r.MasjidTeacherUserTeacherID = toNilIfEmpty(r.MasjidTeacherUserTeacherID)

	r.MasjidTeacherCode = toNilIfEmpty(r.MasjidTeacherCode)
	r.MasjidTeacherSlug = toLowerTrimmedOrNil(r.MasjidTeacherSlug) // slug → lower
	r.MasjidTeacherEmployment = toLowerTrimmedOrNil(r.MasjidTeacherEmployment)

	r.MasjidTeacherJoinedAt = toNilIfEmpty(r.MasjidTeacherJoinedAt)
	r.MasjidTeacherLeftAt = toNilIfEmpty(r.MasjidTeacherLeftAt)
	r.MasjidTeacherVerifiedAt = toNilIfEmpty(r.MasjidTeacherVerifiedAt)

	r.MasjidTeacherNotes = toNilIfEmpty(r.MasjidTeacherNotes)
}

func (r *UpdateMasjidTeacherRequest) Normalize() {
	r.MasjidTeacherUserTeacherID = toNilIfEmpty(r.MasjidTeacherUserTeacherID)

	r.MasjidTeacherCode = toNilIfEmpty(r.MasjidTeacherCode)
	r.MasjidTeacherSlug = toLowerTrimmedOrNil(r.MasjidTeacherSlug)
	r.MasjidTeacherEmployment = toLowerTrimmedOrNil(r.MasjidTeacherEmployment)

	r.MasjidTeacherJoinedAt = toNilIfEmpty(r.MasjidTeacherJoinedAt)
	r.MasjidTeacherLeftAt = toNilIfEmpty(r.MasjidTeacherLeftAt)
	r.MasjidTeacherVerifiedAt = toNilIfEmpty(r.MasjidTeacherVerifiedAt)

	r.MasjidTeacherNotes = toNilIfEmpty(r.MasjidTeacherNotes)
}

/* ========================
   🔁 Converters (Model -> DTO)
   ======================== */

func NewMasjidTeacherResponse(m *yModel.MasjidTeacherModel) *MasjidTeacher {
	if m == nil {
		return nil
	}

	var emp *string
	if m.MasjidTeacherEmployment != nil {
		s := string(*m.MasjidTeacherEmployment)
		emp = &s
	}

	var delAt *time.Time
	if m.MasjidTeacherDeletedAt.Valid {
		delAt = &m.MasjidTeacherDeletedAt.Time
	}

	// Unmarshal JSONB (sections & csst)
	var sections []DTOTeacherSectionItem
	if len(m.MasjidTeacherSections) > 0 {
		_ = json.Unmarshal(m.MasjidTeacherSections, &sections)
	}
	if sections == nil {
		sections = []DTOTeacherSectionItem{}
	}

	var cssts []DTOTeacherCSSTItem
	if len(m.MasjidTeacherCSST) > 0 {
		_ = json.Unmarshal(m.MasjidTeacherCSST, &cssts)
	}
	if cssts == nil {
		cssts = []DTOTeacherCSSTItem{}
	}

	return &MasjidTeacher{
		MasjidTeacherID:            m.MasjidTeacherID.String(),
		MasjidTeacherMasjidID:      m.MasjidTeacherMasjidID.String(),
		MasjidTeacherUserTeacherID: m.MasjidTeacherUserTeacherID.String(),

		MasjidTeacherCode:       m.MasjidTeacherCode,
		MasjidTeacherSlug:       m.MasjidTeacherSlug,
		MasjidTeacherEmployment: emp,
		MasjidTeacherIsActive:   m.MasjidTeacherIsActive,

		MasjidTeacherJoinedAt: m.MasjidTeacherJoinedAt,
		MasjidTeacherLeftAt:   m.MasjidTeacherLeftAt,

		MasjidTeacherIsVerified: m.MasjidTeacherIsVerified,
		MasjidTeacherVerifiedAt: m.MasjidTeacherVerifiedAt,

		MasjidTeacherIsPublic: m.MasjidTeacherIsPublic,
		MasjidTeacherNotes:    m.MasjidTeacherNotes,

		MasjidTeacherUserTeacherNameSnapshot:        m.MasjidTeacherUserTeacherNameSnapshot,
		MasjidTeacherUserTeacherAvatarURLSnapshot:   m.MasjidTeacherUserTeacherAvatarURLSnapshot,
		MasjidTeacherUserTeacherWhatsappURLSnapshot: m.MasjidTeacherUserTeacherWhatsappURLSnapshot,
		MasjidTeacherUserTeacherTitlePrefixSnapshot: m.MasjidTeacherUserTeacherTitlePrefixSnapshot,
		MasjidTeacherUserTeacherTitleSuffixSnapshot: m.MasjidTeacherUserTeacherTitleSuffixSnapshot,

		// Masjid snapshots (termasuk 2 field BARU)
		MasjidTeacherMasjidNameSnapshot:          m.MasjidTeacherMasjidNameSnapshot,
		MasjidTeacherMasjidSlugSnapshot:          m.MasjidTeacherMasjidSlugSnapshot,
		MasjidTeacherMasjidLogoURLSnapshot:       m.MasjidTeacherMasjidLogoURLSnapshot,
		MasjidTeacherMasjidIconURLSnapshot:       m.MasjidTeacherMasjidIconURLSnapshot,       // ⟵ BARU
		MasjidTeacherMasjidBackgroundURLSnapshot: m.MasjidTeacherMasjidBackgroundURLSnapshot, // ⟵ BARU

		// JSONB
		MasjidTeacherSections: sections,
		MasjidTeacherCSST:     cssts,

		// Audit
		MasjidTeacherCreatedAt: m.MasjidTeacherCreatedAt,
		MasjidTeacherUpdatedAt: m.MasjidTeacherUpdatedAt,
		MasjidTeacherDeletedAt: delAt,
	}
}

func NewMasjidTeacherResponses(items []yModel.MasjidTeacherModel) []*MasjidTeacher {
	out := make([]*MasjidTeacher, 0, len(items))
	for i := range items {
		out = append(out, NewMasjidTeacherResponse(&items[i]))
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

func (r CreateMasjidTeacherRequest) ToModel(masjidID string) (*yModel.MasjidTeacherModel, error) {
	rc := r
	rc.Normalize()

	if strings.TrimSpace(masjidID) == "" {
		return nil, fmt.Errorf("masjid_id wajib")
	}
	if rc.MasjidTeacherUserTeacherID == nil || strings.TrimSpace(*rc.MasjidTeacherUserTeacherID) == "" {
		return nil, fmt.Errorf("user_teacher_id wajib")
	}

	mzID, err := uuidFrom(masjidID)
	if err != nil {
		return nil, fmt.Errorf("masjid_id tidak valid")
	}
	utID, err := uuidFrom(*rc.MasjidTeacherUserTeacherID)
	if err != nil {
		return nil, fmt.Errorf("user_teacher_id tidak valid")
	}

	emp, err := parseEmploymentPtr(rc.MasjidTeacherEmployment)
	if err != nil {
		return nil, err
	}
	joinedAt, err := parseDateYYYYMMDD(rc.MasjidTeacherJoinedAt)
	if err != nil {
		return nil, err
	}
	leftAt, err := parseDateYYYYMMDD(rc.MasjidTeacherLeftAt)
	if err != nil {
		return nil, err
	}
	if joinedAt != nil && leftAt != nil && leftAt.Before(*joinedAt) {
		return nil, fmt.Errorf("left_at harus >= joined_at")
	}
	verifiedAt, err := parseRFC3339(rc.MasjidTeacherVerifiedAt)
	if err != nil {
		return nil, err
	}

	isActive := true
	if rc.MasjidTeacherIsActive != nil {
		isActive = *rc.MasjidTeacherIsActive
	}
	isPublic := true
	if rc.MasjidTeacherIsPublic != nil {
		isPublic = *rc.MasjidTeacherIsPublic
	}
	isVerified := false
	if rc.MasjidTeacherIsVerified != nil {
		isVerified = *rc.MasjidTeacherIsVerified
	}

	// Penting: inisialisasi JSONB agar tidak NULL pada insert
	emptyArr := datatypes.JSON([]byte("[]"))

	return &yModel.MasjidTeacherModel{
		MasjidTeacherMasjidID:      mzID,
		MasjidTeacherUserTeacherID: utID,

		MasjidTeacherCode:       rc.MasjidTeacherCode,
		MasjidTeacherSlug:       rc.MasjidTeacherSlug,
		MasjidTeacherEmployment: emp,

		MasjidTeacherIsActive: isActive,
		MasjidTeacherJoinedAt: joinedAt,
		MasjidTeacherLeftAt:   leftAt,

		MasjidTeacherIsVerified: isVerified,
		MasjidTeacherVerifiedAt: verifiedAt,

		MasjidTeacherIsPublic: isPublic,
		MasjidTeacherNotes:    rc.MasjidTeacherNotes,

		// JSONB defaults
		MasjidTeacherSections: emptyArr,
		MasjidTeacherCSST:     emptyArr,
	}, nil
}

func (r UpdateMasjidTeacherRequest) ApplyToModel(m *yModel.MasjidTeacherModel) error {
	ru := r
	ru.Normalize()

	// scope/user_teacher
	if ru.MasjidTeacherUserTeacherID != nil {
		id, err := uuidFrom(*ru.MasjidTeacherUserTeacherID)
		if err != nil {
			return fmt.Errorf("user_teacher_id tidak valid")
		}
		m.MasjidTeacherUserTeacherID = id
	}

	// identitas
	if ru.MasjidTeacherCode != nil {
		m.MasjidTeacherCode = ru.MasjidTeacherCode
	}
	if ru.MasjidTeacherSlug != nil {
		m.MasjidTeacherSlug = ru.MasjidTeacherSlug
	}
	if ru.MasjidTeacherEmployment != nil {
		emp, err := parseEmploymentPtr(ru.MasjidTeacherEmployment)
		if err != nil {
			return err
		}
		m.MasjidTeacherEmployment = emp
	}

	// flags
	if ru.MasjidTeacherIsActive != nil {
		m.MasjidTeacherIsActive = *ru.MasjidTeacherIsActive
	}
	if ru.MasjidTeacherIsPublic != nil {
		m.MasjidTeacherIsPublic = *ru.MasjidTeacherIsPublic
	}
	if ru.MasjidTeacherIsVerified != nil {
		m.MasjidTeacherIsVerified = *ru.MasjidTeacherIsVerified
	}

	// tanggal
	if ru.MasjidTeacherJoinedAt != nil {
		t, err := parseDateYYYYMMDD(ru.MasjidTeacherJoinedAt)
		if err != nil {
			return err
		}
		m.MasjidTeacherJoinedAt = t
	}
	if ru.MasjidTeacherLeftAt != nil {
		t, err := parseDateYYYYMMDD(ru.MasjidTeacherLeftAt)
		if err != nil {
			return err
		}
		m.MasjidTeacherLeftAt = t
	}
	if m.MasjidTeacherJoinedAt != nil && m.MasjidTeacherLeftAt != nil &&
		m.MasjidTeacherLeftAt.Before(*m.MasjidTeacherJoinedAt) {
		return fmt.Errorf("left_at harus >= joined_at")
	}

	if ru.MasjidTeacherVerifiedAt != nil {
		t, err := parseRFC3339(ru.MasjidTeacherVerifiedAt)
		if err != nil {
			return err
		}
		m.MasjidTeacherVerifiedAt = t
	}

	// notes
	if ru.MasjidTeacherNotes != nil {
		m.MasjidTeacherNotes = ru.MasjidTeacherNotes
	}

	return nil
}

