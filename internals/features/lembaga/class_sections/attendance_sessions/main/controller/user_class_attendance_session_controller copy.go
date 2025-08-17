// internals/features/lembaga/class_sections/attendance_sessions/main/controller/teacher_class_attendance_session_controller.go
package controller

import (
	"time"

	"masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/dto"
	"masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/model"
	helper "masjidku_backend/internals/helpers"

	semstats "masjidku_backend/internals/features/lembaga/stats/semester_stats/service"

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

	// 1) Simpan entry kehadiran (per user)
	m := req.ToModel() // NOTE: ToModel() kamu sudah return *Model
	if err := tx.Create(m).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat entri kehadiran")
	}

	// 2) Ambil info Sesi: section_id + tanggal sesi (anchor) dari parent class_attendance_sessions
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

	// 3) Hitung delta counter dari status/score/kelulusan
	var dPresent, dSick, dLeave, dAbsent int
	switch req.UserClassAttendanceSessionsAttendanceStatus {
	case "present":
		dPresent = 1
	case "sick":
		dSick = 1
	case "leave":
		dLeave = 1
	case "absent":
		dAbsent = 1
	}
	var dSum *int
	if req.UserClassAttendanceSessionsScore != nil {
		dSum = req.UserClassAttendanceSessionsScore
	}
	var dPassed, dFailed *int
	if req.UserClassAttendanceSessionsGradePassed != nil {
		one := 1
		if *req.UserClassAttendanceSessionsGradePassed {
			dPassed = &one
		} else {
			dFailed = &one
		}
	}

	// 4) Upsert + bump counters semester stats untuk user ini
	semSvc := semstats.NewSemesterStatsService()
	if err := semSvc.BumpCounters(
		tx,
		masjidID,
		req.UserClassAttendanceSessionsUserClassID,
		sectionID,
		anchor,
		dPresent, dSick, dLeave, dAbsent,
		dSum, dPassed, dFailed,
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

	v := validator.New()
	if err := v.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// siapkan updates hanya field yang dikirim
	updates := map[string]any{}

	if req.UserClassAttendanceSessionsAttendanceStatus != nil {
		updates["user_class_attendance_sessions_attendance_status"] = *req.UserClassAttendanceSessionsAttendanceStatus
	}
	if req.UserClassAttendanceSessionsScore != nil {
		updates["user_class_attendance_sessions_score"] = *req.UserClassAttendanceSessionsScore
	}
	if req.UserClassAttendanceSessionsGradePassed != nil {
		updates["user_class_attendance_sessions_grade_passed"] = *req.UserClassAttendanceSessionsGradePassed
	}
	if req.UserClassAttendanceSessionsMaterialPersonal != nil {
		updates["user_class_attendance_sessions_material_personal"] = *req.UserClassAttendanceSessionsMaterialPersonal
	}
	if req.UserClassAttendanceSessionsPersonalNote != nil {
		updates["user_class_attendance_sessions_personal_note"] = *req.UserClassAttendanceSessionsPersonalNote
	}
	if req.UserClassAttendanceSessionsMemorization != nil {
		updates["user_class_attendance_sessions_memorization"] = *req.UserClassAttendanceSessionsMemorization
	}
	if req.UserClassAttendanceSessionsHomework != nil {
		updates["user_class_attendance_sessions_homework"] = *req.UserClassAttendanceSessionsHomework
	}

	if len(updates) == 0 {
		// tidak ada perubahan; tetap balikan ID biar klien tahu resource-nya
		return helper.JsonOK(c, "Tidak ada perubahan", dto.UserClassAttendanceSessionResponse{
			UserClassAttendanceSessionsID: entryID,
		})
	}

	// update dengan guard masjid + returning row
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
