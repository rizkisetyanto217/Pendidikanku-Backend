// internals/features/school/class_attendance_result/user_result/user_quran_url/controller/user_quran_url_controller.go
package controller

import (
	"errors"
	"strings"
	"time"

	urlDTO "masjidku_backend/internals/features/school/class_attendance_result/user_result/user_quran/dto"
	urlModel "masjidku_backend/internals/features/school/class_attendance_result/user_result/user_quran/model"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserQuranURLController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewUserQuranURLController(db *gorm.DB) *UserQuranURLController {
	return &UserQuranURLController{
		DB:        db,
		Validator: validator.New(),
	}
}


// Pastikan record parent milik masjid ini (tenant-safe)
func (ctl *UserQuranURLController) ensureRecordBelongsToMasjid(ctx *fiber.Ctx, recordID, masjidID uuid.UUID) error {
	var count int64
	if err := ctl.DB.WithContext(ctx.Context()).
		Model(&urlModel.UserQuranRecordModel{}).
		Where("user_quran_records_id = ? AND user_quran_records_masjid_id = ? AND user_quran_records_deleted_at IS NULL", recordID, masjidID).
		Count(&count).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if count == 0 {
		return fiber.NewError(fiber.StatusForbidden, "Record tidak ditemukan/diizinkan untuk masjid ini")
	}
	return nil
}

// ==== BUILD LIST QUERY ====

func (ctl *UserQuranURLController) buildListQuery(c *fiber.Ctx, q urlDTO.ListUserQuranURLQuery, masjidID uuid.UUID) (*gorm.DB, error) {
	tx := ctl.DB.WithContext(c.Context()).Model(&urlModel.UserQuranURLModel{})

	// Tenant guard via join ke parent
	tx = tx.Joins("JOIN user_quran_records r ON r.user_quran_records_id = user_quran_urls.user_quran_urls_record_id").
		Where("r.user_quran_records_masjid_id = ? AND (r.user_quran_records_deleted_at IS NULL)", masjidID)

	// Filters
	if q.RecordID != nil {
		tx = tx.Where("user_quran_urls_record_id = ?", *q.RecordID)
	}
	if q.TeacherID != nil {
		tx = tx.Where("user_quran_urls_uploader_teacher_id = ?", *q.TeacherID)
	}
	if q.UserID != nil {
		tx = tx.Where("user_quran_urls_uploader_user_id = ?", *q.UserID)
	}
	if q.Q != nil && strings.TrimSpace(*q.Q) != "" {
		needle := "%" + strings.TrimSpace(*q.Q) + "%"
		tx = tx.Where("(user_quran_urls_label ILIKE ? OR user_quran_urls_href ILIKE ?)", needle, needle)
	}
	// created range
	if q.CreatedFrom != nil && strings.TrimSpace(*q.CreatedFrom) != "" {
		t, err := time.Parse(dateLayout, strings.TrimSpace(*q.CreatedFrom))
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "created_from invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_quran_urls_created_at >= ?", t)
	}
	if q.CreatedTo != nil && strings.TrimSpace(*q.CreatedTo) != "" {
		t, err := time.Parse(dateLayout, strings.TrimSpace(*q.CreatedTo))
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "created_to invalid format, expected YYYY-MM-DD")
		}
		tx = tx.Where("user_quran_urls_created_at < ?", t.Add(24*time.Hour))
	}

	// Sorting
	order := "user_quran_urls_created_at DESC"
	if q.Sort != nil {
		switch strings.ToLower(strings.TrimSpace(*q.Sort)) {
		case "created_at_asc":
			order = "user_quran_urls_created_at ASC"
		case "created_at_desc":
			order = "user_quran_urls_created_at DESC"
		}
	}
	tx = tx.Order(order)

	return tx, nil
}

// ===============================
// Handlers
// ===============================

// POST /user-quran-urls (JSON body, href sudah ada)
func (ctl *UserQuranURLController) Create(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	var req urlDTO.CreateUserQuranURLRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	// tenant guard by parent
	if err := ctl.ensureRecordBelongsToMasjid(c, req.UserQuranURLsRecordID, masjidID); err != nil {
		return err
	}

	m := req.ToModel()
	if err := ctl.DB.WithContext(c.Context()).Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": urlDTO.NewUserQuranURLResponse(m)})
}

// POST /user-quran-urls/upload (multipart form: file, record_id, label?, uploader_teacher_id?/uploader_user_id?)
//
// Field form yang diterima:
// - file (wajib)
// - record_id (wajib, UUID)
// - label (opsional)
// - slot (opsional, subdir tambahan; default "default")
// - uploader_teacher_id (opsional UUID) / uploader_user_id (opsional UUID) – salah satu boleh diisi
func (ctl *UserQuranURLController) Upload(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	recordIDStr := strings.TrimSpace(c.FormValue("record_id"))
	if recordIDStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "record_id wajib diisi")
	}
	recordID, err := uuid.Parse(recordIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "record_id tidak valid")
	}
	if err := ctl.ensureRecordBelongsToMasjid(c, recordID, masjidID); err != nil {
		return err
	}

	// uploader (opsional)
	var uploaderTeacherID *uuid.UUID
	var uploaderUserID *uuid.UUID
	if s := strings.TrimSpace(c.FormValue("uploader_teacher_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			uploaderTeacherID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "uploader_teacher_id tidak valid")
		}
	}
	if s := strings.TrimSpace(c.FormValue("uploader_user_id")); s != "" {
		if id, e := uuid.Parse(s); e == nil {
			uploaderUserID = &id
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "uploader_user_id tidak valid")
		}
	}

	// file
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return fiber.NewError(fiber.StatusBadRequest, "file wajib diunggah")
	}
	slot := strings.TrimSpace(c.FormValue("slot"))
	if slot == "" {
		slot = "quran-url"
	}

	// Inisialisasi OSS dari ENV, upload sebagai WebP ke directory masjid/slot
	svc, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "OSS init gagal: "+err.Error())
	}
	ctx := c.Context()
	publicURL, err := helperOSS.UploadImageToOSS(ctx, svc, masjidID, slot, fh)
	if err != nil {
		return err // sudah fiber.Error di helper (size/format/etc)
	}

	// Simpan row
	var label *string
	if l := strings.TrimSpace(c.FormValue("label")); l != "" {
		label = &l
	}
	row := &urlModel.UserQuranURLModel{
		UserQuranURLsRecordID:        recordID,
		UserQuranURLsLabel:           label,
		UserQuranURLsHref:            publicURL,
		UserQuranURLsUploaderTeacherID: uploaderTeacherID,
		UserQuranURLsUploaderUserID:    uploaderUserID,
	}
	if err := ctl.DB.WithContext(c.Context()).Create(row).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": urlDTO.NewUserQuranURLResponse(row)})
}

// GET /user-quran-urls
func (ctl *UserQuranURLController) List(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	var q urlDTO.ListUserQuranURLQuery
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx, err := ctl.buildListQuery(c, q, masjidID)
	if err != nil {
		return err
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	limit := q.Limit
	offset := q.Offset
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var rows []urlModel.UserQuranURLModel
	if err := tx.Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"data":   urlDTO.FromUserQuranURLModels(rows),
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GET /user-quran-urls/:id
func (ctl *UserQuranURLController) GetByID(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var row urlModel.UserQuranURLModel
	// join ke parent untuk tenant guard
	if err := ctl.DB.WithContext(c.Context()).
		Model(&urlModel.UserQuranURLModel{}).
		Joins("JOIN user_quran_records r ON r.user_quran_records_id = user_quran_urls.user_quran_urls_record_id").
		Where("user_quran_urls_id = ? AND r.user_quran_records_masjid_id = ? AND r.user_quran_records_deleted_at IS NULL", id, masjidID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"data": urlDTO.NewUserQuranURLResponse(&row)})
}

// PATCH /user-quran-urls/:id
func (ctl *UserQuranURLController) Update(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// ambil row dengan tenant guard
	var m urlModel.UserQuranURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Model(&urlModel.UserQuranURLModel{}).
		Joins("JOIN user_quran_records r ON r.user_quran_records_id = user_quran_urls.user_quran_urls_record_id").
		Where("user_quran_urls_id = ? AND r.user_quran_records_masjid_id = ? AND r.user_quran_records_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var req urlDTO.UpdateUserQuranURLRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validator.Struct(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	req.ApplyToModel(&m)
	if err := ctl.DB.WithContext(c.Context()).
		Model(&m).
		Select(
			"user_quran_urls_label",
			"user_quran_urls_href",
			"user_quran_urls_trash_url",
			"user_quran_urls_delete_pending_until",
			"user_quran_urls_uploader_teacher_id",
			"user_quran_urls_uploader_user_id",
			"user_quran_urls_updated_at",
		).
		Updates(&m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"data": urlDTO.NewUserQuranURLResponse(&m)})
}

// DELETE /user-quran-urls/:id (soft delete)
func (ctl *UserQuranURLController) Delete(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	// tenant-safe delete
	if err := ctl.DB.WithContext(c.Context()).
		Where("user_quran_urls_id IN (SELECT u.user_quran_urls_id FROM user_quran_urls u JOIN user_quran_records r ON r.user_quran_records_id = u.user_quran_urls_record_id WHERE u.user_quran_urls_id = ? AND r.user_quran_records_masjid_id = ? AND r.user_quran_records_deleted_at IS NULL)", id, masjidID).
		Delete(&urlModel.UserQuranURLModel{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// (Opsional) DELETE /user-quran-urls/:id/oss : hapus object di OSS berdasarkan public URL yang disimpan
// gunakan bila kamu ingin benar-benar menghapus file dari bucket juga.
func (ctl *UserQuranURLController) DeleteFromOSS(c *fiber.Ctx) error {
	masjidID, err := getMasjidIDFromToken(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}
	var row urlModel.UserQuranURLModel
	if err := ctl.DB.WithContext(c.Context()).
		Model(&urlModel.UserQuranURLModel{}).
		Joins("JOIN user_quran_records r ON r.user_quran_records_id = user_quran_urls.user_quran_urls_record_id").
		Where("user_quran_urls_id = ? AND r.user_quran_records_masjid_id = ? AND r.user_quran_records_deleted_at IS NULL", id, masjidID).
		First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// hapus dari OSS menggunakan helper ENV
	if err := helperOSS.DeleteByPublicURLENV(row.UserQuranURLsHref, 20*time.Second); err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "Gagal delete file di OSS: "+err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
