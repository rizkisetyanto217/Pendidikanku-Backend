// file: internals/features/users/profile/controller/users_profile_document_controller.go
package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	pq "github.com/lib/pq"
	"gorm.io/gorm"

	// DTO & Model (users/profile)
	"masjidku_backend/internals/features/users/user_profiles/dto"
	"masjidku_backend/internals/features/users/user_profiles/model"

	// Helpers
	helper "masjidku_backend/internals/helpers"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
)

type UsersProfileDocumentController struct {
	DB        *gorm.DB
	Validator *validator.Validate
}

func NewUsersProfileDocumentController(db *gorm.DB) *UsersProfileDocumentController {
	return &UsersProfileDocumentController{
		DB:        db,
		Validator: validator.New(),
	}
}

/* =========================
   Helpers
========================= */

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

func parseRFC3339Ptr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

/* =========================
   CREATE - MULTIPART (many)
========================= */
// POST /users/profile/documents/upload/many
func (uc *UsersProfileDocumentController) CreateMultipartMany(c *fiber.Ctx) error {
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	form, err := c.MultipartForm()
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Form-data tidak valid")
	}
	files := form.File["files"]
	docTypes := form.Value["doc_type"] // opsional: sejajarkan dengan files

	if len(files) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Tidak ada file yang diunggah (field 'files')")
	}
	if len(docTypes) > 0 && len(docTypes) != len(files) {
		return helper.JsonError(c, fiber.StatusBadRequest, "Jumlah doc_type harus sama dengan jumlah files")
	}

	// Init OSS
	svc, err := helperOSS.NewOSSServiceFromEnv("")
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal init OSS")
	}
	ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
	defer cancel()

	// Transaksi DB
	tx := uc.DB.WithContext(c.Context()).Begin()
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memulai transaksi")
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback().Error
		}
	}()

	out := make([]dto.UserProfileDocumentResponse, 0, len(files))
	baseDir := fmt.Sprintf("users/documents/%s", userID.String())

	for i, fh := range files {
		// Tentukan docType
		var docType string
		if len(docTypes) > 0 {
			docType = strings.TrimSpace(docTypes[i])
		} else {
			ext := strings.ToLower(filepath.Ext(fh.Filename))
			base := strings.TrimSuffix(fh.Filename, ext)
			if base == "" {
				base = "document"
			}
			if len(base) > 50 {
				base = base[:50]
			}
			docType = base
		}
		// Validasi docType
		if err := uc.Validator.Var(docType, "required,max=50"); err != nil {
			_ = tx.Rollback().Error
			return helper.JsonError(c, fiber.StatusBadRequest, fmt.Sprintf("doc_type[%d] tidak valid", i))
		}

		// Upload → webp
		url, upErr := svc.UploadAsWebP(ctx, fh, baseDir)
		if upErr != nil {
			_ = tx.Rollback().Error
			lc := strings.ToLower(upErr.Error())
			if strings.Contains(lc, "format tidak didukung") {
				return helper.JsonError(c, fiber.StatusUnsupportedMediaType, "Format tidak didukung (jpg/png/webp)")
			}
			return helper.JsonError(c, fiber.StatusBadGateway, fmt.Sprintf("Gagal upload ke OSS: %v", upErr))
		}

		// Simpan DB
		m := dto.ToModelCreateMultipart(userID, docType, url)
		if err := tx.Create(&m).Error; err != nil {
			if isUniqueViolation(err) {
				_ = tx.Rollback().Error
				return helper.JsonError(c, fiber.StatusConflict, fmt.Sprintf("Dokumen '%s' sudah ada", docType))
			}
			_ = tx.Rollback().Error
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan dokumen")
		}

		out = append(out, dto.ToResponse(m))
	}

	if err := tx.Commit().Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal commit transaksi")
	}
	return helper.JsonCreated(c, "Dokumen berhasil diunggah", out)
}

/* =========================
   LIST + FILTER + PAGINATION
========================= */
func (uc *UsersProfileDocumentController) List(c *fiber.Ctx) error {
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var q dto.ListUserProfileDocumentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	page := 1
	limit := 20
	if q.Page > 0 {
		page = q.Page
	}
	if q.Limit > 0 {
		limit = q.Limit
	}
	offset := (page - 1) * limit

	onlyAlive := true
	if q.OnlyAlive != nil {
		onlyAlive = *q.OnlyAlive
	}

	dbq := uc.DB.WithContext(c.Context()).Model(&model.UsersProfileDocumentModel{}).
		Where("user_id = ?", userID)

	if onlyAlive {
		dbq = dbq.Where("deleted_at IS NULL")
	}
	if q.DocType != nil && *q.DocType != "" {
		dbq = dbq.Where("doc_type = ?", *q.DocType)
	}

	// count
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// data
	var rows []model.UsersProfileDocumentModel
	if err := dbq.Order("uploaded_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	respRows := make([]dto.UserProfileDocumentResponse, 0, len(rows))
	for _, r := range rows {
		respRows = append(respRows, dto.ToResponse(r))
	}
	totalPages := (int(total) + limit - 1) / limit

	return helper.JsonList(c, respRows, dto.PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalItems: int(total),
		TotalPages: totalPages,
	})
}

/* =========================
   GET BY DOC TYPE
========================= */
func (uc *UsersProfileDocumentController) GetByDocType(c *fiber.Ctx) error {
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	docType := c.Params("doc_type")
	if docType == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "doc_type wajib di path")
	}

	var m model.UsersProfileDocumentModel
	if err := uc.DB.WithContext(c.Context()).
		Where("user_id = ? AND doc_type = ?", userID, docType).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Dokumen tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	return helper.JsonOK(c, "Sukses mengambil dokumen", dto.ToResponse(m))
}

/* =========================
   UPDATE - MULTIPART (partial)
========================= */
func (uc *UsersProfileDocumentController) UpdateMultipart(c *fiber.Ctx) error {
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	docType := strings.TrimSpace(c.Params("doc_type"))
	if docType == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "doc_type wajib di path")
	}

	// Ambil record existing
	var m model.UsersProfileDocumentModel
	if err := uc.DB.WithContext(c.Context()).
		Where("user_id = ? AND doc_type = ?", userID, docType).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Dokumen tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Ambil field lain (opsional)
	var in dto.UpdateUserProfileDocumentMultipart
	if err := c.BodyParser(&in); err != nil {
		log.Println("[WARN] BodyParser multipart:", err)
	}

	// Upload file baru (opsional)
	var newURL *string
	if fh, err := c.FormFile("file"); err == nil && fh != nil {
		svc, err := helperOSS.NewOSSServiceFromEnv("")
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal init OSS")
		}
		ctx, cancel := context.WithTimeout(c.Context(), 45*time.Second)
		defer cancel()

		dir := fmt.Sprintf("users/documents/%s", userID.String())
		u, upErr := svc.UploadAsWebP(ctx, fh, dir)
		if upErr != nil {
			lc := strings.ToLower(upErr.Error())
			if strings.Contains(lc, "format tidak didukung") {
				return helper.JsonError(c, fiber.StatusUnsupportedMediaType, "Format tidak didukung (jpg/png/webp)")
			}
			return helper.JsonError(c, fiber.StatusBadGateway, fmt.Sprintf("Gagal upload ke OSS: %v", upErr))
		}
		newURL = &u
	}

	// Validasi URL opsional
	if in.FileURL != nil {
		if err := uc.Validator.Var(in.FileURL, "omitempty,url"); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "file_url tidak valid")
		}
	}
	if in.FileTrashURL != nil {
		if err := uc.Validator.Var(in.FileTrashURL, "omitempty,url"); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "file_trash_url tidak valid")
		}
	}

	// Parse pending delete
	var pendingAt *time.Time
	if in.FileDeletePendingUntil != nil {
		t, err := parseRFC3339Ptr(in.FileDeletePendingUntil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "Format file_delete_pending_until harus RFC3339")
		}
		pendingAt = t
	}

	// Apply perubahan non-URL
	dto.ApplyModelUpdateMultipart(&m, nil, in.FileTrashURL, pendingAt)

	// Sumber FileURL: upload baru > file_url
	if newURL != nil {
		m.FileURL = *newURL
	} else if in.FileURL != nil {
		m.FileURL = *in.FileURL
	}

	// Simpan
	if err := uc.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui dokumen")
	}

	return helper.JsonUpdated(c, "Dokumen berhasil diperbarui", dto.ToResponse(m))
}

/* =========================
   DELETE - SOFT / HARD
========================= */
func (uc *UsersProfileDocumentController) DeleteSoft(c *fiber.Ctx) error {
	userID, err := helper.GetUserIDFromToken(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	docType := c.Params("doc_type")
	if docType == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "doc_type wajib di path")
	}

	// ?hard=true untuk hard delete
	hard := false
	if v := c.Query("hard"); v != "" {
		b, _ := strconv.ParseBool(v)
		hard = b
	}

	var m model.UsersProfileDocumentModel
	tx := uc.DB.WithContext(c.Context()).
		Where("user_id = ? AND doc_type = ?", userID, docType).
		First(&m)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Dokumen tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	if hard {
		if err := uc.DB.WithContext(c.Context()).Unscoped().Delete(&m).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
		}
		return helper.JsonDeleted(c, "Dokumen dihapus permanen", fiber.Map{"doc_type": docType})
	}

	if err := uc.DB.WithContext(c.Context()).Delete(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}
	return helper.JsonDeleted(c, "Dokumen berhasil dihapus", fiber.Map{"doc_type": docType})
}
