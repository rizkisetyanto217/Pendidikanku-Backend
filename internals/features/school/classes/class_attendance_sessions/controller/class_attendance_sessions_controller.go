// internals/features/lembaga/class_sections/attendance_sessions/controller/class_attendance_sessions_user_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"

	attendanceDTO "schoolku_backend/internals/features/school/classes/class_attendance_sessions/dto"
	attendanceModel "schoolku_backend/internals/features/school/classes/class_attendance_sessions/model"
	helperOSS "schoolku_backend/internals/helpers/oss"

	snapshotTeacher "schoolku_backend/internals/features/lembaga/school_yayasans/teachers_students/snapshot"
	snapshotClassRoom "schoolku_backend/internals/features/school/academics/rooms/snapshot"
	serviceSchedule "schoolku_backend/internals/features/school/classes/class_schedules/services"
	snapshotCSST "schoolku_backend/internals/features/school/classes/class_section_subject_teachers/snapshot"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
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

// --- helper kecil di atas file (boleh taruh di luar fungsi) ---
func getCSSTName(tx *gorm.DB, csstID uuid.UUID) (string, error) {
	var row struct {
		Name *string `gorm:"column:name"`
	}
	const q = `
SELECT
  COALESCE(class_section_subject_teacher_name, name) AS name
FROM class_section_subject_teachers
WHERE class_section_subject_teacher_id = ?
  AND (class_section_subject_teacher_deleted_at IS NULL OR class_section_subject_teacher_deleted_at IS NULL)
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

		// URLs via JSON field
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

	// ✅ Coerce zero-UUID schedule → nil (DTO Normalize)
	req.Normalize()

	// Validasi payload (sesuai tag DTO)
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

		// 3) Cek duplikasi aktif (school, date, [schedule nullable])
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

		// --- Auto-Title (opsional) ---
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

				title := fmt.Sprintf("%s pertemuan ke-%d", baseName, n)
				req.ClassAttendanceSessionTitle = &title
			}
		}

		// ====== SNAPSHOTS & EFEKTIF ASSIGNMENTS ======
		var (
			effCSSTID    *uuid.UUID
			effTeacherID *uuid.UUID
			effRoomID    *uuid.UUID

			csstSnapJSON    datatypes.JSONMap
			teacherSnapJSON datatypes.JSONMap
			roomSnapJSON    datatypes.JSONMap
		)

		// 1) CSST efektif
		if req.ClassAttendanceSessionCSSTId != nil && *req.ClassAttendanceSessionCSSTId != uuid.Nil {
			effCSSTID = req.ClassAttendanceSessionCSSTId

			if cs, err := snapshotCSST.ValidateAndSnapshotCSST(tx, schoolID, *effCSSTID); err == nil {
				jb := snapshotCSST.ToJSON(cs)
				var mm map[string]any
				_ = json.Unmarshal(jb, &mm)
				csstSnapJSON = datatypes.JSONMap(mm)

				// override teacher dari CSST jika ada
				if cs.TeacherID != nil {
					req.ClassAttendanceSessionTeacherId = cs.TeacherID
				}
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "CSST tidak valid / bukan milik school Anda")
			}
		}

		// 2) Teacher efektif
		if req.ClassAttendanceSessionTeacherId != nil && *req.ClassAttendanceSessionTeacherId != uuid.Nil {
			effTeacherID = req.ClassAttendanceSessionTeacherId

			if ts, err := snapshotTeacher.ValidateAndSnapshotTeacher(tx, schoolID, *effTeacherID); err == nil {
				jb := snapshotTeacher.ToJSON(ts)
				var mm map[string]any
				_ = json.Unmarshal(jb, &mm)
				teacherSnapJSON = datatypes.JSONMap(mm)
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "Guru tidak valid / bukan milik school Anda")
			}
		}

		// 3) Room efektif
		if effCSSTID != nil {
			gen := &serviceSchedule.Generator{DB: tx}
			roomID, roomSnap, rerr := gen.ResolveRoomFromCSSTOrSection(
				c.Context(),
				schoolID,
				effCSSTID,
			)
			if rerr == nil && roomID != nil {
				effRoomID = roomID
				if roomSnap != nil {
					roomSnapJSON = roomSnap
				}
			}
		}
		if effRoomID == nil && req.ClassAttendanceSessionClassRoomId != nil && *req.ClassAttendanceSessionClassRoomId != uuid.Nil {
			rid := *req.ClassAttendanceSessionClassRoomId
			if rs, err := snapshotClassRoom.ValidateAndSnapshotRoom(tx, schoolID, rid); err == nil {
				jb := snapshotClassRoom.ToJSON(rs)
				var mm map[string]any
				_ = json.Unmarshal(jb, &mm)
				effRoomID = &rid
				roomSnapJSON = datatypes.JSONMap(mm)
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak valid / bukan milik school Anda")
			}
		}

		// 4) Build model dari DTO
		m := req.ToModel()
		m.ClassAttendanceSessionSchoolID = schoolID
		// (Schedule sudah pointer-aware di ToModel, tak perlu nulis apa-apa lagi)

		if effCSSTID != nil {
			m.ClassAttendanceSessionCSSTID = effCSSTID
			if csstSnapJSON != nil {
				m.ClassAttendanceSessionCSSTSnapshot = csstSnapJSON
			}
		}
		if effTeacherID != nil {
			m.ClassAttendanceSessionTeacherID = effTeacherID
			if teacherSnapJSON != nil {
				m.ClassAttendanceSessionTeacherSnapshot = teacherSnapJSON
			}
		}
		if effRoomID != nil {
			m.ClassAttendanceSessionClassRoomID = effRoomID
			if roomSnapJSON != nil {
				m.ClassAttendanceSessionRoomSnapshot = roomSnapJSON
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

	// Ambil URLs ringkas utk response
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

// PATCH /admin/class-attendance-sessions/:id/urls/:url_id
func (ctrl *ClassAttendanceSessionController) PatchClassAttendanceSessionUrl(c *fiber.Ctx) error {
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
	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		schoolID = id
	case helperAuth.IsTeacher(c):
		if mc.ID != uuid.Nil {
			schoolID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, http.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id
		} else if id, er := helperAuth.GetActiveSchoolID(c); er == nil && id != uuid.Nil {
			schoolID = id
		}
		if schoolID == uuid.Nil || !helperAuth.UserHasSchool(c, schoolID) {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope school tidak valid untuk Teacher")
		}
	default:
		return fiber.NewError(fiber.StatusUnauthorized, "Tidak diizinkan")
	}

	// ✅ Path params
	sessionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Session ID tidak valid")
	}
	urlID, err := uuid.Parse(strings.TrimSpace(c.Params("url_id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "URL ID tidak valid")
	}

	// ✅ Parse + validate payload
	var p attendanceDTO.ClassAttendanceSessionURLPatch
	if err := c.BodyParser(&p); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	p.ID = urlID
	p.Normalize()
	if err := validator.New().Struct(p); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 🔒 TX
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 0) Pastikan sesi target milik tenant & belum deleted
		var sess struct {
			ID       uuid.UUID  `gorm:"column:id"`
			SchoolID uuid.UUID  `gorm:"column:school_id"`
			Deleted  *time.Time `gorm:"column:deleted_at"`
		}
		if err := tx.Table("class_attendance_sessions").
			Select(`
				class_attendance_session_id AS id,
				class_attendance_session_school_id AS school_id,
				class_attendance_session_deleted_at AS deleted_at
			`).
			Where("class_attendance_session_id = ? AND class_attendance_session_school_id = ? AND class_attendance_session_deleted_at IS NULL",
				sessionID, schoolID).
			Take(&sess).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Session tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil session")
		}

		// 1) Load target URL (tenant + owner + live)
		var row attendanceModel.ClassAttendanceSessionURLModel
		if err := tx.Where(`
				class_attendance_session_url_id = ?
				AND class_attendance_session_url_session_id = ?
				AND class_attendance_session_url_school_id = ?
				AND class_attendance_session_url_deleted_at IS NULL
			`, urlID, sessionID, schoolID).
			Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "URL tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil URL")
		}

		patch := map[string]any{}
		kindChanged := false

		// 2) Apply field-by-field
		if p.Label != nil {
			patch["class_attendance_session_url_label"] = strings.TrimSpace(*p.Label)
			row.ClassAttendanceSessionURLLabel = p.Label
		}
		if p.Order != nil {
			patch["class_attendance_session_url_order"] = *p.Order
			row.ClassAttendanceSessionURLOrder = *p.Order
		}
		if p.Kind != nil {
			kind := strings.TrimSpace(*p.Kind)
			if kind == "" {
				kind = "attachment"
			}
			patch["class_attendance_session_url_kind"] = kind
			kindChanged = kindChanged || (row.ClassAttendanceSessionURLKind != kind)
			row.ClassAttendanceSessionURLKind = kind
		}

		// 2a) Href/ObjectKey handling
		if p.Href != nil {
			newHref := strings.TrimSpace(*p.Href)
			if newHref == "" {
				patch["class_attendance_session_url_href"] = nil
				patch["class_attendance_session_url_object_key"] = nil
				row.ClassAttendanceSessionURLHref = nil
				row.ClassAttendanceSessionURLObjectKey = nil
			} else {
				// Pindahkan file lama ke spam (kalau ada lama & object_key ada)
				if row.ClassAttendanceSessionURLHref != nil && row.ClassAttendanceSessionURLObjectKey != nil {
					if spamURL, err := helperOSS.MoveToSpamByPublicURLENV(*row.ClassAttendanceSessionURLHref, 10*time.Second); err == nil {
						_ = spamURL
						patch["class_attendance_session_url_object_key_old"] = *row.ClassAttendanceSessionURLObjectKey
						patch["class_attendance_session_url_delete_pending_until"] = time.Now().Add(7 * 24 * time.Hour)
					}
				}
				patch["class_attendance_session_url_href"] = newHref
				row.ClassAttendanceSessionURLHref = &newHref

				if key, kerr := helperOSS.ExtractKeyFromPublicURL(newHref); kerr == nil && strings.TrimSpace(key) != "" {
					patch["class_attendance_session_url_object_key"] = key
					row.ClassAttendanceSessionURLObjectKey = &key
				} else {
					patch["class_attendance_session_url_object_key"] = nil
					row.ClassAttendanceSessionURLObjectKey = nil
				}
			}
		}

		// If user explicitly sends object_key (manual override)
		if p.ObjectKey != nil {
			okey := strings.TrimSpace(*p.ObjectKey)
			if okey == "" {
				patch["class_attendance_session_url_object_key"] = nil
				row.ClassAttendanceSessionURLObjectKey = nil
			} else {
				patch["class_attendance_session_url_object_key"] = okey
				row.ClassAttendanceSessionURLObjectKey = &okey
			}
		}

		if p.IsPrimary != nil {
			patch["class_attendance_session_url_is_primary"] = *p.IsPrimary
			row.ClassAttendanceSessionURLIsPrimary = *p.IsPrimary
		}

		// 3) Persist patch
		if len(patch) > 0 {
			if err := tx.Model(&attendanceModel.ClassAttendanceSessionURLModel{}).
				Where("class_attendance_session_url_id = ?", row.ClassAttendanceSessionURLID).
				Updates(patch).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui URL")
			}
		}

		// 4) Enforce unique primary per (session, kind) jika:
		// - is_primary diset ke true, ATAU
		// - kind berubah & tetap primary di record ini
		if (p.IsPrimary != nil && *p.IsPrimary) || (kindChanged && row.ClassAttendanceSessionURLIsPrimary) {
			if err := tx.Model(&attendanceModel.ClassAttendanceSessionURLModel{}).
				Where(`
					class_attendance_session_url_school_id = ?
					AND class_attendance_session_url_session_id = ?
					AND class_attendance_session_url_kind = ?
					AND class_attendance_session_url_id <> ?
					AND class_attendance_session_url_deleted_at IS NULL
				`,
					schoolID, sessionID, row.ClassAttendanceSessionURLKind, row.ClassAttendanceSessionURLID).
				Update("class_attendance_session_url_is_primary", false).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal set primary lampiran")
			}
		}

		c.Locals("updated_url", row)
		return nil
	}); err != nil {
		return err
	}

	// ✅ Response
	u := c.Locals("updated_url").(attendanceModel.ClassAttendanceSessionURLModel)
	return helper.JsonUpdated(c, "URL berhasil diperbarui", attendanceDTO.ToClassAttendanceSessionURLLite(&u))
}

/*
=========================================================
DELETE /admin/class-attendance-sessions/:id/urls/:url_id?hard=true
=========================================================
*/
func (ctrl *ClassAttendanceSessionController) DeleteClassAttendanceSessionUrl(c *fiber.Ctx) error {
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}
	mc, er := helperAuth.ResolveSchoolContext(c)
	if er != nil {
		return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
	}

	var schoolID uuid.UUID
	isAdmin := false
	switch {
	case helperAuth.IsOwner(c) || helperAuth.IsDKM(c):
		id, er := helperAuth.EnsureSchoolAccessDKM(c, mc)
		if er != nil {
			return helper.JsonError(c, er.(*fiber.Error).Code, er.Error())
		}
		schoolID = id
		isAdmin = true
	case helperAuth.IsTeacher(c):
		if mc.ID != uuid.Nil {
			schoolID = mc.ID
		} else if strings.TrimSpace(mc.Slug) != "" {
			id, er := helperAuth.GetSchoolIDBySlug(c, mc.Slug)
			if er != nil {
				return helper.JsonError(c, http.StatusNotFound, "School (slug) tidak ditemukan")
			}
			schoolID = id
		} else if id, er := helperAuth.GetActiveSchoolID(c); er == nil && id != uuid.Nil {
			schoolID = id
		}
		if schoolID == uuid.Nil || !helperAuth.UserHasSchool(c, schoolID) {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope school tidak valid untuk Teacher")
		}
	default:
		return fiber.NewError(fiber.StatusUnauthorized, "Tidak diizinkan")
	}

	sessionID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Session ID tidak valid")
	}
	urlID, err := uuid.Parse(strings.TrimSpace(c.Params("url_id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "URL ID tidak valid")
	}

	hard := strings.EqualFold(c.Query("hard"), "true")
	if hard && !isAdmin {
		return fiber.NewError(fiber.StatusForbidden, "Hanya admin yang boleh hard delete")
	}

	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		var row attendanceModel.ClassAttendanceSessionURLModel
		if err := tx.Where(`
				class_attendance_session_url_id = ?
				AND class_attendance_session_url_session_id = ?
				AND class_attendance_session_url_school_id = ?
				AND class_attendance_session_url_deleted_at IS NULL
			`, urlID, sessionID, schoolID).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "URL tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil URL")
		}

		if hard {
			// hard delete: hapus baris & coba hapus objek OSS (best-effort)
			if err := tx.Unscoped().Delete(&attendanceModel.ClassAttendanceSessionURLModel{}, "class_attendance_session_url_id = ?", row.ClassAttendanceSessionURLID).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal hard delete URL")
			}
			if row.ClassAttendanceSessionURLHref != nil {
				_ = helperOSS.DeleteByPublicURLENV(*row.ClassAttendanceSessionURLHref, 10*time.Second) // best-effort
			}
			return nil
		}

		// soft delete: tandai deleted_at; jika punya object_key → set delete_pending_until (dipurge oleh reaper)
		patch := map[string]interface{}{
			"class_attendance_session_url_deleted_at": time.Now(),
		}
		if row.ClassAttendanceSessionURLObjectKey != nil && *row.ClassAttendanceSessionURLObjectKey != "" {
			patch["class_attendance_session_url_delete_pending_until"] = time.Now().Add(30 * 24 * time.Hour)
		}
		if err := tx.Model(&attendanceModel.ClassAttendanceSessionURLModel{}).
			Where("class_attendance_session_url_id = ?", row.ClassAttendanceSessionURLID).
			Updates(patch).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal soft delete URL")
		}
		c.Locals("deleted_url", row)
		return nil
	}); err != nil {
		return err
	}

	return helper.JsonDeleted(c, "URL berhasil dihapus", fiber.Map{"id": urlID})
}
