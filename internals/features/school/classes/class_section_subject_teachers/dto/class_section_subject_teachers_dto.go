// file: internals/features/lembaga/class_section_subject_teachers/dto/csst_dto.go
package dto

import (
	"encoding/json"
	"strings"
	"time"

	teacherSnap "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/snapshot"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

/* =========================================================
   Helpers
========================================================= */

func trimLowerPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.ToLower(strings.TrimSpace(*p))
	if s == "" {
		return nil
	}
	return &s
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// decode JSONB → *TeacherSnapshot (dipakai di response)
func teacherSnapshotFromJSON(j *datatypes.JSON) *teacherSnap.TeacherSnapshot {
	if j == nil {
		return nil
	}
	raw := []byte(*j)
	if len(raw) == 0 {
		return nil
	}
	// handle literal "null"
	if strings.TrimSpace(string(raw)) == "null" {
		return nil
	}

	var ts teacherSnap.TeacherSnapshot
	if err := json.Unmarshal(raw, &ts); err != nil {
		// kalau gagal parse, jangan panik – cukup kembalikan nil
		return nil
	}
	// opsional: trim string di dalam snapshot supaya bersih
	trim := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	ts.Name = trim(ts.Name)
	ts.AvatarURL = trim(ts.AvatarURL)
	ts.WhatsappURL = trim(ts.WhatsappURL)
	ts.TitlePrefix = trim(ts.TitlePrefix)
	ts.TitleSuffix = trim(ts.TitleSuffix)
	ts.Gender = trim(ts.Gender)
	ts.TeacherNumber = trim(ts.TeacherNumber)
	ts.TeacherCode = trim(ts.TeacherCode)

	ts.ID = strings.TrimSpace(ts.ID)

	if ts.ID == "" &&
		ts.Name == nil &&
		ts.AvatarURL == nil &&
		ts.WhatsappURL == nil &&
		ts.TitlePrefix == nil &&
		ts.TitleSuffix == nil &&
		ts.Gender == nil &&
		ts.TeacherNumber == nil &&
		ts.TeacherCode == nil {
		return nil
	}

	return &ts
}

/* =========================================================
   1) REQUEST DTO (FOLLOW SQL/MODEL TERBARU)
========================================================= */

// Create
type CreateClassSectionSubjectTeacherRequest struct {
	// Biasanya diisi dari context auth pada controller
	ClassSectionSubjectTeacherSchoolID *uuid.UUID `json:"class_section_subject_teacher_school_id"  validate:"omitempty,uuid"`

	// Relasi utama
	ClassSectionSubjectTeacherClassSectionID uuid.UUID `json:"class_section_subject_teacher_class_section_id" validate:"required,uuid"`
	ClassSectionSubjectTeacherClassSubjectID uuid.UUID `json:"class_section_subject_teacher_class_subject_id" validate:"required,uuid"`

	// pakai school_teachers.school_teacher_id
	ClassSectionSubjectTeacherSchoolTeacherID uuid.UUID `json:"class_section_subject_teacher_school_teacher_id" validate:"required,uuid"`

	// ➕ Asisten (opsional)
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id" validate:"omitempty,uuid"`

	// Opsional
	ClassSectionSubjectTeacherSlug        *string    `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription *string    `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherClassRoomID *uuid.UUID `json:"class_section_subject_teacher_class_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL    *string    `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`
	ClassSectionSubjectTeacherCapacity    *int       `json:"class_section_subject_teacher_capacity" validate:"omitempty"` // >=0 divalidasi di DB (CHECK)

	// enum: offline|online|hybrid
	ClassSectionSubjectTeacherDeliveryMode *csstModel.ClassDeliveryMode `json:"class_section_subject_teacher_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	// Target pertemuan & KKM spesifik CSST (opsional)
	ClassSectionSubjectTeacherTotalMeetingsTarget *int `json:"class_section_subject_teacher_total_meetings_target" validate:"omitempty"`
	ClassSectionSubjectTeacherMinPassingScore     *int `json:"class_section_subject_teacher_min_passing_score" validate:"omitempty,gte=0"`

	// Status
	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

// Update (partial)
type UpdateClassSectionSubjectTeacherRequest struct {
	ClassSectionSubjectTeacherSchoolID        *uuid.UUID `json:"class_section_subject_teacher_school_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSectionID  *uuid.UUID `json:"class_section_subject_teacher_class_section_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherClassSubjectID  *uuid.UUID `json:"class_section_subject_teacher_class_subject_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_school_teacher_id" validate:"omitempty,uuid"`

	// ➕ Asisten (opsional)
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id" validate:"omitempty,uuid"`

	ClassSectionSubjectTeacherSlug         *string                      `json:"class_section_subject_teacher_slug" validate:"omitempty,max=160"`
	ClassSectionSubjectTeacherDescription  *string                      `json:"class_section_subject_teacher_description" validate:"omitempty"`
	ClassSectionSubjectTeacherClassRoomID  *uuid.UUID                   `json:"class_section_subject_teacher_class_room_id" validate:"omitempty,uuid"`
	ClassSectionSubjectTeacherGroupURL     *string                      `json:"class_section_subject_teacher_group_url" validate:"omitempty,max=2000"`
	ClassSectionSubjectTeacherCapacity     *int                         `json:"class_section_subject_teacher_capacity" validate:"omitempty"`
	ClassSectionSubjectTeacherDeliveryMode *csstModel.ClassDeliveryMode `json:"class_section_subject_teacher_delivery_mode" validate:"omitempty,oneof=offline online hybrid"`

	ClassSectionSubjectTeacherTotalMeetingsTarget *int `json:"class_section_subject_teacher_total_meetings_target" validate:"omitempty"`
	ClassSectionSubjectTeacherMinPassingScore     *int `json:"class_section_subject_teacher_min_passing_score" validate:"omitempty,gte=0"`

	ClassSectionSubjectTeacherIsActive *bool `json:"class_section_subject_teacher_is_active" validate:"omitempty"`
}

/*
=========================================================
 2. RESPONSE DTO — sinkron SQL/model terbaru
    + decode teacher snapshot JSONB → TeacherSnapshot struct

=========================================================
*/
type ClassSectionSubjectTeacherResponse struct {
	/* ===== IDs & Relations ===== */
	ClassSectionSubjectTeacherID                       uuid.UUID  `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherSchoolID                 uuid.UUID  `json:"class_section_subject_teacher_school_id"`
	ClassSectionSubjectTeacherClassSectionID           uuid.UUID  `json:"class_section_subject_teacher_class_section_id"`
	ClassSectionSubjectTeacherClassSubjectID           uuid.UUID  `json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherSchoolTeacherID          uuid.UUID  `json:"class_section_subject_teacher_school_teacher_id"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherID *uuid.UUID `json:"class_section_subject_teacher_assistant_school_teacher_id,omitempty"`
	ClassSectionSubjectTeacherClassRoomID              *uuid.UUID `json:"class_section_subject_teacher_class_room_id,omitempty"`

	/* ===== Identitas & Fasilitas ===== */
	ClassSectionSubjectTeacherSlug        *string `json:"class_section_subject_teacher_slug,omitempty"`
	ClassSectionSubjectTeacherDescription *string `json:"class_section_subject_teacher_description,omitempty"`
	ClassSectionSubjectTeacherGroupURL    *string `json:"class_section_subject_teacher_group_url,omitempty"`

	/* ===== Agregat & kapasitas ===== */
	ClassSectionSubjectTeacherTotalAttendance          int    `json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherTotalMeetingsTarget      *int   `json:"class_section_subject_teacher_total_meetings_target,omitempty"`
	ClassSectionSubjectTeacherCapacity                 *int   `json:"class_section_subject_teacher_capacity,omitempty"`
	ClassSectionSubjectTeacherEnrolledCount            int    `json:"class_section_subject_teacher_enrolled_count"`
	ClassSectionSubjectTeacherTotalAssessments         int    `json:"class_section_subject_teacher_total_assessments"`
	ClassSectionSubjectTeacherTotalAssessmentsGraded   int    `json:"class_section_subject_teacher_total_assessments_graded"`
	ClassSectionSubjectTeacherTotalAssessmentsUngraded int    `json:"class_section_subject_teacher_total_assessments_ungraded"`
	ClassSectionSubjectTeacherTotalStudentsPassed      int    `json:"class_section_subject_teacher_total_students_passed"`
	ClassSectionSubjectTeacherDeliveryMode             string `json:"class_section_subject_teacher_delivery_mode"`

	// ➕ NEW: total books
	ClassSectionSubjectTeacherTotalBooks int `json:"class_section_subject_teacher_total_books"`

	/* ===== SECTION snapshots (varchar/text) ===== */
	ClassSectionSubjectTeacherClassSectionSlugSnapshot *string `json:"class_section_subject_teacher_class_section_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSectionNameSnapshot *string `json:"class_section_subject_teacher_class_section_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSectionCodeSnapshot *string `json:"class_section_subject_teacher_class_section_code_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassSectionURLSnapshot  *string `json:"class_section_subject_teacher_class_section_url_snapshot,omitempty"`

	/* ===== ROOM snapshot ===== */
	ClassSectionSubjectTeacherClassRoomSlugSnapshot *string         `json:"class_section_subject_teacher_class_room_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassRoomSnapshot     *datatypes.JSON `json:"class_section_subject_teacher_class_room_snapshot,omitempty"`
	// generated
	ClassSectionSubjectTeacherClassRoomNameSnapshot     *string `json:"class_section_subject_teacher_class_room_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherClassRoomSlugSnapshotGen  *string `json:"class_section_subject_teacher_class_room_slug_snapshot_gen,omitempty"`
	ClassSectionSubjectTeacherClassRoomLocationSnapshot *string `json:"class_section_subject_teacher_class_room_location_snapshot,omitempty"`

	/* ===== PEOPLE snapshots ===== */
	ClassSectionSubjectTeacherSchoolTeacherSlugSnapshot          *string                      `json:"class_section_subject_teacher_school_teacher_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherSchoolTeacherSnapshot              *teacherSnap.TeacherSnapshot `json:"class_section_subject_teacher_school_teacher_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherSlugSnapshot *string                      `json:"class_section_subject_teacher_assistant_school_teacher_slug_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherSnapshot     *teacherSnap.TeacherSnapshot `json:"class_section_subject_teacher_assistant_school_teacher_snapshot,omitempty"`
	// generated names
	ClassSectionSubjectTeacherSchoolTeacherNameSnapshot          *string `json:"class_section_subject_teacher_school_teacher_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherAssistantSchoolTeacherNameSnapshot *string `json:"class_section_subject_teacher_assistant_school_teacher_name_snapshot,omitempty"`

	/* ===== SUBJECT (via CLASS_SUBJECT) snapshot ===== */
	ClassSectionSubjectTeacherSubjectIDSnapshot   *uuid.UUID `json:"class_section_subject_teacher_subject_id_snapshot,omitempty"`
	ClassSectionSubjectTeacherSubjectNameSnapshot *string    `json:"class_section_subject_teacher_subject_name_snapshot,omitempty"`
	ClassSectionSubjectTeacherSubjectCodeSnapshot *string    `json:"class_section_subject_teacher_subject_code_snapshot,omitempty"`
	ClassSectionSubjectTeacherSubjectSlugSnapshot *string    `json:"class_section_subject_teacher_subject_slug_snapshot,omitempty"`

	/* ===== KKM SNAPSHOT (opsional override per CSST) ===== */
	ClassSectionSubjectTeacherMinPassingScore *int `json:"class_section_subject_teacher_min_passing_score,omitempty"`

	/* ===== Status & audit ===== */
	ClassSectionSubjectTeacherIsActive  bool       `json:"class_section_subject_teacher_is_active"`
	ClassSectionSubjectTeacherCreatedAt time.Time  `json:"class_section_subject_teacher_created_at"`
	ClassSectionSubjectTeacherUpdatedAt time.Time  `json:"class_section_subject_teacher_updated_at"`
	ClassSectionSubjectTeacherDeletedAt *time.Time `json:"class_section_subject_teacher_deleted_at,omitempty"`
}

/* =========================================================
   3) MAPPERS
========================================================= */

func (r CreateClassSectionSubjectTeacherRequest) ToModel() csstModel.ClassSectionSubjectTeacherModel {
	m := csstModel.ClassSectionSubjectTeacherModel{
		// Wajib
		ClassSectionSubjectTeacherClassSectionID:  r.ClassSectionSubjectTeacherClassSectionID,
		ClassSectionSubjectTeacherClassSubjectID:  r.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherSchoolTeacherID: r.ClassSectionSubjectTeacherSchoolTeacherID,

		// Opsional basic
		ClassSectionSubjectTeacherSlug:        trimLowerPtr(r.ClassSectionSubjectTeacherSlug), // slug → lowercase
		ClassSectionSubjectTeacherDescription: trimPtr(r.ClassSectionSubjectTeacherDescription),
		ClassSectionSubjectTeacherClassRoomID: r.ClassSectionSubjectTeacherClassRoomID,
		ClassSectionSubjectTeacherGroupURL:    trimPtr(r.ClassSectionSubjectTeacherGroupURL),
	}

	if r.ClassSectionSubjectTeacherAssistantSchoolTeacherID != nil {
		m.ClassSectionSubjectTeacherAssistantSchoolTeacherID = r.ClassSectionSubjectTeacherAssistantSchoolTeacherID
	}
	if r.ClassSectionSubjectTeacherSchoolID != nil {
		m.ClassSectionSubjectTeacherSchoolID = *r.ClassSectionSubjectTeacherSchoolID
	}
	if r.ClassSectionSubjectTeacherIsActive != nil {
		m.ClassSectionSubjectTeacherIsActive = *r.ClassSectionSubjectTeacherIsActive
	} else {
		m.ClassSectionSubjectTeacherIsActive = true
	}
	if r.ClassSectionSubjectTeacherCapacity != nil {
		m.ClassSectionSubjectTeacherCapacity = r.ClassSectionSubjectTeacherCapacity
	}
	if r.ClassSectionSubjectTeacherDeliveryMode != nil {
		m.ClassSectionSubjectTeacherDeliveryMode = *r.ClassSectionSubjectTeacherDeliveryMode
	}
	if r.ClassSectionSubjectTeacherTotalMeetingsTarget != nil {
		m.ClassSectionSubjectTeacherTotalMeetingsTarget = r.ClassSectionSubjectTeacherTotalMeetingsTarget
	}
	if r.ClassSectionSubjectTeacherMinPassingScore != nil {
		m.ClassSectionSubjectTeacherMinPassingScore = r.ClassSectionSubjectTeacherMinPassingScore
	}

	return m
}

func (r UpdateClassSectionSubjectTeacherRequest) Apply(m *csstModel.ClassSectionSubjectTeacherModel) {
	if r.ClassSectionSubjectTeacherSchoolID != nil {
		m.ClassSectionSubjectTeacherSchoolID = *r.ClassSectionSubjectTeacherSchoolID
	}
	if r.ClassSectionSubjectTeacherClassSectionID != nil {
		m.ClassSectionSubjectTeacherClassSectionID = *r.ClassSectionSubjectTeacherClassSectionID
	}
	if r.ClassSectionSubjectTeacherClassSubjectID != nil {
		m.ClassSectionSubjectTeacherClassSubjectID = *r.ClassSectionSubjectTeacherClassSubjectID
	}
	if r.ClassSectionSubjectTeacherSchoolTeacherID != nil {
		m.ClassSectionSubjectTeacherSchoolTeacherID = *r.ClassSectionSubjectTeacherSchoolTeacherID
	}

	if r.ClassSectionSubjectTeacherAssistantSchoolTeacherID != nil {
		m.ClassSectionSubjectTeacherAssistantSchoolTeacherID = r.ClassSectionSubjectTeacherAssistantSchoolTeacherID
	}

	if r.ClassSectionSubjectTeacherSlug != nil {
		m.ClassSectionSubjectTeacherSlug = trimLowerPtr(r.ClassSectionSubjectTeacherSlug)
	}
	if r.ClassSectionSubjectTeacherDescription != nil {
		m.ClassSectionSubjectTeacherDescription = trimPtr(r.ClassSectionSubjectTeacherDescription)
	}
	if r.ClassSectionSubjectTeacherClassRoomID != nil {
		m.ClassSectionSubjectTeacherClassRoomID = r.ClassSectionSubjectTeacherClassRoomID
	}
	if r.ClassSectionSubjectTeacherGroupURL != nil {
		m.ClassSectionSubjectTeacherGroupURL = trimPtr(r.ClassSectionSubjectTeacherGroupURL)
	}
	if r.ClassSectionSubjectTeacherCapacity != nil {
		m.ClassSectionSubjectTeacherCapacity = r.ClassSectionSubjectTeacherCapacity
	}
	if r.ClassSectionSubjectTeacherDeliveryMode != nil {
		m.ClassSectionSubjectTeacherDeliveryMode = *r.ClassSectionSubjectTeacherDeliveryMode
	}
	if r.ClassSectionSubjectTeacherTotalMeetingsTarget != nil {
		m.ClassSectionSubjectTeacherTotalMeetingsTarget = r.ClassSectionSubjectTeacherTotalMeetingsTarget
	}
	if r.ClassSectionSubjectTeacherMinPassingScore != nil {
		m.ClassSectionSubjectTeacherMinPassingScore = r.ClassSectionSubjectTeacherMinPassingScore
	}
	if r.ClassSectionSubjectTeacherIsActive != nil {
		m.ClassSectionSubjectTeacherIsActive = *r.ClassSectionSubjectTeacherIsActive
	}
}

func FromClassSectionSubjectTeacherModel(m csstModel.ClassSectionSubjectTeacherModel) ClassSectionSubjectTeacherResponse {
	var deletedAt *time.Time
	if m.ClassSectionSubjectTeacherDeletedAt.Valid {
		t := m.ClassSectionSubjectTeacherDeletedAt.Time
		deletedAt = &t
	}

	return ClassSectionSubjectTeacherResponse{
		// IDs & Relations
		ClassSectionSubjectTeacherID:                       m.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherSchoolID:                 m.ClassSectionSubjectTeacherSchoolID,
		ClassSectionSubjectTeacherClassSectionID:           m.ClassSectionSubjectTeacherClassSectionID,
		ClassSectionSubjectTeacherClassSubjectID:           m.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherSchoolTeacherID:          m.ClassSectionSubjectTeacherSchoolTeacherID,
		ClassSectionSubjectTeacherAssistantSchoolTeacherID: m.ClassSectionSubjectTeacherAssistantSchoolTeacherID,
		ClassSectionSubjectTeacherClassRoomID:              m.ClassSectionSubjectTeacherClassRoomID,

		// Identitas / fasilitas
		ClassSectionSubjectTeacherSlug:        m.ClassSectionSubjectTeacherSlug,
		ClassSectionSubjectTeacherDescription: m.ClassSectionSubjectTeacherDescription,
		ClassSectionSubjectTeacherGroupURL:    m.ClassSectionSubjectTeacherGroupURL,

		// Agregat & kapasitas
		ClassSectionSubjectTeacherTotalAttendance:          m.ClassSectionSubjectTeacherTotalAttendance,
		ClassSectionSubjectTeacherTotalMeetingsTarget:      m.ClassSectionSubjectTeacherTotalMeetingsTarget,
		ClassSectionSubjectTeacherCapacity:                 m.ClassSectionSubjectTeacherCapacity,
		ClassSectionSubjectTeacherEnrolledCount:            m.ClassSectionSubjectTeacherEnrolledCount,
		ClassSectionSubjectTeacherTotalBooks:               m.ClassSectionSubjectTeacherTotalBooks,
		ClassSectionSubjectTeacherTotalAssessments:         m.ClassSectionSubjectTeacherTotalAssessments,
		ClassSectionSubjectTeacherTotalAssessmentsGraded:   m.ClassSectionSubjectTeacherTotalAssessmentsGraded,
		ClassSectionSubjectTeacherTotalAssessmentsUngraded: m.ClassSectionSubjectTeacherTotalAssessmentsUngraded,
		ClassSectionSubjectTeacherTotalStudentsPassed:      m.ClassSectionSubjectTeacherTotalStudentsPassed,
		ClassSectionSubjectTeacherDeliveryMode:             string(m.ClassSectionSubjectTeacherDeliveryMode),

		// SECTION snapshots
		ClassSectionSubjectTeacherClassSectionSlugSnapshot: m.ClassSectionSubjectTeacherClassSectionSlugSnapshot,
		ClassSectionSubjectTeacherClassSectionNameSnapshot: m.ClassSectionSubjectTeacherClassSectionNameSnapshot,
		ClassSectionSubjectTeacherClassSectionCodeSnapshot: m.ClassSectionSubjectTeacherClassSectionCodeSnapshot,
		ClassSectionSubjectTeacherClassSectionURLSnapshot:  m.ClassSectionSubjectTeacherClassSectionURLSnapshot,

		// ROOM snapshot + generated
		ClassSectionSubjectTeacherClassRoomSlugSnapshot:     m.ClassSectionSubjectTeacherClassRoomSlugSnapshot,
		ClassSectionSubjectTeacherClassRoomSnapshot:         m.ClassSectionSubjectTeacherClassRoomSnapshot,
		ClassSectionSubjectTeacherClassRoomNameSnapshot:     m.ClassSectionSubjectTeacherClassRoomNameSnapshot,
		ClassSectionSubjectTeacherClassRoomSlugSnapshotGen:  m.ClassSectionSubjectTeacherClassRoomSlugSnapshotGen,
		ClassSectionSubjectTeacherClassRoomLocationSnapshot: m.ClassSectionSubjectTeacherClassRoomLocationSnapshot,

		// PEOPLE snapshots + generated names
		ClassSectionSubjectTeacherSchoolTeacherSlugSnapshot:          m.ClassSectionSubjectTeacherSchoolTeacherSlugSnapshot,
		ClassSectionSubjectTeacherSchoolTeacherSnapshot:              teacherSnapshotFromJSON(m.ClassSectionSubjectTeacherSchoolTeacherSnapshot),
		ClassSectionSubjectTeacherAssistantSchoolTeacherSlugSnapshot: m.ClassSectionSubjectTeacherAssistantSchoolTeacherSlugSnapshot,
		ClassSectionSubjectTeacherAssistantSchoolTeacherSnapshot:     teacherSnapshotFromJSON(m.ClassSectionSubjectTeacherAssistantSchoolTeacherSnapshot),
		ClassSectionSubjectTeacherSchoolTeacherNameSnapshot:          m.ClassSectionSubjectTeacherSchoolTeacherNameSnapshot,
		ClassSectionSubjectTeacherAssistantSchoolTeacherNameSnapshot: m.ClassSectionSubjectTeacherAssistantSchoolTeacherNameSnapshot,

		// SUBJECT snapshot
		ClassSectionSubjectTeacherSubjectIDSnapshot:   m.ClassSectionSubjectTeacherSubjectIDSnapshot,
		ClassSectionSubjectTeacherSubjectNameSnapshot: m.ClassSectionSubjectTeacherSubjectNameSnapshot,
		ClassSectionSubjectTeacherSubjectCodeSnapshot: m.ClassSectionSubjectTeacherSubjectCodeSnapshot,
		ClassSectionSubjectTeacherSubjectSlugSnapshot: m.ClassSectionSubjectTeacherSubjectSlugSnapshot,

		// KKM
		ClassSectionSubjectTeacherMinPassingScore: m.ClassSectionSubjectTeacherMinPassingScore,

		// Status & audit
		ClassSectionSubjectTeacherIsActive:  m.ClassSectionSubjectTeacherIsActive,
		ClassSectionSubjectTeacherCreatedAt: m.ClassSectionSubjectTeacherCreatedAt,
		ClassSectionSubjectTeacherUpdatedAt: m.ClassSectionSubjectTeacherUpdatedAt,
		ClassSectionSubjectTeacherDeletedAt: deletedAt,
	}
}

func FromClassSectionSubjectTeacherModels(rows []csstModel.ClassSectionSubjectTeacherModel) []ClassSectionSubjectTeacherResponse {
	out := make([]ClassSectionSubjectTeacherResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, FromClassSectionSubjectTeacherModel(r))
	}
	return out
}
