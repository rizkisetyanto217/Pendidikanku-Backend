// file: internals/features/school/school_teachers/snapshot/teacher_snapshot.go
package cache

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TeacherCache: struktur JSON yang disimpan untuk guru.
// Struktur & nama field disesuaikan dengan kebutuhan FE (class_section, CSST, dll).
type TeacherCache struct {
	ID            string  `json:"id"`                       // school_teacher_id (UUID → string)
	Name          *string `json:"name,omitempty"`           // nama guru
	AvatarURL     *string `json:"avatar_url,omitempty"`     // URL avatar
	WhatsappURL   *string `json:"whatsapp_url,omitempty"`   // link whatsapp
	TitlePrefix   *string `json:"title_prefix,omitempty"`   // contoh: "Ustadz"
	TitleSuffix   *string `json:"title_suffix,omitempty"`   // contoh: "Lc"
	Gender        *string `json:"gender,omitempty"`         // jenis kelamin
	TeacherCode   *string `json:"teacher_code,omitempty"`   // kode guru (raw dari school_teacher_code)
}

// ValidateAndCacheTeacher baca data guru + validasi tenant (school).
// Return: struct snapshot yang siap di-serialize ke JSONB.
func ValidateAndCacheTeacher(
	tx *gorm.DB,
	expectSchoolID uuid.UUID,
	teacherID uuid.UUID,
) (*TeacherCache, error) {
	var row struct {
		SchoolID    string  `gorm:"column:school_id"`
		Name        *string `gorm:"column:teacher_name"`
		WhatsappURL *string `gorm:"column:whatsapp_url"`
		TitlePre    *string `gorm:"column:title_prefix"`
		TitleSuf    *string `gorm:"column:title_suffix"`
		AvatarURL   *string `gorm:"column:avatar_url"`
		Gender      *string `gorm:"column:gender"`
		TeacherCode *string `gorm:"column:teacher_code"`
	}

	// Ambil dari school_teachers + fallback dari user_teachers (schema terbaru)
	const q = `
SELECT
  mt.school_teacher_school_id::text                                           AS school_id,
  COALESCE(mt.school_teacher_user_teacher_full_name_cache,
           ut.user_teacher_user_full_name_cache)                                     AS teacher_name,
  COALESCE(mt.school_teacher_user_teacher_whatsapp_url_cache,
           ut.user_teacher_whatsapp_url)                                      AS whatsapp_url,
  COALESCE(mt.school_teacher_user_teacher_title_prefix_cache,
           ut.user_teacher_title_prefix)                                      AS title_prefix,
  COALESCE(mt.school_teacher_user_teacher_title_suffix_cache,
           ut.user_teacher_title_suffix)                                      AS title_suffix,
  COALESCE(mt.school_teacher_user_teacher_avatar_url_cache,
           ut.user_teacher_avatar_url)                                        AS avatar_url,
  COALESCE(mt.school_teacher_user_teacher_gender_cache,
           ut.user_teacher_gender)                                            AS gender,
  mt.school_teacher_code                                                      AS teacher_code
FROM school_teachers mt
LEFT JOIN user_teachers ut
  ON ut.user_teacher_id = mt.school_teacher_user_teacher_id
 AND ut.user_teacher_deleted_at IS NULL
WHERE mt.school_teacher_id = ?
  AND mt.school_teacher_deleted_at IS NULL
LIMIT 1`

	if err := tx.Raw(q, teacherID).Scan(&row).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memuat data guru")
	}

	if strings.TrimSpace(row.SchoolID) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Guru tidak ditemukan")
	}

	rmz, perr := uuid.Parse(strings.TrimSpace(row.SchoolID))
	if perr != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Format school_id guru tidak valid")
	}
	if rmz != expectSchoolID {
		return nil, fiber.NewError(fiber.StatusForbidden, "Guru bukan milik school Anda")
	}

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

	// Sementara:  teacher_code sama-sama ambil dari school_teacher_code
	code := trimPtr(row.TeacherCode)

	snap := &TeacherCache{
		ID:            teacherID.String(),
		Name:          trimPtr(row.Name),
		AvatarURL:     trimPtr(row.AvatarURL),
		WhatsappURL:   trimPtr(row.WhatsappURL),
		TitlePrefix:   trimPtr(row.TitlePre),
		TitleSuffix:   trimPtr(row.TitleSuf),
		Gender:        trimPtr(row.Gender),
		TeacherCode:   code,
	}

	return snap, nil
}

// ToJSON ubah TeacherCache → datatypes.JSON (untuk disimpan di kolom JSONB).
func ToJSON(ts *TeacherCache) datatypes.JSON {
	if ts == nil {
		return datatypes.JSON([]byte("null"))
	}

	m := map[string]any{}

	// ID selalu diset (school_teacher_id)
	if strings.TrimSpace(ts.ID) != "" {
		m["id"] = strings.TrimSpace(ts.ID)
	}

	trim := func(p *string) (string, bool) {
		if p == nil {
			return "", false
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return "", false
		}
		return v, true
	}

	if v, ok := trim(ts.Name); ok {
		m["name"] = v
	}
	if v, ok := trim(ts.AvatarURL); ok {
		m["avatar_url"] = v
	}
	if v, ok := trim(ts.WhatsappURL); ok {
		m["whatsapp_url"] = v
	}
	if v, ok := trim(ts.TitlePrefix); ok {
		m["title_prefix"] = v
	}
	if v, ok := trim(ts.TitleSuffix); ok {
		m["title_suffix"] = v
	}
	if v, ok := trim(ts.Gender); ok {
		m["gender"] = v
	}

	if v, ok := trim(ts.TeacherCode); ok {
		m["teacher_code"] = v
	}

	if len(m) == 0 {
		return datatypes.JSON([]byte("null"))
	}

	if b, err := json.Marshal(m); err == nil {
		return datatypes.JSON(b)
	}
	return datatypes.JSON([]byte("null"))
}
