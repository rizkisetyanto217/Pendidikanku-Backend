// file: internals/features/lembaga/classes/user_classes/main/controller/user_my_class_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	ucDTO "masjidku_backend/internals/features/school/classes/classes/dto"
	ucModel "masjidku_backend/internals/features/school/classes/classes/model"
)

type UserMyClassController struct {
	DB *gorm.DB
}

func NewUserMyClassController(db *gorm.DB) *UserMyClassController {
	return &UserMyClassController{DB: db}
}

var userValidate = validator.New()

/* ================== USER: LIST & DETAIL ================== */

// GET /api/u/user-classes
func (h *UserMyClassController) ListMyUserClasses(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	// Default query
	var q ucDTO.ListUserClassQuery
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	// Base query (join classes untuk memastikan kelas tidak terhapus/pending delete)
	tx := h.DB.Model(&ucModel.UserClassesModel{}).
		Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
		Where(
			"user_classes_user_id = ? AND classes.class_deleted_at IS NULL AND classes.class_delete_pending_until IS NULL AND user_classes_deleted_at IS NULL",
			userID,
		)

	// Filter opsional
	if q.ClassID != nil {
		tx = tx.Where("user_classes_class_id = ?", *q.ClassID)
	}
	if q.MasjidID != nil {
		tx = tx.Where("user_classes_masjid_id = ?", *q.MasjidID)
	}
	if q.TermID != nil {
		tx = tx.Where("user_classes_term_id = ?", *q.TermID)
	}
	if q.MasjidStudentID != nil {
		tx = tx.Where("user_classes_masjid_student_id = ?", *q.MasjidStudentID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		tx = tx.Where("user_classes_status = ?", strings.TrimSpace(*q.Status))
	}
	if q.ActiveNow != nil && *q.ActiveNow {
		// Active now = status active & belum left
		tx = tx.Where("user_classes_status = 'active' AND user_classes_left_at IS NULL")
	}
	// Rentang joined_at (inklusif)
	if q.JoinedFrom != nil {
		tx = tx.Where("user_classes_joined_at IS NOT NULL AND user_classes_joined_at >= ?", *q.JoinedFrom)
	}
	if q.JoinedTo != nil {
		tx = tx.Where("user_classes_joined_at IS NOT NULL AND user_classes_joined_at <= ?", *q.JoinedTo)
	}

	// Sorting
	sort := "created_at_desc"
	if q.Sort != nil {
		sort = strings.ToLower(strings.TrimSpace(*q.Sort))
	}
	switch sort {
	case "created_at_asc":
		tx = tx.Order("user_classes_created_at ASC")
	case "joined_at_desc":
		tx = tx.Order("user_classes_joined_at DESC NULLS LAST").Order("user_classes_created_at DESC")
	case "joined_at_asc":
		tx = tx.Order("user_classes_joined_at ASC NULLS LAST").Order("user_classes_created_at ASC")
	case "created_at_desc":
		fallthrough
	default:
		tx = tx.Order("user_classes_created_at DESC")
	}

	// Limit & Offset
	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	// Eksekusi
	var rows []ucModel.UserClassesModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Response
	resp := make([]*ucDTO.UserClassResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, ucDTO.NewUserClassResponse(&rows[i]))
	}
	return helper.JsonOK(c, "OK", resp)
}

// GET /api/u/user-classes/:id
func (h *UserMyClassController) GetMyUserClassByID(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}
	ucID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m ucModel.UserClassesModel
	err = h.DB.Model(&ucModel.UserClassesModel{}).
		Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
		Where(
			"user_classes_id = ? AND user_classes_user_id = ? AND classes.class_deleted_at IS NULL AND classes.class_delete_pending_until IS NULL AND user_classes_deleted_at IS NULL",
			ucID, userID,
		).
		First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Enrolment tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "OK", ucDTO.NewUserClassResponse(&m))
}

/* ================== USER: SELF ENROLL ================== */

// POST /api/u/user-classes
func (h *UserMyClassController) SelfEnroll(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	type selfEnrollRequest struct {
		ClassID          uuid.UUID  `json:"user_classes_class_id" validate:"required"`
		TermID           uuid.UUID  `json:"user_classes_term_id"  validate:"required"`
		MasjidStudentID  *uuid.UUID `json:"user_classes_masjid_student_id" validate:"omitempty"`
		// Opsional jika ingin set joined_at saat self enroll:
		JoinedAt         *time.Time `json:"user_classes_joined_at" validate:"omitempty"`
	}
	var req selfEnrollRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := userValidate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil kelas aktif + masjid_id (kelas tidak boleh soft-deleted & tidak pending delete)
	var cls ucModel.ClassModel
	if err := h.DB.
		Select("class_id, class_is_active, class_masjid_id").
		Where("class_id = ? AND class_deleted_at IS NULL AND class_delete_pending_until IS NULL", req.ClassID).
		First(&cls).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa kelas")
	}
	if !cls.ClassIsActive {
		return fiber.NewError(fiber.StatusBadRequest, "Kelas sedang tidak aktif")
	}
	masjidID := cls.ClassMasjidID

	// Validasi term milik masjid ini
	{
		var count int64
		if err := h.DB.Table("academic_terms").
			Where("academic_terms_id = ? AND academic_terms_masjid_id = ?", req.TermID, masjidID).
			Count(&count).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa term akademik")
		}
		if count == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Term akademik tidak ditemukan di masjid ini")
		}
	}

	// (Opsional) validasi masjid_student milik tenant ini
	if req.MasjidStudentID != nil {
		var count int64
		if err := h.DB.Table("masjid_students").
			Where("masjid_student_id = ? AND masjid_student_masjid_id = ?", *req.MasjidStudentID, masjidID).
			Count(&count).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa data siswa masjid")
		}
		if count == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Data siswa masjid tidak valid untuk tenant ini")
		}
	}

	// Cegah duplikasi berjalan (active/inactive) pada (user,class,term,masjid)
	{
		var cnt int64
		if err := h.DB.Table("user_classes").
			Where("user_classes_deleted_at IS NULL").
			Where("user_classes_user_id = ? AND user_classes_class_id = ? AND user_classes_term_id = ? AND user_classes_masjid_id = ?",
				userID, req.ClassID, req.TermID, masjidID).
			Where("user_classes_status IN ('active','inactive')").
			Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa duplikasi pendaftaran")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Anda sudah memiliki pendaftaran berjalan untuk kelas & term ini")
		}
	}

	// Buat enrolment 'inactive' (pending approval); joined_at opsional
	m := &ucModel.UserClassesModel{
		UserClassesUserID:          userID,
		UserClassesClassID:         req.ClassID,
		UserClassesMasjidID:        masjidID,
		UserClassesTermID:          req.TermID,
		UserClassesMasjidStudentID: req.MasjidStudentID,
		UserClassesStatus:          ucModel.UserClassStatusInactive,
		UserClassesJoinedAt:        req.JoinedAt,
		// left_at biarkan NULL
	}

	if err := h.DB.Create(m).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fiber.NewError(fiber.StatusConflict, "Pendaftaran untuk kelas & term ini sudah ada")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat pendaftaran")
	}

	return helper.JsonCreated(c, "Pendaftaran berhasil dikirim, menunggu persetujuan admin", ucDTO.NewUserClassResponse(m))
}
