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
	StudentClassSectionSectionSlugCache *string `json:"student_class_section_section_slug_cache,omitempty"`

	// NEW: cache NIS / kode siswa
	StudentClassSectionStudentCodeCache *string `json:"student_class_section_student_code_cache,omitempty"`

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

	// Cache users_profile (opsional; dibekukan saat enrol)
	StudentClassSectionUserProfileNameCache              *string `json:"student_class_section_user_profile_name_cache,omitempty"`
	StudentClassSectionUserProfileAvatarURLCache         *string `json:"student_class_section_user_profile_avatar_url_cache,omitempty"`
	StudentClassSectionUserProfileWhatsappURLCache       *string `json:"student_class_section_user_profile_whatsapp_url_cache,omitempty"`
	StudentClassSectionUserProfileParentNameCache        *string `json:"student_class_section_user_profile_parent_name_cache,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLCache *string `json:"student_class_section_user_profile_parent_whatsapp_url_cache,omitempty"`
	StudentClassSectionUserProfileGenderCache            *string `json:"student_class_section_user_profile_gender_cache,omitempty"` // NEW

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

	// trim caches
	r.StudentClassSectionSectionSlugCache = trimPtr(r.StudentClassSectionSectionSlugCache)
	r.StudentClassSectionStudentCodeCache = trimPtr(r.StudentClassSectionStudentCodeCache) // NEW

	r.StudentClassSectionUserProfileNameCache = trimPtr(r.StudentClassSectionUserProfileNameCache)
	r.StudentClassSectionUserProfileAvatarURLCache = trimPtr(r.StudentClassSectionUserProfileAvatarURLCache)
	r.StudentClassSectionUserProfileWhatsappURLCache = trimPtr(r.StudentClassSectionUserProfileWhatsappURLCache)
	r.StudentClassSectionUserProfileParentNameCache = trimPtr(r.StudentClassSectionUserProfileParentNameCache)
	r.StudentClassSectionUserProfileParentWhatsappURLCache = trimPtr(r.StudentClassSectionUserProfileParentWhatsappURLCache)
	r.StudentClassSectionUserProfileGenderCache = trimPtr(r.StudentClassSectionUserProfileGenderCache) // NEW

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

	// Cache slug section (bila disediakan); kalau nil, service layer wajib mengisi sebelum INSERT
	if r.StudentClassSectionSectionSlugCache != nil {
		m.StudentClassSectionSectionSlugCache = *r.StudentClassSectionSectionSlugCache
	}

	// NEW: student code cache
	if r.StudentClassSectionStudentCodeCache != nil {
		m.StudentClassSectionStudentCodeCache = r.StudentClassSectionStudentCodeCache
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

	// caches user
	m.StudentClassSectionUserProfileNameCache = r.StudentClassSectionUserProfileNameCache
	m.StudentClassSectionUserProfileAvatarURLCache = r.StudentClassSectionUserProfileAvatarURLCache
	m.StudentClassSectionUserProfileWhatsappURLCache = r.StudentClassSectionUserProfileWhatsappURLCache
	m.StudentClassSectionUserProfileParentNameCache = r.StudentClassSectionUserProfileParentNameCache
	m.StudentClassSectionUserProfileParentWhatsappURLCache = r.StudentClassSectionUserProfileParentWhatsappURLCache
	m.StudentClassSectionUserProfileGenderCache = r.StudentClassSectionUserProfileGenderCache // NEW

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

	// Cache users_profile
	StudentClassSectionUserProfileNameCache              *PatchField[*string] `json:"student_class_section_user_profile_name_cache,omitempty"`
	StudentClassSectionUserProfileAvatarURLCache         *PatchField[*string] `json:"student_class_section_user_profile_avatar_url_cache,omitempty"`
	StudentClassSectionUserProfileWhatsappURLCache       *PatchField[*string] `json:"student_class_section_user_profile_whatsapp_url_cache,omitempty"`
	StudentClassSectionUserProfileParentNameCache        *PatchField[*string] `json:"student_class_section_user_profile_parent_name_cache,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLCache *PatchField[*string] `json:"student_class_section_user_profile_parent_whatsapp_url_cache,omitempty"`
	StudentClassSectionUserProfileGenderCache            *PatchField[*string] `json:"student_class_section_user_profile_gender_cache,omitempty"` // NEW

	// NEW: student code cache
	StudentClassSectionStudentCodeCache *PatchField[*string] `json:"student_class_section_student_code_cache,omitempty"`

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

	// trim cache fields
	if r.StudentClassSectionUserProfileNameCache != nil && r.StudentClassSectionUserProfileNameCache.Set {
		r.StudentClassSectionUserProfileNameCache.Value = trimPtr(r.StudentClassSectionUserProfileNameCache.Value)
	}
	if r.StudentClassSectionUserProfileAvatarURLCache != nil && r.StudentClassSectionUserProfileAvatarURLCache.Set {
		r.StudentClassSectionUserProfileAvatarURLCache.Value = trimPtr(r.StudentClassSectionUserProfileAvatarURLCache.Value)
	}
	if r.StudentClassSectionUserProfileWhatsappURLCache != nil && r.StudentClassSectionUserProfileWhatsappURLCache.Set {
		r.StudentClassSectionUserProfileWhatsappURLCache.Value = trimPtr(r.StudentClassSectionUserProfileWhatsappURLCache.Value)
	}
	if r.StudentClassSectionUserProfileParentNameCache != nil && r.StudentClassSectionUserProfileParentNameCache.Set {
		r.StudentClassSectionUserProfileParentNameCache.Value = trimPtr(r.StudentClassSectionUserProfileParentNameCache.Value)
	}
	if r.StudentClassSectionUserProfileParentWhatsappURLCache != nil && r.StudentClassSectionUserProfileParentWhatsappURLCache.Set {
		r.StudentClassSectionUserProfileParentWhatsappURLCache.Value = trimPtr(r.StudentClassSectionUserProfileParentWhatsappURLCache.Value)
	}
	if r.StudentClassSectionUserProfileGenderCache != nil && r.StudentClassSectionUserProfileGenderCache.Set {
		r.StudentClassSectionUserProfileGenderCache.Value = trimPtr(r.StudentClassSectionUserProfileGenderCache.Value)
	}

	if r.StudentClassSectionStudentCodeCache != nil && r.StudentClassSectionStudentCodeCache.Set { // NEW
		r.StudentClassSectionStudentCodeCache.Value = trimPtr(r.StudentClassSectionStudentCodeCache.Value)
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

	// caches
	if r.StudentClassSectionUserProfileNameCache != nil && r.StudentClassSectionUserProfileNameCache.Set {
		m.StudentClassSectionUserProfileNameCache = r.StudentClassSectionUserProfileNameCache.Value
	}
	if r.StudentClassSectionUserProfileAvatarURLCache != nil && r.StudentClassSectionUserProfileAvatarURLCache.Set {
		m.StudentClassSectionUserProfileAvatarURLCache = r.StudentClassSectionUserProfileAvatarURLCache.Value
	}
	if r.StudentClassSectionUserProfileWhatsappURLCache != nil && r.StudentClassSectionUserProfileWhatsappURLCache.Set {
		m.StudentClassSectionUserProfileWhatsappURLCache = r.StudentClassSectionUserProfileWhatsappURLCache.Value
	}
	if r.StudentClassSectionUserProfileParentNameCache != nil && r.StudentClassSectionUserProfileParentNameCache.Set {
		m.StudentClassSectionUserProfileParentNameCache = r.StudentClassSectionUserProfileParentNameCache.Value
	}
	if r.StudentClassSectionUserProfileParentWhatsappURLCache != nil && r.StudentClassSectionUserProfileParentWhatsappURLCache.Set {
		m.StudentClassSectionUserProfileParentWhatsappURLCache = r.StudentClassSectionUserProfileParentWhatsappURLCache.Value
	}
	if r.StudentClassSectionUserProfileGenderCache != nil && r.StudentClassSectionUserProfileGenderCache.Set {
		m.StudentClassSectionUserProfileGenderCache = r.StudentClassSectionUserProfileGenderCache.Value
	}
	if r.StudentClassSectionStudentCodeCache != nil && r.StudentClassSectionStudentCodeCache.Set { // NEW
		m.StudentClassSectionStudentCodeCache = r.StudentClassSectionStudentCodeCache.Value
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

	StudentClassSectionSectionSlugCache string  `json:"student_class_section_section_slug_cache"`
	StudentClassSectionStudentCodeCache *string `json:"student_class_section_student_code_cache,omitempty"` // NEW

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

	// Cache users_profile
	StudentClassSectionUserProfileNameCache              *string `json:"student_class_section_user_profile_name_cache,omitempty"`
	StudentClassSectionUserProfileAvatarURLCache         *string `json:"student_class_section_user_profile_avatar_url_cache,omitempty"`
	StudentClassSectionUserProfileWhatsappURLCache       *string `json:"student_class_section_user_profile_whatsapp_url_cache,omitempty"`
	StudentClassSectionUserProfileParentNameCache        *string `json:"student_class_section_user_profile_parent_name_cache,omitempty"`
	StudentClassSectionUserProfileParentWhatsappURLCache *string `json:"student_class_section_user_profile_parent_whatsapp_url_cache,omitempty"`
	StudentClassSectionUserProfileGenderCache            *string `json:"student_class_section_user_profile_gender_cache,omitempty"` // NEW

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

		StudentClassSectionSectionSlugCache: m.StudentClassSectionSectionSlugCache,
		StudentClassSectionStudentCodeCache: m.StudentClassSectionStudentCodeCache, // NEW

		StudentClassSectionStatus: string(m.StudentClassSectionStatus),
		StudentClassSectionResult: res,

		StudentClassSectionFinalScore:        m.StudentClassSectionFinalScore,
		StudentClassSectionFinalGradeLetter:  m.StudentClassSectionFinalGradeLetter,
		StudentClassSectionFinalGradePoint:   m.StudentClassSectionFinalGradePoint,
		StudentClassSectionFinalRank:         m.StudentClassSectionFinalRank,
		StudentClassSectionFinalRemarks:      m.StudentClassSectionFinalRemarks,
		StudentClassSectionGradedByTeacherID: m.StudentClassSectionGradedByTeacherID,
		StudentClassSectionGradedAt:          m.StudentClassSectionGradedAt,

		StudentClassSectionUserProfileNameCache:              m.StudentClassSectionUserProfileNameCache,
		StudentClassSectionUserProfileAvatarURLCache:         m.StudentClassSectionUserProfileAvatarURLCache,
		StudentClassSectionUserProfileWhatsappURLCache:       m.StudentClassSectionUserProfileWhatsappURLCache,
		StudentClassSectionUserProfileParentNameCache:        m.StudentClassSectionUserProfileParentNameCache,
		StudentClassSectionUserProfileParentWhatsappURLCache: m.StudentClassSectionUserProfileParentWhatsappURLCache,
		StudentClassSectionUserProfileGenderCache:            m.StudentClassSectionUserProfileGenderCache,

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
