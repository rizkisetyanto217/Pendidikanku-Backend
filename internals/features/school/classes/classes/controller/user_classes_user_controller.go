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

/* ================ helpers ================ */

// pastikan kelas valid & ambil masjid_id
type classInfo struct {
	ClassID       uuid.UUID `gorm:"column:class_id"`
	ClassStatus   string    `gorm:"column:class_status"`
	ClassMasjidID uuid.UUID `gorm:"column:class_masjid_id"`
}

// ensure/resolve masjid_student untuk (user, masjid)
// - kalau preferredID diberikan, validasi kepemilikan & tenant
// - kalau tidak ada, cari yang alive; jika tidak ada → buat baru (status=inactive)
func (h *UserMyClassController) ensureMasjidStudentForUser(tx *gorm.DB, userID, masjidID uuid.UUID, preferredID *uuid.UUID) (uuid.UUID, error) {
	// jika user menyuplai ID → validasi
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

	// cari yang alive lebih dulu (scan sebagai TEXT → parse UUID)
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

	// buat baru (status INACTIVE) + RETURNING ::text → parse
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

	// Default query
	var q ucDTO.ListUserClassQuery
	q.Limit, q.Offset = 20, 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	// Base query:
	// join classes (agar kelas bukan deleted/pending) + join masjid_students untuk limit ke user yang login
	tx := h.DB.Model(&ucModel.UserClassesModel{}).
		Joins("JOIN classes ON classes.class_id = user_classes.user_classes_class_id").
		Joins("JOIN masjid_students ms ON ms.masjid_student_id = user_classes.user_classes_masjid_student_id AND ms.masjid_student_deleted_at IS NULL").
		Where(`
			ms.masjid_student_user_id = ?
			AND classes.class_deleted_at IS NULL 
			AND classes.class_delete_pending_until IS NULL 
			AND user_classes_deleted_at IS NULL
		`, userID)

	// Filter opsional (tanpa term_id & user_id karena tidak ada di DDL)
	if q.ClassID != nil {
		tx = tx.Where("user_classes_class_id = ?", *q.ClassID)
	}
	if q.MasjidID != nil {
		tx = tx.Where("user_classes_masjid_id = ?", *q.MasjidID)
	}
	if q.MasjidStudentID != nil {
		tx = tx.Where("user_classes_masjid_student_id = ?", *q.MasjidStudentID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		tx = tx.Where("user_classes_status = ?", strings.TrimSpace(*q.Status))
	}
	if q.Result != nil && strings.TrimSpace(*q.Result) != "" {
		tx = tx.Where("user_classes_result = ?", strings.TrimSpace(*q.Result))
	}
	if q.ActiveNow != nil && *q.ActiveNow {
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
	case "completed_at_desc":
		tx = tx.Order("user_classes_completed_at DESC NULLS LAST").Order("user_classes_created_at DESC")
	case "completed_at_asc":
		tx = tx.Order("user_classes_completed_at ASC NULLS LAST").Order("user_classes_created_at ASC")
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
		Joins("JOIN masjid_students ms ON ms.masjid_student_id = user_classes.user_classes_masjid_student_id AND ms.masjid_student_deleted_at IS NULL").
		Where(`
			user_classes_id = ?
			AND ms.masjid_student_user_id = ?
			AND classes.class_deleted_at IS NULL
			AND classes.class_delete_pending_until IS NULL
			AND user_classes_deleted_at IS NULL
		`, ucID, userID).
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
		ClassID         uuid.UUID  `json:"user_classes_class_id" validate:"required"`
		// Opsional: biarkan kosong, nanti di-resolve/auto-create
		MasjidStudentID *uuid.UUID `json:"user_classes_masjid_student_id" validate:"omitempty"`
		// Opsional jika ingin set joined_at saat self-enroll
		JoinedAt        *time.Time `json:"user_classes_joined_at" validate:"omitempty"`
	}
	var req selfEnrollRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := userValidate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Ambil kelas (aktif) + masjid_id (kelas tidak boleh soft-deleted & tidak pending delete)
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
	if !strings.EqualFold(cls.ClassStatus, ucModel.UserClassStatusActive) {
		return fiber.NewError(fiber.StatusBadRequest, "Kelas sedang tidak aktif")
	}
	masjidID := cls.ClassMasjidID

	return h.DB.Transaction(func(tx *gorm.DB) error {
		// Resolve / buat masjid_student_id untuk user+tenant ini
		msID, err := h.ensureMasjidStudentForUser(tx, userID, masjidID, req.MasjidStudentID)
		if err != nil {
			return err
		}

		// Cegah duplikasi berjalan pada kombinasi (masjid_student, class, masjid)
		// (larang ada 'active' atau 'inactive' kedua; 'completed' boleh buat histori)
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
		m := &ucModel.UserClassesModel{
			UserClassesMasjidStudentID: msID,
			UserClassesClassID:         req.ClassID,
			UserClassesMasjidID:        masjidID,
			UserClassesStatus:          ucModel.UserClassStatusInactive,
			UserClassesJoinedAt:        req.JoinedAt,
			// left_at biarkan NULL; result/completed_at hanya untuk status completed
		}

		if err := tx.Create(m).Error; err != nil {
			// unik index aktif hanya mengikat status='active', tapi guard di atas melarang inactive duplikat juga
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return fiber.NewError(fiber.StatusConflict, "Pendaftaran untuk kelas ini sudah ada")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat pendaftaran")
		}

		return helper.JsonCreated(c, "Pendaftaran berhasil dikirim, menunggu persetujuan admin", ucDTO.NewUserClassResponse(m))
	})
}