// file: internals/features/school/classes/class_sections/controller/user_class_section_join_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	dto "masjidku_backend/internals/features/school/classes/class_sections/dto"
	model "masjidku_backend/internals/features/school/classes/class_sections/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
)

// Ptr mengembalikan pointer ke nilai T
func Ptr[T any](v T) *T { return &v }

func forUpdate() clause.Expression { return clause.Locking{Strength: "UPDATE"} }

// cek kode ke hash (bcrypt)
func verifyJoinCode(stored []byte, code string) bool {
	if len(stored) == 0 || strings.TrimSpace(code) == "" {
		return false
	}
	// bcrypt prefix
	if strings.HasPrefix(string(stored), "$2a$") || strings.HasPrefix(string(stored), "$2b$") || strings.HasPrefix(string(stored), "$2y$") {
		return bcrypt.CompareHashAndPassword(stored, []byte(code)) == nil
	}
	return false
}

type JoinAssignment string

const (
	AssignedAsTeacher   JoinAssignment = "teacher"
	AssignedAsAssistant JoinAssignment = "assistant"
)

// POST /u/class-sections/:id/join
func (ctl *UserClassSectionController) JoinByCode(c *fiber.Ctx) error {
	// Auth user (cukup user_id; masjid_id kita ambil dari section)
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	// Path param
	sectionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID section tidak valid")
	}

	// Body
	var req dto.ClassSectionJoinRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// TX + lock baris section
	tx := ctl.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var sec model.ClassSectionModel
	if err := tx.
		Clauses(forUpdate()).
		Where("class_section_id = ? AND class_section_deleted_at IS NULL", sectionID).
		First(&sec).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil section")
	}

	if !sec.ClassSectionIsActive {
		tx.Rollback()
		return helper.JsonError(c, fiber.StatusConflict, "Section tidak aktif")
	}

	now := time.Now()
	masjidID := sec.ClassSectionMasjidID // sumber kebenaran masjid_id = dari section

	switch req.Role {
	case dto.JoinRoleStudent:
		// 1) verifikasi kode student
		if !verifyJoinCode(sec.ClassSectionStudentCodeHash, req.Code) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusUnauthorized, "Kode salah atau sudah tidak berlaku")
		}

		// 2) dapatkan/buat masjid_student (unik: masjid_id + user_id)
		var masjidStudentID uuid.UUID
		type MasjidStudent struct {
			MasjidStudentID       uuid.UUID `gorm:"column:masjid_student_id"`
			MasjidStudentMasjidID uuid.UUID `gorm:"column:masjid_student_masjid_id"`
			MasjidStudentUserID   uuid.UUID `gorm:"column:masjid_student_user_id"`
		}
		var ms MasjidStudent
		err = tx.Table("masjid_students").
			Where("masjid_student_masjid_id = ? AND masjid_student_user_id = ?", masjidID, userID).
			First(&ms).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek status student")
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ms = MasjidStudent{
				MasjidStudentID:       uuid.New(),
				MasjidStudentMasjidID: masjidID,
				MasjidStudentUserID:   userID,
			}
			if err := tx.Table("masjid_students").Create(&ms).Error; err != nil {
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat profil student")
			}
		}
		masjidStudentID = ms.MasjidStudentID

		// 3) kapasitas
		if sec.ClassSectionCapacity != nil && *sec.ClassSectionCapacity > 0 &&
			sec.ClassSectionTotalStudents >= *sec.ClassSectionCapacity {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusConflict, "Kelas penuh")
		}

		// 4) belum terdaftar di section ini
		var exists int64
		if err := tx.Table("user_class_sections").
			Where("user_class_section_masjid_student_id = ? AND user_class_section_section_id = ? AND user_class_section_deleted_at IS NULL",
				masjidStudentID, sec.ClassSectionID).
			Count(&exists).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek keanggotaan")
		}
		if exists > 0 {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusConflict, "Sudah tergabung di section ini")
		}

		// 5) create user_class_section
		ucs := &model.UserClassSection{
			UserClassSectionID:              uuid.New(),
			UserClassSectionMasjidStudentID: masjidStudentID,
			UserClassSectionSectionID:       sec.ClassSectionID,
			UserClassSectionMasjidID:        masjidID,
			UserClassSectionStatus:          model.UserClassSectionActive,
			// fee_snapshot nullable → biarkan NULL saat create
			UserClassSectionAssignedAt: now, // kolom DATE; waktu akan terpotong di DB
			UserClassSectionCreatedAt:  now,
			UserClassSectionUpdatedAt:  now,
		}
		if err := tx.Create(ucs).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menambahkan ke section")
		}

		// 6) update counter
		if err := tx.Model(&sec).
			Where("class_section_id = ?", sec.ClassSectionID).
			UpdateColumn("class_section_total_students", gorm.Expr("class_section_total_students + 1")).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal update counter")
		}

		if err := tx.Commit().Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Transaksi gagal")
		}

		return helper.JsonOK(c, "Berhasil bergabung sebagai student", fiber.Map{
			"item": dto.ClassSectionJoinResponse{
				UserClassSection: Ptr(dto.FromModel(ucs)),
				ClassSectionID:   sec.ClassSectionID.String(),
			},
		})

	case dto.JoinRoleTeacher:
		// 1) verifikasi kode teacher
		if !verifyJoinCode(sec.ClassSectionTeacherCodeHash, req.Code) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusUnauthorized, "Kode salah atau sudah tidak berlaku")
		}

		// 2) dapatkan/buat masjid_teacher
		type MasjidTeacher struct {
			MasjidTeacherID       uuid.UUID `gorm:"column:masjid_teacher_id"`
			MasjidTeacherMasjidID uuid.UUID `gorm:"column:masjid_teacher_masjid_id"`
			MasjidTeacherUserID   uuid.UUID `gorm:"column:masjid_teacher_user_id"`
		}
		var mt MasjidTeacher
		err = tx.Table("masjid_teachers").
			Where("masjid_teacher_masjid_id = ? AND masjid_teacher_user_id = ?", masjidID, userID).
			First(&mt).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek status teacher")
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			mt = MasjidTeacher{
				MasjidTeacherID:       uuid.New(),
				MasjidTeacherMasjidID: masjidID,
				MasjidTeacherUserID:   userID,
			}
			if err := tx.Table("masjid_teachers").Create(&mt).Error; err != nil {
				tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat profil teacher")
			}
		}

		// 3) isi slot teacher/assistant
		assignAs := JoinAssignment("")
		if sec.ClassSectionTeacherID == nil {
			sec.ClassSectionTeacherID = Ptr(mt.MasjidTeacherID)
			assignAs = AssignedAsTeacher
		} else if sec.ClassSectionAssistantTeacherID == nil {
			sec.ClassSectionAssistantTeacherID = Ptr(mt.MasjidTeacherID)
			assignAs = AssignedAsAssistant
		} else {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusConflict, "Slot teacher & assistant sudah penuh")
		}

		sec.ClassSectionUpdatedAt = now
		if err := tx.Model(&sec).
			Select("class_section_teacher_id", "class_section_assistant_teacher_id", "class_section_updated_at").
			Updates(&sec).Error; err != nil {
			tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal assign teacher")
		}

		if err := tx.Commit().Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Transaksi gagal")
		}

		resp := dto.ClassSectionJoinResponse{
			ClassSectionID: sec.ClassSectionID.String(),
			AssignedAs:     string(assignAs),
		}
		if sec.ClassSectionTeacherID != nil {
			resp.ClassSectionTeacherID = sec.ClassSectionTeacherID.String()
		}
		if sec.ClassSectionAssistantTeacherID != nil {
			resp.ClassSectionAssistantTeacherID = sec.ClassSectionAssistantTeacherID.String()
		}
		return helper.JsonOK(c, "Berhasil bergabung sebagai teacher", fiber.Map{"item": resp})
	}

	tx.Rollback()
	return helper.JsonError(c, fiber.StatusBadRequest, "Role tidak didukung")
}
