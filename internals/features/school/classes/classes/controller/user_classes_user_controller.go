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

	// ⬇️ gunakan DTO & Model dari enrolments/user_classes (bukan classes/classes)
	ucdto "masjidku_backend/internals/features/school/classes/classes/dto"
	ucmodel "masjidku_backend/internals/features/school/classes/classes/model"
)

type UserMyClassController struct {
	DB *gorm.DB
}

func NewUserMyClassController(db *gorm.DB) *UserMyClassController {
	return &UserMyClassController{DB: db}
}

var userValidate = validator.New()

/* ================ helpers ================ */

// pastikan kelas valid & ambil masjid_id
type classInfo struct {
	ClassID       uuid.UUID `gorm:"column:class_id"`
	ClassStatus   string    `gorm:"column:class_status"`
	ClassMasjidID uuid.UUID `gorm:"column:class_masjid_id"`
}

// ensure/resolve masjid_student untuk (user, masjid)
func (h *UserMyClassController) ensureMasjidStudentForUser(tx *gorm.DB, userID, masjidID uuid.UUID, preferredID *uuid.UUID) (uuid.UUID, error) {
	if preferredID != nil && *preferredID != uuid.Nil {
		var ok bool
		err := tx.Raw(`
			SELECT EXISTS(
			  SELECT 1 FROM masjid_students
			  WHERE masjid_student_id = ? 
			    AND masjid_student_user_id = ?
			    AND masjid_student_masjid_id = ?
			    AND masjid_student_deleted_at IS NULL
			)
		`, *preferredID, userID, masjidID).Scan(&ok).Error
		if err != nil {
			return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa data siswa masjid")
		}
		if !ok {
			return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "masjid_student_id tidak valid untuk akun/tenant ini")
		}
		return *preferredID, nil
	}

	var msIDStr string
	if err := tx.Raw(`
		SELECT masjid_student_id::text
		FROM masjid_students
		WHERE masjid_student_user_id = ? 
		  AND masjid_student_masjid_id = ? 
		  AND masjid_student_deleted_at IS NULL
		ORDER BY masjid_student_created_at DESC
		LIMIT 1
	`, userID, masjidID).Scan(&msIDStr).Error; err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data siswa masjid")
	}
	if msIDStr != "" {
		msID, err := uuid.Parse(msIDStr)
		if err != nil {
			return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "masjid_student_id tidak valid")
		}
		return msID, nil
	}

	if err := tx.Raw(`
		INSERT INTO masjid_students (masjid_student_masjid_id, masjid_student_user_id, masjid_student_status)
		VALUES (?, ?, 'inactive')
		RETURNING masjid_student_id::text
	`, masjidID, userID).Scan(&msIDStr).Error; err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data siswa masjid")
	}

	msID, err := uuid.Parse(msIDStr)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "masjid_student_id tidak valid")
	}
	return msID, nil
}

/* ================== USER: LIST & DETAIL ================== */

// GET /api/u/user-classes
func (h *UserMyClassController) ListMyUserClasses(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	// Query params (sesuai DTO baru)
	var q ucdto.ListUserClassesQuery
	// default
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := h.DB.Model(&ucmodel.UserClassesModel{}).
		Joins(`JOIN classes ON classes.class_id = user_classes.user_classes_class_id`).
		Joins(`JOIN masjid_students ms
			   ON ms.masjid_student_id = user_classes.user_classes_masjid_student_id
			  AND ms.masjid_student_deleted_at IS NULL`).
		Where(`
			ms.masjid_student_user_id = ?
			AND classes.class_deleted_at IS NULL
			AND classes.class_delete_pending_until IS NULL
			AND user_classes_deleted_at IS NULL
		`, userID)

	// Filter yang tersedia di DTO
	if q.ClassID != nil {
		tx = tx.Where("user_classes_class_id = ?", *q.ClassID)
	}
	if q.StudentID != nil {
		tx = tx.Where("user_classes_masjid_student_id = ?", *q.StudentID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		tx = tx.Where("user_classes_status = ?", strings.TrimSpace(*q.Status))
	}
	if q.Result != nil && strings.TrimSpace(*q.Result) != "" {
		tx = tx.Where("user_classes_result = ?", strings.TrimSpace(*q.Result))
	}
	if q.JoinedGt != nil {
		tx = tx.Where("user_classes_joined_at IS NOT NULL AND user_classes_joined_at >= ?", *q.JoinedGt)
	}
	if q.JoinedLt != nil {
		tx = tx.Where("user_classes_joined_at IS NOT NULL AND user_classes_joined_at <= ?", *q.JoinedLt)
	}
	if q.PaidDueLt != nil {
		tx = tx.Where("user_classes_paid_until IS NOT NULL AND user_classes_paid_until < ?", *q.PaidDueLt)
	}
	if q.PaidDueGt != nil {
		tx = tx.Where("user_classes_paid_until IS NOT NULL AND user_classes_paid_until > ?", *q.PaidDueGt)
	}
	if s := strings.TrimSpace(q.Search); s != "" {
		// cari di kelas (nama/kode) — optional, sesuaikan kolom yang ada
		p := "%" + strings.ToLower(s) + "%"
		tx = tx.Where(`LOWER(classes.class_slug) LIKE ? OR LOWER(classes.class_code) LIKE ?`, p, p)
	}

	// Sorting (pakai query string langsung, DTO tidak pegang Sort)
	sort := strings.ToLower(strings.TrimSpace(c.Query("sort", "created_at_desc")))
	switch sort {
	case "created_at_asc":
		tx = tx.Order("user_classes_created_at ASC")
	case "joined_at_desc":
		tx = tx.Order("user_classes_joined_at DESC NULLS LAST").Order("user_classes_created_at DESC")
	case "joined_at_asc":
		tx = tx.Order("user_classes_joined_at ASC NULLS LAST").Order("user_classes_created_at ASC")
	case "completed_at_desc":
		tx = tx.Order("user_classes_completed_at DESC NULLS LAST").Order("user_classes_created_at DESC")
	case "completed_at_asc":
		tx = tx.Order("user_classes_completed_at ASC NULLS LAST").Order("user_classes_created_at ASC")
	default: // created_at_desc
		tx = tx.Order("user_classes_created_at DESC")
	}

	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	var rows []ucmodel.UserClassesModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resps := ucdto.ToUserClassesResponses(rows)
	return helper.JsonOK(c, "OK", resps)
}

/* ================== USER: SELF ENROLL ================== */

// POST /api/u/user-classes
func (h *UserMyClassController) SelfEnroll(c *fiber.Ctx) error {
	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil {
		return err
	}

	type selfEnrollRequest struct {
		ClassID         uuid.UUID  `json:"user_classes_class_id" validate:"required"`
		MasjidStudentID *uuid.UUID `json:"user_classes_masjid_student_id" validate:"omitempty"`
		JoinedAt        *time.Time `json:"user_classes_joined_at" validate:"omitempty"`
	}
	var req selfEnrollRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := userValidate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil kelas & tenant
	var cls classInfo
	if err := h.DB.
		Select("class_id, class_status, class_masjid_id").
		Table("classes").
		Where("class_id = ? AND class_deleted_at IS NULL AND class_delete_pending_until IS NULL", req.ClassID).
		First(&cls).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa kelas")
	}
	// pastikan kelas aktif
	if !strings.EqualFold(cls.ClassStatus, "active") {
		return fiber.NewError(fiber.StatusBadRequest, "Kelas sedang tidak aktif")
	}
	masjidID := cls.ClassMasjidID

	return h.DB.Transaction(func(tx *gorm.DB) error {
		// Resolve/buat masjid_student_id
		msID, err := h.ensureMasjidStudentForUser(tx, userID, masjidID, req.MasjidStudentID)
		if err != nil {
			return err
		}

		// Cegah duplikasi berjalan
		{
			var cnt int64
			if err := tx.Table("user_classes").
				Where("user_classes_deleted_at IS NULL").
				Where("user_classes_masjid_student_id = ? AND user_classes_class_id = ? AND user_classes_masjid_id = ?",
					msID, req.ClassID, masjidID).
				Where("user_classes_status IN ('active','inactive')").
				Count(&cnt).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa duplikasi pendaftaran")
			}
			if cnt > 0 {
				return fiber.NewError(fiber.StatusConflict, "Anda sudah memiliki pendaftaran berjalan untuk kelas ini")
			}
		}

		// Buat enrolment 'inactive' (pending approval); joined_at opsional
		m := &ucmodel.UserClassesModel{
			UserClassesMasjidStudentID: msID,
			UserClassesClassID:         req.ClassID,
			UserClassesMasjidID:        masjidID,
			UserClassesStatus:          "inactive", // gunakan literal supaya tidak bergantung konstanta
			UserClassesJoinedAt:        req.JoinedAt,
		}

		if err := tx.Create(m).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return fiber.NewError(fiber.StatusConflict, "Pendaftaran untuk kelas ini sudah ada")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat pendaftaran")
		}

		return helper.JsonCreated(c, "Pendaftaran berhasil dikirim, menunggu persetujuan admin", ucdto.FromModelUserClasses(m))
	})
}
