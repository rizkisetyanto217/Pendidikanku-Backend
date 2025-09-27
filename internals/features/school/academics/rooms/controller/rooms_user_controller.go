// file: internals/features/school/class_rooms/controller/class_room_controller.go
package controller

import (
	"errors"
	dto "masjidku_backend/internals/features/school/academics/rooms/dto"
	model "masjidku_backend/internals/features/school/academics/rooms/model"
	"strconv"
	"strings"
	"time"

	helper "masjidku_backend/internals/helpers"
	helperAuth "masjidku_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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

	// ===== Parse query params =====
	search := strings.TrimSpace(c.Query("search"))
	sortParam := strings.ToLower(strings.TrimSpace(c.Query("sort")))
	isActivePtr := parseBoolPtr(c.Query("is_active"))
	isVirtualPtr := parseBoolPtr(c.Query("is_virtual"))
	hasCodeOnly := parseBoolTrue(c.Query("has_code_only"))

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
		"name":       "class_room_name",
		"slug":       "class_room_slug",
		"created_at": "class_room_created_at",
		"updated_at": "class_room_updated_at",
	}
	orderCol := allowedSort["created_at"]
	if col, ok := allowedSort[strings.ToLower(p.SortBy)]; ok {
		orderCol = col
	}
	orderDir := "DESC"
	if strings.ToLower(p.SortOrder) == "asc" {
		orderDir = "ASC"
	}

	// legacy override via ?sort=
	switch sortParam {
	case "name_asc":
		orderCol, orderDir = "class_room_name", "ASC"
	case "name_desc":
		orderCol, orderDir = "class_room_name", "DESC"
	case "slug_asc":
		orderCol, orderDir = "class_room_slug", "ASC"
	case "slug_desc":
		orderCol, orderDir = "class_room_slug", "DESC"
	case "created_asc":
		orderCol, orderDir = "class_room_created_at", "ASC"
	case "created_desc":
		orderCol, orderDir = "class_room_created_at", "DESC"
	}

	db := ctl.DB.WithContext(reqCtx(c)).Model(&model.ClassRoomModel{}).
		Where("class_room_masjid_id = ? AND class_room_deleted_at IS NULL", masjidID)

	// filter by id
	if roomID := strings.TrimSpace(c.Query("class_room_id", c.Query("id"))); roomID != "" {
		id, err := uuid.Parse(roomID)
		if err != nil {
			return helper.JsonError(c, fiber.StatusBadRequest, "class_room_id tidak valid")
		}
		db = db.Where("class_room_id = ?", id)
	}

	// filter by slug (support kedua nama param)
	if slug := strings.TrimSpace(c.Query("class_room_slug", c.Query("class_rooms_slug", c.Query("slug")))); slug != "" {
		db = db.Where("LOWER(class_room_slug) = LOWER(?)", slug)
	}

	// search & flags
	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		db = db.Where(`
			LOWER(class_room_name) LIKE ?
			OR LOWER(COALESCE(class_room_code,'')) LIKE ?
			OR LOWER(COALESCE(class_room_slug,'')) LIKE ?
			OR LOWER(COALESCE(class_room_location,'')) LIKE ?
			OR LOWER(COALESCE(class_room_description,'')) LIKE ?
		`, s, s, s, s, s)
	}
	if isActivePtr != nil {
		db = db.Where("class_room_is_active = ?", *isActivePtr)
	}
	if isVirtualPtr != nil {
		db = db.Where("class_room_is_virtual = ?", *isVirtualPtr)
	}
	if hasCodeOnly {
		db = db.Where("class_room_code IS NOT NULL AND length(trim(class_room_code)) > 0")
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

/* ============================ helpers (local) ============================ */

func parseBoolPtr(v string) *bool {
	s := strings.TrimSpace(strings.ToLower(v))
	if s == "" {
		return nil
	}
	// true-ish
	if s == "1" || s == "true" || s == "yes" || s == "y" || s == "on" {
		b := true
		return &b
	}
	// false-ish
	if s == "0" || s == "false" || s == "no" || s == "n" || s == "off" {
		b := false
		return &b
	}
	// fallback: try parse
	if b, err := strconv.ParseBool(s); err == nil {
		return &b
	}
	return nil
}

func parseBoolTrue(v string) bool {
	if b := parseBoolPtr(v); b != nil {
		return *b
	}
	return false
}
