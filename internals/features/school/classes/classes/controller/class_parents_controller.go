// file: internals/features/school/classes/classes/controller/class_parents_controller.go
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"strings"
	"time"

	cpdto "schoolku_backend/internals/features/school/classes/classes/dto"
	classModel "schoolku_backend/internals/features/school/classes/classes/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	helperOSS "schoolku_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClassParentController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewClassParentController(db *gorm.DB, v interface{ Struct(any) error }) *ClassParentController {
	return &ClassParentController{DB: db, Validator: v}
}

// Ambil file dari multipart (nama yang didukung)
func getImageFormFile(c *fiber.Ctx) (*multipart.FileHeader, error) {
	names := []string{"image", "avatar", "photo", "file", "picture"}
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil {
			return fh, nil
		}
	}
	return nil, errors.New("gambar tidak ditemukan")
}

/*
=========================================================
CREATE (staff only) — slug unik + optional upload (save to DB)
=========================================================
*/
func (ctl *ClassParentController) Create(c *fiber.Ctx) error {
	// 1) Parse payload
	var p cpdto.ClassParentCreateRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	p.Normalize()

	if len(p.ClassParentRequirements.ToJSONMap()) == 0 {
		// BodyParser pada multipart kadang tidak memanggil UnmarshalText → parse manual dari form-data
		raw := strings.TrimSpace(c.FormValue("class_parent_requirements"))
		if raw != "" {
			var tmp datatypes.JSONMap
			if err := json.Unmarshal([]byte(raw), &tmp); err != nil {
				return helper.JsonError(c, fiber.StatusBadRequest,
					"class_parent_requirements harus JSON object yang valid: "+err.Error())
			}
			p.ClassParentRequirements = cpdto.JSONMapFlexible(tmp)
		}
	}

	// 2) Resolve school **dari URL/slug saja** (TANPA fallback token)
	var schoolID uuid.UUID
	// a) coba param :school_id
	if s := strings.TrimSpace(c.Params("school_id")); s != "" {
		id, err := uuid.Parse(s)
		if err != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "school_id pada URL tidak valid")
		}
		schoolID = id
	} else if s := strings.TrimSpace(c.Params("school_slug")); s != "" {
		// b) atau param :school_slug (kalau route pakai slug)
		id, er := helperAuth.GetSchoolIDBySlug(c, s)
		if er != nil || id == uuid.Nil {
			return helper.JsonError(c, fiber.StatusNotFound, "School (slug) tidak ditemukan")
		}
		schoolID = id
	} else {
		// ❗ TIDAK ADA fallback ke GetSchoolIDFromTokenPreferTeacher di endpoint admin/staff
		return helper.JsonError(c, fiber.StatusBadRequest, "School context wajib via URL (school_id/school_slug)")
	}

	// 3) Staff guard di MASJID TARGET
	if err := helperAuth.EnsureStaffSchoolStrict(c, schoolID); err != nil {
		return err
	}

	// 4) Paksa body sesuai context (abaikan yang datang dari client)
	if p.ClassParentSchoolID == uuid.Nil {
		p.ClassParentSchoolID = schoolID
	} else if p.ClassParentSchoolID != schoolID {
		return helper.JsonError(c, fiber.StatusConflict, "class_parent_school_id pada body tidak cocok dengan konteks school")
	}

	// 5) Uniqueness: code (opsional)
	if p.ClassParentCode != nil {
		code := strings.TrimSpace(*p.ClassParentCode)
		if code != "" {
			var cnt int64
			if err := ctl.DB.Model(&classModel.ClassParentModel{}).
				Where("class_parent_school_id = ? AND class_parent_code = ? AND class_parent_deleted_at IS NULL",
					schoolID, code).
				Count(&cnt).Error; err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek kode")
			}
			if cnt > 0 {
				return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan")
			}
		}
	}

	// 6) Slug unik (CI) per school
	var baseSlug string
	if p.ClassParentSlug != nil && strings.TrimSpace(*p.ClassParentSlug) != "" {
		baseSlug = helper.Slugify(*p.ClassParentSlug, 160)
	} else {
		baseSlug = helper.Slugify(p.ClassParentName, 160)
		if baseSlug == "" {
			baseSlug = "item"
		}
	}
	scope := func(q *gorm.DB) *gorm.DB {
		return q.Where("class_parent_school_id = ?", schoolID).
			Where("class_parent_deleted_at IS NULL")
	}
	uniqueSlug, err := helper.EnsureUniqueSlugCI(
		c.Context(), ctl.DB,
		"class_parents", "class_parent_slug",
		baseSlug, scope, 160,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug")
	}

	// 7) Build entity & simpan (pakai schoolID yang kita lock)
	ent := p.ToModel()
	ent.ClassParentSchoolID = schoolID
	entSlug := uniqueSlug
	ent.ClassParentSlug = &entSlug

	if err := ctl.DB.Create(ent).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	// 8) Optional upload file → simpan ke DB (image_url + object_key)
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		url, upErr := helperOSS.UploadImageToOSSScoped(schoolID, "classes/class-parents", fh)
		if upErr == nil {
			uploadedURL = url

			objKey := ""
			if k, er := helperOSS.ExtractKeyFromPublicURL(uploadedURL); er == nil {
				objKey = k
			} else if k2, er2 := helperOSS.KeyFromPublicURL(uploadedURL); er2 == nil {
				objKey = k2
			}

			_ = ctl.DB.WithContext(c.Context()).
				Model(&classModel.ClassParentModel{}).
				Where("class_parent_id = ?", ent.ClassParentID).
				Updates(map[string]any{
					"class_parent_image_url":        uploadedURL,
					"class_parent_image_object_key": objKey,
				}).Error
		}
		// upload gagal → biarkan create tetap sukses
	}

	// 9) Reload entity (ambil field terbaru)
	_ = ctl.DB.WithContext(c.Context()).
		First(&ent, "class_parent_id = ?", ent.ClassParentID).Error

	// 10) Response — data langsung object class_parent
	return helper.JsonCreated(c, "Berhasil membuat parent kelas", cpdto.FromModelClassParent(ent))
}

/*
=========================================================
PATCH /api/a/:school_id/class-parents/:id
=========================================================
*/
func (ctl *ClassParentController) Patch(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	tx := ctl.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, tx.Error.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	var ent classModel.ClassParentModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("class_parent_id = ? AND class_parent_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {

		_ = tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard staff/tenant
	if err := helperAuth.EnsureStaffSchool(c, ent.ClassParentSchoolID); err != nil {
		_ = tx.Rollback()
		return err
	}

	// === Parse payload (JSON / multipart) → tri-state
	var p cpdto.ClassParentPatchRequest
	if err := cpdto.DecodePatchClassParentFromRequest(c, &p); err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// === Uniqueness: code (jika diubah & non-empty)
	if p.ClassParentCode.Present && p.ClassParentCode.Value != nil {
		if v := strings.TrimSpace(**p.ClassParentCode.Value); v != "" {
			var cnt int64
			if err := tx.Model(&classModel.ClassParentModel{}).
				Where(`class_parent_school_id = ? AND class_parent_code = ? AND class_parent_id <> ? AND class_parent_deleted_at IS NULL`,
					ent.ClassParentSchoolID, v, ent.ClassParentID).
				Count(&cnt).Error; err != nil {

				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek kode")
			}
			if cnt > 0 {
				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan")
			}
		}
	}

	// === SIMPAN NILAI LAMA untuk deteksi refresh snapshot
	oldCode := ""
	if ent.ClassParentCode != nil {
		oldCode = *ent.ClassParentCode
	}
	oldName := ent.ClassParentName

	var oldSlug *string
	if ent.ClassParentSlug != nil {
		s := *ent.ClassParentSlug
		oldSlug = &s
	}

	var oldLevel *int16
	if ent.ClassParentLevel != nil {
		lv := *ent.ClassParentLevel
		oldLevel = &lv
	}

	// === Apply patch ke entity in-memory
	p.Apply(&ent)

	// === Slug handling (unique per school)
	if p.ClassParentSlug.Present {
		if p.ClassParentSlug.Value != nil {
			base := helper.Slugify(strings.TrimSpace(**p.ClassParentSlug.Value), 100)
			if base == "" {
				base = helper.SuggestSlugFromName(ent.ClassParentName)
			}
			uniq, err := helper.EnsureUniqueSlugCI(
				c.Context(), tx,
				"class_parents", "class_parent_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where(
						"class_parent_school_id = ? AND class_parent_id <> ? AND class_parent_deleted_at IS NULL",
						ent.ClassParentSchoolID, ent.ClassParentID,
					)
				},
				100,
			)
			if err != nil {
				_ = tx.Rollback()
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
			}
			ent.ClassParentSlug = &uniq
		}
	} else if p.ClassParentName.Present && ent.ClassParentName != "" {
		base := helper.Slugify(ent.ClassParentName, 100)
		if base == "" {
			base = "item"
		}
		uniq, err := helper.EnsureUniqueSlugCI(
			c.Context(), tx,
			"class_parents", "class_parent_slug",
			base,
			func(q *gorm.DB) *gorm.DB {
				return q.Where(
					"class_parent_school_id = ? AND class_parent_id <> ? AND class_parent_deleted_at IS NULL",
					ent.ClassParentSchoolID, ent.ClassParentID,
				)
			},
			100,
		)
		if err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghasilkan slug unik")
		}
		ent.ClassParentSlug = &uniq
	}

	// === Simpan perubahan utama (tanpa image dulu)
	if err := tx.Model(&classModel.ClassParentModel{}).
		Where("class_parent_id = ?", ent.ClassParentID).
		Updates(&ent).Error; err != nil {
		_ = tx.Rollback()
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	// === Upload image (multipart optional) → 2-slot + retensi
	uploadedURL := ""
	movedOld := ""

	if fh, ferr := getImageFormFile(c); ferr == nil && fh != nil {
		svc, e := helperOSS.NewOSSServiceFromEnv("")
		if e != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusServiceUnavailable, "OSS belum terkonfigurasi")
		}

		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()

		// folder: classes/class-parents  (rapi & konsisten)
		url, upErr := helperOSS.UploadImageToOSS(ctx, svc, ent.ClassParentSchoolID, "classes/class-parents", fh)
		if upErr != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, upErr.Error())
		}
		key, kerr := helperOSS.KeyFromPublicURL(url)
		if kerr != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusBadRequest, "Gagal ekstrak object key (image)")
		}

		imap := map[string]any{
			"class_parent_image_url":        url,
			"class_parent_image_object_key": key,
			"class_parent_updated_at":       time.Now(),
		}

		if ent.ClassParentImageURL != nil && *ent.ClassParentImageURL != "" {
			due := time.Now().Add(helperOSS.GetRetentionDuration())
			imap["class_parent_image_url_old"] = ent.ClassParentImageURL
			imap["class_parent_image_object_key_old"] = ent.ClassParentImageObjectKey
			imap["class_parent_image_delete_pending_until"] = &due
			movedOld = strings.TrimSpace(*ent.ClassParentImageURL)
		}

		if err := tx.Model(&classModel.ClassParentModel{}).
			Where("class_parent_id = ?", ent.ClassParentID).
			Updates(imap).Error; err != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan image")
		}

		// sinkron ent untuk response
		ent.ClassParentImageURL = &url
		ent.ClassParentImageObjectKey = &key
		uploadedURL = url
	}

	// === Tentukan perlu REFRESH SNAPSHOT classes (denormalized fields)
	newCode := ""
	if ent.ClassParentCode != nil {
		newCode = *ent.ClassParentCode
	}
	newName := ent.ClassParentName
	newSlug := ent.ClassParentSlug
	newLevel := ent.ClassParentLevel

	needRefresh := false
	if p.ClassParentCode.Present && (strings.TrimSpace(newCode) != strings.TrimSpace(oldCode)) {
		needRefresh = true
	}
	if p.ClassParentName.Present && newName != oldName {
		needRefresh = true
	}
	if p.ClassParentSlug.Present {
		needRefresh = true
	}
	if !needRefresh {
		switch {
		case (oldSlug == nil && newSlug != nil),
			(oldSlug != nil && newSlug == nil):
			needRefresh = true
		case (oldSlug != nil && newSlug != nil) && (*oldSlug != *newSlug):
			needRefresh = true
		}
	}
	if p.ClassParentLevel.Present {
		switch {
		case (oldLevel == nil && newLevel != nil),
			(oldLevel != nil && newLevel == nil):
			needRefresh = true
		case (oldLevel != nil && newLevel != nil) && (*oldLevel != *newLevel):
			needRefresh = true
		}
	}

	if needRefresh {
		// --- 1) Refresh snapshot di tabel classes ---
		type classmodel = classModel.ClassModel
		if err := tx.Model(&classmodel{}).
			Where("class_school_id = ? AND class_class_parent_id = ?", ent.ClassParentSchoolID, ent.ClassParentID).
			Updates(map[string]any{
				"class_class_parent_code_snapshot": func() any {
					if ent.ClassParentCode == nil {
						return gorm.Expr("NULL")
					}
					return *ent.ClassParentCode
				}(),
				"class_class_parent_name_snapshot":  newName,
				"class_class_parent_slug_snapshot":  newSlug,
				"class_class_parent_level_snapshot": newLevel,
				"class_updated_at":                  time.Now(),
			}).Error; err != nil {

			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyegarkan snapshot classes")
		}

		// --- 2) Refresh snapshot di tabel class_sections ---
		execErr := tx.Exec(`
        UPDATE class_sections AS cs
        SET
            class_section_class_parent_id             = $1,
            class_section_class_parent_name_snapshot  = $2,
            class_section_class_parent_slug_snapshot  = $3,
            class_section_class_parent_level_snapshot = $4,
            class_section_snapshot_updated_at         = NOW(),
            class_section_updated_at                  = NOW()
        FROM classes AS c
        WHERE
            cs.class_section_class_id = c.class_id
            AND c.class_class_parent_id = $1
            AND cs.class_section_school_id = $5
    `,
			ent.ClassParentID,
			ent.ClassParentName,
			func() any {
				if ent.ClassParentSlug == nil {
					return nil
				}
				return *ent.ClassParentSlug
			}(),
			func() any {
				if ent.ClassParentLevel == nil {
					return nil
				}
				return *ent.ClassParentLevel
			}(),
			ent.ClassParentSchoolID,
		).Error

		if execErr != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyegarkan snapshot class_sections")
		}

		// --- 3) Refresh snapshot di CSST (class_section_subject_teachers) ---
		// Untuk sekarang: cukup update updated_at berdasarkan section & parent.
		execErr2 := tx.Exec(`
    UPDATE class_section_subject_teachers AS csst
    SET
        class_section_subject_teacher_updated_at = NOW()
    FROM class_sections AS sec
    JOIN classes AS c
      ON sec.class_section_class_id = c.class_id
    WHERE
        csst.class_section_subject_teacher_class_section_id = sec.class_section_id
        AND c.class_class_parent_id = $1
        AND csst.class_section_subject_teacher_school_id = $2
`,
			ent.ClassParentID,
			ent.ClassParentSchoolID,
		).Error

		if execErr2 != nil {
			_ = tx.Rollback()
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyegarkan snapshot CSST")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, err.Error())
	}

	// Response: pakai JsonUpdated, data = single class_parent object
	_ = uploadedURL
	_ = movedOld

	return helper.JsonUpdated(c, "Berhasil memperbarui parent kelas", cpdto.FromModelClassParent(&ent))
}

/*
=========================================================
DELETE (soft delete, staff only) + optional file cleanup
=========================================================
*/
func (ctl *ClassParentController) Delete(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var ent classModel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Guard akses staff pada school terkait
	if err := helperAuth.EnsureStaffSchool(c, ent.ClassParentSchoolID); err != nil {
		return err
	}

	now := time.Now()
	if err := ctl.DB.WithContext(c.Context()).
		Model(&classModel.ClassParentModel{}).
		Where("class_parent_id = ?", ent.ClassParentID).
		Updates(map[string]any{
			"class_parent_deleted_at": &now,
			"class_parent_updated_at": now,
		}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	// OPSIONAL: soft-delete file terkait jika image_url disertakan
	imageURL := strings.TrimSpace(c.Query("image_url"))
	if imageURL == "" {
		if v := strings.TrimSpace(c.FormValue("image_url")); v != "" {
			imageURL = v
		}
	}
	if imageURL != "" {
		if _, err := helperOSS.MoveToSpamByPublicURLENV(imageURL, 0); err != nil {
			// optional: log.Print(err)
		}
		// Kalau mau hard-delete:
		// _, _ = helperOSS.DeleteByPublicURLENV(imageURL, 15*time.Second)
	}

	return helper.JsonDeleted(c, "Berhasil menghapus parent kelas", fiber.Map{
		"class_parent_id": ent.ClassParentID,
	})
}
