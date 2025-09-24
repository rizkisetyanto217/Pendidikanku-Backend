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
	"gorm.io/gorm/clause"

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

	// Require DKM/Admin + resolve masjidID (slug/id)
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err
	}
	masjidID, err := helperAuth.EnsureMasjidAccessDKM(c, mc)
	if err != nil {
		return err
	}

	var req dto.CreateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()

	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// === AUTO SLUG (unik per masjid, case-insensitive, panjang <= 50) ===
	base := ""
	if req.ClassRoomSlug != nil {
		base = strings.TrimSpace(*req.ClassRoomSlug)
	}
	if base == "" {
		base = helper.Slugify(req.ClassRoomName, 50)
	} else {
		base = helper.Slugify(base, 50)
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

	m := model.ClassRoomModel{
		ClassRoomMasjidID:    masjidID,
		ClassRoomName:        req.ClassRoomName,
		ClassRoomCode:        req.ClassRoomCode,
		ClassRoomSlug:        &slug, // pakai slug hasil generate/unique
		ClassRoomLocation:    req.ClassRoomLocation,
		ClassRoomCapacity:    req.ClassRoomCapacity,
		ClassRoomDescription: req.ClassRoomDescription,
		ClassRoomIsVirtual:   req.ClassRoomIsVirtual,
		ClassRoomIsActive:    req.ClassRoomIsActive,
		ClassRoomFeatures:    req.ClassRoomFeatures,
	}

	if err := ctl.DB.WithContext(reqCtx(c)).Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode/Slug ruang sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menyimpan data")
	}

	return helper.JsonCreated(c, "Created", dto.ToClassRoomResponse(m))
}

/* ============================ UPDATE ============================ */

func (ctl *ClassRoomController) Update(c *fiber.Ctx) error {
	ctl.ensureValidator()

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

	var req dto.UpdateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// Ambil record yang masih alive & tenant match
	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Apply perubahan hanya yang dikirim (gunakan nilai, bukan pointer)
	updates := map[string]interface{}{}
	if req.ClassRoomName != nil {
		updates["class_room_name"] = *req.ClassRoomName
	}
	if req.ClassRoomCode != nil {
		updates["class_room_code"] = *req.ClassRoomCode
	}
	if req.ClassRoomSlug != nil {
		updates["class_room_slug"] = *req.ClassRoomSlug
	}
	if req.ClassRoomLocation != nil {
		updates["class_room_location"] = *req.ClassRoomLocation
	}
	if req.ClassRoomCapacity != nil {
		updates["class_room_capacity"] = *req.ClassRoomCapacity
	}
	if req.ClassRoomDescription != nil {
		updates["class_room_description"] = *req.ClassRoomDescription
	}
	if req.ClassRoomIsVirtual != nil {
		updates["class_room_is_virtual"] = *req.ClassRoomIsVirtual
	}
	if req.ClassRoomIsActive != nil {
		updates["class_room_is_active"] = *req.ClassRoomIsActive
	}
	if req.ClassRoomFeatures != nil {
		updates["class_room_features"] = *req.ClassRoomFeatures
	}
	updates["class_room_updated_at"] = time.Now()

	if len(updates) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Tidak ada field untuk diupdate")
	}

	if err := ctl.DB.WithContext(reqCtx(c)).Model(&m).Clauses(clause.Returning{}).Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode/Slug ruang sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengubah data")
	}

	return helper.JsonUpdated(c, "Updated", dto.ToClassRoomResponse(m))
}

/* ============================ PATCH ============================ */

func (ctl *ClassRoomController) Patch(c *fiber.Ctx) error {
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

	var req dto.PatchClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()

	// Ambil record alive & tenant match
	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_room_masjid_id = ? AND class_room_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	updates := req.BuildUpdateMap()
	if len(updates) == 0 {
		return helper.JsonError(c, fiber.StatusBadRequest, "Tidak ada field untuk diupdate")
	}
	updates["class_room_updated_at"] = time.Now()

	if err := ctl.DB.WithContext(reqCtx(c)).Model(&m).Clauses(clause.Returning{}).Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode/Slug ruang sudah digunakan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengubah data")
	}

	return helper.JsonUpdated(c, "Updated", dto.ToClassRoomResponse(m))
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
