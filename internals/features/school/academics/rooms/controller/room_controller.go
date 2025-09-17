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

// ambil context standar (kalau Fiber mendukung UserContext)
func reqCtx(c *fiber.Ctx) context.Context {
	if uc := c.UserContext(); uc != nil {
		return uc
	}
	return context.Background()
}

/* ============================ LIST ============================ */

func (ctl *ClassRoomController) List(c *fiber.Ctx) error {
	// Resolve konteks masjid (path/header/cookie/query/host/token)
	mc, err := helperAuth.ResolveMasjidContext(c)
	if err != nil {
		return err // fiber.Error dari resolver
	}

	// Dapatkan masjidID (slug→id jika perlu)
	var masjidID uuid.UUID
	if mc.ID != uuid.Nil {
		masjidID = mc.ID
	} else {
		id, er := helperAuth.GetMasjidIDBySlug(c, mc.Slug)
		if er != nil {
			if errors.Is(er, gorm.ErrRecordNotFound) {
				return fiber.NewError(fiber.StatusNotFound, "Masjid (slug) tidak ditemukan")
			}
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal resolve masjid")
		}
		masjidID = id
	}

	// Authorization: read = member OR DKM/Admin
	if err := helperAuth.EnsureDKMMasjid(c, masjidID); err != nil {
		// bukan DKM/Admin → cek membership
		if !helperAuth.UserHasMasjid(c, masjidID) {
			return fiber.NewError(fiber.StatusForbidden, "Anda tidak terdaftar pada masjid ini (membership).")
		}
	}

	// parse qparams legacy
	var q dto.ListClassRoomsQuery
	if err := c.QueryParser(&q); err != nil {
	 return helper.JsonError(c, fiber.StatusBadRequest, "Query tidak valid")
	}
	q.Normalize()

	// include flags (toleran)
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	withSections := strings.EqualFold(strings.TrimSpace(c.Query("with_sections")), "1") ||
		strings.EqualFold(strings.TrimSpace(c.Query("with_sections")), "true")
	if !withSections && includeStr != "" {
		for _, tok := range strings.Split(includeStr, ",") {
			switch strings.TrimSpace(tok) {
			case "sections", "section", "all":
				withSections = true
			}
		}
	}

	// sort & paging
	p := helper.ParseFiber(c, "created_at", "desc", helper.AdminOpts)
	allowedSort := map[string]string{
		"name":       "class_rooms_name",
		"slug":       "class_rooms_slug",
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
	// legacy override
	switch strings.ToLower(strings.TrimSpace(q.Sort)) {
	case "name_asc":
		orderCol, orderDir = "class_rooms_name", "ASC"
	case "name_desc":
		orderCol, orderDir = "class_rooms_name", "DESC"
	case "slug_asc":
		orderCol, orderDir = "class_rooms_slug", "ASC"
	case "slug_desc":
		orderCol, orderDir = "class_rooms_slug", "DESC"
	case "created_asc":
		orderCol, orderDir = "class_rooms_created_at", "ASC"
	case "created_desc":
		orderCol, orderDir = "class_rooms_created_at", "DESC"
	}

	db := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", masjidID)

	// filter by id
	if roomID := strings.TrimSpace(c.Query("class_room_id", c.Query("id"))); roomID != "" {
		id, err := uuid.Parse(roomID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_room_id tidak valid")
		}
		db = db.Where("class_room_id = ?", id)
	}

	// filter by slug (support kedua nama param)
	if slug := strings.TrimSpace(c.Query("class_rooms_slug", c.Query("slug"))); slug != "" {
		db = db.Where("LOWER(class_rooms_slug) = LOWER(?)", slug)
	}

	// search & flags
	if q.Search != "" {
		s := "%" + strings.ToLower(q.Search) + "%"
		db = db.Where(`
			LOWER(class_rooms_name) LIKE ?
			OR LOWER(COALESCE(class_rooms_code,'')) LIKE ?
			OR LOWER(COALESCE(class_rooms_slug,'')) LIKE ?
			OR LOWER(COALESCE(class_rooms_location,'')) LIKE ?
			OR LOWER(COALESCE(class_rooms_description,'')) LIKE ?
		`, s, s, s, s, s)
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

	// count
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal menghitung data")
	}

	// order & paging
	db = db.Order(orderCol + " " + orderDir)
	if !p.All {
		db = db.Limit(p.Limit()).Offset(p.Offset())
	}

	// fetch rooms
	var rooms []model.ClassRoomModel
	if err := db.Find(&rooms).Error; err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// payload
	type sectionLite struct {
		ID            uuid.UUID  `json:"id"`
		ClassID       uuid.UUID  `json:"class_id"`
		TeacherID     *uuid.UUID `json:"teacher_id"`
		ClassRoomID   *uuid.UUID `json:"class_room_id"`
		Slug          string     `json:"slug"`
		Name          string     `json:"name"`
		Code          *string    `json:"code"`
		Schedule      *string    `json:"schedule"`
		Capacity      *int       `json:"capacity"`
		TotalStudents int        `json:"total_students"`
		GroupURL      *string    `json:"group_url"`
		IsActive      bool       `json:"is_active"`
		CreatedAt     time.Time  `json:"created_at"`
		UpdatedAt     time.Time  `json:"updated_at"`
	}
	type roomWithExpand struct {
		dto.ClassRoomResponse
		Sections      []sectionLite `json:"sections,omitempty"`
		SectionsCount *int          `json:"sections_count,omitempty"`
	}

	out := make([]roomWithExpand, 0, len(rooms))
	for _, m := range rooms {
		out = append(out, roomWithExpand{ClassRoomResponse: dto.ToClassRoomResponse(m)})
	}

	// include sections (batch)
	if withSections && len(rooms) > 0 {
		roomIDs := make([]uuid.UUID, 0, len(rooms))
		indexByID := make(map[uuid.UUID]int, len(rooms))
		for i := range rooms {
			roomIDs = append(roomIDs, rooms[i].ClassRoomID)
			indexByID[rooms[i].ClassRoomID] = i
		}

		var secs []sectionLite
		if err := ctl.DB.WithContext(reqCtx(c)).
			Table("class_sections").
			Select(`
				class_sections_id               AS id,
				class_sections_class_id         AS class_id,
				class_sections_teacher_id       AS teacher_id,
				class_sections_class_room_id    AS class_room_id,
				class_sections_slug             AS slug,
				class_sections_name             AS name,
				class_sections_code             AS code,
				class_sections_schedule         AS schedule,
				class_sections_capacity         AS capacity,
				class_sections_total_students   AS total_students,
				class_sections_group_url        AS group_url,
				class_sections_is_active        AS is_active,
				class_sections_created_at       AS created_at,
				class_sections_updated_at       AS updated_at
			`).
			Where("class_sections_masjid_id = ? AND class_sections_deleted_at IS NULL", masjidID).
			Where("class_sections_class_room_id IN ?", roomIDs).
			Order("class_sections_created_at DESC").
			Scan(&secs).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil class sections")
		}

		if len(secs) > 0 {
			byRoom := make(map[uuid.UUID][]sectionLite, len(roomIDs))
			for i := range secs {
				if secs[i].ClassRoomID == nil || *secs[i].ClassRoomID == uuid.Nil {
					continue
				}
				byRoom[*secs[i].ClassRoomID] = append(byRoom[*secs[i].ClassRoomID], secs[i])
			}
			for rid, arr := range byRoom {
				if idx, ok := indexByID[rid]; ok {
					out[idx].Sections = arr
					n := len(arr)
					out[idx].SectionsCount = &n
				}
			}
		}
	}

	meta := helper.BuildMeta(total, p)
	return helper.JsonList(c, out, meta)
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

	// === AUTO SLUG (unik per masjid, CI, panjang <= 50) ===
	base := ""
	if req.ClassRoomsSlug != nil {
		base = strings.TrimSpace(*req.ClassRoomsSlug)
	}
	if base == "" {
		base = helper.Slugify(req.ClassRoomsName, 50)
	} else {
		base = helper.Slugify(base, 50)
	}
	slug, err := helper.EnsureUniqueSlugCI(
		reqCtx(c), ctl.DB,
		"class_rooms", "class_rooms_slug",
		base,
		func(q *gorm.DB) *gorm.DB {
			return q.Where("class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", masjidID)
		},
		50,
	)
	if err != nil {
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal membuat slug unik")
	}

	m := model.ClassRoomModel{
		ClassRoomsMasjidID:    masjidID,
		ClassRoomsName:        req.ClassRoomsName,
		ClassRoomsCode:        req.ClassRoomsCode,
		ClassRoomsSlug:        &slug, // ← pakai slug hasil generate/unique
		ClassRoomsLocation:    req.ClassRoomsLocation,
		ClassRoomsCapacity:    req.ClassRoomsCapacity,
		ClassRoomsDescription: req.ClassRoomsDescription,
		ClassRoomsIsVirtual:   req.ClassRoomsIsVirtual,
		ClassRoomsIsActive:    req.ClassRoomsIsActive,
		ClassRoomsFeatures:    req.ClassRoomsFeatures,
	}

	if err := ctl.DB.WithContext(reqCtx(c)).Create(&m).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode ruang sudah digunakan")
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
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data")
	}

	// Apply perubahan hanya yang dikirim (gunakan nilai, bukan pointer)
	updates := map[string]interface{}{}
	if req.ClassRoomsName != nil {
		updates["class_rooms_name"] = *req.ClassRoomsName
	}
	if req.ClassRoomsCode != nil {
		updates["class_rooms_code"] = *req.ClassRoomsCode
	}
	if req.ClassRoomsSlug != nil {
		updates["class_rooms_slug"] = *req.ClassRoomsSlug
	}
	if req.ClassRoomsLocation != nil {
		updates["class_rooms_location"] = *req.ClassRoomsLocation
	}
	if req.ClassRoomsCapacity != nil {
		updates["class_rooms_capacity"] = *req.ClassRoomsCapacity
	}
	if req.ClassRoomsDescription != nil {
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
		return helper.JsonError(c, fiber.StatusBadRequest, "Tidak ada field untuk diupdate")
	}

	if err := ctl.DB.WithContext(reqCtx(c)).Model(&m).Clauses(clause.Returning{}).Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode ruang sudah digunakan")
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
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
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
	updates["class_rooms_updated_at"] = time.Now()

	if err := ctl.DB.WithContext(reqCtx(c)).Model(&m).Clauses(clause.Returning{}).Updates(updates).Error; err != nil {
		if isUniqueViolation(err) {
			return helper.JsonError(c, fiber.StatusConflict, "Nama/Kode ruang sudah digunakan")
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
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NULL", id, masjidID).
		Update("class_rooms_deleted_at", time.Now())
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
		Where("class_room_id = ? AND class_rooms_masjid_id = ? AND class_rooms_deleted_at IS NOT NULL", id, masjidID).
		Updates(map[string]interface{}{
			"class_rooms_deleted_at": nil,
			"class_rooms_updated_at": time.Now(),
		})
	if tx.Error != nil {
		if isUniqueViolation(tx.Error) {
			// Restore bisa bentrok dengan partial unique (nama/kode sudah dipakai baris alive lain)
			return helper.JsonError(c, fiber.StatusConflict, "Gagal restore: nama/kode sudah dipakai entri lain")
		}
		return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal restore data")
	}
	if tx.RowsAffected == 0 {
		return helper.JsonError(c, fiber.StatusNotFound, "Data tidak ditemukan / tidak dalam keadaan terhapus")
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
