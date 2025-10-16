// file: internals/features/school/classes/class_sections/snapshot/room_snapshot.go
package snapshot

import (
	"encoding/json"
	"strings"

	secModel "masjidku_backend/internals/features/school/classes/class_sections/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RoomSnapshot berisi data yang disimpan ke JSON snapshot & kolom turunan.
type RoomSnapshot struct {
	Name     string
	Slug     *string
	Location *string
	// Metadata opsional ikut disimpan ke JSON snapshot:
	Code      *string
	Capacity  *int
	IsVirtual bool
	Platform  *string
	JoinURL   *string
}

// ValidateAndSnapshotRoom mengambil data ruang + cek tenant (masjid) aman.
// Catatan: class_room_masjid_id di-cast ke TEXT supaya aman bila kolom belum bertipe UUID native.
func ValidateAndSnapshotRoom(
	tx *gorm.DB,
	expectMasjidID uuid.UUID,
	roomID uuid.UUID,
) (*RoomSnapshot, error) {
	var row struct {
		MasjidID  string  `gorm:"column:masjid_id"`
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
			class_room_masjid_id::text AS masjid_id,
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

	if strings.TrimSpace(row.MasjidID) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
	}
	rmz, perr := uuid.Parse(strings.TrimSpace(row.MasjidID))
	if perr != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Format masjid_id ruang kelas tidak valid")
	}
	if rmz != expectMasjidID {
		return nil, fiber.NewError(fiber.StatusForbidden, "Ruang kelas bukan milik masjid Anda")
	}

	// Normalisasi ringan
	trimPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}
	name := strings.TrimSpace(row.Name)
	if name == "" {
		name = "Ruang"
	}

	return &RoomSnapshot{
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

// ApplyRoomSnapshotToSection menulis snapshot ruang ke model section
// (JSON + kolom turunan: *_name_snap, *_slug_snap, *_location_snap).
func ApplyRoomSnapshotToSection(mcs *secModel.ClassSectionModel, rs *RoomSnapshot) {
	if rs == nil {
		// clear snapshot
		mcs.ClassSectionRoomSnapshot = datatypes.JSON([]byte("null"))
		mcs.ClassSectionRoomNameSnap = nil
		mcs.ClassSectionRoomSlugSnap = nil
		mcs.ClassSectionRoomLocationSnap = nil
		return
	}

	snap := map[string]any{
		"name": rs.Name,
	}
	if rs.Slug != nil {
		snap["slug"] = *rs.Slug
	}
	if rs.Location != nil {
		snap["location"] = *rs.Location
	}
	// metadata opsional
	if rs.Code != nil {
		snap["code"] = *rs.Code
	}
	if rs.Capacity != nil {
		snap["capacity"] = *rs.Capacity
	}
	snap["is_virtual"] = rs.IsVirtual
	if rs.Platform != nil {
		snap["platform"] = *rs.Platform
	}
	if rs.JoinURL != nil {
		snap["join_url"] = *rs.JoinURL
	}

	if b, err := json.Marshal(snap); err == nil {
		mcs.ClassSectionRoomSnapshot = datatypes.JSON(b)
	} else {
		// fallback defensif
		mcs.ClassSectionRoomSnapshot = datatypes.JSON([]byte("null"))
	}

	// kolom turunan (string) untuk filter/sort cepat
	name := rs.Name
	mcs.ClassSectionRoomNameSnap = &name

	if rs.Slug != nil && strings.TrimSpace(*rs.Slug) != "" {
		mcs.ClassSectionRoomSlugSnap = rs.Slug
	} else {
		mcs.ClassSectionRoomSlugSnap = nil
	}

	if rs.Location != nil && strings.TrimSpace(*rs.Location) != "" {
		mcs.ClassSectionRoomLocationSnap = rs.Location
	} else {
		mcs.ClassSectionRoomLocationSnap = nil
	}
}

// ToJSON mengubah RoomSnapshot → datatypes.JSON (schema sama dengan ApplyRoomSnapshotToSection)
func ToJSON(rs *RoomSnapshot) datatypes.JSON {
	if rs == nil {
		return datatypes.JSON([]byte("null"))
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
		return datatypes.JSON(b)
	}
	return datatypes.JSON([]byte("null"))
}
