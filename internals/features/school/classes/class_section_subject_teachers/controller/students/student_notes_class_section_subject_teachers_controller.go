// file: internals/features/school/classes/class_section_subject_teachers/controller/student_csst_notes_controller.go
package controller

import (
	"errors"
	"time"

	dto "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/dto"
	model "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PATCH /api/a/student-csst/:id/student-notes
func (ctl *StudentCSSTController) UpdateStudentNotes(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var path dto.IDParam
	if err := c.ParamsParser(&path); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var req dto.StudentCSSTUpdateNotesRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "payload tidak valid")
	}
	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var m model.StudentClassSectionSubjectTeacher
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("student_class_section_subject_teacher_id = ?", path.ID).
		Where("student_class_section_subject_teacher_school_id = ?", schoolID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusNotFound, "data tidak ditemukan")
		}
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	now := time.Now()

	// Notes: nil -> clear
	if req.Notes == nil {
		m.StudentClassSectionSubjectTeacherStudentNotes = nil
	} else {
		m.StudentClassSectionSubjectTeacherStudentNotes = req.Notes
	}
	m.StudentClassSectionSubjectTeacherStudentNotesUpdatedAt = &now

	if err := tx.Save(&m).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengupdate catatan siswa")
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal commit transaksi")
	}

	item := toStudentCSSTItem(&m)
	return helper.JsonUpdated(c, "catatan siswa berhasil diupdate", dto.StudentCSSTDetailResponse{Data: item})
}

// PATCH /api/a/student-csst/:id/homeroom-notes
func (ctl *StudentCSSTController) UpdateHomeroomNotes(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var path dto.IDParam
	if err := c.ParamsParser(&path); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var req dto.StudentCSSTUpdateNotesRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "payload tidak valid")
	}
	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var m model.StudentClassSectionSubjectTeacher
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("student_class_section_subject_teacher_id = ?", path.ID).
		Where("student_class_section_subject_teacher_school_id = ?", schoolID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusNotFound, "data tidak ditemukan")
		}
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	now := time.Now()

	if req.Notes == nil {
		m.StudentClassSectionSubjectTeacherHomeroomNotes = nil
	} else {
		m.StudentClassSectionSubjectTeacherHomeroomNotes = req.Notes
	}
	m.StudentClassSectionSubjectTeacherHomeroomNotesUpdatedAt = &now

	if err := tx.Save(&m).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengupdate catatan wali kelas")
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal commit transaksi")
	}

	item := toStudentCSSTItem(&m)
	return helper.JsonUpdated(c, "catatan wali kelas berhasil diupdate", dto.StudentCSSTDetailResponse{Data: item})
}

// PATCH /api/a/student-csst/:id/subject-teacher-notes
func (ctl *StudentCSSTController) UpdateSubjectTeacherNotes(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var path dto.IDParam
	if err := c.ParamsParser(&path); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var req dto.StudentCSSTUpdateNotesRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "payload tidak valid")
	}
	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var m model.StudentClassSectionSubjectTeacher
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("student_class_section_subject_teacher_id = ?", path.ID).
		Where("student_class_section_subject_teacher_school_id = ?", schoolID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusNotFound, "data tidak ditemukan")
		}
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	now := time.Now()

	if req.Notes == nil {
		m.StudentClassSectionSubjectTeacherSubjectTeacherNotes = nil
	} else {
		m.StudentClassSectionSubjectTeacherSubjectTeacherNotes = req.Notes
	}
	m.StudentClassSectionSubjectTeacherSubjectTeacherNotesUpdatedAt = &now

	if err := tx.Save(&m).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengupdate catatan guru mapel")
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal commit transaksi")
	}

	item := toStudentCSSTItem(&m)
	return helper.JsonUpdated(c, "catatan guru mapel berhasil diupdate", dto.StudentCSSTDetailResponse{Data: item})
}
