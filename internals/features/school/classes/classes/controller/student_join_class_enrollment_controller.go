// file: internals/features/school/classes/class_enrollments/controller/join_section.go
package controller

import (
	"errors"
	"log"
	"strings"

	sectionModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	dto "madinahsalam_backend/internals/features/school/classes/classes/dto"
	enrollModel "madinahsalam_backend/internals/features/school/classes/classes/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	studentModel "madinahsalam_backend/internals/features/lembaga/school_yayasans/teachers_students/model"
	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	snapsvc "madinahsalam_backend/internals/features/users/users/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// POST /api/u/classes/my-enrollments/:id/join-section
func (ctl *StudentClassEnrollmentController) JoinSectionCSST(c *fiber.Ctx) error {
	// ========== school context dari TOKEN ==========
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err // helper sudah JsonError
	}

	// Hanya murid school ini
	if err := helperAuth.EnsureStudentSchool(c, schoolID); err != nil {
		return err
	}

	// Ambil student_id dari token
	studentID, err := helperAuth.GetPrimarySchoolStudentID(c)
	if err != nil || studentID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Konteks murid tidak ditemukan")
	}

	// Enrollment ID dari path
	rawID := strings.TrimSpace(c.Params("id"))
	enrollmentID, err := uuid.Parse(rawID)
	if err != nil || enrollmentID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "enrollment_id path tidak valid")
	}

	// Body: class_section_id
	var body dto.JoinClassSectionRequest
	if err := c.BodyParser(&body); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "invalid body")
	}
	if body.ClassSectionID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "class_section_id wajib")
	}

	// DEBUG: context awal
	log.Printf("[JoinSectionCSST] school_id=%s student_id=%s enrollment_id=%s class_section_id=%s",
		schoolID, studentID, enrollmentID, body.ClassSectionID)

	// ========== TX ==========
	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to start transaction")
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[JoinSectionCSST] panic: %+v, rollback tx", r)
			tx.Rollback()
			panic(r)
		}
	}()

	// 1) Ambil enrollment (FOR UPDATE) — pastikan milik murid ini & school ini
	var enr enrollModel.StudentClassEnrollmentModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("student_class_enrollments_id = ?", enrollmentID).
		Where("student_class_enrollments_school_id = ?", schoolID).
		Where("student_class_enrollments_school_student_id = ?", studentID).
		First(&enr).Error; err != nil {

		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[JoinSectionCSST] enrollment not found: enrollment_id=%s school_id=%s student_id=%s",
				enrollmentID, schoolID, studentID)
			return helper.JsonError(c, fiber.StatusNotFound, "Enrollment tidak ditemukan")
		}
		log.Printf("[JoinSectionCSST] failed load enrollment: err=%v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to load enrollment")
	}

	log.Printf("[JoinSectionCSST] loaded enrollment: id=%s class_id=%s status=%s",
		enr.StudentClassEnrollmentsID,
		enr.StudentClassEnrollmentsClassID,
		enr.StudentClassEnrollmentsStatus)

	// Sudah punya class section? → tolak (hanya boleh satu)
	if enr.StudentClassEnrollmentsClassSectionID != nil && *enr.StudentClassEnrollmentsClassSectionID != uuid.Nil {
		tx.Rollback()
		log.Printf("[JoinSectionCSST] reject: enrollment already has class_section_id=%s",
			*enr.StudentClassEnrollmentsClassSectionID)
		return helper.JsonError(c, fiber.StatusBadRequest, "Anda sudah bergabung ke class section")
	}

	// (opsional) hanya boleh kalau status accepted
	if enr.StudentClassEnrollmentsStatus != enrollModel.ClassEnrollmentAccepted {
		tx.Rollback()
		log.Printf("[JoinSectionCSST] reject: enrollment status=%s not accepted",
			enr.StudentClassEnrollmentsStatus)
		return helper.JsonError(c, fiber.StatusBadRequest, "Enrollment belum berstatus accepted")
	}

	// 2) Ambil class section (FOR UPDATE)
	var sec sectionModel.ClassSectionModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_section_id = ?", body.ClassSectionID).
		Where("class_section_school_id = ?", schoolID).
		Where("class_section_deleted_at IS NULL").
		First(&sec).Error; err != nil {

		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[JoinSectionCSST] class section not found: id=%s school_id=%s",
				body.ClassSectionID, schoolID)
			return helper.JsonError(c, fiber.StatusNotFound, "Class section tidak ditemukan")
		}
		log.Printf("[JoinSectionCSST] failed load class section: err=%v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to load class section")
	}

	log.Printf("[JoinSectionCSST] loaded section: id=%s class_id=%v enroll_mode=%s quota_taken=%d quota_total=%v",
		sec.ClassSectionID,
		sec.ClassSectionClassID,
		sec.ClassSectionSubjectTeachersEnrollmentMode,
		sec.ClassSectionQuotaTaken,
		sec.ClassSectionQuotaTotal)

	// Pastikan section ini memang untuk class yang sama (jika terisi)
	if sec.ClassSectionClassID != nil && *sec.ClassSectionClassID != enr.StudentClassEnrollmentsClassID {
		tx.Rollback()
		log.Printf("[JoinSectionCSST] reject: section_class_id=%s != enrollment_class_id=%s",
			*sec.ClassSectionClassID, enr.StudentClassEnrollmentsClassID)
		return helper.JsonError(c, fiber.StatusBadRequest, "Class section tidak sesuai dengan kelas enrollment")
	}

	// Mode enrollment subject-teachers: kalau full assigned → jangan boleh self-join section
	if sec.ClassSectionSubjectTeachersEnrollmentMode == sectionModel.EnrollAssigned {
		tx.Rollback()
		log.Printf("[JoinSectionCSST] reject: section enrollment_mode=%s (EnrollAssigned)",
			sec.ClassSectionSubjectTeachersEnrollmentMode)
		return helper.JsonError(c, fiber.StatusBadRequest, "Class section ini tidak mengizinkan self-enroll")
	}

	// Kapasitas section penuh? (pakai quota baru)
	if sec.ClassSectionQuotaTotal != nil && *sec.ClassSectionQuotaTotal > 0 &&
		sec.ClassSectionQuotaTaken >= *sec.ClassSectionQuotaTotal {

		tx.Rollback()
		log.Printf("[JoinSectionCSST] reject: section full, quota_taken=%d quota_total=%d",
			sec.ClassSectionQuotaTaken, *sec.ClassSectionQuotaTotal)
		return helper.JsonError(c, fiber.StatusBadRequest, "Class section sudah penuh")
	}

	// ==== 3a) Build cache user_profile + student_code murid (sekali saja, dipakai di section & CSST) ====
	var (
		userProfileSnap *snapsvc.UserProfileCache
		studentCode     *string
	)

	{
		var stu studentModel.SchoolStudentModel
		if err := tx.
			Where("school_student_id = ?", studentID).
			Where("school_student_school_id = ?", schoolID).
			Where("school_student_deleted_at IS NULL").
			First(&stu).Error; err != nil {

			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("[JoinSectionCSST] school_student not found for cache: student_id=%s school_id=%s",
					studentID, schoolID)
			} else {
				log.Printf("[JoinSectionCSST] failed load school_student for cache: err=%v", err)
			}
		} else {
			// simpan student_code cache (kalau ada)
			if stu.SchoolStudentCode != nil {
				studentCode = stu.SchoolStudentCode
			}

			if stu.SchoolStudentUserProfileID != uuid.Nil {
				snap, errSnap := snapsvc.BuildUserProfileCacheByProfileID(
					c.Context(),
					tx,
					stu.SchoolStudentUserProfileID,
				)
				if errSnap != nil && !errors.Is(errSnap, gorm.ErrRecordNotFound) {
					log.Printf("[JoinSectionCSST] failed build user_profile cache: profile_id=%s err=%v",
						stu.SchoolStudentUserProfileID, errSnap)
				} else if errSnap == nil {
					userProfileSnap = snap
					log.Printf("[JoinSectionCSST] loaded user_profile cache for student: user_id=%s name=%s",
						snap.ID, snap.Name)
				} else {
					log.Printf("[JoinSectionCSST] no user_profile cache found for profile_id=%s",
						stu.SchoolStudentUserProfileID)
				}
			} else {
				log.Printf("[JoinSectionCSST] school_student has no user_profile_id for cache: student_id=%s", studentID)
			}
		}
	}

	// 3b) Update enrollment: set class_section + caches
	enr.StudentClassEnrollmentsClassSectionID = &sec.ClassSectionID
	enr.StudentClassEnrollmentsClassSectionNameCache = &sec.ClassSectionName
	enr.StudentClassEnrollmentsClassSectionSlugCache = &sec.ClassSectionSlug

	if err := tx.Save(&enr).Error; err != nil {
		tx.Rollback()
		log.Printf("[JoinSectionCSST] failed to update enrollment with section: err=%v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to update enrollment")
	}

	log.Printf("[JoinSectionCSST] enrollment updated with section_id=%s", sec.ClassSectionID)

	// 4) Tambahkan row di student_class_sections (kalau belum ada)
	var existing sectionModel.StudentClassSection
	err = tx.
		Where("student_class_section_school_id = ?", schoolID).
		Where("student_class_section_school_student_id = ?", studentID).
		Where("student_class_section_section_id = ?", sec.ClassSectionID).
		Where("student_class_section_deleted_at IS NULL").
		First(&existing).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		log.Printf("[JoinSectionCSST] failed to check student_class_sections: err=%v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to check student_class_sections")
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[JoinSectionCSST] creating student_class_section row: school_id=%s student_id=%s section_id=%s",
			schoolID, studentID, sec.ClassSectionID)

		scs := sectionModel.StudentClassSection{
			StudentClassSectionSchoolStudentID:  studentID,
			StudentClassSectionSchoolID:         schoolID,
			StudentClassSectionSectionID:        sec.ClassSectionID,
			StudentClassSectionSectionSlugCache: sec.ClassSectionSlug,
			// status & assigned_at pakai default DB (active + current_date)
		}

		// Isi cache user_profile ke student_class_sections (kalau ada)
		if userProfileSnap != nil {
			name := userProfileSnap.Name
			scs.StudentClassSectionUserProfileNameCache = &name
			scs.StudentClassSectionUserProfileAvatarURLCache = userProfileSnap.AvatarURL
			scs.StudentClassSectionUserProfileWhatsappURLCache = userProfileSnap.WhatsappURL
			scs.StudentClassSectionUserProfileParentNameCache = userProfileSnap.ParentName
			scs.StudentClassSectionUserProfileParentWhatsappURLCache = userProfileSnap.ParentWhatsappURL
			scs.StudentClassSectionUserProfileGenderCache = userProfileSnap.Gender
			log.Printf("[JoinSectionCSST] filled student_class_section user_profile cache: name=%s", name)
		}

		// Isi student_code cache kalau ada
		if studentCode != nil {
			scs.StudentClassSectionStudentCodeCache = studentCode
			log.Printf("[JoinSectionCSST] filled student_class_section student_code_cache: code=%s", *studentCode)
		}

		if err := tx.Create(&scs).Error; err != nil {
			tx.Rollback()
			log.Printf("[JoinSectionCSST] failed to create student_class_section: err=%v", err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "failed to create student_class_section")
		}
	} else {
		log.Printf("[JoinSectionCSST] student_class_section already exists: id=%s", existing.StudentClassSectionID)
	}

	// 4b) AUTO JOIN ke semua ClassSectionSubjectTeacher (CSST) di section ini
	var cssts []csstModel.ClassSectionSubjectTeacherModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_section_subject_teacher_school_id = ?", schoolID).
		Where("class_section_subject_teacher_class_section_id = ?", sec.ClassSectionID).
		Where("class_section_subject_teacher_is_active = ?", true).
		Where("class_section_subject_teacher_deleted_at IS NULL").
		Find(&cssts).Error; err != nil {

		tx.Rollback()
		log.Printf("[JoinSectionCSST] failed to load CSST list: err=%v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to load subject teachers")
	}

	log.Printf("[JoinSectionCSST] loaded %d CSST(s) for section_id=%s", len(cssts), sec.ClassSectionID)

	for _, csst := range cssts {
		log.Printf("[JoinSectionCSST] processing CSST id=%s enrolled_count=%d capacity=%v",
			csst.ClassSectionSubjectTeacherID,
			csst.ClassSectionSubjectTeacherEnrolledCount,
			csst.ClassSectionSubjectTeacherCapacity)

		// Cek kapasitas CSST (kalau di-set & penuh → skip CSST ini saja)
		if csst.ClassSectionSubjectTeacherCapacity != nil && *csst.ClassSectionSubjectTeacherCapacity > 0 {
			if csst.ClassSectionSubjectTeacherEnrolledCount >= *csst.ClassSectionSubjectTeacherCapacity {
				log.Printf("[JoinSectionCSST] skip CSST id=%s: full (enrolled=%d capacity=%d)",
					csst.ClassSectionSubjectTeacherID,
					csst.ClassSectionSubjectTeacherEnrolledCount,
					*csst.ClassSectionSubjectTeacherCapacity)
				continue
			}
		}

		// Cek apakah sudah ada mapping student ↔ CSST
		var link csstModel.StudentClassSectionSubjectTeacher
		err := tx.
			Where("student_class_section_subject_teacher_school_id = ?", schoolID).
			Where("student_class_section_subject_teacher_student_id = ?", studentID).
			Where("student_class_section_subject_teacher_csst_id = ?", csst.ClassSectionSubjectTeacherID).
			Where("student_class_section_subject_teacher_deleted_at IS NULL").
			First(&link).Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			log.Printf("[JoinSectionCSST] failed to check CSST mapping for csst_id=%s: err=%v",
				csst.ClassSectionSubjectTeacherID, err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "failed to check CSST mapping")
		}

		// Kalau sudah ada mapping → skip
		if err == nil {
			log.Printf("[JoinSectionCSST] mapping already exists for student_id=%s csst_id=%s (mapping_id=%s)",
				studentID, csst.ClassSectionSubjectTeacherID, link.StudentClassSectionSubjectTeacherID)
			continue
		}

		// Kalau belum ada → buat baru
		log.Printf("[JoinSectionCSST] creating CSST mapping: student_id=%s csst_id=%s",
			studentID, csst.ClassSectionSubjectTeacherID)

		newLink := csstModel.StudentClassSectionSubjectTeacher{
			StudentClassSectionSubjectTeacherSchoolID:  schoolID,
			StudentClassSectionSubjectTeacherStudentID: studentID,
			StudentClassSectionSubjectTeacherCSSTID:    csst.ClassSectionSubjectTeacherID,
			// is_active + created_at pakai default DB
		}

		// Isi cache user_profile ke student_class_section_subject_teachers (kalau ada)
		if userProfileSnap != nil {
			name := userProfileSnap.Name
			newLink.StudentClassSectionSubjectTeacherUserProfileNameCache = &name
			newLink.StudentClassSectionSubjectTeacherUserProfileAvatarURLCache = userProfileSnap.AvatarURL
			newLink.StudentClassSectionSubjectTeacherUserProfileWhatsappURLCache = userProfileSnap.WhatsappURL
			newLink.StudentClassSectionSubjectTeacherUserProfileParentNameCache = userProfileSnap.ParentName
			newLink.StudentClassSectionSubjectTeacherUserProfileParentWhatsappURLCache = userProfileSnap.ParentWhatsappURL
			newLink.StudentClassSectionSubjectTeacherUserProfileGenderCache = userProfileSnap.Gender

			log.Printf("[JoinSectionCSST] filled CSST mapping user_profile cache: name=%s", name)
		}

		// Isi student_code cache kalau ada
		if studentCode != nil {
			newLink.StudentClassSectionSubjectTeacherStudentCodeCache = studentCode
			log.Printf("[JoinSectionCSST] filled CSST mapping student_code_cache: code=%s", *studentCode)
		}

		if err := tx.Create(&newLink).Error; err != nil {
			tx.Rollback()
			log.Printf("[JoinSectionCSST] failed to create CSST mapping for csst_id=%s: err=%v",
				csst.ClassSectionSubjectTeacherID, err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "failed to create CSST enrollment")
		}

		// Increment enrolled_count di CSST
		if err := tx.Model(&csstModel.ClassSectionSubjectTeacherModel{}).
			Where("class_section_subject_teacher_id = ?", csst.ClassSectionSubjectTeacherID).
			Update("class_section_subject_teacher_enrolled_count",
				gorm.Expr("class_section_subject_teacher_enrolled_count + 1")).Error; err != nil {

			tx.Rollback()
			log.Printf("[JoinSectionCSST] failed to increment csst_enrolled_count for csst_id=%s: err=%v",
				csst.ClassSectionSubjectTeacherID, err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "failed to update CSST counter")
		}

		log.Printf("[JoinSectionCSST] CSST mapping + counter updated for csst_id=%s", csst.ClassSectionSubjectTeacherID)
	}

	// 5) Increment counter quota_taken untuk class_section
	if err := tx.Model(&sec).
		Update("class_section_quota_taken", gorm.Expr("class_section_quota_taken + 1")).Error; err != nil {

		tx.Rollback()
		log.Printf("[JoinSectionCSST] failed to increment class_section_quota_taken for section_id=%s: err=%v",
			sec.ClassSectionID, err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to update class section counter")
	}

	log.Printf("[JoinSectionCSST] class_section_quota_taken incremented for section_id=%s", sec.ClassSectionID)

	if err := tx.Commit().Error; err != nil {
		log.Printf("[JoinSectionCSST] failed to commit transaction: err=%v", err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "failed to commit")
	}

	log.Printf("[JoinSectionCSST] SUCCESS: student_id=%s joined section_id=%s (enrollment_id=%s)",
		studentID, sec.ClassSectionID, enrollmentID)

	// Balikkan enrollment versi DTO full (biar FE langsung punya data terbaru)
	resp := dto.FromModelStudentClassEnrollment(&enr)
	return helper.JsonOK(c, "joined class section", resp)
}
