// file: internals/features/school/class_rooms/controller/class_room_controller.go
package controller

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	dto "masjidku_backend/internals/features/school/academics/rooms/dto"
	model "masjidku_backend/internals/features/school/academics/rooms/model"
)

/* =======================================================
   CONTROLLER
   ======================================================= */

type ClassRoomController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewClassRoomController(db *gorm.DB, v *validator.Validate) *ClassRoomController {
	if v == nil {
		v = validator.New(validator.WithRequiredStructEnabled())
	}
	return &ClassRoomController{DB: db, Validate: v}
}

// jaga-jaga kalau ada controller lama yang di-init tanpa validator
func (ctl *ClassRoomController) ensureValidator() {
	if ctl.Validate == nil {
		ctl.Validate = validator.New(validator.WithRequiredStructEnabled())
	}
}

// ambil context standar (kalau Fiber mendukung UserContext)
func reqCtx(c *fiber.Ctx) context.Context {
	if uc := c.UserContext(); uc != nil {
		return uc
	}
	return context.Background()
}

/* ============================ CREATE ============================ */
func (ctl *ClassRoomController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	// 🔒 Ambil context masjid
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	// 📦 Parse body
	var req dto.CreateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}

	// 🚩 Isi dari context server
	req.ClassRoomMasjidID = masjidID

	// ✅ Validasi payload
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// 🔁 Auto-generate slug unik dari nama (jika kosong)
	base := ""
	if req.ClassRoomSlug != nil {
		base = strings.TrimSpace(*req.ClassRoomSlug)
	}
	if base == "" {
		base = helper.Slugify(req.ClassRoomName, 50)
	}
	slug, err := helper.EnsureUniqueSlugCI(
		reqCtx(c), ctl.DB,
		"class_rooms", "class_room_slug",
		base,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_room_masjid_id = ? AND class_room_deleted_at IS NULL", masjidID)
		},
		50,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
	}

	// 🧭 Map DTO → model
	m, err := req.ToModel()
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid (features/virtual_links)")
	}

	// 🔒 Pastikan masjid_id dan slug fix dari server
	m.ClassRoomMasjidID = masjidID
	m.ClassRoomSlug = &slug

	// 💾 Simpan ke DB
	if err := ctl.DB.WithContext(reqCtx(c)).Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode/Slug ruang sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	return helper.JsonCreated(c, "Created", dto.ToClassRoomResponse(m))
}

/* ============================ UPDATE (PUT/PATCH semantics) ============================ */
func (ctl *ClassRoomController) Update(c *fiber.Ctx) error {
	ctl.ensureValidator()

	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Terapkan patch (mutasi in-place)
	if err := req.ApplyPatch(&m); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Gagal menerapkan perubahan: "+err.Error())
	}

	// === NEW: Auto-update slug ketika nama berubah, kecuali slug diisi eksplisit ===
	if req.ClassRoomName != nil {
		// Hanya generate otomatis jika user TIDAK kirim slug baru
		if req.ClassRoomSlug == nil || strings.TrimSpace(*req.ClassRoomSlug) == "" {
			base := helper.Slugify(*req.ClassRoomName, 50)
			slug, err := helper.EnsureUniqueSlugCI(
				reqCtx(c), ctl.DB,
				"class_rooms", "class_room_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					// unik per masjid, exclude diri sendiri, hanya alive
					return q.Where("class_room_masjid_id = ? AND class_room_id <> ? AND class_room_deleted_at IS NULL", masjidID, id)
				},
				50,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
			}
			m.ClassRoomSlug = &slug
		} else {
			// Jika user kirim slug, pastikan unik juga (normalisasi + unik)
			base := helper.Slugify(strings.TrimSpace(*req.ClassRoomSlug), 50)
			slug, err := helper.EnsureUniqueSlugCI(
				reqCtx(c), ctl.DB,
				"class_rooms", "class_room_slug",
				base,
				func(q *gorm.DB) *gorm.DB {
					return q.Where("class_room_masjid_id = ? AND class_room_id <> ? AND class_room_deleted_at IS NULL", masjidID, id)
				},
				50,
			)
			if err != nil {
				return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
			}
			m.ClassRoomSlug = &slug
		}
	}
	// === END NEW ===

	if err := ctl.DB.WithContext(reqCtx(c)).Save(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode/Slug ruang sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengubah data")
	}

	return helper.JsonUpdated(c, "Updated", dto.ToClassRoomResponse(m))
}

/* ============================ PATCH (alias Update) ============================ */

func (ctl *ClassRoomController) Patch(c *fiber.Ctx) error {
	// Gunakan payload yang sama dengan Update
	return ctl.Update(c)
}

/* ============================ DELETE ============================ */

func (ctl *ClassRoomController) Delete(c *fiber.Ctx) error {
	// Require DKM/Admin + resolve masjidID
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Pastikan tenant match & alive → soft delete
	tx := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NULL", id, masjidID).
		Update("class_room_deleted_at", time.Now())
	if tx.Error != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghapus data")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan / sudah terhapus")
	}
	return helper.JsonDeleted(c, "Deleted", fiber.Map{"deleted": true})
}

/* ============================ RESTORE ============================ */

func (ctl *ClassRoomController) Restore(c *fiber.Ctx) error {
	// Require DKM/Admin + resolve masjidID
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "ID tidak valid")
	}

	// Hanya bisa restore jika baris soft-deleted & tenant match
	tx := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NOT NULL", id, masjidID).
		Updates(map[string]interface{}{
			"class_room_deleted_at": nil,
			"class_room_updated_at": time.Now(),
		})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error) {
			// Restore bisa bentrok dengan partial unique (nama/kode/slug sudah dipakai baris alive lain)
			return helper.JsonError(c, fiber.StatusConflict, "Gagal restore: nama/kode/slug sudah dipakai entri lain")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal restore data")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan / tidak dalam keadaan terhapus")
	}

	// Return row terbaru
	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		// kalau gagal ambil ulang, minimal beri flag restored
		return helper.JsonOK(c, "Restored", fiber.Map{"restored": true})
	}
	return helper.JsonOK(c, "Restored", dto.ToClassRoomResponse(m))
}

/* =======================================================
   HELPERS (local)
   ======================================================= */

// Deteksi unique violation Postgres (kode "23505")
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint") || strings.Contains(s, "23505")
}
