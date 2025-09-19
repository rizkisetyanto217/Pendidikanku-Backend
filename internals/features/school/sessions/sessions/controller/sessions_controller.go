// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	attendanceDTO "masjidku_backend/internals/features/school/sessions/sessions/dto"
	attendanceModel "masjidku_backend/internals/features/school/sessions/sessions/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClassAttendanceSessionController struct{ DB *gorm.DB }

func NewClassAttendanceSessionController(db *gorm.DB) *ClassAttendanceSessionController {
	return &ClassAttendanceSessionController{DB: db}
}

/* ========== small helpers ========== */

func parseYMDLocal(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, err
	}
	t0 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return &t0, nil
}

/*
	=========================================================
	  LIST by TEACHER (SELF)
	  GET /api/u/sessions/teacher/me?section_id=&schedule_id=&date_from=&date_to=&limit=&offset=&q=

=========================================================
*/
// LIST by TEACHER (SELF)
// GET /api/u/sessions/teacher/me?section_id=&schedule_id=&date_from=&date_to=&limit=&offset=&q=
func (ctrl *ClassAttendanceSessionController) ListMyTeachingSessions(c *fiber.Ctx) error {
	// Hanya guru (atau admin/DKM) yang boleh akses endpoint ini
	if !helperAuth.IsTeacher(c) && !helperAuth.IsDKM(c) && !helperAuth.IsOwner(c) {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya guru (atau admin) yang diizinkan")
	}

	// 🎯 Resolusi context masjid
	mc, er := helperAuth.ResolveMasjidContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}

	var masjidID uuid.UUID
	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		masjidID = id

	default: // Teacher ⇒ wajib member pada masjid context
		if mc.ID != uuid.Nil {
			masjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, http.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			masjidID = id
		} else if id, er := helperAuth.GetActiveMasjidID(c); er == nil && id != uuid.Nil {
			masjidID = id
		}
		if masjidID == uuid.Nil || !helperAuth.UserHasMasjid(c, masjidID) {
			return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak valid untuk Teacher")
		}
	}

	userID, err := helperAuth.GetUserIDFromToken(c)
	if err != nil || userID == uuid.Nil {
		return fiber.NewError(fiber.StatusUnauthorized, "User tidak terautentik")
	}

	// ==== lanjutkan kode asli (pagination, query, mapping) ====
	// Pagination & sorting
	rawQ := string(c.Request().URI().QueryString())
	httpReq := &http.Request{URL: &url.URL{RawQuery: rawQ}}
	p := helper.ParseWith(httpReq, "date", "desc", helper.AdminOpts)

	allowedSort := map[string]string{
		"date":  "cas.class_attendance_sessions_date",
		"title": "cas.class_attendance_sessions_title",
	}
	orderClause, err := p.SafeOrderClause(allowedSort, "date")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "sort_by tidak valid")
	}
	orderExpr := strings.TrimPrefix(orderClause, "ORDER BY ")

	// Rentang tanggal
	df, err := parseYMDLocal(c.Query("date_from"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_from tidak valid (YYYY-MM-DD)")
	}
	dt, err := parseYMDLocal(c.Query("date_to"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_to tidak valid (YYYY-MM-DD)")
	}
	if df != nil && dt != nil && dt.Before(*df) {
		return fiber.NewError(fiber.StatusBadRequest, "date_to harus >= date_from")
	}
	var lo, hi *time.Time
	if df != nil {
		lo = df
	}
	if dt != nil {
		h := dt.Add(24 * time.Hour)
		hi = &h
	}

	db := ctrl.DB
	qBase := db.Table("class_attendance_sessions AS cas").
		Joins(`
			LEFT JOIN masjid_teachers AS mt
			  ON mt.masjid_teacher_id = cas.class_attendance_sessions_teacher_id
			 AND mt.masjid_teacher_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_schedules AS cs
			  ON cs.class_schedule_id = cas.class_attendance_sessions_schedule_id
			 AND cs.class_schedules_deleted_at IS NULL
			 AND cs.class_schedules_is_active
		`).
		Joins(`
			LEFT JOIN masjid_teachers AS mt2
			  ON mt2.masjid_teacher_id = cs.class_schedules_teacher_id
			 AND mt2.masjid_teacher_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN class_section_subject_teachers AS csst
			  ON csst.class_section_subject_teachers_id = cs.class_schedules_csst_id
			 AND csst.class_section_subject_teachers_deleted_at IS NULL
		`).
		Joins(`
			LEFT JOIN masjid_teachers AS mt3
			  ON mt3.masjid_teacher_id = csst.class_section_subject_teachers_teacher_id
			 AND mt3.masjid_teacher_deleted_at IS NULL
		`).
		Where(`
			cas.class_attendance_sessions_masjid_id = ?
			AND cas.class_attendance_sessions_deleted_at IS NULL
			AND (
			     mt.masjid_teacher_user_id = ?
			  OR mt2.masjid_teacher_user_id = ?
			  OR mt3.masjid_teacher_user_id = ?
			)
		`, masjidID, userID, userID, userID)

	// Filter tanggal opsional
	if lo != nil && hi != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ? AND cas.class_attendance_sessions_date < ?", *lo, *hi)
	} else if lo != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date >= ?", *lo)
	} else if hi != nil {
		qBase = qBase.Where("cas.class_attendance_sessions_date < ?", *hi)
	}

	// Opsional: section_id
	if s := strings.TrimSpace(c.Query("section_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "section_id tidak valid")
		}
		qBase = qBase.Where("COALESCE(cs.class_schedules_section_id, csst.class_section_subject_teachers_section_id) = ?", id)
	}

	// Opsional: schedule_id
	if s := strings.TrimSpace(c.Query("schedule_id")); s != "" {
		id, e := uuid.Parse(s)
		if e != nil {
			return fiber.NewError(fiber.StatusBadRequest, "schedule_id tidak valid")
		}
		qBase = qBase.Where("cas.class_attendance_sessions_schedule_id = ?", id)
	}

	// Keyword
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		pat := "%" + q + "%"
		qBase = qBase.Where(`(cas.class_attendance_sessions_title ILIKE ? OR cas.class_attendance_sessions_general_info ILIKE ?)`, pat, pat)
	}

	// Total distinct
	var total int64
	if err := qBase.Session(&gorm.Session{}).Distinct("cas.class_attendance_sessions_id").Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung total data")
	}

	// Data
	type row struct {
		ID         uuid.UUID  `gorm:"column:id"`
		MasjidID   uuid.UUID  `gorm:"column:masjid_id"`
		Date       time.Time  `gorm:"column:date"`
		Title      *string    `gorm:"column:title"`
		General    string     `gorm:"column:general"`
		Note       *string    `gorm:"column:note"`
		TeacherID  *uuid.UUID `gorm:"column:teacher_id"`
		RoomID     *uuid.UUID `gorm:"column:room_id"`
		ScheduleID uuid.UUID  `gorm:"column:schedule_id"`
		SectionID  *uuid.UUID `gorm:"column:section_id"`
		SubjectID  *uuid.UUID `gorm:"column:subject_id"`
		DeletedAt  *time.Time `gorm:"column:deleted_at"`
	}
	var rows []row
	if err := qBase.
		Select(`
			cas.class_attendance_sessions_id         AS id,
			cas.class_attendance_sessions_masjid_id  AS masjid_id,
			cas.class_attendance_sessions_date       AS date,
			cas.class_attendance_sessions_title      AS title,
			cas.class_attendance_sessions_general_info AS general,
			cas.class_attendance_sessions_note       AS note,
			cas.class_attendance_sessions_teacher_id AS teacher_id,
			cas.class_attendance_sessions_class_room_id AS room_id,
			cas.class_attendance_sessions_schedule_id   AS schedule_id,
			cas.class_attendance_sessions_deleted_at AS deleted_at,
			COALESCE(cs.class_schedules_section_id, csst.class_section_subject_teachers_section_id) AS section_id,
			COALESCE(cs.class_schedules_class_subject_id, csst.class_section_subject_teachers_class_subjects_id) AS subject_id
		`).
		Order(orderExpr).
		Order("cas.class_attendance_sessions_date DESC, cas.class_attendance_sessions_id DESC").
		Limit(p.Limit()).
		Offset(p.Offset()).
		Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]attendanceDTO.ClassAttendanceSessionResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, attendanceDTO.ClassAttendanceSessionResponse{
			ClassAttendanceSessionId:          r.ID,
			ClassAttendanceSessionMasjidId:    r.MasjidID,
			ClassAttendanceSessionScheduleId:  r.ScheduleID,
			ClassAttendanceSessionClassRoomId: r.RoomID,
			ClassAttendanceSessionDate:        r.Date,
			ClassAttendanceSessionTitle:       r.Title,
			ClassAttendanceSessionGeneralInfo: r.General,
			ClassAttendanceSessionNote:        r.Note,
			ClassAttendanceSessionTeacherId:   r.TeacherID,
			ClassAttendanceSessionDeletedAt:   r.DeletedAt,
			ClassSectionId:                    r.SectionID,
			ClassSubjectId:                    r.SubjectID,
		})
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, resp, meta)
}

/*
=========================================================

	POST /admin/class-attendance-sessions
	Body: CreateClassAttendanceSessionRequest (pakai SCHEDULE)

=========================================================
*/
func (ctrl *ClassAttendanceSessionController) CreateClassAttendanceSession(c *fiber.Ctx) error {
	// ✅ Role guard
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// ✅ Resolve masjid context (path/header/cookie/query/subdomain/token)
	mc, er := helperAuth.ResolveMasjidContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}

	// ✅ Tentukan masjidID dari context dengan aturan role
	var masjidID uuid.UUID
	isTeacher := false

	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		masjidID = id

	default: // Teacher ⇒ harus member pada masjid context
		if mc.ID != uuid.Nil {
			masjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, http.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			masjidID = id
		} else {
			if id, er := helperAuth.GetActiveMasjidID(c); er == nil && id != uuid.Nil {
				masjidID = id
			}
		}
		if masjidID == uuid.Nil || !helperAuth.UserHasMasjid(c, masjidID) {
			return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak valid untuk Teacher")
		}
		isTeacher = true
	}

	// Info user (dipakai untuk self-check guru)
	teacherMasjidID, _ := helperAuth.GetTeacherMasjidIDFromToken(c)
	userID, _ := helperAuth.GetUserIDFromToken(c)

	// Parse payload
	var req attendanceDTO.CreateClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Force tenant & normalisasi tanggal
	req.ClassAttendanceSessionMasjidId = masjidID
	if req.ClassAttendanceSessionDate != nil {
		d := req.ClassAttendanceSessionDate.In(time.Local)
		dd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
		req.ClassAttendanceSessionDate = &dd
	}

	// Trim
	if req.ClassAttendanceSessionTitle != nil {
		t := strings.TrimSpace(*req.ClassAttendanceSessionTitle)
		req.ClassAttendanceSessionTitle = &t
	}
	req.ClassAttendanceSessionGeneralInfo = strings.TrimSpace(req.ClassAttendanceSessionGeneralInfo)
	if req.ClassAttendanceSessionNote != nil {
		n := strings.TrimSpace(*req.ClassAttendanceSessionNote)
		req.ClassAttendanceSessionNote = &n
	}

	// Validasi payload
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Transaksi (isi logika sama seperti sebelumnya)
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Validasi SCHEDULE (wajib) & default guru/room jika kosong
		if req.ClassAttendanceSessionScheduleId == uuid.Nil {
			return fiber.NewError(fiber.StatusBadRequest, "class_attendance_session_schedule_id wajib diisi")
		}
		var sch struct {
			MasjidID  uuid.UUID  `gorm:"column:masjid_id"`
			TeacherID *uuid.UUID `gorm:"column:teacher_id"`
			RoomID    *uuid.UUID `gorm:"column:room_id"`
			IsActive  bool       `gorm:"column:is_active"`
			DeletedAt *time.Time `gorm:"column:deleted_at"`
		}
		if err := tx.Table("class_schedules").
			Select(`
				class_schedules_masjid_id    AS masjid_id,
				class_schedules_teacher_id   AS teacher_id,
				class_schedules_room_id      AS room_id,
				class_schedules_is_active    AS is_active,
				class_schedules_deleted_at   AS deleted_at
			`).
			Where("class_schedule_id = ?", req.ClassAttendanceSessionScheduleId).
			Take(&sch).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusBadRequest, "Schedule tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil schedule")
		}
		if sch.MasjidID != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Schedule bukan milik masjid Anda")
		}
		if sch.DeletedAt != nil || !sch.IsActive {
			return fiber.NewError(fiber.StatusBadRequest, "Schedule tidak aktif / sudah dihapus")
		}

		// Default guru/room jika kosong
		if req.ClassAttendanceSessionTeacherId == nil {
			req.ClassAttendanceSessionTeacherId = sch.TeacherID
		}
		if req.ClassAttendanceSessionClassRoomId == nil {
			req.ClassAttendanceSessionClassRoomId = sch.RoomID
		}

		// 2) Validasi TEACHER (opsional)
		if req.ClassAttendanceSessionTeacherId != nil {
			var row struct {
				MasjidID uuid.UUID `gorm:"column:masjid_id"`
				UserID   uuid.UUID `gorm:"column:user_id"`
			}
			if err := tx.Table("masjid_teachers mt").
				Select("mt.masjid_teacher_masjid_id AS masjid_id, mt.masjid_teacher_user_id AS user_id").
				Where("mt.masjid_teacher_id = ? AND mt.masjid_teacher_deleted_at IS NULL", *req.ClassAttendanceSessionTeacherId).
				Take(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Guru (masjid_teacher) tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi guru")
			}
			if row.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik masjid Anda")
			}
			// Jika caller TEACHER → harus milik dirinya
			if isTeacher && teacherMasjidID != uuid.Nil && userID != uuid.Nil && row.UserID != userID {
				return fiber.NewError(fiber.StatusForbidden, "Guru pada payload bukan akun Anda")
			}
		}

		// 3) Validasi ROOM (opsional)
		if req.ClassAttendanceSessionClassRoomId != nil {
			var room struct {
				MasjidID  uuid.UUID  `gorm:"column:class_rooms_masjid_id"`
				DeletedAt *time.Time `gorm:"column:deleted_at"`
			}
			if err := tx.Table("class_rooms").
				Select("class_rooms_masjid_id, deleted_at").
				Where("class_room_id = ?", *req.ClassAttendanceSessionClassRoomId).
				Take(&room).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang")
			}
			if room.DeletedAt != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas sudah dihapus")
			}
			if room.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Ruang bukan milik masjid Anda")
			}
		}

		// 4) Cek duplikasi aktif (masjid, schedule, date)
		effDate := func() time.Time {
			if req.ClassAttendanceSessionDate != nil {
				return *req.ClassAttendanceSessionDate
			}
			now := time.Now().In(time.Local)
			return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		}()

		var dupeCount int64
		if err := tx.Table("class_attendance_sessions").
			Where(`
				class_attendance_sessions_masjid_id = ?
				AND class_attendance_sessions_schedule_id = ?
				AND class_attendance_sessions_date = ?
				AND class_attendance_sessions_deleted_at IS NULL
			`,
				req.ClassAttendanceSessionMasjidId,
				req.ClassAttendanceSessionScheduleId,
				effDate,
			).
			Count(&dupeCount).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if dupeCount > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// 5) Simpan
		m := req.ToModel()
		if err := tx.Create(&m).Error; err != nil {
			low := strings.ToLower(err.Error())
			if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi kehadiran")
		}

		c.Locals("created_model", m)
		return nil
	}); err != nil {
		return err
	}

	m := c.Locals("created_model").(attendanceModel.ClassAttendanceSessionModel)
	resp := attendanceDTO.FromClassAttendanceSessionModel(m)
	c.Set("Location", fmt.Sprintf("/admin/class-attendance-sessions/%s", m.ClassAttendanceSessionId.String()))
	return helper.JsonCreated(c, "Sesi kehadiran berhasil dibuat", resp)
}

/*
=========================================================

	PUT /admin/class-attendance-sessions/:id
	(pakai schedule; konsistensi lewat schedule)

=========================================================
*/
func (ctrl *ClassAttendanceSessionController) UpdateClassAttendanceSession(c *fiber.Ctx) error {
	// ✅ Role guard
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// ✅ Resolve masjid context (path/header/cookie/query/subdomain/token)
	mc, er := helperAuth.ResolveMasjidContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}

	// ✅ Tentukan masjidID dari context dengan aturan role
	var masjidID uuid.UUID
	isTeacher := false

	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		masjidID = id

	default: // Teacher ⇒ harus member pada masjid context
		if mc.ID != uuid.Nil {
			masjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, http.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			masjidID = id
		} else {
			if id, er := helperAuth.GetActiveMasjidID(c); er == nil && id != uuid.Nil {
				masjidID = id
			}
		}
		if masjidID == uuid.Nil || !helperAuth.UserHasMasjid(c, masjidID) {
			return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak valid untuk Teacher")
		}
		isTeacher = true
	}

	userID, _ := helperAuth.GetUserIDFromToken(c)

	sessionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var req attendanceDTO.UpdateClassAttendanceSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// enforce tenant
	req.ClassAttendanceSessionMasjidId = &masjidID
	// trim
	if req.ClassAttendanceSessionTitle != nil {
		t := strings.TrimSpace(*req.ClassAttendanceSessionTitle)
		req.ClassAttendanceSessionTitle = &t
	}
	if req.ClassAttendanceSessionGeneralInfo != nil {
		g := strings.TrimSpace(*req.ClassAttendanceSessionGeneralInfo)
		req.ClassAttendanceSessionGeneralInfo = &g
	}
	if req.ClassAttendanceSessionNote != nil {
		n := strings.TrimSpace(*req.ClassAttendanceSessionNote)
		req.ClassAttendanceSessionNote = &n
	}
	// normalize date
	if req.ClassAttendanceSessionDate != nil {
		d := req.ClassAttendanceSessionDate.In(time.Local)
		dd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
		req.ClassAttendanceSessionDate = &dd
	}
	// validate
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Tx (isi logika sama seperti sebelumnya)
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// load existing
		var existing attendanceModel.ClassAttendanceSessionModel
		if err := tx.First(&existing, "class_attendance_sessions_id = ?", sessionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if existing.ClassAttendanceSessionMasjidId != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah sesi milik masjid lain")
		}

		// Validasi SCHEDULE (bila diubah)
		if req.ClassAttendanceSessionScheduleId != nil {
			var sch struct {
				MasjidID  uuid.UUID  `gorm:"column:masjid_id"`
				TeacherID *uuid.UUID `gorm:"column:teacher_id"`
				IsActive  bool       `gorm:"column:is_active"`
				DeletedAt *time.Time `gorm:"column:deleted_at"`
			}
			if err := tx.Table("class_schedules").
				Select(`
					class_schedules_masjid_id  AS masjid_id,
					class_schedules_teacher_id AS teacher_id,
					class_schedules_is_active  AS is_active,
					class_schedules_deleted_at AS deleted_at
				`).
				Where("class_schedule_id = ?", *req.ClassAttendanceSessionScheduleId).
				Take(&sch).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Schedule tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil schedule")
			}
			if sch.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Schedule bukan milik masjid Anda")
			}
			if sch.DeletedAt != nil || !sch.IsActive {
				return fiber.NewError(fiber.StatusBadRequest, "Schedule tidak aktif / sudah dihapus")
			}
			// sinkron teacher jika kosong (opsional)
			if req.ClassAttendanceSessionTeacherId == nil {
				req.ClassAttendanceSessionTeacherId = sch.TeacherID
			}
		}

		// Validasi TEACHER (bila diubah)
		if req.ClassAttendanceSessionTeacherId != nil {
			var row struct {
				MasjidID uuid.UUID `gorm:"column:masjid_id"`
				UserID   uuid.UUID `gorm:"column:user_id"`
			}
			if err := tx.Table("masjid_teachers mt").
				Select("mt.masjid_teacher_masjid_id AS masjid_id, mt.masjid_teacher_user_id AS user_id").
				Where("mt.masjid_teacher_id = ? AND mt.masjid_teacher_deleted_at IS NULL", *req.ClassAttendanceSessionTeacherId).
				Take(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Guru (masjid_teacher) tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi guru")
			}
			if row.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik masjid Anda")
			}
			if isTeacher && userID != uuid.Nil && row.UserID != userID {
				return fiber.NewError(fiber.StatusForbidden, "Guru pada payload bukan akun Anda")
			}
		}

		// Validasi ROOM (bila diubah)
		if req.ClassAttendanceSessionClassRoomId != nil {
			var room struct {
				MasjidID  uuid.UUID  `gorm:"column:class_rooms_masjid_id"`
				DeletedAt *time.Time `gorm:"column:deleted_at"`
			}
			if err := tx.Table("class_rooms").
				Select("class_rooms_masjid_id, deleted_at").
				Where("class_room_id = ?", *req.ClassAttendanceSessionClassRoomId).
				Take(&room).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang")
			}
			if room.DeletedAt != nil {
				return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas sudah dihapus")
			}
			if room.MasjidID != masjidID {
				return fiber.NewError(fiber.StatusForbidden, "Ruang bukan milik masjid Anda")
			}
		}

		// Cek unik (masjid, schedule, date) excluding current
		targetDate := func() time.Time {
			if req.ClassAttendanceSessionDate != nil {
				return *req.ClassAttendanceSessionDate
			}
			if existing.ClassAttendanceSessionDate != nil {
				return *existing.ClassAttendanceSessionDate
			}
			now := time.Now().In(time.Local)
			return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		}()
		targetSchedule := existing.ClassAttendanceSessionScheduleId
		if req.ClassAttendanceSessionScheduleId != nil {
			targetSchedule = *req.ClassAttendanceSessionScheduleId
		}
		var cnt int64
		if err := tx.Table("class_attendance_sessions").
			Where(`
				class_attendance_sessions_masjid_id = ?
				AND class_attendance_sessions_schedule_id = ?
				AND class_attendance_sessions_date = ?
				AND class_attendance_sessions_id <> ?
				AND class_attendance_sessions_deleted_at IS NULL
			`,
				masjidID, targetSchedule, targetDate, existing.ClassAttendanceSessionId,
			).Count(&cnt).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// Patch
		patch := map[string]interface{}{
			"class_attendance_sessions_masjid_id": masjidID,
		}
		if req.ClassAttendanceSessionDate != nil {
			patch["class_attendance_sessions_date"] = *req.ClassAttendanceSessionDate
			existing.ClassAttendanceSessionDate = req.ClassAttendanceSessionDate
		}
		if req.ClassAttendanceSessionTitle != nil {
			t := strings.TrimSpace(*req.ClassAttendanceSessionTitle)
			patch["class_attendance_sessions_title"] = t
			existing.ClassAttendanceSessionTitle = &t
		}
		if req.ClassAttendanceSessionGeneralInfo != nil {
			g := strings.TrimSpace(*req.ClassAttendanceSessionGeneralInfo)
			patch["class_attendance_sessions_general_info"] = g
			existing.ClassAttendanceSessionGeneralInfo = g
		}
		if req.ClassAttendanceSessionNote != nil {
			n := strings.TrimSpace(*req.ClassAttendanceSessionNote)
			patch["class_attendance_sessions_note"] = n
			existing.ClassAttendanceSessionNote = &n
		}
		if req.ClassAttendanceSessionScheduleId != nil {
			patch["class_attendance_sessions_schedule_id"] = *req.ClassAttendanceSessionScheduleId
			existing.ClassAttendanceSessionScheduleId = *req.ClassAttendanceSessionScheduleId
		}
		if req.ClassAttendanceSessionTeacherId != nil {
			patch["class_attendance_sessions_teacher_id"] = *req.ClassAttendanceSessionTeacherId
			existing.ClassAttendanceSessionTeacherId = req.ClassAttendanceSessionTeacherId
		}
		if req.ClassAttendanceSessionClassRoomId != nil {
			patch["class_attendance_sessions_class_room_id"] = *req.ClassAttendanceSessionClassRoomId
			existing.ClassAttendanceSessionClassRoomId = req.ClassAttendanceSessionClassRoomId
		}

		if err := tx.Model(&attendanceModel.ClassAttendanceSessionModel{}).
			Where("class_attendance_sessions_id = ?", existing.ClassAttendanceSessionId).
			Select([]string{
				"class_attendance_sessions_masjid_id",
				"class_attendance_sessions_date",
				"class_attendance_sessions_title",
				"class_attendance_sessions_general_info",
				"class_attendance_sessions_note",
				"class_attendance_sessions_schedule_id",
				"class_attendance_sessions_teacher_id",
				"class_attendance_sessions_class_room_id",
			}).
			Updates(patch).Error; err != nil {

			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui sesi kehadiran")
		}

		c.Locals("updated_model", existing)
		return nil
	}); err != nil {
		return err
	}

	updated := c.Locals("updated_model").(attendanceModel.ClassAttendanceSessionModel)
	resp := attendanceDTO.FromClassAttendanceSessionModel(updated)
	return helper.JsonUpdated(c, "Sesi kehadiran berhasil diperbarui", resp)
}

/*
	=========================================================
	  DELETE /admin/class-attendance-sessions/:id?force=true

=========================================================
*/
// DELETE /admin/class-attendance-sessions/:id?force=true
func (ctrl *ClassAttendanceSessionController) DeleteClassAttendanceSession(c *fiber.Ctx) error {
	// Role & tenant via helpers
	mc, er := helperAuth.ResolveMasjidContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}

	var masjidID uuid.UUID
	isAdmin := false

	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureMasjidAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		masjidID = id
		isAdmin = true

	case helperAuth.IsTeacher(c):
		if mc.ID != uuid.Nil {
			masjidID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, http.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			masjidID = id
		} else if id, er := helperAuth.GetActiveMasjidID(c); er == nil && id != uuid.Nil {
			masjidID = id
		}
		if masjidID == uuid.Nil || !helperAuth.UserHasMasjid(c, masjidID) {
			return helper.JsonError(c, http.StatusForbidden, "Scope masjid tidak valid untuk Teacher")
		}

	default:
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	sessionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// force delete khusus admin/DKM
	force := strings.EqualFold(c.Query("force"), "true")
	if force && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	// === lanjut kode asli (TX delete) ===
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		var existing attendanceModel.ClassAttendanceSessionModel
		if err := tx.First(&existing, "class_attendance_sessions_id = ?", sessionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Sesi tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
		}
		if existing.ClassAttendanceSessionMasjidId != masjidID {
			return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus sesi milik masjid lain")
		}

		var delErr error
		if force {
			delErr = tx.Unscoped().Delete(&attendanceModel.ClassAttendanceSessionModel{}, "class_attendance_sessions_id = ?", sessionID).Error
		} else {
			delErr = tx.Delete(&attendanceModel.ClassAttendanceSessionModel{}, "class_attendance_sessions_id = ?", sessionID).Error
		}
		if delErr != nil {
			msg := strings.ToLower(delErr.Error())
			if strings.Contains(msg, "constraint") || strings.Contains(msg, "foreign") || strings.Contains(msg, "violat") {
				return fiber.NewError(fiber.StatusBadRequest, "Tidak dapat menghapus karena masih ada data terkait")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus sesi kehadiran")
		}

		c.Locals("deleted_model", existing)
		return nil
	}); err != nil {
		return err
	}

	deleted := c.Locals("deleted_model").(attendanceModel.ClassAttendanceSessionModel)
	resp := attendanceDTO.FromClassAttendanceSessionModel(deleted)
	return helper.JsonDeleted(c, "Sesi kehadiran berhasil dihapus", resp)
}
