package controller

import (
	"errors"
	"fmt"
	"regexp"
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

func (ctl *ClassParentController) v() *validator.Validate {
	if ctl.Validate == nil {
		ctl.Validate = validator.New()
	}
	return ctl.Validate
}

// ---------- helpers ----------//
//  ---------- helpers ----------
func clampLimit(limit, def, max int) int {
	if limit <= 0 { return def }
	if limit > max { return max }
	return limit
}

// slugify code dari nama: huruf/angka + '-' ; uppercase; pangkas max 32
var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)
const maxCodeLen = 32

func slugifyCode(from string) string {
	s := strings.TrimSpace(from)
	if s == "" {
		return "CP"
	}
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "CP"
	}
	s = strings.ToUpper(s)
	if len(s) > maxCodeLen {
		s = s[:maxCodeLen]
	}
	return s
}

// cek eksistensi unik (sudah ada di file kamu)
func (ctl *ClassParentController) codeExists(masjidID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	tx := ctl.DB.
		Model(&cpmodel.ClassParentModel{}).
		Where(`
			class_parent_masjid_id = ?
			AND class_parent_deleted_at IS NULL
			AND class_parent_delete_pending_until IS NULL
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

// generate code unik dari base; tambahkan -1, -2, ... bila bentrok
func (ctl *ClassParentController) ensureUniqueCode(masjidID uuid.UUID, base string, excludeID *uuid.UUID) (string, error) {
	if base = strings.TrimSpace(base); base == "" {
		base = "CP"
	}
	base = slugifyCode(base)

	code := base
	for i := 0; ; i++ {
		if i > 0 {
			suf := fmt.Sprintf("-%d", i)
			code = base
			if len(code)+len(suf) > maxCodeLen {
				code = code[:maxCodeLen-len(suf)]
			}
			code += suf
		}
		exists, err := ctl.codeExists(masjidID, code, excludeID)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
}


// ---------- CREATE ----------
// ---------- CREATE ----------
func (ctl *ClassParentController) Create(c *fiber.Ctx) error {
	var req cpdto.CreateClassParentRequest

	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.v().Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return err
	}
	if req.ClassParentMasjidID != uuid.Nil && req.ClassParentMasjidID != masjidID {
		return helper.JsonError(c, fiber.StatusForbidden, "class_parent_masjid_id pada body tidak boleh berbeda dengan token")
	}

	// Validasi unik jika user KIRIM code
	if code := strings.TrimSpace(req.ClassParentCode); code != "" {
		exists, err := ctl.codeExists(masjidID, code, nil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
		}
		if exists {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
		}
	}

	m := req.ToModel()
	m.ClassParentMasjidID = masjidID

	// >>> NEW: generate code kalau kosong
	if strings.TrimSpace(m.ClassParentCode) == "" {
		gen, err := ctl.ensureUniqueCode(masjidID, m.ClassParentName, nil)
		if err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat kode unik")
		}
		m.ClassParentCode = gen
	}

	// multipart image (opsional)
	if fh, err := helperOSS.GetImageFile(c); err == nil && fh != nil {
		publicURL, upErr := helperOSS.UploadImageToOSSScoped(masjidID, "class-parents", fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		m.ClassParentImageURL = publicURL
	}

	if err := ctl.DB.WithContext(c.Context()).Create(&m).Error; err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "uq_class_parent") && strings.Contains(low, "code") {
			return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat class parent")
	}

	return helper.JsonCreated(c, "Class parent berhasil dibuat", cpdto.ToClassParentResponse(m))
}




// ---------- UPDATE (PATCH, tenant-safe) ----------
// ---------- UPDATE (PATCH, tenant-safe) ----------
func (ctl *ClassParentController) Update(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req cpdto.UpdateClassParentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.v().Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, err.Error())
	}

	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	// Terapkan patch (nama dkk mungkin berubah)
	req.ApplyPatch(&m)

	// >>> NEW: handle code bila dikirim di PATCH
	if req.ClassParentCode != nil {
		newCode := strings.TrimSpace(*req.ClassParentCode)
		if newCode == "" {
			// user mengosongkan → generate dari nama terbaru
			gen, err := ctl.ensureUniqueCode(masjidID, m.ClassParentName, &m.ClassParentID)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat kode unik")
			}
			m.ClassParentCode = gen
		} else {
			exists, err := ctl.codeExists(masjidID, newCode, &m.ClassParentID)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
			}
			if exists {
				return helper.JsonError(c, fiber.StatusConflict, "Kode sudah digunakan pada masjid ini")
			}
			m.ClassParentCode = newCode
		}
	}

	// clear image via empty string → spam-kan
	if req.ClassParentImageURL != nil &&
		strings.TrimSpace(*req.ClassParentImageURL) == "" &&
		strings.TrimSpace(m.ClassParentImageURL) != "" {
		_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second)
		m.ClassParentImageURL = ""
	}

	// file baru? upload & replace + spam-kan lama
	if fh, err := helperOSS.GetImageFile(c); err == nil && fh != nil {
		publicURL, upErr := helperOSS.UploadImageToOSSScoped(masjidID, "class-parents", fh)
		if upErr != nil {
			return helper.JsonError(c, fiber.StatusBadGateway, "Upload gambar gagal: "+upErr.Error())
		}
		if strings.TrimSpace(m.ClassParentImageURL) != "" {
			_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second)
		}
		m.ClassParentImageURL = publicURL
	}

	if err := ctl.DB.WithContext(c.Context()).Save(&m).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal memperbarui data")
	}
	return helper.JsonUpdated(c, "Class parent berhasil diperbarui", cpdto.ToClassParentResponse(m))
}

// ---------- DELETE (soft, tenant-safe) ----------
func (ctl *ClassParentController) Delete(c *fiber.Ctx) error {
	masjidID, err := helperAuth.GetMasjidIDFromTokenPreferTeacher(c)
	if err != nil {
		return helper.JsonError(c, fiber.StatusUnauthorized, err.Error())
	}
	id, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var m cpmodel.ClassParentModel
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ? AND class_parent_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "DB error")
	}

	if strings.TrimSpace(m.ClassParentImageURL) != "" {
		_, _ = helperOSS.MoveToSpamByPublicURLENV(m.ClassParentImageURL, 15*time.Second)
	}

	// soft delete (pastikan model pakai gorm.DeletedAt di kolom class_parent_deleted_at)
	if err := ctl.DB.WithContext(c.Context()).
		Where("class_parent_id = ? AND class_parent_masjid_id = ?", id, masjidID).
		Delete(&cpmodel.ClassParentModel{}).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	return helper.JsonDeleted(c, "Class parent berhasil dihapus", fiber.Map{"class_parent_id": id})
}
