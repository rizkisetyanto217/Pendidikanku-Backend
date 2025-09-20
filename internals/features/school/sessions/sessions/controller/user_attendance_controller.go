// internals/features/school/attendance_assesment/user_result/user_attendance/controller/user_attendance_controller.go
package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"time"

	attDTO "masjidku_backend/internals/features/school/sessions/sessions/dto"
	attModel "masjidku_backend/internals/features/school/sessions/sessions/model"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserAttendanceController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewUserAttendanceController(db *gorm.DB) *UserAttendanceController {
	return &UserAttendanceController{
		DB:        db,
		Validator: validator.New(),
	}
}

// letakkan di file controller yang sama
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	// umumnya driver menuliskan salah satu dari ini
	return strings.Contains(s, "duplicate key") ||
		strings.Contains(s, "violates unique constraint") ||
		strings.Contains(s, "unique constraint") ||
		strings.Contains(s, "sqlstate 23505")
}

const dateLayout = "2006-01-02"

// ===============================
// Helpers
// ===============================

// Pastikan session milik masjid ini (tenant-safe)
func (ctl *UserAttendanceController) ensureSessionBelongsToMasjid(c *fiber.Ctx, sessionID, masjidID uuid.UUID) error {
	var count int64
	if err := ctl.DB.WithContext(c.Context()).
		Table("class_attendance_sessions").
		Where("class_attendance_sessions_id = ? AND class_attendance_sessions_masjid_id = ? AND class_attendance_sessions_deleted_at IS NULL",
			sessionID, masjidID).
		Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if count == 0 {
		return fiber.NewError(fiber.StatusForbidden, "Session tidak ditemukan/diizinkan untuk masjid ini")
	}
	return nil
}

// Build list query with filters/sort (tenant-aware)
func (ctl *UserAttendanceController) buildListQuery(c *fiber.Ctx, q attDTO.ListUserAttendanceQuery, masjidID uuid.UUID) (*gorm.DB, error) {
	tx := ctl.DB.WithContext(c.Context()).Model(&attModel.UserAttendanceModel{}).
		Where("user_attendance_masjid_id = ?", masjidID)

	if q.SessionID != nil {
		tx = tx.Where("user_attendance_session_id = ?", *q.SessionID)
	}
	if q.StudentID != nil {
		tx = tx.Where("user_attendance_masjid_student_id = ?", *q.StudentID)
	}
	if q.TypeID != nil {
		tx = tx.Where("user_attendance_type_id = ?", *q.TypeID)
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		s := strings.ToLower(strings.TrimSpace(*q.Status))
		switch s {
		case "present", "absent", "excused", "late":
			tx = tx.Where("user_attendance_status = ?", s)
		default:
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "status tidak valid (present/absent/excused/late)")
		}
	}
	// filter nilai
	if q.ScoreFrom != nil {
		tx = tx.Where("user_attendance_score IS NOT NULL AND user_attendance_score >= ?", *q.ScoreFrom)
	}
	if q.ScoreTo != nil {
		tx = tx.Where("user_attendance_score IS NOT NULL AND user_attendance_score <= ?", *q.ScoreTo)
	}
	if q.IsPassed != nil {
		tx = tx.Where("user_attendance_is_passed = ?", *q.IsPassed)
	}

	if q.CreatedFrom != nil && strings.TrimSpace(*q.CreatedFrom) != "" {
		t, err := time.Parse(dateLayout, strings.TrimSpace(*q.CreatedFrom))
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "created_from invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_attendance_created_at >= ?", t)
	}
	if q.CreatedTo != nil && strings.TrimSpace(*q.CreatedTo) != "" {
		t, err := time.Parse(dateLayout, strings.TrimSpace(*q.CreatedTo))
		if err != nil {
			return nil, helper.JsonError(c, fiber.StatusBadRequest, "created_to invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_attendance_created_at < ?", t.Add(24*time.Hour))
	}

	order := "user_attendance_created_at DESC"
	if q.Sort != nil {
		switch strings.ToLower(strings.TrimSpace(*q.Sort)) {
		case "created_at_asc":
			order = "user_attendance_created_at ASC"
		case "created_at_desc":
			order = "user_attendance_created_at DESC"
		}
	}
	tx = tx.Order(order)
	return tx, nil
}

// ===============================
// Handlers
// ===============================

/*
=========================================================
POST /user-attendance (versi with URLs)
Mendukung:
  - JSON:
    {
    "attendance": { ...CreateUserAttendanceRequest... },
    "urls": [ {kind,label,href,object_key,order,is_primary,...}, ... ]
    }

- multipart/form-data:
  - attendance_json: JSON CreateUserAttendanceRequest (wajib)
  - urls_json: JSON array UAUUpsert (opsional)
  - bracket/array: urls[0][kind], urls[0][label], urls[0][href], urls[0][object_key], urls[0][order], urls[0][is_primary]
  - file uploads: otomatis upload ke OSS → dibuat baris URL

=========================================================
*/
func (ctl *UserAttendanceController) CreateWithURLs(c *fiber.Ctx) error {
	// ── Masjid context via helpers (slug/ID di path/header/query/host) ──
	c.Locals("DB", ctl.DB)
	var masjidID uuid.UUID

	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		// Wajib DKM/Admin untuk context eksplisit
		if id, er := helperAuth.EnsureMasjidAccessDKM(c, mc); er == nil {
			masjidID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		// Fallback: prefer TEACHER → DKM/Admin
		if id, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			masjidID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak ditemukan")
		}
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ----- Parse payload -----
	var attReq attDTO.CreateUserAttendanceRequest
	var urlUpserts []attDTO.UAUUpsert

	if strings.HasPrefix(ct, "multipart/form-data") {
		// attendance_json wajib
		aj := strings.TrimSpace(c.FormValue("attendance_json"))
		if aj == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "attendance_json wajib diisi (JSON CreateUserAttendanceRequest)")
		}
		if err := json.Unmarshal([]byte(aj), &attReq); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "attendance_json tidak valid: "+err.Error())
		}

		// urls_json opsional
		if uj := strings.TrimSpace(c.FormValue("urls_json")); uj != "" {
			_ = json.Unmarshal([]byte(uj), &urlUpserts) // kalau gagal, abaikan & tetap pakai bracket/files
		}

		// Bracket/array style → helperOSS.ParseURLUpsertsFromMultipart
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			parsed := helperOSS.ParseURLUpsertsFromMultipart(form, &helperOSS.URLParseOptions{
				BracketPrefix: "urls",
				DefaultKind:   "attachment",
			})
			for _, p := range parsed {
				up := attDTO.UAUUpsert{
					Kind:      p.Kind,
					Label:     p.Label,
					Href:      p.Href,
					ObjectKey: p.ObjectKey,
				}
				if p.Order != 0 {
					o := int32(p.Order)
					up.Order = &o
				}
				if p.IsPrimary {
					ip := true
					up.IsPrimary = &ip
				}
				up.Normalize()
				urlUpserts = append(urlUpserts, up)
			}

		}
	} else {
		// JSON murni
		var body struct {
			Attendance attDTO.CreateUserAttendanceRequest `json:"attendance"`
			URLs       []attDTO.UAUUpsert                 `json:"urls"`
		}
		bodyRaw := bytes.TrimSpace(c.Body())
		if len(bodyRaw) == 0 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload kosong")
		}
		if err := json.Unmarshal(bodyRaw, &body); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
		attReq = body.Attendance
		urlUpserts = body.URLs
	}

	// ----- Validasi attendance -----
	if err := ctl.Validator.Struct(&attReq); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ----- Tenant guard: pastikan session milik masjid ini -----
	if err := ctl.ensureSessionBelongsToMasjid(c, attReq.UserAttendanceSessionID, masjidID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ----- Normalisasi URL upserts -----
	for i := range urlUpserts {
		urlUpserts[i].Normalize()
	}

	// =========================
	// Transaksi
	// =========================
	var createdAtt *attModel.UserAttendanceModel

	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) buat attendance
		m := attReq.ToModel(masjidID)
		if err := tx.Create(m).Error; err != nil {
			if isDuplicateKey(err) {
				return fiber.NewError(fiber.StatusConflict, "Kehadiran sudah tercatat (duplikat)")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		createdAtt = m

		// 2) build URL models dari upserts JSON/bracket
		var urlModels []attModel.UserAttendanceURL
		for _, u := range urlUpserts {
			row := attModel.UserAttendanceURL{
				UserAttendanceURLMasjidID:   masjidID,
				UserAttendanceURLAttendance: m.UserAttendanceID, // sesuaikan nama field PK attendance kamu
				UserAttendanceTypeID:        nil,                // isi jika kamu punya type lookup di upsert
				UserAttendanceURLKind:       u.Kind,
				UserAttendanceURLHref:       u.Href,
				UserAttendanceURLObjectKey:  u.ObjectKey,
				UserAttendanceURLLabel:      u.Label,
				UserAttendanceURLIsPrimary:  false,
				UserAttendanceURLOrder:      0,
				// uploader (opsional):
				UserAttendanceURLUploaderTeacherID: u.UploaderTeacherID,
				UserAttendanceURLUploaderStudentID: u.UploaderStudentID,
			}
			if u.IsPrimary != nil {
				row.UserAttendanceURLIsPrimary = *u.IsPrimary
			}
			if u.Order != nil {
				row.UserAttendanceURLOrder = *u.Order
			}
			if strings.TrimSpace(row.UserAttendanceURLKind) == "" {
				row.UserAttendanceURLKind = "attachment"
			}
			urlModels = append(urlModels, row)
		}

		// 3) dari multipart files → upload ke OSS dan isi baris URL
		if strings.HasPrefix(ct, "multipart/form-data") {
			if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
				fhs, _ := helperOSS.CollectUploadFiles(form, nil)
				if len(fhs) > 0 {
					oss, oerr := helperOSS.NewOSSServiceFromEnv("")
					if oerr != nil {
						return fiber.NewError(fiber.StatusBadGateway, "OSS tidak siap")
					}
					ctx := context.Background()
					for _, fh := range fhs {
						publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, masjidID, "user_attendance", fh)
						if uerr != nil {
							// helper biasanya sudah balikin fiber.Error friendly
							return uerr
						}
						var key *string
						if k, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
							key = &k
						}
						urlModels = append(urlModels, attModel.UserAttendanceURL{
							UserAttendanceURLMasjidID:   masjidID,
							UserAttendanceURLAttendance: createdAtt.UserAttendanceID,
							UserAttendanceURLKind:       "attachment",
							UserAttendanceURLHref:       &publicURL,
							UserAttendanceURLObjectKey:  key,
							UserAttendanceURLOrder:      int32(len(urlModels) + 1),
						})
					}
				}
			}
		}

		// 4) Simpan URL models (jika ada)
		if len(urlModels) > 0 {
			if err := tx.Create(&urlModels).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan lampiran")
			}
			// Enforce: satu primary per (attendance, kind) untuk baris live
			for _, it := range urlModels {
				if it.UserAttendanceURLIsPrimary {
					if err := tx.Model(&attModel.UserAttendanceURL{}).
						Where(`
							user_attendance_url_masjid_id = ?
							AND user_attendance_url_attendance_id = ?
							AND user_attendance_url_kind = ?
							AND user_attendance_url_id <> ?
							AND user_attendance_url_deleted_at IS NULL
						`,
							masjidID, createdAtt.UserAttendanceID, it.UserAttendanceURLKind, it.UserAttendanceURLID,
						).
						Update("user_attendance_url_is_primary", false).Error; err != nil {
						return fiber.NewError(fiber.StatusInternalServerError, "Gagal set primary lampiran")
					}
				}
			}
		}

		return nil
	}); err != nil {
		// error dari Transaction sudah berupa fiber.Error friendly
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// ----- Build response -----
	resp := attDTO.NewUserAttendanceResponse(createdAtt)

	// Ambil URLs (live) supaya FE langsung dapat
	var rows []attModel.UserAttendanceURL
	_ = ctl.DB.
		Where("user_attendance_url_attendance_id = ? AND user_attendance_url_deleted_at IS NULL", createdAtt.UserAttendanceID).
		Order("user_attendance_url_is_primary DESC, user_attendance_url_order ASC, user_attendance_url_created_at ASC").
		Find(&rows)

	// Sisipkan ke response bila DTO attendance kamu punya slot lampiran.
	// Contoh: tambahkan field generic "urls" ke map response:
	out := fiber.Map{
		"attendance": resp,
		"urls":       make([]fiber.Map, 0, len(rows)),
	}
	for i := range rows {
		item := fiber.Map{
			"id":            rows[i].UserAttendanceURLID,
			"masjid_id":     rows[i].UserAttendanceURLMasjidID,
			"attendance_id": rows[i].UserAttendanceURLAttendance,
			"type_id":       rows[i].UserAttendanceTypeID,
			"kind":          rows[i].UserAttendanceURLKind,
			"href":          rows[i].UserAttendanceURLHref,
			"object_key":    rows[i].UserAttendanceURLObjectKey,
			"label":         rows[i].UserAttendanceURLLabel,
			"order":         rows[i].UserAttendanceURLOrder,
			"is_primary":    rows[i].UserAttendanceURLIsPrimary,
			"created_at":    rows[i].UserAttendanceURLCreatedAt,
			"updated_at":    rows[i].UserAttendanceURLUpdatedAt,
		}
		out["urls"] = append(out["urls"].([]fiber.Map), item)
	}

	c.Set("Location", "/user-attendance/"+createdAtt.UserAttendanceID.String())
	return helper.JsonCreated(c, "Kehadiran & lampiran berhasil dibuat", out)
}

// PATCH /user-attendance/:id
// PATCH /user-attendance/:id/urls
// Body (JSON):
//
//	{
//	  "urls": [
//	    { "id":"<optional>", "kind":"image|video|attachment|link|audio", "label":"...", "href":"...", "object_key":"...", "order":1, "is_primary":true },
//	    { "id":"...", "replace_file": true } // jika multipart, file akan mengganti object; object_key_old diisi
//	  ]
//	}
//
// Multipart form (opsional untuk replace/upload baru):
// - urls_json: payload JSON di atas
// - files[]: file yang dipasangkan berurutan dengan item urls yg punya replace_file=true ATAU item yg tanpa id (insert)
// Update user_attendance URLs (insert/update + optional file replace)
func (ctl *UserAttendanceController) Update(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// ── Resolve masjid (DKM/Admin prefer; fallback Teacher) ──
	var masjidID uuid.UUID
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureMasjidAccessDKM(c, mc); er == nil {
			masjidID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			masjidID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak ditemukan")
		}
	}

	attID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "attendance id tidak valid")
	}

	// ── Pastikan attendance milik masjid ini ──
	var attCount int64
	if err := ctl.DB.WithContext(c.Context()).
		Table("user_attendance").
		Where("user_attendance_id = ? AND user_attendance_masjid_id = ? AND user_attendance_deleted_at IS NULL", attID, masjidID).
		Count(&attCount).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	if attCount == 0 {
		return helper.JsonError(c, fiber.StatusForbidden, "Attendance tidak ditemukan/diizinkan")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ── Parse payload upserts ──
	type upsert struct {
		ID                *uuid.UUID `json:"id"`
		Kind              string     `json:"kind"`
		Label             *string    `json:"label"`
		Href              *string    `json:"href"`
		ObjectKey         *string    `json:"object_key"`
		Order             *int32     `json:"order"`
		IsPrimary         *bool      `json:"is_primary"`
		ReplaceFile       bool       `json:"replace_file"` // multipart helper
		UploaderTeacherID *uuid.UUID `json:"uploader_teacher_id"`
		UploaderStudentID *uuid.UUID `json:"uploader_student_id"`
	}
	var ups []upsert

	if strings.HasPrefix(ct, "multipart/form-data") {
		payload := strings.TrimSpace(c.FormValue("urls_json"))
		if payload == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "urls_json wajib diisi pada multipart")
		}
		if err := json.Unmarshal([]byte(payload), &ups); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "urls_json tidak valid: "+err.Error())
		}
	} else {
		var body struct {
			URLs []upsert `json:"urls"`
		}
		raw := bytes.TrimSpace(c.Body())
		if len(raw) == 0 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload kosong")
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
		ups = body.URLs
	}

	// Normalisasi ringan
	for i := range ups {
		ups[i].Kind = strings.TrimSpace(strings.ToLower(ups[i].Kind))
		if ups[i].Kind == "" {
			ups[i].Kind = "attachment"
		}
		if ups[i].Label != nil {
			l := strings.TrimSpace(*ups[i].Label)
			ups[i].Label = &l
		}
		if ups[i].Href != nil {
			h := strings.TrimSpace(*ups[i].Href)
			if h == "" {
				ups[i].Href = nil
			} else {
				ups[i].Href = &h
			}
		}
		if ups[i].ObjectKey != nil {
			k := strings.TrimSpace(*ups[i].ObjectKey)
			if k == "" {
				ups[i].ObjectKey = nil
			} else {
				ups[i].ObjectKey = &k
			}
		}
	}

	// ── Transaksi ──
	if err := ctl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// Ambil existing URLs utk attendance ini
		var existing []attModel.UserAttendanceURL
		if err := tx.Where(`
			user_attendance_url_attendance_id = ?
			AND user_attendance_url_masjid_id = ?
			AND user_attendance_url_deleted_at IS NULL
		`, attID, masjidID).Find(&existing).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil lampiran")
		}
		byID := map[uuid.UUID]*attModel.UserAttendanceURL{}
		for i := range existing {
			byID[existing[i].UserAttendanceURLID] = &existing[i]
		}

		// OSS service + files (multipart only)
		hasMultipart := strings.HasPrefix(ct, "multipart/form-data")
		var svc *helperOSS.OSSService
		var fhs []*multipart.FileHeader
		if hasMultipart {
			if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
				if tmp, _ := helperOSS.CollectUploadFiles(form, nil); len(tmp) > 0 {
					fhs = tmp
					if s, oerr := helperOSS.NewOSSServiceFromEnv(""); oerr == nil {
						svc = s
					} else {
						return fiber.NewError(fiber.StatusBadGateway, "OSS tidak siap")
					}
				}
			}
		}
		nextFile := func() (*multipart.FileHeader, bool) {
			if len(fhs) == 0 {
				return nil, false
			}
			f := fhs[0]
			fhs = fhs[1:]
			return f, true
		}

		uploadAndFill := func(patch map[string]any, oldKey *string) error {
			if svc == nil {
				return fiber.NewError(fiber.StatusBadRequest, "OSS service tidak siap")
			}
			fh, ok := nextFile()
			if !ok {
				return fiber.NewError(fiber.StatusBadRequest, "Jumlah file tidak cukup untuk replace")
			}
			publicURL, uerr := helperOSS.UploadAnyToOSS(c.Context(), svc, masjidID, "user_attendance", fh)
			if uerr != nil {
				return uerr
			}
			patch["user_attendance_url_href"] = publicURL
			if key, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
				patch["user_attendance_url_object_key"] = key
			}
			if oldKey != nil && strings.TrimSpace(*oldKey) != "" {
				patch["user_attendance_url_object_key_old"] = *oldKey
			}
			return nil
		}

		var touched []attModel.UserAttendanceURL

		for _, u := range ups {
			if u.ID == nil {
				// INSERT baru
				row := attModel.UserAttendanceURL{
					UserAttendanceURLMasjidID:          masjidID,
					UserAttendanceURLAttendance:        attID,
					UserAttendanceURLKind:              u.Kind,
					UserAttendanceURLLabel:             u.Label,
					UserAttendanceURLHref:              u.Href,
					UserAttendanceURLObjectKey:         u.ObjectKey,
					UserAttendanceURLIsPrimary:         false,
					UserAttendanceURLOrder:             0,
					UserAttendanceURLUploaderTeacherID: u.UploaderTeacherID,
					UserAttendanceURLUploaderStudentID: u.UploaderStudentID,
				}
				if u.IsPrimary != nil {
					row.UserAttendanceURLIsPrimary = *u.IsPrimary
				}
				if u.Order != nil {
					row.UserAttendanceURLOrder = *u.Order
				}
				if u.ReplaceFile && hasMultipart {
					patch := map[string]any{}
					if err := uploadAndFill(patch, nil); err != nil {
						return err
					}
					if v, ok := patch["user_attendance_url_href"].(string); ok {
						row.UserAttendanceURLHref = &v
					}
					if v, ok := patch["user_attendance_url_object_key"].(string); ok {
						row.UserAttendanceURLObjectKey = &v
					}
				}
				if err := tx.Create(&row).Error; err != nil {
					if isDuplicateKey(err) {
						return fiber.NewError(fiber.StatusConflict, "URL duplikat untuk attendance ini")
					}
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal menambah URL")
				}
				touched = append(touched, row)
				continue
			}

			// UPDATE
			ex, ok := byID[*u.ID]
			if !ok {
				return fiber.NewError(fiber.StatusNotFound, "URL tidak ditemukan untuk attendance ini")
			}
			patch := map[string]any{}
			if u.Kind != "" && u.Kind != ex.UserAttendanceURLKind {
				patch["user_attendance_url_kind"] = u.Kind
			}
			if u.Label != nil {
				patch["user_attendance_url_label"] = u.Label
			}
			if u.IsPrimary != nil {
				patch["user_attendance_url_is_primary"] = *u.IsPrimary
			}
			if u.Order != nil {
				patch["user_attendance_url_order"] = *u.Order
			}
			if u.ReplaceFile && hasMultipart {
				if err := uploadAndFill(patch, ex.UserAttendanceURLObjectKey); err != nil {
					return err
				}
			} else {
				if u.Href != nil {
					patch["user_attendance_url_href"] = *u.Href
				}
				if u.ObjectKey != nil {
					patch["user_attendance_url_object_key"] = *u.ObjectKey
				}
			}
			if len(patch) > 0 {
				patch["user_attendance_url_updated_at"] = time.Now()
				if err := tx.Model(&attModel.UserAttendanceURL{}).
					Where("user_attendance_url_id = ? AND user_attendance_url_masjid_id = ? AND user_attendance_url_attendance_id = ? AND user_attendance_url_deleted_at IS NULL",
						*u.ID, masjidID, attID).
					Updates(patch).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengubah URL")
				}
			}
			var latest attModel.UserAttendanceURL
			_ = tx.Where("user_attendance_url_id = ?", *u.ID).First(&latest).Error
			touched = append(touched, latest)
		}

		// enforce: satu primary per (attendance, kind)
		for _, it := range touched {
			if it.UserAttendanceURLIsPrimary {
				if err := tx.Model(&attModel.UserAttendanceURL{}).
					Where(`
						user_attendance_url_masjid_id = ?
						AND user_attendance_url_attendance_id = ?
						AND user_attendance_url_kind = ?
						AND user_attendance_url_id <> ?
						AND user_attendance_url_deleted_at IS NULL
					`, masjidID, attID, it.UserAttendanceURLKind, it.UserAttendanceURLID).
					Update("user_attendance_url_is_primary", false).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal set primary unik")
				}
			}
		}

		return nil
	}); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Balikan list terbaru
	var rows []attModel.UserAttendanceURL
	_ = ctl.DB.
		Where("user_attendance_url_attendance_id = ? AND user_attendance_url_masjid_id = ? AND user_attendance_url_deleted_at IS NULL", attID, masjidID).
		Order("user_attendance_url_is_primary DESC, user_attendance_url_order ASC, user_attendance_url_created_at ASC").
		Find(&rows)

	return helper.JsonUpdated(c, "Lampiran berhasil di-update", fiber.Map{
		"attendance_id": attID,
		"urls":          rows,
	})
}

// DELETE /user-attendance/:id  (soft delete)
// DELETE /user-attendance/:id/urls/:url_id
// Soft-delete baris URL + (opsional) pindahkan objek aktif ke folder spam/* di OSS,
// set trash_url dan delete_pending_until (ikut reaper).
func (ctl *UserAttendanceController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctl.DB)

	// Resolve masjid seperti sebelumnya
	var masjidID uuid.UUID
	if mc, err := helperAuth.ResolveMasjidContext(c); err == nil && (mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "") {
		if id, er := helperAuth.EnsureMasjidAccessDKM(c, mc); er == nil {
			masjidID = id
		} else {
			if fe, ok := er.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusForbidden, er.Error())
		}
	} else {
		if id, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
			masjidID = id
		} else {
			return helper.JsonError(c, fiber.StatusForbidden, "Scope masjid tidak ditemukan")
		}
	}

	attID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "attendance id tidak valid")
	}
	urlID, err := uuid.Parse(strings.TrimSpace(c.Params("url_id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "url id tidak valid")
	}

	// Ambil row, pastikan tenant & owner benar
	var row attModel.UserAttendanceURL
	if err := ctl.DB.WithContext(c.Context()).
		Where(`
			user_attendance_url_id = ?
			AND user_attendance_url_attendance_id = ?
			AND user_attendance_url_masjid_id = ?
			AND user_attendance_url_deleted_at IS NULL
		`, urlID, attID, masjidID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "URL tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Opsional: pindahkan object aktif ke spam/ dan set trash_url
	var trashURL *string
	if row.UserAttendanceURLHref != nil && strings.TrimSpace(*row.UserAttendanceURLHref) != "" {
		if moved, mErr := helperOSS.MoveToSpamByPublicURLENV(*row.UserAttendanceURLHref, 0); mErr == nil && strings.TrimSpace(moved) != "" {
			trashURL = &moved
		}
	}

	// Hitung delete_pending_until (ikut RETENTION_DAYS pada reaper; default 30d)
	retentionDays := 30
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("RETENTION_DAYS"))); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			retentionDays = n
		}
	}
	cutoff := time.Now().Add(time.Duration(retentionDays) * 24 * time.Hour)

	// Soft-delete: set deleted_at + (optional) trash fields utk reaper
	if err := ctl.DB.WithContext(c.Context()).
		Model(&attModel.UserAttendanceURL{}).
		Where("user_attendance_url_id = ?", row.UserAttendanceURLID).
		Updates(map[string]any{
			"user_attendance_url_deleted_at":           time.Now(),
			"user_attendance_url_trash_url":            trashURL,
			"user_attendance_url_delete_pending_until": cutoff,
			"user_attendance_url_updated_at":           time.Now(),
		}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus URL")
	}

	return helper.JsonDeleted(c, "Lampiran dihapus (soft-delete)", fiber.Map{
		"attendance_id": attID,
		"url_id":        urlID,
		"trash_url":     trashURL,
		"purge_after":   cutoff,
	})
}
