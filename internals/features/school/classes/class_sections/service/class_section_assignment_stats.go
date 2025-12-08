// file: internals/features/lembaga/school_yayasans/teachers_students/services/section_assignment_stats.go
package service

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	tsModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	secModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
)

/* ============================
   Schema JSON: section snapshot
   Dipakai oleh:
   - school_teachers.school_teacher_sections
   - school_students.school_student_class_sections
   ============================ */

type SectionSnapshot struct {
	ClassSectionID             uuid.UUID `json:"class_section_id"`
	Role                       string    `json:"role"` // "homeroom" | "assistant" | "member" | "leader" | dll
	IsActive                   bool      `json:"is_active"`
	From                       *string   `json:"from,omitempty"` // YYYY-MM-DD
	To                         *string   `json:"to,omitempty"`   // YYYY-MM-DD | null
	ClassSectionName           *string   `json:"class_section_name,omitempty"`
	ClassSectionSlug           *string   `json:"class_section_slug,omitempty"`
	ClassSectionImageURL       *string   `json:"class_section_image_url,omitempty"`
	ClassSectionImageObjectKey *string   `json:"class_section_image_object_key,omitempty"`

	// ===== Stats penting dari class_sections =====
	ClassSectionTotalStudentsActive       int `json:"class_section_total_students_active"`
	ClassSectionTotalStudentsMale         int `json:"class_section_total_students_male"`
	ClassSectionTotalStudentsFemale       int `json:"class_section_total_students_female"`
	ClassSectionTotalStudentsMaleActive   int `json:"class_section_total_students_male_active"`
	ClassSectionTotalStudentsFemaleActive int `json:"class_section_total_students_female_active"`

	// ===== Academic term cache =====
	ClassSectionAcademicTermID                *uuid.UUID `json:"class_section_academic_term_id,omitempty"`
	ClassSectionAcademicTermNameCache         *string    `json:"class_section_academic_term_name_cache,omitempty"`
	ClassSectionAcademicTermSlugCache         *string    `json:"class_section_academic_term_slug_cache,omitempty"`
	ClassSectionAcademicTermAcademicYearCache *string    `json:"class_section_academic_term_academic_year_cache,omitempty"`
	ClassSectionAcademicTermAngkatanCache     *int       `json:"class_section_academic_term_angkatan_cache,omitempty"`

	// ===== Class parent cache =====
	ClassSectionClassParentID         *uuid.UUID `json:"class_section_class_parent_id,omitempty"`
	ClassSectionClassParentNameCache  *string    `json:"class_section_class_parent_name_cache,omitempty"`
	ClassSectionClassParentSlugCache  *string    `json:"class_section_class_parent_slug_cache,omitempty"`
	ClassSectionClassParentLevelCache *int16     `json:"class_section_class_parent_level_cache,omitempty"`
}

// decode JSONB → slice
func decodeSectionSnapshots(js datatypes.JSON) []SectionSnapshot {
	if len(js) == 0 {
		return []SectionSnapshot{}
	}
	var out []SectionSnapshot
	if err := json.Unmarshal(js, &out); err != nil {
		return []SectionSnapshot{}
	}
	return out
}

// encode slice → JSONB
func encodeSectionSnapshots(items []SectionSnapshot) datatypes.JSON {
	if items == nil {
		items = []SectionSnapshot{}
	}
	b, err := json.Marshal(items)
	if err != nil {
		return datatypes.JSON([]byte("[]"))
	}
	return datatypes.JSON(b)
}

// build snapshot dari ClassSectionModel
func buildSectionSnapshot(sec *secModel.ClassSectionModel, role string) SectionSnapshot {
	if sec == nil {
		return SectionSnapshot{}
	}

	name := sec.ClassSectionName
	slug := sec.ClassSectionSlug

	return SectionSnapshot{
		ClassSectionID:             sec.ClassSectionID,
		Role:                       role,
		IsActive:                   sec.ClassSectionIsActive,
		From:                       nil,
		To:                         nil,
		ClassSectionName:           &name,
		ClassSectionSlug:           &slug,
		ClassSectionImageURL:       sec.ClassSectionImageURL,
		ClassSectionImageObjectKey: sec.ClassSectionImageObjectKey,

		// Stats
		ClassSectionTotalStudentsActive:       sec.ClassSectionTotalStudentsActive,
		ClassSectionTotalStudentsMale:         sec.ClassSectionTotalStudentsMale,
		ClassSectionTotalStudentsFemale:       sec.ClassSectionTotalStudentsFemale,
		ClassSectionTotalStudentsMaleActive:   sec.ClassSectionTotalStudentsMaleActive,
		ClassSectionTotalStudentsFemaleActive: sec.ClassSectionTotalStudentsFemaleActive,

		// Academic term
		ClassSectionAcademicTermID:                sec.ClassSectionAcademicTermID,
		ClassSectionAcademicTermNameCache:         sec.ClassSectionAcademicTermNameCache,
		ClassSectionAcademicTermSlugCache:         sec.ClassSectionAcademicTermSlugCache,
		ClassSectionAcademicTermAcademicYearCache: sec.ClassSectionAcademicTermAcademicYearCache,
		ClassSectionAcademicTermAngkatanCache:     sec.ClassSectionAcademicTermAngkatanCache,

		// Class parent
		ClassSectionClassParentID:         sec.ClassSectionClassParentID,
		ClassSectionClassParentNameCache:  sec.ClassSectionClassParentNameCache,
		ClassSectionClassParentSlugCache:  sec.ClassSectionClassParentSlugCache,
		ClassSectionClassParentLevelCache: sec.ClassSectionClassParentLevelCache,
	}
}

/* ============================================================
   GURU: school_teachers.*_sections + stats
   ============================================================ */

// ✅ panggil saat SECTION di-assign ke guru (homeroom/assistant)
func AddSectionToTeacher(
	tx *gorm.DB,
	schoolID uuid.UUID,
	teacherID uuid.UUID,
	sec *secModel.ClassSectionModel,
	role string, // "homeroom" | "assistant"
) error {
	if sec == nil {
		return nil
	}

	var t tsModel.SchoolTeacherModel
	if err := tx.
		Where("school_teacher_id = ? AND school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL",
			teacherID, schoolID).
		First(&t).Error; err != nil {
		// kalau teacher belum ada, jangan gagalkan proses utama
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	items := decodeSectionSnapshots(t.SchoolTeacherSections)

	found := false
	for i := range items {
		if items[i].ClassSectionID == sec.ClassSectionID && items[i].Role == role {
			// update snapshot kalau sudah ada
			items[i] = buildSectionSnapshot(sec, role)
			found = true
			break
		}
	}

	if !found {
		// baru ⇒ append + inc stats
		items = append(items, buildSectionSnapshot(sec, role))
		t.SchoolTeacherTotalClassSections++
		if sec.ClassSectionIsActive {
			t.SchoolTeacherTotalClassSectionsActive++
		}
	}

	t.SchoolTeacherSections = encodeSectionSnapshots(items)

	return tx.Save(&t).Error
}

// ✅ panggil saat SECTION di-unassign dari guru (atau pindah ke guru lain)
func RemoveSectionFromTeacher(
	tx *gorm.DB,
	schoolID uuid.UUID,
	teacherID uuid.UUID,
	sec *secModel.ClassSectionModel,
	role string,
) error {
	if sec == nil {
		return nil
	}

	var t tsModel.SchoolTeacherModel
	if err := tx.
		Where("school_teacher_id = ? AND school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL",
			teacherID, schoolID).
		First(&t).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	items := decodeSectionSnapshots(t.SchoolTeacherSections)

	newItems := make([]SectionSnapshot, 0, len(items))
	removed := false
	for _, it := range items {
		if it.ClassSectionID == sec.ClassSectionID && it.Role == role {
			removed = true
			continue
		}
		newItems = append(newItems, it)
	}

	if removed {
		if t.SchoolTeacherTotalClassSections > 0 {
			t.SchoolTeacherTotalClassSections--
		}
		if sec.ClassSectionIsActive && t.SchoolTeacherTotalClassSectionsActive > 0 {
			t.SchoolTeacherTotalClassSectionsActive--
		}
	}

	t.SchoolTeacherSections = encodeSectionSnapshots(newItems)

	return tx.Save(&t).Error
}

/* ============================================================
   SISWA: school_students.*_sections + stats
   (pakai SchoolStudentClassSections)
   ============================================================ */

// ✅ panggil saat SECTION di-assign ke siswa
func AddSectionToStudent(
	tx *gorm.DB,
	schoolID uuid.UUID,
	studentID uuid.UUID,
	sec *secModel.ClassSectionModel,
	role string, // "member" | "leader" | dll
) error {
	if sec == nil {
		return nil
	}

	var s tsModel.SchoolStudentModel
	if err := tx.
		Where("school_student_id = ? AND school_student_school_id = ? AND school_student_deleted_at IS NULL",
			studentID, schoolID).
		First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	items := decodeSectionSnapshots(s.SchoolStudentClassSections)

	found := false
	for i := range items {
		if items[i].ClassSectionID == sec.ClassSectionID && items[i].Role == role {
			items[i] = buildSectionSnapshot(sec, role)
			found = true
			break
		}
	}

	if !found {
		items = append(items, buildSectionSnapshot(sec, role))
		s.SchoolStudentTotalClassSections++
		if sec.ClassSectionIsActive {
			s.SchoolStudentTotalClassSectionsActive++
		}
	}

	s.SchoolStudentClassSections = encodeSectionSnapshots(items)

	return tx.Save(&s).Error
}

// ✅ panggil saat SECTION di-unassign dari siswa
func RemoveSectionFromStudent(
	tx *gorm.DB,
	schoolID uuid.UUID,
	studentID uuid.UUID,
	sec *secModel.ClassSectionModel,
	role string,
) error {
	if sec == nil {
		return nil
	}

	var s tsModel.SchoolStudentModel
	if err := tx.
		Where("school_student_id = ? AND school_student_school_id = ? AND school_student_deleted_at IS NULL",
			studentID, schoolID).
		First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	items := decodeSectionSnapshots(s.SchoolStudentClassSections)

	newItems := make([]SectionSnapshot, 0, len(items))
	removed := false
	for _, it := range items {
		if it.ClassSectionID == sec.ClassSectionID && it.Role == role {
			removed = true
			continue
		}
		newItems = append(newItems, it)
	}

	if removed {
		if s.SchoolStudentTotalClassSections > 0 {
			s.SchoolStudentTotalClassSections--
		}
		if sec.ClassSectionIsActive && s.SchoolStudentTotalClassSectionsActive > 0 {
			s.SchoolStudentTotalClassSectionsActive--
		}
	}

	s.SchoolStudentClassSections = encodeSectionSnapshots(newItems)

	return tx.Save(&s).Error
}
