// file: internals/features/lembaga/schools/controller/school_service_plan_controller.go
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	dto "schoolku_backend/internals/features/lembaga/school_yayasans/schools/dto"
	mModel "schoolku_backend/internals/features/lembaga/school_yayasans/schools/model"
	helper "schoolku_backend/internals/helpers"
	helperAuth "schoolku_backend/internals/helpers/auth"
	helperOSS "schoolku_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SchoolServicePlanController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewSchoolServicePlanController(db *gorm.DB, v interface{ Struct(any) error }) *SchoolServicePlanController {
	return &SchoolServicePlanController{DB: db, Validator: v}
}

/* ============================== Util ============================== */

func clampLimit(v, def, max int) int {
	if v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func pickImageFile(c *fiber.Ctx, names ...string) *multipart.FileHeader {
	for _, n := range names {
		if fh, err := c.FormFile(n); err == nil && fh != nil {
			return fh
		}
	}
	return nil
}

/* ============================== CREATE (owner only) ============================== */
/* ============================== CREATE (owner only) ============================== */

func (ctl *SchoolServicePlanController) Create(c *fiber.Ctx) error {
	// 1. Auth: hanya owner
	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang diizinkan")
	}

	// 2. Parse body
	var p dto.CreateSchoolServicePlanRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(&p); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// 3. Precheck unik code (case-insensitive)
	code := strings.TrimSpace(p.SchoolServicePlanCode)
	if code != "" {
		var cnt int64
		if err := ctl.DB.Model(&mModel.SchoolServicePlan{}).
			Where("LOWER(school_service_plan_code) = LOWER(?) AND school_service_plan_deleted_at IS NULL", code).
			Count(&cnt).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal cek kode unik")
		}
		if cnt > 0 {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan")
		}
	}

	// 4. Build entity & insert
	ent := p.ToModel()
	if err := ctl.DB.WithContext(c.Context()).Create(ent).Error; err != nil {
		if dto.IsUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	// 5. Optional upload file (image)
	uploadedURL := ""
	if fh := pickImageFile(c, "image", "file"); fh != nil {
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "OSS init error: "+er.Error())
		}

		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()

		const keyPrefix = "service-plans"
		url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Upload gagal: "+upErr.Error())
		}
		uploadedURL = url

		// derive object key
		objKey := ""
		if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
			objKey = k
		} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
			objKey = k2
		}

		// update DB
		if err := ctl.DB.WithContext(c.Context()).
			Model(&mModel.SchoolServicePlan{}).
			Where("school_service_plan_id = ?", ent.SchoolServicePlanID).
			Updates(map[string]any{
				"school_service_plan_image_url":        uploadedURL,
				"school_service_plan_image_object_key": objKey,
			}).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan URL gambar: "+err.Error())
		}

		// sinkronkan ent
		ent.SchoolServicePlanImageURL = &uploadedURL
		if objKey != "" {
			ent.SchoolServicePlanImageObjectKey = &objKey
		}
	}

	// 6. Reload entity untuk response final
	_ = ctl.DB.WithContext(c.Context()).
		First(ent, "school_service_plan_id = ?", ent.SchoolServicePlanID).Error

	// 7. Response
	return helper.JsonCreated(c, "Berhasil membuat service plan", fiber.Map{
		"service_plan":       dto.NewSchoolServicePlanResponse(ent),
		"uploaded_image_url": uploadedURL,
	})
}

/* ============================== PATCH (owner only) ============================== */

func (ctl *SchoolServicePlanController) Patch(c *fiber.Ctx) error {
	// 1) Auth
	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang diizinkan")
	}

	// 2) Param id
	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	// 3) Load entity
	var ent mModel.SchoolServicePlan
	if err := ctl.DB.WithContext(c.Context()).
		Where("school_service_plan_id = ? AND school_service_plan_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// 4) Parse payload (JSON / multipart)
	var p dto.UpdateSchoolServicePlanRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if raw := strings.TrimSpace(c.FormValue("json_body")); raw != "" {
		_ = json.Unmarshal([]byte(raw), &p) // best-effort
	}

	// 5) Terapkan patch ke entity (image pair + retensi 30 hari)
	if err := p.ApplyToModelWithImageSwap(&ent, 30*24*time.Hour); err != nil {
		if err == dto.ErrImagePairMismatch {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
		return helper.JsonError(c, fiber.StatusBadRequest, "Patch tidak valid: "+err.Error())
	}

	// 6) Simpan perubahan non-file
	if err := ctl.DB.WithContext(c.Context()).
		Model(&mModel.SchoolServicePlan{}).
		Where("school_service_plan_id = ?", ent.SchoolServicePlanID).
		Updates(&ent).Error; err != nil {
		if dto.IsUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan perubahan")
	}

	// 7) Optional: upload file → override kolom image hasil JSON patch
	uploadedURL := ""
	movedOld := ""

	if fh := pickImageFile(c, "image", "file"); fh != nil {
		svc, er := helperOSS.NewOSSServiceFromEnv("")
		if er != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "OSS init error: "+er.Error())
		}

		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()

		const keyPrefix = "service-plans"
		url, upErr := svc.UploadAsWebP(ctx, fh, keyPrefix)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Upload gagal: "+upErr.Error())
		}
		uploadedURL = url

		// object key baru
		newObjKey := ""
		if k, e := helperOSS.ExtractKeyFromPublicURL(uploadedURL); e == nil {
			newObjKey = k
		} else if k2, e2 := helperOSS.KeyFromPublicURL(uploadedURL); e2 == nil {
			newObjKey = k2
		}

		// data lama (post-patch) → pindah ke spam (soft-delete)
		oldURL := ""
		if ent.SchoolServicePlanImageURL != nil {
			oldURL = *ent.SchoolServicePlanImageURL
		}
		oldObjKey := ""
		if ent.SchoolServicePlanImageObjectKey != nil {
			oldObjKey = *ent.SchoolServicePlanImageObjectKey
		}

		movedURL := ""
		if strings.TrimSpace(oldURL) != "" {
			if mv, mvErr := helperOSS.MoveToSpamByPublicURLENV(oldURL, 0); mvErr == nil {
				movedURL = mv
				movedOld = mv
				// sinkronkan key lama sesuai lokasi baru
				if k, e := helperOSS.ExtractKeyFromPublicURL(movedURL); e == nil {
					oldObjKey = k
				} else if k2, e2 := helperOSS.KeyFromPublicURL(movedURL); e2 == nil {
					oldObjKey = k2
				}
			}
		}

		deletePendingUntil := time.Now().Add(30 * 24 * time.Hour)

		// tulis override ke DB
		if err := ctl.DB.WithContext(c.Context()).
			Model(&mModel.SchoolServicePlan{}).
			Where("school_service_plan_id = ?", ent.SchoolServicePlanID).
			Updates(map[string]any{
				"school_service_plan_image_url":        uploadedURL,
				"school_service_plan_image_object_key": newObjKey,
				"school_service_plan_image_url_old": func() any {
					if movedURL == "" {
						return gorm.Expr("NULL")
					}
					return movedURL
				}(),
				"school_service_plan_image_object_key_old": func() any {
					if oldObjKey == "" {
						return gorm.Expr("NULL")
					}
					return oldObjKey
				}(),
				"school_service_plan_image_delete_pending_until": deletePendingUntil,
				"school_service_plan_updated_at":                 time.Now(),
			}).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan gambar: "+err.Error())
		}

		// sinkronkan ent untuk response
		ent.SchoolServicePlanImageURL = &uploadedURL
		if newObjKey != "" {
			ent.SchoolServicePlanImageObjectKey = &newObjKey
		} else {
			ent.SchoolServicePlanImageObjectKey = nil
		}
		if movedURL != "" {
			ent.SchoolServicePlanImageURLOld = &movedURL
		} else {
			ent.SchoolServicePlanImageURLOld = nil
		}
		if oldObjKey != "" {
			ent.SchoolServicePlanImageObjectKeyOld = &oldObjKey
		} else {
			ent.SchoolServicePlanImageObjectKeyOld = nil
		}
		t := deletePendingUntil
		ent.SchoolServicePlanImageDeletePendingUntil = &t
	}

	// 8) Reload best-effort
	_ = ctl.DB.WithContext(c.Context()).
		First(&ent, "school_service_plan_id = ?", ent.SchoolServicePlanID).Error

	// 9) Response
	return helper.JsonOK(c, "Berhasil memperbarui service plan", fiber.Map{
		"service_plan":        dto.NewSchoolServicePlanResponse(&ent),
		"uploaded_image_url":  uploadedURL,
		"moved_old_image_url": movedOld,
	})
}

/* ============================== DELETE (owner only) ============================== */

func (ctl *SchoolServicePlanController) Delete(c *fiber.Ctx) error {
	// auth: hanya owner
	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang diizinkan")
	}

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var ent mModel.SchoolServicePlan
	if err := ctl.DB.WithContext(c.Context()).
		Where("school_service_plan_id = ? AND school_service_plan_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	now := time.Now()
	if err := ctl.DB.WithContext(c.Context()).
		Model(&mModel.SchoolServicePlan{}).
		Where("school_service_plan_id = ?", ent.SchoolServicePlanID).
		Updates(map[string]any{
			"school_service_plan_deleted_at": &now,
			"school_service_plan_updated_at": now,
		}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	imageURL := strings.TrimSpace(c.Query("image_url"))
	if imageURL == "" {
		if v := strings.TrimSpace(c.FormValue("image_url")); v != "" {
			imageURL = v
		}
	}
	if imageURL != "" {
		_, _ = helperOSS.MoveToSpamByPublicURLENV(imageURL, 0)
	}

	return helper.JsonOK(c, "Berhasil menghapus service plan", fiber.Map{"school_service_plan_id": ent.SchoolServicePlanID})
}

/* ============================== OPTIONAL: validate code (owner only?) ============================== */

func (ctl *SchoolServicePlanController) ValidateCode(c *fiber.Ctx) error {
	// Jika mau batasi, aktifkan owner check:
	// if !helperAuth.IsOwner(c) { return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang diizinkan") }

	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter code wajib diisi")
	}
	var cnt int64
	if err := ctl.DB.WithContext(c.Context()).
		Model(&mModel.SchoolServicePlan{}).
		Where("LOWER(school_service_plan_code) = LOWER(?) AND school_service_plan_deleted_at IS NULL", code).
		Count(&cnt).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}
	return helper.JsonOK(c, "OK", fiber.Map{
		"code":     code,
		"is_taken": cnt > 0,
		"suggestion": func() string {
			if cnt == 0 {
				return code
			}
			return fmt.Sprintf("%s-%d", helper.Slugify(code, 30), time.Now().Unix()%1000)
		}(),
	})
}
