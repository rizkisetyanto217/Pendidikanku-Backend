// file: internals/features/assessments/submissions/controller/submission_controller.go
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

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	dto "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "madinahsalam_backend/internals/features/school/submissions_assesments/submissions/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	dbtime "madinahsalam_backend/internals/helpers/dbtime"
	helperOSS "madinahsalam_backend/internals/helpers/oss"
)

type SubmissionController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewSubmissionController(db *gorm.DB) *SubmissionController {
	return &SubmissionController{
		DB:        db,
		Validator: validator.New(),
	}
}

func applySort(q *gorm.DB, sort string) *gorm.DB {
	switch strings.TrimSpace(sort) {
	case "created_at":
		return q.Order("submission_created_at ASC")
	case "desc_created_at", "":
		return q.Order("submission_created_at DESC")
	case "submitted_at":
		return q.Order("submission_submitted_at ASC NULLS LAST")
	case "desc_submitted_at":
		return q.Order("submission_submitted_at DESC NULLS LAST")
	case "score":
		return q.Order("submission_score ASC NULLS LAST")
	case "desc_score":
		return q.Order("submission_score DESC NULLS LAST")
	default:
		return q.Order("submission_created_at DESC")
	}
}

/* =========================
   Helpers (local)
========================= */

// Ambil school_id dari token aktif (bukan dari path)
func parseSchoolIDParam(c *fiber.Ctx) (uuid.UUID, error) {
	schoolID, err := helperAuth.GetActiveSchoolID(c)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "School aktif di token tidak ditemukan")
	}
	if schoolID == uuid.Nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "School aktif di token tidak valid")
	}
	return schoolID, nil
}

// Student-only: pastikan user adalah student di school ini
func resolveStudentSchoolFromParam(c *fiber.Ctx) (uuid.UUID, error) {
	schoolID, err := parseSchoolIDParam(c)
	if err != nil {
		return uuid.Nil, err
	}
	if err := helperAuth.EnsureStudentSchool(c, schoolID); err != nil {
		return uuid.Nil, err
	}
	return schoolID, nil
}

// DKM/Teacher/Owner: untuk kelola attachment submission
func resolveTeacherSchoolFromParam(c *fiber.Ctx) (uuid.UUID, error) {
	schoolID, err := parseSchoolIDParam(c)
	if err != nil {
		return uuid.Nil, err
	}
	if err := helperAuth.EnsureDKMOrTeacherSchool(c, schoolID); err != nil && !helperAuth.IsOwner(c) {
		return uuid.Nil, err
	}
	return schoolID, nil
}

/* =========================
   Handlers
========================= */

// POST /submissions  (STUDENT ONLY, school dari token)
func (ctrl *SubmissionController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// ---------- Role & School context (STUDENT via token) ----------
	schoolID, err := resolveStudentSchoolFromParam(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil student_id milik caller pada school ini
	sid, err := helperAuth.GetSchoolStudentIDSmart(c, ctrl.DB, schoolID)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}
	if sid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya siswa terdaftar yang diizinkan membuat submission")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ---------- Parse payload ----------
	var subReq dto.CreateSubmissionRequest

	// Upsert URL lokal (optional)
	type URLUpsert struct {
		SubmissionURLKind      string  `json:"submission_url_kind"`
		SubmissionURLLabel     *string `json:"submission_url_label"`
		SubmissionURL          *string `json:"submission_url_href"`
		SubmissionURLObjectKey *string `json:"submission_url_object_key"`
		SubmissionURLOrder     *int    `json:"submission_url_order"`
		SubmissionURLIsPrimary *bool   `json:"submission_url_is_primary"`
	}
	var urlUpserts []URLUpsert

	if strings.HasPrefix(ct, "multipart/form-data") {
		// submission_json wajib
		raw := strings.TrimSpace(c.FormValue("submission_json"))
		if raw == "" {
			return helper.JsonError(c, fiber.StatusBadRequest, "submission_json wajib diisi (JSON CreateSubmissionRequest)")
		}
		if err := json.Unmarshal([]byte(raw), &subReq); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "submission_json tidak valid: "+err.Error())
		}

		// urls_json opsional
		if uj := strings.TrimSpace(c.FormValue("urls_json")); uj != "" {
			_ = json.Unmarshal([]byte(uj), &urlUpserts)
		}

		// bracket/array style
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			parsed := helperOSS.ParseURLUpsertsFromMultipart(form, &helperOSS.URLParseOptions{
				BracketPrefix: "urls",
				DefaultKind:   "attachment",
			})
			for _, p := range parsed {
				up := URLUpsert{
					SubmissionURLKind:      strings.TrimSpace(strings.ToLower(p.Kind)),
					SubmissionURLLabel:     p.Label,
					SubmissionURL:          p.Href,
					SubmissionURLObjectKey: p.ObjectKey,
				}
				if up.SubmissionURLKind == "" {
					up.SubmissionURLKind = "attachment"
				}
				if p.Order != 0 {
					o := int(p.Order)
					up.SubmissionURLOrder = &o
				}
				if p.IsPrimary {
					ip := true
					up.SubmissionURLIsPrimary = &ip
				}

				// trim
				if up.SubmissionURLLabel != nil {
					l := strings.TrimSpace(*up.SubmissionURLLabel)
					up.SubmissionURLLabel = &l
				}
				if up.SubmissionURL != nil {
					h := strings.TrimSpace(*up.SubmissionURL)
					if h == "" {
						up.SubmissionURL = nil
					} else {
						up.SubmissionURL = &h
					}
				}
				if up.SubmissionURLObjectKey != nil {
					ok := strings.TrimSpace(*up.SubmissionURLObjectKey)
					if ok == "" {
						up.SubmissionURLObjectKey = nil
					} else {
						up.SubmissionURLObjectKey = &ok
					}
				}

				urlUpserts = append(urlUpserts, up)
			}
		}
	} else {
		// JSON murni
		var body struct {
			Submission dto.CreateSubmissionRequest `json:"submission"`
			URLs       []URLUpsert                 `json:"urls"`
		}
		raw := bytes.TrimSpace(c.Body())
		if len(raw) == 0 {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload kosong")
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
		subReq = body.Submission
		urlUpserts = body.URLs
	}

	// ---------- Force tenant & caller identity ----------
	subReq.SubmissionSchoolID = schoolID
	subReq.SubmissionStudentID = sid

	// ---------- Validasi submission ----------
	if err := ctrl.Validator.Struct(&subReq); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil waktu server (DB) sekali untuk request ini
	now, err := dbtime.GetDBTime(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mendapatkan waktu server")
	}

	// ---------- Transaksi ----------
	var created *model.SubmissionModel

	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {

		// =====================================================
		// 1) Hitung next attempt_count (LOCK row terakhir)
		// =====================================================
		var last model.SubmissionModel
		errLast := tx.Model(&model.SubmissionModel{}).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(`
        submission_school_id = ?
        AND submission_assessment_id = ?
        AND submission_student_id = ?
        AND submission_deleted_at IS NULL
    `, schoolID, subReq.SubmissionAssessmentID, sid).
			Order("submission_attempt_count DESC").
			Limit(1).
			First(&last).Error

		nextAttempt := 1
		if errLast == nil {
			nextAttempt = last.SubmissionAttemptCount + 1
		} else if !errors.Is(errLast, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghitung attempt submission")
		}

		// =====================================================
		// 2) Build model dari DTO (attempt_count dari backend)
		// =====================================================
		sub := subReq.ToModel(nextAttempt)

		// Rekomendasi: create selalu draft (biar cocok flow upload)
		// Tapi kalau kamu mau auto submitted juga, kamu boleh ubah di FE dan logic ini tetap aman.
		if sub.SubmissionStatus == "" {
			sub.SubmissionStatus = model.SubmissionStatusDraft
		}

		// Auto-submitted_at kalau status submitted/resubmitted tapi belum ada waktu
		if (sub.SubmissionStatus == model.SubmissionStatusSubmitted ||
			sub.SubmissionStatus == model.SubmissionStatusResubmitted) &&
			sub.SubmissionSubmittedAt == nil {

			t := now
			sub.SubmissionSubmittedAt = &t
		}

		// ✅ NOT NULL
		sub.SubmissionIsLate = false

		if err := tx.Create(&sub).Error; err != nil {
			if isDuplicateKey(err) {
				return fiber.NewError(fiber.StatusConflict, "Submission attempt ini sudah ada")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		created = &sub

		// =====================================================
		// 3) Build URL models dari upserts JSON/bracket
		// =====================================================
		var urlModels []model.SubmissionURLModel
		for _, u := range urlUpserts {
			row := model.SubmissionURLModel{
				SubmissionURLSchoolID:     schoolID,
				SubmissionURLSubmissionID: sub.SubmissionID,
				SubmissionURLKind:         strings.TrimSpace(strings.ToLower(u.SubmissionURLKind)),
				SubmissionURLLabel:        u.SubmissionURLLabel,
				SubmissionURL:             u.SubmissionURL,
				SubmissionURLObjectKey:    u.SubmissionURLObjectKey,
				SubmissionURLIsPrimary:    false,
				SubmissionURLOrder:        0,

				// ✅ WAJIB (FK NOT NULL)
				SubmissionURLStudentID: sid,

				// ✅ student create → teacher null
				SubmissionURLTeacherID: nil,
			}

			if row.SubmissionURLKind == "" {
				row.SubmissionURLKind = "attachment"
			}
			if u.SubmissionURLIsPrimary != nil {
				row.SubmissionURLIsPrimary = *u.SubmissionURLIsPrimary
			}
			if u.SubmissionURLOrder != nil {
				row.SubmissionURLOrder = *u.SubmissionURLOrder
			}
			urlModels = append(urlModels, row)
		}

		// =====================================================
		// 4) Dari files multipart → upload ke OSS dan isi baris URL
		// =====================================================
		if strings.HasPrefix(ct, "multipart/form-data") {
			if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
				var fhs []*multipart.FileHeader
				if tmp, _ := helperOSS.CollectUploadFiles(form, nil); len(tmp) > 0 {
					fhs = tmp
				}
				if len(fhs) > 0 {
					oss, oerr := helperOSS.NewOSSServiceFromEnv("")
					if oerr != nil {
						return fiber.NewError(fiber.StatusBadGateway, "OSS tidak siap")
					}
					ctx := context.Background()
					for _, fh := range fhs {
						publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, schoolID, "submissions", fh)
						if uerr != nil {
							return uerr
						}
						// slot kosong → reuse, jika tidak ada → append baru
						var row *model.SubmissionURLModel
						for i := range urlModels {
							if urlModels[i].SubmissionURL == nil && urlModels[i].SubmissionURLObjectKey == nil {
								row = &urlModels[i]
								break
							}
						}
						urlModels = append(urlModels, model.SubmissionURLModel{
							SubmissionURLSchoolID:     schoolID,
							SubmissionURLSubmissionID: sub.SubmissionID,
							SubmissionURLKind:         "attachment",
							SubmissionURLOrder:        len(urlModels) + 1,

							// ✅ WAJIB
							SubmissionURLStudentID: sid,
							SubmissionURLTeacherID: nil,
						})

						row.SubmissionURL = &publicURL
						if key, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
							row.SubmissionURLObjectKey = &key
						}
						if strings.TrimSpace(row.SubmissionURLKind) == "" {
							row.SubmissionURLKind = "attachment"
						}
					}
				}
			}
		}

		// =====================================================
		// 5) Simpan URL models (jika ada) + enforce primary unik per (submission, kind)
		// =====================================================
		if len(urlModels) > 0 {
			if err := tx.Create(&urlModels).Error; err != nil {
				if isDuplicateKey(err) {
					return fiber.NewError(fiber.StatusConflict, "Terdapat lampiran duplikat")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan lampiran")
			}
			for _, it := range urlModels {
				if it.SubmissionURLIsPrimary {
					if err := tx.Model(&model.SubmissionURLModel{}).
						Where(`
							submission_url_school_id = ?
							AND submission_url_submission_id = ?
							AND submission_url_kind = ?
							AND submission_url_id <> ?
							AND submission_url_deleted_at IS NULL
						`, schoolID, sub.SubmissionID, it.SubmissionURLKind, it.SubmissionURLID).
						Update("submission_url_is_primary", false).Error; err != nil {
						return fiber.NewError(fiber.StatusInternalServerError, "Gagal set primary lampiran")
					}
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

	// ---------- Response ----------
	resp := dto.FromModel(created)

	// Ambil URLs (live) supaya FE langsung dapat
	var rows []model.SubmissionURLModel
	_ = ctrl.DB.
		Where("submission_url_submission_id = ? AND submission_url_deleted_at IS NULL", created.SubmissionID).
		Order("submission_url_is_primary DESC, submission_url_order ASC, submission_url_created_at ASC").
		Find(&rows)

	return helper.JsonCreated(c, "Submission & lampiran berhasil dibuat", fiber.Map{
		"submission": resp,
		"urls": func() []fiber.Map {
			out := make([]fiber.Map, 0, len(rows))
			for i := range rows {
				out = append(out, fiber.Map{
					"submission_url_id":            rows[i].SubmissionURLID,
					"submission_url_school_id":     rows[i].SubmissionURLSchoolID,
					"submission_url_submission_id": rows[i].SubmissionURLSubmissionID,
					"submission_url_kind":          rows[i].SubmissionURLKind,
					"submission_url_href":          rows[i].SubmissionURL,
					"submission_url_object_key":    rows[i].SubmissionURLObjectKey,
					"submission_url_label":         rows[i].SubmissionURLLabel,
					"submission_url_order":         rows[i].SubmissionURLOrder,
					"submission_url_is_primary":    rows[i].SubmissionURLIsPrimary,
					"submission_url_created_at":    rows[i].SubmissionURLCreatedAt,
					"submission_url_updated_at":    rows[i].SubmissionURLUpdatedAt,
				})
			}
			return out
		}(),
	})
}

/*
PATCH /submissions/:id/urls   (WRITE — DKM/Teacher/Admin/Owner, school via token)
*/
func (ctrl *SubmissionController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// ── Resolve school + role guard (DKM/Teacher/Owner) via token ──
	schoolID, err := resolveTeacherSchoolFromParam(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	subID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil || subID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "submission id tidak valid")
	}

	// Pastikan submission milik school ini
	{
		var count int64
		if err := ctrl.DB.WithContext(c.Context()).
			Model(&model.SubmissionModel{}).
			Where("submission_id = ? AND submission_school_id = ? AND submission_deleted_at IS NULL", subID, schoolID).
			Count(&count).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
		}
		if count == 0 {
			return helper.JsonError(c, fiber.StatusForbidden, "Submission tidak ditemukan/diizinkan")
		}
	}

	// ── Parse payload upserts ──
	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	type upsert struct {
		ID          *uuid.UUID `json:"submission_url_id"`
		Kind        string     `json:"submission_url_kind"`
		Label       *string    `json:"submission_url_label"`
		Href        *string    `json:"submission_url_href"`
		ObjectKey   *string    `json:"submission_url_object_key"`
		Order       *int       `json:"submission_url_order"`
		IsPrimary   *bool      `json:"submission_url_is_primary"`
		ReplaceFile bool       `json:"replace_file"`
	}
	var ups []upsert

	if strings.HasPrefix(ct, "multipart/form-data") {
		payload := strings.TrimSpace(c.FormValue("urls_json"))
		if payload == "" {
			if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
				parsed := helperOSS.ParseURLUpsertsFromMultipart(form, &helperOSS.URLParseOptions{
					BracketPrefix: "urls",
					DefaultKind:   "attachment",
				})
				for _, p := range parsed {
					u := upsert{
						Kind:      p.Kind,
						Label:     p.Label,
						Href:      p.Href,
						ObjectKey: p.ObjectKey,
					}
					if p.Order != 0 {
						o := int(p.Order)
						u.Order = &o
					}
					if p.IsPrimary {
						ip := true
						u.IsPrimary = &ip
					}
					ups = append(ups, u)
				}
			}
			if len(ups) == 0 {
				return helper.JsonError(c, fiber.StatusBadRequest, "urls_json wajib diisi pada multipart")
			}
		} else {
			if err := json.Unmarshal([]byte(payload), &ups); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest, "urls_json tidak valid: "+err.Error())
			}
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

	err = ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		now, errNow := dbtime.GetDBTime(c)
		if errNow != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mendapatkan waktu server")
		}

		var existing []model.SubmissionURLModel
		if err := tx.Where(`
			submission_url_submission_id = ?
			AND submission_url_school_id = ?
			AND submission_url_deleted_at IS NULL
		`, subID, schoolID).Find(&existing).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil lampiran")
		}
		byID := map[uuid.UUID]*model.SubmissionURLModel{}
		for i := range existing {
			byID[existing[i].SubmissionURLID] = &existing[i]
		}

		var bucket *helperOSS.OSSService
		var haveOSS bool
		if strings.HasPrefix(ct, "multipart/form-data") {
			if svc, oerr := helperOSS.NewOSSServiceFromEnv(""); oerr == nil && svc != nil {
				bucket = svc
				haveOSS = true
			}
		}

		var files []*multipart.FileHeader
		if strings.HasPrefix(ct, "multipart/form-data") {
			if form, e := c.MultipartForm(); e == nil && form != nil {
				files, _ = helperOSS.CollectUploadFiles(form, nil)
			}
		}
		fileIdx := 0

		var touched []model.SubmissionURLModel

		for _, u := range ups {
			if u.ID == nil {
				// INSERT
				row := model.SubmissionURLModel{
					SubmissionURLSchoolID:     schoolID,
					SubmissionURLSubmissionID: subID,
					SubmissionURLKind:         u.Kind,
					SubmissionURLLabel:        u.Label,
					SubmissionURL:             u.Href,
					SubmissionURLObjectKey:    u.ObjectKey,
					SubmissionURLIsPrimary:    false,
					SubmissionURLOrder:        0,
				}
				if u.IsPrimary != nil {
					row.SubmissionURLIsPrimary = *u.IsPrimary
				}
				if u.Order != nil {
					row.SubmissionURLOrder = *u.Order
				}

				if u.ReplaceFile && haveOSS && fileIdx < len(files) {
					publicURL, uerr := helperOSS.UploadAnyToOSS(c.Context(), bucket, schoolID, "submissions", files[fileIdx])
					if uerr != nil {
						return uerr
					}
					fileIdx++
					row.SubmissionURL = &publicURL
					if key, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
						row.SubmissionURLObjectKey = &key
					}
				}

				if err := tx.Create(&row).Error; err != nil {
					if isDuplicateKey(err) {
						return fiber.NewError(fiber.StatusConflict, "URL duplikat untuk submission ini")
					}
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal menambah URL")
				}
				touched = append(touched, row)
				continue
			}

			// UPDATE
			ex, ok := byID[*u.ID]
			if !ok {
				return fiber.NewError(fiber.StatusNotFound, "URL tidak ditemukan untuk submission ini")
			}

			patch := map[string]any{}
			if u.Kind != "" && u.Kind != ex.SubmissionURLKind {
				patch["submission_url_kind"] = u.Kind
			}
			if u.Label != nil {
				patch["submission_url_label"] = u.Label
			}
			if u.IsPrimary != nil {
				patch["submission_url_is_primary"] = *u.IsPrimary
			}
			if u.Order != nil {
				patch["submission_url_order"] = *u.Order
			}

			if u.ReplaceFile && haveOSS && fileIdx < len(files) {
				publicURL, uerr := helperOSS.UploadAnyToOSS(c.Context(), bucket, schoolID, "submissions", files[fileIdx])
				if uerr != nil {
					return uerr
				}
				fileIdx++
				patch["submission_url_href"] = publicURL
				if key, kerr := helperOSS.ExtractKeyFromPublicURL(publicURL); kerr == nil {
					patch["submission_url_object_key"] = key
				}
				if ex.SubmissionURLObjectKey != nil && strings.TrimSpace(*ex.SubmissionURLObjectKey) != "" {
					patch["submission_url_object_key_old"] = *ex.SubmissionURLObjectKey
				}
			} else {
				if u.Href != nil {
					patch["submission_url_href"] = *u.Href
				}
				if u.ObjectKey != nil {
					patch["submission_url_object_key"] = *u.ObjectKey
				}
			}

			if len(patch) > 0 {
				patch["submission_url_updated_at"] = now
				if err := tx.Model(&model.SubmissionURLModel{}).
					Where("submission_url_id = ? AND submission_url_school_id = ? AND submission_url_submission_id = ? AND submission_url_deleted_at IS NULL",
						*u.ID, schoolID, subID).
					Updates(patch).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengubah URL")
				}
			}

			var latest model.SubmissionURLModel
			_ = tx.Where("submission_url_id = ?", *u.ID).First(&latest).Error
			touched = append(touched, latest)
		}

		// Enforce: satu primary per (submission, kind)
		for _, it := range touched {
			if it.SubmissionURLIsPrimary {
				if err := tx.Model(&model.SubmissionURLModel{}).
					Where(`
						submission_url_school_id = ?
						AND submission_url_submission_id = ?
						AND submission_url_kind = ?
						AND submission_url_id <> ?
						AND submission_url_deleted_at IS NULL
					`, schoolID, subID, it.SubmissionURLKind, it.SubmissionURLID).
					Update("submission_url_is_primary", false).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal set primary unik")
				}
			}
		}

		return nil
	})
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	var rows []model.SubmissionURLModel
	_ = ctrl.DB.
		Where("submission_url_submission_id = ? AND submission_url_school_id = ? AND submission_url_deleted_at IS NULL", subID, schoolID).
		Order("submission_url_is_primary DESC, submission_url_order ASC, submission_url_created_at ASC").
		Find(&rows)

	return helper.JsonUpdated(c, "Lampiran submission diperbarui", fiber.Map{
		"submission_id": subID,
		"urls":          rows,
	})
}

/*
DELETE /submissions/:id/urls/:urlId
*/
func (ctrl *SubmissionController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	schoolID, err := resolveTeacherSchoolFromParam(c)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	subID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil || subID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "submission id tidak valid")
	}
	urlID, err := uuid.Parse(strings.TrimSpace(c.Params("urlId")))
	if err != nil || urlID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "url id tidak valid")
	}

	var row model.SubmissionURLModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where(`
			submission_url_id = ?
			AND submission_url_submission_id = ?
			AND submission_url_school_id = ?
			AND submission_url_deleted_at IS NULL
		`, urlID, subID, schoolID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "URL tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	now, err := dbtime.GetDBTime(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mendapatkan waktu server")
	}

	move := true
	if v := strings.TrimSpace(c.Query("move")); v != "" {
		move = !(v == "0" || strings.EqualFold(v, "false") || strings.EqualFold(v, "no"))
	}

	updates := map[string]any{
		"submission_url_deleted_at": now,
	}

	if move && row.SubmissionURL != nil && strings.TrimSpace(*row.SubmissionURL) != "" {
		dstURL, merr := helperOSS.MoveToSpamByPublicURLENV(*row.SubmissionURL, 0)
		if merr == nil && strings.TrimSpace(dstURL) != "" {
			updates["submission_url_href"] = dstURL
			if key, kerr := helperOSS.ExtractKeyFromPublicURL(dstURL); kerr == nil {
				updates["submission_url_object_key"] = key
			}

			days := 30
			if v := os.Getenv("RETENTION_DAYS"); v != "" {
				if n, e := strconv.Atoi(v); e == nil && n > 0 {
					days = n
				}
			}
			updates["submission_url_delete_pending_until"] = now.Add(time.Duration(days) * 24 * time.Hour)
		}
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Model(&model.SubmissionURLModel{}).
		Where("submission_url_id = ? AND submission_url_school_id = ? AND submission_url_submission_id = ?", urlID, schoolID, subID).
		Updates(updates).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	return helper.JsonDeleted(c, "Lampiran submission dihapus", fiber.Map{
		"submission_id": subID,
		"url_id":        urlID,
	})
}

/* ===== Utils ===== */

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") ||
		strings.Contains(s, "violates unique constraint") ||
		strings.Contains(s, "unique constraint") ||
		strings.Contains(s, "sqlstate 23505")
}
