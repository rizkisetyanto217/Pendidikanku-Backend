package controller

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"masjidku_backend/internals/features/lembaga/classes/main/dto"
	"masjidku_backend/internals/features/lembaga/classes/main/model"
	helper "masjidku_backend/internals/helpers"

	"github.com/go-playground/validator/v10"
)

/* ================= Controller & Constructor ================= */

type ClassController struct {
	DB *gorm.DB
}

func NewClassController(db *gorm.DB) *ClassController {
	return &ClassController{DB: db}
}

// single validator instance for this package (tidak perlu di-inject)
var validate = validator.New()

/* ================= Handlers ================= */

// POST /admin/classes
// internals/features/lembaga/classes/main/controller/class_controller.go

// POST /admin/classes
// POST /api/a/classes  (multipart/form-data ATAU JSON)
func (ctrl *ClassController) CreateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var req dto.CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// Paksa tenant dari token (abaikan input klien)
	req.ClassMasjidID = &masjidID

	// === Auto-generate slug ===
	if strings.TrimSpace(req.ClassSlug) == "" {
		req.ClassSlug = helper.NormalizeSlug(req.ClassName)
	} else {
		req.ClassSlug = helper.NormalizeSlug(req.ClassSlug)
	}

	// Validasi DTO
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// === (Opsional) Upload gambar ke Supabase jika ada file ===
	// Form field: "class_image"
	if fileHeader, err := c.FormFile("class_image_url"); err == nil && fileHeader != nil {
		publicURL, upErr := helper.UploadImageToSupabase("classes", fileHeader)
		if upErr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
		req.ClassImageURL = &publicURL
	}

	// Mapping ke model
	m := req.ToModel()

	// Guard slug unik (non-deleted)
	if err := ctrl.DB.Where("class_slug = ? AND class_deleted_at IS NULL", m.ClassSlug).
		First(&model.ClassModel{}).Error; err == nil {
		return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
	}

	// Simpan
	if err := ctrl.DB.Create(m).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat data kelas")
	}

	return helper.JsonCreated(c, "Kelas berhasil dibuat", dto.NewClassResponse(m))
}



func (ctrl *ClassController) UpdateClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var existing model.ClassModel
	if err := ctrl.DB.First(&existing, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if existing.ClassMasjidID == nil || *existing.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengubah kelas di masjid lain")
	}

	var req dto.UpdateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Payload tidak valid")
	}

	// --- Normalize slug/name ---
	if req.ClassSlug != nil {
		s := helper.NormalizeSlug(*req.ClassSlug)
		req.ClassSlug = &s
	} else if req.ClassName != nil { // regen slug dari name jika slug tidak dikirim
		s := helper.NormalizeSlug(*req.ClassName)
		req.ClassSlug = &s
	}

	// Cegah pindah tenant
	req.ClassMasjidID = &masjidID

	// === (Opsional) Upload gambar baru jika ada file "class_image" ===
	if fileHeader, err := c.FormFile("class_image"); err == nil && fileHeader != nil {
		// Hapus file lama (jika dari Supabase public)
		if existing.ClassImageURL != nil && *existing.ClassImageURL != "" {
			if bucket, path, exErr := helper.ExtractSupabasePath(*existing.ClassImageURL); exErr == nil {
				_ = helper.DeleteFromSupabase(bucket, path) // abaikan error non-kritis
			}
		}
		publicURL, upErr := helper.UploadImageToSupabase("classes", fileHeader)
		if upErr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Upload gambar gagal: "+upErr.Error())
		}
		req.ClassImageURL = &publicURL
	}

	// Validasi DTO
	if err := validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Cek unik slug (exclude current id) bila akan mengubah slug
	if req.ClassSlug != nil && *req.ClassSlug != existing.ClassSlug {
		var cnt int64
		if err := ctrl.DB.Model(&model.ClassModel{}).
			Where("class_slug = ? AND class_id <> ? AND class_deleted_at IS NULL", *req.ClassSlug, existing.ClassID).
			Count(&cnt).Error; err == nil && cnt > 0 {
			return fiber.NewError(fiber.StatusConflict, "Slug sudah digunakan")
		}
	}

	// Jika klien mengirim ClassImageURL berbeda (tanpa file), hapus lama dulu
	if req.ClassImageURL != nil && existing.ClassImageURL != nil && *existing.ClassImageURL != *req.ClassImageURL {
		if bucket, path, exErr := helper.ExtractSupabasePath(*existing.ClassImageURL); exErr == nil {
			_ = helper.DeleteFromSupabase(bucket, path)
		}
	}

	// Apply & save
	req.ApplyToModel(&existing)
	if err := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_id = ?", existing.ClassID).
		Updates(&existing).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Kelas berhasil diperbarui", dto.NewClassResponse(&existing))
}

// GET /admin/classes/:id
func (ctrl *ClassController) GetClassByID(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.ClassModel
	if err := ctrl.DB.First(&m, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	// tenant check
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh mengakses kelas di masjid lain")
	}
	return helper.JsonOK(c, "Data diterima", dto.NewClassResponse(&m))
}


// GET /admin/classes
func (ctrl *ClassController) ListClasses(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	var q dto.ListClassQuery
	// default paging
	q.Limit = 20
	q.Offset = 0
	if err := c.QueryParser(&q); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Query tidak valid")
	}

	tx := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_deleted_at IS NULL").
		Where("class_masjid_id = ?", masjidID)

	if q.ActiveOnly != nil {
		tx = tx.Where("class_is_active = ?", *q.ActiveOnly)
	}
	if q.Search != nil && strings.TrimSpace(*q.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(*q.Search)) + "%"
		tx = tx.Where("(LOWER(class_name) LIKE ? OR LOWER(class_level) LIKE ?)", s, s)
	}
	sortVal := ""
	if q.Sort != nil {
		sortVal = strings.ToLower(strings.TrimSpace(*q.Sort))
	}

	switch sortVal {
	case "name_asc":
		tx = tx.Order("class_name ASC")
	case "name_desc":
		tx = tx.Order("class_name DESC")
	case "created_at_asc":
		tx = tx.Order("class_created_at ASC")
	default:
		tx = tx.Order("class_created_at DESC")
	}

	if q.Limit > 0 {
		tx = tx.Limit(q.Limit)
	}
	if q.Offset > 0 {
		tx = tx.Offset(q.Offset)
	}

	var rows []model.ClassModel
	if err := tx.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	resp := make([]*dto.ClassResponse, 0, len(rows))
	for i := range rows {
		resp = append(resp, dto.NewClassResponse(&rows[i]))
	}
	return helper.JsonOK(c, "OK", resp)
}


// DELETE /admin/classes/:id  (soft delete)
// Tambahan: ?delete_image=true untuk ikut hapus file gambar di Supabase
func (ctrl *ClassController) SoftDeleteClass(c *fiber.Ctx) error {
	masjidID, err := helper.GetMasjidIDFromToken(c)
	if err != nil {
		return err
	}

	classID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "ID tidak valid")
	}

	var m model.ClassModel
	if err := ctrl.DB.First(&m, "class_id = ? AND class_deleted_at IS NULL", classID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.NewError(fiber.StatusNotFound, "Kelas tidak ditemukan")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil data")
	}
	if m.ClassMasjidID == nil || *m.ClassMasjidID != masjidID {
		return fiber.NewError(fiber.StatusForbidden, "Tidak boleh menghapus kelas di masjid lain")
	}

	// Opsional: hapus file gambar di Supabase
	deletedImage := false
	if strings.EqualFold(c.Query("delete_image"), "true") && m.ClassImageURL != nil && *m.ClassImageURL != "" {
		if bucket, path, exErr := helper.ExtractSupabasePath(*m.ClassImageURL); exErr == nil {
			_ = helper.DeleteFromSupabase(bucket, path) // abaikan error non-kritis
			deletedImage = true
		}
		// Kosongkan URL di DB biar konsisten
		m.ClassImageURL = nil
	}

	now := time.Now()
	updates := map[string]any{
		"class_deleted_at": now,
		"class_is_active":  false,
		"class_updated_at": now,
	}
	// Jika barusan dihapus gambarnya, sekalian null-kan field-nya
	if deletedImage {
		updates["class_image_url"] = nil
	}

	if err := ctrl.DB.Model(&model.ClassModel{}).
		Where("class_id = ?", m.ClassID).
		Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Kelas berhasil dihapus", fiber.Map{
		"class_id":      m.ClassID,
		"deleted_image": deletedImage,
	})
}
