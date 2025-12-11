// file: internals/features/lembaga/subjects/main/controller/subjects_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strings"
	"time"

	subjectDTO "madinahsalam_backend/internals/features/school/academics/subjects/dto"
	subjectModel "madinahsalam_backend/internals/features/school/academics/subjects/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"
	helperOSS "madinahsalam_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

/* =======================================================
   CONTROLLER
   ======================================================= */

type SubjectsController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewSubjectsController(db *gorm.DB, v interface{ Struct(any) error }) *SubjectsController {
	return &SubjectsController{DB: db, Validator: v}
}

/*
=========================================================

	CREATE (DKM/Admin only) — slug unik + optional upload

=========================================================
*/
func (h *SubjectsController) Create(c *fiber.Ctx) error {
	log.Printf("[SUBJECTS][CREATE] ▶️ incoming request")
	c.Locals("DB", h.DB)

	// 1) Parse payload
	var p subjectDTO.CreateSubjectRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// Normalisasi ringan
	p.Code = strings.TrimSpace(p.Code)
	p.Name = strings.TrimSpace(p.Name)
	if p.Desc != nil {
		d := strings.TrimSpace(*p.Desc)
		if d == "" {
			p.Desc = nil
		} else {
			p.Desc = &d
		}
	}

	// 2) Resolve school context (TOKEN dulu, lalu fallback) + DKM/Admin guard
	var schoolID uuid.UUID

	// 2a) PRIORITAS: ambil school_id dari token (owner/teacher/dkm/admin)
	if id, err := helperAuth.GetSchoolIDFromTokenPreferTeacher(c); err == nil && id != uuid.Nil {
		schoolID = id
	} else {
		// 2b) FALLBACK: ResolveSchoolContext (path/header/slug/host)
		mc, err := helperAuth.ResolveSchoolContext(c)
		if err != nil {
			if fe, ok := err.(*fiber.Error); ok {
				return helper.JsonError(c, fe.Code, fe.Message)
			}
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}

		switch {
		case mc.ID != uuid.Nil:
			schoolID = mc.ID

		case strings.TrimSpace(mc.Slug) != "":
			id, er := helperAuth.GetSchoolIDBySlug(c, strings.TrimSpace(mc.Slug))
			if er != nil {
				if errors.Is(er, gorm.ErrRecordNotFound) {
					return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
				}
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve school dari slug")
			}
			schoolID = id

		default:
			return helper.JsonError(c, fiber.StatusBadRequest, "Konteks sekolah tidak ditemukan")
		}
	}

	// 🔒 Hanya DKM/Admin di school ini
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	// Paksa body sesuai context
	p.SchoolID = schoolID

	// 3) Uniqueness: code unik per school (alive)
	if strings.TrimSpace(p.Code) != "" {
		var cnt int64
		if err := h.DB.Model(&subjectModel.SubjectModel{}).
			Where(`
				subject_school_id = ?
				AND lower(subject_code) = lower(?)
				AND subject_deleted_at IS NULL
			`, schoolID, p.Code).
			Count(&cnt).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek duplikasi kode")
		}
		if cnt > 0 {
			return helper.JsonError(c, fiber.StatusConflict, "Kode mapel sudah digunakan")
		}
	}

	// 4) Slug unik (CI) per school
	var baseSlug string
	if p.Slug != nil && strings.TrimSpace(*p.Slug) != "" {
		baseSlug = helper.Slugify(*p.Slug, 160)
	} else {
		baseSlug = helper.Slugify(p.Name, 160)
		if baseSlug == "" {
			baseSlug = "subject"
		}
	}
	scope := func(q *gorm.DB) *gorm.DB {
		return q.Where("subject_school_id = ? AND subject_deleted_at IS NULL", schoolID)
	}
	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(),
		h.DB,
		"subjects",
		"subject_slug",
		baseSlug,
		scope,
		160,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
	}

	// 5) Build entity & simpan
	ent := p.ToModel()
	ent.SubjectSchoolID = schoolID
	ent.SubjectSlug = uniqueSlug

	if err := h.DB.Create(&ent).Error; err != nil {
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "uq_subjects_code_per_school_alive"):
			return helper.JsonError(c, fiber.StatusConflict, "Kode mapel sudah digunakan")
		case strings.Contains(msg, "uq_subjects_slug_per_school_alive"):
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di school ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan subject")
	}

	// 6) Optional upload image → update kolom image di DB
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		keyPrefix := fmt.Sprintf("schools/%s/classes/subjects", schoolID.String())
		if svc, er := helperOSS.NewOSSServiceFromEnv(""); er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			if url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {
				uploadedURL = url

				objKey := ""
				if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
					objKey = k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
					objKey = k2
				}

				_ = h.DB.WithContext(c.Context()).
					Model(&subjectModel.SubjectModel{}).
					Where("subject_id = ?", ent.SubjectID).
					Updates(map[string]any{
						"subject_image_url":        uploadedURL,
						"subject_image_object_key": objKey,
					}).Error

				ent.SubjectImageURL = &uploadedURL
				if objKey != "" {
					ent.SubjectImageObjectKey = &objKey
				} else {
					ent.SubjectImageObjectKey = nil
				}
			}
		}
	}

	// 7) Reload (best effort)
	_ = h.DB.WithContext(c.Context()).
		First(&ent, "subject_id = ?", ent.SubjectID).Error

	// 🔹 Response: pakai timezone sekolah (dbtime)
	return helper.JsonCreated(
		c,
		"Berhasil membuat subject",
		subjectDTO.FromSubjectModelWithSchoolTime(c, ent),
	)
}

/*
=========================================================

	PATCH (DKM/Admin only) — tri-state + slug unique + optional upload
	+ sync cache ke class_subjects (name/code/slug[/url])

=========================================================
*/
func (h *SubjectsController) Patch(c *fiber.Ctx) error {
	log.Printf("[SUBJECTS][PATCH] ▶️ incoming request")
	c.Locals("DB", h.DB)

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Ambil record lama (alive)
	var ent subjectModel.SubjectModel
	if err := h.DB.WithContext(c.Context()).
		Where("subject_id = ? AND subject_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Subject tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard: Hanya DKM/Admin pada school terkait
	if err := helperAuth.EnsureDKMSchool(c, ent.SubjectSchoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	// ===== Parse payload (JSON vs multipart) =====
	var req subjectDTO.UpdateSubjectRequest
	var fh *multipart.FileHeader

	ct := strings.ToLower(c.Get("Content-Type"))
	if strings.HasPrefix(ct, "multipart/form-data") {
		r, f, perr := subjectDTO.BindMultipartPatch(c)
		if perr != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, perr.Error())
		}
		req = r
		fh = f
	} else {
		if err := c.BodyParser(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
		}
	}

	// Paksa context & normalisasi
	req.SchoolID = &ent.SubjectSchoolID
	req.Normalize()

	// ====== Uniqueness checks (bila berubah) ======
	// code
	if req.Code.Present && req.Code.Value != nil && !strings.EqualFold(ent.SubjectCode, *req.Code.Value) {
		var cnt int64
		if err := h.DB.Model(&subjectModel.SubjectModel{}).
			Where(`
                subject_school_id = ?
                AND subject_id <> ?
                AND subject_deleted_at IS NULL
                AND lower(subject_code) = lower(?)
            `, ent.SubjectSchoolID, ent.SubjectID, *req.Code.Value).
			Count(&cnt).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek duplikasi kode")
		}
		if cnt > 0 {
			return helper.JsonError(c, fiber.StatusConflict, "Kode mapel sudah digunakan")
		}
	}
	// slug
	if req.Slug.Present && req.Slug.Value != nil && !strings.EqualFold(ent.SubjectSlug, *req.Slug.Value) {
		var cnt int64
		if err := h.DB.Model(&subjectModel.SubjectModel{}).
			Where(`
                subject_school_id = ?
                AND subject_id <> ?
                AND subject_deleted_at IS NULL
                AND lower(subject_slug) = lower(?)
            `, ent.SubjectSchoolID, ent.SubjectID, *req.Slug.Value).
			Count(&cnt).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek duplikasi slug")
		}
		if cnt > 0 {
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di school ini")
		}
	}

	// Terapkan patch ke entity (+ timestamp di Apply)
	req.Apply(&ent)

	// Jika slug tidak dikirim tetapi name berubah, regen slug unik
	if !req.Slug.Present && req.Name.Present && ent.SubjectName != "" {
		base := helper.Slugify(ent.SubjectName, 160)
		if base == "" {
			base = "subject"
		}
		uniq, er := helper.EnsureUniqueSlugCI(
			c.Context(),
			h.DB,
			"subjects",
			"subject_slug",
			base,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"subject_school_id = ? AND subject_id <> ? AND subject_deleted_at IS NULL",
					ent.SubjectSchoolID, ent.SubjectID,
				)
			},
			160,
		)
		if er != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		ent.SubjectSlug = uniq
	}

	// Build patch map (ikutkan kolom image hanya jika Present)
	patch := map[string]any{
		"subject_school_id":  ent.SubjectSchoolID,
		"subject_code":       ent.SubjectCode,
		"subject_name":       ent.SubjectName,
		"subject_desc":       ent.SubjectDesc,
		"subject_is_active":  ent.SubjectIsActive,
		"subject_slug":       ent.SubjectSlug,
		"subject_updated_at": ent.SubjectUpdatedAt,
	}
	if req.ImageURL.Present {
		patch["subject_image_url"] = ent.SubjectImageURL
	}
	if req.ImageObjectKey.Present {
		patch["subject_image_object_key"] = ent.SubjectImageObjectKey
	}
	if req.ImageURLOld.Present {
		patch["subject_image_url_old"] = ent.SubjectImageURLOld
	}
	if req.ImageObjectKeyOld.Present {
		patch["subject_image_object_key_old"] = ent.SubjectImageObjectKeyOld
	}
	if req.ImageDeletePendingUntil.Present {
		patch["subject_image_delete_pending_until"] = ent.SubjectImageDeletePendingUntil
	}

	// Simpan patch dasar
	if err := h.DB.WithContext(c.Context()).
		Model(&subjectModel.SubjectModel{}).
		Where("subject_id = ?", ent.SubjectID).
		Updates(patch).Error; err != nil {
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "uq_subjects_code_per_school_alive"):
			return helper.JsonError(c, fiber.StatusConflict, "Kode mapel sudah digunakan")
		case strings.Contains(msg, "uq_subjects_slug_per_school_alive"):
			return helper.JsonError(c, fiber.StatusConflict, "Slug sudah digunakan di school ini")
		case strings.Contains(msg, "duplicate"), strings.Contains(msg, "unique"):
			return helper.JsonError(c, fiber.StatusConflict, "Duplikasi data (kode/slug)")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	// ==== Sinkronisasi cache di class_subjects (name/code/slug[/url opsional]) ====
	trimOrNil := func(s string) *string {
		t := strings.TrimSpace(s)
		if t == "" {
			return nil
		}
		return &t
	}
	snapName := trimOrNil(ent.SubjectName)
	snapCode := trimOrNil(ent.SubjectCode)
	snapSlug := trimOrNil(ent.SubjectSlug)

	snapPatch := map[string]any{
		"class_subject_subject_name_cache": func() any {
			if snapName == nil {
				return gorm.Expr("NULL")
			}
			return *snapName
		}(),
		"class_subject_subject_code_cache": func() any {
			if snapCode == nil {
				return gorm.Expr("NULL")
			}
			return *snapCode
		}(),
		"class_subject_subject_slug_cache": func() any {
			if snapSlug == nil {
				return gorm.Expr("NULL")
			}
			return *snapSlug
		}(),
	}
	txSync := h.DB.WithContext(c.Context()).
		Table("class_subjects").
		Where(`
			class_subject_subject_id = ?
			AND class_subject_school_id = ?
			AND class_subject_deleted_at IS NULL
		`, ent.SubjectID, ent.SubjectSchoolID).
		Updates(snapPatch)
	if txSync.Error != nil {
		log.Printf("[SUBJECTS][PATCH] sync class_subjects cache error: %v", txSync.Error)
	} else {
		log.Printf("[SUBJECTS][PATCH] sync class_subjects cache ok, rows=%d", txSync.RowsAffected)
	}

	// ===== Optional: upload image jika ada file =====
	uploadedURL := ""

	if fh == nil {
		fh = pickImageFile(c, "image", "file")
	}
	if fh != nil {
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er == nil {
			ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
			defer cancel()

			keyPrefix := fmt.Sprintf("schools/%s/classes/subjects", ent.SubjectSchoolID.String())
			if url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix); upErr == nil {
				uploadedURL = url

				// object key baru
				newObjKey := ""
				if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
					newObjKey = k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
					newObjKey = k2
				}

				// --- ambil url & key lama dari DB (best effort) ---
				var oldURL, oldObjKey string
				{
					type row struct {
						URL string `gorm:"column:subject_image_url"`
						Key string `gorm:"column:subject_image_object_key"`
					}
					var r row
					_ = h.DB.WithContext(c.Context()).
						Table("subjects").
						Select("subject_image_url, subject_image_object_key").
						Where("subject_id = ?", ent.SubjectID).
						Take(&r).Error
					oldURL = strings.TrimSpace(r.URL)
					oldObjKey = strings.TrimSpace(r.Key)
				}

				// --- pindahkan lama ke spam (kalau ada) ---
				movedURL := ""
				if oldURL != "" {
					if mv, mvErr := helperOSS.MoveToSpamByPublicURLENV(oldURL, 0); mvErr == nil {
						movedURL = mv
						if k, e := helperOSS.ExtractKeyFromPublicURL(movedURL); e == nil {
							oldObjKey = k
						} else if k2, e2 := helperOSS.KeyFromPublicURL(movedURL); e2 == nil {
							oldObjKey = k2
						}
					}
				}

				deletePendingUntil := time.Now().Add(30 * 24 * time.Hour)

				_ = h.DB.WithContext(c.Context()).
					Model(&subjectModel.SubjectModel{}).
					Where("subject_id = ?", ent.SubjectID).
					Updates(map[string]any{
						"subject_image_url":        uploadedURL,
						"subject_image_object_key": newObjKey,
						"subject_image_url_old": func() any {
							if movedURL == "" {
								return gorm.Expr("NULL")
							}
							return movedURL
						}(),
						"subject_image_object_key_old": func() any {
							if oldObjKey == "" {
								return gorm.Expr("NULL")
							}
							return oldObjKey
						}(),
						"subject_image_delete_pending_until": deletePendingUntil,
					}).Error

				ent.SubjectImageURL = &uploadedURL
				if newObjKey != "" {
					ent.SubjectImageObjectKey = &newObjKey
				} else {
					ent.SubjectImageObjectKey = nil
				}
				if movedURL != "" {
					ent.SubjectImageURLOld = &movedURL
				} else {
					ent.SubjectImageURLOld = nil
				}
				if oldObjKey != "" {
					ent.SubjectImageObjectKeyOld = &oldObjKey
				} else {
					ent.SubjectImageObjectKeyOld = nil
				}
				ent.SubjectImageDeletePendingUntil = &deletePendingUntil
			}
		}
	}

	// Reload (best effort)
	_ = h.DB.WithContext(c.Context()).
		First(&ent, "subject_id = ?", ent.SubjectID).Error

	// 🔹 Response: pakai versi timezone-aware
	return helper.JsonOK(
		c,
		"Berhasil memperbarui subject",
		subjectDTO.FromSubjectModelWithSchoolTime(c, ent),
	)
}

/*
=========================================================

	DELETE (soft delete, DKM/Admin only) + optional file cleanup

=========================================================
*/
func (h *SubjectsController) Delete(c *fiber.Ctx) error {
	log.Printf("[SUBJECTS][DELETE] ▶️ incoming request")
	c.Locals("DB", h.DB)

	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var ent subjectModel.SubjectModel
	if err := h.DB.WithContext(c.Context()).
		First(&ent, "subject_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Subject tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	if err := helperAuth.EnsureDKMSchool(c, ent.SubjectSchoolID); err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return helper.JsonError(c, fe.Code, fe.Message)
		}
		return helper.JsonError(c, fiber.StatusForbidden, err.Error())
	}

	var usedCount int64
	if err := h.DB.WithContext(c.Context()).
		Model(&subjectModel.ClassSubjectModel{}).
		Where(`
			class_subject_school_id = ?
			AND class_subject_subject_id = ?
			AND class_subject_deleted_at IS NULL
		`, ent.SubjectSchoolID, ent.SubjectID).
		Count(&usedCount).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengecek pemakaian subject")
	}

	if usedCount > 0 {
		return helper.JsonError(
			c,
			fiber.StatusBadRequest,
			"Subject tidak dapat dihapus karena masih digunakan pada mapel kelas (class_subjects). Hapus/ubah relasi tersebut terlebih dahulu.",
		)
	}

	now := time.Now()
	justDeleted := false
	if !ent.SubjectDeletedAt.Valid {
		if err := h.DB.WithContext(c.Context()).
			Model(&subjectModel.SubjectModel{}).
			Where("subject_id = ?", ent.SubjectID).
			Updates(map[string]any{
				"subject_deleted_at": &now,
				"subject_updated_at": now,
			}).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus subject")
		}
		justDeleted = true
	}

	imageURL := strings.TrimSpace(c.Query("image_url"))
	if imageURL == "" {
		if v := strings.TrimSpace(c.FormValue("image_url")); v != "" {
			imageURL = v
		}
	}
	if imageURL == "" && ent.SubjectImageURL != nil && strings.TrimSpace(*ent.SubjectImageURL) != "" {
		imageURL = strings.TrimSpace(*ent.SubjectImageURL)
	}

	if imageURL != "" {
		_, _ = helperOSS.MoveToSpamByPublicURLENV(imageURL, 0)
	}

	msg := "Subject sudah dihapus"
	if justDeleted {
		msg = "Berhasil menghapus subject"
	}
	return helper.JsonOK(c, msg, fiber.Map{
		"subject_id": ent.SubjectID,
		"image_url":  imageURL,
	})
}

/* =======================================================
   Util
   ======================================================= */

func pickImageFile(c *fiber.Ctx, keys ...string) *multipart.FileHeader {
	if form, err := c.MultipartForm(); err == nil && form != nil {
		for _, k := range keys {
			if files, ok := form.File[k]; ok && len(files) > 0 {
				if files[0] != nil && files[0].Size > 0 {
					return files[0]
				}
			}
		}
	}
	if len(keys) > 0 {
		if fh, err := c.FormFile(keys[0]); err == nil && fh != nil && fh.Size > 0 {
			return fh
		}
	}
	return nil
}
