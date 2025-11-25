// file: internals/features/school/classes/class_sections/dto/student_class_section_dto.go
package dto

import (
	"errors"
	"strings"
	"time"

	model "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	"github.com/google/uuid"
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
	StudentClassSectionSchoolStudentID uuid.UUID `json:"student_class_section_school_student_id"`
	StudentClassSectionSectionID       uuid.UUID `json:"student_class_section_section_id"`
	StudentClassSectionSchoolID        uuid.UUID `json:"student_class_section_school_id"`

	// biasanya diisi service dari class_sections; tetap disediakan optional kalau mau override
	StudentClassSectionSectionSlugSnapshot *string `json:"student_class_section_section_slug_snapshot,omitempty"`

	// NEW: snapshot NIS / kode siswa
	StudentClassSectionStudentCodeSnapshot *string `json:"student_class_section_student_code_snapshot,omitempty"`

	StudentClassSectionStatus string  `json:"student_class_section_status,omitempty"` // default: active
	StudentClassSectionResult *string `json:"student_class_section_result,omitempty"`

	// ==========================
	// NILAI AKHIR (optional saat create; WAJIB kalau status=completed)
	// ==========================
	StudentClassSectionFinalScore        *float64   `json:"student_class_section_final_score,omitempty"`
	StudentClassSectionFinalGradeLetter  *string    `json:"student_class_section_final_grade_letter,omitempty"`
	StudentClassSectionFinalGradePoint   *float64   `json:"student_class_section_final_grade_point,omitempty"`
	StudentClassSectionFinalRank         *int       `json:"student_class_section_final_rank,omitempty"`
	StudentClassSectionFinalRemarks      *string    `json:"student_class_section_final_remarks,omitempty"`
	StudentClassSectionGradedByTeacherID *uuid.UUID `json:"student_class_section_graded_by_teacher_id,omitempty"`
	StudentClassSectionGradedAt          *time.Time `json:"student_class_section_graded_at,omitempty"`

	// Snapshot users_profile (opsional; dibekukan saat enrol)
	StudentClassSectionUserProfileNameSnapshot              *string `json:"student_class_section_user_profile_name_snapshot,omitempty"`
	StudentClassSectionUserProfileAvatarURLSnapshot         *string `json:"student_class_section_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionUserProfileWhatsappURLSnapshot       *string `json:"student_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileParentNameSnapshot        *string `json:"student_class_section_user_profile_parent_name_snapshot,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLSnapshot *string `json:"student_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileGenderSnapshot            *string `json:"student_class_section_user_profile_gender_snapshot,omitempty"` // NEW

	StudentClassSectionAssignedAt   *time.Time `json:"student_class_section_assigned_at,omitempty"`
	StudentClassSectionUnassignedAt *time.Time `json:"student_class_section_unassigned_at,omitempty"`
	StudentClassSectionCompletedAt  *time.Time `json:"student_class_section_completed_at,omitempty"`

	// Catatan dari siswa
	StudentClassSectionStudentNotes          *string    `json:"student_class_section_student_notes,omitempty"`
	StudentClassSectionStudentNotesUpdatedAt *time.Time `json:"student_class_section_student_notes_updated_at,omitempty"`

	// Catatan dari wali kelas
	StudentClassSectionHomeroomNotes          *string    `json:"student_class_section_homeroom_notes,omitempty"`
	StudentClassSectionHomeroomNotesUpdatedAt *time.Time `json:"student_class_section_homeroom_notes_updated_at,omitempty"`
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

	// trim snapshots
	r.StudentClassSectionSectionSlugSnapshot = trimPtr(r.StudentClassSectionSectionSlugSnapshot)
	r.StudentClassSectionStudentCodeSnapshot = trimPtr(r.StudentClassSectionStudentCodeSnapshot) // NEW

	r.StudentClassSectionUserProfileNameSnapshot = trimPtr(r.StudentClassSectionUserProfileNameSnapshot)
	r.StudentClassSectionUserProfileAvatarURLSnapshot = trimPtr(r.StudentClassSectionUserProfileAvatarURLSnapshot)
	r.StudentClassSectionUserProfileWhatsappURLSnapshot = trimPtr(r.StudentClassSectionUserProfileWhatsappURLSnapshot)
	r.StudentClassSectionUserProfileParentNameSnapshot = trimPtr(r.StudentClassSectionUserProfileParentNameSnapshot)
	r.StudentClassSectionUserProfileParentWhatsappURLSnapshot = trimPtr(r.StudentClassSectionUserProfileParentWhatsappURLSnapshot)
	r.StudentClassSectionUserProfileGenderSnapshot = trimPtr(r.StudentClassSectionUserProfileGenderSnapshot) // NEW

	// trim notes (optional, biar rapi)
	r.StudentClassSectionStudentNotes = trimPtr(r.StudentClassSectionStudentNotes)
	r.StudentClassSectionHomeroomNotes = trimPtr(r.StudentClassSectionHomeroomNotes)
}

func (r *StudentClassSectionCreateReq) Validate() error {
	// UUID wajib
	if isZeroUUID(r.StudentClassSectionSchoolStudentID) {
		return errors.New("student_class_section_school_student_id wajib diisi")
	}
	if isZeroUUID(r.StudentClassSectionSectionID) {
		return errors.New("student_class_section_section_id wajib diisi")
	}
	if isZeroUUID(r.StudentClassSectionSchoolID) {
		return errors.New("student_class_section_school_id wajib diisi")
	}
	// status/result
	if !isValidStatus(r.StudentClassSectionStatus) {
		return errors.New("invalid student_class_section_status")
	}
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
		// minimal salah satu metrik nilai terisi
		if r.StudentClassSectionFinalScore == nil &&
			r.StudentClassSectionFinalGradeLetter == nil &&
			r.StudentClassSectionFinalGradePoint == nil {
			return errors.New("minimal salah satu dari final_score/final_grade_letter/final_grade_point wajib diisi saat completed")
		}
	} else {
		// jika bukan completed, larang isi fields nilai
		if r.StudentClassSectionResult != nil ||
			r.StudentClassSectionCompletedAt != nil ||
			r.StudentClassSectionFinalScore != nil ||
			r.StudentClassSectionFinalGradeLetter != nil ||
			r.StudentClassSectionFinalGradePoint != nil ||
			r.StudentClassSectionFinalRank != nil ||
			r.StudentClassSectionFinalRemarks != nil ||
			r.StudentClassSectionGradedAt != nil {
			return errors.New("field nilai akhir & completed_at harus kosong jika status bukan 'completed'")
		}
	}

	// validasi range nilai (jika diisi)
	if r.StudentClassSectionFinalScore != nil {
		if *r.StudentClassSectionFinalScore < 0 || *r.StudentClassSectionFinalScore > 100 {
			return errors.New("final_score harus di antara 0..100")
		}
	}
	if r.StudentClassSectionFinalGradePoint != nil {
		if *r.StudentClassSectionFinalGradePoint < 0 || *r.StudentClassSectionFinalGradePoint > 4 {
			return errors.New("final_grade_point harus di antara 0..4")
		}
	}
	if r.StudentClassSectionFinalRank != nil && *r.StudentClassSectionFinalRank <= 0 {
		return errors.New("final_rank harus > 0")
	}

	// tanggal
	if r.StudentClassSectionAssignedAt != nil && r.StudentClassSectionUnassignedAt != nil &&
		r.StudentClassSectionUnassignedAt.Before(*r.StudentClassSectionAssignedAt) {
		return errors.New("student_class_section_unassigned_at tidak boleh lebih awal dari student_class_section_assigned_at")
	}
	return nil
}

func (r *StudentClassSectionCreateReq) ToModel() *model.StudentClassSection {
	m := &model.StudentClassSection{
		StudentClassSectionSchoolStudentID: r.StudentClassSectionSchoolStudentID,
		StudentClassSectionSectionID:       r.StudentClassSectionSectionID,
		StudentClassSectionSchoolID:        r.StudentClassSectionSchoolID,
		StudentClassSectionStatus:          model.StudentClassSectionStatus(r.StudentClassSectionStatus),
	}

	// Snapshot slug section (bila disediakan); kalau nil, service layer wajib mengisi sebelum INSERT
	if r.StudentClassSectionSectionSlugSnapshot != nil {
		m.StudentClassSectionSectionSlugSnapshot = *r.StudentClassSectionSectionSlugSnapshot
	}

	// NEW: student code snapshot
	if r.StudentClassSectionStudentCodeSnapshot != nil {
		m.StudentClassSectionStudentCodeSnapshot = r.StudentClassSectionStudentCodeSnapshot
	}

	// result
	if r.StudentClassSectionResult != nil {
		res := model.StudentClassSectionResult(*r.StudentClassSectionResult)
		m.StudentClassSectionResult = &res
	}

	// nilai akhir
	m.StudentClassSectionFinalScore = r.StudentClassSectionFinalScore
	m.StudentClassSectionFinalGradeLetter = r.StudentClassSectionFinalGradeLetter
	m.StudentClassSectionFinalGradePoint = r.StudentClassSectionFinalGradePoint
	m.StudentClassSectionFinalRank = r.StudentClassSectionFinalRank
	m.StudentClassSectionFinalRemarks = r.StudentClassSectionFinalRemarks
	m.StudentClassSectionGradedByTeacherID = r.StudentClassSectionGradedByTeacherID
	m.StudentClassSectionGradedAt = r.StudentClassSectionGradedAt

	// snapshots user
	m.StudentClassSectionUserProfileNameSnapshot = r.StudentClassSectionUserProfileNameSnapshot
	m.StudentClassSectionUserProfileAvatarURLSnapshot = r.StudentClassSectionUserProfileAvatarURLSnapshot
	m.StudentClassSectionUserProfileWhatsappURLSnapshot = r.StudentClassSectionUserProfileWhatsappURLSnapshot
	m.StudentClassSectionUserProfileParentNameSnapshot = r.StudentClassSectionUserProfileParentNameSnapshot
	m.StudentClassSectionUserProfileParentWhatsappURLSnapshot = r.StudentClassSectionUserProfileParentWhatsappURLSnapshot
	m.StudentClassSectionUserProfileGenderSnapshot = r.StudentClassSectionUserProfileGenderSnapshot // NEW

	// waktu
	if r.StudentClassSectionAssignedAt != nil {
		m.StudentClassSectionAssignedAt = *r.StudentClassSectionAssignedAt
	}
	m.StudentClassSectionUnassignedAt = r.StudentClassSectionUnassignedAt
	m.StudentClassSectionCompletedAt = r.StudentClassSectionCompletedAt

	// notes
	m.StudentClassSectionStudentNotes = r.StudentClassSectionStudentNotes
	m.StudentClassSectionStudentNotesUpdatedAt = r.StudentClassSectionStudentNotesUpdatedAt
	m.StudentClassSectionHomeroomNotes = r.StudentClassSectionHomeroomNotes
	m.StudentClassSectionHomeroomNotesUpdatedAt = r.StudentClassSectionHomeroomNotesUpdatedAt

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
	StudentClassSectionStatus *PatchField[string]  `json:"student_class_section_status,omitempty"`
	StudentClassSectionResult *PatchField[*string] `json:"student_class_section_result,omitempty"`

	// nilai akhir (patchable)
	StudentClassSectionFinalScore        *PatchField[*float64]   `json:"student_class_section_final_score,omitempty"`
	StudentClassSectionFinalGradeLetter  *PatchField[*string]    `json:"student_class_section_final_grade_letter,omitempty"`
	StudentClassSectionFinalGradePoint   *PatchField[*float64]   `json:"student_class_section_final_grade_point,omitempty"`
	StudentClassSectionFinalRank         *PatchField[*int]       `json:"student_class_section_final_rank,omitempty"`
	StudentClassSectionFinalRemarks      *PatchField[*string]    `json:"student_class_section_final_remarks,omitempty"`
	StudentClassSectionGradedByTeacherID *PatchField[*uuid.UUID] `json:"student_class_section_graded_by_teacher_id,omitempty"`
	StudentClassSectionGradedAt          *PatchField[*time.Time] `json:"student_class_section_graded_at,omitempty"`

	// Snapshot users_profile
	StudentClassSectionUserProfileNameSnapshot              *PatchField[*string] `json:"student_class_section_user_profile_name_snapshot,omitempty"`
	StudentClassSectionUserProfileAvatarURLSnapshot         *PatchField[*string] `json:"student_class_section_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionUserProfileWhatsappURLSnapshot       *PatchField[*string] `json:"student_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileParentNameSnapshot        *PatchField[*string] `json:"student_class_section_user_profile_parent_name_snapshot,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLSnapshot *PatchField[*string] `json:"student_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileGenderSnapshot            *PatchField[*string] `json:"student_class_section_user_profile_gender_snapshot,omitempty"` // NEW

	// NEW: student code snapshot
	StudentClassSectionStudentCodeSnapshot *PatchField[*string] `json:"student_class_section_student_code_snapshot,omitempty"`

	StudentClassSectionAssignedAt   *PatchField[*time.Time] `json:"student_class_section_assigned_at,omitempty"`
	StudentClassSectionUnassignedAt *PatchField[*time.Time] `json:"student_class_section_unassigned_at,omitempty"`
	StudentClassSectionCompletedAt  *PatchField[*time.Time] `json:"student_class_section_completed_at,omitempty"`

	// Catatan siswa/guru (patchable)
	StudentClassSectionStudentNotes          *PatchField[*string]    `json:"student_class_section_student_notes,omitempty"`
	StudentClassSectionStudentNotesUpdatedAt *PatchField[*time.Time] `json:"student_class_section_student_notes_updated_at,omitempty"`

	StudentClassSectionHomeroomNotes          *PatchField[*string]    `json:"student_class_section_homeroom_notes,omitempty"`
	StudentClassSectionHomeroomNotesUpdatedAt *PatchField[*time.Time] `json:"student_class_section_homeroom_notes_updated_at,omitempty"`
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

	// trim snapshot fields
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
	if r.StudentClassSectionUserProfileGenderSnapshot != nil && r.StudentClassSectionUserProfileGenderSnapshot.Set {
		r.StudentClassSectionUserProfileGenderSnapshot.Value = trimPtr(r.StudentClassSectionUserProfileGenderSnapshot.Value)
	}

	if r.StudentClassSectionStudentCodeSnapshot != nil && r.StudentClassSectionStudentCodeSnapshot.Set { // NEW
		r.StudentClassSectionStudentCodeSnapshot.Value = trimPtr(r.StudentClassSectionStudentCodeSnapshot.Value)
	}

	// trim notes
	if r.StudentClassSectionStudentNotes != nil && r.StudentClassSectionStudentNotes.Set {
		r.StudentClassSectionStudentNotes.Value = trimPtr(r.StudentClassSectionStudentNotes.Value)
	}
	if r.StudentClassSectionHomeroomNotes != nil && r.StudentClassSectionHomeroomNotes.Set {
		r.StudentClassSectionHomeroomNotes.Value = trimPtr(r.StudentClassSectionHomeroomNotes.Value)
	}
}

func (r *StudentClassSectionPatchReq) Validate() error {
	// status valid?
	if r.StudentClassSectionStatus != nil && r.StudentClassSectionStatus.Set {
		if !isValidStatus(r.StudentClassSectionStatus.Value) {
			return errors.New("invalid status")
		}
	}

	// result valid?
	if r.StudentClassSectionResult != nil && r.StudentClassSectionResult.Set && r.StudentClassSectionResult.Value != nil {
		if !isValidResult(*r.StudentClassSectionResult.Value) {
			return errors.New("invalid result (gunakan 'passed' atau 'failed')")
		}
	}

	// jika status di-set menjadi completed → completed_at & result wajib,
	// dan minimal salah satu metrik nilai harus diset (non-nil)
	if r.StudentClassSectionStatus != nil && r.StudentClassSectionStatus.Set &&
		r.StudentClassSectionStatus.Value == string(model.StudentClassSectionCompleted) {

		if r.StudentClassSectionCompletedAt == nil || !r.StudentClassSectionCompletedAt.Set || r.StudentClassSectionCompletedAt.Value == nil {
			return errors.New("completed_at wajib diisi jika status completed")
		}
		if r.StudentClassSectionResult == nil || !r.StudentClassSectionResult.Set || r.StudentClassSectionResult.Value == nil {
			return errors.New("result wajib diisi jika status completed")
		}

		hasAnyGrade :=
			(r.StudentClassSectionFinalScore != nil && r.StudentClassSectionFinalScore.Set && r.StudentClassSectionFinalScore.Value != nil) ||
				(r.StudentClassSectionFinalGradeLetter != nil && r.StudentClassSectionFinalGradeLetter.Set && r.StudentClassSectionFinalGradeLetter.Value != nil) ||
				(r.StudentClassSectionFinalGradePoint != nil && r.StudentClassSectionFinalGradePoint.Set && r.StudentClassSectionFinalGradePoint.Value != nil)
		if !hasAnyGrade {
			return errors.New("minimal salah satu dari final_score/final_grade_letter/final_grade_point harus diisi saat completed")
		}
	}

	// jika status di-set menjadi non-completed → larang isi result/completed_at atau kolom nilai
	if r.StudentClassSectionStatus != nil && r.StudentClassSectionStatus.Set &&
		r.StudentClassSectionStatus.Value != string(model.StudentClassSectionCompleted) {

		if r.StudentClassSectionResult != nil && r.StudentClassSectionResult.Set && r.StudentClassSectionResult.Value != nil {
			return errors.New("result harus dikosongkan jika status bukan 'completed'")
		}
		if r.StudentClassSectionCompletedAt != nil && r.StudentClassSectionCompletedAt.Set && r.StudentClassSectionCompletedAt.Value != nil {
			return errors.New("completed_at harus dikosongkan jika status bukan 'completed'")
		}
		if (r.StudentClassSectionFinalScore != nil && r.StudentClassSectionFinalScore.Set && r.StudentClassSectionFinalScore.Value != nil) ||
			(r.StudentClassSectionFinalGradeLetter != nil && r.StudentClassSectionFinalGradeLetter.Set && r.StudentClassSectionFinalGradeLetter.Value != nil) ||
			(r.StudentClassSectionFinalGradePoint != nil && r.StudentClassSectionFinalGradePoint.Set && r.StudentClassSectionFinalGradePoint.Value != nil) ||
			(r.StudentClassSectionFinalRank != nil && r.StudentClassSectionFinalRank.Set && r.StudentClassSectionFinalRank.Value != nil) ||
			(r.StudentClassSectionFinalRemarks != nil && r.StudentClassSectionFinalRemarks.Set && r.StudentClassSectionFinalRemarks.Value != nil) ||
			(r.StudentClassSectionGradedAt != nil && r.StudentClassSectionGradedAt.Set && r.StudentClassSectionGradedAt.Value != nil) {
			return errors.New("field nilai akhir harus dikosongkan jika status bukan 'completed'")
		}
	}

	// range nilai (kalau di-set)
	if r.StudentClassSectionFinalScore != nil && r.StudentClassSectionFinalScore.Set && r.StudentClassSectionFinalScore.Value != nil {
		v := *r.StudentClassSectionFinalScore.Value
		if v < 0 || v > 100 {
			return errors.New("final_score harus di antara 0..100")
		}
	}
	if r.StudentClassSectionFinalGradePoint != nil && r.StudentClassSectionFinalGradePoint.Set && r.StudentClassSectionFinalGradePoint.Value != nil {
		v := *r.StudentClassSectionFinalGradePoint.Value
		if v < 0 || v > 4 {
			return errors.New("final_grade_point harus di antara 0..4")
		}
	}
	if r.StudentClassSectionFinalRank != nil && r.StudentClassSectionFinalRank.Set && r.StudentClassSectionFinalRank.Value != nil {
		if *r.StudentClassSectionFinalRank.Value <= 0 {
			return errors.New("final_rank harus > 0")
		}
	}

	// tanggal
	if r.StudentClassSectionAssignedAt != nil && r.StudentClassSectionAssignedAt.Set &&
		r.StudentClassSectionAssignedAt.Value != nil &&
		r.StudentClassSectionUnassignedAt != nil && r.StudentClassSectionUnassignedAt.Set &&
		r.StudentClassSectionUnassignedAt.Value != nil {
		if r.StudentClassSectionUnassignedAt.Value.Before(*r.StudentClassSectionAssignedAt.Value) {
			return errors.New("unassigned_at tidak boleh lebih awal dari assigned_at")
		}
	}

	// assigned_at NOT NULL → jangan izinkan clear eksplisit
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

	// nilai akhir
	if r.StudentClassSectionFinalScore != nil && r.StudentClassSectionFinalScore.Set {
		m.StudentClassSectionFinalScore = r.StudentClassSectionFinalScore.Value
	}
	if r.StudentClassSectionFinalGradeLetter != nil && r.StudentClassSectionFinalGradeLetter.Set {
		m.StudentClassSectionFinalGradeLetter = r.StudentClassSectionFinalGradeLetter.Value
	}
	if r.StudentClassSectionFinalGradePoint != nil && r.StudentClassSectionFinalGradePoint.Set {
		m.StudentClassSectionFinalGradePoint = r.StudentClassSectionFinalGradePoint.Value
	}
	if r.StudentClassSectionFinalRank != nil && r.StudentClassSectionFinalRank.Set {
		m.StudentClassSectionFinalRank = r.StudentClassSectionFinalRank.Value
	}
	if r.StudentClassSectionFinalRemarks != nil && r.StudentClassSectionFinalRemarks.Set {
		m.StudentClassSectionFinalRemarks = r.StudentClassSectionFinalRemarks.Value
	}
	if r.StudentClassSectionGradedByTeacherID != nil && r.StudentClassSectionGradedByTeacherID.Set {
		m.StudentClassSectionGradedByTeacherID = r.StudentClassSectionGradedByTeacherID.Value
	}
	if r.StudentClassSectionGradedAt != nil && r.StudentClassSectionGradedAt.Set {
		m.StudentClassSectionGradedAt = r.StudentClassSectionGradedAt.Value
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
	if r.StudentClassSectionUserProfileGenderSnapshot != nil && r.StudentClassSectionUserProfileGenderSnapshot.Set {
		m.StudentClassSectionUserProfileGenderSnapshot = r.StudentClassSectionUserProfileGenderSnapshot.Value
	}
	if r.StudentClassSectionStudentCodeSnapshot != nil && r.StudentClassSectionStudentCodeSnapshot.Set { // NEW
		m.StudentClassSectionStudentCodeSnapshot = r.StudentClassSectionStudentCodeSnapshot.Value
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

	// notes
	if r.StudentClassSectionStudentNotes != nil && r.StudentClassSectionStudentNotes.Set {
		m.StudentClassSectionStudentNotes = r.StudentClassSectionStudentNotes.Value
	}
	if r.StudentClassSectionStudentNotesUpdatedAt != nil && r.StudentClassSectionStudentNotesUpdatedAt.Set {
		m.StudentClassSectionStudentNotesUpdatedAt = r.StudentClassSectionStudentNotesUpdatedAt.Value
	}
	if r.StudentClassSectionHomeroomNotes != nil && r.StudentClassSectionHomeroomNotes.Set {
		m.StudentClassSectionHomeroomNotes = r.StudentClassSectionHomeroomNotes.Value
	}
	if r.StudentClassSectionHomeroomNotesUpdatedAt != nil && r.StudentClassSectionHomeroomNotesUpdatedAt.Set {
		m.StudentClassSectionHomeroomNotesUpdatedAt = r.StudentClassSectionHomeroomNotesUpdatedAt.Value
	}
}

/* =========================================================
   RESPONSE DTO
========================================================= */

type StudentClassSectionResp struct {
	StudentClassSectionID uuid.UUID `json:"student_class_section_id"`

	StudentClassSectionSchoolStudentID uuid.UUID `json:"student_class_section_school_student_id"`
	StudentClassSectionSectionID       uuid.UUID `json:"student_class_section_section_id"`
	StudentClassSectionSchoolID        uuid.UUID `json:"student_class_section_school_id"`

	StudentClassSectionSectionSlugSnapshot string  `json:"student_class_section_section_slug_snapshot"`
	StudentClassSectionStudentCodeSnapshot *string `json:"student_class_section_student_code_snapshot,omitempty"` // NEW

	StudentClassSectionStatus string  `json:"student_class_section_status"`
	StudentClassSectionResult *string `json:"student_class_section_result,omitempty"`

	// nilai akhir
	StudentClassSectionFinalScore        *float64   `json:"student_class_section_final_score,omitempty"`
	StudentClassSectionFinalGradeLetter  *string    `json:"student_class_section_final_grade_letter,omitempty"`
	StudentClassSectionFinalGradePoint   *float64   `json:"student_class_section_final_grade_point,omitempty"`
	StudentClassSectionFinalRank         *int       `json:"student_class_section_final_rank,omitempty"`
	StudentClassSectionFinalRemarks      *string    `json:"student_class_section_final_remarks,omitempty"`
	StudentClassSectionGradedByTeacherID *uuid.UUID `json:"student_class_section_graded_by_teacher_id,omitempty"`
	StudentClassSectionGradedAt          *time.Time `json:"student_class_section_graded_at,omitempty"`

	// Snapshot users_profile
	StudentClassSectionUserProfileNameSnapshot              *string `json:"student_class_section_user_profile_name_snapshot,omitempty"`
	StudentClassSectionUserProfileAvatarURLSnapshot         *string `json:"student_class_section_user_profile_avatar_url_snapshot,omitempty"`
	StudentClassSectionUserProfileWhatsappURLSnapshot       *string `json:"student_class_section_user_profile_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileParentNameSnapshot        *string `json:"student_class_section_user_profile_parent_name_snapshot,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLSnapshot *string `json:"student_class_section_user_profile_parent_whatsapp_url_snapshot,omitempty"`
	StudentClassSectionUserProfileGenderSnapshot            *string `json:"student_class_section_user_profile_gender_snapshot,omitempty"` // NEW

	StudentClassSectionAssignedAt   time.Time  `json:"student_class_section_assigned_at"`
	StudentClassSectionUnassignedAt *time.Time `json:"student_class_section_unassigned_at,omitempty"`
	StudentClassSectionCompletedAt  *time.Time `json:"student_class_section_completed_at,omitempty"`

	// Notes
	StudentClassSectionStudentNotes           *string    `json:"student_class_section_student_notes,omitempty"`
	StudentClassSectionStudentNotesUpdatedAt  *time.Time `json:"student_class_section_student_notes_updated_at,omitempty"`
	StudentClassSectionHomeroomNotes          *string    `json:"student_class_section_homeroom_notes,omitempty"`
	StudentClassSectionHomeroomNotesUpdatedAt *time.Time `json:"student_class_section_homeroom_notes_updated_at,omitempty"`

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

	return StudentClassSectionResp{
		StudentClassSectionID: m.StudentClassSectionID,

		StudentClassSectionSchoolStudentID: m.StudentClassSectionSchoolStudentID,
		StudentClassSectionSectionID:       m.StudentClassSectionSectionID,
		StudentClassSectionSchoolID:        m.StudentClassSectionSchoolID,

		StudentClassSectionSectionSlugSnapshot: m.StudentClassSectionSectionSlugSnapshot,
		StudentClassSectionStudentCodeSnapshot: m.StudentClassSectionStudentCodeSnapshot, // NEW

		StudentClassSectionStatus: string(m.StudentClassSectionStatus),
		StudentClassSectionResult: res,

		StudentClassSectionFinalScore:        m.StudentClassSectionFinalScore,
		StudentClassSectionFinalGradeLetter:  m.StudentClassSectionFinalGradeLetter,
		StudentClassSectionFinalGradePoint:   m.StudentClassSectionFinalGradePoint,
		StudentClassSectionFinalRank:         m.StudentClassSectionFinalRank,
		StudentClassSectionFinalRemarks:      m.StudentClassSectionFinalRemarks,
		StudentClassSectionGradedByTeacherID: m.StudentClassSectionGradedByTeacherID,
		StudentClassSectionGradedAt:          m.StudentClassSectionGradedAt,

		StudentClassSectionUserProfileNameSnapshot:              m.StudentClassSectionUserProfileNameSnapshot,
		StudentClassSectionUserProfileAvatarURLSnapshot:         m.StudentClassSectionUserProfileAvatarURLSnapshot,
		StudentClassSectionUserProfileWhatsappURLSnapshot:       m.StudentClassSectionUserProfileWhatsappURLSnapshot,
		StudentClassSectionUserProfileParentNameSnapshot:        m.StudentClassSectionUserProfileParentNameSnapshot,
		StudentClassSectionUserProfileParentWhatsappURLSnapshot: m.StudentClassSectionUserProfileParentWhatsappURLSnapshot,
		StudentClassSectionUserProfileGenderSnapshot:            m.StudentClassSectionUserProfileGenderSnapshot,

		StudentClassSectionAssignedAt:   m.StudentClassSectionAssignedAt,
		StudentClassSectionUnassignedAt: m.StudentClassSectionUnassignedAt,
		StudentClassSectionCompletedAt:  m.StudentClassSectionCompletedAt,

		StudentClassSectionStudentNotes:           m.StudentClassSectionStudentNotes,
		StudentClassSectionStudentNotesUpdatedAt:  m.StudentClassSectionStudentNotesUpdatedAt,
		StudentClassSectionHomeroomNotes:          m.StudentClassSectionHomeroomNotes,
		StudentClassSectionHomeroomNotesUpdatedAt: m.StudentClassSectionHomeroomNotesUpdatedAt,

		StudentClassSectionCreatedAt: m.StudentClassSectionCreatedAt,
		StudentClassSectionUpdatedAt: m.StudentClassSectionUpdatedAt,
		StudentClassSectionDeletedAt: delAt,
	}
}
