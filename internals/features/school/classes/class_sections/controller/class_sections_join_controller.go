package controller

import (
	"errors"
	"strings"
	"time"

	secModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- helpers: ambil section + guard DKM/Admin + tenant ---
func (ctrl *ClassSectionController) loadSectionForDKM(c *fiber.Ctx) (*secModel.ClassSectionModel, error) {
	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return nil, helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// 🔐 Selalu pakai school_id dari token/context
	schoolID, err := helperAuth.ResolveSchoolIDFromContext(c)
	if err != nil {
		return nil, err
	}

	var m secModel.ClassSectionModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where(
			"class_section_id = ? AND class_section_school_id = ? AND class_section_deleted_at IS NULL",
			sectionID, schoolID,
		).
		First(&m).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, helper.JsonError(c, fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return nil, helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data section")
	}

	// Guard akses: hanya DKM/Admin untuk school di token
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return nil, err
	}
	return &m, nil
}

// --- helpers: (re)generate student code (+hash) dan simpan ---
// NOTE: menyimpan plaintext student code ke kolom m.ClassSectionCode (sudah ada di model)
func (ctrl *ClassSectionController) ensureStudentJoinCode(tx *gorm.DB, m *secModel.ClassSectionModel) (string, error) {
	// jika sudah ada plaintext code tersimpan, pakai itu
	if m.ClassSectionCode != nil && strings.TrimSpace(*m.ClassSectionCode) != "" {
		return strings.TrimSpace(*m.ClassSectionCode), nil
	}

	// buat baru
	plain, err := buildSectionJoinCode(m.ClassSectionSlug, m.ClassSectionID)
	if err != nil {
		return "", fiber.NewError(fiber.StatusInternalServerError, "Gagal membangun student join code")
	}
	hashed, err := bcryptHash(plain)
	if err != nil {
		return "", fiber.NewError(fiber.StatusInternalServerError, "Gagal meng-hash student join code")
	}
	now := time.Now()
	m.ClassSectionCode = &plain
	m.ClassSectionStudentCodeHash = hashed
	m.ClassSectionStudentCodeSetAt = &now

	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_section_id = ?", m.ClassSectionID).
		Updates(map[string]any{
			"class_section_code":                plain,
			"class_section_student_code_hash":   hashed,
			"class_section_student_code_set_at": now,
		}).Error; err != nil {
		return "", fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan student join code")
	}
	return plain, nil
}

// --- helpers: (re)generate teacher code (+hash) dan simpan ---
// NOTE: plaintext teacher code TIDAK disimpan di DB; dikembalikan di response sekali saat generate/rotate
func (ctrl *ClassSectionController) rotateTeacherJoinCode(tx *gorm.DB, m *secModel.ClassSectionModel) (string, error) {
	plain, err := buildSectionJoinCode(m.ClassSectionSlug+"-t", m.ClassSectionID)
	if err != nil {
		return "", fiber.NewError(fiber.StatusInternalServerError, "Gagal membangun teacher join code")
	}
	hashed, err := bcryptHash(plain)
	if err != nil {
		return "", fiber.NewError(fiber.StatusInternalServerError, "Gagal meng-hash teacher join code")
	}
	now := time.Now()
	m.ClassSectionTeacherCodeHash = hashed
	m.ClassSectionTeacherCodeSetAt = &now

	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_section_id = ?", m.ClassSectionID).
		Updates(map[string]any{
			"class_section_teacher_code_hash":   hashed,
			"class_section_teacher_code_set_at": now,
		}).Error; err != nil {
		return "", fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan teacher join code")
	}
	return plain, nil
}

// GET /admin/class-sections/:id/join-code/student
func (ctrl *ClassSectionController) GetStudentJoinCode(c *fiber.Ctx) error {
	m, err := ctrl.loadSectionForDKM(c)
	if err != nil {
		return err // sudah JSON error di helper
	}

	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	code, er := ctrl.ensureStudentJoinCode(tx, m)
	if er != nil {
		_ = tx.Rollback()
		return er
	}
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "Berhasil mengambil kode join siswa", fiber.Map{
		"class_section_id": m.ClassSectionID,
		"role":             "student",
		"code":             code, // plaintext student code
		"set_at":           m.ClassSectionStudentCodeSetAt,
	})
}

// GET /admin/class-sections/:id/join-code/teacher
func (ctrl *ClassSectionController) GetTeacherJoinCode(c *fiber.Ctx) error {
	m, err := ctrl.loadSectionForDKM(c)
	if err != nil {
		return err
	}

	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	code, er := ctrl.rotateTeacherJoinCode(tx, m)
	if er != nil {
		_ = tx.Rollback()
		return er
	}
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonOK(c, "Berhasil membuat kode join pengajar (baru)", fiber.Map{
		"class_section_id": m.ClassSectionID,
		"role":             "teacher",
		"code":             code, // plaintext teacher code (one-time show)
		"set_at":           m.ClassSectionTeacherCodeSetAt,
		"note":             "Simpan kode ini. Plaintext tidak disimpan di server.",
	})
}

// GET /admin/class-sections/:id/join-codes
// Optional: ?rotate_teacher=1
func (ctrl *ClassSectionController) GetJoinCodes(c *fiber.Ctx) error {
	m, err := ctrl.loadSectionForDKM(c)
	if err != nil {
		return err
	}
	rotateTeacher := strings.TrimSpace(c.Query("rotate_teacher")) == "1"

	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	studentCode, er := ctrl.ensureStudentJoinCode(tx, m)
	if er != nil {
		_ = tx.Rollback()
		return er
	}

	var teacherCode string
	if rotateTeacher {
		tc, er2 := ctrl.rotateTeacherJoinCode(tx, m)
		if er2 != nil {
			_ = tx.Rollback()
			return er2
		}
		teacherCode = tc
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	resp := fiber.Map{
		"class_section_id": m.ClassSectionID,
		"student": fiber.Map{
			"code":   studentCode,
			"set_at": m.ClassSectionStudentCodeSetAt,
		},
	}
	if rotateTeacher {
		resp["teacher"] = fiber.Map{
			"code":   teacherCode,
			"set_at": m.ClassSectionTeacherCodeSetAt,
			"note":   "Simpan kode ini. Plaintext tidak disimpan di server.",
		}
	} else {
		resp["teacher"] = fiber.Map{
			"code": nil,
			"note": "Tambahkan ?rotate_teacher=1 untuk membuat & menampilkan kode teacher baru.",
		}
	}

	return helper.JsonOK(c, "Berhasil mengambil kode join", resp)
}

// POST /admin/class-sections/:id/join-code/student/rotate
func (ctrl *ClassSectionController) RotateStudentJoinCode(c *fiber.Ctx) error {
	m, err := ctrl.loadSectionForDKM(c)
	if err != nil {
		return err
	}
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// force new code (abaikan yang lama)
	plain, er := buildSectionJoinCode(m.ClassSectionSlug, m.ClassSectionID)
	if er != nil {
		_ = tx.Rollback()
		return er
	}
	hashed, er := bcryptHash(plain)
	if er != nil {
		_ = tx.Rollback()
		return er
	}
	now := time.Now()
	if err := tx.Model(&secModel.ClassSectionModel{}).
		Where("class_section_id = ?", m.ClassSectionID).
		Updates(map[string]any{
			"class_section_code":                plain,
			"class_section_student_code_hash":   hashed,
			"class_section_student_code_set_at": now,
		}).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan student join code")
	}
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "Student join code berhasil diganti", fiber.Map{
		"class_section_id": m.ClassSectionID,
		"role":             "student",
		"code":             plain,
	})
}

// POST /admin/class-sections/:id/join-code/teacher/rotate
func (ctrl *ClassSectionController) RotateTeacherJoinCode(c *fiber.Ctx) error {
	m, err := ctrl.loadSectionForDKM(c)
	if err != nil {
		return err
	}
	tx := ctrl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	plain, er := ctrl.rotateTeacherJoinCode(tx, m)
	if er != nil {
		_ = tx.Rollback()
		return er
	}
	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	return helper.JsonOK(c, "Teacher join code berhasil diganti", fiber.Map{
		"class_section_id": m.ClassSectionID,
		"role":             "teacher",
		"code":             plain,
		"note":             "Simpan kode ini. Plaintext tidak disimpan di server.",
	})
}
