package controller

import (
	"errors"
	"strings"
	"time"

	cpdto "masjidku_backend/internals/features/school/classes/classes/dto"
	cpmodel "masjidku_backend/internals/features/school/classes/classes/model"
	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"
	helperOSS "masjidku_backend/internals/helpers/oss"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// controller/class_parents_controller.go
type ClassParentController struct {
    DB       *gorm.DB
    Validate *validator.Validate
}

func NewClassParentController(db *gorm.DB, v *validator.Validate) *ClassParentController {
    if v == nil {
        v = validator.New()
    }
    return &ClassParentController{DB: db, Validate: v}
}

// optional helper biar selalu aman
func (ctl *ClassParentController) v() *validator.Validate {
    if ctl.Validate == nil {
        ctl.Validate = validator.New()
    }
    return ctl.Validate
}


/* =========================
   Helpers (private)
========================= */

func clampLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

func (ctl *ClassParentController) codeExists(c *fiber.Ctx, masjidID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	tx := ctl.DB.WithContext(c.Context()).
		Model(&cpmodel.ClassParentModel{}).
		Where(`
			class_parent_masjid_id = ?
			AND class_parent_deleted_at IS NULL
			AND class_parent_code IS NOT NULL
			AND LOWER(class_parent_code) = LOWER(?)
		`, masjidID, code)

	if excludeID != nil {
		tx = tx.Where("class_parent_id <> ?", *excludeID)
	}

	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}


/* =========================
   CREATE
========================= */

// import helper di atas file ini:
// helper "masjidku_backend/internals/helpers"

func (ctl *ClassParentController) Create(c *fiber.Ctx) error {
	var req cpdto.CreateClassParentRequest

	// 1) Parse payload (JSON atau multipart text fields)
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	// 2) Ambil masjid_id dari DTO (WAJIB)
	masjidID := req.ClassParentMasjidID
	if masjidID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "class_parent_masjid_id wajib")
	}

	// (Opsional tapi disarankan) – pastikan user punya akses ke masjid ini
	// if !(helperAuth.IsOwner(c) ||
	//      helperAuth.HasRoleInMasjid(c, masjidID, "admin") ||
	//      helperAuth.HasRoleInMasjid(c, masjidID, "dkm") ||
	//      helperAuth.HasRoleInMasjid(c, masjidID, "teacher")) {
	// 	return helper.JsonError(c, fiber.StatusForbidden, "Tidak punya akses ke masjid ini")
	// }

	// 3) Cek kode unik per masjid (pakai masjidID dari DTO)
	if code := strings.TrimSpace(req.ClassParentCode); code != "" {
		exists, err := ctl.codeExists(c, masjidID, code, nil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
		}
		if exists {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
		}
	}

	// 4) Build model dari request, PAKSA masjid_id = dari DTO
	m := req.ToModel()
	m.ClassParentMasjidID = masjidID

	// 5) Jika ada file gambar (multipart) → upload ke OSS (scoped ke masjid dari DTO)
	if fh, err := helperOSS.GetImageFile(c); err == nil && fh != nil {
		url, upErr := helperOSS.UploadImageToOSSScoped(masjidID, "class-parents", fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		m.ClassParentImageURL = url
	}

	// 6) Simpan
	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat class parent")
	}

	return helper.JsonCreated(c, "Class parent berhasil dibuat", cpdto.ToClassParentResponse(m))
}


/* =========================
   GET BY ID
========================= */

func (ctl *ClassParentController) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ?", id).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	return helper.JsonOK(c, "OK", cpdto.ToClassParentResponse(m))
}

/* =========================
   LIST
========================= */

func (ctl *ClassParentController) List(c *fiber.Ctx) error {
	var q cpdto.ListClassParentQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}

	// paging
	q.Limit = clampLimit(q.Limit, 20, 100)
	if q.Offset < 0 {
		q.Offset = 0
	}

	tx := ctl.DB.WithContext(c.Context()).Model(&cpmodel.ClassParentModel{})

	if q.MasjidID != nil {
		tx = tx.Where("class_parent_masjid_id = ?", *q.MasjidID)
	}
	if q.Active != nil {
		tx = tx.Where("class_parent_is_active = ?", *q.Active)
	}
	if q.LevelMin != nil {
		tx = tx.Where("(class_parent_level IS NOT NULL AND class_parent_level >= ?)", *q.LevelMin)
	}
	if q.LevelMax != nil {
		tx = tx.Where("(class_parent_level IS NOT NULL AND class_parent_level <= ?)", *q.LevelMax)
	}
	if q.CreatedGt != nil {
		tx = tx.Where("class_parent_created_at > ?", *q.CreatedGt)
	}
	if q.CreatedLt != nil {
		tx = tx.Where("class_parent_created_at < ?", *q.CreatedLt)
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		pat := "%" + s + "%"
		tx = tx.Where(`
			class_parent_name ILIKE ? OR
			class_parent_code ILIKE ? OR
			class_parent_description ILIKE ?
		`, pat, pat, pat)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	var rows []cpmodel.ClassParentModel
	if err := tx.Order("class_parent_created_at DESC").
		Limit(q.Limit).Offset(q.Offset).
		Find(&rows).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	resps := cpdto.ToClassParentResponses(rows)
	meta := cpdto.NewPaginationMeta(total, q.Limit, q.Offset, len(resps))

	return helper.JsonList(c, resps, meta)
}

/* =========================
   UPDATE (PATCH)
========================= */
// pastikan import:
// helper "masjidku_backend/internals/helpers"
// helperOSS "masjidku_backend/internals/helpers/oss"
// cpdto "masjidku_backend/internals/features/school/classes/classes/dto"
// cpmodel "masjidku_backend/internals/features/school/classes/classes/model"

func (ctl *ClassParentController) Update(c *fiber.Ctx) error {
	// --- Ambil masjid_id dari token (teacher/admin/DKM aware)
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// --- Parse ID
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// --- Parse payload
	var req cpdto.UpdateClassParentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	// guard validator agar tidak nil
	if ctl.Validate != nil {
		if err := ctl.Validate.Struct(&req); err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// --- Load data yang memang milik masjid dari token
	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ?", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// --- Cek unique code (per masjid) bila berubah
	if req.ClassParentCode != nil && strings.TrimSpace(*req.ClassParentCode) != "" {
		exists, err := ctl.codeExists(c, masjidID, *req.ClassParentCode, &m.ClassParentID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
		}
		if exists {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
		}
	}

	// --- Terapkan patch field teks
	req.ApplyPatch(&m)

	// --- Clear image via payload (string kosong) → pindah ke spam/
	if req.ClassParentImageURL != nil &&
		strings.TrimSpace(*req.ClassParentImageURL) == "" &&
		strings.TrimSpace(m.ClassParentImageURL) != "" {
		_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second) // best-effort
		m.ClassParentImageURL = ""
	}

	// --- Ada file baru di multipart? upload & replace (lama dipindah ke spam/)
	if fh, err := helperOSS.GetImageFile(c); err == nil && fh != nil {
		newURL, upErr := helperOSS.UploadImageToOSSScoped(masjidID, "class-parents", fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		if strings.TrimSpace(m.ClassParentImageURL) != "" {
			_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second) // best-effort
		}
		m.ClassParentImageURL = newURL
	}

	// --- Simpan
	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}

	return helper.JsonUpdated(c, "Class parent berhasil diperbarui", cpdto.ToClassParentResponse(m))
}

func (ctl *ClassParentController) Delete(c *fiber.Ctx) error {
	// --- Ambil masjid_id dari token
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}

	// --- Parse ID
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// --- Ambil record milik masjid bersangkutan (untuk pindahkan gambar)
	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ?", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// --- Pindahkan file lama ke spam/ (best-effort)
	if strings.TrimSpace(m.ClassParentImageURL) != "" {
		_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second)
	}

	// --- Soft delete scoped by tenant
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ?", id, masjidID).
		Delete(&cpmodel.ClassParentModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}

	return helper.JsonDeleted(c, "Class parent berhasil dihapus", fiber.Map{"class_parent_id": id})
}
