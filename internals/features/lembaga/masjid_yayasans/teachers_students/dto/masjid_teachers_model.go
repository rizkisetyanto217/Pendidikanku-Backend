// internals/features/lembaga/teachers/dto/masjid_teacher_dto.go
package dto

import (
	"fmt"
	"strings"
	"time"

	yModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/teachers_students/model"

	"github.com/google/uuid"
)

// ========================
// 📦 DTO Full Mirror (Entity Snapshot)
// ========================
type MasjidTeacher struct {
	MasjidTeacherID            string `json:"masjid_teacher_id"`
	MasjidTeacherMasjidID      string `json:"masjid_teacher_masjid_id"`
	MasjidTeacherUserTeacherID string `json:"masjid_teacher_user_teacher_id"` // ← updated

	// Identitas/kepegawaian
	MasjidTeacherCode       *string `json:"masjid_teacher_code,omitempty"`
	MasjidTeacherSlug       *string `json:"masjid_teacher_slug,omitempty"`
	MasjidTeacherEmployment *string `json:"masjid_teacher_employment,omitempty"` // enum as string
	MasjidTeacherIsActive   bool    `json:"masjid_teacher_is_active"`

	// Periode kerja
	MasjidTeacherJoinedAt *time.Time `json:"masjid_teacher_joined_at,omitempty"` // date
	MasjidTeacherLeftAt   *time.Time `json:"masjid_teacher_left_at,omitempty"`   // date

	// Verifikasi internal
	MasjidTeacherIsVerified bool       `json:"masjid_teacher_is_verified"`
	MasjidTeacherVerifiedAt *time.Time `json:"masjid_teacher_verified_at,omitempty"`

	// Visibilitas & catatan
	MasjidTeacherIsPublic bool    `json:"masjid_teacher_is_public"`
	MasjidTeacherNotes    *string `json:"masjid_teacher_notes,omitempty"`

	// Snapshot dari user_teachers
	MasjidTeacherNameUserSnapshot        *string `json:"masjid_teacher_name_user_snapshot,omitempty"`
	MasjidTeacherAvatarURLUserSnapshot   *string `json:"masjid_teacher_avatar_url_user_snapshot,omitempty"`
	MasjidTeacherWhatsappURLUserSnapshot *string `json:"masjid_teacher_whatsapp_url_user_snapshot,omitempty"`
	MasjidTeacherTitlePrefixUserSnapshot *string `json:"masjid_teacher_title_prefix_user_snapshot,omitempty"`
	MasjidTeacherTitleSuffixUserSnapshot *string `json:"masjid_teacher_title_suffix_user_snapshot,omitempty"`

	// Audit
	MasjidTeacherCreatedAt time.Time  `json:"masjid_teacher_created_at"`
	MasjidTeacherUpdatedAt time.Time  `json:"masjid_teacher_updated_at"`
	MasjidTeacherDeletedAt *time.Time `json:"masjid_teacher_deleted_at,omitempty"`
}

// ========================
// 📥 Create Request DTO
// ========================
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
	MasjidTeacherJoinedAt   *string `json:"masjid_teacher_joined_at,omitempty"` // YYYY-MM-DD
	MasjidTeacherLeftAt     *string `json:"masjid_teacher_left_at,omitempty"`   // YYYY-MM-DD
	MasjidTeacherIsVerified *bool   `json:"masjid_teacher_is_verified,omitempty"`
	MasjidTeacherVerifiedAt *string `json:"masjid_teacher_verified_at,omitempty"` // RFC3339

	// Notes
	MasjidTeacherNotes *string `json:"masjid_teacher_notes,omitempty"`
}

// ========================
// ✏️ Update Request DTO (tri-state)
// ========================
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

// ========================
// 📤 Response DTO (alias)
// ========================
type MasjidTeacherResponse = MasjidTeacher

// ========================
// 🔁 Converters
// ========================
func NewMasjidTeacherResponse(m *yModel.MasjidTeacherModel) *MasjidTeacherResponse {
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

	return &MasjidTeacherResponse{
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

		MasjidTeacherNameUserSnapshot:        m.MasjidTeacherNameUserSnapshot,
		MasjidTeacherAvatarURLUserSnapshot:   m.MasjidTeacherAvatarURLUserSnapshot,
		MasjidTeacherWhatsappURLUserSnapshot: m.MasjidTeacherWhatsappURLUserSnapshot,
		MasjidTeacherTitlePrefixUserSnapshot: m.MasjidTeacherTitlePrefixUserSnapshot,
		MasjidTeacherTitleSuffixUserSnapshot: m.MasjidTeacherTitleSuffixUserSnapshot,

		MasjidTeacherCreatedAt: m.MasjidTeacherCreatedAt,
		MasjidTeacherUpdatedAt: m.MasjidTeacherUpdatedAt,
		MasjidTeacherDeletedAt: delAt,
	}
}

func NewMasjidTeacherResponses(items []yModel.MasjidTeacherModel) []*MasjidTeacherResponse {
	out := make([]*MasjidTeacherResponse, 0, len(items))
	for i := range items {
		out = append(out, NewMasjidTeacherResponse(&items[i]))
	}
	return out
}

// ========================
// 🛠️ Helpers (parse & normalize)
// ========================
func parseDateYYYYMMDD(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	ss := strings.TrimSpace(*s)
	if ss == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", ss)
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

func normStrPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
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

// ========================
// 🧱 Mapping ke Model
// ========================

// Catatan: masjid_id biasanya dari path/context; user_teacher_id bisa dari body.
// Untuk konsistensi, fungsi ini menerima masjidID (wajib), sedangkan
// UserTeacherID diambil dari body (wajib juga).
func (r CreateMasjidTeacherRequest) ToModel(masjidID string) (*yModel.MasjidTeacherModel, error) {
	if strings.TrimSpace(masjidID) == "" {
		return nil, fmt.Errorf("masjid_id wajib")
	}
	if r.MasjidTeacherUserTeacherID == nil || strings.TrimSpace(*r.MasjidTeacherUserTeacherID) == "" {
		return nil, fmt.Errorf("user_teacher_id wajib")
	}

	mzID, err := uuidFrom(masjidID)
	if err != nil {
		return nil, fmt.Errorf("masjid_id tidak valid")
	}
	utID, err := uuidFrom(*r.MasjidTeacherUserTeacherID)
	if err != nil {
		return nil, fmt.Errorf("user_teacher_id tidak valid")
	}

	emp, err := parseEmploymentPtr(r.MasjidTeacherEmployment)
	if err != nil {
		return nil, err
	}
	joinedAt, err := parseDateYYYYMMDD(r.MasjidTeacherJoinedAt)
	if err != nil {
		return nil, err
	}
	leftAt, err := parseDateYYYYMMDD(r.MasjidTeacherLeftAt)
	if err != nil {
		return nil, err
	}
	if joinedAt != nil && leftAt != nil && leftAt.Before(*joinedAt) {
		return nil, fmt.Errorf("left_at harus >= joined_at")
	}
	verifiedAt, err := parseRFC3339(r.MasjidTeacherVerifiedAt)
	if err != nil {
		return nil, err
	}

	isActive := true
	if r.MasjidTeacherIsActive != nil {
		isActive = *r.MasjidTeacherIsActive
	}
	isPublic := true
	if r.MasjidTeacherIsPublic != nil {
		isPublic = *r.MasjidTeacherIsPublic
	}
	isVerified := false
	if r.MasjidTeacherIsVerified != nil {
		isVerified = *r.MasjidTeacherIsVerified
	}

	return &yModel.MasjidTeacherModel{
		MasjidTeacherMasjidID:      mzID,
		MasjidTeacherUserTeacherID: utID,

		MasjidTeacherCode:       normStrPtr(r.MasjidTeacherCode),
		MasjidTeacherSlug:       normStrPtr(r.MasjidTeacherSlug),
		MasjidTeacherEmployment: emp,

		MasjidTeacherIsActive: isActive,
		MasjidTeacherJoinedAt: joinedAt,
		MasjidTeacherLeftAt:   leftAt,

		MasjidTeacherIsVerified: isVerified,
		MasjidTeacherVerifiedAt: verifiedAt,

		MasjidTeacherIsPublic: isPublic,
		MasjidTeacherNotes:    normStrPtr(r.MasjidTeacherNotes),
	}, nil
}

func (r UpdateMasjidTeacherRequest) ApplyToModel(m *yModel.MasjidTeacherModel) error {
	// scope/user_teacher
	if r.MasjidTeacherUserTeacherID != nil {
		id, err := uuidFrom(*r.MasjidTeacherUserTeacherID)
		if err != nil {
			return fmt.Errorf("user_teacher_id tidak valid")
		}
		m.MasjidTeacherUserTeacherID = id
	}

	// identitas
	if r.MasjidTeacherCode != nil {
		m.MasjidTeacherCode = normStrPtr(r.MasjidTeacherCode)
	}
	if r.MasjidTeacherSlug != nil {
		m.MasjidTeacherSlug = normStrPtr(r.MasjidTeacherSlug)
	}
	if r.MasjidTeacherEmployment != nil {
		emp, err := parseEmploymentPtr(r.MasjidTeacherEmployment)
		if err != nil {
			return err
		}
		m.MasjidTeacherEmployment = emp
	}

	// flags
	if r.MasjidTeacherIsActive != nil {
		m.MasjidTeacherIsActive = *r.MasjidTeacherIsActive
	}
	if r.MasjidTeacherIsPublic != nil {
		m.MasjidTeacherIsPublic = *r.MasjidTeacherIsPublic
	}
	if r.MasjidTeacherIsVerified != nil {
		m.MasjidTeacherIsVerified = *r.MasjidTeacherIsVerified
	}

	// tanggal
	if r.MasjidTeacherJoinedAt != nil {
		t, err := parseDateYYYYMMDD(r.MasjidTeacherJoinedAt)
		if err != nil {
			return err
		}
		m.MasjidTeacherJoinedAt = t
	}
	if r.MasjidTeacherLeftAt != nil {
		t, err := parseDateYYYYMMDD(r.MasjidTeacherLeftAt)
		if err != nil {
			return err
		}
		m.MasjidTeacherLeftAt = t
	}
	if m.MasjidTeacherJoinedAt != nil && m.MasjidTeacherLeftAt != nil &&
		m.MasjidTeacherLeftAt.Before(*m.MasjidTeacherJoinedAt) {
		return fmt.Errorf("left_at harus >= joined_at")
	}

	if r.MasjidTeacherVerifiedAt != nil {
		t, err := parseRFC3339(r.MasjidTeacherVerifiedAt)
		if err != nil {
			return err
		}
		m.MasjidTeacherVerifiedAt = t
	}

	// notes
	if r.MasjidTeacherNotes != nil {
		m.MasjidTeacherNotes = normStrPtr(r.MasjidTeacherNotes)
	}

	return nil
}
