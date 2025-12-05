// file: internals/features/school/class_rooms/controller/class_room_controller.go
package controller

import (
	"strconv"
	"strings"
	"time"

	dto "madinahsalam_backend/internals/features/school/academics/rooms/dto"
	model "madinahsalam_backend/internals/features/school/academics/rooms/model"

	helper "madinahsalam_backend/internals/helpers"
	helperAuth "madinahsalam_backend/internals/helpers/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

/* ============================ LIST ============================ */
// file: internals/features/school/class_rooms/controller/class_room_controller.go

/* ============================ LIST ============================ */
func (ctl *ClassRoomController) List(c *fiber.Ctx) error {
	// Kalau helper lain butuh DB dari Locals
	c.Locals("DB", ctl.DB)

	// =====================================================
	// 1) Tentukan schoolID:
	//    Sekarang: WAJIB dari token, bukan dari slug/path
	// =====================================================

	schoolID, err := helperAuth.GetSchoolIDFromToken(c)
	if err != nil || schoolID == uuid.Nil {
		return helper.JsonError(c, fiber.StatusBadRequest, "School context tidak ditemukan di token")
	}

	// =====================================================
	// 2) Authorization: HANYA DKM/Admin school ini
	// =====================================================
	if err := helperAuth.EnsureDKMSchool(c, schoolID); err != nil {
		return err
	}

	// ===== Parse query params =====
	search := strings.TrimSpace(c.Query("q"))
	name := strings.TrimSpace(c.Query("name")) // 🔍 filter khusus by room name
	sortParam := strings.ToLower(strings.TrimSpace(c.Query("sort")))
	isActivePtr := parseBoolPtr(c.Query("is_active"))
	isVirtualPtr := parseBoolPtr(c.Query("is_virtual"))
	hasCodeOnly := parseBoolTrue(c.Query("has_code_only"))

	// =====================================================
	// 3) INCLUDE flags: ?include=class_sections,csst
	//    - legacy: ?with_sections=true masih didukung
	// =====================================================
	includeStr := strings.ToLower(strings.TrimSpace(c.Query("include")))
	includeClassSections := false
	includeCSST := false

	// legacy flag
	if parseBoolTrue(c.Query("with_sections")) {
		includeClassSections = true
	}

	if includeStr != "" {
		for _, tok := range strings.Split(includeStr, ",") {
			t := strings.TrimSpace(tok)
			switch t {
			case "class_sections", "sections", "section":
				includeClassSections = true
			case "csst", "class_section_subject_teachers", "class_section_subject_teacher":
				includeCSST = true
			case "all":
				includeClassSections = true
				includeCSST = true
			}
		}
	}

	// ===== sort & paging =====
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
		Where("class_room_school_id = ? AND class_room_deleted_at IS NULL", schoolID)

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

	// 🔍 filter spesifik by room name: ?name=
	if name != "" {
		s := "%" + strings.ToLower(name) + "%"
		db = db.Where("LOWER(class_room_name) LIKE ?", s)
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

	// Kalau tidak ada data sama sekali → langsung balikin kosong (tanpa include)
	if len(rooms) == 0 {
		pg := helper.BuildPaginationFromOffset(total, p.Offset(), p.Limit())
		return helper.JsonList(c, "ok", []any{}, pg)
	}

	// =========================================================
	// Payload utama (data) + include
	// =========================================================

	type sectionLite struct {
		ID            uuid.UUID  `json:"id"`
		ClassID       *uuid.UUID `json:"class_id,omitempty"`
		TeacherID     *uuid.UUID `json:"teacher_id,omitempty"`
		ClassRoomID   *uuid.UUID `json:"class_room_id,omitempty"`
		Slug          string     `json:"slug"`
		Name          string     `json:"name"`
		Code          *string    `json:"code,omitempty"`
		Schedule      *string    `json:"schedule,omitempty"`
		Capacity      *int       `json:"capacity,omitempty"`
		TotalStudents int        `json:"total_students"`
		GroupURL      *string    `json:"group_url,omitempty"`
		IsActive      bool       `json:"is_active"`
		CreatedAt     time.Time  `json:"created_at"`
		UpdatedAt     time.Time  `json:"updated_at"`
	}

	type csstLite struct {
		ID              uuid.UUID  `json:"id"`
		ClassSectionID  *uuid.UUID `json:"class_section_id,omitempty"`
		SchoolTeacherID *uuid.UUID `json:"school_teacher_id,omitempty"`
		ClassRoomID     *uuid.UUID `json:"class_room_id,omitempty"`
		Role            *string    `json:"role,omitempty"`
		IsActive        bool       `json:"is_active"`
		CreatedAt       time.Time  `json:"created_at"`
		UpdatedAt       time.Time  `json:"updated_at"`
	}

	// room utama + optional count
	type roomWithCounts struct {
		dto.ClassRoomResponse
		ClassSectionsCount *int `json:"class_sections_count,omitempty"`
		// bisa ditambah nanti: CSSTCount *int `json:"csst_count,omitempty"`
	}

	out := make([]roomWithCounts, 0, len(rooms))
	for _, m := range rooms {
		out = append(out, roomWithCounts{
			ClassRoomResponse: dto.ToClassRoomResponse(m),
		})
	}

	// Precompute roomIDs
	roomIDs := make([]uuid.UUID, 0, len(rooms))
	for i := range rooms {
		roomIDs = append(roomIDs, rooms[i].ClassRoomID)
	}

	// include map: akan dikirim di "include": { ... }
	includeMap := fiber.Map{}

	// =========================================================
	// INCLUDE: class_sections (flat array) → include.class_sections
	// =========================================================
	if includeClassSections && len(roomIDs) > 0 {
		indexByID := make(map[uuid.UUID]int, len(rooms))
		for i := range rooms {
			indexByID[rooms[i].ClassRoomID] = i
		}

		var secs []sectionLite
		if err := ctl.DB.WithContext(reqCtx(c)).
			Table("class_sections").
			Select(`
				class_section_id                AS id,
				class_section_class_id          AS class_id,
				class_section_school_teacher_id AS teacher_id,
				class_section_class_room_id     AS class_room_id,
				class_section_slug              AS slug,
				class_section_name              AS name,
				class_section_code              AS code,
				class_section_schedule          AS schedule,
				class_section_quota_total          AS capacity,
				class_section_total_students_active    AS total_students,
				class_section_group_url         AS group_url,
				class_section_is_active         AS is_active,
				class_section_created_at        AS created_at,
				class_section_updated_at        AS updated_at
			`).
			Where("class_section_school_id = ? AND class_section_deleted_at IS NULL", schoolID).
			Where("class_section_class_room_id IN ?", roomIDs).
			Order("class_section_created_at DESC").
			Scan(&secs).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil class sections")
		}

		if len(secs) > 0 {
			// hitung count per room di payload utama
			byRoom := make(map[uuid.UUID][]sectionLite, len(roomIDs))
			for i := range secs {
				if secs[i].ClassRoomID == nil || *secs[i].ClassRoomID == uuid.Nil {
					continue
				}
				byRoom[*secs[i].ClassRoomID] = append(byRoom[*secs[i].ClassRoomID], secs[i])
			}
			for rid, arr := range byRoom {
				if idx, ok := indexByID[rid]; ok {
					n := len(arr)
					out[idx].ClassSectionsCount = &n
				}
			}

			includeMap["class_sections"] = secs
		} else {
			includeMap["class_sections"] = []sectionLite{}
		}
	}

	// =========================================================
	// INCLUDE: CSST (class_section_subject_teachers) → include.csst
	// =========================================================
	if includeCSST && len(roomIDs) > 0 {
		var cssts []csstLite
		if err := ctl.DB.WithContext(reqCtx(c)).
			Table("class_section_subject_teachers").
			Select(`
				class_section_subject_teacher_id                  AS id,
				class_section_subject_teacher_class_section_id    AS class_section_id,
				class_section_subject_teacher_school_teacher_id   AS school_teacher_id,
				class_section_subject_teacher_class_room_id       AS class_room_id,
				class_section_subject_teacher_role                AS role,
				class_section_subject_teacher_is_active           AS is_active,
				class_section_subject_teacher_created_at          AS created_at,
				class_section_subject_teacher_updated_at          AS updated_at
			`).
			Where("class_section_subject_teacher_school_id = ? AND class_section_subject_teacher_deleted_at IS NULL", schoolID).
			Where("class_section_subject_teacher_class_room_id IN ?", roomIDs).
			Order("class_section_subject_teacher_created_at DESC").
			Scan(&cssts).Error; err != nil {
			return helper.JsonError(c, fiber.StatusInternalServerError, "Gagal mengambil data CSST")
		}

		if len(cssts) > 0 {
			includeMap["csst"] = cssts
		} else {
			includeMap["csst"] = []csstLite{}
		}
	}

	// 🔹 Pagination final
	pg := helper.BuildPaginationFromOffset(total, p.Offset(), p.Limit())

	// 🔹 Response final:
	// - kalau ada sesuatu di includeMap → JsonListWithInclude (pakai key "include")
	// - kalau tidak, JsonList biasa
	if len(includeMap) > 0 {
		return helper.JsonListWithInclude(c, "ok", out, includeMap, pg)
	}

	return helper.JsonList(c, "ok", out, pg)
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
