// internals/features/lembaga/class_sections/attendance_sessions/main/controller/class_attendance_session_controller.go
package controller

import (
	"errors"
	"strconv"
	"strings"
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



// GET /admin/class-attendance-sessions/section/:section_id?date_from=&date_to=&limit=&offset=
func (ctrl *ClassAttendanceSessionController) ListBySection(c *fiber.Ctx) error {
	// Tenant (admin ATAU teacher boleh)
	masjidID, err := helper.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil { return err }

	// Role & user
	userID, _ := helper.GetUserIDFromToken(c)
	isAdmin := func() bool {
		if mid, err := helper.GetMasjidIDFromToken(c); err == nil && mid == masjidID { return true }
		return false
	}()
	isTeacher := func() bool {
		if mid, err := helper.GetTeacherMasjidIDFromToken(c); err == nil && mid == masjidID { return true }
		return false
	}()

	// Path param: section
	secID, err := uuid.Parse(strings.TrimSpace(c.Params("section_id")))
	if err != nil { return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid") }

	// Pastikan section milik masjid (tenant guard)
	var sec secModel.ClassSectionModel
	if err := ctrl.DB.
		Select("class_sections_id, class_sections_masjid_id").
		First(&sec, "class_sections_id = ? AND class_sections_deleted_at IS NULL", secID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Section tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi section")
	}
	if sec.ClassSectionsMasjidID == nil || *sec.ClassSectionsMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Section bukan milik masjid Anda")
	}

	// Jika siswa/ortu → hanya boleh lihat jika user memang terdaftar aktif di section tsb
	if !isAdmin && !isTeacher {
		if userID == uuid.Nil {
			return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
		}
		var cnt int64
		if err := ctrl.DB.Table("user_class_sections AS ucs").
			Joins("JOIN user_classes uc ON uc.user_classes_id = ucs.user_class_sections_user_class_id").
			Where(`
				ucs.user_class_sections_section_id = ?
				AND ucs.user_class_sections_masjid_id = ?
				AND uc.user_classes_user_id = ?
				AND ucs.user_class_sections_unassigned_at IS NULL
				AND uc.user_classes_status = 'active'
				AND uc.user_classes_ended_at IS NULL
			`, secID, masjidID, userID).
			Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek keanggotaan section")
		}
		if cnt == 0 {
			return fiber.NewError(fiber.StatusForbidden, "Anda tidak terdaftar pada section ini")
		}
	}

	// Query dasar
	q := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where("class_attendance_sessions_masjid_id = ? AND class_attendance_sessions_section_id = ?", masjidID, secID)

	// === HANYA filter tanggal kalau param dikirim ===
	df := strings.TrimSpace(c.Query("date_from"))
	dt := strings.TrimSpace(c.Query("date_to"))

	parseDate := func(s string) (time.Time, error) { return time.Parse("2006-01-02", s) } // kolom bertipe DATE aman kirim time.Date
	if df != "" {
		t, e := parseDate(df); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)") }
		q = q.Where("class_attendance_sessions_date >= ?", t)
	}
	if dt != "" {
		t, e := parseDate(dt); if e != nil { return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)") }
		q = q.Where("class_attendance_sessions_date <= ?", t)
	}

	// Pagination
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 { limit = 20 }
	if offset < 0 { offset = 0 }

	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := q.
		Order("class_attendance_sessions_date DESC, class_attendance_sessions_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows { out = append(out, attendanceDTO.FromClassAttendanceSessionModel(r)) }

	return helper.JsonOK(c, "Daftar sesi per section berhasil diambil", fiber.Map{
		"limit":  limit,
		"offset": offset,
		"count":  len(out),
		"items":  out,
	})
}



// ===============================
// LIST by TEACHER (SELF)
// GET /admin/class-attendance-sessions/teacher/me?section_id=&date_from=&date_to=&limit=&offset=
// ===============================
func (ctrl *ClassAttendanceSessionController) ListMyTeachingSessions(c *fiber.Ctx) error {
	// Pastikan pemanggil adalah TEACHER di sebuah masjid (ambil masjidnya dari klaim teacher)
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		// kalau tidak punya klaim teacher, tolak
		return fiber.NewError(fiber.StatusUnauthorized, "masjid_teacher_ids tidak ditemukan di token")
	}
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
	}

	q := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where("class_attendance_sessions_masjid_id = ? AND class_attendance_sessions_teacher_user_id = ?", masjidID, userID)

	// ----- Filter tanggal (default: hari ini & besok) -----
	df := strings.TrimSpace(c.Query("date_from"))
	dt := strings.TrimSpace(c.Query("date_to"))

	if df == "" && dt == "" {
		// kolom bertipe DATE → pakai string YYYY-MM-DD
		today := time.Now().Format("2006-01-02")
		tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
		q = q.Where("class_attendance_sessions_date BETWEEN ? AND ?", today, tomorrow)
	} else {
		parse := func(s string) (string, error) {
			t, err := time.Parse("2006-01-02", s)
			if err != nil { return "", err }
			return t.Format("2006-01-02"), nil
		}
		if df != "" {
			if s, err := parse(df); err == nil {
				q = q.Where("class_attendance_sessions_date >= ?", s)
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
			}
		}
		if dt != "" {
			if s, err := parse(dt); err == nil {
				q = q.Where("class_attendance_sessions_date <= ?", s)
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
			}
		}
	}

	// ----- Filter opsional -----
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		if sid, err := uuid.Parse(s); err == nil {
			q = q.Where("class_attendance_sessions_section_id = ?", sid)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
	}

	// ----- Pagination -----
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 { limit = 20 }
	if offset < 0 { offset = 0 }

	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := q.
		Order("class_attendance_sessions_date DESC, class_attendance_sessions_created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, attendanceDTO.FromClassAttendanceSessionModel(r))
	}

	return helper.JsonOK(c, "Daftar sesi mengajar (by token) berhasil diambil", fiber.Map{
		"limit":  limit,
		"offset": offset,
		"count":  len(resp),
		"items":  resp,
	})
}



// GET /admin/class-attendance-sessions/by-masjid?masjid_id=&date_from=&date_to=&sort=&limit=&offset=
func (ctrl *ClassAttendanceSessionController) ListByMasjid(c *fiber.Ctx) error {
	// Role & tenant guard
	adminMasjidID, _ := helper.GetMasjidIDFromToken(c)
	teacherMasjidID, _ := helper.GetTeacherMasjidIDFromToken(c)

	isAdmin := adminMasjidID != uuid.Nil
	isTeacher := teacherMasjidID != uuid.Nil
	if !isAdmin && !isTeacher {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// Tentukan masjid target:
	// - Admin boleh kirim ?masjid_id (harus sama dengan token-nya)
	// - Teacher dipaksa ke masjid klaim teacher
	targetMasjidID := uuid.Nil
	if isTeacher {
		targetMasjidID = teacherMasjidID
	}
	if isAdmin {
		targetMasjidID = adminMasjidID
		if s := strings.TrimSpace(c.Query("masjid_id")); s != "" {
			if qid, err := uuid.Parse(s); err == nil {
				if qid != adminMasjidID {
					return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses masjid lain")
				}
				targetMasjidID = qid
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "masjid_id tidak valid")
			}
		}
	}

	q := ctrl.DB.Model(&attendanceModel.ClassAttendanceSessionModel{}).
		Where("class_attendance_sessions_masjid_id = ?", targetMasjidID)

	// ====== Filter tanggal ======
	// Default: hari ini → ke depan
	df := strings.TrimSpace(c.Query("date_from"))
	dt := strings.TrimSpace(c.Query("date_to"))
	parse := func(s string) (string, error) {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return "", err
		}
		return t.Format("2006-01-02"), nil
	}

	if df == "" && dt == "" {
		df = time.Now().Format("2006-01-02")
	}

	if df != "" {
		if s, err := parse(df); err == nil {
			q = q.Where("class_attendance_sessions_date >= ?", s)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
		}
	}
	if dt != "" {
		if s, err := parse(dt); err == nil {
			q = q.Where("class_attendance_sessions_date <= ?", s)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
		}
	}

	// ====== Sort ======
	// ?sort=asc | desc  (default asc: terdekat dulu)
	sort := strings.ToLower(strings.TrimSpace(c.Query("sort", "asc")))
	orderClause := "class_attendance_sessions_date ASC, class_attendance_sessions_created_at DESC"
	if sort == "desc" {
		orderClause = "class_attendance_sessions_date DESC, class_attendance_sessions_created_at DESC"
	}

	// ====== Pagination ======
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var rows []attendanceModel.ClassAttendanceSessionModel
	if err := q.Order(orderClause).Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	items := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		items = append(items, attendanceDTO.FromClassAttendanceSessionModel(r))
	}

	return helper.JsonOK(c, "Daftar sesi per masjid berhasil diambil", fiber.Map{
		"limit":  limit,
		"offset": offset,
		"count":  len(items),
		"items":  items,
	})
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