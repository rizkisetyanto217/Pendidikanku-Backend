package controller

import (
	"errors"
	"strings"

	resp "masjidku_backend/internals/helpers"

	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/dto"
	"masjidku_backend/internals/features/masjids/lecture_sessions/materials/model"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var validate2 = validator.New()

type LectureSessionsMaterialController struct {
	DB *gorm.DB
}

func NewLectureSessionsMaterialController(db *gorm.DB) *LectureSessionsMaterialController {
	return &LectureSessionsMaterialController{DB: db}
}

// =============================
// ➕ Create Lecture Session Material (maksimal 1 per session per masjid)
// =============================
func (ctrl *LectureSessionsMaterialController) CreateLectureSessionsMaterial(c *fiber.Ctx) error {
	var body dto.CreateLectureSessionsMaterialRequest

	// Parse body
	if err := c.BodyParser(&body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Invalid request body: "+err.Error())
	}

	// Ambil masjid_id dari token/middleware
	masjidID, ok := c.Locals("masjid_id").(string)
	if !ok || masjidID == "" {
		return resp.JsonError(c, fiber.StatusUnauthorized, "Masjid ID tidak ditemukan dalam token")
	}

	// Validasi payload
	if err := validate2.Struct(&body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// Cek duplikasi: per (lecture_session_id, masjid_id) – hanya baris hidup (soft delete otomatis di-skip oleh GORM)
	var existing model.LectureSessionsMaterialModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_material_lecture_session_id = ? AND lecture_sessions_material_masjid_id = ?",
			body.LectureSessionsMaterialLectureSessionID, masjidID).
		First(&existing).Error; err == nil {
		return resp.JsonError(c, fiber.StatusConflict, "Materi untuk sesi ini sudah tersedia")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal memeriksa duplikasi: "+err.Error())
	}

	// Simpan
	material := body.ToModel(masjidID)
	if err := ctrl.DB.WithContext(c.Context()).Create(material).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan materi: "+err.Error())
	}

	return resp.JsonCreated(c, "Materi berhasil ditambahkan", dto.ToLectureSessionsMaterialDTO(*material))
}

// =============================
// 📄 Get All Materials (baris hidup saja)
// =============================
func (ctrl *LectureSessionsMaterialController) GetAllLectureSessionsMaterials(c *fiber.Ctx) error {
	var materials []model.LectureSessionsMaterialModel
	if err := ctrl.DB.WithContext(c.Context()).Find(&materials).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to retrieve materials")
	}

	out := make([]dto.LectureSessionsMaterialDTO, 0, len(materials))
	for _, m := range materials {
		out = append(out, dto.ToLectureSessionsMaterialDTO(m))
	}
	return resp.JsonOK(c, "OK", out)
}

// =============================
// 🔍 Get Material by ID
// =============================
func (ctrl *LectureSessionsMaterialController) GetLectureSessionsMaterialByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var material model.LectureSessionsMaterialModel

	if err := ctrl.DB.WithContext(c.Context()).
		First(&material, "lecture_sessions_material_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Material not found")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to get material")
	}

	return resp.JsonOK(c, "OK", dto.ToLectureSessionsMaterialDTO(material))
}

// =============================
// ✏️ PATCH (Partial) Update Material by ID
// =============================
// Catatan: untuk "clear ke NULL", kirim string kosong "" pada field yang ingin dihapus.
func (ctrl *LectureSessionsMaterialController) UpdateLectureSessionsMaterial(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "ID materi tidak ditemukan di URL")
	}

	var body dto.UpdateLectureSessionsMaterialRequest
	if err := c.BodyParser(&body); err != nil {
		return resp.JsonError(c, fiber.StatusBadRequest, "Gagal parsing body: "+err.Error())
	}

	var material model.LectureSessionsMaterialModel
	if err := ctrl.DB.WithContext(c.Context()).
		First(&material, "lecture_sessions_material_id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp.JsonError(c, fiber.StatusNotFound, "Materi tidak ditemukan")
		}
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil materi")
	}

	updates := map[string]any{}

	// Summary: pointer-aware + allow clear-to-NULL via "" (trimmed)
	if body.LectureSessionsMaterialSummary != nil {
		s := strings.TrimSpace(*body.LectureSessionsMaterialSummary)
		if s == "" {
			updates["lecture_sessions_material_summary"] = gorm.Expr("NULL")
		} else {
			updates["lecture_sessions_material_summary"] = s
		}
	}

	// Transcript: pointer-aware + allow clear-to-NULL via ""
	if body.LectureSessionsMaterialTranscriptFull != nil {
		t := strings.TrimSpace(*body.LectureSessionsMaterialTranscriptFull)
		if t == "" {
			updates["lecture_sessions_material_transcript_full"] = gorm.Expr("NULL")
		} else {
			updates["lecture_sessions_material_transcript_full"] = t
		}
	}

	// Tidak mengizinkan pindah session_id melalui update biasa (hindari migrasi data tak sengaja)

	if len(updates) == 0 {
		return resp.JsonOK(c, "No changes", dto.ToLectureSessionsMaterialDTO(material))
	}

	if err := ctrl.DB.WithContext(c.Context()).
		Model(&material).
		Updates(updates).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengupdate materi: "+err.Error())
	}

	// re-fetch
	if err := ctrl.DB.WithContext(c.Context()).
		First(&material, "lecture_sessions_material_id = ?", id).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Updated but failed to re-fetch")
	}

	return resp.JsonUpdated(c, "Materi berhasil diperbarui", dto.ToLectureSessionsMaterialDTO(material))
}

// =============================
// ❌ Delete Material (soft delete)
// =============================
func (ctrl *LectureSessionsMaterialController) DeleteLectureSessionsMaterial(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctrl.DB.WithContext(c.Context()).
		Delete(&model.LectureSessionsMaterialModel{}, "lecture_sessions_material_id = ?", id).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Failed to delete material")
	}
	return resp.JsonDeleted(c, "Material deleted", nil)
}

// =============================
// 📦 Get Content (materials + assets) by Lecture ID
// =============================
func (ctrl *LectureSessionsMaterialController) GetContentByLectureID(c *fiber.Ctx) error {
	lectureID := c.Query("lecture_id")
	if strings.TrimSpace(lectureID) == "" {
		return resp.JsonError(c, fiber.StatusBadRequest, "lecture_id wajib diisi")
	}

	// Ambil semua session ID dari lecture ini
	var sessionIDs []string
	if err := ctrl.DB.WithContext(c.Context()).
		Table("lecture_sessions").
		Where("lecture_session_lecture_id = ?", lectureID).
		Pluck("lecture_session_id", &sessionIDs).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil sesi kajian")
	}

	// Jika lecture valid tapi belum punya sesi → kembalikan konten kosong
	if len(sessionIDs) == 0 {
		return resp.JsonOK(c, "success", []map[string]any{})
	}

	// Materials (soft-deleted otomatis ter-skip)
	var materials []model.LectureSessionsMaterialModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_material_lecture_session_id IN ?", sessionIDs).
		Find(&materials).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil materi")
	}

	// Assets (soft-deleted otomatis ter-skip)
	var assets []model.LectureSessionsAssetModel
	if err := ctrl.DB.WithContext(c.Context()).
		Where("lecture_sessions_asset_lecture_session_id IN ?", sessionIDs).
		Find(&assets).Error; err != nil {
		return resp.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil asset")
	}

	// Gabungkan ke satu list
	content := make([]map[string]any, 0, len(materials)+len(assets))
	for _, m := range materials {
		content = append(content, map[string]any{
			"type":       "material",
			"id":         m.LectureSessionsMaterialID,
			"summary":    m.LectureSessionsMaterialSummary,
			"transcript": m.LectureSessionsMaterialTranscriptFull,
			"session_id": m.LectureSessionsMaterialLectureSessionID,
			"created_at": m.LectureSessionsMaterialCreatedAt,
		})
	}
	for _, a := range assets {
		content = append(content, map[string]any{
			"type":       "asset",
			"id":         a.LectureSessionsAssetID,
			"title":      a.LectureSessionsAssetTitle,
			"file_url":   a.LectureSessionsAssetFileURL,
			"file_type":  a.LectureSessionsAssetFileType,
			"session_id": a.LectureSessionsAssetLectureSessionID,
			"created_at": a.LectureSessionsAssetCreatedAt,
		})
	}

	return resp.JsonOK(c, "Berhasil mengambil seluruh konten kajian berdasarkan lecture_id", content)
}
