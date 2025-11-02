package controller

import (
	lectureSessionModel "schoolku_backend/internals/features/schools/lecture_sessions/main/model"
	"schoolku_backend/internals/features/schools/lecture_sessions/materials/dto"
	"schoolku_backend/internals/features/schools/lecture_sessions/materials/model"
	lectureModel "schoolku_backend/internals/features/schools/lectures/main/model"

	resp "schoolku_backend/internals/helpers"

	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// =============================
// Filter by lecture_session_id atau lecture_id
// =============================
func (ctrl *LectureSessionsAssetController) FilterLectureLectureSessionsAssets(c *fiber.Ctx) error {
	lectureSessionID := c.Query("lecture_session_id")
	lectureID := c.Query("lecture_id")
	fileTypeQuery := c.Query("file_type")

	// file_type wajib
	if strings.TrimSpace(fileTypeQuery) == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "Missing file_type")
	}

	// parse file_type (comma-separated ints)
	fileTypes := make([]int, 0, 4)
	for _, s := range strings.Split(fileTypeQuery, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		ft, err := strconv.Atoi(s)
		if err != nil {
			return resp.JsonError(c, fiber.StatusBadRequest, "Invalid file_type value")
		}
		fileTypes = append(fileTypes, ft)
	}
	if len(fileTypes) == 0 {
		return resp.JsonError(c, fiber.StatusBadRequest, "file_type tidak boleh kosong")
	}

	// setidaknya salah satu harus ada
	if lectureSessionID == "" && lectureID == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "Either lecture_session_id or lecture_id must be provided")
	}

	var sessionIDs []string

	// jika lecture_id disediakan → ambil semua session_id dari lecture tsb
	if lectureID != "" {
		if err := ctrl.DB.WithContext(c.Context()).
			Table("lecture_sessions").
			Where("lecture_session_lecture_id = ?", lectureID).
			Pluck("lecture_session_id", &sessionIDs).Error; err != nil {
			return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi dari lecture_id")
		}
		// kalau memang tidak ada sesi, kembalikan array kosong (bukan 404)
		if len(sessionIDs) == 0 && lectureSessionID == "" {
			return resp.JsonOK(c, "OK", []dto.LectureSessionsAssetDTO{})
		}
	}

	// Ambil data asset
	var assets []model.LectureSessionsAssetModel
	q := ctrl.DB.WithContext(c.Context()).
		Model(&model.LectureSessionsAssetModel{})

	if lectureSessionID != "" {
		q = q.Where("lecture_sessions_asset_lecture_session_id = ?", lectureSessionID)
	} else {
		q = q.Where("lecture_sessions_asset_lecture_session_id IN ?", sessionIDs)
	}

	if err := q.
		Where("lecture_sessions_asset_file_type IN ?", fileTypes).
		Find(&assets).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve assets")
	}

	// Mapping ke DTO
	out := make([]dto.LectureSessionsAssetDTO, 0, len(assets))
	for _, a := range assets {
		out = append(out, dto.ToLectureSessionsAssetDTO(a))
	}

	return resp.JsonOK(c, "OK", out)
}

// =============================
// Filter by lecture_slug atau lecture_session_slug
// =============================
func (ctrl *LectureSessionsAssetController) FilterLectureSessionsAssetsBySlug(c *fiber.Ctx) error {
	lectureSlug := c.Query("lecture_slug")
	lectureSessionSlug := c.Query("lecture_session_slug")
	fileTypeQuery := c.Query("file_type")

	// file_type wajib
	if strings.TrimSpace(fileTypeQuery) == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "Missing file_type")
	}

	// parse file_type (comma-separated ints)
	fileTypes := make([]int, 0, 4)
	for _, s := range strings.Split(fileTypeQuery, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		ft, err := strconv.Atoi(s)
		if err != nil {
			return resp.JsonError(c, fiber.StatusBadRequest, "Invalid file_type value")
		}
		fileTypes = append(fileTypes, ft)
	}
	if len(fileTypes) == 0 {
		return resp.JsonError(c, fiber.StatusBadRequest, "file_type tidak boleh kosong")
	}

	// minimal salah satu slug harus ada
	if lectureSlug == "" && lectureSessionSlug == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "Either lecture_slug or lecture_session_slug must be provided")
	}

	var sessionIDs []string

	// dari lecture_slug → lecture_id → semua session_id
	if lectureSlug != "" {
		var lecture lectureModel.LectureModel
		if err := ctrl.DB.WithContext(c.Context()).
			Where("lecture_slug = ?", lectureSlug).
			First(&lecture).Error; err != nil {
			return resp.JsonError(c, fiber.StatusNotFound, "Lecture tidak ditemukan")
		}

		if err := ctrl.DB.WithContext(c.Context()).
			Table("lecture_sessions").
			Where("lecture_session_lecture_id = ?", lecture.LectureID).
			Pluck("lecture_session_id", &sessionIDs).Error; err != nil {
			return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi dari lecture_slug")
		}
		// jika lecture ada tapi belum punya sesi, kembalikan array kosong
		if len(sessionIDs) == 0 && lectureSessionSlug == "" {
			return resp.JsonOK(c, "OK", []dto.LectureSessionsAssetDTO{})
		}
	}

	// dari lecture_session_slug → satu session_id
	if lectureSessionSlug != "" {
		var session lectureSessionModel.LectureSessionModel
		if err := ctrl.DB.WithContext(c.Context()).
			Where("lecture_session_slug = ?", lectureSessionSlug).
			First(&session).Error; err != nil {
			return resp.JsonError(c, fiber.StatusNotFound, "Sesi kajian tidak ditemukan")
		}
		sessionIDs = []string{session.LectureSessionID.String()}
	}

	// Query aset
	var assets []model.LectureSessionsAssetModel
	if err := ctrl.DB.WithContext(c.Context()).
		Model(&model.LectureSessionsAssetModel{}).
		Where("lecture_sessions_asset_lecture_session_id IN ?", sessionIDs).
		Where("lecture_sessions_asset_file_type IN ?", fileTypes).
		Find(&assets).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data asset")
	}

	// Map ke DTO
	out := make([]dto.LectureSessionsAssetDTO, 0, len(assets))
	for _, a := range assets {
		out = append(out, dto.ToLectureSessionsAssetDTO(a))
	}

	return resp.JsonOK(c, "OK", out)
}

// =============================
// Grouped by Lecture ID
// =============================
func (ctl *LectureSessionsAssetController) FindGroupedByLectureID(c *fiber.Ctx) error {
	lectureID := c.Query("lecture_id")
	fileTypes := c.Query("file_type") // contoh: "1,2"

	if strings.TrimSpace(lectureID) == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "lecture_id wajib diisi")
	}

	q := ctl.DB.WithContext(c.Context()).
		Model(&model.LectureSessionsAssetModel{}).
		Joins("JOIN lecture_sessions ON lecture_sessions.lecture_session_id = lecture_sessions_assets.lecture_sessions_asset_lecture_session_id").
		Preload("LectureSession"). // untuk ambil title
		Where("lecture_sessions.lecture_session_lecture_id = ?", lectureID)

	if strings.TrimSpace(fileTypes) != "" {
		types := make([]string, 0, 4)
		for _, s := range strings.Split(fileTypes, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				types = append(types, s)
			}
		}
		if len(types) > 0 {
			q = q.Where("lecture_sessions_assets.lecture_sessions_asset_file_type IN (?)", types)
		}
	}

	var assets []model.LectureSessionsAssetModel
	if err := q.
		Order("lecture_sessions_assets.lecture_sessions_asset_lecture_session_id DESC").
		Find(&assets).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	if len(assets) == 0 {
		return resp.JsonOK(c, "success", []GroupedAssets{})
	}

	grouped := groupByLectureSessionID(assets)
	return resp.JsonOK(c, "success", grouped)
}

// =============================
// Grouped by Lecture Slug
// =============================
func (ctl *LectureSessionsAssetController) FindGroupedByLectureSlug(c *fiber.Ctx) error {
	lectureSlug := c.Query("lecture_slug")
	fileTypes := c.Query("file_type") // contoh: "1,2"

	if strings.TrimSpace(lectureSlug) == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "lecture_slug wajib diisi")
	}

	// Ambil Lecture ID dari slug
	var lecture lectureModel.LectureModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("lecture_slug = ?", lectureSlug).
		First(&lecture).Error; err != nil {
		return resp.JsonError(c, fiber.StatusNotFound, "Lecture tidak ditemukan")
	}

	q := ctl.DB.WithContext(c.Context()).
		Model(&model.LectureSessionsAssetModel{}).
		Joins("JOIN lecture_sessions ON lecture_sessions.lecture_session_id = lecture_sessions_assets.lecture_sessions_asset_lecture_session_id").
		Preload("LectureSession").
		Where("lecture_sessions.lecture_session_lecture_id = ?", lecture.LectureID)

	if strings.TrimSpace(fileTypes) != "" {
		types := make([]string, 0, 4)
		for _, s := range strings.Split(fileTypes, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				types = append(types, s)
			}
		}
		if len(types) > 0 {
			q = q.Where("lecture_sessions_assets.lecture_sessions_asset_file_type IN (?)", types)
		}
	}

	var assets []model.LectureSessionsAssetModel
	if err := q.
		Order("lecture_sessions_assets.lecture_sessions_asset_lecture_session_id DESC").
		Find(&assets).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "gagal mengambil data")
	}

	if len(assets) == 0 {
		return resp.JsonOK(c, "success", []GroupedAssets{})
	}

	grouped := groupByLectureSessionID(assets)
	return resp.JsonOK(c, "success", grouped)
}

// =============================
// Struct & Grouping helper
// =============================
type GroupedAssets struct {
	LectureSessionID    string                             `json:"lecture_session_id"`
	LectureSessionTitle string                             `json:"lecture_session_title"`
	Assets              []dto.LectureSessionsAssetResponse `json:"assets"`
}

func groupByLectureSessionID(data []model.LectureSessionsAssetModel) []GroupedAssets {
	type groupKey struct {
		SessionID string
		Title     string
	}
	groupMap := make(map[groupKey][]dto.LectureSessionsAssetResponse)

	for _, item := range data {
		key := groupKey{
			SessionID: item.LectureSessionsAssetLectureSessionID,
			Title:     item.LectureSession.LectureSessionTitle, // pastikan relasi dimuat
		}
		groupMap[key] = append(groupMap[key], dto.ToLectureSessionsAssetResponse(item))
	}

	result := make([]GroupedAssets, 0, len(groupMap))
	for key, assets := range groupMap {
		result = append(result, GroupedAssets{
			LectureSessionID:    key.SessionID,
			LectureSessionTitle: key.Title,
			Assets:              assets,
		})
	}

	// urutkan desc by session_id (ubah kalau mau by title/time)
	sort.Slice(result, func(i, j int) bool {
		return result[i].LectureSessionID > result[j].LectureSessionID
	})

	return result
}
