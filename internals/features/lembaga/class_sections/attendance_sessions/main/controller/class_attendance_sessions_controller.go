// internals/features/lembaga/class_sections/attendance_sessions/main/controller/class_attendance_session_controller.go
package controller

import (
	"errors"
	"time"

	helper "masjidku_backend/internals/helpers"

	attendanceDTO "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/dto"
	attendanceModel "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/model"
	secModel "masjidku_backend/internals/features/lembaga/class_sections/main/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* ===============================
   Controller & Constructor (gaya sama)
=============================== */

type ClassAttendanceSessionController struct {
	DB *gorm.DB
}

func NewClassAttendanceSessionController(db *gorm.DB) *ClassAttendanceSessionController {
	return &ClassAttendanceSessionController{DB: db}
}

/* ===============================
   CREATE
=============================== */

// POST /admin/class-attendance-sessions
func (ctrl *ClassAttendanceSessionController) CreateClassAttendanceSession(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req attendanceDTO.CreateClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant
	req.MasjidID = masjidID

	// Validasi payload (validator lokal)
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 1) Validasi section milik masjid (dan masih aktif jika diperlukan)
	var sec secModel.ClassSectionModel
	if err := ctrl.DB.
		Select("class_sections_id, class_sections_masjid_id, class_sections_teacher_id, class_sections_is_active").
		First(&sec, "class_sections_id = ? AND class_sections_deleted_at IS NULL", req.SectionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil section")
	}
	if sec.ClassSectionsMasjidID == nil || *sec.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
	}
	// (Opsional) cek aktif, jika field tersedia
	// if !sec.ClassSectionsIsActive { return fiber.NewError(fiber.StatusBadRequest, "Section tidak aktif") }

	// 2) Cek unik (section_id, date)
	var dupeCount int64
	if err := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where("class_attendance_sessions_section_id = ? AND class_attendance_sessions_date = ?",
			req.SectionID, req.Date).
		Count(&dupeCount).Error; err == nil && dupeCount > 0 {
		return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
	}

	// 3) Default guru sesi jika tidak diisi → pakai guru utama dari section (jika ada)
	if req.TeacherUserID == nil && sec.ClassSectionsTeacherID != nil {
		req.TeacherUserID = sec.ClassSectionsTeacherID
	}

	// 4) Build model & simpan
	m := req.ToModel()
	if err := ctrl.DB.Create(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi kehadiran")
	}

	return helper.JsonCreated(c, "Sesi kehadiran berhasil dibuat", attendanceDTO.FromClassAttendanceSessionModel(m))
}

/* ===============================
   UPDATE (partial)
=============================== */

// PUT /admin/class-attendance-sessions/:id
func (ctrl *ClassAttendanceSessionController) UpdateClassAttendanceSession(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// 1) Ambil existing
	var existing attendanceModel.ClassAttendanceSessionModel
	if err := ctrl.DB.
		First(&existing, "class_attendance_sessions_id = ?", sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	// Guard tenant
	if existing.MasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah sesi milik masjid lain")
	}

	// 2) Parse payload
	var req attendanceDTO.UpdateClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant (jangan bisa dipindah masjid)
	req.MasjidID = &masjidID

	// 3) Validasi payload
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 4) Jika section diganti → validasi section baru milik masjid
	targetSectionID := existing.SectionID
	if req.SectionID != nil {
		targetSectionID = *req.SectionID
		var sec secModel.ClassSectionModel
		if err := ctrl.DB.
			Select("class_sections_id, class_sections_masjid_id, class_sections_teacher_id, class_sections_is_active").
			First(&sec, "class_sections_id = ? AND class_sections_deleted_at IS NULL", targetSectionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Section tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil section")
		}
		if sec.ClassSectionsMasjidID == nil || *sec.ClassSectionsMasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
		}
		// sinkron guru jika kosong
		if req.TeacherUserID == nil && sec.ClassSectionsTeacherID != nil {
			req.TeacherUserID = sec.ClassSectionsTeacherID
		}
	}

	// 5) Jika date atau section berubah → cek unik (section_id, date) exclude current
	targetDate := existing.Date
	if req.Date != nil {
		targetDate = *req.Date
	}
	{
		var cnt int64
		if err := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
			Where("class_attendance_sessions_section_id = ? AND class_attendance_sessions_date = ? AND class_attendance_sessions_id <> ?",
				targetSectionID, targetDate, existing.ClassAttendanceSessionID).
			Count(&cnt).Error; err == nil && cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}
	}

	// 6) Terapkan perubahan ke model
	if req.SectionID != nil {
		existing.SectionID = *req.SectionID
	}
	// MasjidID tidak bisa diubah (force)
	existing.MasjidID = masjidID

	if req.Date != nil {
		d := *req.Date
		existing.Date = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	}
	if req.Title != nil {
		existing.Title = req.Title
	}
	if req.GeneralInfo != nil {
		existing.GeneralInfo = *req.GeneralInfo
	}
	if req.Note != nil {
		existing.Note = req.Note
	}
	if req.TeacherUserID != nil {
		existing.TeacherUserID = req.TeacherUserID
	}

	// 7) Simpan
	if err := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where("class_attendance_sessions_id = ?", existing.ClassAttendanceSessionID).
		Updates(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui sesi kehadiran")
	}

	return helper.JsonUpdated(c, "Sesi kehadiran berhasil diperbarui", attendanceDTO.FromClassAttendanceSessionModel(existing))
}


// DELETE /admin/class-attendance-sessions/:id
func (ctrl *ClassAttendanceSessionController) DeleteClassAttendanceSession(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c) // atau pakai GetMasjidIDFromTokenPreferTeacher
	if err != nil {
		return err
	}

	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// 1) pastikan sesi ada & milik masjid pemanggil
	var existing attendanceModel.ClassAttendanceSessionModel
	if err := ctrl.DB.
		First(&existing, "class_attendance_sessions_id = ?", sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if existing.MasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus sesi milik masjid lain")
	}

	// 2) hapus (FK ke entries disarankan ON DELETE CASCADE)
	if err := ctrl.DB.Delete(&attendanceModel.ClassAttendanceSessionModel{}, "class_attendance_sessions_id = ?", sessionID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus sesi kehadiran")
	}

	// jika kamu punya helper.JsonDeleted, boleh pakai itu.
	return helper.JsonDeleted(c, "Sesi kehadiran berhasil dihapus", attendanceDTO.FromClassAttendanceSessionModel(existing))

}