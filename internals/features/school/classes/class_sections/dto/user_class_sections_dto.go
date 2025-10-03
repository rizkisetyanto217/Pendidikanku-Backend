// file: internals/features/school/classes/class_sections/dto/user_class_section_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	model "masjidku_backend/internals/features/school/classes/class_sections/model"
)

/* =========================================================
   HELPERS
========================================================= */

func isZeroUUID(id uuid.UUID) bool { return id == uuid.Nil }

func isValidStatus(s string) bool {
	switch s {
	case string(model.UserClassSectionActive),
		string(model.UserClassSectionInactive),
		string(model.UserClassSectionCompleted):
		return true
	}
	return false
}

func isValidResult(r string) bool {
	switch r {
	case string(model.UserClassSectionPassed),
		string(model.UserClassSectionFailed):
		return true
	}
	return false
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

/* =========================================================
   REQUEST: CREATE
========================================================= */

type UserClassSectionCreateReq struct {
	UserClassSectionMasjidStudentID uuid.UUID `json:"user_class_section_masjid_student_id"`
	UserClassSectionSectionID       uuid.UUID `json:"user_class_section_section_id"`
	UserClassSectionMasjidID        uuid.UUID `json:"user_class_section_masjid_id"`

	UserClassSectionStatus string          `json:"user_class_section_status,omitempty"` // default: active
	UserClassSectionResult *string         `json:"user_class_section_result,omitempty"`
	UserClassSectionFee    *datatypes.JSON `json:"user_class_section_fee_snapshot,omitempty"`

	// Snapshot users_profile (opsional; dibekukan saat enrol)
	UserClassSectionUserProfileNameSnapshot               *string `json:"user_class_section_user_profile_name_snapshot,omitempty"`
	UserClassSectionUserProfileAvatarURLSnapshot         *string `json:"user_class_section_user_profile_avatar_url_snapshot,omitempty"`
	UserClassSectionUserProfileWhatsappURLSnapshot       *string `json:"user_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	UserClassSectionUserProfileParentNameSnapshot        *string `json:"user_class_section_user_profile_parent_name_snapshot,omitempty"`
	UserClassSectionUserProfileParentWhatsappURLSnapshot *string `json:"user_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	UserClassSectionAssignedAt   *time.Time `json:"user_class_section_assigned_at,omitempty"`
	UserClassSectionUnassignedAt *time.Time `json:"user_class_section_unassigned_at,omitempty"`
	UserClassSectionCompletedAt  *time.Time `json:"user_class_section_completed_at,omitempty"`
}

func (r *UserClassSectionCreateReq) Normalize() {
	// status & result
	if r.UserClassSectionStatus == "" {
		r.UserClassSectionStatus = string(model.UserClassSectionActive)
	}
	r.UserClassSectionStatus = strings.ToLower(strings.TrimSpace(r.UserClassSectionStatus))

	if r.UserClassSectionResult != nil {
		res := strings.ToLower(strings.TrimSpace(*r.UserClassSectionResult))
		if res == "" {
			r.UserClassSectionResult = nil
		} else {
			r.UserClassSectionResult = &res
		}
	}

	// snapshots → trim whitespace; kosong → nil
	r.UserClassSectionUserProfileNameSnapshot = trimPtr(r.UserClassSectionUserProfileNameSnapshot)
	r.UserClassSectionUserProfileAvatarURLSnapshot = trimPtr(r.UserClassSectionUserProfileAvatarURLSnapshot)
	r.UserClassSectionUserProfileWhatsappURLSnapshot = trimPtr(r.UserClassSectionUserProfileWhatsappURLSnapshot)
	r.UserClassSectionUserProfileParentNameSnapshot = trimPtr(r.UserClassSectionUserProfileParentNameSnapshot)
	r.UserClassSectionUserProfileParentWhatsappURLSnapshot = trimPtr(r.UserClassSectionUserProfileParentWhatsappURLSnapshot)
}

func (r *UserClassSectionCreateReq) Validate() error {
	// UUID wajib
	if isZeroUUID(r.UserClassSectionMasjidStudentID) {
		return errors.New("user_class_section_masjid_student_id wajib diisi")
	}
	if isZeroUUID(r.UserClassSectionSectionID) {
		return errors.New("user_class_section_section_id wajib diisi")
	}
	if isZeroUUID(r.UserClassSectionMasjidID) {
		return errors.New("user_class_section_masjid_id wajib diisi")
	}

	// status
	if !isValidStatus(r.UserClassSectionStatus) {
		return errors.New("invalid user_class_section_status")
	}

	// result (jika ada) harus valid
	if r.UserClassSectionResult != nil && !isValidResult(*r.UserClassSectionResult) {
		return errors.New("invalid user_class_section_result (gunakan 'passed' atau 'failed')")
	}

	// aturan completed
	if r.UserClassSectionStatus == string(model.UserClassSectionCompleted) {
		if r.UserClassSectionResult == nil {
			return errors.New("user_class_section_result wajib diisi jika status completed")
		}
		if r.UserClassSectionCompletedAt == nil {
			return errors.New("user_class_section_completed_at wajib diisi jika status completed")
		}
	} else {
		// jika bukan completed, larang pengisian result/completed_at
		if r.UserClassSectionResult != nil {
			return errors.New("user_class_section_result harus kosong jika status bukan 'completed'")
		}
		if r.UserClassSectionCompletedAt != nil {
			return errors.New("user_class_section_completed_at harus kosong jika status bukan 'completed'")
		}
	}

	// validasi tanggal unassigned >= assigned (jika keduanya ada)
	if r.UserClassSectionAssignedAt != nil && r.UserClassSectionUnassignedAt != nil {
		if r.UserClassSectionUnassignedAt.Before(*r.UserClassSectionAssignedAt) {
			return errors.New("user_class_section_unassigned_at tidak boleh lebih awal dari user_class_section_assigned_at")
		}
	}

	return nil
}

func (r *UserClassSectionCreateReq) ToModel() *model.UserClassSection {
	m := &model.UserClassSection{
		UserClassSectionMasjidStudentID: r.UserClassSectionMasjidStudentID,
		UserClassSectionSectionID:       r.UserClassSectionSectionID,
		UserClassSectionMasjidID:        r.UserClassSectionMasjidID,
		UserClassSectionStatus:          model.UserClassSectionStatus(r.UserClassSectionStatus),
		// kolom JSONB nullable; kalau tidak dikirim, biarkan NULL
	}

	if r.UserClassSectionResult != nil {
		res := model.UserClassSectionResult(*r.UserClassSectionResult)
		m.UserClassSectionResult = &res
	}
	if r.UserClassSectionFee != nil {
		m.UserClassSectionFeeSnapshot = *r.UserClassSectionFee
	}

	// snapshots
	m.UserClassSectionUserProfileNameSnapshot = r.UserClassSectionUserProfileNameSnapshot
	m.UserClassSectionUserProfileAvatarURLSnapshot = r.UserClassSectionUserProfileAvatarURLSnapshot
	m.UserClassSectionUserProfileWhatsappURLSnapshot = r.UserClassSectionUserProfileWhatsappURLSnapshot
	m.UserClassSectionUserProfileParentNameSnapshot = r.UserClassSectionUserProfileParentNameSnapshot
	m.UserClassSectionUserProfileParentWhatsappURLSnapshot = r.UserClassSectionUserProfileParentWhatsappURLSnapshot

	// waktu
	if r.UserClassSectionAssignedAt != nil {
		m.UserClassSectionAssignedAt = *r.UserClassSectionAssignedAt
	}
	if r.UserClassSectionUnassignedAt != nil {
		m.UserClassSectionUnassignedAt = r.UserClassSectionUnassignedAt
	}
	if r.UserClassSectionCompletedAt != nil {
		m.UserClassSectionCompletedAt = r.UserClassSectionCompletedAt
	}
	return m
}

/* =========================================================
   REQUEST: PATCH (partial update)
========================================================= */

type PatchField[T any] struct {
	Set   bool `json:"set"`
	Value T    `json:"value,omitempty"`
}

func (p *PatchField[T]) IsZero() bool { return p == nil || !p.Set }

type UserClassSectionPatchReq struct {
	UserClassSectionStatus *PatchField[string]          `json:"user_class_section_status,omitempty"`
	UserClassSectionResult *PatchField[*string]         `json:"user_class_section_result,omitempty"`
	UserClassSectionFee    *PatchField[*datatypes.JSON] `json:"user_class_section_fee_snapshot,omitempty"`

	// Snapshot users_profile
	UserClassSectionUserProfileNameSnapshot               *PatchField[*string] `json:"user_class_section_user_profile_name_snapshot,omitempty"`
	UserClassSectionUserProfileAvatarURLSnapshot         *PatchField[*string] `json:"user_class_section_user_profile_avatar_url_snapshot,omitempty"`
	UserClassSectionUserProfileWhatsappURLSnapshot       *PatchField[*string] `json:"user_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	UserClassSectionUserProfileParentNameSnapshot        *PatchField[*string] `json:"user_class_section_user_profile_parent_name_snapshot,omitempty"`
	UserClassSectionUserProfileParentWhatsappURLSnapshot *PatchField[*string] `json:"user_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	UserClassSectionAssignedAt   *PatchField[*time.Time] `json:"user_class_section_assigned_at,omitempty"`
	UserClassSectionUnassignedAt *PatchField[*time.Time] `json:"user_class_section_unassigned_at,omitempty"`
	UserClassSectionCompletedAt  *PatchField[*time.Time] `json:"user_class_section_completed_at,omitempty"`
}

func (r *UserClassSectionPatchReq) Normalize() {
	if r.UserClassSectionStatus != nil && r.UserClassSectionStatus.Set {
		r.UserClassSectionStatus.Value = strings.ToLower(strings.TrimSpace(r.UserClassSectionStatus.Value))
	}
	if r.UserClassSectionResult != nil && r.UserClassSectionResult.Set && r.UserClassSectionResult.Value != nil {
		res := strings.ToLower(strings.TrimSpace(*r.UserClassSectionResult.Value))
		if res == "" {
			r.UserClassSectionResult.Value = nil
		} else {
			r.UserClassSectionResult.Value = &res
		}
	}

	// snapshots: trim; kosong → nil
	if r.UserClassSectionUserProfileNameSnapshot != nil && r.UserClassSectionUserProfileNameSnapshot.Set {
		r.UserClassSectionUserProfileNameSnapshot.Value = trimPtr(r.UserClassSectionUserProfileNameSnapshot.Value)
	}
	if r.UserClassSectionUserProfileAvatarURLSnapshot != nil && r.UserClassSectionUserProfileAvatarURLSnapshot.Set {
		r.UserClassSectionUserProfileAvatarURLSnapshot.Value = trimPtr(r.UserClassSectionUserProfileAvatarURLSnapshot.Value)
	}
	if r.UserClassSectionUserProfileWhatsappURLSnapshot != nil && r.UserClassSectionUserProfileWhatsappURLSnapshot.Set {
		r.UserClassSectionUserProfileWhatsappURLSnapshot.Value = trimPtr(r.UserClassSectionUserProfileWhatsappURLSnapshot.Value)
	}
	if r.UserClassSectionUserProfileParentNameSnapshot != nil && r.UserClassSectionUserProfileParentNameSnapshot.Set {
		r.UserClassSectionUserProfileParentNameSnapshot.Value = trimPtr(r.UserClassSectionUserProfileParentNameSnapshot.Value)
	}
	if r.UserClassSectionUserProfileParentWhatsappURLSnapshot != nil && r.UserClassSectionUserProfileParentWhatsappURLSnapshot.Set {
		r.UserClassSectionUserProfileParentWhatsappURLSnapshot.Value = trimPtr(r.UserClassSectionUserProfileParentWhatsappURLSnapshot.Value)
	}
}

func (r *UserClassSectionPatchReq) Validate() error {
	// status (jika di-set) harus valid
	if r.UserClassSectionStatus != nil && r.UserClassSectionStatus.Set {
		if !isValidStatus(r.UserClassSectionStatus.Value) {
			return errors.New("invalid status")
		}
	}

	// result (jika di-set & tidak nil) harus valid
	if r.UserClassSectionResult != nil && r.UserClassSectionResult.Set && r.UserClassSectionResult.Value != nil {
		if !isValidResult(*r.UserClassSectionResult.Value) {
			return errors.New("invalid result (gunakan 'passed' atau 'failed')")
		}
	}

	// aturan completed (jika status di-set menjadi completed)
	if r.UserClassSectionStatus != nil && r.UserClassSectionStatus.Set &&
		r.UserClassSectionStatus.Value == string(model.UserClassSectionCompleted) {

		if r.UserClassSectionCompletedAt == nil || !r.UserClassSectionCompletedAt.Set || r.UserClassSectionCompletedAt.Value == nil {
			return errors.New("completed_at wajib diisi jika status completed")
		}
		if r.UserClassSectionResult == nil || !r.UserClassSectionResult.Set || r.UserClassSectionResult.Value == nil {
			return errors.New("result wajib diisi jika status completed")
		}
	}

	// jika status di-set menjadi non-completed, larang set result/completed_at bersamaan
	if r.UserClassSectionStatus != nil && r.UserClassSectionStatus.Set &&
		r.UserClassSectionStatus.Value != string(model.UserClassSectionCompleted) {

		if r.UserClassSectionResult != nil && r.UserClassSectionResult.Set && r.UserClassSectionResult.Value != nil {
			return errors.New("result harus dikosongkan jika status bukan 'completed'")
		}
		if r.UserClassSectionCompletedAt != nil && r.UserClassSectionCompletedAt.Set && r.UserClassSectionCompletedAt.Value != nil {
			return errors.New("completed_at harus dikosongkan jika status bukan 'completed'")
		}
	}

	// validasi tanggal unassigned >= assigned jika keduanya di-set
	if r.UserClassSectionAssignedAt != nil && r.UserClassSectionAssignedAt.Set &&
		r.UserClassSectionAssignedAt.Value != nil &&
		r.UserClassSectionUnassignedAt != nil && r.UserClassSectionUnassignedAt.Set &&
		r.UserClassSectionUnassignedAt.Value != nil {

		if r.UserClassSectionUnassignedAt.Value.Before(*r.UserClassSectionAssignedAt.Value) {
			return errors.New("unassigned_at tidak boleh lebih awal dari assigned_at")
		}
	}

	// Kolom assigned_at di DB NOT NULL → jangan izinkan clear (nil) eksplisit
	if r.UserClassSectionAssignedAt != nil && r.UserClassSectionAssignedAt.Set && r.UserClassSectionAssignedAt.Value == nil {
		return errors.New("assigned_at tidak boleh dikosongkan (NOT NULL)")
	}

	return nil
}

// Terapkan perubahan ke model (hanya yang Set=true)
func (r *UserClassSectionPatchReq) Apply(m *model.UserClassSection) {
	if r.UserClassSectionStatus != nil && r.UserClassSectionStatus.Set {
		m.UserClassSectionStatus = model.UserClassSectionStatus(r.UserClassSectionStatus.Value)
	}
	if r.UserClassSectionResult != nil && r.UserClassSectionResult.Set {
		if r.UserClassSectionResult.Value != nil {
			res := model.UserClassSectionResult(*r.UserClassSectionResult.Value)
			m.UserClassSectionResult = &res
		} else {
			m.UserClassSectionResult = nil
		}
	}
	if r.UserClassSectionFee != nil && r.UserClassSectionFee.Set {
		if r.UserClassSectionFee.Value != nil {
			m.UserClassSectionFeeSnapshot = *r.UserClassSectionFee.Value
		} else {
			// nullable → jika ingin clear, set ke NULL (bukan "{}")
			m.UserClassSectionFeeSnapshot = nil
		}
	}

	// snapshots
	if r.UserClassSectionUserProfileNameSnapshot != nil && r.UserClassSectionUserProfileNameSnapshot.Set {
		m.UserClassSectionUserProfileNameSnapshot = r.UserClassSectionUserProfileNameSnapshot.Value
	}
	if r.UserClassSectionUserProfileAvatarURLSnapshot != nil && r.UserClassSectionUserProfileAvatarURLSnapshot.Set {
		m.UserClassSectionUserProfileAvatarURLSnapshot = r.UserClassSectionUserProfileAvatarURLSnapshot.Value
	}
	if r.UserClassSectionUserProfileWhatsappURLSnapshot != nil && r.UserClassSectionUserProfileWhatsappURLSnapshot.Set {
		m.UserClassSectionUserProfileWhatsappURLSnapshot = r.UserClassSectionUserProfileWhatsappURLSnapshot.Value
	}
	if r.UserClassSectionUserProfileParentNameSnapshot != nil && r.UserClassSectionUserProfileParentNameSnapshot.Set {
		m.UserClassSectionUserProfileParentNameSnapshot = r.UserClassSectionUserProfileParentNameSnapshot.Value
	}
	if r.UserClassSectionUserProfileParentWhatsappURLSnapshot != nil && r.UserClassSectionUserProfileParentWhatsappURLSnapshot.Set {
		m.UserClassSectionUserProfileParentWhatsappURLSnapshot = r.UserClassSectionUserProfileParentWhatsappURLSnapshot.Value
	}

	// waktu
	if r.UserClassSectionAssignedAt != nil && r.UserClassSectionAssignedAt.Set && r.UserClassSectionAssignedAt.Value != nil {
		m.UserClassSectionAssignedAt = *r.UserClassSectionAssignedAt.Value
	}
	if r.UserClassSectionUnassignedAt != nil && r.UserClassSectionUnassignedAt.Set {
		m.UserClassSectionUnassignedAt = r.UserClassSectionUnassignedAt.Value
	}
	if r.UserClassSectionCompletedAt != nil && r.UserClassSectionCompletedAt.Set {
		m.UserClassSectionCompletedAt = r.UserClassSectionCompletedAt.Value
	}
}

/* =========================================================
   RESPONSE DTO
========================================================= */

type UserClassSectionResp struct {
	UserClassSectionID uuid.UUID `json:"user_class_section_id"`

	UserClassSectionMasjidStudentID uuid.UUID `json:"user_class_section_masjid_student_id"`
	UserClassSectionSectionID       uuid.UUID `json:"user_class_section_section_id"`
	UserClassSectionMasjidID        uuid.UUID `json:"user_class_section_masjid_id"`

	UserClassSectionStatus string          `json:"user_class_section_status"`
	UserClassSectionResult *string         `json:"user_class_section_result,omitempty"`
	UserClassSectionFee    *datatypes.JSON `json:"user_class_section_fee_snapshot,omitempty"`

	// Snapshot users_profile
	UserClassSectionUserProfileNameSnapshot               *string `json:"user_class_section_user_profile_name_snapshot,omitempty"`
	UserClassSectionUserProfileAvatarURLSnapshot         *string `json:"user_class_section_user_profile_avatar_url_snapshot,omitempty"`
	UserClassSectionUserProfileWhatsappURLSnapshot       *string `json:"user_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	UserClassSectionUserProfileParentNameSnapshot        *string `json:"user_class_section_user_profile_parent_name_snapshot,omitempty"`
	UserClassSectionUserProfileParentWhatsappURLSnapshot *string `json:"user_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	UserClassSectionAssignedAt   time.Time  `json:"user_class_section_assigned_at"`
	UserClassSectionUnassignedAt *time.Time `json:"user_class_section_unassigned_at,omitempty"`
	UserClassSectionCompletedAt  *time.Time `json:"user_class_section_completed_at,omitempty"`

	UserClassSectionCreatedAt time.Time  `json:"user_class_section_created_at"`
	UserClassSectionUpdatedAt time.Time  `json:"user_class_section_updated_at"`
	UserClassSectionDeletedAt *time.Time `json:"user_class_section_deleted_at,omitempty"`
}

func FromModel(m *model.UserClassSection) UserClassSectionResp {
	var res *string
	if m.UserClassSectionResult != nil {
		v := string(*m.UserClassSectionResult)
		res = &v
	}
	var delAt *time.Time
	if m.UserClassSectionDeletedAt.Valid {
		t := m.UserClassSectionDeletedAt.Time
		delAt = &t
	}
	var feePtr *datatypes.JSON
	if m.UserClassSectionFeeSnapshot != nil {
		tmp := m.UserClassSectionFeeSnapshot
		feePtr = &tmp
	}

	return UserClassSectionResp{
		UserClassSectionID: m.UserClassSectionID,

		UserClassSectionMasjidStudentID: m.UserClassSectionMasjidStudentID,
		UserClassSectionSectionID:       m.UserClassSectionSectionID,
		UserClassSectionMasjidID:        m.UserClassSectionMasjidID,

		UserClassSectionStatus: string(m.UserClassSectionStatus),
		UserClassSectionResult: res,
		UserClassSectionFee:    feePtr,

		// snapshots
		UserClassSectionUserProfileNameSnapshot:               m.UserClassSectionUserProfileNameSnapshot,
		UserClassSectionUserProfileAvatarURLSnapshot:         m.UserClassSectionUserProfileAvatarURLSnapshot,
		UserClassSectionUserProfileWhatsappURLSnapshot:       m.UserClassSectionUserProfileWhatsappURLSnapshot,
		UserClassSectionUserProfileParentNameSnapshot:        m.UserClassSectionUserProfileParentNameSnapshot,
		UserClassSectionUserProfileParentWhatsappURLSnapshot: m.UserClassSectionUserProfileParentWhatsappURLSnapshot,

		UserClassSectionAssignedAt:   m.UserClassSectionAssignedAt,
		UserClassSectionUnassignedAt: m.UserClassSectionUnassignedAt,
		UserClassSectionCompletedAt:  m.UserClassSectionCompletedAt,

		UserClassSectionCreatedAt: m.UserClassSectionCreatedAt,
		UserClassSectionUpdatedAt: m.UserClassSectionUpdatedAt,
		UserClassSectionDeletedAt: delAt,
	}
}
