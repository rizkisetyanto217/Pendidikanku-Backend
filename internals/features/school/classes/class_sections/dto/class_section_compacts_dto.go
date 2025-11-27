package dto

import (
	"encoding/json"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	m "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	"github.com/google/uuid"
)

// ===== COMPACT DTO (untuk embed di tempat lain, misal enrollment) =====

type ClassSectionCompact struct {
	ClassSectionID      uuid.UUID  `json:"class_section_id"`
	ClassSectionName    string     `json:"class_section_name"`
	ClassSectionClassID *uuid.UUID `json:"class_section_class_id,omitempty"`
	ClassSectionSlug    *string    `json:"class_section_slug,omitempty"`

	// --- NEW: class snapshots ---
	ClassSectionClassNameSnapshot *string `json:"class_section_class_name_snapshot,omitempty"`
	ClassSectionClassSlugSnapshot *string `json:"class_section_class_slug_snapshot,omitempty"`

	// --- NEW: parent snapshots (kalau mau dipakai di FE) ---
	ClassSectionClassParentID            *uuid.UUID `json:"class_section_class_parent_id,omitempty"`
	ClassSectionClassParentNameSnapshot  *string    `json:"class_section_class_parent_name_snapshot,omitempty"`
	ClassSectionClassParentSlugSnapshot  *string    `json:"class_section_class_parent_slug_snapshot,omitempty"`
	ClassSectionClassParentLevelSnapshot *int16     `json:"class_section_class_parent_level_snapshot,omitempty"`

	// Info dasar tambahan
	ClassSectionCode     *string `json:"class_section_code,omitempty"`
	ClassSectionSchedule *string `json:"class_section_schedule,omitempty"`

	ClassSectionCapacity      *int `json:"class_section_capacity,omitempty"`
	ClassSectionTotalStudents int  `json:"class_section_total_students"`
	ClassSectionIsActive      bool `json:"class_section_is_active"`

	// Stats (ALL & ACTIVE)
	ClassSectionTotalStudentsActive       int             `json:"class_section_total_students_active"`
	ClassSectionTotalStudentsMale         int             `json:"class_section_total_students_male"`
	ClassSectionTotalStudentsFemale       int             `json:"class_section_total_students_female"`
	ClassSectionTotalStudentsMaleActive   int             `json:"class_section_total_students_male_active"`
	ClassSectionTotalStudentsFemaleActive int             `json:"class_section_total_students_female_active"`
	ClassSectionStats                     json.RawMessage `json:"class_section_stats,omitempty"`

	// CSST totals
	ClassSectionTotalClassClassSectionSubjectTeachers       int `json:"class_section_total_class_class_section_subject_teachers"`
	ClassSectionTotalClassClassSectionSubjectTeachersActive int `json:"class_section_total_class_class_section_subject_teachers_active"`

	// Link & image
	ClassSectionGroupURL *string `json:"class_section_group_url,omitempty"`
	ClassSectionImageURL *string `json:"class_section_image_url,omitempty"`

	// Homeroom teacher (wali kelas) - ID + slug lama
	ClassSectionSchoolTeacherID           *uuid.UUID `json:"class_section_school_teacher_id,omitempty"`
	ClassSectionSchoolTeacherSlugSnapshot *string    `json:"class_section_school_teacher_slug_snapshot,omitempty"`

	// NEW: objek guru dari snapshot (nama, avatar, gender, nomor induk, dll)
	HomeroomTeacher  *TeacherPersonLite `json:"homeroom_teacher,omitempty"`
	AssistantTeacher *TeacherPersonLite `json:"assistant_teacher,omitempty"`

	// Room
	ClassSectionClassRoomID               *uuid.UUID `json:"class_section_class_room_id,omitempty"`
	ClassSectionClassRoomSlugSnapshot     *string    `json:"class_section_class_room_slug_snapshot,omitempty"`
	ClassSectionClassRoomNameSnapshot     *string    `json:"class_section_class_room_name_snapshot,omitempty"`
	ClassSectionClassRoomLocationSnapshot *string    `json:"class_section_class_room_location_snapshot,omitempty"`

	// TERM
	ClassSectionAcademicTermID                   *uuid.UUID `json:"class_section_academic_term_id,omitempty"`
	ClassSectionAcademicTermNameSnapshot         *string    `json:"class_section_academic_term_name_snapshot,omitempty"`
	ClassSectionAcademicTermSlugSnapshot         *string    `json:"class_section_academic_term_slug_snapshot,omitempty"`
	ClassSectionAcademicTermAcademicYearSnapshot *string    `json:"class_section_academic_year_snapshot,omitempty"`
	ClassSectionAcademicTermAngkatanSnapshot     *int       `json:"class_section_angkatan_snapshot,omitempty"`

	SubjectTeachers []csstModel.ClassSectionSubjectTeacherModel `json:"class_section_subject_teachers,omitempty"`
}

// FromModelsCompact: mapping dari []ClassSectionModel → []ClassSectionCompact
func FromModelsClassSectionCompact(rows []m.ClassSectionModel) []ClassSectionCompact {
	out := make([]ClassSectionCompact, 0, len(rows))
	for _, cs := range rows {
		slug := cs.ClassSectionSlug

		var statsRaw json.RawMessage
		if len(cs.ClassSectionStats) > 0 {
			statsRaw = json.RawMessage(cs.ClassSectionStats)
		}

		item := ClassSectionCompact{
			ClassSectionID:      cs.ClassSectionID,
			ClassSectionName:    cs.ClassSectionName,
			ClassSectionClassID: cs.ClassSectionClassID,
			ClassSectionSlug:    &slug,

			ClassSectionCode:     cs.ClassSectionCode,
			ClassSectionSchedule: cs.ClassSectionSchedule,

			ClassSectionCapacity:      cs.ClassSectionCapacity,
			ClassSectionTotalStudents: cs.ClassSectionTotalStudents,
			ClassSectionIsActive:      cs.ClassSectionIsActive,

			ClassSectionTotalStudentsActive:       cs.ClassSectionTotalStudentsActive,
			ClassSectionTotalStudentsMale:         cs.ClassSectionTotalStudentsMale,
			ClassSectionTotalStudentsFemale:       cs.ClassSectionTotalStudentsFemale,
			ClassSectionTotalStudentsMaleActive:   cs.ClassSectionTotalStudentsMaleActive,
			ClassSectionTotalStudentsFemaleActive: cs.ClassSectionTotalStudentsFemaleActive,
			ClassSectionStats:                     statsRaw,

			ClassSectionTotalClassClassSectionSubjectTeachers:       cs.ClassSectionTotalClassClassSectionSubjectTeachers,
			ClassSectionTotalClassClassSectionSubjectTeachersActive: cs.ClassSectionTotalClassClassSectionSubjectTeachersActive,

			ClassSectionGroupURL: cs.ClassSectionGroupURL,
			ClassSectionImageURL: cs.ClassSectionImageURL,

			ClassSectionSchoolTeacherID:           cs.ClassSectionSchoolTeacherID,
			ClassSectionSchoolTeacherSlugSnapshot: cs.ClassSectionSchoolTeacherSlugSnapshot,

			ClassSectionClassRoomID:               cs.ClassSectionClassRoomID,
			ClassSectionClassRoomSlugSnapshot:     cs.ClassSectionClassRoomSlugSnapshot,
			ClassSectionClassRoomNameSnapshot:     cs.ClassSectionClassRoomNameSnapshot,
			ClassSectionClassRoomLocationSnapshot: cs.ClassSectionClassRoomLocationSnapshot,

			// --- NEW: class snapshots ---
			ClassSectionClassNameSnapshot: cs.ClassSectionClassNameSnapshot,
			ClassSectionClassSlugSnapshot: cs.ClassSectionClassSlugSnapshot,

			// --- NEW: parent snapshots ---
			ClassSectionClassParentID:            cs.ClassSectionClassParentID,
			ClassSectionClassParentNameSnapshot:  cs.ClassSectionClassParentNameSnapshot,
			ClassSectionClassParentSlugSnapshot:  cs.ClassSectionClassParentSlugSnapshot,
			ClassSectionClassParentLevelSnapshot: cs.ClassSectionClassParentLevelSnapshot,

			// TERM
			ClassSectionAcademicTermID:                   cs.ClassSectionAcademicTermID,
			ClassSectionAcademicTermNameSnapshot:         cs.ClassSectionAcademicTermNameSnapshot,
			ClassSectionAcademicTermSlugSnapshot:         cs.ClassSectionAcademicTermSlugSnapshot,
			ClassSectionAcademicTermAcademicYearSnapshot: cs.ClassSectionAcademicTermAcademicYearSnapshot,
			ClassSectionAcademicTermAngkatanSnapshot:     cs.ClassSectionAcademicTermAngkatanSnapshot,
		}

		if t := teacherLiteFromJSON(cs.ClassSectionSchoolTeacherSnapshot); t != nil {
			item.HomeroomTeacher = t
		}
		if t := teacherLiteFromJSON(cs.ClassSectionAssistantSchoolTeacherSnapshot); t != nil {
			item.AssistantTeacher = t
		}

		out = append(out, item)
	}
	return out
}
