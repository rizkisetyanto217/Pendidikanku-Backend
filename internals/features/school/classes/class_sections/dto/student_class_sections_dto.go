// file: internals/features/school/classes/class_sections/dto/student_class_section_dto.go
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
	case string(model.StudentClassSectionActive),
		string(model.StudentClassSectionInactive),
		string(model.StudentClassSectionCompleted):
		return true
	}
	return false
}

func isValidResult(r string) bool {
	switch r {
	case string(model.StudentClassSectionPassed),
		string(model.StudentClassSectionFailed):
		return true
	}
	return false
}

/* =========================================================
   REQUEST: CREATE
========================================================= */

type StudentClassSectionCreateReq struct {
	StudentClassSectionMasjidStudentID uuid.UUID `json:"student_class_section_masjid_student_id"`
	StudentClassSectionSectionID       uuid.UUID `json:"student_class_section_section_id"`
	StudentClassSectionMasjidID        uuid.UUID `json:"student_class_section_masjid_id"`

	StudentClassSectionStatus string          `json:"student_class_section_status,omitempty"` // default: active
	StudentClassSectionResult *string         `json:"student_class_section_result,omitempty"`
	StudentClassSectionFee    *datatypes.JSON `json:"student_class_section_fee_snapshot,omitempty"`

	// Snapshot users_profile (opsional; dibekukan saat enrol)
	StudentClassSectionUserProfileNameSnapshot              *string `json:"student_class_section_user_profile_name_snapshot,omitempty"`
	StudentClassSectionUserProfileAvatarURLSnapshot         *string `json:"student_class_section_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionUserProfileWhatsappURLSnapshot       *string `json:"student_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileParentNameSnapshot        *string `json:"student_class_section_user_profile_parent_name_snapshot,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLSnapshot *string `json:"student_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	StudentClassSectionAssignedAt   *time.Time `json:"student_class_section_assigned_at,omitempty"`
	StudentClassSectionUnassignedAt *time.Time `json:"student_class_section_unassigned_at,omitempty"`
	StudentClassSectionCompletedAt  *time.Time `json:"student_class_section_completed_at,omitempty"`
}

func (r *StudentClassSectionCreateReq) Normalize() {
	// status & result
	if r.StudentClassSectionStatus == "" {
		r.StudentClassSectionStatus = string(model.StudentClassSectionActive)
	}
	r.StudentClassSectionStatus = strings.ToLower(strings.TrimSpace(r.StudentClassSectionStatus))

	if r.StudentClassSectionResult != nil {
		res := strings.ToLower(strings.TrimSpace(*r.StudentClassSectionResult))
		if res == "" {
			r.StudentClassSectionResult = nil
		} else {
			r.StudentClassSectionResult = &res
		}
	}

	// snapshots → trim whitespace; kosong → nil
	r.StudentClassSectionUserProfileNameSnapshot = trimPtr(r.StudentClassSectionUserProfileNameSnapshot)
	r.StudentClassSectionUserProfileAvatarURLSnapshot = trimPtr(r.StudentClassSectionUserProfileAvatarURLSnapshot)
	r.StudentClassSectionUserProfileWhatsappURLSnapshot = trimPtr(r.StudentClassSectionUserProfileWhatsappURLSnapshot)
	r.StudentClassSectionUserProfileParentNameSnapshot = trimPtr(r.StudentClassSectionUserProfileParentNameSnapshot)
	r.StudentClassSectionUserProfileParentWhatsappURLSnapshot = trimPtr(r.StudentClassSectionUserProfileParentWhatsappURLSnapshot)
}

func (r *StudentClassSectionCreateReq) Validate() error {
	// UUID wajib
	if isZeroUUID(r.StudentClassSectionMasjidStudentID) {
		return errors.New("student_class_section_masjid_student_id wajib diisi")
	}
	if isZeroUUID(r.StudentClassSectionSectionID) {
		return errors.New("student_class_section_section_id wajib diisi")
	}
	if isZeroUUID(r.StudentClassSectionMasjidID) {
		return errors.New("student_class_section_masjid_id wajib diisi")
	}

	// status
	if !isValidStatus(r.StudentClassSectionStatus) {
		return errors.New("invalid student_class_section_status")
	}

	// result (jika ada) harus valid
	if r.StudentClassSectionResult != nil && !isValidResult(*r.StudentClassSectionResult) {
		return errors.New("invalid student_class_section_result (gunakan 'passed' atau 'failed')")
	}

	// aturan completed
	if r.StudentClassSectionStatus == string(model.StudentClassSectionCompleted) {
		if r.StudentClassSectionResult == nil {
			return errors.New("student_class_section_result wajib diisi jika status completed")
		}
		if r.StudentClassSectionCompletedAt == nil {
			return errors.New("student_class_section_completed_at wajib diisi jika status completed")
		}
	} else {
		// jika bukan completed, larang pengisian result/completed_at
		if r.StudentClassSectionResult != nil {
			return errors.New("student_class_section_result harus kosong jika status bukan 'completed'")
		}
		if r.StudentClassSectionCompletedAt != nil {
			return errors.New("student_class_section_completed_at harus kosong jika status bukan 'completed'")
		}
	}

	// validasi tanggal unassigned >= assigned (jika keduanya ada)
	if r.StudentClassSectionAssignedAt != nil && r.StudentClassSectionUnassignedAt != nil {
		if r.StudentClassSectionUnassignedAt.Before(*r.StudentClassSectionAssignedAt) {
			return errors.New("student_class_section_unassigned_at tidak boleh lebih awal dari student_class_section_assigned_at")
		}
	}

	return nil
}

func (r *StudentClassSectionCreateReq) ToModel() *model.StudentClassSection {
	m := &model.StudentClassSection{
		StudentClassSectionMasjidStudentID: r.StudentClassSectionMasjidStudentID,
		StudentClassSectionSectionID:       r.StudentClassSectionSectionID,
		StudentClassSectionMasjidID:        r.StudentClassSectionMasjidID,
		StudentClassSectionStatus:          model.StudentClassSectionStatus(r.StudentClassSectionStatus),
		// kolom JSONB nullable; kalau tidak dikirim, biarkan NULL
	}

	if r.StudentClassSectionResult != nil {
		res := model.StudentClassSectionResult(*r.StudentClassSectionResult)
		m.StudentClassSectionResult = &res
	}
	if r.StudentClassSectionFee != nil {
		m.StudentClassSectionFeeSnapshot = *r.StudentClassSectionFee
	}

	// snapshots
	m.StudentClassSectionUserProfileNameSnapshot = r.StudentClassSectionUserProfileNameSnapshot
	m.StudentClassSectionUserProfileAvatarURLSnapshot = r.StudentClassSectionUserProfileAvatarURLSnapshot
	m.StudentClassSectionUserProfileWhatsappURLSnapshot = r.StudentClassSectionUserProfileWhatsappURLSnapshot
	m.StudentClassSectionUserProfileParentNameSnapshot = r.StudentClassSectionUserProfileParentNameSnapshot
	m.StudentClassSectionUserProfileParentWhatsappURLSnapshot = r.StudentClassSectionUserProfileParentWhatsappURLSnapshot

	// waktu
	if r.StudentClassSectionAssignedAt != nil {
		m.StudentClassSectionAssignedAt = *r.StudentClassSectionAssignedAt
	}
	if r.StudentClassSectionUnassignedAt != nil {
		m.StudentClassSectionUnassignedAt = r.StudentClassSectionUnassignedAt
	}
	if r.StudentClassSectionCompletedAt != nil {
		m.StudentClassSectionCompletedAt = r.StudentClassSectionCompletedAt
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

type StudentClassSectionPatchReq struct {
	StudentClassSectionStatus *PatchField[string]          `json:"student_class_section_status,omitempty"`
	StudentClassSectionResult *PatchField[*string]         `json:"student_class_section_result,omitempty"`
	StudentClassSectionFee    *PatchField[*datatypes.JSON] `json:"student_class_section_fee_snapshot,omitempty"`

	// Snapshot users_profile
	StudentClassSectionUserProfileNameSnapshot              *PatchField[*string] `json:"student_class_section_user_profile_name_snapshot,omitempty"`
	StudentClassSectionUserProfileAvatarURLSnapshot         *PatchField[*string] `json:"student_class_section_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionUserProfileWhatsappURLSnapshot       *PatchField[*string] `json:"student_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileParentNameSnapshot        *PatchField[*string] `json:"student_class_section_user_profile_parent_name_snapshot,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLSnapshot *PatchField[*string] `json:"student_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	StudentClassSectionAssignedAt   *PatchField[*time.Time] `json:"student_class_section_assigned_at,omitempty"`
	StudentClassSectionUnassignedAt *PatchField[*time.Time] `json:"student_class_section_unassigned_at,omitempty"`
	StudentClassSectionCompletedAt  *PatchField[*time.Time] `json:"student_class_section_completed_at,omitempty"`
}

func (r *StudentClassSectionPatchReq) Normalize() {
	if r.StudentClassSectionStatus != nil && r.StudentClassSectionStatus.Set {
		r.StudentClassSectionStatus.Value = strings.ToLower(strings.TrimSpace(r.StudentClassSectionStatus.Value))
	}
	if r.StudentClassSectionResult != nil && r.StudentClassSectionResult.Set && r.StudentClassSectionResult.Value != nil {
		res := strings.ToLower(strings.TrimSpace(*r.StudentClassSectionResult.Value))
		if res == "" {
			r.StudentClassSectionResult.Value = nil
		} else {
			r.StudentClassSectionResult.Value = &res
		}
	}

	// snapshots: trim; kosong → nil
	if r.StudentClassSectionUserProfileNameSnapshot != nil && r.StudentClassSectionUserProfileNameSnapshot.Set {
		r.StudentClassSectionUserProfileNameSnapshot.Value = trimPtr(r.StudentClassSectionUserProfileNameSnapshot.Value)
	}
	if r.StudentClassSectionUserProfileAvatarURLSnapshot != nil && r.StudentClassSectionUserProfileAvatarURLSnapshot.Set {
		r.StudentClassSectionUserProfileAvatarURLSnapshot.Value = trimPtr(r.StudentClassSectionUserProfileAvatarURLSnapshot.Value)
	}
	if r.StudentClassSectionUserProfileWhatsappURLSnapshot != nil && r.StudentClassSectionUserProfileWhatsappURLSnapshot.Set {
		r.StudentClassSectionUserProfileWhatsappURLSnapshot.Value = trimPtr(r.StudentClassSectionUserProfileWhatsappURLSnapshot.Value)
	}
	if r.StudentClassSectionUserProfileParentNameSnapshot != nil && r.StudentClassSectionUserProfileParentNameSnapshot.Set {
		r.StudentClassSectionUserProfileParentNameSnapshot.Value = trimPtr(r.StudentClassSectionUserProfileParentNameSnapshot.Value)
	}
	if r.StudentClassSectionUserProfileParentWhatsappURLSnapshot != nil && r.StudentClassSectionUserProfileParentWhatsappURLSnapshot.Set {
		r.StudentClassSectionUserProfileParentWhatsappURLSnapshot.Value = trimPtr(r.StudentClassSectionUserProfileParentWhatsappURLSnapshot.Value)
	}
}

func (r *StudentClassSectionPatchReq) Validate() error {
	// status (jika di-set) harus valid
	if r.StudentClassSectionStatus != nil && r.StudentClassSectionStatus.Set {
		if !isValidStatus(r.StudentClassSectionStatus.Value) {
			return errors.New("invalid status")
		}
	}

	// result (jika di-set & tidak nil) harus valid
	if r.StudentClassSectionResult != nil && r.StudentClassSectionResult.Set && r.StudentClassSectionResult.Value != nil {
		if !isValidResult(*r.StudentClassSectionResult.Value) {
			return errors.New("invalid result (gunakan 'passed' atau 'failed')")
		}
	}

	// aturan completed (jika status di-set menjadi completed)
	if r.StudentClassSectionStatus != nil && r.StudentClassSectionStatus.Set &&
		r.StudentClassSectionStatus.Value == string(model.StudentClassSectionCompleted) {

		if r.StudentClassSectionCompletedAt == nil || !r.StudentClassSectionCompletedAt.Set || r.StudentClassSectionCompletedAt.Value == nil {
			return errors.New("completed_at wajib diisi jika status completed")
		}
		if r.StudentClassSectionResult == nil || !r.StudentClassSectionResult.Set || r.StudentClassSectionResult.Value == nil {
			return errors.New("result wajib diisi jika status completed")
		}
	}

	// jika status di-set menjadi non-completed, larang set result/completed_at bersamaan
	if r.StudentClassSectionStatus != nil && r.StudentClassSectionStatus.Set &&
		r.StudentClassSectionStatus.Value != string(model.StudentClassSectionCompleted) {

		if r.StudentClassSectionResult != nil && r.StudentClassSectionResult.Set && r.StudentClassSectionResult.Value != nil {
			return errors.New("result harus dikosongkan jika status bukan 'completed'")
		}
		if r.StudentClassSectionCompletedAt != nil && r.StudentClassSectionCompletedAt.Set && r.StudentClassSectionCompletedAt.Value != nil {
			return errors.New("completed_at harus dikosongkan jika status bukan 'completed'")
		}
	}

	// validasi tanggal unassigned >= assigned jika keduanya di-set
	if r.StudentClassSectionAssignedAt != nil && r.StudentClassSectionAssignedAt.Set &&
		r.StudentClassSectionAssignedAt.Value != nil &&
		r.StudentClassSectionUnassignedAt != nil && r.StudentClassSectionUnassignedAt.Set &&
		r.StudentClassSectionUnassignedAt.Value != nil {

		if r.StudentClassSectionUnassignedAt.Value.Before(*r.StudentClassSectionAssignedAt.Value) {
			return errors.New("unassigned_at tidak boleh lebih awal dari assigned_at")
		}
	}

	// Kolom assigned_at di DB NOT NULL → jangan izinkan clear (nil) eksplisit
	if r.StudentClassSectionAssignedAt != nil && r.StudentClassSectionAssignedAt.Set && r.StudentClassSectionAssignedAt.Value == nil {
		return errors.New("assigned_at tidak boleh dikosongkan (NOT NULL)")
	}

	return nil
}

// Terapkan perubahan ke model (hanya yang Set=true)
func (r *StudentClassSectionPatchReq) Apply(m *model.StudentClassSection) {
	if r.StudentClassSectionStatus != nil && r.StudentClassSectionStatus.Set {
		m.StudentClassSectionStatus = model.StudentClassSectionStatus(r.StudentClassSectionStatus.Value)
	}
	if r.StudentClassSectionResult != nil && r.StudentClassSectionResult.Set {
		if r.StudentClassSectionResult.Value != nil {
			res := model.StudentClassSectionResult(*r.StudentClassSectionResult.Value)
			m.StudentClassSectionResult = &res
		} else {
			m.StudentClassSectionResult = nil
		}
	}
	if r.StudentClassSectionFee != nil && r.StudentClassSectionFee.Set {
		if r.StudentClassSectionFee.Value != nil {
			m.StudentClassSectionFeeSnapshot = *r.StudentClassSectionFee.Value
		} else {
			// nullable → jika ingin clear, set ke NULL (bukan "{}")
			m.StudentClassSectionFeeSnapshot = nil
		}
	}

	// snapshots
	if r.StudentClassSectionUserProfileNameSnapshot != nil && r.StudentClassSectionUserProfileNameSnapshot.Set {
		m.StudentClassSectionUserProfileNameSnapshot = r.StudentClassSectionUserProfileNameSnapshot.Value
	}
	if r.StudentClassSectionUserProfileAvatarURLSnapshot != nil && r.StudentClassSectionUserProfileAvatarURLSnapshot.Set {
		m.StudentClassSectionUserProfileAvatarURLSnapshot = r.StudentClassSectionUserProfileAvatarURLSnapshot.Value
	}
	if r.StudentClassSectionUserProfileWhatsappURLSnapshot != nil && r.StudentClassSectionUserProfileWhatsappURLSnapshot.Set {
		m.StudentClassSectionUserProfileWhatsappURLSnapshot = r.StudentClassSectionUserProfileWhatsappURLSnapshot.Value
	}
	if r.StudentClassSectionUserProfileParentNameSnapshot != nil && r.StudentClassSectionUserProfileParentNameSnapshot.Set {
		m.StudentClassSectionUserProfileParentNameSnapshot = r.StudentClassSectionUserProfileParentNameSnapshot.Value
	}
	if r.StudentClassSectionUserProfileParentWhatsappURLSnapshot != nil && r.StudentClassSectionUserProfileParentWhatsappURLSnapshot.Set {
		m.StudentClassSectionUserProfileParentWhatsappURLSnapshot = r.StudentClassSectionUserProfileParentWhatsappURLSnapshot.Value
	}

	// waktu
	if r.StudentClassSectionAssignedAt != nil && r.StudentClassSectionAssignedAt.Set && r.StudentClassSectionAssignedAt.Value != nil {
		m.StudentClassSectionAssignedAt = *r.StudentClassSectionAssignedAt.Value
	}
	if r.StudentClassSectionUnassignedAt != nil && r.StudentClassSectionUnassignedAt.Set {
		m.StudentClassSectionUnassignedAt = r.StudentClassSectionUnassignedAt.Value
	}
	if r.StudentClassSectionCompletedAt != nil && r.StudentClassSectionCompletedAt.Set {
		m.StudentClassSectionCompletedAt = r.StudentClassSectionCompletedAt.Value
	}
}

/* =========================================================
   RESPONSE DTO
========================================================= */

type StudentClassSectionResp struct {
	StudentClassSectionID uuid.UUID `json:"student_class_section_id"`

	StudentClassSectionMasjidStudentID uuid.UUID `json:"student_class_section_masjid_student_id"`
	StudentClassSectionSectionID       uuid.UUID `json:"student_class_section_section_id"`
	StudentClassSectionMasjidID        uuid.UUID `json:"student_class_section_masjid_id"`

	StudentClassSectionStatus string          `json:"student_class_section_status"`
	StudentClassSectionResult *string         `json:"student_class_section_result,omitempty"`
	StudentClassSectionFee    *datatypes.JSON `json:"student_class_section_fee_snapshot,omitempty"`

	// Snapshot users_profile
	StudentClassSectionUserProfileNameSnapshot              *string `json:"student_class_section_user_profile_name_snapshot,omitempty"`
	StudentClassSectionUserProfileAvatarURLSnapshot         *string `json:"student_class_section_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionUserProfileWhatsappURLSnapshot       *string `json:"student_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileParentNameSnapshot        *string `json:"student_class_section_user_profile_parent_name_snapshot,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLSnapshot *string `json:"student_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`

	StudentClassSectionAssignedAt   time.Time  `json:"student_class_section_assigned_at"`
	StudentClassSectionUnassignedAt *time.Time `json:"student_class_section_unassigned_at,omitempty"`
	StudentClassSectionCompletedAt  *time.Time `json:"student_class_section_completed_at,omitempty"`

	StudentClassSectionCreatedAt time.Time  `json:"student_class_section_created_at"`
	StudentClassSectionUpdatedAt time.Time  `json:"student_class_section_updated_at"`
	StudentClassSectionDeletedAt *time.Time `json:"student_class_section_deleted_at,omitempty"`
}

func FromModel(m *model.StudentClassSection) StudentClassSectionResp {
	var res *string
	if m.StudentClassSectionResult != nil {
		v := string(*m.StudentClassSectionResult)
		res = &v
	}
	var delAt *time.Time
	if m.StudentClassSectionDeletedAt.Valid {
		t := m.StudentClassSectionDeletedAt.Time
		delAt = &t
	}
	var feePtr *datatypes.JSON
	if m.StudentClassSectionFeeSnapshot != nil {
		tmp := m.StudentClassSectionFeeSnapshot
		feePtr = &tmp
	}

	return StudentClassSectionResp{
		StudentClassSectionID: m.StudentClassSectionID,

		StudentClassSectionMasjidStudentID: m.StudentClassSectionMasjidStudentID,
		StudentClassSectionSectionID:       m.StudentClassSectionSectionID,
		StudentClassSectionMasjidID:        m.StudentClassSectionMasjidID,

		StudentClassSectionStatus: string(m.StudentClassSectionStatus),
		StudentClassSectionResult: res,
		StudentClassSectionFee:    feePtr,

		// snapshots
		StudentClassSectionUserProfileNameSnapshot:              m.StudentClassSectionUserProfileNameSnapshot,
		StudentClassSectionUserProfileAvatarURLSnapshot:         m.StudentClassSectionUserProfileAvatarURLSnapshot,
		StudentClassSectionUserProfileWhatsappURLSnapshot:       m.StudentClassSectionUserProfileWhatsappURLSnapshot,
		StudentClassSectionUserProfileParentNameSnapshot:        m.StudentClassSectionUserProfileParentNameSnapshot,
		StudentClassSectionUserProfileParentWhatsappURLSnapshot: m.StudentClassSectionUserProfileParentWhatsappURLSnapshot,

		StudentClassSectionAssignedAt:   m.StudentClassSectionAssignedAt,
		StudentClassSectionUnassignedAt: m.StudentClassSectionUnassignedAt,
		StudentClassSectionCompletedAt:  m.StudentClassSectionCompletedAt,

		StudentClassSectionCreatedAt: m.StudentClassSectionCreatedAt,
		StudentClassSectionUpdatedAt: m.StudentClassSectionUpdatedAt,
		StudentClassSectionDeletedAt: delAt,
	}
}
