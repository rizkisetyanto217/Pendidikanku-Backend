// file: internals/features/school/class_rooms/controller/class_room_controller.go
package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"masjidku_backend/internals/features/school/schedule_daily_rooms/rooms/dto"
	"masjidku_backend/internals/features/school/schedule_daily_rooms/rooms/model"
)

/* =======================================================
   CONTROLLER
   ======================================================= */

type ClassRoomController struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewClassRoomController(db *gorm.DB, v *validator.Validate) *ClassRoomController {
	return &ClassRoomController{DB: db, Validate: v}
}

/* =======================================================
   ROUTES
   ======================================================= */
// Contoh mount:
//   func RegisterClassRoomRoutes(r fiber.Router, db *gorm.DB, v *validator.Validate) {
//       ctl := NewClassRoomController(db, v)
//       g := r.Group("/class-rooms")
//       g.Get("/", ctl.List)
//       g.Get("/:id", ctl.GetByID)
//       g.Post("/", ctl.Create)
//       g.Put("/:id", ctl.Update)
//       g.Patch("/:id", ctl.Patch)
//       g.Delete("/:id", ctl.Delete)      // soft delete
//       g.Post("/:id/restore", ctl.Restore)
//   }

func (ctl *ClassRoomController) List(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(errResp("Masjid scope tidak ditemukan"))
	}

	var q dto.ListClassRoomsQuery
	if err := c.QueryParser(&q); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errResp("Query tidak valid"))
	}
	q.Normalize()

	db := ctl.DB.Model(&model.ClassRoomModel{}).
		Where("class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", masjidID)

	// search → ILIKE ke name + location
	if q.Search != "" {
		s := "%" + strings.ToLower(q.Search) + "%"
		db = db.Where("(LOWER(class_rooms_name) LIKE ? OR LOWER(COALESCE(class_rooms_location,'')) LIKE ?)", s, s)
	}

	if q.IsActive != nil {
		db = db.Where("class_rooms_is_active = ?", *q.IsActive)
	}
	if q.IsVirtual != nil {
		db = db.Where("class_rooms_is_virtual = ?", *q.IsVirtual)
	}
	if q.HasCodeOnly != nil && *q.HasCodeOnly {
		db = db.Where("class_rooms_code IS NOT NULL AND length(trim(class_rooms_code)) > 0")
	}

	// sorting sederhana
	switch q.Sort {
	case "name_asc":
		db = db.Order("class_rooms_name ASC")
	case "name_desc":
		db = db.Order("class_rooms_name DESC")
	case "created_asc":
		db = db.Order("class_rooms_created_at ASC")
	case "created_desc", "":
		db = db.Order("class_rooms_created_at DESC")
	default:
		db = db.Order("class_rooms_created_at DESC")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal menghitung data"))
	}

	var rows []model.ClassRoomModel
	if err := db.Limit(q.Limit).Offset(q.Offset).Find(&rows).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal mengambil data"))
	}

	out := make([]dto.ClassRoomResponse, 0, len(rows))
	for _, m := range rows {
		out = append(out, dto.ToClassRoomResponse(m))
	}

	return c.JSON(okResp(fiber.Map{
		"items": out,
		"total": total,
		"limit": q.Limit,
		"offset": q.Offset,
	}))
}

func (ctl *ClassRoomController) GetByID(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(errResp("Masjid scope tidak ditemukan"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errResp("ID tidak valid"))
	}

	var m model.ClassRoomModel
	if err := ctl.DB.Where(
		"class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL",
		id, masjidID,
	).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(http.StatusNotFound).JSON(errResp("Data tidak ditemukan"))
		}
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal mengambil data"))
	}

	return c.JSON(okResp(dto.ToClassRoomResponse(m)))
}

func (ctl *ClassRoomController) Create(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(errResp("Masjid scope tidak ditemukan"))
	}

	var req dto.CreateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errResp("Payload tidak valid"))
	}
	req.Normalize()

	if err := ctl.Validate.Struct(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errValid(err))
	}

	m := model.ClassRoomModel{
		ClassRoomsMasjidID:  masjidID,
		ClassRoomsName:      req.ClassRoomsName,
		ClassRoomsCode:      req.ClassRoomsCode,
		ClassRoomsLocation:  req.ClassRoomsLocation,
		ClassRoomsFloor:     req.ClassRoomsFloor,
		ClassRoomsCapacity:  req.ClassRoomsCapacity,
		ClassRoomsIsVirtual: req.ClassRoomsIsVirtual,
		ClassRoomsIsActive:  req.ClassRoomsIsActive,
		ClassRoomsFeatures:  req.ClassRoomsFeatures,
	}

	if err := ctl.DB.Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return c.Status(http.StatusConflict).JSON(errResp("Nama/Kode ruang sudah digunakan"))
		}
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal menyimpan data"))
	}

	return c.Status(http.StatusCreated).JSON(okResp(dto.ToClassRoomResponse(m)))
}

func (ctl *ClassRoomController) Update(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(errResp("Masjid scope tidak ditemukan"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errResp("ID tidak valid"))
	}

	var req dto.UpdateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errResp("Payload tidak valid"))
	}
	req.Normalize()
	if err := ctl.Validate.Struct(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errValid(err))
	}

	// Ambil record yang masih alive & tenant match
	var m model.ClassRoomModel
	if err := ctl.DB.
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(http.StatusNotFound).JSON(errResp("Data tidak ditemukan"))
		}
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal mengambil data"))
	}

	// Apply perubahan hanya yang dikirim
	updates := map[string]interface{}{}
	if req.ClassRoomsName != nil {
		updates["class_rooms_name"] = *req.ClassRoomsName
	}
	if req.ClassRoomsCode != nil {
		// kosongkan string → tetap string kosong (bukan NULL)
		updates["class_rooms_code"] = req.ClassRoomsCode
	}
	if req.ClassRoomsLocation != nil {
		updates["class_rooms_location"] = req.ClassRoomsLocation
	}
	if req.ClassRoomsFloor != nil {
		updates["class_rooms_floor"] = req.ClassRoomsFloor
	}
	if req.ClassRoomsCapacity != nil {
		updates["class_rooms_capacity"] = req.ClassRoomsCapacity
	}
	if req.ClassRoomsIsVirtual != nil {
		updates["class_rooms_is_virtual"] = *req.ClassRoomsIsVirtual
	}
	if req.ClassRoomsIsActive != nil {
		updates["class_rooms_is_active"] = *req.ClassRoomsIsActive
	}
	if req.ClassRoomsFeatures != nil {
		updates["class_rooms_features"] = *req.ClassRoomsFeatures
	}
	updates["class_rooms_updated_at"] = time.Now()

	if len(updates) == 0 {
		return c.Status(http.StatusBadRequest).JSON(errResp("Tidak ada field untuk diupdate"))
	}

	if err := ctl.DB.Model(&m).Clauses(clause.Returning{}).Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return c.Status(http.StatusConflict).JSON(errResp("Nama/Kode ruang sudah digunakan"))
		}
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal mengubah data"))
	}

	return c.JSON(okResp(dto.ToClassRoomResponse(m)))
}

func (ctl *ClassRoomController) Patch(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(errResp("Masjid scope tidak ditemukan"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errResp("ID tidak valid"))
	}

	var req dto.PatchClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(errResp("Payload tidak valid"))
	}
	req.Normalize()

	// Ambil record alive & tenant match
	var m model.ClassRoomModel
	if err := ctl.DB.
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(http.StatusNotFound).JSON(errResp("Data tidak ditemukan"))
		}
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal mengambil data"))
	}

	updates := req.BuildUpdateMap()
	if len(updates) == 0 {
		return c.Status(http.StatusBadRequest).JSON(errResp("Tidak ada field untuk diupdate"))
	}
	updates["class_rooms_updated_at"] = time.Now()

	if err := ctl.DB.Model(&m).Clauses(clause.Returning{}).Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return c.Status(http.StatusConflict).JSON(errResp("Nama/Kode ruang sudah digunakan"))
		}
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal mengubah data"))
	}

	return c.JSON(okResp(dto.ToClassRoomResponse(m)))
}

func (ctl *ClassRoomController) Delete(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(errResp("Masjid scope tidak ditemukan"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errResp("ID tidak valid"))
	}

	// Pastikan tenant match
	tx := ctl.DB.Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		Update("class_rooms_deleted_at", time.Now())
	if tx.Error != nil {
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal menghapus data"))
	}
	if tx.RowsAffected == 0 {
		return c.Status(http.StatusNotFound).JSON(errResp("Data tidak ditemukan / sudah terhapus"))
	}
	return c.JSON(okResp(fiber.Map{"deleted": true}))
}

func (ctl *ClassRoomController) Restore(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(errResp("Masjid scope tidak ditemukan"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(errResp("ID tidak valid"))
	}

	// Hanya bisa restore jika baris soft-deleted & tenant match
	tx := ctl.DB.Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NOT NULL", id, masjidID).
		Updates(map[string]interface{}{
			"class_rooms_deleted_at": nil,
			"class_rooms_updated_at": time.Now(),
		})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error) {
			// Restore bisa bentrok dengan partial unique (nama/kode sudah dipakai baris alive lain)
			return c.Status(http.StatusConflict).JSON(errResp("Gagal restore: nama/kode sudah dipakai entri lain"))
		}
		return c.Status(http.StatusInternalServerError).JSON(errResp("Gagal restore data"))
	}
	if tx.RowsAffected == 0 {
		return c.Status(http.StatusNotFound).JSON(errResp("Data tidak ditemukan / tidak dalam keadaan terhapus"))
	}

	// Return row terbaru
	var m model.ClassRoomModel
	if err := ctl.DB.
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		return c.Status(http.StatusOK).JSON(okResp(fiber.Map{"restored": true}))
	}
	return c.JSON(okResp(dto.ToClassRoomResponse(m)))
}

/* =======================================================
   HELPERS
   ======================================================= */

func mustMasjidID(c *fiber.Ctx) (uuid.UUID, bool) {
	v := c.Locals("masjid_id")
	if v == nil {
		return uuid.Nil, false
	}
	switch t := v.(type) {
	case string:
		id, err := uuid.Parse(strings.TrimSpace(t))
		if err != nil {
			return uuid.Nil, false
		}
		return id, true
	case uuid.UUID:
		return t, true
	default:
		return uuid.Nil, false
	}
}

// Sederhana: adaptasikan ke helper error kamu kalau ada
func okResp(data interface{}) fiber.Map {
	return fiber.Map{"code": 200, "status": "success", "data": data}
}
func errResp(msg string) fiber.Map {
	return fiber.Map{"code": 400, "status": "error", "message": msg}
}
func errValid(err error) fiber.Map {
	return fiber.Map{"code": 422, "status": "error", "message": "Validasi gagal", "details": err.Error()}
}

// Deteksi unique violation Postgres (kode "23505")
func isUniqueViolation(err error) bool {
	// tanpa import pgx/pgconn biar portable: cek substring
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint")
}
