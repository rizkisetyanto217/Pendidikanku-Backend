// file: internals/features/school/class_rooms/controller/class_room_controller.go
package controller

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	helper "masjidku_backend/internals/helpers"

	"masjidku_backend/internals/features/school/academics/rooms/dto"
	"masjidku_backend/internals/features/school/academics/rooms/model"
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

/* =========================
   Fiber → *http.Request (untuk pagination helper)
   ========================= */
func stdReqFromFiber(c *fiber.Ctx) *http.Request {
	u := &url.URL{RawQuery: string(c.Request().URI().QueryString())}
	return &http.Request{URL: u}
}

// ambil context standar (kalau Fiber mendukung UserContext)
func reqCtx(c *fiber.Ctx) context.Context {
	if uc := c.UserContext(); uc != nil {
		return uc
	}
	return context.Background()
}

/* =======================================================
   ROUTES
   ======================================================= */

func (ctl *ClassRoomController) List(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}

	// Filters legacy (search, is_active, dll) + legacy sort (q.Sort)
	var q dto.ListClassRoomsQuery
	if err := c.QueryParser(&q); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Query tidak valid")
	}
	q.Normalize()

	// Pagination & sorting via helper (default: created_at DESC)
	p := helper.ParseWith(stdReqFromFiber(c), "created_at", "desc", helper.AdminOpts)

	// Whitelist kolom sorting utk sort_by/order dari helper
	allowedSort := map[string]string{
		"name":       "class_rooms_name",
		"created_at": "class_rooms_created_at",
		"updated_at": "class_rooms_updated_at",
	}
	orderCol := allowedSort["created_at"]
	if col, ok := allowedSort[strings.ToLower(p.SortBy)]; ok {
		orderCol = col
	}
	orderDir := "DESC"
	if strings.ToLower(p.SortOrder) == "asc" {
		orderDir = "ASC"
	}

	// Override jika user pakai legacy q.Sort (name_asc|name_desc|created_asc|created_desc)
	if s := strings.ToLower(strings.TrimSpace(q.Sort)); s != "" {
		switch s {
		case "name_asc":
			orderCol, orderDir = "class_rooms_name", "ASC"
		case "name_desc":
			orderCol, orderDir = "class_rooms_name", "DESC"
		case "created_asc":
			orderCol, orderDir = "class_rooms_created_at", "ASC"
		case "created_desc":
			orderCol, orderDir = "class_rooms_created_at", "DESC"
		}
	}

	db := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", masjidID)

	// search → LIKE ke name + location + description (case-insensitive)
	if q.Search != "" {
		s := "%" + strings.ToLower(q.Search) + "%"
		db = db.Where(`
			LOWER(class_rooms_name) LIKE ?
			OR LOWER(COALESCE(class_rooms_location,'')) LIKE ?
			OR LOWER(COALESCE(class_rooms_description,'')) LIKE ?
		`, s, s, s)
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

	// Count total
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal menghitung data")
	}

	// Sorting & pagination (kolom di-whitelist di atas, aman untuk concat)
	db = db.Order(orderCol + " " + orderDir)
	if !p.All {
		db = db.Limit(p.Limit()).Offset(p.Offset())
	}

	// Query data
	var rows []model.ClassRoomModel
	if err := db.Find(&rows).Error; err != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengambil data")
	}

	out := make([]dto.ClassRoomResponse, 0, len(rows))
	for _, m := range rows {
		out = append(out, dto.ToClassRoomResponse(m))
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
}

func (ctl *ClassRoomController) GetByID(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).Where(
		"class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL",
		id, masjidID,
	).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengambil data")
	}

	return helper.JsonOK(c, "OK", dto.ToClassRoomResponse(m))
}

func (ctl *ClassRoomController) Create(c *fiber.Ctx) error {
	ctl.ensureValidator()

	masjidID, ok := mustMasjidID(c)
	if !ok {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}

	var req dto.CreateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()

	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	m := model.ClassRoomModel{
		ClassRoomsMasjidID:     masjidID,
		ClassRoomsName:         req.ClassRoomsName,
		ClassRoomsCode:         req.ClassRoomsCode,
		ClassRoomsLocation:     req.ClassRoomsLocation,
		ClassRoomsFloor:        req.ClassRoomsFloor,
		ClassRoomsCapacity:     req.ClassRoomsCapacity,
		ClassRoomsDescription:  req.ClassRoomsDescription, // NEW
		ClassRoomsIsVirtual:    req.ClassRoomsIsVirtual,
		ClassRoomsIsActive:     req.ClassRoomsIsActive,
		ClassRoomsFeatures:     req.ClassRoomsFeatures,
	}

	if err := ctl.DB.WithContext(reqCtx(c)).Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "Nama/Kode ruang sudah digunakan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal menyimpan data")
	}

	return helper.JsonCreated(c, "Created", dto.ToClassRoomResponse(m))
}

func (ctl *ClassRoomController) Update(c *fiber.Ctx) error {
	ctl.ensureValidator()

	masjidID, ok := mustMasjidID(c)
	if !ok {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.UpdateClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()
	if err := ctl.Validate.Struct(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Validasi gagal: "+err.Error())
	}

	// Ambil record yang masih alive & tenant match
	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengambil data")
	}

	// Apply perubahan hanya yang dikirim (gunakan nilai, bukan pointer)
	updates := map[string]interface{}{}
	if req.ClassRoomsName != nil {
		updates["class_rooms_name"] = *req.ClassRoomsName
	}
	if req.ClassRoomsCode != nil {
		updates["class_rooms_code"] = *req.ClassRoomsCode // empty string => disimpan "", bukan NULL
	}
	if req.ClassRoomsLocation != nil {
		updates["class_rooms_location"] = *req.ClassRoomsLocation
	}
	if req.ClassRoomsFloor != nil {
		updates["class_rooms_floor"] = *req.ClassRoomsFloor
	}
	if req.ClassRoomsCapacity != nil {
		updates["class_rooms_capacity"] = *req.ClassRoomsCapacity
	}
	if req.ClassRoomsDescription != nil { // NEW
		updates["class_rooms_description"] = *req.ClassRoomsDescription
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
		return helper.JsonError(c, http.StatusBadRequest, "Tidak ada field untuk diupdate")
	}

	if err := ctl.DB.WithContext(reqCtx(c)).Model(&m).Clauses(clause.Returning{}).Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "Nama/Kode ruang sudah digunakan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengubah data")
	}

	return helper.JsonUpdated(c, "Updated", dto.ToClassRoomResponse(m))
}

func (ctl *ClassRoomController) Patch(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	var req dto.PatchClassRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "Payload tidak valid")
	}
	req.Normalize()

	// Ambil record alive & tenant match
	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengambil data")
	}

	updates := req.BuildUpdateMap()
	if len(updates) == 0 {
		return helper.JsonError(c, http.StatusBadRequest, "Tidak ada field untuk diupdate")
	}
	updates["class_rooms_updated_at"] = time.Now()

	if err := ctl.DB.WithContext(reqCtx(c)).Model(&m).Clauses(clause.Returning{}).Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, http.StatusConflict, "Nama/Kode ruang sudah digunakan")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal mengubah data")
	}

	return helper.JsonUpdated(c, "Updated", dto.ToClassRoomResponse(m))
}

func (ctl *ClassRoomController) Delete(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	// Pastikan tenant match & alive → soft delete
	tx := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		Update("class_rooms_deleted_at", time.Now())
	if tx.Error != nil {
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal menghapus data")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan / sudah terhapus")
	}
	return helper.JsonDeleted(c, "Deleted", fiber.Map{"deleted": true})
}

func (ctl *ClassRoomController) Restore(c *fiber.Ctx) error {
	masjidID, ok := mustMasjidID(c)
	if !ok {
		return helper.JsonError(c, http.StatusUnauthorized, "Masjid scope tidak ditemukan")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return helper.JsonError(c, http.StatusBadRequest, "ID tidak valid")
	}

	// Hanya bisa restore jika baris soft-deleted & tenant match
	tx := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NOT NULL", id, masjidID).
		Updates(map[string]interface{}{
			"class_rooms_deleted_at": nil,
			"class_rooms_updated_at": time.Now(),
		})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error) {
			// Restore bisa bentrok dengan partial unique (nama/kode sudah dipakai baris alive lain)
			return helper.JsonError(c, http.StatusConflict, "Gagal restore: nama/kode sudah dipakai entri lain")
		}
		return helper.JsonError(c, http.StatusInternalServerError, "Gagal restore data")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, http.StatusNotFound, "Data tidak ditemukan / tidak dalam keadaan terhapus")
	}

	// Return row terbaru
	var m model.ClassRoomModel
	if err := ctl.DB.WithContext(reqCtx(c)).
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		// kalau gagal ambil ulang, minimal beri flag restored
		return helper.JsonOK(c, "Restored", fiber.Map{"restored": true})
	}
	return helper.JsonOK(c, "Restored", dto.ToClassRoomResponse(m))
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

// Deteksi unique violation Postgres (kode "23505")
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// string fallback (kompatibel untuk lib/pq & pgx yang dibungkus)
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint") || strings.Contains(s, "23505") {
		return true
	}
	return false
}
