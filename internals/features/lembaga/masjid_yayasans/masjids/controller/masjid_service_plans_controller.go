// file: internals/features/lembaga/masjids/controller/masjid_service_plan_controller.go
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	dto "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/dto"
	mModel "masjidku_backend/internals/features/lembaga/masjid_yayasans/masjids/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MasjidServicePlanController struct {
	DB        *gorm.DB
	Validator interface{ Struct(any) error }
}

func NewMasjidServicePlanController(db *gorm.DB, v interface{ Struct(any) error }) *MasjidServicePlanController {
	return &MasjidServicePlanController{DB: db, Validator: v}
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

func (ctl *MasjidServicePlanController) Create(c *fiber.Ctx) error {
	// 1. Auth: hanya owner
	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang diizinkan")
	}

	// 2. Parse body
	var p dto.CreateMasjidServicePlanRequest
	if err := c.BodyParser(&p); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid: "+err.Error())
	}
	if ctl.Validator != nil {
		if err := ctl.Validator.Struct(&p); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// 3. Precheck unik code (case-insensitive)
	code := strings.TrimSpace(p.MasjidServicePlanCode)
	if code != "" {
		var cnt int64
		if err := ctl.DB.Model(&mModel.MasjidServicePlan{}).
			Where("LOWER(masjid_service_plan_code) = LOWER(?) AND masjid_service_plan_deleted_at IS NULL", code).
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
			Model(&mModel.MasjidServicePlan{}).
			Where("masjid_service_plan_id = ?", ent.MasjidServicePlanID).
			Updates(map[string]any{
				"masjid_service_plan_image_url":        uploadedURL,
				"masjid_service_plan_image_object_key": objKey,
			}).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan URL gambar: "+err.Error())
		}

		// sinkronkan ent
		ent.MasjidServicePlanImageURL = &uploadedURL
		if objKey != "" {
			ent.MasjidServicePlanImageObjectKey = &objKey
		}
	}

	// 6. Reload entity untuk response final
	_ = ctl.DB.WithContext(c.Context()).
		First(ent, "masjid_service_plan_id = ?", ent.MasjidServicePlanID).Error

	// 7. Response
	return helper.JsonCreated(c, "Berhasil membuat service plan", fiber.Map{
		"service_plan":       dto.NewMasjidServicePlanResponse(ent),
		"uploaded_image_url": uploadedURL,
	})
}

/* ============================== PATCH (owner only) ============================== */

func (ctl *MasjidServicePlanController) Patch(c *fiber.Ctx) error {
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
	var ent mModel.MasjidServicePlan
	if err := ctl.DB.WithContext(c.Context()).
		Where("masjid_service_plan_id = ? AND masjid_service_plan_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// 4) Parse payload (JSON / multipart)
	var p dto.UpdateMasjidServicePlanRequest
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
		Model(&mModel.MasjidServicePlan{}).
		Where("masjid_service_plan_id = ?", ent.MasjidServicePlanID).
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
		if ent.MasjidServicePlanImageURL != nil {
			oldURL = *ent.MasjidServicePlanImageURL
		}
		oldObjKey := ""
		if ent.MasjidServicePlanImageObjectKey != nil {
			oldObjKey = *ent.MasjidServicePlanImageObjectKey
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
			Model(&mModel.MasjidServicePlan{}).
			Where("masjid_service_plan_id = ?", ent.MasjidServicePlanID).
			Updates(map[string]any{
				"masjid_service_plan_image_url":        uploadedURL,
				"masjid_service_plan_image_object_key": newObjKey,
				"masjid_service_plan_image_url_old": func() any {
					if movedURL == "" {
						return gorm.Expr("NULL")
					}
					return movedURL
				}(),
				"masjid_service_plan_image_object_key_old": func() any {
					if oldObjKey == "" {
						return gorm.Expr("NULL")
					}
					return oldObjKey
				}(),
				"masjid_service_plan_image_delete_pending_until": deletePendingUntil,
				"masjid_service_plan_updated_at":                 time.Now(),
			}).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan gambar: "+err.Error())
		}

		// sinkronkan ent untuk response
		ent.MasjidServicePlanImageURL = &uploadedURL
		if newObjKey != "" {
			ent.MasjidServicePlanImageObjectKey = &newObjKey
		} else {
			ent.MasjidServicePlanImageObjectKey = nil
		}
		if movedURL != "" {
			ent.MasjidServicePlanImageURLOld = &movedURL
		} else {
			ent.MasjidServicePlanImageURLOld = nil
		}
		if oldObjKey != "" {
			ent.MasjidServicePlanImageObjectKeyOld = &oldObjKey
		} else {
			ent.MasjidServicePlanImageObjectKeyOld = nil
		}
		t := deletePendingUntil
		ent.MasjidServicePlanImageDeletePendingUntil = &t
	}

	// 8) Reload best-effort
	_ = ctl.DB.WithContext(c.Context()).
		First(&ent, "masjid_service_plan_id = ?", ent.MasjidServicePlanID).Error

	// 9) Response
	return helper.JsonOK(c, "Berhasil memperbarui service plan", fiber.Map{
		"service_plan":        dto.NewMasjidServicePlanResponse(&ent),
		"uploaded_image_url":  uploadedURL,
		"moved_old_image_url": movedOld,
	})
}

/* ============================== DELETE (owner only) ============================== */

func (ctl *MasjidServicePlanController) Delete(c *fiber.Ctx) error {
	// auth: hanya owner
	if !helperAuth.IsOwner(c) {
		return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang diizinkan")
	}

	idStr := strings.TrimSpace(c.Params("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "id tidak valid")
	}

	var ent mModel.MasjidServicePlan
	if err := ctl.DB.WithContext(c.Context()).
		Where("masjid_service_plan_id = ? AND masjid_service_plan_deleted_at IS NULL", id).
		First(&ent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	now := time.Now()
	if err := ctl.DB.WithContext(c.Context()).
		Model(&mModel.MasjidServicePlan{}).
		Where("masjid_service_plan_id = ?", ent.MasjidServicePlanID).
		Updates(map[string]any{
			"masjid_service_plan_deleted_at": &now,
			"masjid_service_plan_updated_at": now,
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

	return helper.JsonOK(c, "Berhasil menghapus service plan", fiber.Map{"masjid_service_plan_id": ent.MasjidServicePlanID})
}

/* ============================== OPTIONAL: validate code (owner only?) ============================== */

func (ctl *MasjidServicePlanController) ValidateCode(c *fiber.Ctx) error {
	// Jika mau batasi, aktifkan owner check:
	// if !helperAuth.IsOwner(c) { return helper.JsonError(c, fiber.StatusForbidden, "Hanya owner yang diizinkan") }

	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Parameter code wajib diisi")
	}
	var cnt int64
	if err := ctl.DB.WithContext(c.Context()).
		Model(&mModel.MasjidServicePlan{}).
		Where("LOWER(masjid_service_plan_code) = LOWER(?) AND masjid_service_plan_deleted_at IS NULL", code).
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
