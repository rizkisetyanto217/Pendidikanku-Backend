// file: internals/features/announcements/announcement_urls/controller/announcement_url_controller.go
package controller

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"masjidku_backend/internals/features/school/announcements/announcement/dto"
	"masjidku_backend/internals/features/school/announcements/announcement/model"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AnnouncementURLController struct {
	DB        *gorm.DB
	validator *validator.Validate
}

func NewAnnouncementURLController(db *gorm.DB) *AnnouncementURLController {
	return &AnnouncementURLController{
		DB:        db,
		validator: validator.New(),
	}
}

/* =========================================================
   CREATE
   POST /api/a/announcement-urls
   Body: CreateAnnouncementURLRequest
   - MasjidID dipaksa dari token (tenant-safe)
========================================================= */
// POST /api/a/announcement-urls
func (ctl *AnnouncementURLController) Create(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	// ============= MULTIPART MODE =============
	if isMultipart {
		form, ferr := c.MultipartForm()
		if ferr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Form-data tidak valid")
		}

		// announcement_id wajib
		annIDStr := strings.TrimSpace(c.FormValue("announcement_url_announcement_id"))
		if annIDStr == "" {
			return fiber.NewError(fiber.StatusBadRequest, "announcement_url_announcement_id wajib diisi")
		}
		announcementID, perr := uuid.Parse(annIDStr)
		if perr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "announcement_url_announcement_id tidak valid (uuid)")
		}

		// Siapkan OSS sekali
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			return fiber.NewError(fiber.StatusBadGateway, fmt.Sprintf("OSS init: %v", err))
		}
		dir := fmt.Sprintf("masjids/%s/announcement-urls", masjidID.String())

		// Prefer batch: files[]
		files := form.File["announcement_url_href"]
		if len(files) > 0 {
			// Pilihan label:
			// - satu label global
			// - atau labels sejajar: announcement_url_labels[]
			var (
				globalLabel *string
				labels      = form.Value["announcement_url_labels"]
			)
			if gl := strings.TrimSpace(c.FormValue("announcement_url_label")); gl != "" {
				l := gl
				globalLabel = &l
			}

			results := make([]dto.AnnouncementURLResponse, 0, len(files))
			for i, fh := range files {
				if fh == nil || fh.Size == 0 {
					return fiber.NewError(fiber.StatusBadRequest, "Salah satu file kosong")
				}

				publicURL, upErr := svc.UploadAsWebP(c.Context(), fh, dir)
				if upErr != nil {
					low := strings.ToLower(upErr.Error())
					if strings.Contains(low, "format tidak didukung") {
						return fiber.NewError(fiber.StatusUnsupportedMediaType, "Unsupported image format (pakai jpg/png/webp)")
					}
					return fiber.NewError(fiber.StatusBadGateway, "Gagal upload file")
				}

				var label *string
				if i < len(labels) && strings.TrimSpace(labels[i]) != "" {
					v := strings.TrimSpace(labels[i])
					label = &v
				} else if globalLabel != nil {
					label = globalLabel
				}

				mdl := model.AnnouncementURLModel{
					AnnouncementURLMasjidID:       masjidID,
					AnnouncementURLAnnouncementID: announcementID,
					AnnouncementURLLabel:          label,
					AnnouncementURLHref:           publicURL,
				}
				if err := ctl.DB.WithContext(c.Context()).Create(&mdl).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan data")
				}

				results = append(results, dto.AnnouncementURLResponse{
					AnnouncementURLID:                 mdl.AnnouncementURLID,
					AnnouncementURLMasjidID:           mdl.AnnouncementURLMasjidID,
					AnnouncementURLAnnouncementID:     mdl.AnnouncementURLAnnouncementID,
					AnnouncementURLLabel:              mdl.AnnouncementURLLabel,
					AnnouncementURLHref:               mdl.AnnouncementURLHref,
					AnnouncementURLTrashURL:           mdl.AnnouncementURLTrashURL,
					AnnouncementURLDeletePendingUntil: mdl.AnnouncementURLDeletePendingUntil,
					AnnouncementURLCreatedAt:          mdl.AnnouncementURLCreatedAt,
					AnnouncementURLUpdatedAt:          mdl.AnnouncementURLUpdatedAt,
					AnnouncementURLDeletedAt:          mdl.AnnouncementURLDeletedAt,
				})
			}

			return c.Status(fiber.StatusCreated).JSON(fiber.Map{
				"message": "Berhasil upload banyak announcement URL",
				"data":    results, // array
			})
		}

		// Fallback single: file
		if fh, ferr := c.FormFile("file"); ferr == nil && fh != nil && fh.Size > 0 {
			publicURL, upErr := svc.UploadAsWebP(c.Context(), fh, dir)
			if upErr != nil {
				low := strings.ToLower(upErr.Error())
				if strings.Contains(low, "format tidak didukung") {
					return fiber.NewError(fiber.StatusUnsupportedMediaType, "Unsupported image format (pakai jpg/png/webp)")
				}
				return fiber.NewError(fiber.StatusBadGateway, "Gagal upload file")
			}

			var label *string
			if lbl := strings.TrimSpace(c.FormValue("announcement_url_label")); lbl != "" {
				l := lbl
				label = &l
			}

			mdl := model.AnnouncementURLModel{
				AnnouncementURLMasjidID:       masjidID,
				AnnouncementURLAnnouncementID: announcementID,
				AnnouncementURLLabel:          label,
				AnnouncementURLHref:           publicURL,
			}
			if err := ctl.DB.WithContext(c.Context()).Create(&mdl).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan data")
			}

			resp := dto.AnnouncementURLResponse{
				AnnouncementURLID:                 mdl.AnnouncementURLID,
				AnnouncementURLMasjidID:           mdl.AnnouncementURLMasjidID,
				AnnouncementURLAnnouncementID:     mdl.AnnouncementURLAnnouncementID,
				AnnouncementURLLabel:              mdl.AnnouncementURLLabel,
				AnnouncementURLHref:               mdl.AnnouncementURLHref,
				AnnouncementURLTrashURL:           mdl.AnnouncementURLTrashURL,
				AnnouncementURLDeletePendingUntil: mdl.AnnouncementURLDeletePendingUntil,
				AnnouncementURLCreatedAt:          mdl.AnnouncementURLCreatedAt,
				AnnouncementURLUpdatedAt:          mdl.AnnouncementURLUpdatedAt,
				AnnouncementURLDeletedAt:          mdl.AnnouncementURLDeletedAt,
			}
			return c.Status(fiber.StatusCreated).JSON(fiber.Map{
				"message": "Berhasil membuat announcement URL",
				"data":    resp, // object
			})
		}

		// Terakhir: tanpa file → pakai href manual (single)
		href := strings.TrimSpace(c.FormValue("announcement_url_href"))
		if href == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Harus unggah file (file/files[]) atau isi announcement_url_href")
		}
		var label *string
		if lbl := strings.TrimSpace(c.FormValue("announcement_url_label")); lbl != "" {
			l := lbl
			label = &l
		}

		mdl := model.AnnouncementURLModel{
			AnnouncementURLMasjidID:       masjidID,
			AnnouncementURLAnnouncementID: announcementID,
			AnnouncementURLLabel:          label,
			AnnouncementURLHref:           href,
		}
		if err := ctl.DB.WithContext(c.Context()).Create(&mdl).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan data")
		}
		resp := dto.AnnouncementURLResponse{
			AnnouncementURLID:                 mdl.AnnouncementURLID,
			AnnouncementURLMasjidID:           mdl.AnnouncementURLMasjidID,
			AnnouncementURLAnnouncementID:     mdl.AnnouncementURLAnnouncementID,
			AnnouncementURLLabel:              mdl.AnnouncementURLLabel,
			AnnouncementURLHref:               mdl.AnnouncementURLHref,
			AnnouncementURLTrashURL:           mdl.AnnouncementURLTrashURL,
			AnnouncementURLDeletePendingUntil: mdl.AnnouncementURLDeletePendingUntil,
			AnnouncementURLCreatedAt:          mdl.AnnouncementURLCreatedAt,
			AnnouncementURLUpdatedAt:          mdl.AnnouncementURLUpdatedAt,
			AnnouncementURLDeletedAt:          mdl.AnnouncementURLDeletedAt,
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "Berhasil membuat announcement URL",
			"data":    resp, // object
		})
	}

	// ============= JSON MODE (single) =============
	var req dto.CreateAnnouncementURLRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.validator.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.AnnouncementURLLabel != nil {
		lbl := strings.TrimSpace(*req.AnnouncementURLLabel)
		req.AnnouncementURLLabel = &lbl
	}
	href := strings.TrimSpace(req.AnnouncementURLHref)
	if href == "" {
		return fiber.NewError(fiber.StatusBadRequest, "URL tidak boleh kosong")
	}

	mdl := model.AnnouncementURLModel{
		AnnouncementURLMasjidID:       masjidID,
		AnnouncementURLAnnouncementID: req.AnnouncementURLAnnouncementID,
		AnnouncementURLLabel:          req.AnnouncementURLLabel,
		AnnouncementURLHref:           href,
	}
	if err := ctl.DB.WithContext(c.Context()).Create(&mdl).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan data")
	}
	resp := dto.AnnouncementURLResponse{
		AnnouncementURLID:                 mdl.AnnouncementURLID,
		AnnouncementURLMasjidID:           mdl.AnnouncementURLMasjidID,
		AnnouncementURLAnnouncementID:     mdl.AnnouncementURLAnnouncementID,
		AnnouncementURLLabel:              mdl.AnnouncementURLLabel,
		AnnouncementURLHref:               mdl.AnnouncementURLHref,
		AnnouncementURLTrashURL:           mdl.AnnouncementURLTrashURL,
		AnnouncementURLDeletePendingUntil: mdl.AnnouncementURLDeletePendingUntil,
		AnnouncementURLCreatedAt:          mdl.AnnouncementURLCreatedAt,
		AnnouncementURLUpdatedAt:          mdl.AnnouncementURLUpdatedAt,
		AnnouncementURLDeletedAt:          mdl.AnnouncementURLDeletedAt,
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Berhasil membuat announcement URL",
		"data":    resp, // object
	})
}

/* =========================================================
   LIST
   GET /api/a/announcement-urls?announcement_id=...&q=...&with_deleted=false
========================================================= */
func (ctl *AnnouncementURLController) List(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	q := strings.TrimSpace(c.Query("q"))
	withDeleted := strings.EqualFold(c.Query("with_deleted"), "true")

	var items []model.AnnouncementURLModel
	tx := ctl.DB.WithContext(c.Context()).
		Where("announcement_url_masjid_id = ?", masjidID)

	if annIDStr := strings.TrimSpace(c.Query("announcement_id")); annIDStr != "" {
		annID, perr := uuid.Parse(annIDStr)
		if perr == nil {
			tx = tx.Where("announcement_url_announcement_id = ?", annID)
		}
	}

	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where(
			"(announcement_url_label ILIKE ? OR announcement_url_href ILIKE ?)",
			like, like,
		)
	}

	if !withDeleted {
		tx = tx.Where("announcement_url_deleted_at IS NULL")
	}

	if err := tx.Order("announcement_url_created_at DESC").
		Find(&items).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// map ke response
	out := make([]dto.AnnouncementURLResponse, 0, len(items))
	for _, it := range items {
		out = append(out, dto.AnnouncementURLResponse{
			AnnouncementURLID:               it.AnnouncementURLID,
			AnnouncementURLMasjidID:         it.AnnouncementURLMasjidID,
			AnnouncementURLAnnouncementID:   it.AnnouncementURLAnnouncementID,
			AnnouncementURLLabel:            it.AnnouncementURLLabel,
			AnnouncementURLHref:             it.AnnouncementURLHref,
			AnnouncementURLTrashURL:         it.AnnouncementURLTrashURL,
			AnnouncementURLDeletePendingUntil: it.AnnouncementURLDeletePendingUntil,
			AnnouncementURLCreatedAt:        it.AnnouncementURLCreatedAt,
			AnnouncementURLUpdatedAt:        it.AnnouncementURLUpdatedAt,
			AnnouncementURLDeletedAt:        it.AnnouncementURLDeletedAt,
		})
	}

	return c.JSON(fiber.Map{
		"data": out,
	})
}

/* =========================================================
   DETAIL
   GET /api/a/announcement-urls/:id
========================================================= */
func (ctl *AnnouncementURLController) Detail(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	idStr := c.Params("id")
	id, perr := uuid.Parse(idStr)
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl model.AnnouncementURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("announcement_url_id = ? AND announcement_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := dto.AnnouncementURLResponse{
		AnnouncementURLID:               mdl.AnnouncementURLID,
		AnnouncementURLMasjidID:         mdl.AnnouncementURLMasjidID,
		AnnouncementURLAnnouncementID:   mdl.AnnouncementURLAnnouncementID,
		AnnouncementURLLabel:            mdl.AnnouncementURLLabel,
		AnnouncementURLHref:             mdl.AnnouncementURLHref,
		AnnouncementURLTrashURL:         mdl.AnnouncementURLTrashURL,
		AnnouncementURLDeletePendingUntil: mdl.AnnouncementURLDeletePendingUntil,
		AnnouncementURLCreatedAt:        mdl.AnnouncementURLCreatedAt,
		AnnouncementURLUpdatedAt:        mdl.AnnouncementURLUpdatedAt,
		AnnouncementURLDeletedAt:        mdl.AnnouncementURLDeletedAt,
	}

	return c.JSON(fiber.Map{"data": resp})
}

/* =========================================================
   UPDATE (partial)
   PATCH /api/a/announcement-urls/:id
   Body: UpdateAnnouncementURLRequest
========================================================= */
/* =========================================================
   UPDATE (JSON or MULTIPART, partial + file rotate)
   PATCH /api/a/announcement-urls/:id
========================================================= */
func (ctl *AnnouncementURLController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	id, perr := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl model.AnnouncementURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("announcement_url_id = ? AND announcement_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	ct := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	isMultipart := strings.HasPrefix(ct, fiber.MIMEMultipartForm)

	if isMultipart {
		// ====== MULTIPART ======
		// Label (opsional)
		if lbl := strings.TrimSpace(c.FormValue("announcement_url_label")); lbl != "" {
			mdl.AnnouncementURLLabel = &lbl
		}

		// AnnouncementID (opsional)
		if annIDStr := strings.TrimSpace(c.FormValue("announcement_url_announcement_id")); annIDStr != "" {
			if annID, e := uuid.Parse(annIDStr); e == nil {
				mdl.AnnouncementURLAnnouncementID = annID
			}
		}

		// Jika ada file baru → upload + rotasi file lama ke spam/
		if fh, ferr := c.FormFile("announcement_url_href"); ferr == nil && fh != nil && fh.Size > 0 {
			// Upload baru
			svc, err := helperOSS.NewOSSServiceFromEnv("")
			if err != nil {
				return fiber.NewError(fiber.StatusBadGateway, fmt.Sprintf("OSS init: %v", err))
			}
			dir := fmt.Sprintf("masjids/%s/announcement-urls", masjidID.String())

			newURL, upErr := svc.UploadAsWebP(c.Context(), fh, dir)
			if upErr != nil {
				low := strings.ToLower(upErr.Error())
				if strings.Contains(low, "format tidak didukung") {
					return fiber.NewError(fiber.StatusUnsupportedMediaType, "Unsupported image format (pakai jpg/png/webp)")
				}
				return fiber.NewError(fiber.StatusBadGateway, "Gagal upload file")
			}

			// Pindahkan file lama (jika ada) ke spam/ dan simpan URL spam di trash_url
			if oldURL := strings.TrimSpace(mdl.AnnouncementURLHref); oldURL != "" {
				if spamURL, mvErr := helperOSS.MoveToSpamByPublicURLENV(oldURL, 15*time.Second); mvErr == nil {
					mdl.AnnouncementURLTrashURL = &spamURL
					due := time.Now().Add(7 * 24 * time.Hour)
					mdl.AnnouncementURLDeletePendingUntil = &due
				} else {
					// Jika gagal memindahkan, tetap lanjut: simpan oldURL sebagai trash_url dan jadwalkan hapus
					mdl.AnnouncementURLTrashURL = &oldURL
					due := time.Now().Add(7 * 24 * time.Hour)
					mdl.AnnouncementURLDeletePendingUntil = &due
				}
			}

			// Set href ke URL baru
			mdl.AnnouncementURLHref = newURL
		} else {
			// Ambil seluruh form supaya bisa cek "apakah field dikirim?"
			form, _ := c.MultipartForm()

			// Tanpa file → boleh update href manual (opsional)
			if h := strings.TrimSpace(c.FormValue("announcement_url_href")); h != "" {
				mdl.AnnouncementURLHref = h
			}

			// Optional: trash_url manual
			if form != nil {
				if vals, exists := form.Value["announcement_url_trash_url"]; exists {
					// Field dikirim (meski kosong)
					if len(vals) == 0 || strings.TrimSpace(vals[0]) == "" {
						// jika dikirim namun kosong → null-kan
						mdl.AnnouncementURLTrashURL = nil
					} else {
						tr := strings.TrimSpace(vals[0])
						mdl.AnnouncementURLTrashURL = &tr
					}
				}
			}

			// Optional: delete_pending_until manual (RFC3339)
			if d := strings.TrimSpace(c.FormValue("announcement_url_delete_pending_until")); d != "" {
				if t, e := time.Parse(time.RFC3339, d); e == nil {
					mdl.AnnouncementURLDeletePendingUntil = &t
				}
			}
		}
	} else {
		// ====== JSON ======
		var req dto.UpdateAnnouncementURLRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
		}
		if err := ctl.validator.Struct(req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		if req.AnnouncementURLLabel != nil {
			lbl := strings.TrimSpace(*req.AnnouncementURLLabel)
			mdl.AnnouncementURLLabel = &lbl
		}
		if req.AnnouncementURLHref != nil {
			h := strings.TrimSpace(*req.AnnouncementURLHref)
			if h == "" {
				return fiber.NewError(fiber.StatusBadRequest, "URL tidak boleh kosong")
			}
			mdl.AnnouncementURLHref = h
		}
		if req.AnnouncementURLTrashURL != nil {
			tr := strings.TrimSpace(*req.AnnouncementURLTrashURL)
			if tr == "" {
				mdl.AnnouncementURLTrashURL = nil
			} else {
				mdl.AnnouncementURLTrashURL = &tr
			}
		}
		if req.AnnouncementURLDeletePendingUntil != nil {
			mdl.AnnouncementURLDeletePendingUntil = req.AnnouncementURLDeletePendingUntil
		}
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&mdl).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	resp := dto.AnnouncementURLResponse{
		AnnouncementURLID:                 mdl.AnnouncementURLID,
		AnnouncementURLMasjidID:           mdl.AnnouncementURLMasjidID,
		AnnouncementURLAnnouncementID:     mdl.AnnouncementURLAnnouncementID,
		AnnouncementURLLabel:              mdl.AnnouncementURLLabel,
		AnnouncementURLHref:               mdl.AnnouncementURLHref,
		AnnouncementURLTrashURL:           mdl.AnnouncementURLTrashURL,
		AnnouncementURLDeletePendingUntil: mdl.AnnouncementURLDeletePendingUntil,
		AnnouncementURLCreatedAt:          mdl.AnnouncementURLCreatedAt,
		AnnouncementURLUpdatedAt:          mdl.AnnouncementURLUpdatedAt,
		AnnouncementURLDeletedAt:          mdl.AnnouncementURLDeletedAt,
	}
	return c.JSON(fiber.Map{
		"message": "Berhasil memperbarui",
		"data":    resp,
	})
}

/* =========================================================
   DELETE (soft)
   DELETE /api/a/announcement-urls/:id
========================================================= */
func (ctl *AnnouncementURLController) Delete(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	idStr := c.Params("id")
	id, perr := uuid.Parse(idStr)
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl model.AnnouncementURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("announcement_url_id = ? AND announcement_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	now := time.Now()
	mdl.AnnouncementURLDeletedAt = &now

	if err := ctl.DB.WithContext(c.Context()).Save(&mdl).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return c.JSON(fiber.Map{
		"message": "Berhasil menghapus",
	})
}

/* =========================================================
   RESTORE (opsional)
   PATCH /api/a/announcement-urls/:id/restore
========================================================= */
func (ctl *AnnouncementURLController) Restore(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromToken(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	idStr := c.Params("id")
	id, perr := uuid.Parse(idStr)
	if perr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var mdl model.AnnouncementURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("announcement_url_id = ? AND announcement_url_masjid_id = ?", id, masjidID).
		First(&mdl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	mdl.AnnouncementURLDeletedAt = nil
	if err := ctl.DB.WithContext(c.Context()).Save(&mdl).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal restore data")
	}

	return c.JSON(fiber.Map{"message": "Berhasil restore"})
}
