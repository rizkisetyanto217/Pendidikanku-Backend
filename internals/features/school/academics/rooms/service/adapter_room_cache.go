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

/*
   RoomCache = snapshot ruangan yang disimpan ke JSONB.

   Penting: key JSON disamakan dengan json tag di ClassRoomModel:
   - class_room_id
   - class_room_school_id
   - class_room_name
   - class_room_code
   - class_room_slug
   - class_room_location
   - class_room_capacity
   - class_room_is_virtual
   - class_room_image_url
   - class_room_platform
   - class_room_join_url
   - class_room_meeting_id
   - class_room_passcode
*/

type RoomCache struct {
	ClassRoomID        uuid.UUID
	ClassRoomSchoolID  uuid.UUID
	ClassRoomName      string
	ClassRoomCode      *string
	ClassRoomSlug      *string
	ClassRoomLocation  *string
	ClassRoomCapacity  *int
	ClassRoomIsVirtual bool

	ClassRoomImageURL  *string
	ClassRoomPlatform  *string
	ClassRoomJoinURL   *string
	ClassRoomMeetingID *string
	ClassRoomPasscode  *string
}

// helper kecil: trim *string → *string (kosong jadi nil)
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

// helper string biasa
func trimStr(s string) string {
	return strings.TrimSpace(s)
}

// ValidateAndCacheRoom membaca room dari DB + validasi tenant.
// Hasil akhirnya RoomCache akan dimarshal ke JSON dengan key sama seperti model.
func ValidateAndCacheRoom(
	tx *gorm.DB,
	expectSchoolID uuid.UUID,
	roomID uuid.UUID,
) (*RoomCache, error) {
	var row struct {
		ClassRoomID        uuid.UUID `gorm:"column:class_room_id"`
		ClassRoomSchoolID  uuid.UUID `gorm:"column:class_room_school_id"`
		ClassRoomName      string    `gorm:"column:class_room_name"`
		ClassRoomCode      *string   `gorm:"column:class_room_code"`
		ClassRoomSlug      *string   `gorm:"column:class_room_slug"`
		ClassRoomLocation  *string   `gorm:"column:class_room_location"`
		ClassRoomCapacity  *int      `gorm:"column:class_room_capacity"`
		ClassRoomIsVirtual bool      `gorm:"column:class_room_is_virtual"`
		ClassRoomImageURL  *string   `gorm:"column:class_room_image_url"`
		ClassRoomPlatform  *string   `gorm:"column:class_room_platform"`
		ClassRoomJoinURL   *string   `gorm:"column:class_room_join_url"`
		ClassRoomMeetingID *string   `gorm:"column:class_room_meeting_id"`
		ClassRoomPasscode  *string   `gorm:"column:class_room_passcode"`
	}

	if err := tx.Raw(`
		SELECT
			class_room_id,
			class_room_school_id,
			class_room_name,
			class_room_code,
			class_room_slug,
			class_room_location,
			class_room_capacity,
			class_room_is_virtual,
			class_room_image_url,
			class_room_platform,
			class_room_join_url,
			class_room_meeting_id,
			class_room_passcode
		FROM class_rooms
		WHERE class_room_id = ? AND class_room_deleted_at IS NULL
	`, roomID).Scan(&row).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal validasi ruang kelas")
	}

	// not found / school mismatch
	if row.ClassRoomID == uuid.Nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Ruang kelas tidak ditemukan")
	}
	if row.ClassRoomSchoolID != expectSchoolID {
		return nil, fiber.NewError(fiber.StatusForbidden, "Ruang kelas bukan milik school Anda")
	}

	name := trimStr(row.ClassRoomName)
	if name == "" {
		name = "Ruang"
	}

	return &RoomCache{
		ClassRoomID:        row.ClassRoomID,
		ClassRoomSchoolID:  row.ClassRoomSchoolID,
		ClassRoomName:      name,
		ClassRoomCode:      trimPtr(row.ClassRoomCode),
		ClassRoomSlug:      trimPtr(row.ClassRoomSlug),
		ClassRoomLocation:  trimPtr(row.ClassRoomLocation),
		ClassRoomCapacity:  row.ClassRoomCapacity,
		ClassRoomIsVirtual: row.ClassRoomIsVirtual,
		ClassRoomImageURL:  trimPtr(row.ClassRoomImageURL),
		ClassRoomPlatform:  trimPtr(row.ClassRoomPlatform),
		ClassRoomJoinURL:   trimPtr(row.ClassRoomJoinURL),
		ClassRoomMeetingID: trimPtr(row.ClassRoomMeetingID),
		ClassRoomPasscode:  trimPtr(row.ClassRoomPasscode),
	}, nil
}

/* =========================================================
   Generic: RoomCache → JSON
   (key disamakan dengan json tag model)
   ========================================================= */

func ToJSONMap(rs *RoomCache) datatypes.JSONMap {
	if rs == nil {
		return nil
	}

	snap := datatypes.JSONMap{
		"class_room_id":         rs.ClassRoomID,
		"class_room_school_id":  rs.ClassRoomSchoolID,
		"class_room_name":       rs.ClassRoomName,
		"class_room_is_virtual": rs.ClassRoomIsVirtual,
	}

	if rs.ClassRoomCode != nil && trimStr(*rs.ClassRoomCode) != "" {
		snap["class_room_code"] = trimStr(*rs.ClassRoomCode)
	}
	if rs.ClassRoomSlug != nil && trimStr(*rs.ClassRoomSlug) != "" {
		snap["class_room_slug"] = trimStr(*rs.ClassRoomSlug)
	}
	if rs.ClassRoomLocation != nil && trimStr(*rs.ClassRoomLocation) != "" {
		snap["class_room_location"] = trimStr(*rs.ClassRoomLocation)
	}
	if rs.ClassRoomCapacity != nil {
		snap["class_room_capacity"] = *rs.ClassRoomCapacity
	}
	if rs.ClassRoomImageURL != nil && trimStr(*rs.ClassRoomImageURL) != "" {
		snap["class_room_image_url"] = trimStr(*rs.ClassRoomImageURL)
	}
	if rs.ClassRoomPlatform != nil && trimStr(*rs.ClassRoomPlatform) != "" {
		snap["class_room_platform"] = trimStr(*rs.ClassRoomPlatform)
	}
	if rs.ClassRoomJoinURL != nil && trimStr(*rs.ClassRoomJoinURL) != "" {
		snap["class_room_join_url"] = trimStr(*rs.ClassRoomJoinURL)
	}
	if rs.ClassRoomMeetingID != nil && trimStr(*rs.ClassRoomMeetingID) != "" {
		snap["class_room_meeting_id"] = trimStr(*rs.ClassRoomMeetingID)
	}
	if rs.ClassRoomPasscode != nil && trimStr(*rs.ClassRoomPasscode) != "" {
		snap["class_room_passcode"] = trimStr(*rs.ClassRoomPasscode)
	}

	return snap
}

func ToJSON(rs *RoomCache) datatypes.JSON {
	if rs == nil {
		return datatypes.JSON([]byte("null"))
	}
	snap := ToJSONMap(rs)

	b, err := json.Marshal(snap)
	if err != nil {
		return datatypes.JSON([]byte("null"))
	}

	return datatypes.JSON(b)
}

func ToJSONPtr(rs *RoomCache) *datatypes.JSON {
	if rs == nil {
		return nil
	}
	j := ToJSON(rs)
	return &j
}

/* =========================================================
   Apply ke ClassSectionModel
   ========================================================= */

func ApplyRoomCacheToSection(mcs *secModel.ClassSectionModel, rs *RoomCache) {
	if rs == nil {
		mcs.ClassSectionClassRoomCache = nil
		mcs.ClassSectionClassRoomSlugCache = nil
		return
	}

	mcs.ClassSectionClassRoomCache = ToJSONMap(rs)

	if rs.ClassRoomSlug != nil && trimStr(*rs.ClassRoomSlug) != "" {
		slug := trimStr(*rs.ClassRoomSlug)
		mcs.ClassSectionClassRoomSlugCache = &slug
	} else {
		mcs.ClassSectionClassRoomSlugCache = nil
	}
}

func ApplyRoomIDAndCacheToSection(mcs *secModel.ClassSectionModel, roomID *uuid.UUID, rs *RoomCache) {
	mcs.ClassSectionClassRoomID = roomID
	ApplyRoomCacheToSection(mcs, rs)
}

/* =========================================================
   Apply ke ClassSectionSubjectTeacherModel (CSST) — MODEL BARU (csst_*)
   ========================================================= */

func ApplyRoomCacheToCSST(mcsst *csstModel.ClassSectionSubjectTeacherModel, rs *RoomCache) {
	if rs == nil {
		mcsst.CSSTClassRoomCache = nil
		mcsst.CSSTClassRoomSlugCache = nil

		// generated fields (read-only di DB), tapi boleh diset biar enak buat response in-memory
		mcsst.CSSTClassRoomNameCache = nil
		mcsst.CSSTClassRoomSlugCacheGen = nil
		mcsst.CSSTClassRoomLocationCache = nil
		return
	}

	// JSON snapshot full (jsonb)
	mcsst.CSSTClassRoomCache = ToJSONPtr(rs)

	// slug cache + gen
	if rs.ClassRoomSlug != nil && trimStr(*rs.ClassRoomSlug) != "" {
		slug := trimStr(*rs.ClassRoomSlug)
		mcsst.CSSTClassRoomSlugCache = &slug

		// generated (read-only)
		mcsst.CSSTClassRoomSlugCacheGen = &slug
	} else {
		mcsst.CSSTClassRoomSlugCache = nil
		mcsst.CSSTClassRoomSlugCacheGen = nil
	}

	// name cache (generated read-only)
	name := trimStr(rs.ClassRoomName)
	if name != "" {
		mcsst.CSSTClassRoomNameCache = &name
	} else {
		mcsst.CSSTClassRoomNameCache = nil
	}

	// location cache (generated read-only)
	if rs.ClassRoomLocation != nil && trimStr(*rs.ClassRoomLocation) != "" {
		loc := trimStr(*rs.ClassRoomLocation)
		mcsst.CSSTClassRoomLocationCache = &loc
	} else {
		mcsst.CSSTClassRoomLocationCache = nil
	}
}

func ApplyRoomIDAndCacheToCSST(
	mcsst *csstModel.ClassSectionSubjectTeacherModel,
	roomID *uuid.UUID,
	rs *RoomCache,
) {
	mcsst.CSSTClassRoomID = roomID
	ApplyRoomCacheToCSST(mcsst, rs)
}
