// file: internals/features/lembaga/school_yayasans/teachers_students/services/csst_assignment_stats.go
package service

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	stModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
)

/* ============================
   Schema JSON: CSST snapshot
   Disimpan di:
   - school_teachers.school_teacher_csst (jsonb, array)
   - school_students.school_student_class_section_subject_teachers (jsonb, array)
   ============================ */

// NOTE: semua json tag disamain prefix-nya:
// "class_section_subject_teacher_..."
type CSSTSnapshot struct {
	// Identitas dasar CSST
	ClassSectionSubjectTeacherID       uuid.UUID `json:"class_section_subject_teacher_id"`
	ClassSectionSubjectTeacherRole     string    `json:"class_section_subject_teacher_role"`      // "main" | "assistant" | dll
	ClassSectionSubjectTeacherIsActive bool      `json:"class_section_subject_teacher_is_active"` // status aktif CSST

	// --- SUBJECT info (mirror dari model) ---
	ClassSectionSubjectTeacherClassSubjectID   uuid.UUID  `json:"class_section_subject_teacher_class_subject_id"`
	ClassSectionSubjectTeacherSubjectID        *uuid.UUID `json:"class_section_subject_teacher_subject_id,omitempty"`
	ClassSectionSubjectTeacherSubjectNameCache *string    `json:"class_section_subject_teacher_subject_name_cache,omitempty"`
	ClassSectionSubjectTeacherSubjectSlugCache *string    `json:"class_section_subject_teacher_subject_slug_cache,omitempty"`

	// --- ACADEMIC TERM info ---
	ClassSectionSubjectTeacherAcademicTermID        *uuid.UUID `json:"class_section_subject_teacher_academic_term_id,omitempty"`
	ClassSectionSubjectTeacherAcademicTermNameCache *string    `json:"class_section_subject_teacher_academic_term_name_cache,omitempty"`
	ClassSectionSubjectTeacherAcademicTermSlugCache *string    `json:"class_section_subject_teacher_academic_term_slug_cache,omitempty"`

	// --- KKM / Passing score ---
	ClassSectionSubjectTeacherMinPassingScoreClassSubjectCache *int `json:"class_section_subject_teacher_min_passing_score_class_subject_cache,omitempty"`
	ClassSectionSubjectTeacherMinPassingScore                  *int `json:"class_section_subject_teacher_min_passing_score,omitempty"`

	// --- Aggregates / stats penting ---
	ClassSectionSubjectTeacherTotalAttendance          int  `json:"class_section_subject_teacher_total_attendance"`
	ClassSectionSubjectTeacherTotalMeetingsTarget      *int `json:"class_section_subject_teacher_total_meetings_target,omitempty"`
	ClassSectionSubjectTeacherTotalAssessments         int  `json:"class_section_subject_teacher_total_assessments"`
	ClassSectionSubjectTeacherTotalAssessmentsGraded   int  `json:"class_section_subject_teacher_total_assessments_graded"`
	ClassSectionSubjectTeacherTotalAssessmentsUngraded int  `json:"class_section_subject_teacher_total_assessments_ungraded"`
	ClassSectionSubjectTeacherTotalStudentsPassed      int  `json:"class_section_subject_teacher_total_students_passed"`

	// --- Quota (yang sering kepakai) ---
	ClassSectionSubjectTeacherQuotaTaken int `json:"class_section_subject_teacher_quota_taken"`
}

/* ============================
   Helpers: encode/decode JSONB
   ============================ */

func decodeCSSTSnapshots(js datatypes.JSON) []CSSTSnapshot {
	if len(js) == 0 {
		return []CSSTSnapshot{}
	}
	var out []CSSTSnapshot
	if err := json.Unmarshal(js, &out); err != nil {
		return []CSSTSnapshot{}
	}
	return out
}

func encodeCSSTSnapshots(items []CSSTSnapshot) datatypes.JSON {
	if items == nil {
		items = []CSSTSnapshot{}
	}
	b, err := json.Marshal(items)
	if err != nil {
		return datatypes.JSON([]byte("[]"))
	}
	return datatypes.JSON(b)
}

// build snapshot dari ClassSectionSubjectTeacherModel
func buildCSSTSnapshot(csst *csstModel.ClassSectionSubjectTeacherModel, role string) CSSTSnapshot {
	if csst == nil {
		return CSSTSnapshot{}
	}

	return CSSTSnapshot{
		ClassSectionSubjectTeacherID:               csst.ClassSectionSubjectTeacherID,
		ClassSectionSubjectTeacherRole:             role,
		ClassSectionSubjectTeacherIsActive:         csst.ClassSectionSubjectTeacherIsActive,
		ClassSectionSubjectTeacherClassSubjectID:   csst.ClassSectionSubjectTeacherClassSubjectID,
		ClassSectionSubjectTeacherSubjectID:        csst.ClassSectionSubjectTeacherSubjectID,
		ClassSectionSubjectTeacherSubjectNameCache: csst.ClassSectionSubjectTeacherSubjectNameCache,
		ClassSectionSubjectTeacherSubjectSlugCache: csst.ClassSectionSubjectTeacherSubjectSlugCache,

		ClassSectionSubjectTeacherAcademicTermID:        csst.ClassSectionSubjectTeacherAcademicTermID,
		ClassSectionSubjectTeacherAcademicTermNameCache: csst.ClassSectionSubjectTeacherAcademicTermNameCache,
		ClassSectionSubjectTeacherAcademicTermSlugCache: csst.ClassSectionSubjectTeacherAcademicTermSlugCache,

		ClassSectionSubjectTeacherMinPassingScoreClassSubjectCache: csst.ClassSectionSubjectTeacherMinPassingScoreClassSubjectCache,
		ClassSectionSubjectTeacherMinPassingScore:                  csst.ClassSectionSubjectTeacherMinPassingScore,

		ClassSectionSubjectTeacherTotalAttendance:          csst.ClassSectionSubjectTeacherTotalAttendance,
		ClassSectionSubjectTeacherTotalMeetingsTarget:      csst.ClassSectionSubjectTeacherTotalMeetingsTarget,
		ClassSectionSubjectTeacherTotalAssessments:         csst.ClassSectionSubjectTeacherTotalAssessments,
		ClassSectionSubjectTeacherTotalAssessmentsGraded:   csst.ClassSectionSubjectTeacherTotalAssessmentsGraded,
		ClassSectionSubjectTeacherTotalAssessmentsUngraded: csst.ClassSectionSubjectTeacherTotalAssessmentsUngraded,
		ClassSectionSubjectTeacherTotalStudentsPassed:      csst.ClassSectionSubjectTeacherTotalStudentsPassed,

		ClassSectionSubjectTeacherQuotaTaken: csst.ClassSectionSubjectTeacherQuotaTaken,
	}
}

/* ============================================================
   CSST → GURU
   Disimpan di: school_teachers.school_teacher_csst (JSONB)
   ============================================================ */

// panggil saat CSST di-assign ke guru tertentu
func AddCSSTToTeacher(
	tx *gorm.DB,
	schoolID uuid.UUID,
	teacherID uuid.UUID,
	csst *csstModel.ClassSectionSubjectTeacherModel,
	role string, // misal: "main" | "assistant"
) error {
	if csst == nil {
		return nil
	}

	var t stModel.SchoolTeacherModel
	if err := tx.
		Where("school_teacher_id = ? AND school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL",
			teacherID, schoolID).
		First(&t).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// guru belum ada record → jangan ganggu proses utama
			return nil
		}
		return err
	}

	items := decodeCSSTSnapshots(t.SchoolTeacherCSST)

	found := false
	for i := range items {
		if items[i].ClassSectionSubjectTeacherID == csst.ClassSectionSubjectTeacherID &&
			items[i].ClassSectionSubjectTeacherRole == role {
			// update snapshot existing
			items[i] = buildCSSTSnapshot(csst, role)
			found = true
			break
		}
	}

	if !found {
		// baru ⇒ append + inc stats
		items = append(items, buildCSSTSnapshot(csst, role))
		t.SchoolTeacherTotalClassSectionSubjectTeachers++
		if csst.ClassSectionSubjectTeacherIsActive {
			t.SchoolTeacherTotalClassSectionSubjectTeachersActive++
		}
	}

	t.SchoolTeacherCSST = encodeCSSTSnapshots(items)

	return tx.Save(&t).Error
}

// panggil saat CSST di-unassign dari guru
func RemoveCSSTFromTeacher(
	tx *gorm.DB,
	schoolID uuid.UUID,
	teacherID uuid.UUID,
	csst *csstModel.ClassSectionSubjectTeacherModel,
	role string,
) error {
	if csst == nil {
		return nil
	}

	var t stModel.SchoolTeacherModel
	if err := tx.
		Where("school_teacher_id = ? AND school_teacher_school_id = ? AND school_teacher_deleted_at IS NULL",
			teacherID, schoolID).
		First(&t).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	items := decodeCSSTSnapshots(t.SchoolTeacherCSST)

	newItems := make([]CSSTSnapshot, 0, len(items))
	removed := false
	for _, it := range items {
		if it.ClassSectionSubjectTeacherID == csst.ClassSectionSubjectTeacherID &&
			it.ClassSectionSubjectTeacherRole == role {
			removed = true
			continue
		}
		newItems = append(newItems, it)
	}

	if removed {
		if t.SchoolTeacherTotalClassSectionSubjectTeachers > 0 {
			t.SchoolTeacherTotalClassSectionSubjectTeachers--
		}
		if csst.ClassSectionSubjectTeacherIsActive &&
			t.SchoolTeacherTotalClassSectionSubjectTeachersActive > 0 {
			t.SchoolTeacherTotalClassSectionSubjectTeachersActive--
		}
	}

	t.SchoolTeacherCSST = encodeCSSTSnapshots(newItems)

	return tx.Save(&t).Error
}

/* ============================================================
   CSST → SISWA
   Disimpan di:
   school_students.school_student_class_section_subject_teachers (JSONB)
   ============================================================ */

// panggil saat CSST di-assign ke siswa (enrol mapel)
func AddCSSTToStudent(
	tx *gorm.DB,
	schoolID uuid.UUID,
	studentID uuid.UUID,
	csst *csstModel.ClassSectionSubjectTeacherModel,
	role string, // misal: "member" | "leader"
) error {
	if csst == nil {
		return nil
	}

	var s stModel.SchoolStudentModel
	if err := tx.
		Where("school_student_id = ? AND school_student_school_id = ? AND school_student_deleted_at IS NULL",
			studentID, schoolID).
		First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	items := decodeCSSTSnapshots(s.SchoolStudentClassSectionSubjectTeachers)

	found := false
	for i := range items {
		if items[i].ClassSectionSubjectTeacherID == csst.ClassSectionSubjectTeacherID &&
			items[i].ClassSectionSubjectTeacherRole == role {
			// update snapshot existing
			items[i] = buildCSSTSnapshot(csst, role)
			found = true
			break
		}
	}

	if !found {
		items = append(items, buildCSSTSnapshot(csst, role))
		s.SchoolStudentTotalClassSectionSubjectTeachers++
		if csst.ClassSectionSubjectTeacherIsActive {
			s.SchoolStudentTotalClassSectionSubjectTeachersActive++
		}
	}

	s.SchoolStudentClassSectionSubjectTeachers = encodeCSSTSnapshots(items)

	return tx.Save(&s).Error
}

// panggil saat CSST di-unassign dari siswa (drop mapel, dll)
func RemoveCSSTFromStudent(
	tx *gorm.DB,
	schoolID uuid.UUID,
	studentID uuid.UUID,
	csst *csstModel.ClassSectionSubjectTeacherModel,
	role string,
) error {
	if csst == nil {
		return nil
	}

	var s stModel.SchoolStudentModel
	if err := tx.
		Where("school_student_id = ? AND school_student_school_id = ? AND school_student_deleted_at IS NULL",
			studentID, schoolID).
		First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	items := decodeCSSTSnapshots(s.SchoolStudentClassSectionSubjectTeachers)

	newItems := make([]CSSTSnapshot, 0, len(items))
	removed := false
	for _, it := range items {
		if it.ClassSectionSubjectTeacherID == csst.ClassSectionSubjectTeacherID &&
			it.ClassSectionSubjectTeacherRole == role {
			removed = true
			continue
		}
		newItems = append(newItems, it)
	}

	if removed {
		if s.SchoolStudentTotalClassSectionSubjectTeachers > 0 {
			s.SchoolStudentTotalClassSectionSubjectTeachers--
		}
		if csst.ClassSectionSubjectTeacherIsActive &&
			s.SchoolStudentTotalClassSectionSubjectTeachersActive > 0 {
			s.SchoolStudentTotalClassSectionSubjectTeachersActive--
		}
	}

	s.SchoolStudentClassSectionSubjectTeachers = encodeCSSTSnapshots(newItems)

	return tx.Save(&s).Error
}
