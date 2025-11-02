package controller

import (
	"net/url"
	"schoolku_backend/internals/constants"
	"schoolku_backend/internals/features/schools/lecture_sessions/materials/dto"
	"schoolku_backend/internals/features/schools/lecture_sessions/materials/model"
	helper "schoolku_backend/internals/helpers"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type LectureSessionsAssetController struct {
	DB *gorm.DB
}

func NewLectureSessionsAssetController(db *gorm.DB) *LectureSessionsAssetController {
	return &LectureSessionsAssetController{DB: db}
}

// POST /api/a/lecture-sessions/assets
func (ctrl *LectureSessionsAssetController) CreateLectureSessionsAsset(c *fiber.Ctx) error {
	schoolID, ok := c.Locals("school_id").(string)
	if !ok || schoolID == "" {
		return helper.JsonError(c, fiber.StatusUnauthorized, "School ID not found in token")
	}

	title := c.FormValue("lecture_sessions_asset_title")
	lectureSessionID := c.FormValue("lecture_sessions_asset_lecture_session_id")
	if title == "" || lectureSessionID == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "Field wajib tidak lengkap")
	}

	var fileURL string
	var fileType int

	if file, err := c.FormFile("lecture_sessions_asset_file_url"); err == nil && file != nil {
		uploadedURL, err := helper.UploadFileToSupabase("lecture_sessions_assets", file)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload file")
		}
		fileURL = uploadedURL
		fileType = constants.DetectFileTypeFromExt(file.Filename)
	} else if val := c.FormValue("lecture_sessions_asset_file_url"); val != "" {
		fileURL = val
		fileType = 1 // link (YouTube/URL)
	} else {
		return helper.JsonError(c, fiber.StatusBadRequest, "Wajib upload file atau berikan URL")
	}

	asset := model.LectureSessionsAssetModel{
		LectureSessionsAssetTitle:            title,
		LectureSessionsAssetFileURL:          fileURL,
		LectureSessionsAssetFileType:         fileType,
		LectureSessionsAssetLectureSessionID: lectureSessionID,
		LectureSessionsAssetSchoolID:         schoolID,
	}

	if err := ctrl.DB.Create(&asset).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan asset")
	}

	return helper.JsonCreated(c, "Asset berhasil dibuat", dto.ToLectureSessionsAssetDTO(asset))
}

// GET /api/a/lecture-sessions/assets
// Query params:
// - page, page_size
// - lecture_session_id, school_id, file_type
// - q (search title), order ∈ newest|oldest|title_asc|title_desc
func (ctrl *LectureSessionsAssetController) GetAllLectureSessionsAssets(c *fiber.Ctx) error {
	// pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// filters
	lectureSessionID := c.Query("lecture_session_id")
	schoolID := c.Query("school_id")
	q := strings.TrimSpace(c.Query("q"))
	fileTypeStr := c.Query("file_type")
	order := strings.ToLower(strings.TrimSpace(c.Query("order", "newest")))

	query := ctrl.DB.Model(&model.LectureSessionsAssetModel{})

	if lectureSessionID != "" {
		query = query.Where("lecture_sessions_asset_lecture_session_id = ?", lectureSessionID)
	}
	if schoolID != "" {
		query = query.Where("lecture_sessions_asset_school_id = ?", schoolID)
	}
	if fileTypeStr != "" {
		if ft, err := strconv.Atoi(fileTypeStr); err == nil && ft > 0 {
			query = query.Where("lecture_sessions_asset_file_type = ?", ft)
		}
	}

	// search by title (FTS + fallback ILIKE)
	if q != "" {
		// gunakan FTS kalau tersedia (kolom tsv sudah dibuat di migration)
		// fallback ILIKE untuk fuzzy sederhana
		query = query.Where(`
			lecture_sessions_asset_title_tsv @@ websearch_to_tsquery('simple', ?) 
			OR LOWER(lecture_sessions_asset_title) LIKE LOWER(?)
		`, q, "%"+q+"%")
	}

	// count sebelum limit/offset
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// sorting
	switch order {
	case "oldest":
		query = query.Order("lecture_sessions_asset_created_at ASC")
	case "title_asc":
		query = query.Order("lecture_sessions_asset_title ASC")
	case "title_desc":
		query = query.Order("lecture_sessions_asset_title DESC")
	default: // newest
		query = query.Order("lecture_sessions_asset_created_at DESC")
	}

	var assets []model.LectureSessionsAssetModel
	if err := query.Limit(pageSize).Offset(offset).Find(&assets).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve assets")
	}

	resp := make([]dto.LectureSessionsAssetDTO, len(assets))
	for i, a := range assets {
		resp[i] = dto.ToLectureSessionsAssetDTO(a)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	pagination := fiber.Map{
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
	}

	return helper.JsonList(c, resp, pagination)
}

// GET /api/a/lecture-sessions/assets/:id
func (ctrl *LectureSessionsAssetController) GetLectureSessionsAssetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	var asset model.LectureSessionsAssetModel
	if err := ctrl.DB.First(&asset, "lecture_sessions_asset_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Asset tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data asset")
	}

	return helper.JsonOK(c, "Berhasil mengambil asset", dto.ToLectureSessionsAssetDTO(asset))
}

// PATCH /api/a/lecture-sessions/assets/:id
func (ctrl *LectureSessionsAssetController) UpdateLectureSessionsAsset(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	var asset model.LectureSessionsAssetModel
	if err := ctrl.DB.First(&asset, "lecture_sessions_asset_id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helper.JsonError(c, fiber.StatusNotFound, "Asset tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data asset")
	}

	if title := c.FormValue("lecture_sessions_asset_title"); title != "" {
		asset.LectureSessionsAssetTitle = title
	}

	if file, errFile := c.FormFile("lecture_sessions_asset_file_url"); errFile == nil && file != nil {
		// hapus file lama bila dari Supabase
		if asset.LectureSessionsAssetFileURL != "" {
			if parsed, err := url.Parse(asset.LectureSessionsAssetFileURL); err == nil {
				prefix := "/storage/v1/object/public/"
				cleaned := strings.TrimPrefix(parsed.Path, prefix)
				if unescaped, err := url.QueryUnescape(cleaned); err == nil {
					if parts := strings.SplitN(unescaped, "/", 2); len(parts) == 2 {
						_ = helper.DeleteFromSupabase(parts[0], parts[1])
					}
				}
			}
		}
		uploadedURL, err := helper.UploadFileToSupabase("lecture_sessions_assets", file)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal upload file")
		}
		asset.LectureSessionsAssetFileURL = uploadedURL
		asset.LectureSessionsAssetFileType = constants.DetectFileTypeFromExt(file.Filename)
	} else if val := c.FormValue("lecture_sessions_asset_file_url"); val != "" {
		asset.LectureSessionsAssetFileURL = val
		asset.LectureSessionsAssetFileType = 1 // link
	}

	if err := ctrl.DB.Save(&asset).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui asset")
	}

	return helper.JsonUpdated(c, "Asset berhasil diperbarui", dto.ToLectureSessionsAssetDTO(asset))
}

// DELETE /api/a/lecture-sessions/assets/:id
func (ctrl *LectureSessionsAssetController) DeleteLectureSessionsAsset(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak boleh kosong")
	}

	var asset model.LectureSessionsAssetModel
	if err := ctrl.DB.First(&asset, "lecture_sessions_asset_id = ?", id).Error; err != nil {
		return helper.JsonError(c, fiber.StatusNotFound, "Asset tidak ditemukan")
	}

	// hapus file dari Supabase jika URL-nya dari bucket
	if asset.LectureSessionsAssetFileURL != "" {
		if parsed, err := url.Parse(asset.LectureSessionsAssetFileURL); err == nil {
			rawPath := parsed.Path
			prefix := "/storage/v1/object/public/"
			cleaned := strings.TrimPrefix(rawPath, prefix)
			if unescaped, err := url.QueryUnescape(cleaned); err == nil {
				if parts := strings.SplitN(unescaped, "/", 2); len(parts) == 2 {
					_ = helper.DeleteFromSupabase(parts[0], parts[1])
				}
			}
		}
	}

	if err := ctrl.DB.Delete(&asset).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus asset")
	}

	return helper.JsonDeleted(c, "Asset berhasil dihapus", fiber.Map{"lecture_sessions_asset_id": id})
}
