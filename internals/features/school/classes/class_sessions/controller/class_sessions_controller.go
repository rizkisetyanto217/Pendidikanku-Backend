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

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	attendanceDTO "masjidku_backend/internals/features/school/classes/class_sessions/dto"
	attendanceModel "masjidku_backend/internals/features/school/classes/class_sessions/model"
	helperOSS "masjidku_backend/internals/helpers/oss"

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
POST /admin/class-attendance-sessions
Body: CreateClassAttendanceSessionRequest (pakai SCHEDULE)
Mendukung:
- JSON biasa
- multipart:
  - form fields session
  - urls_json (array upserts)
  - bracket/array style (urls[0][...]/url_kind[] dst.)
  - file uploads (otomatis diupload ke OSS → dibuat baris URL)

=========================================================
*/
func (ctrl *ClassAttendanceSessionController) CreateClassAttendanceSession(c *fiber.Ctx) error {
	// ✅ Role guard
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}

	// ✅ Resolve masjid context
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

	// ---------- Parse payload ----------
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))
	var req attendanceDTO.CreateClassAttendanceSessionRequest

	if strings.HasPrefix(ct, "multipart/form-data") {
		// Wajib
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_schedule_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionScheduleId = id
			}
		}
		// Opsional
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_teacher_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionTeacherId = &id
			}
		}
		// Date (YYYY-MM-DD)
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_date")); v != "" {
			if d, err := time.ParseInLocation("2006-01-02", v, time.Local); err == nil {
				dd := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
				req.ClassAttendanceSessionDate = &dd
			}
		}
		// Metadata
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

		// override event/resources (UUID opsional)
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_override_event_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionOverrideEventId = &id
			}
		}
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_override_attendance_event_id")); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				req.ClassAttendanceSessionOverrideAttendanceEventId = &id
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
	req.ClassAttendanceSessionMasjidId = masjidID
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

	// Validasi payload
	if err := validator.New().Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// ---------- Transaksi ----------
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Validasi SCHEDULE (wajib) & default guru jika kosong
		if req.ClassAttendanceSessionScheduleId == uuid.Nil {
			return fiber.NewError(fiber.StatusBadRequest, "class_attendance_session_schedule_id wajib diisi")
		}
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
		// Default guru
		if req.ClassAttendanceSessionTeacherId == nil {
			req.ClassAttendanceSessionTeacherId = sch.TeacherID
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

		// 3) Cek duplikasi aktif (masjid, schedule, date)
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

		// 4) Simpan sesi
		m := req.ToModel()
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
					ClassAttendanceSessionURLMasjidID:  masjidID,
					ClassAttendanceSessionURLSessionID: m.ClassAttendanceSessionsID,
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

		// (b) dari bracket/array style (helper.URLUpsert)
		if ups, ok := c.Locals("urls_form_upserts").([]helperOSS.URLUpsert); ok && len(ups) > 0 {
			for _, u := range ups {
				u.Normalize()
				row := attendanceModel.ClassAttendanceSessionURLModel{
					ClassAttendanceSessionURLMasjidID:  masjidID,
					ClassAttendanceSessionURLSessionID: m.ClassAttendanceSessionsID,
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
						publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, masjidID, "class_attendance_sessions", fh)
						if uerr != nil {
							return uerr // sudah fiber.Error friendly dari helper
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
								ClassAttendanceSessionURLMasjidID:  masjidID,
								ClassAttendanceSessionURLSessionID: m.ClassAttendanceSessionsID,
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
			if it.ClassAttendanceSessionURLSessionID != m.ClassAttendanceSessionsID {
				return fiber.NewError(fiber.StatusBadRequest, "URL item tidak merujuk ke sesi yang sama")
			}
			if it.ClassAttendanceSessionURLMasjidID != masjidID {
				return fiber.NewError(fiber.StatusBadRequest, "URL item tidak merujuk ke masjid yang sama")
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
							class_attendance_session_url_masjid_id = ?
							AND class_attendance_session_url_session_id = ?
							AND class_attendance_session_url_kind = ?
							AND class_attendance_session_url_id <> ?
							AND class_attendance_session_url_deleted_at IS NULL
						`,
							masjidID, m.ClassAttendanceSessionsID, it.ClassAttendanceSessionURLKind, it.ClassAttendanceSessionURLID,
						).
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
		Where("class_attendance_session_url_session_id = ? AND class_attendance_session_url_deleted_at IS NULL", m.ClassAttendanceSessionsID).
		Order("class_attendance_session_url_order ASC, class_attendance_session_url_created_at ASC").
		Find(&rows)

	for i := range rows {
		lite := attendanceDTO.ToClassAttendanceSessionURLLite(&rows[i])
		// Pastikan Href tidak kosong supaya FE enak render
		if strings.TrimSpace(lite.Href) != "" {
			resp.ClassAttendanceSessionUrls = append(resp.ClassAttendanceSessionUrls, lite)
		}
	}

	c.Set("Location", fmt.Sprintf("/admin/class-attendance-sessions/%s", m.ClassAttendanceSessionsID.String()))
	return helper.JsonCreated(c, "Sesi kehadiran & lampiran berhasil dibuat", resp)
}

// PATCH /admin/class-attendance-sessions/:id/urls/:url_id
func (ctrl *ClassAttendanceSessionController) PatchClassAttendanceSessionUrl(c *fiber.Ctx) error {
	if !(helperAuth.IsOwner(c) || helperAuth.IsDKM(c) || helperAuth.IsTeacher(c)) {
		return fiber.NewError(fiber.StatusUnauthorized, "Hanya admin atau guru yang diizinkan")
	}
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
			return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak valid untuk Teacher")
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

	var p attendanceDTO.ClassAttendanceSessionURLPatch
	if err := c.BodyParser(&p); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	// pastikan ID patch sesuai path
	p.ID = urlID
	p.Normalize()
	if err := validator.New().Struct(p); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// TX
	if err := ctrl.DB.Transaction(func(tx *gorm.DB) error {
		// load target URL (ensure tenant+owner)
		var row attendanceModel.ClassAttendanceSessionURLModel
		if err := tx.Where(`
				class_attendance_session_url_id = ?
				AND class_attendance_session_url_session_id = ?
				AND class_attendance_session_url_masjid_id = ?
				AND class_attendance_session_url_deleted_at IS NULL
			`, urlID, sessionID, masjidID).
			Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "URL tidak ditemukan")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil URL")
		}

		patch := map[string]interface{}{}

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
			row.ClassAttendanceSessionURLKind = kind
		}
		// pergantian href/object_key → simpan lama ke spam (opsional), set *_old + delete_pending_until
		if p.Href != nil {
			newHref := strings.TrimSpace(*p.Href)
			if newHref == "" { // clear
				patch["class_attendance_session_url_href"] = nil
				row.ClassAttendanceSessionURLHref = nil
			} else {
				if row.ClassAttendanceSessionURLHref != nil && row.ClassAttendanceSessionURLObjectKey != nil {
					if spamURL, err := helperOSS.MoveToSpamByPublicURLENV(*row.ClassAttendanceSessionURLHref, 10*time.Second); err == nil {
						patch["class_attendance_session_url_object_key_old"] = *row.ClassAttendanceSessionURLObjectKey
						patch["class_attendance_session_url_delete_pending_until"] = time.Now().Add(7 * 24 * time.Hour)
						_ = spamURL
					}
				}
				patch["class_attendance_session_url_href"] = newHref
				row.ClassAttendanceSessionURLHref = &newHref

				if key, kerr := helperOSS.ExtractKeyFromPublicURL(newHref); kerr == nil {
					patch["class_attendance_session_url_object_key"] = key
					row.ClassAttendanceSessionURLObjectKey = &key
				} else {
					patch["class_attendance_session_url_object_key"] = nil
					row.ClassAttendanceSessionURLObjectKey = nil
				}
			}
		}
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

		if len(patch) > 0 {
			if err := tx.Model(&attendanceModel.ClassAttendanceSessionURLModel{}).
				Where("class_attendance_session_url_id = ?", row.ClassAttendanceSessionURLID).
				Updates(patch).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui URL")
			}
		}

		// enforce primary unik (kalau flag jadi true)
		if p.IsPrimary != nil && *p.IsPrimary {
			if err := tx.Model(&attendanceModel.ClassAttendanceSessionURLModel{}).
				Where(`
					class_attendance_session_url_masjid_id = ?
					AND class_attendance_session_url_session_id = ?
					AND class_attendance_session_url_kind = ?
					AND class_attendance_session_url_id <> ?
					AND class_attendance_session_url_deleted_at IS NULL
				`, masjidID, sessionID, row.ClassAttendanceSessionURLKind, row.ClassAttendanceSessionURLID).
				Update("class_attendance_session_url_is_primary", false).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal set primary lampiran")
			}
		}
		c.Locals("updated_url", row)
		return nil
	}); err != nil {
		return err
	}

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
			return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak valid untuk Teacher")
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
				AND class_attendance_session_url_masjid_id = ?
				AND class_attendance_session_url_deleted_at IS NULL
			`, urlID, sessionID, masjidID).Take(&row).Error; err != nil {
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
