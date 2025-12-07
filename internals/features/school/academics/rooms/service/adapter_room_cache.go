// file: internals/features/school/academics/rooms/service/room_cache.go
package service

import (
	"encoding/json"
	"strings"

	csstModel "madinahsalam_backend/internals/features/school/classes/class_section_subject_teachers/model"
	secModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RoomCache berisi data yang disimpan ke JSONB room_cache & bisa dipakai lintas fitur.
// Schema JSONB (top-level keys) sengaja konsisten dengan kolom generated di SQL:
//   - name       : string
//   - slug       : string (opsional)
//   - location   : string (opsional)
//   - code       : string (opsional)
//   - capacity   : int    (opsional)
//   - is_virtual : bool
//   - platform   : string (opsional, mis. "zoom")
//   - join_url   : string (opsional)
type RoomCache struct {
	Name     string
	Slug     *string
	Location *string

	Code      *string
	Capacity  *int
	IsVirtual bool
	Platform  *string
	JoinURL   *string
}

// helper kecil: trim string pointer, kalau kosong jadi nil
func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

// ValidateAndCacheRoom membaca room dari DB + validasi tenant.
// Catatan: class_room_school_id di-cast ke TEXT agar aman walau bukan UUID native.
func ValidateAndCacheRoom(
	tx *gorm.DB,
	expectSchoolID uuid.UUID,
	roomID uuid.UUID,
) (*RoomCache, error) {
	var row struct {
		SchoolID  string  `gorm:"column:school_id"`
		Name      string  `gorm:"column:name"`
		Slug      *string `gorm:"column:slug"`
		Location  *string `gorm:"column:location"`
		Code      *string `gorm:"column:code"`
		Capacity  *int    `gorm:"column:capacity"`
		IsVirtual bool    `gorm:"column:is_virtual"`
		Platform  *string `gorm:"column:platform"`
		JoinURL   *string `gorm:"column:join_url"`
	}

	if err := tx.Raw(`
		SELECT
			class_room_school_id::text AS school_id,
			class_room_name             AS name,
			class_room_slug             AS slug,
			class_room_location         AS location,
			class_room_code             AS code,
			class_room_capacity         AS capacity,
			class_room_is_virtual       AS is_virtual,
			class_room_platform         AS platform,
			class_room_join_url         AS join_url
		FROM class_rooms
		WHERE class_room_id = ? AND class_room_deleted_at IS NULL
	`, roomID).Scan(&row).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
	}

	if strings.TrimSpace(row.SchoolID) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
	}
	rmz, perr := uuid.Parse(strings.TrimSpace(row.SchoolID))
	if perr != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Format school_id ruang kelas tidak valid")
	}
	if rmz != expectSchoolID {
		return nil, fiber.NewError(fiber.StatusForbidden, "Ruang kelas bukan milik school Anda")
	}

	name := strings.TrimSpace(row.Name)
	if name == "" {
		name = "Ruang"
	}

	return &RoomCache{
		Name:      name,
		Slug:      trimPtr(row.Slug),
		Location:  trimPtr(row.Location),
		Code:      trimPtr(row.Code),
		Capacity:  row.Capacity,
		IsVirtual: row.IsVirtual,
		Platform:  trimPtr(row.Platform),
		JoinURL:   trimPtr(row.JoinURL),
	}, nil
}

/* =========================================================
   Generic: RoomCache → JSON
   ========================================================= */

// ToJSONMap mengubah RoomCache → datatypes.JSONMap
// Dipakai di model yang pakai JSONMap (misalnya: class_sections).
func ToJSONMap(rs *RoomCache) datatypes.JSONMap {
	if rs == nil {
		return nil
	}

	snap := datatypes.JSONMap{
		"name":       rs.Name,
		"is_virtual": rs.IsVirtual,
	}

	if rs.Slug != nil && strings.TrimSpace(*rs.Slug) != "" {
		snap["slug"] = strings.TrimSpace(*rs.Slug)
	}
	if rs.Location != nil && strings.TrimSpace(*rs.Location) != "" {
		snap["location"] = strings.TrimSpace(*rs.Location)
	}
	if rs.Code != nil && strings.TrimSpace(*rs.Code) != "" {
		snap["code"] = strings.TrimSpace(*rs.Code)
	}
	if rs.Capacity != nil {
		snap["capacity"] = *rs.Capacity
	}
	if rs.Platform != nil && strings.TrimSpace(*rs.Platform) != "" {
		snap["platform"] = strings.TrimSpace(*rs.Platform)
	}
	if rs.JoinURL != nil && strings.TrimSpace(*rs.JoinURL) != "" {
		snap["join_url"] = strings.TrimSpace(*rs.JoinURL)
	}

	return snap
}

// ToJSON mengubah RoomCache → datatypes.JSON (bentuk []byte).
// Dipakai di model yang pakai JSON (misalnya: CSST, class_schedule_rules).
func ToJSON(rs *RoomCache) datatypes.JSON {
	if rs == nil {
		// pakai literal null biar aman
		return datatypes.JSON([]byte("null"))
	}

	snap := ToJSONMap(rs)

	b, err := json.Marshal(snap)
	if err != nil {
		// fallback: null
		return datatypes.JSON([]byte("null"))
	}

	return datatypes.JSON(b)
}

// ToJSONPtr helper kalau butuh *datatypes.JSON (contoh: field pointer di CSST)
func ToJSONPtr(rs *RoomCache) *datatypes.JSON {
	if rs == nil {
		return nil
	}
	j := ToJSON(rs)
	return &j
}

/* =========================================================
   Spesifik: apply ke ClassSectionModel
   ========================================================= */

// ApplyRoomCacheToSection menulis cache ruang ke model section
// (JSONB + slug_cache).
// Kolom generated (name/location/slug_gen) akan otomatis diisi oleh DB.
func ApplyRoomCacheToSection(mcs *secModel.ClassSectionModel, rs *RoomCache) {
	if rs == nil {
		mcs.ClassSectionClassRoomCache = nil
		mcs.ClassSectionClassRoomSlugCache = nil
		return
	}

	mcs.ClassSectionClassRoomCache = ToJSONMap(rs)

	if rs.Slug != nil && strings.TrimSpace(*rs.Slug) != "" {
		slug := strings.TrimSpace(*rs.Slug)
		mcs.ClassSectionClassRoomSlugCache = &slug
	} else {
		mcs.ClassSectionClassRoomSlugCache = nil
	}
}

// ApplyRoomIDAndCacheToSection mengisi ID room + JSONB cache sekaligus.
func ApplyRoomIDAndCacheToSection(mcs *secModel.ClassSectionModel, roomID *uuid.UUID, rs *RoomCache) {
	mcs.ClassSectionClassRoomID = roomID
	ApplyRoomCacheToSection(mcs, rs)
}

/* =========================================================
   Spesifik: apply ke ClassSectionSubjectTeacherModel (CSST)
   ========================================================= */

// ApplyRoomCacheToCSST menulis cache ruang ke model CSST
// (JSONB + slug_cache).
// Kolom generated (name/location/slug_gen) akan otomatis diisi oleh DB.
func ApplyRoomCacheToCSST(mcsst *csstModel.ClassSectionSubjectTeacherModel, rs *RoomCache) {
	if rs == nil {
		mcsst.ClassSectionSubjectTeacherClassRoomCache = nil
		mcsst.ClassSectionSubjectTeacherClassRoomSlugCache = nil
		return
	}

	// kolom di model: *datatypes.JSON
	mcsst.ClassSectionSubjectTeacherClassRoomCache = ToJSONPtr(rs)

	if rs.Slug != nil && strings.TrimSpace(*rs.Slug) != "" {
		slug := strings.TrimSpace(*rs.Slug)
		mcsst.ClassSectionSubjectTeacherClassRoomSlugCache = &slug
	} else {
		mcsst.ClassSectionSubjectTeacherClassRoomSlugCache = nil
	}
}

// ApplyRoomIDAndCacheToCSST mengisi ID room + JSONB cache sekaligus.
func ApplyRoomIDAndCacheToCSST(mcsst *csstModel.ClassSectionSubjectTeacherModel, roomID *uuid.UUID, rs *RoomCache) {
	mcsst.ClassSectionSubjectTeacherClassRoomID = roomID
	ApplyRoomCacheToCSST(mcsst, rs)
}
