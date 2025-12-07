// file: internals/features/school/sectionsubjectteachers/controller/student_class_section_subject_teacher_controller.go
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

/* =========================================================
   Controller
========================================================= */

type StudentCSSTController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewStudentCSSTController(db *gorm.DB, v interface{ Struct(any) error }) *StudentCSSTController {
	return &StudentCSSTController{DB: db, Validator: v}
}

/* =========================================================
   Mapper: Model → DTO (1:1 dengan model)
========================================================= */

func toStudentCSSTItem(m *model.StudentClassSectionSubjectTeacherModel) dto.StudentCSSTItem {
	var deletedAt *time.Time
	if m.StudentCSSTDeletedAt.Valid {
		t := m.StudentCSSTDeletedAt.Time
		deletedAt = &t
	}

	return dto.StudentCSSTItem{
		StudentCSSTID:       m.StudentCSSTID,
		StudentCSSTSchoolID: m.StudentCSSTSchoolID,

		StudentCSSTStudentID: m.StudentCSSTStudentID,
		StudentCSSTCSSTID:    m.StudentCSSTCSSTID,

		StudentCSSTIsActive: m.StudentCSSTIsActive,
		StudentCSSTFrom:     m.StudentCSSTFrom,
		StudentCSSTTo:       m.StudentCSSTTo,

		StudentCSSTScoreTotal:    m.StudentCSSTScoreTotal,
		StudentCSSTScoreMaxTotal: m.StudentCSSTScoreMaxTotal,
		StudentCSSTScorePercent:  m.StudentCSSTScorePercent,
		StudentCSSTGradeLetter:   m.StudentCSSTGradeLetter,
		StudentCSSTGradePoint:    m.StudentCSSTGradePoint,
		StudentCSSTIsPassed:      m.StudentCSSTIsPassed,

		// 🆕 diselaraskan dengan nama field di model + DTO + SQL
		StudentCSSTUserProfileNameCache:        m.StudentCSSTUserProfileNameCache,
		StudentCSSTUserProfileAvatarURLCache:   m.StudentCSSTUserProfileAvatarURLCache,
		StudentCSSTUserProfileWAURLCache:       m.StudentCSSTUserProfileWAURLCache,
		StudentCSSTUserProfileParentNameCache:  m.StudentCSSTUserProfileParentNameCache,
		StudentCSSTUserProfileParentWAURLCache: m.StudentCSSTUserProfileParentWAURLCache,
		StudentCSSTUserProfileGenderCache:      m.StudentCSSTUserProfileGenderCache,
		StudentCSSTSchoolStudentCodeCache:      m.StudentCSSTSchoolStudentCodeCache,

		StudentCSSTEditsHistory: m.StudentCSSTEditsHistory,

		// NOTES
		StudentCSSTStudentNotes:                 m.StudentCSSTStudentNotes,
		StudentCSSTStudentNotesUpdatedAt:        m.StudentCSSTStudentNotesUpdatedAt,
		StudentCSSTHomeroomNotes:                m.StudentCSSTHomeroomNotes,
		StudentCSSTHomeroomNotesUpdatedAt:       m.StudentCSSTHomeroomNotesUpdatedAt,
		StudentCSSTSubjectTeacherNotes:          m.StudentCSSTSubjectTeacherNotes,
		StudentCSSTSubjectTeacherNotesUpdatedAt: m.StudentCSSTSubjectTeacherNotesUpdatedAt,

		StudentCSSTSlug: m.StudentCSSTSlug,
		StudentCSSTMeta: m.StudentCSSTMeta,

		StudentCSSTCreatedAt: m.StudentCSSTCreatedAt,
		StudentCSSTUpdatedAt: m.StudentCSSTUpdatedAt,
		StudentCSSTDeletedAt: deletedAt,

		// Expanded relasi (Student/Section/ClassSubject/Teacher) sementara kosong.
		Student:      nil,
		Section:      nil,
		ClassSubject: nil,
		Teacher:      nil,
	}
}

/* =========================================================
   CREATE
========================================================= */

// POST /api/a/student-csst
func (ctl *StudentCSSTController) Create(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var req dto.StudentCSSTCreateRequest
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

	now := time.Now()

	m := model.StudentClassSectionSubjectTeacherModel{
		StudentCSSTSchoolID:  schoolID,
		StudentCSSTStudentID: req.StudentID,
		StudentCSSTCSSTID:    req.CSSTID,
	}

	if req.IsActive != nil {
		m.StudentCSSTIsActive = *req.IsActive
	}
	if req.From != nil {
		m.StudentCSSTFrom = req.From
	}
	if req.To != nil {
		m.StudentCSSTTo = req.To
	}

	// Notes (opsional) + updated_at
	if req.StudentNotes != nil {
		m.StudentCSSTStudentNotes = req.StudentNotes
		m.StudentCSSTStudentNotesUpdatedAt = &now
	}
	if req.HomeroomNotes != nil {
		m.StudentCSSTHomeroomNotes = req.HomeroomNotes
		m.StudentCSSTHomeroomNotesUpdatedAt = &now
	}
	if req.SubjectTeacherNotes != nil {
		m.StudentCSSTSubjectTeacherNotes = req.SubjectTeacherNotes
		m.StudentCSSTSubjectTeacherNotesUpdatedAt = &now
	}

	if err := tx.Create(&m).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal membuat mapping student-csst")
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal commit transaksi")
	}

	item := toStudentCSSTItem(&m)
	resp := dto.StudentCSSTCreateResponse{Data: item}
	return helper.JsonCreated(c, "mapping student-csst berhasil dibuat", resp)
}

/* =========================================================
   BULK CREATE
========================================================= */

// POST /api/a/student-csst/bulk
func (ctl *StudentCSSTController) BulkCreate(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var req dto.StudentCSSTBulkCreateRequest
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

	results := make([]dto.StudentCSSTBulkCreateResult, 0, len(req.Items))
	inserted := 0
	skipped := 0
	existingCount := 0

	for _, it := range req.Items {
		// cek duplikat by (school_id, student_id, csst_id)
		var existing model.StudentClassSectionSubjectTeacherModel
		err := tx.
			Where("student_csst_school_id = ?", schoolID).
			Where("student_csst_student_id = ?", it.StudentID).
			Where("student_csst_csst_id = ?", it.CSSTID).
			Where("student_csst_deleted_at IS NULL").
			First(&existing).Error

		if err == nil {
			// duplikat
			existingCount++
			if req.SkipDuplicates {
				skipped++
				res := dto.StudentCSSTBulkCreateResult{
					Item:      toStudentCSSTItem(&existing),
					ClientRef: it.ClientRef,
					Duplicate: true,
				}
				if req.ReturnExisting {
					results = append(results, res)
				}
				continue
			}
			// kalau tidak SkipDuplicates, biarkan DB unique constraint yang nge-blok (kalau ada)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal cek duplikat mapping student-csst")
		}

		now := time.Now()

		m := model.StudentClassSectionSubjectTeacherModel{
			StudentCSSTSchoolID:  schoolID,
			StudentCSSTStudentID: it.StudentID,
			StudentCSSTCSSTID:    it.CSSTID,
		}
		if it.IsActive != nil {
			m.StudentCSSTIsActive = *it.IsActive
		}
		if it.From != nil {
			m.StudentCSSTFrom = it.From
		}
		if it.To != nil {
			m.StudentCSSTTo = it.To
		}

		// Notes (opsional) + updated_at
		if it.StudentNotes != nil {
			m.StudentCSSTStudentNotes = it.StudentNotes
			m.StudentCSSTStudentNotesUpdatedAt = &now
		}
		if it.HomeroomNotes != nil {
			m.StudentCSSTHomeroomNotes = it.HomeroomNotes
			m.StudentCSSTHomeroomNotesUpdatedAt = &now
		}
		if it.SubjectTeacherNotes != nil {
			m.StudentCSSTSubjectTeacherNotes = it.SubjectTeacherNotes
			m.StudentCSSTSubjectTeacherNotesUpdatedAt = &now
		}

		if err := tx.Create(&m).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal bulk create mapping student-csst")
		}

		inserted++
		results = append(results, dto.StudentCSSTBulkCreateResult{
			Item:      toStudentCSSTItem(&m),
			ClientRef: it.ClientRef,
			Duplicate: false,
		})
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal commit transaksi")
	}

	resp := dto.StudentCSSTBulkCreateResponse{
		Results: results,
	}
	resp.Meta.Inserted = inserted
	resp.Meta.Skipped = skipped
	resp.Meta.Existing = existingCount

	return helper.JsonCreated(c, "bulk mapping student-csst berhasil diproses", resp)
}

/* =========================================================
   UPSERT (by school_id + csst_id + student_id)
========================================================= */

// POST /api/a/student-csst/upsert
func (ctl *StudentCSSTController) Upsert(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var req dto.StudentCSSTUpsertRequest
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

	now := time.Now()

	var m model.StudentClassSectionSubjectTeacherModel
	err = tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("student_csst_school_id = ?", schoolID).
		Where("student_csst_student_id = ?", req.StudentID).
		Where("student_csst_csst_id = ?", req.CSSTID).
		First(&m).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// create baru
		m = model.StudentClassSectionSubjectTeacherModel{
			StudentCSSTSchoolID:  schoolID,
			StudentCSSTStudentID: req.StudentID,
			StudentCSSTCSSTID:    req.CSSTID,
		}
		if req.IsActive != nil {
			m.StudentCSSTIsActive = *req.IsActive
		}
		if req.From != nil {
			m.StudentCSSTFrom = req.From
		}
		if req.To != nil {
			m.StudentCSSTTo = req.To
		}

		// Notes (opsional) + updated_at
		if req.StudentNotes != nil {
			m.StudentCSSTStudentNotes = req.StudentNotes
			m.StudentCSSTStudentNotesUpdatedAt = &now
		}
		if req.HomeroomNotes != nil {
			m.StudentCSSTHomeroomNotes = req.HomeroomNotes
			m.StudentCSSTHomeroomNotesUpdatedAt = &now
		}
		if req.SubjectTeacherNotes != nil {
			m.StudentCSSTSubjectTeacherNotes = req.SubjectTeacherNotes
			m.StudentCSSTSubjectTeacherNotesUpdatedAt = &now
		}

		if err := tx.Create(&m).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal membuat mapping student-csst")
		}
	} else if err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil mapping student-csst")
	} else {
		// update existing
		if req.IsActive != nil {
			m.StudentCSSTIsActive = *req.IsActive
		}
		if req.From != nil {
			m.StudentCSSTFrom = req.From
		}
		if req.To != nil {
			m.StudentCSSTTo = req.To
		}

		// Notes: di-upsert hanya kalau dikirim (non-nil)
		if req.StudentNotes != nil {
			m.StudentCSSTStudentNotes = req.StudentNotes
			m.StudentCSSTStudentNotesUpdatedAt = &now
		}
		if req.HomeroomNotes != nil {
			m.StudentCSSTHomeroomNotes = req.HomeroomNotes
			m.StudentCSSTHomeroomNotesUpdatedAt = &now
		}
		if req.SubjectTeacherNotes != nil {
			m.StudentCSSTSubjectTeacherNotes = req.SubjectTeacherNotes
			m.StudentCSSTSubjectTeacherNotesUpdatedAt = &now
		}

		if err := tx.Save(&m).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengupdate mapping student-csst")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal commit transaksi")
	}

	item := toStudentCSSTItem(&m)
	resp := dto.StudentCSSTCreateResponse{Data: item}
	return helper.JsonOK(c, "upsert mapping student-csst berhasil", resp)
}

/* =========================================================
   TOGGLE ACTIVE (single)
========================================================= */

// POST /api/a/student-csst/:id/toggle-active
func (ctl *StudentCSSTController) ToggleActive(c *fiber.Ctx) error {
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

	var req dto.StudentCSSTToggleActiveRequest
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

	var m model.StudentClassSectionSubjectTeacherModel
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("student_csst_id = ?", path.ID).
		Where("student_csst_school_id = ?", schoolID).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusNotFound, "data tidak ditemukan")
		}
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	m.StudentCSSTIsActive = req.IsActive

	if err := tx.Save(&m).Error; err != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengupdate is_active")
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal commit transaksi")
	}

	item := toStudentCSSTItem(&m)
	resp := dto.StudentCSSTDetailResponse{Data: item}
	return helper.JsonUpdated(c, "status aktif berhasil diubah", resp)
}

/* =========================================================
   BULK TOGGLE ACTIVE
========================================================= */

// POST /api/a/student-csst/bulk/toggle-active
func (ctl *StudentCSSTController) BulkToggleActive(c *fiber.Ctx) error {
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return err
	}
	if err := helperAuth.EnsureStaffSchool(c, schoolID); err != nil {
		return err
	}

	var req dto.StudentCSSTBulkToggleActiveRequest
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

	res := tx.
		Model(&model.StudentClassSectionSubjectTeacherModel{}).
		Where("student_csst_school_id = ?", schoolID).
		Where("student_csst_id IN ?", req.IDs).
		Updates(map[string]any{
			"student_csst_is_active": req.IsActive,
		})
	if res.Error != nil {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal mengupdate data")
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal commit transaksi")
	}

	return helper.JsonUpdated(c, "status aktif berhasil diubah (bulk)", dto.AffectedResponse{
		Affected: int(res.RowsAffected),
	})
}

/* =========================================================
   DELETE (soft / hard)
========================================================= */

// DELETE /api/a/student-csst/:id
func (ctl *StudentCSSTController) Delete(c *fiber.Ctx) error {
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

	var req dto.StudentCSSTDeleteRequest
	_ = c.BodyParser(&req) // body opsional

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

	if req.Force {
		// hard delete
		if err := tx.
			Unscoped().
			Where("student_csst_id = ?", path.ID).
			Where("student_csst_school_id = ?", schoolID).
			Delete(&model.StudentClassSectionSubjectTeacherModel{}).Error; err != nil {

			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal hard delete data")
		}
	} else {
		// soft delete
		if err := tx.
			Where("student_csst_id = ?", path.ID).
			Where("student_csst_school_id = ?", schoolID).
			Delete(&model.StudentClassSectionSubjectTeacherModel{}).Error; err != nil {

			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "gagal delete data")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "gagal commit transaksi")
	}

	return helper.JsonDeleted(c, "mapping student-csst berhasil dihapus", dto.AffectedResponse{Affected: 1})
}
