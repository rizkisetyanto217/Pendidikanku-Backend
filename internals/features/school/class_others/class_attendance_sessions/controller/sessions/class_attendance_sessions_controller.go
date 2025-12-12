// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv" // buat parse meeting_number dari multipart
	"strings"
	"time"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	attendanceDTO "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/dto"
	attendanceModel "madinahsalam_backend/internals/features/school/class_others/class_attendance_sessions/model"
	helperOSS "madinahsalam_backend/internals/helpers/oss"

	serviceSchedule "madinahsalam_backend/internals/features/school/class_others/class_schedules/services"
	snapshotCSST "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type ClassAttendanceSessionController struct{ DB *gorm.DB }

func NewClassAttendanceSessionController(db *gorm.DB) *ClassAttendanceSessionController {
	return &ClassAttendanceSessionController{DB: db}
}

/* ========== small helpers ========== */

// Gabungkan tanggal (local) + waktu-of-day (time type dari DB) → timestamptz (local tz)
func combineDateAndTime(date time.Time, tod time.Time) time.Time {
	loc := time.Local
	year, month, day := date.In(loc).Date()
	h, m, s := tod.In(loc).Clock()
	return time.Date(year, month, day, h, m, s, 0, loc)
}

// Parse JSON map dari string (kalau kosong → nil)
func parseJSONMapPtr(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Ambil nama tampilan CSST (fallback name)
func getCSSTName(tx *gorm.DB, csstID uuid.UUID) (string, error) {
	var row struct {
		Name *string `gorm:"column:name"`
	}

	// ✅ pakai cache yang memang ada di model terbaru
	const q = `
SELECT
  COALESCE(
    csst.csst_class_section_name_cache,
    csst.csst_subject_name_cache,
    csst.csst_slug,
    csst.csst_id::text
  ) AS name
FROM class_section_subject_teachers csst
WHERE csst.csst_id = ?
  AND csst.csst_deleted_at IS NULL
LIMIT 1`

	if err := tx.Raw(q, csstID).Scan(&row).Error; err != nil {
		return "", err
	}
	if row.Name == nil {
		return "", nil
	}
	return strings.TrimSpace(*row.Name), nil
}

func (ctrl *ClassAttendanceSessionController) CreateClassAttendanceSession(c *fiber.Ctx) error {
	// ✅ Role guard
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// ✅ Resolve school context
	mc, er := helperAuth.ResolveSchoolContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}

	// ✅ Tentukan schoolID dari context dengan aturan role
	var schoolID uuid.UUID
	isTeacher := false

	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		schoolID = id

	default: // Teacher ⇒ harus member pada school context
		if mc.ID != uuid.Nil {
			schoolID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, http.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id
		} else {
			if id, er := helperAuth.GetActiveSchoolID(c); er == nil && id != uuid.Nil {
				schoolID = id
			}
		}
		if schoolID == uuid.Nil || !helperAuth.UserHasSchool(c, schoolID) {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope school tidak valid untuk Teacher")
		}
		isTeacher = true
	}

	var teacherSchoolID uuid.UUID
	if helperAuth.IsTeacher(c) {
		teacherSchoolID, _ = helperAuth.GetSchoolIDFromTokenPreferTeacher(c)
	}
	userID, _ := helperAuth.GetUserIDFromToken(c)

	// ---------- Parse payload ----------
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	var req attendanceDTO.CreateClassAttendanceSessionRequest

	if strings.HasPrefix(ct, "multipart/form-data") {
		// Schedule (opsional) → pointer
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_schedule_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionScheduleId = &id
			}
		}
		// Opsional lain
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_teacher_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionTeacherId = &id
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_date")); v != "" {
			if d, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
				dd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
				req.ClassAttendanceSessionDate = &dd
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_starts_at")); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				req.ClassAttendanceSessionStartsAt = &t
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_ends_at")); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				req.ClassAttendanceSessionEndsAt = &t
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_slug")); v != "" {
			req.ClassAttendanceSessionSlug = &v
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_title")); v != "" {
			req.ClassAttendanceSessionTitle = &v
		}
		req.ClassAttendanceSessionGeneralInfo = strings.TrimSpace(c.FormValue("class_attendance_session_general_info"))
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_note")); v != "" {
			req.ClassAttendanceSessionNote = &v
		}

		// meeting number dari multipart
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_meeting_number")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				req.ClassAttendanceSessionMeetingNumber = &n
			}
		}

		// Lifecycle (opsional)
		parseBoolPtr := func(name string) *bool {
			if s := strings.TrimSpace(c.FormValue(name)); s != "" {
				b := s == "1" || strings.EqualFold(s, "true")
				return &b
			}
			return nil
		}
		req.ClassAttendanceSessionLocked = parseBoolPtr("class_attendance_session_locked")
		req.ClassAttendanceSessionIsOverride = parseBoolPtr("class_attendance_session_is_override")
		req.ClassAttendanceSessionIsCanceled = parseBoolPtr("class_attendance_session_is_canceled")

		// status & attendance_status (optional enum string)
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_status")); v != "" {
			v = strings.ToLower(v)
			req.ClassAttendanceSessionStatus = &v
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_attendance_status")); v != "" {
			v = strings.ToLower(v)
			req.ClassAttendanceSessionAttendanceStatus = &v
		}

		// override times / kind / reason
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_original_start_at")); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				req.ClassAttendanceSessionOriginalStartAt = &t
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_original_end_at")); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				req.ClassAttendanceSessionOriginalEndAt = &t
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_kind")); v != "" {
			req.ClassAttendanceSessionKind = &v
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_override_reason")); v != "" {
			req.ClassAttendanceSessionOverrideReason = &v
		}

		// override event / resources
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_override_event_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionOverrideEventId = &id
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_class_room_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionClassRoomId = &id
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_csst_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionCSSTId = &id
			}
		}

		// ===== Rule (id saja) =====
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_rule_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionRuleId = &id
			}
		}

		// ===== TYPE =====
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_type_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionTypeId = &id
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_type_snapshot")); v != "" {
			if m, err := parseJSONMapPtr(v); err == nil {
				req.ClassAttendanceSessionTypeSnapshot = m
			} else {
				return helper.JsonError(c, fiber.StatusBadRequest, "type_snapshot tidak valid: "+err.Error())
			}
		}

		// URLs via JSON field (sesuai DTO: URLs []ClassAttendanceSessionURLUpsert)
		var urlsJSON []attendanceDTO.ClassAttendanceSessionURLUpsert
		if uj := strings.TrimSpace(c.FormValue("urls_json")); uj != "" {
			if err := json.Unmarshal([]byte(uj), &urlsJSON); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "urls_json tidak valid: "+err.Error())
			}
		}
		c.Locals("urls_json_upserts", urlsJSON)

		// URLs via bracket/array style
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			ups := helperOSS.ParseURLUpsertsFromMultipart(form, &helperOSS.URLParseOptions{
				BracketPrefix: "urls",
				DefaultKind:   "attachment",
			})
			c.Locals("urls_form_upserts", ups)
		}
	} else {
		// JSON murni
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// Force tenant & normalisasi tanggal + trim
	req.ClassAttendanceSessionSchoolId = schoolID
	if req.ClassAttendanceSessionDate != nil {
		d := req.ClassAttendanceSessionDate.In(time.Local)
		dd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
		req.ClassAttendanceSessionDate = &dd
	}
	if req.ClassAttendanceSessionTitle != nil {
		t := strings.TrimSpace(*req.ClassAttendanceSessionTitle)
		req.ClassAttendanceSessionTitle = &t
	}
	req.ClassAttendanceSessionGeneralInfo = strings.TrimSpace(req.ClassAttendanceSessionGeneralInfo)
	if req.ClassAttendanceSessionNote != nil {
		n := strings.TrimSpace(*req.ClassAttendanceSessionNote)
		req.ClassAttendanceSessionNote = &n
	}

	// ✅ Coerce zero-UUIDs (DTO Normalize)
	req.Normalize()

	// Validasi payload (sesuai tag DTO baru)
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 🔒 Guard opsional: bila schedule kosong, wajib ada csst/teacher (min salah satu)
	if (req.ClassAttendanceSessionScheduleId == nil) &&
		(req.ClassAttendanceSessionCSSTId == nil || *req.ClassAttendanceSessionCSSTId == uuid.Nil) &&
		(req.ClassAttendanceSessionTeacherId == nil || *req.ClassAttendanceSessionTeacherId == uuid.Nil) {
		return fiber.NewError(fiber.StatusBadRequest, "Minimal isi salah satu: schedule_id / csst_id / teacher_id")
	}

	// ---------- Transaksi ----------
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {

		// 1) Validasi SCHEDULE (opsional)
		if req.ClassAttendanceSessionScheduleId != nil {
			var sch struct {
				SchoolID  uuid.UUID  `gorm:"column:school_id"`
				IsActive  bool       `gorm:"column:is_active"`
				DeletedAt *time.Time `gorm:"column:deleted_at"`
			}
			if err := tx.Table("class_schedules").
				Select(`
					class_schedule_school_id  AS school_id,
					class_schedule_is_active  AS is_active,
					class_schedule_deleted_at AS deleted_at
				`).
				Where("class_schedule_id = ?", *req.ClassAttendanceSessionScheduleId).
				Take(&sch).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Schedule tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil schedule")
			}
			if sch.SchoolID != schoolID {
				return fiber.NewError(fiber.StatusForbidden, "Schedule bukan milik school Anda")
			}
			if sch.DeletedAt != nil || !sch.IsActive {
				return fiber.NewError(fiber.StatusBadRequest, "Schedule tidak aktif / sudah dihapus")
			}
		}

		// 2) Validasi TEACHER (opsional)
		if req.ClassAttendanceSessionTeacherId != nil && *req.ClassAttendanceSessionTeacherId != uuid.Nil {
			var row struct {
				SchoolID uuid.UUID `gorm:"column:school_id"`
				UserID   uuid.UUID `gorm:"column:user_id"`
			}
			if err := tx.Table("school_teachers mt").
				Select("mt.school_teacher_school_id AS school_id, mt.school_teacher_user_id AS user_id").
				Where("mt.school_teacher_id = ? AND mt.school_teacher_deleted_at IS NULL", *req.ClassAttendanceSessionTeacherId).
				Take(&row).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(fiber.StatusBadRequest, "Guru (school_teacher) tidak ditemukan")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi guru")
			}
			if row.SchoolID != schoolID {
				return fiber.NewError(fiber.StatusForbidden, "Guru bukan milik school Anda")
			}
			// Jika caller TEACHER → harus milik dirinya
			if isTeacher && teacherSchoolID != uuid.Nil && userID != uuid.Nil && row.UserID != userID {
				return fiber.NewError(fiber.StatusForbidden, "Guru pada payload bukan akun Anda")
			}
		}

		// 2b) Validasi TYPE (opsional) + bangun snapshot jika perlu
		if req.ClassAttendanceSessionTypeId != nil && *req.ClassAttendanceSessionTypeId != uuid.Nil {
			var t struct {
				SchoolID    uuid.UUID  `gorm:"column:school_id"`
				Slug        string     `gorm:"column:slug"`
				Name        string     `gorm:"column:name"`
				Description *string    `gorm:"column:description"`
				Color       *string    `gorm:"column:color"`
				Icon        *string    `gorm:"column:icon"`
				IsActive    bool       `gorm:"column:is_active"`
				DeletedAt   *time.Time `gorm:"column:deleted_at"`

				AllowStudentSelfAttendance bool           `gorm:"column:allow_student_self_attendance"`
				AllowTeacherMarkAttendance bool           `gorm:"column:allow_teacher_mark_attendance"`
				RequireTeacherAttendance   bool           `gorm:"column:require_teacher_attendance"`
				RequireAttendanceReason    pq.StringArray `gorm:"column:require_attendance_reason"`

				AttendanceWindowMode         string `gorm:"column:attendance_window_mode"`
				AttendanceOpenOffsetMinutes  *int   `gorm:"column:attendance_open_offset_minutes"`
				AttendanceCloseOffsetMinutes *int   `gorm:"column:attendance_close_offset_minutes"`
			}

			const qType = `
SELECT
  class_attendance_session_type_school_id   AS school_id,
  class_attendance_session_type_slug        AS slug,
  class_attendance_session_type_name        AS name,
  class_attendance_session_type_description AS description,
  class_attendance_session_type_color       AS color,
  class_attendance_session_type_icon        AS icon,
  class_attendance_session_type_is_active   AS is_active,
  class_attendance_session_type_deleted_at  AS deleted_at,

  class_attendance_session_type_allow_student_self_attendance AS allow_student_self_attendance,
  class_attendance_session_type_allow_teacher_mark_attendance AS allow_teacher_mark_attendance,
  class_attendance_session_type_require_teacher_attendance    AS require_teacher_attendance,
  class_attendance_session_type_require_attendance_reason     AS require_attendance_reason,

  class_attendance_session_type_attendance_window_mode         AS attendance_window_mode,
  class_attendance_session_type_attendance_open_offset_minutes AS attendance_open_offset_minutes,
  class_attendance_session_type_attendance_close_offset_minutes AS attendance_close_offset_minutes
FROM class_attendance_session_types
WHERE class_attendance_session_type_id = ?
LIMIT 1`

			if err := tx.Raw(qType, *req.ClassAttendanceSessionTypeId).Scan(&t).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil tipe sesi")
			}
			if t.SchoolID == uuid.Nil {
				return fiber.NewError(fiber.StatusBadRequest, "Tipe sesi tidak ditemukan")
			}
			if t.SchoolID != schoolID {
				return fiber.NewError(fiber.StatusForbidden, "Tipe sesi bukan milik school Anda")
			}
			if t.DeletedAt != nil || !t.IsActive {
				return fiber.NewError(fiber.StatusBadRequest, "Tipe sesi tidak aktif / sudah dihapus")
			}

			// ✅ snapshot default kalau belum dikirim dari FE
			if req.ClassAttendanceSessionTypeSnapshot == nil {
				requireStates := []string{}
				if len(t.RequireAttendanceReason) > 0 {
					requireStates = append(requireStates, t.RequireAttendanceReason...)
				}

				snap := map[string]any{
					"id":          *req.ClassAttendanceSessionTypeId,
					"slug":        t.Slug,
					"name":        t.Name,
					"description": t.Description,
					"color":       t.Color,
					"icon":        t.Icon,

					"allow_student_self_attendance": t.AllowStudentSelfAttendance,
					"allow_teacher_mark_attendance": t.AllowTeacherMarkAttendance,
					"require_teacher_attendance":    t.RequireTeacherAttendance,
					"require_attendance_reason":     requireStates,

					"attendance_window_mode":          t.AttendanceWindowMode,
					"attendance_open_offset_minutes":  t.AttendanceOpenOffsetMinutes,
					"attendance_close_offset_minutes": t.AttendanceCloseOffsetMinutes,
				}

				req.ClassAttendanceSessionTypeSnapshot = snap
			}
		}

		// 3) Validasi RULE (opsional) + ambil jam start/end untuk auto-time
		var (
			ruleStart time.Time
			ruleEnd   time.Time
			haveRule  bool
		)
		if req.ClassAttendanceSessionRuleId != nil && *req.ClassAttendanceSessionRuleId != uuid.Nil {
			var r struct {
				SchoolID   uuid.UUID `gorm:"column:school_id"`
				ScheduleID uuid.UUID `gorm:"column:schedule_id"`
				StartTime  time.Time `gorm:"column:start_time"`
				EndTime    time.Time `gorm:"column:end_time"`
			}
			const q = `
SELECT
  class_schedule_rule_school_id   AS school_id,
  class_schedule_rule_schedule_id AS schedule_id,
  class_schedule_rule_start_time  AS start_time,
  class_schedule_rule_end_time    AS end_time
FROM class_schedule_rules
WHERE class_schedule_rule_id = ?
  AND class_schedule_rule_deleted_at IS NULL
LIMIT 1`
			if err := tx.Raw(q, *req.ClassAttendanceSessionRuleId).Scan(&r).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil rule")
			}
			if r.SchoolID == uuid.Nil {
				return fiber.NewError(fiber.StatusBadRequest, "Rule tidak ditemukan")
			}
			if r.SchoolID != schoolID {
				return fiber.NewError(fiber.StatusForbidden, "Rule bukan milik school Anda")
			}
			if req.ClassAttendanceSessionScheduleId != nil && r.ScheduleID != *req.ClassAttendanceSessionScheduleId {
				return fiber.NewError(fiber.StatusBadRequest, "Rule tidak cocok dengan schedule yang dipilih")
			}

			ruleStart = r.StartTime
			ruleEnd = r.EndTime
			haveRule = true
		}

		// 4) Cek duplikasi aktif (school, date, [schedule nullable])
		effDate := func() time.Time {
			if req.ClassAttendanceSessionDate != nil {
				return *req.ClassAttendanceSessionDate
			}
			now := time.Now().In(time.Local)
			return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		}()

		var dupeCount int64
		dupe := tx.Table("class_attendance_sessions").
			Where(`
				class_attendance_session_school_id = ?
				AND class_attendance_session_deleted_at IS NULL
				AND class_attendance_session_date = ?
			`, req.ClassAttendanceSessionSchoolId, effDate)

		if req.ClassAttendanceSessionScheduleId != nil {
			dupe = dupe.Where("class_attendance_session_schedule_id = ?", *req.ClassAttendanceSessionScheduleId)
		} else {
			dupe = dupe.Where("class_attendance_session_schedule_id IS NULL")
		}

		if err := dupe.Count(&dupeCount).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal cek duplikasi")
		}
		if dupeCount > 0 {
			return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
		}

		// --- Auto-Title (opsional) + sinkron meeting_number ---
		if (req.ClassAttendanceSessionTitle == nil || strings.TrimSpace(*req.ClassAttendanceSessionTitle) == "") &&
			req.ClassAttendanceSessionCSSTId != nil && *req.ClassAttendanceSessionCSSTId != uuid.Nil {

			// 1) Ambil nama CSST
			baseName, _ := getCSSTName(tx, *req.ClassAttendanceSessionCSSTId)
			if strings.TrimSpace(baseName) != "" {

				// 2) Tentukan timestamp pembanding
				var cmp time.Time
				if req.ClassAttendanceSessionOriginalStartAt != nil {
					cmp = req.ClassAttendanceSessionOriginalStartAt.UTC()
				} else if req.ClassAttendanceSessionDate != nil {
					d := req.ClassAttendanceSessionDate.UTC()
					cmp = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
				} else {
					cmp = time.Now().UTC()
				}

				// 3) Hitung urutan pertemuan
				var n int64
				q := tx.Table("class_attendance_sessions").
					Where(`
						class_attendance_session_csst_id = ?
						AND class_attendance_session_deleted_at IS NULL
						AND COALESCE(class_attendance_session_starts_at, class_attendance_session_date) <= ?
					`, *req.ClassAttendanceSessionCSSTId, cmp)

				if req.ClassAttendanceSessionScheduleId != nil {
					q = q.Where("class_attendance_session_schedule_id = ?", *req.ClassAttendanceSessionScheduleId)
				}

				_ = q.Count(&n).Error
				n++ // sesi yang akan dibuat ini

				// sinkron ke meeting_number kalau belum ada
				var meetingNo int
				if req.ClassAttendanceSessionMeetingNumber != nil && *req.ClassAttendanceSessionMeetingNumber > 0 {
					meetingNo = *req.ClassAttendanceSessionMeetingNumber
				} else {
					meetingNo = int(n)
					req.ClassAttendanceSessionMeetingNumber = &meetingNo
				}

				title := fmt.Sprintf("%s pertemuan ke-%d", baseName, meetingNo)
				req.ClassAttendanceSessionTitle = &title
			}
		}

		// ===== EFEKTIF ASSIGNMENTS (CSST, teacher, room) =====
		var (
			effCSSTID    *uuid.UUID
			effTeacherID *uuid.UUID
			effRoomID    *uuid.UUID
		)

		// 1) CSST efektif → validate & mungkin override teacher
		if req.ClassAttendanceSessionCSSTId != nil && *req.ClassAttendanceSessionCSSTId != uuid.Nil {
			effCSSTID = req.ClassAttendanceSessionCSSTId

			if cs, err := snapshotCSST.ValidateAndCacheCSST(tx, schoolID, *effCSSTID); err == nil {
				// override teacher dari CSST jika ada
				if cs.TeacherID != nil {
					req.ClassAttendanceSessionTeacherId = cs.TeacherID
				}
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "CSST tidak valid / bukan milik school Anda")
			}
		}

		// 2) Teacher efektif (ID saja)
		if req.ClassAttendanceSessionTeacherId != nil && *req.ClassAttendanceSessionTeacherId != uuid.Nil {
			effTeacherID = req.ClassAttendanceSessionTeacherId
		}

		// 3) Room efektif (coba resolve dari CSST/Section; tanpa snapshot)
		if effCSSTID != nil {
			gen := &serviceSchedule.Generator{DB: tx}

			// sebelum: roomID, _, rerr := gen.ResolveRoomFromCSSTOrSection(...)
			roomID, rerr := gen.ResolveRoomFromCSSTOrSection(
				c.Context(),
				schoolID,
				effCSSTID,
			)
			if rerr == nil && roomID != nil {
				effRoomID = roomID
			}
		}

		if effRoomID == nil && req.ClassAttendanceSessionClassRoomId != nil && *req.ClassAttendanceSessionClassRoomId != uuid.Nil {
			rid := *req.ClassAttendanceSessionClassRoomId
			// validasi kepemilikan room
			var rc int64
			if err := tx.Table("class_rooms").
				Where("class_room_id = ? AND class_room_school_id = ? AND class_room_deleted_at IS NULL", rid, schoolID).
				Count(&rc).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang")
			}
			if rc == 0 {
				return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak valid / bukan milik school Anda")
			}
			effRoomID = &rid
		}

		// 4) Build model dari DTO (pakai DTO baru)
		m := req.ToModel()
		m.ClassAttendanceSessionSchoolID = schoolID

		if effCSSTID != nil {
			m.ClassAttendanceSessionCSSTID = effCSSTID
		}
		if effTeacherID != nil {
			m.ClassAttendanceSessionTeacherID = effTeacherID
		}
		if effRoomID != nil {
			m.ClassAttendanceSessionClassRoomID = effRoomID
		}

		// Rule assignment ke model (id saja; snapshot sudah dihapus di schema baru)
		if req.ClassAttendanceSessionRuleId != nil && *req.ClassAttendanceSessionRuleId != uuid.Nil {
			m.ClassAttendanceSessionRuleID = req.ClassAttendanceSessionRuleId
		}

		// TYPE snapshot sudah ada di req dan sudah dimap di ToModel()

		// ===== Auto set starts_at/ends_at dari rule kalau kosong =====
		if haveRule && req.ClassAttendanceSessionDate != nil {
			if m.ClassAttendanceSessionStartsAt == nil {
				t := combineDateAndTime(*req.ClassAttendanceSessionDate, ruleStart)
				m.ClassAttendanceSessionStartsAt = &t
			}
			if m.ClassAttendanceSessionEndsAt == nil {
				t := combineDateAndTime(*req.ClassAttendanceSessionDate, ruleEnd)
				m.ClassAttendanceSessionEndsAt = &t
			}
		}

		// 5) Simpan sesi
		if err := tx.Create(&m).Error; err != nil {
			low := strings.ToLower(err.Error())
			if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
				return fiber.NewError(fiber.StatusConflict, "Sesi kehadiran untuk tanggal tersebut sudah ada")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat sesi kehadiran")
		}

		// ---------- Build URL items ----------
		var urlItems []attendanceModel.ClassAttendanceSessionURLModel

		// (a) dari urls_json (DTO upsert)
		if raws, ok := c.Locals("urls_json_upserts").([]attendanceDTO.ClassAttendanceSessionURLUpsert); ok && len(raws) > 0 {
			for _, u := range raws {
				u.Normalize()
				row := attendanceModel.ClassAttendanceSessionURLModel{
					ClassAttendanceSessionURLSchoolID:  schoolID,
					ClassAttendanceSessionURLSessionID: m.ClassAttendanceSessionID,
					ClassAttendanceSessionURLKind:      u.Kind,
					ClassAttendanceSessionURLLabel:     u.Label,
					ClassAttendanceSessionURLHref:      u.Href,
					ClassAttendanceSessionURLObjectKey: u.ObjectKey,
					ClassAttendanceSessionURLOrder:     u.Order,
					ClassAttendanceSessionURLIsPrimary: u.IsPrimary,
				}
				if strings.TrimSpace(row.ClassAttendanceSessionURLKind) == "" {
					row.ClassAttendanceSessionURLKind = "attachment"
				}
				urlItems = append(urlItems, row)
			}
		}

		// (b) dari bracket/array style
		if ups, ok := c.Locals("urls_form_upserts").([]helperOSS.URLUpsert); ok && len(ups) > 0 {
			for _, u := range ups {
				u.Normalize()
				row := attendanceModel.ClassAttendanceSessionURLModel{
					ClassAttendanceSessionURLSchoolID:  schoolID,
					ClassAttendanceSessionURLSessionID: m.ClassAttendanceSessionID,
					ClassAttendanceSessionURLKind:      u.Kind,
					ClassAttendanceSessionURLLabel:     u.Label,
					ClassAttendanceSessionURLHref:      u.Href,
					ClassAttendanceSessionURLObjectKey: u.ObjectKey,
					ClassAttendanceSessionURLOrder:     u.Order,
					ClassAttendanceSessionURLIsPrimary: u.IsPrimary,
				}
				if strings.TrimSpace(row.ClassAttendanceSessionURLKind) == "" {
					row.ClassAttendanceSessionURLKind = "attachment"
				}
				urlItems = append(urlItems, row)
			}
		}

		// (c) dari files multipart → upload ke OSS → isi href/object_key
		if strings.HasPrefix(ct, "multipart/form-data") {
			if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
				fhs, _ := helperOSS.CollectUploadFiles(form, nil)
				if len(fhs) > 0 {
					oss, oerr := helperOSS.NewOSSServiceFromEnv("")
					if oerr != nil {
						return helper.JsonError(c, fiber.StatusBadGateway, "OSS tidak siap")
					}
					ctx := context.Background()
					for _, fh := range fhs {
						publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, schoolID, "class_attendance_sessions", fh)
						if uerr != nil {
							return uerr
						}
						// Cari slot kosong, jika tak ada buat baru
						var row *attendanceModel.ClassAttendanceSessionURLModel
						for i := range urlItems {
							if urlItems[i].ClassAttendanceSessionURLHref == nil && urlItems[i].ClassAttendanceSessionURLObjectKey == nil {
								row = &urlItems[i]
								break
							}
						}
						if row == nil {
							urlItems = append(urlItems, attendanceModel.ClassAttendanceSessionURLModel{
								ClassAttendanceSessionURLSchoolID:  schoolID,
								ClassAttendanceSessionURLSessionID: m.ClassAttendanceSessionID,
								ClassAttendanceSessionURLKind:      "attachment",
								ClassAttendanceSessionURLOrder:     len(urlItems) + 1,
							})
							row = &urlItems[len(urlItems)-1]
						}
						row.ClassAttendanceSessionURLHref = &publicURL
						if key, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
							row.ClassAttendanceSessionURLObjectKey = &key
						} else {
							row.ClassAttendanceSessionURLObjectKey = nil
						}
						if strings.TrimSpace(row.ClassAttendanceSessionURLKind) == "" {
							row.ClassAttendanceSessionURLKind = "attachment"
						}
					}
				}
			}
		}

		// Konsistensi foreign & tenant
		for _, it := range urlItems {
			if it.ClassAttendanceSessionURLSessionID != m.ClassAttendanceSessionID {
				return fiber.NewError(fiber.StatusBadRequest, "URL item tidak merujuk ke sesi yang sama")
			}
			if it.ClassAttendanceSessionURLSchoolID != schoolID {
				return fiber.NewError(fiber.StatusBadRequest, "URL item tidak merujuk ke school yang sama")
			}
		}

		// Simpan URLs (jika ada)
		if len(urlItems) > 0 {
			if err := tx.Create(&urlItems).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan lampiran")
			}
			// enforce satu primary per (session, kind) yang live
			for _, it := range urlItems {
				if it.ClassAttendanceSessionURLIsPrimary {
					if err := tx.Model(&attendanceModel.ClassAttendanceSessionURLModel{}).
						Where(`
							class_attendance_session_url_school_id = ?
							AND class_attendance_session_url_session_id = ?
							AND class_attendance_session_url_kind = ?
							AND class_attendance_session_url_id <> ?
							AND class_attendance_session_url_deleted_at IS NULL
						`,
							schoolID, m.ClassAttendanceSessionID, it.ClassAttendanceSessionURLKind, it.ClassAttendanceSessionURLID).
						Update("class_attendance_session_url_is_primary", false).Error; err != nil {
						return fiber.NewError(fiber.StatusInternalServerError, "Gagal set primary lampiran")
					}
				}
			}
		}

		c.Locals("created_model", m)
		return nil
	}); err != nil {
		return err
	}

	// ---------- Response ----------
	m := c.Locals("created_model").(attendanceModel.ClassAttendanceSessionModel)
	resp := attendanceDTO.FromClassAttendanceSessionModel(m)

	// Ambil URLs ringkas utk response → isi ke ClassAttendanceSessionUrls (DTO baru)
	var rows []attendanceModel.ClassAttendanceSessionURLModel
	_ = ctrl.DB.
		Where("class_attendance_session_url_session_id = ? AND class_attendance_session_url_deleted_at IS NULL", m.ClassAttendanceSessionID).
		Order("class_attendance_session_url_order ASC, class_attendance_session_url_created_at ASC").
		Find(&rows)

	for i := range rows {
		lite := attendanceDTO.ToClassAttendanceSessionURLLite(&rows[i])
		if strings.TrimSpace(lite.Href) != "" {
			resp.ClassAttendanceSessionUrls = append(resp.ClassAttendanceSessionUrls, lite)
		}
	}

	c.Set("Location", fmt.Sprintf("/admin/class-attendance-sessions/%s", m.ClassAttendanceSessionID.String()))
	return helper.JsonCreated(c, "Sesi kehadiran & lampiran berhasil dibuat", resp)
}
