// file: internals/features/school/class_attendance_sessions/controller/class_attendance_session_url_controller.go
package controller

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	sessiondto "masjidku_backend/internals/features/school/sessions_assesment/sessions/dto"
	sessionmodel "masjidku_backend/internals/features/school/sessions_assesment/sessions/model"
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"
)

type ClassAttendanceSessionURLController struct {
	DB        *gorm.DB
	validator *validator.Validate
}

func NewClassAttendanceSessionURLController(db *gorm.DB) *ClassAttendanceSessionURLController {
	return &ClassAttendanceSessionURLController{
		DB:        db,
		validator: validator.New(),
	}
}

/* =========================================================
 * CREATE (JSON or MULTIPART)
 * POST /api/a/class-attendance-session-urls
 * ========================================================= */
// CREATE (JSON or MULTIPART)
// CREATE (JSON or MULTIPART)
// CREATE (JSON or MULTIPART)
func (ctl *ClassAttendanceSessionURLController) Create(c *fiber.Ctx) error {
	const logp = "[CASURL:create]"
	const (
		fSessionID = "class_attendance_session_url_session_id"
		fLabel     = "class_attendance_session_url_label"
		fHref      = "class_attendance_session_url_href" // bisa TEXT (URL) atau FILE (fallback)
		fFile      = "file"                               // FILE utama
	)

	// ===== Auth (teacher)
	masjidID, err := helper.GetTeacherMasjidIDFromToken(c)
	if err != nil {
		log.Printf("%s auth failed: %v", logp, err)
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)
	log.Printf("%s start masjid_id=%s content_type=%q multipart=%v path=%s", logp, masjidID, ct, isMultipart, c.Path())

	// Util: cek sesi milik masjid
	checkSessionTenant := func(sessID uuid.UUID) (bool, error) {
		var ok bool
		err := ctl.DB.
			Raw(`SELECT EXISTS(
					SELECT 1 FROM class_attendance_sessions
					WHERE class_attendance_sessions_id = ?
					  AND class_attendance_sessions_masjid_id = ?
					  AND class_attendance_sessions_deleted_at IS NULL
				)`, sessID, masjidID).
			Scan(&ok).Error
		return ok, err
	}

	// Util: ambil label (opsional)
	getLabel := func() *string {
		if v := strings.TrimSpace(c.FormValue(fLabel)); v != "" {
			return &v
		}
		return nil
	}

	// =========================
	// MULTIPART MODE
	// =========================
	if isMultipart {
		// --- session id
		sessIDStr := strings.TrimSpace(c.FormValue(fSessionID))
		if sessIDStr == "" {
			log.Printf("%s missing %s in multipart form", logp, fSessionID)
			return helper.JsonError(c, fiber.StatusBadRequest, fSessionID+" wajib diisi")
		}
		sessID, perr := uuid.Parse(sessIDStr)
		if perr != nil {
			log.Printf("%s invalid %s=%q: %v", logp, fSessionID, sessIDStr, perr)
			return helper.JsonError(c, fiber.StatusBadRequest, fSessionID+" tidak valid")
		}

		// --- tenant guard
		if ok, e := checkSessionTenant(sessID); e != nil {
			log.Printf("%s tenant check failed session_id=%s: %v", logp, sessID, e)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa sesi")
		} else if !ok {
			log.Printf("%s session not found or cross-tenant: session_id=%s masjid_id=%s", logp, sessID, masjidID)
			return helper.JsonError(c, fiber.StatusNotFound, "Sesi tidak ditemukan / berbeda masjid")
		}

		// --- ambil file:
		// 1) coba field utama "file"
		// 2) fallback: kalau ada FILE di "class_attendance_session_url_href", pakai itu
		// --- ambil file: 1) "file", 2) fallback: file di field href
		fh, ferr := c.FormFile(fFile)
		if ferr != nil {
			if alt, altErr := c.FormFile(fHref); altErr == nil {
				fh = alt
				ferr = nil
			}
		}

		var href string
		if ferr == nil {
			// fh pasti non-nil di sini; cukup cek size saja kalau perlu
			if fh.Size <= 0 {
				return helper.JsonError(c, fiber.StatusBadRequest, "File kosong")
			}

			ctFile := ""
			if fh.Header != nil {
				ctFile = fh.Header.Get("Content-Type")
			}
			if !strings.HasPrefix(strings.ToLower(ctFile), "image/") {
				return helper.JsonError(c, fiber.StatusUnsupportedMediaType,
					"File harus berupa gambar. Untuk dokumen, kirimkan tautan via "+fHref)
			}

			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, "OSS init gagal")
			}
			slot := fmt.Sprintf("class-attendance-session-urls/%s", sessID.String())
			newURL, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, masjidID, slot, fh)
			if upErr != nil {
				if fe, ok := upErr.(*fiber.Error); ok {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload file")
			}
			href = newURL
		} else {
			// tanpa file → pakai TEXT URL di field href
			h := strings.TrimSpace(c.FormValue(fHref))
			if h == "" {
				return helper.JsonError(c, fiber.StatusBadRequest, "Wajib mengirim file atau "+fHref)
			}
			href = h
		}


		// --- persist
		mdl := sessionmodel.ClassAttendanceSessionURLModel{
			ClassAttendanceSessionURLMasjidID:  masjidID,
			ClassAttendanceSessionURLSessionID: sessID,
			ClassAttendanceSessionURLLabel:     getLabel(),
			ClassAttendanceSessionURLHref:      href,
		}
		if err := ctl.DB.WithContext(c.Context()).Create(&mdl).Error; err != nil {
			low := strings.ToLower(err.Error())
			if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
				log.Printf("%s duplicate href for session: session_id=%s href=%s", logp, sessID, href)
				return helper.JsonError(c, fiber.StatusConflict, "URL sudah ada untuk sesi ini")
			}
			log.Printf("%s DB create error: %v", logp, err)
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan URL sesi")
		}
		log.Printf("%s created OK: id=%s session_id=%s href=%s", logp, mdl.ClassAttendanceSessionURLID, sessID, href)
		return helper.JsonCreated(c, "Berhasil membuat URL sesi", sessiondto.NewClassAttendanceSessionURLResponse(mdl))
	}

	// =========================
	// JSON MODE (tanpa file)
	// =========================
	var req sessiondto.CreateClassAttendanceSessionURLRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("%s json parse error: %v", logp, err)
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(req); err != nil {
		log.Printf("%s json validation error: %v", logp, err)
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// tenant guard
	if ok, e := checkSessionTenant(req.ClassAttendanceSessionURLSessionID); e != nil {
		log.Printf("%s json tenant check error: %v", logp, e)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa sesi")
	} else if !ok {
		log.Printf("%s json session not found / cross-tenant: session_id=%s masjid_id=%s",
			logp, req.ClassAttendanceSessionURLSessionID, masjidID)
		return helper.JsonError(c, fiber.StatusNotFound, "Sesi tidak ditemukan / berbeda masjid")
	}

	mdl := req.ToModel(masjidID)
	if err := ctl.DB.WithContext(c.Context()).Create(&mdl).Error; err != nil {
		low := strings.ToLower(err.Error())
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
			log.Printf("%s json duplicate href: session_id=%s href=%s", logp, req.ClassAttendanceSessionURLSessionID, req.ClassAttendanceSessionURLHref)
			return helper.JsonError(c, fiber.StatusConflict, "URL sudah ada untuk sesi ini")
		}
		log.Printf("%s json DB create error: %v", logp, err)
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan URL sesi")
	}
	log.Printf("%s json created OK: id=%s session_id=%s href=%s",
		logp, mdl.ClassAttendanceSessionURLID, mdl.ClassAttendanceSessionURLSessionID, mdl.ClassAttendanceSessionURLHref)

	return helper.JsonCreated(c, "Berhasil membuat URL sesi", sessiondto.NewClassAttendanceSessionURLResponse(mdl))
}


/* =========================================================
 * UPDATE (JSON or MULTIPART, partial + optional file rotate)
 * PATCH /api/a/class-attendance-session-urls/:id
 * ========================================================= */
/* =========================================================
 * UPDATE (JSON or MULTIPART, partial + optional file rotate)
 * PATCH /api/a/class-attendance-session-urls/:id
 * ========================================================= */
func (ctl *ClassAttendanceSessionURLController) Update(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl sessionmodel.ClassAttendanceSessionURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_attendance_session_url_id = ? AND class_attendance_session_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	if isMultipart {
		// Label (opsional)
		if v := strings.TrimSpace(c.FormValue("class_attendance_session_url_label")); v != "" {
			mdl.ClassAttendanceSessionURLLabel = &v
		}

		// File baru?
		if fh, ferr := c.FormFile("file"); ferr == nil && fh != nil && fh.Size > 0 {
			if fh.Size > 5*1024*1024 {
			 return helper.JsonError(c, fiber.StatusRequestEntityTooLarge, "Ukuran gambar maksimal 5MB")
			}

			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return helper.JsonError(c, fiber.StatusBadGateway, "OSS init gagal")
			}

			// pakai helper upload + convert → WebP, slot khusus per session_id
			slot := fmt.Sprintf("class-attendance-session-urls/%s", mdl.ClassAttendanceSessionURLSessionID.String())
			newURL, upErr := helperOSS.UploadImageToOSS(c.Context(), svc, masjidID, slot, fh)
			if upErr != nil {
				if fe, ok := upErr.(*fiber.Error); ok {
					return helper.JsonError(c, fe.Code, fe.Message)
				}
				return helper.JsonError(c, fiber.StatusBadGateway, "Gagal upload file")
			}

			// Move old to spam (best-effort) + jadwalkan reaper
			if old := strings.TrimSpace(mdl.ClassAttendanceSessionURLHref); old != "" {
				if spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(old, 15*time.Second); mvErr == nil {
					mdl.ClassAttendanceSessionURLTrashURL = &spamURL
				} else {
					mdl.ClassAttendanceSessionURLTrashURL = &old
				}
				due := time.Now().Add(7 * 24 * time.Hour)
				mdl.ClassAttendanceSessionURLDeletePendingUntil = &due
			}

			mdl.ClassAttendanceSessionURLHref = newURL
		} else {
			// Tanpa file → bisa update href manual
			if h := strings.TrimSpace(c.FormValue("class_attendance_session_url_href")); h != "" {
				mdl.ClassAttendanceSessionURLHref = h
			}

			// Optional: trash_url
			form, _ := c.MultipartForm()
			if form != nil {
				if vals, ok := form.Value["class_attendance_session_url_trash_url"]; ok {
					if len(vals) == 0 || strings.TrimSpace(vals[0]) == "" {
						mdl.ClassAttendanceSessionURLTrashURL = nil
					} else {
						tr := strings.TrimSpace(vals[0])
						mdl.ClassAttendanceSessionURLTrashURL = &tr
					}
				}
			}

			// Optional: delete_pending_until (RFC3339)
			if d := strings.TrimSpace(c.FormValue("class_attendance_session_url_delete_pending_until")); d != "" {
				if t, e := time.Parse(time.RFC3339, d); e == nil {
					mdl.ClassAttendanceSessionURLDeletePendingUntil = &t
				}
			}
		}
	} else {
		// ===== JSON mode =====
		var req sessiondto.UpdateClassAttendanceSessionURLRequest
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
		if err := ctl.validator.Struct(req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		if req.ClassAttendanceSessionURLLabel != nil {
			v := strings.TrimSpace(*req.ClassAttendanceSessionURLLabel)
			if v == "" {
				mdl.ClassAttendanceSessionURLLabel = nil
			} else {
				mdl.ClassAttendanceSessionURLLabel = &v
			}
		}
		if req.ClassAttendanceSessionURLHref != nil {
			mdl.ClassAttendanceSessionURLHref = strings.TrimSpace(*req.ClassAttendanceSessionURLHref)
		}
		if req.ClassAttendanceSessionURLTrashURL != nil {
			tr := strings.TrimSpace(*req.ClassAttendanceSessionURLTrashURL)
			if tr == "" {
				mdl.ClassAttendanceSessionURLTrashURL = nil
			} else {
				mdl.ClassAttendanceSessionURLTrashURL = &tr
			}
		}
		if req.ClassAttendanceSessionURLDeletePendingUntil != nil {
			mdl.ClassAttendanceSessionURLDeletePendingUntil = req.ClassAttendanceSessionURLDeletePendingUntil
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return helper.JsonError(c, fiber.StatusConflict, "URL sudah ada untuk sesi ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Berhasil memperbarui", sessiondto.NewClassAttendanceSessionURLResponse(mdl))
}


/* =========================================================
 * GET BY ID
 * GET /api/a/class-attendance-session-urls/:id
 * ========================================================= */
func (ctl *ClassAttendanceSessionURLController) GetByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl sessionmodel.ClassAttendanceSessionURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_attendance_session_url_id = ? AND class_attendance_session_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "OK", sessiondto.NewClassAttendanceSessionURLResponse(mdl))
}

/* =========================================================
 * FILTER / LIST
 * GET /api/a/class-attendance-session-urls/filter?session_id=&search=&only_alive=&page=&limit=&sort=
 * ========================================================= */
func (ctl *ClassAttendanceSessionURLController) Filter(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var q sessiondto.FilterClassAttendanceSessionURLRequest
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	if err := ctl.validator.Struct(q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	page := 1
	limit := 20
	if q.Page != nil && *q.Page > 0 {
		page = *q.Page
	}
	if q.Limit != nil && *q.Limit > 0 {
		limit = *q.Limit
	}
	offset := (page - 1) * limit

	dbq := ctl.DB.WithContext(c.Context()).
		Model(&sessionmodel.ClassAttendanceSessionURLModel{}).
		Where("class_attendance_session_url_masjid_id = ?", masjidID)

	onlyAlive := true
	if q.OnlyAlive != nil {
		onlyAlive = *q.OnlyAlive
	}
	if onlyAlive {
		dbq = dbq.Where("class_attendance_session_url_deleted_at IS NULL")
	}

	if q.SessionID != nil {
		dbq = dbq.Where("class_attendance_session_url_session_id = ?", *q.SessionID)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.TrimSpace(*q.Search) + "%"
		dbq = dbq.Where("(class_attendance_session_url_label ILIKE ? OR class_attendance_session_url_href ILIKE ?)", s, s)
	}

	order := "class_attendance_session_url_created_at DESC"
	if q.Sort != nil {
		switch *q.Sort {
		case "created_at_asc":
			order = "class_attendance_session_url_created_at ASC"
		case "label_asc":
			order = "class_attendance_session_url_label ASC NULLS LAST, class_attendance_session_url_created_at DESC"
		case "label_desc":
			order = "class_attendance_session_url_label DESC NULLS LAST, class_attendance_session_url_created_at DESC"
		case "created_at_desc":
			order = "class_attendance_session_url_created_at DESC"
		}
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	var rows []sessionmodel.ClassAttendanceSessionURLModel
	if err := dbq.Order(order).Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resps := make([]sessiondto.ClassAttendanceSessionURLResponse, 0, len(rows))
	for _, m := range rows {
		resps = append(resps, sessiondto.NewClassAttendanceSessionURLResponse(m))
	}

	pagination := fiber.Map{
		"page":  page,
		"limit": limit,
		"total": total,
		"pages": int(math.Ceil(float64(total) / float64(limit))),
	}
	return helper.JsonList(c, resps, pagination)
}


/* =========================================================
 * DELETE (soft) + move file to spam/
 * DELETE /api/a/class-attendance-session-urls/:id
 * ========================================================= */
func (ctl *ClassAttendanceSessionURLController) Delete(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl sessionmodel.ClassAttendanceSessionURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_attendance_session_url_id = ? AND class_attendance_session_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	var (
		spamURL string
		duePtr  *time.Time
	)

	// Pindahkan file aktif ke spam/ (best-effort)
	if h := strings.TrimSpace(mdl.ClassAttendanceSessionURLHref); h != "" {
		if s, mvErr := helperOSS.MoveToSpamByPublicURLENV(h, 15*time.Second); mvErr == nil {
			spamURL = s
		} else {
			// kalau gagal dipindah, tetap catat href lama supaya reaper bisa follow-up
			spamURL = h
		}
		due := time.Now().Add(7 * 24 * time.Hour)
		duePtr = &due
	}

	// Transaksi: simpan status trash (jika ada) lalu soft-delete
	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if spamURL != "" {
			mdl.ClassAttendanceSessionURLTrashURL = &spamURL
			mdl.ClassAttendanceSessionURLDeletePendingUntil = duePtr
			if err := tx.Save(&mdl).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&mdl).Error
	}); err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	data := fiber.Map{
		"class_attendance_session_url_id": mdl.ClassAttendanceSessionURLID,
		"moved_to_spam_url":               spamURL,               // bisa kosong kalau tidak ada href
		"delete_pending_until":            duePtr,                // nil jika tidak ada href
		"deleted_at":                      time.Now(),
	}
	return helper.JsonDeleted(c, "Berhasil menghapus", data)
}
