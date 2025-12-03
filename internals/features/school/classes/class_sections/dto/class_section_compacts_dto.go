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

	// --- class cache ---
	ClassSectionClassNameCache *string `json:"class_section_class_name_cache,omitempty"`
	ClassSectionClassSlugCache *string `json:"class_section_class_slug_cache,omitempty"`

	// --- parent cache (kalau mau dipakai di FE) ---
	ClassSectionClassParentID         *uuid.UUID `json:"class_section_class_parent_id,omitempty"`
	ClassSectionClassParentNameCache  *string    `json:"class_section_class_parent_name_cache,omitempty"`
	ClassSectionClassParentSlugCache  *string    `json:"class_section_class_parent_slug_cache,omitempty"`
	ClassSectionClassParentLevelCache *int16     `json:"class_section_class_parent_level_cache,omitempty"`

	// Info dasar tambahan
	ClassSectionCode     *string `json:"class_section_code,omitempty"`
	ClassSectionSchedule *string `json:"class_section_schedule,omitempty"`

	// Kuota (mirror ke model: quota_total / quota_taken)
	ClassSectionQuotaTotal *int `json:"class_section_quota_total,omitempty"`
	ClassSectionQuotaTaken int  `json:"class_section_quota_taken"`

	ClassSectionIsActive bool `json:"class_section_is_active"`

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

	// Homeroom teacher (wali kelas) - ID + slug cache
	ClassSectionSchoolTeacherID        *uuid.UUID `json:"class_section_school_teacher_id,omitempty"`
	ClassSectionSchoolTeacherSlugCache *string    `json:"class_section_school_teacher_slug_cache,omitempty"`

	// NEW: objek guru dari cache (nama, avatar, gender, nomor induk, dll)
	HomeroomTeacher  *TeacherPersonLite `json:"homeroom_teacher,omitempty"`
	AssistantTeacher *TeacherPersonLite `json:"assistant_teacher,omitempty"`

	// Room
	ClassSectionClassRoomID            *uuid.UUID `json:"class_section_class_room_id,omitempty"`
	ClassSectionClassRoomSlugCache     *string    `json:"class_section_class_room_slug_cache,omitempty"`
	ClassSectionClassRoomNameCache     *string    `json:"class_section_class_room_name_cache,omitempty"`
	ClassSectionClassRoomLocationCache *string    `json:"class_section_class_room_location_cache,omitempty"`

	// TERM
	ClassSectionAcademicTermID                *uuid.UUID `json:"class_section_academic_term_id,omitempty"`
	ClassSectionAcademicTermNameCache         *string    `json:"class_section_academic_term_name_cache,omitempty"`
	ClassSectionAcademicTermSlugCache         *string    `json:"class_section_academic_term_slug_cache,omitempty"`
	ClassSectionAcademicTermAcademicYearCache *string    `json:"class_section_academic_term_academic_year_cache,omitempty"`
	ClassSectionAcademicTermAngkatanCache     *int       `json:"class_section_academic_term_angkatan_cache,omitempty"`

	SubjectTeachers []csstModel.ClassSectionSubjectTeacherModel `json:"class_section_subject_teachers,omitempty"`
}

// FromModelsClassSectionCompact: mapping dari []ClassSectionModel → []ClassSectionCompact
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

			// Kuota
			ClassSectionQuotaTotal: cs.ClassSectionQuotaTotal,
			ClassSectionQuotaTaken: cs.ClassSectionQuotaTaken,

			ClassSectionIsActive: cs.ClassSectionIsActive,

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

			ClassSectionSchoolTeacherID:        cs.ClassSectionSchoolTeacherID,
			ClassSectionSchoolTeacherSlugCache: cs.ClassSectionSchoolTeacherSlugCache,

			ClassSectionClassRoomID:            cs.ClassSectionClassRoomID,
			ClassSectionClassRoomSlugCache:     cs.ClassSectionClassRoomSlugCache,
			ClassSectionClassRoomNameCache:     cs.ClassSectionClassRoomNameCache,
			ClassSectionClassRoomLocationCache: cs.ClassSectionClassRoomLocationCache,

			// class cache
			ClassSectionClassNameCache: cs.ClassSectionClassNameCache,
			ClassSectionClassSlugCache: cs.ClassSectionClassSlugCache,

			// parent cache
			ClassSectionClassParentID:         cs.ClassSectionClassParentID,
			ClassSectionClassParentNameCache:  cs.ClassSectionClassParentNameCache,
			ClassSectionClassParentSlugCache:  cs.ClassSectionClassParentSlugCache,
			ClassSectionClassParentLevelCache: cs.ClassSectionClassParentLevelCache,

			// TERM
			ClassSectionAcademicTermID:                cs.ClassSectionAcademicTermID,
			ClassSectionAcademicTermNameCache:         cs.ClassSectionAcademicTermNameCache,
			ClassSectionAcademicTermSlugCache:         cs.ClassSectionAcademicTermSlugCache,
			ClassSectionAcademicTermAcademicYearCache: cs.ClassSectionAcademicTermAcademicYearCache,
			ClassSectionAcademicTermAngkatanCache:     cs.ClassSectionAcademicTermAngkatanCache,
		}

		// Homeroom & assistant teacher dari JSON cache
		if t := teacherLiteFromJSON(cs.ClassSectionSchoolTeacherCache); t != nil {
			item.HomeroomTeacher = t
		}
		if t := teacherLiteFromJSON(cs.ClassSectionAssistantSchoolTeacherCache); t != nil {
			item.AssistantTeacher = t
		}

		out = append(out, item)
	}
	return out
}
