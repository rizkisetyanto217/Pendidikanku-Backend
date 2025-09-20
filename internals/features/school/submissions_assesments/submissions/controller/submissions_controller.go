// file: internals/features/school/attendance_assesment/submissions/controller/submission_controller.go
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

	dto "masjidku_backend/internals/features/school/submissions_assesments/submissions/dto"
	model "masjidku_backend/internals/features/school/submissions_assesments/submissions/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"
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

func clampPage(n int) int {
	if n <= 0 {
		return 1
	}
	return n
}
func clampPerPage(n int) int {
	if n <= 0 {
		return 20
	}
	if n > 200 {
		return 200
	}
	return n
}

func applyFilters(q *gorm.DB, f *dto.ListSubmissionsQuery) *gorm.DB {
	if f == nil {
		return q
	}
	if f.MasjidID != nil {
		q = q.Where("submissions_masjid_id = ?", *f.MasjidID)
	}
	if f.AssessmentID != nil {
		q = q.Where("submissions_assessment_id = ?", *f.AssessmentID)
	}
	if f.StudentID != nil {
		q = q.Where("submissions_student_id = ?", *f.StudentID)
	}
	if f.Status != nil {
		q = q.Where("submissions_status = ?", *f.Status)
	}
	if f.SubmittedFrom != nil {
		q = q.Where("submissions_submitted_at >= ?", *f.SubmittedFrom)
	}
	if f.SubmittedTo != nil {
		q = q.Where("submissions_submitted_at < ?", *f.SubmittedTo)
	}
	return q
}

func applySort(q *gorm.DB, sort string) *gorm.DB {
	switch strings.TrimSpace(sort) {
	case "created_at":
		return q.Order("submissions_created_at ASC")
	case "desc_created_at", "":
		return q.Order("submissions_created_at DESC")
	case "submitted_at":
		return q.Order("submissions_submitted_at ASC NULLS LAST")
	case "desc_submitted_at":
		return q.Order("submissions_submitted_at DESC NULLS LAST")
	case "score":
		return q.Order("submissions_score ASC NULLS LAST")
	case "desc_score":
		return q.Order("submissions_score DESC NULLS LAST")
	default:
		return q.Order("submissions_created_at DESC")
	}
}

/* =========================
   Handlers
========================= */

// POST / (STUDENT ONLY)
// - multipart/form-data:
//   - submission_json: JSON CreateSubmissionRequest (wajib)
//   - urls_json: JSON array upserts (opsional)
//   - bracket/array style: urls[0][kind], urls[0][label], urls[0][href], urls[0][object_key], urls[0][order], urls[0][is_primary]
//   - file uploads: diupload ke OSS → dibuat URL rows ========================================================= */
func (ctrl *SubmissionController) Create(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// ---------- Role & Masjid context (STUDENT) ----------
	// Student boleh tidak menyertakan context eksplisit; pakai active masjid di token.
	// Jika ada context via path/header/query/host, hormati dan pastikan caller adalah student pada masjid tsb.
	mc, _ := helperAuth.ResolveMasjidContext(c)

	var masjidID uuid.UUID
	if mc.ID != uuid.Nil || strings.TrimSpace(mc.Slug) != "" {
		id, er := func() (uuid.UUID, error) {
			if mc.ID != uuid.Nil {
				return mc.ID, nil
			}
			return helperAuth.GetMasjidIDBySlug(c, mc.Slug)
		}()
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusNotFound, "Masjid (context) tidak ditemukan")
		}
		masjidID = id
		if err := helperAuth.EnsureStudentMasjid(c, masjidID); err != nil {
			return err
		}
	} else {
		id, err := helperAuth.GetActiveMasjidID(c)
		if err != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid aktif tidak ditemukan di token")
		}
		masjidID = id
		if err := helperAuth.EnsureStudentMasjid(c, masjidID); err != nil {
			return err
		}
	}

	// Ambil student_id milik caller pada masjid ini
	sid, err := helperAuth.GetMasjidStudentIDForMasjid(c, masjidID)
	if err != nil || sid == uuid.Nil {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya siswa terdaftar yang diizinkan membuat submission")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get("Content-Type")))

	// ---------- Parse payload ----------
	var subReq dto.CreateSubmissionRequest

	// Upsert URL lokal (optional)
	type URLUpsert struct {
		Kind      string  `json:"kind"`
		Label     *string `json:"label"`
		Href      *string `json:"href"`
		ObjectKey *string `json:"object_key"`
		Order     *int32  `json:"order"`
		IsPrimary *bool   `json:"is_primary"`
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
			_ = json.Unmarshal([]byte(uj), &urlUpserts) // jika gagal, abaikan (tetap bisa dari bracket/files)
		}

		// Bracket/array style → helperOSS.ParseURLUpsertsFromMultipart
		if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
			parsed := helperOSS.ParseURLUpsertsFromMultipart(form, &helperOSS.URLParseOptions{
				BracketPrefix: "urls",
				DefaultKind:   "attachment",
			})
			for _, p := range parsed {
				up := URLUpsert{
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
				// normalisasi ringan
				k := strings.TrimSpace(strings.ToLower(up.Kind))
				if k == "" {
					k = "attachment"
				}
				up.Kind = k
				if up.Label != nil {
					l := strings.TrimSpace(*up.Label)
					up.Label = &l
				}
				if up.Href != nil {
					h := strings.TrimSpace(*up.Href)
					if h == "" {
						up.Href = nil
					} else {
						up.Href = &h
					}
				}
				if up.ObjectKey != nil {
					ok := strings.TrimSpace(*up.ObjectKey)
					if ok == "" {
						up.ObjectKey = nil
					} else {
						up.ObjectKey = &ok
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
	subReq.SubmissionMasjidID = masjidID
	subReq.SubmissionStudentID = sid

	// ---------- Validasi submission ----------
	if err := ctrl.Validator.Struct(&subReq); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// ---------- Normalisasi minor submission ----------
	// (jika status submitted/resubmitted dan submitted_at kosong, isi now)
	status := model.SubmissionStatusSubmitted
	if subReq.SubmissionStatus != nil {
		status = *subReq.SubmissionStatus
	}

	// ---------- Transaksi ----------
	var created *model.Submission

	if err := ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// 1) Simpan submission
		sub := &model.Submission{
			SubmissionMasjidID:     subReq.SubmissionMasjidID,
			SubmissionAssessmentID: subReq.SubmissionAssessmentID,
			SubmissionStudentID:    subReq.SubmissionStudentID,
			SubmissionText:         subReq.SubmissionText,
			SubmissionStatus:       status,
			SubmissionSubmittedAt:  subReq.SubmissionSubmittedAt,
			SubmissionIsLate:       subReq.SubmissionIsLate,
		}
		if (sub.SubmissionStatus == model.SubmissionStatusSubmitted || sub.SubmissionStatus == model.SubmissionStatusResubmitted) &&
			sub.SubmissionSubmittedAt == nil {
			now := time.Now()
			sub.SubmissionSubmittedAt = &now
		}

		if err := tx.Create(sub).Error; err != nil {
			le := strings.ToLower(err.Error())
			if strings.Contains(le, "duplicate key") || strings.Contains(le, "unique constraint") {
				return fiber.NewError(fiber.StatusConflict, "Submission untuk assessment & student ini sudah ada")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		created = sub

		// 2) Build URL models dari upserts JSON/bracket
		var urlModels []model.SubmissionURL
		for _, u := range urlUpserts {
			row := model.SubmissionURL{
				SubmissionURLMasjidID:     masjidID,
				SubmissionURLSubmissionID: sub.SubmissionID, // <-- jika fieldmu bernama SubmissionsID, sesuaikan di sini
				SubmissionURLKind:         strings.TrimSpace(strings.ToLower(u.Kind)),
				SubmissionURLLabel:        u.Label,
				SubmissionURLHref:         u.Href,
				SubmissionURLObjectKey:    u.ObjectKey,
				SubmissionURLIsPrimary:    false,
				SubmissionURLOrder:        0,
			}
			if row.SubmissionURLKind == "" {
				row.SubmissionURLKind = "attachment"
			}
			if u.IsPrimary != nil {
				row.SubmissionURLIsPrimary = *u.IsPrimary
			}
			if u.Order != nil {
				row.SubmissionURLOrder = *u.Order
			}
			urlModels = append(urlModels, row)
		}

		// 3) Dari files multipart → upload ke OSS dan isi baris URL
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
						publicURL, uerr := helperOSS.UploadAnyToOSS(ctx, oss, masjidID, "submissions", fh)
						if uerr != nil {
							return uerr // helper biasanya sudah fiber.Error-friendly
						}
						// Cari slot kosong (yang belum ada href/object_key), jika tak ada → buat baru
						var row *model.SubmissionURL
						for i := range urlModels {
							if urlModels[i].SubmissionURLHref == nil && urlModels[i].SubmissionURLObjectKey == nil {
								row = &urlModels[i]
								break
							}
						}
						if row == nil {
							urlModels = append(urlModels, model.SubmissionURL{
								SubmissionURLMasjidID:     masjidID,
								SubmissionURLSubmissionID: sub.SubmissionID,
								SubmissionURLKind:         "attachment",
								SubmissionURLOrder:        int32(len(urlModels) + 1),
							})
							row = &urlModels[len(urlModels)-1]
						}
						row.SubmissionURLHref = &publicURL
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

		// 4) Simpan URL models (jika ada)
		if len(urlModels) > 0 {
			if err := tx.Create(&urlModels).Error; err != nil {
				low := strings.ToLower(err.Error())
				if strings.Contains(low, "duplicate") || strings.Contains(low, "unique") {
					return fiber.NewError(fiber.StatusConflict, "Terdapat lampiran duplikat")
				}
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan lampiran")
			}
			// Enforce: satu primary per (submission, kind) untuk baris live
			for _, it := range urlModels {
				if it.SubmissionURLIsPrimary {
					if err := tx.Model(&model.SubmissionURL{}).
						Where(`
							submission_url_masjid_id = ?
							AND submission_url_submission_id = ?
							AND submission_url_kind = ?
							AND submission_url_id <> ?
							AND submission_url_deleted_at IS NULL
						`,
							masjidID, sub.SubmissionID, it.SubmissionURLKind, it.SubmissionURLID,
						).
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
	var rows []model.SubmissionURL
	_ = ctrl.DB.
		Where("submission_url_submission_id = ? AND submission_url_deleted_at IS NULL", created.SubmissionID).
		Order("submission_url_is_primary DESC, submission_url_order ASC, submission_url_created_at ASC").
		Find(&rows)

	// Kamu bisa menaruh ke field `urls` pada response, atau kirimkan sebagai properti terpisah
	return helper.JsonCreated(c, "Submission & lampiran berhasil dibuat", fiber.Map{
		"submission": resp,
		"urls": func() []fiber.Map {
			out := make([]fiber.Map, 0, len(rows))
			for i := range rows {
				out = append(out, fiber.Map{
					"id":            rows[i].SubmissionURLID,
					"masjid_id":     rows[i].SubmissionURLMasjidID,
					"submission_id": rows[i].SubmissionURLSubmissionID,
					"kind":          rows[i].SubmissionURLKind,
					"href":          rows[i].SubmissionURLHref,
					"object_key":    rows[i].SubmissionURLObjectKey,
					"label":         rows[i].SubmissionURLLabel,
					"order":         rows[i].SubmissionURLOrder,
					"is_primary":    rows[i].SubmissionURLIsPrimary,
					"created_at":    rows[i].SubmissionURLCreatedAt,
					"updated_at":    rows[i].SubmissionURLUpdatedAt,
				})
			}
			return out
		}(),
	})
}

/*
PATCH /submissions/:id/urls   (WRITE — DKM/Teacher/Admin/Owner)
Mendukung:
  - JSON:
    {
    "urls": [
    { "id": "...(opsional utk update)", "kind":"attachment","label":"..","href":"..","object_key":"..","order":1,"is_primary":true },
    ...
    ]
    }
  - multipart/form-data:
  - urls_json: JSON array seperti di atas (wajib utk multipart)
  - file uploads: akan dipasangkan berurutan ke item yg punya flag replace_file=true
    atau jika item insert baru tanpa href/object_key, akan diisi otomatis
  - bracket style opsional: urls[0][...] akan ikut diparse bila disediakan
*/
func (ctrl *SubmissionController) Patch(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	// ── Resolve masjid + role guard (DKM/Teacher/Owner) ──
	subID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "submission id tidak valid")
	}
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, masjidID); err != nil && !helperAuth.IsOwner(c) {
		return err
	}

	// Pastikan submission milik masjid ini
	{
		var count int64
		if err := ctrl.DB.WithContext(c.Context()).
			Model(&model.Submission{}).
			Where("submissions_id = ? AND submissions_masjid_id = ? AND submissions_deleted_at IS NULL", subID, masjidID).
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
		ID        *uuid.UUID `json:"id"`
		Kind      string     `json:"kind"`
		Label     *string    `json:"label"`
		Href      *string    `json:"href"`
		ObjectKey *string    `json:"object_key"`
		Order     *int32     `json:"order"`
		IsPrimary *bool      `json:"is_primary"`
		// multipart helper
		ReplaceFile bool `json:"replace_file"`
	}
	var ups []upsert

	if strings.HasPrefix(ct, "multipart/form-data") {
		// urls_json wajib
		payload := strings.TrimSpace(c.FormValue("urls_json"))
		if payload == "" {
			// izinkan bracket parse sebagai fallback
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
						o := int32(p.Order)
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
		// JSON murni
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
	err = ctrl.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// existing live rows
		var existing []model.SubmissionURL
		if err := tx.Where(`
			submission_url_submission_id = ?
			AND submission_url_masjid_id = ?
			AND submission_url_deleted_at IS NULL
		`, subID, masjidID).Find(&existing).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil lampiran")
		}
		byID := map[uuid.UUID]*model.SubmissionURL{}
		for i := range existing {
			byID[existing[i].SubmissionURLID] = &existing[i]
		}

		// OSS + file list (opsional)
		// OSS + file list (opsional)
		var bucket *helperOSS.OSSService
		var haveOSS bool
		if strings.HasPrefix(ct, "multipart/form-data") {
			if svc, oerr := helperOSS.NewOSSServiceFromEnv(""); oerr == nil && svc != nil {
				bucket = svc
				haveOSS = true
			}
		}

		// Kumpulkan file dari multipart (pakai helper default)
		var files []*multipart.FileHeader
		if strings.HasPrefix(ct, "multipart/form-data") {
			if form, e := c.MultipartForm(); e == nil && form != nil {
				files, _ = helperOSS.CollectUploadFiles(form, nil)
			}
		}
		fileIdx := 0

		var touched []model.SubmissionURL

		for _, u := range ups {
			if u.ID == nil {
				// INSERT
				row := model.SubmissionURL{
					SubmissionURLMasjidID:     masjidID,
					SubmissionURLSubmissionID: subID,
					SubmissionURLKind:         u.Kind,
					SubmissionURLLabel:        u.Label,
					SubmissionURLHref:         u.Href,
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

				// upload file baru jika diminta (replace_file saat insert = upload)
				if u.ReplaceFile && haveOSS && fileIdx < len(files) {
					publicURL, uerr := helperOSS.UploadAnyToOSS(c.Context(), bucket, masjidID, "submissions", files[fileIdx])
					if uerr != nil {
						return uerr
					}
					fileIdx++
					row.SubmissionURLHref = &publicURL
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

			// replace file → simpan key lama di *_old
			if u.ReplaceFile && haveOSS && fileIdx < len(files) {
				publicURL, uerr := helperOSS.UploadAnyToOSS(c.Context(), bucket, masjidID, "submissions", files[fileIdx])
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
				// update manual href/object_key via JSON
				if u.Href != nil {
					patch["submission_url_href"] = *u.Href
				}
				if u.ObjectKey != nil {
					patch["submission_url_object_key"] = *u.ObjectKey
				}
			}

			if len(patch) > 0 {
				patch["submission_url_updated_at"] = time.Now()
				if err := tx.Model(&model.SubmissionURL{}).
					Where("submission_url_id = ? AND submission_url_masjid_id = ? AND submission_url_submission_id = ? AND submission_url_deleted_at IS NULL",
						*u.ID, masjidID, subID).
					Updates(patch).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengubah URL")
				}
			}

			var latest model.SubmissionURL
			_ = tx.Where("submission_url_id = ?", *u.ID).First(&latest).Error
			touched = append(touched, latest)
		}

		// Enforce: satu primary per (submission, kind)
		for _, it := range touched {
			if it.SubmissionURLIsPrimary {
				if err := tx.Model(&model.SubmissionURL{}).
					Where(`
						submission_url_masjid_id = ?
						AND submission_url_submission_id = ?
						AND submission_url_kind = ?
						AND submission_url_id <> ?
						AND submission_url_deleted_at IS NULL
					`, masjidID, subID, it.SubmissionURLKind, it.SubmissionURLID).
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

	// Response: list terbaru
	var rows []model.SubmissionURL
	_ = ctrl.DB.
		Where("submission_url_submission_id = ? AND submission_url_masjid_id = ? AND submission_url_deleted_at IS NULL", subID, masjidID).
		Order("submission_url_is_primary DESC, submission_url_order ASC, submission_url_created_at ASC").
		Find(&rows)

	return helper.JsonUpdated(c, "Lampiran submission diperbarui", fiber.Map{
		"submission_id": subID,
		"urls":          rows,
	})
}

/*
DELETE /submissions/:submissionId/urls/:urlId (WRITE — DKM/Teacher/Admin/Owner)
Query opsional:
  - ?move=1 (default) → pindahkan objek ke folder spam/.. di OSS dan update href/object_key ke lokasi spam,
    serta set delete_pending_until (supaya reaper bisa hapus permanen setelah retensi)
  - ?move=0 → TIDAK memindahkan objek. Hanya soft delete; object_key dibiarkan apa adanya.
*/
func (ctrl *SubmissionController) Delete(c *fiber.Ctx) error {
	c.Locals("DB", ctrl.DB)

	subID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "submission id tidak valid")
	}
	urlID, err := uuid.Parse(strings.TrimSpace(c.Params("urlId")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "url id tidak valid")
	}

	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil || masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan di token")
	}
	if err := helperAuth.EnsureDKMOrTeacherMasjid(c, masjidID); err != nil && !helperAuth.IsOwner(c) {
		return err
	}

	// Ambil rownya (live)
	var row model.SubmissionURL
	if err := ctrl.DB.WithContext(c.Context()).
		Where(`
			submission_url_id = ?
			AND submission_url_submission_id = ?
			AND submission_url_masjid_id = ?
			AND submission_url_deleted_at IS NULL
		`, urlID, subID, masjidID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "URL tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Opsional move ke spam
	move := true
	if v := strings.TrimSpace(c.Query("move")); v != "" {
		move = !(v == "0" || strings.EqualFold(v, "false") || strings.EqualFold(v, "no"))
	}

	updates := map[string]any{
		"submission_url_deleted_at": time.Now(),
	}

	// jika move, copy ke spam & update href/object_key → lalu set delete_pending_until
	if move && row.SubmissionURLHref != nil && strings.TrimSpace(*row.SubmissionURLHref) != "" {
		dstURL, merr := helperOSS.MoveToSpamByPublicURLENV(*row.SubmissionURLHref, 0)
		if merr == nil && strings.TrimSpace(dstURL) != "" {
			updates["submission_url_href"] = dstURL
			if key, kerr := helperOSS.KeyFromPublicURL(dstURL); kerr == nil {
				updates["submission_url_object_key"] = key
			}
			// Retention window utk purge: pakai ENV RETENTION_DAYS (default 30)
			days := 30
			if v := os.Getenv("RETENTION_DAYS"); v != "" {
				if n, e := strconv.Atoi(v); e == nil && n > 0 {
					days = n
				}
			}
			updates["submission_url_delete_pending_until"] = time.Now().Add(time.Duration(days) * 24 * time.Hour)
		}
		// kalau move error → tetap soft-delete tanpa update href/object_key (best-effort)
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Model(&model.SubmissionURL{}).
		Where("submission_url_id = ? AND submission_url_masjid_id = ? AND submission_url_submission_id = ?", urlID, masjidID, subID).
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
