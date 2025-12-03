// file: internals/features/school/classes/class_sections/cache/adapter_section_room.go
package service

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	ars "madinahsalam_backend/internals/features/school/academics/rooms/service"
	secModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
)

/*
   Adapter untuk kebutuhan Section:

   - Re-ekspos tipe dan fungsi validasi dari paket academics/rooms/cache
   - Sediakan ApplyRoomCacheToSection yang menulis JSONB + kolom turunan
*/

// Alias tipe supaya pengguna paket ini cukup pakai cache.RoomCache
type RoomCache = ars.RoomCache

// Delegasikan validasi+cache ke paket akademik (hemat duplikasi logic).
func ValidateAndCacheRoom(tx *gorm.DB, expectSchoolID uuid.UUID, roomID uuid.UUID) (*RoomCache, error) {
	return ars.ValidateAndCacheRoom(tx, expectSchoolID, roomID)
}

// Tulis cache ke model section (JSONB + turunan name/slug/location).
// CATATAN: tidak mengubah ClassSectionClassRoomIDCache; kalau perlu, set ID terpisah di controller/DTO layer.
func ApplyRoomCacheToSection(mcs *secModel.ClassSectionModel, rs *RoomCache) {
	if rs == nil {
		// clear cache
		mcs.ClassSectionClassRoomCache = datatypes.JSON([]byte("null"))
		mcs.ClassSectionClassRoomNameCache = nil
		mcs.ClassSectionClassRoomSlugCache = nil
		mcs.ClassSectionClassRoomLocationCache = nil
		return
	}

	snap := map[string]any{
		"name":       rs.Name,
		"is_virtual": rs.IsVirtual,
	}
	if rs.Slug != nil && strings.TrimSpace(*rs.Slug) != "" {
		snap["slug"] = *rs.Slug
	}
	if rs.Location != nil && strings.TrimSpace(*rs.Location) != "" {
		snap["location"] = *rs.Location
	}
	// metadata opsional — ikut disimpan supaya future-proof
	if rs.Code != nil && strings.TrimSpace(*rs.Code) != "" {
		snap["code"] = *rs.Code
	}
	if rs.Capacity != nil {
		snap["capacity"] = *rs.Capacity
	}
	if rs.Platform != nil && strings.TrimSpace(*rs.Platform) != "" {
		snap["platform"] = *rs.Platform
	}
	if rs.JoinURL != nil && strings.TrimSpace(*rs.JoinURL) != "" {
		snap["join_url"] = *rs.JoinURL
	}

	if b, err := json.Marshal(snap); err == nil {
		mcs.ClassSectionClassRoomCache = datatypes.JSON(b)
	} else {
		mcs.ClassSectionClassRoomCache = datatypes.JSON([]byte("null"))
	}

	// kolom turunan untuk query cepat
	name := rs.Name
	mcs.ClassSectionClassRoomNameCache = &name

	if rs.Slug != nil && strings.TrimSpace(*rs.Slug) != "" {
		mcs.ClassSectionClassRoomSlugCache = rs.Slug
	} else {
		mcs.ClassSectionClassRoomSlugCache = nil
	}

	if rs.Location != nil && strings.TrimSpace(*rs.Location) != "" {
		mcs.ClassSectionClassRoomLocationCache = rs.Location
	} else {
		mcs.ClassSectionClassRoomLocationCache = nil
	}
}
