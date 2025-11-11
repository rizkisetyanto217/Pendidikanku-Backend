// file: internals/features/school/classes/class_sections/snapshot/adapter_section_room.go
package snapshot

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	ars "schoolku_backend/internals/features/school/academics/rooms/snapshot"
	secModel "schoolku_backend/internals/features/school/classes/class_sections/model"
)

/*
   Adapter untuk kebutuhan Section:

   - Re-ekspos tipe dan fungsi validasi dari paket academics/rooms/snapshot
   - Sediakan ApplyRoomSnapshotToSection yang menulis JSONB + kolom turunan
*/

// Alias tipe supaya pengguna paket ini cukup pakai snapshot.RoomSnapshot
type RoomSnapshot = ars.RoomSnapshot

// Delegasikan validasi+snapshot ke paket akademik (hemat duplikasi logic).
func ValidateAndSnapshotRoom(tx *gorm.DB, expectSchoolID uuid.UUID, roomID uuid.UUID) (*RoomSnapshot, error) {
	return ars.ValidateAndSnapshotRoom(tx, expectSchoolID, roomID)
}

// Tulis snapshot ke model section (JSONB + turunan name/slug/location).
// CATATAN: tidak mengubah ClassSectionClassRoomIDSnapshot; kalau perlu, set ID terpisah di controller/DTO layer.
func ApplyRoomSnapshotToSection(mcs *secModel.ClassSectionModel, rs *RoomSnapshot) {
	if rs == nil {
		// clear snapshot
		mcs.ClassSectionClassRoomSnapshot = datatypes.JSON([]byte("null"))
		mcs.ClassSectionClassRoomNameSnapshot = nil
		mcs.ClassSectionClassRoomSlugSnapshot = nil
		mcs.ClassSectionClassRoomLocationSnapshot = nil
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
		mcs.ClassSectionClassRoomSnapshot = datatypes.JSON(b)
	} else {
		mcs.ClassSectionClassRoomSnapshot = datatypes.JSON([]byte("null"))
	}

	// kolom turunan untuk query cepat
	name := rs.Name
	mcs.ClassSectionClassRoomNameSnapshot = &name

	if rs.Slug != nil && strings.TrimSpace(*rs.Slug) != "" {
		mcs.ClassSectionClassRoomSlugSnapshot = rs.Slug
	} else {
		mcs.ClassSectionClassRoomSlugSnapshot = nil
	}

	if rs.Location != nil && strings.TrimSpace(*rs.Location) != "" {
		mcs.ClassSectionClassRoomLocationSnapshot = rs.Location
	} else {
		mcs.ClassSectionClassRoomLocationSnapshot = nil
	}
}
