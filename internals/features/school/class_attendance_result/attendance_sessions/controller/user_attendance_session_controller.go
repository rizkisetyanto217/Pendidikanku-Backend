// internals/features/lembaga/class_sections/attendance_sessions/main/controller/teacher_class_attendance_session_controller.go
package controller

import (
	"time"

	"masjidku_backend/internals/features/school/class_attendance_result/attendance_sessions/dto"
	"masjidku_backend/internals/features/school/class_attendance_result/attendance_sessions/model"
	helper "masjidku_backend/internals/helpers"

	semstats "masjidku_backend/internals/features/lembaga/stats/semester_stats/service"
	attendanceservice "masjidku_backend/internals/features/school/class_attendance_result/attendance_sessions_settings/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TeacherClassAttendanceSessionController struct {
	DB *gorm.DB
}

func NewTeacherClassAttendanceSessionController(db *gorm.DB) *TeacherClassAttendanceSessionController {
	return &TeacherClassAttendanceSessionController{DB: db}
}



/* ===================== CREATE ===================== */
// POST /teacher/class-attendance-sessions
func (ctrl *TeacherClassAttendanceSessionController) CreateAttendanceSession(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.CreateUserClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.UserClassAttendanceSessionsMasjidID = &masjidID

	// Validasi dasar (type/enum/range)
	v := validator.New()
	if err := v.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ===== TRANSACTION START =====
	tx := ctrl.DB.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 0) Ambil settings & normalize payload sesuai aturan (ENABLE/REQUIRE)
	svc := attendanceservice.New(ctrl.DB) // import pakai alias: attendanceservice "masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/service"
	set, err := svc.GetSettings(masjidID, tx)
	if err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca settings: "+err.Error())
	}
	if err := svc.NormalizeCreate(&req, set); err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 1) Simpan entry kehadiran (per user) — payload sudah ternormalisasi
	m := req.ToModel()
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat entri kehadiran")
	}

	// 2) Ambil info sesi: section_id + tanggal sesi (anchor)
	type sessRow struct {
		SectionID uuid.UUID `gorm:"column:class_attendance_sessions_section_id"`
		Date      time.Time `gorm:"column:class_attendance_sessions_date"`
	}
	var sr sessRow
	if err := tx.Table("class_attendance_sessions").
		Select("class_attendance_sessions_section_id, class_attendance_sessions_date").
		Where("class_attendance_sessions_id = ? AND class_attendance_sessions_masjid_id = ?", req.UserClassAttendanceSessionsSessionID, masjidID).
		Take(&sr).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusBadRequest, "Sesi tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	anchor := sr.Date
	if anchor.IsZero() {
		anchor = time.Now()
	}

	sectionID := sr.SectionID
	if sectionID == uuid.Nil {
		// fallback: cari section aktif dari user_class_sections pada tanggal sesi
		type secRow struct {
			SectionID uuid.UUID `gorm:"column:user_class_sections_section_id"`
		}
		var sec secRow
		if err := tx.Table("user_class_sections").
			Select("user_class_sections_section_id").
			Where("user_class_sections_masjid_id = ?", masjidID).
			Where("user_class_sections_user_class_id = ?", req.UserClassAttendanceSessionsUserClassID).
			Where("user_class_sections_assigned_at <= ?", anchor).
			Where("(user_class_sections_unassigned_at IS NULL OR user_class_sections_unassigned_at > ?)", anchor).
			Order("user_class_sections_assigned_at DESC").
			Limit(1).
			Take(&sec).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				return fiber.NewError(fiber.StatusBadRequest, "Section untuk user_class pada tanggal sesi tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		sectionID = sec.SectionID
	}

	// 3) Hitung delta counter (hormati ENABLE di settings)
	cnt := svc.ComputeCountersOnCreate(&req, set)

	// 4) Upsert + bump counters semester stats untuk user ini
	semSvc := semstats.NewSemesterStatsService()
	if err := semSvc.BumpCounters(
		tx,
		masjidID,
		req.UserClassAttendanceSessionsUserClassID,
		sectionID,
		anchor,
		cnt.DPresent, cnt.DSick, cnt.DLeave, cnt.DAbsent,
		cnt.DSum, cnt.DPassed, cnt.DFailed,
	); err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal update semester stats: "+err.Error())
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// ===== TRANSACTION END =====

	return helper.JsonCreated(c, "Entri kehadiran berhasil dibuat", dto.FromUserClassAttendanceSessionModel(*m))
}


/* ===================== UPDATE (partial) ===================== */
// PATCH /teacher/class-attendance-sessions/:id
func (ctrl *TeacherClassAttendanceSessionController) UpdateAttendanceSession(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	entryID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateUserClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Validasi dasar (type/enum/range)
	v := validator.New()
	if err := v.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// === Load settings dan normalize update ===
	svc := attendanceservice.New(ctrl.DB)
	set, err := svc.GetSettings(masjidID, nil)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca settings: "+err.Error())
	}
	updates, err := svc.NormalizeUpdate(&req, set)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if len(updates) == 0 {
		// tidak ada perubahan valid; balikan ID biar klien tahu resource-nya
		return helper.JsonOK(c, "Tidak ada perubahan", dto.UserClassAttendanceSessionResponse{
			UserClassAttendanceSessionsID: entryID,
		})
	}

	// === Update dengan guard masjid + returning row ===
	var updated model.UserClassAttendanceSessionModel
	tx := ctrl.DB.Model(&model.UserClassAttendanceSessionModel{}).
		Where("user_class_attendance_sessions_id = ? AND user_class_attendance_sessions_masjid_id = ?", entryID, masjidID).
		Clauses(clause.Returning{}).
		Updates(updates).
		Scan(&updated)

	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengubah entri kehadiran")
	}
	if tx.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Entri kehadiran tidak ditemukan")
	}

	return helper.JsonOK(c, "Entri kehadiran berhasil diubah", dto.FromUserClassAttendanceSessionModel(updated))
}

