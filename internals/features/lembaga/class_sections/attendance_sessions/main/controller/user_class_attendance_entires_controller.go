package controller

import (
	"strconv"
	"time"

	"masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/dto"
	"masjidku_backend/internals/features/lembaga/class_sections/attendance_sessions/main/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TeacherClassAttendanceEntryController struct {
	DB *gorm.DB
}

func NewTeacherClassAttendanceEntryController(db *gorm.DB) *TeacherClassAttendanceEntryController {
	return &TeacherClassAttendanceEntryController{DB: db}
}

// POST /teacher/class-attendance-entries
// POST /teacher/class-attendance-entries
func (ctrl *TeacherClassAttendanceEntryController) CreateAttendanceEntry(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.CreateUserClassAttendanceEntryRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// harus pointer karena field di DTO bertipe *uuid.UUID
	req.UserClassAttendanceEntriesMasjidID = &masjidID

	// validator lokal (kalau controller tidak punya ctrl.Validate)
	v := validator.New()
	if err := v.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// build model dari DTO
	m := req.ToModel() // *entryModel.UserClassAttendanceEntryModel

	// simpan
	if err := ctrl.DB.Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat entri kehadiran")
	}

	// From* menerima value, jadi deref
	return helper.JsonCreated(c, "Entri kehadiran berhasil dibuat", dto.FromUserClassAttendanceEntryModel(*m))
}


// GET /teacher/class-attendance-entries?session_id=...
func (ctrl *TeacherClassAttendanceEntryController) ListAttendanceEntries(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	q := ctrl.DB.Model(&model.UserClassAttendanceEntryModel{}).
		Where("user_class_attendance_entries_masjid_id = ?", masjidID)

	// filter by session_id
	if s := c.Query("session_id"); s != "" {
		if sid, err := uuid.Parse(s); err == nil {
			q = q.Where("user_class_attendance_entries_session_id = ?", sid)
		}
	}

	// filter by user_class_id
	if uc := c.Query("user_class_id"); uc != "" {
		if u, err := uuid.Parse(uc); err == nil {
			q = q.Where("user_class_attendance_entries_user_class_id = ?", u)
		}
	}

	// filter by status
	if st := c.Query("status"); st != "" {
		q = q.Where("user_class_attendance_entries_attendance_status = ?", st)
	}

	// filter tanggal via JOIN ke sessions (karena entries tidak punya kolom date)
	df := c.Query("date_from")
	dt := c.Query("date_to")
	if df != "" || dt != "" {
		q = q.Joins(`JOIN class_attendance_sessions s 
                      ON s.class_attendance_sessions_id = user_class_attendance_entries_session_id`).
			Where("s.class_attendance_sessions_masjid_id = ?", masjidID)

		if df != "" {
			if t, err := time.Parse("2006-01-02", df); err == nil {
				q = q.Where("s.class_attendance_sessions_date >= ?", t)
			}
		}
		if dt != "" {
			if t, err := time.Parse("2006-01-02", dt); err == nil {
				q = q.Where("s.class_attendance_sessions_date <= ?", t)
			}
		}
	}

	// pagination
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var rows []model.UserClassAttendanceEntryModel
	if err := q.
		Order("user_class_attendance_entries_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]dto.UserClassAttendanceEntryResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, dto.FromUserClassAttendanceEntryModel(r))
	}

	// ✅ Kembalikan LIST, bukan single item "m"
	return helper.JsonOK(c, "Daftar kehadiran ditemukan", resp)
}


// PATCH /teacher/class-attendance-entries/:id
func (ctrl *TeacherClassAttendanceEntryController) UpdateAttendanceEntry(c *fiber.Ctx) error {
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	idStr := c.Params("id")
	entryID, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateUserClassAttendanceEntryRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	v := validator.New()
	if err := v.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// siapkan map update hanya field yang dikirim
	updates := map[string]interface{}{}

	if req.UserClassAttendanceEntriesAttendanceStatus != nil {
		updates["user_class_attendance_entries_attendance_status"] = *req.UserClassAttendanceEntriesAttendanceStatus
	}
	if req.UserClassAttendanceEntriesScore != nil {
		updates["user_class_attendance_entries_score"] = *req.UserClassAttendanceEntriesScore
	}
	if req.UserClassAttendanceEntriesGradePassed != nil {
		updates["user_class_attendance_entries_grade_passed"] = *req.UserClassAttendanceEntriesGradePassed
	}
	if req.UserClassAttendanceEntriesMaterialPersonal != nil {
		updates["user_class_attendance_entries_material_personal"] = *req.UserClassAttendanceEntriesMaterialPersonal
	}
	if req.UserClassAttendanceEntriesPersonalNote != nil {
		updates["user_class_attendance_entries_personal_note"] = *req.UserClassAttendanceEntriesPersonalNote
	}
	if req.UserClassAttendanceEntriesMemorization != nil {
		updates["user_class_attendance_entries_memorization"] = *req.UserClassAttendanceEntriesMemorization
	}
	if req.UserClassAttendanceEntriesHomework != nil {
		updates["user_class_attendance_entries_homework"] = *req.UserClassAttendanceEntriesHomework
	}

	if len(updates) == 0 {
		return helper.JsonOK(c, "Tidak ada perubahan", dto.UserClassAttendanceEntryResponse{
			UserClassAttendanceEntriesID: entryID,
		})
	}

	// jalankan update dengan scope masjid + returning row terbaru
	var updated model.UserClassAttendanceEntryModel
	tx := ctrl.DB.Model(&model.UserClassAttendanceEntryModel{}).
		Where("user_class_attendance_entries_id = ? AND user_class_attendance_entries_masjid_id = ?", entryID, masjidID).
		Clauses(clause.Returning{}).
		Updates(updates).
		Scan(&updated)

	if tx.Error != nil {
		// mapping error umum
		// NOTE: kalau pakai pgx, bisa errors.As(err, *pgconn.PgError) untuk detail kode.
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengubah entri kehadiran")
	}
	if tx.RowsAffected == 0 {
		// tidak ditemukan di masjid ini
		return fiber.NewError(fiber.StatusNotFound, "Entri kehadiran tidak ditemukan")
	}

	return helper.JsonOK(c, "Entri kehadiran berhasil diubah", dto.FromUserClassAttendanceEntryModel(updated))
}

