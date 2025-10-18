// file: internals/features/school/masjid_teachers/snapshot/teacher_snapshot.go
package snapshot

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TeacherSnapshot: struktur JSON yang disimpan untuk guru.
type TeacherSnapshot struct {
	Name        *string `json:"name,omitempty"`
	WhatsappURL *string `json:"whatsapp_url,omitempty"`
	TitlePrefix *string `json:"title_prefix,omitempty"`
	TitleSuffix *string `json:"title_suffix,omitempty"`
}

// ValidateAndSnapshotTeacher baca data guru + validasi tenant (masjid).
func ValidateAndSnapshotTeacher(
	tx *gorm.DB,
	expectMasjidID uuid.UUID,
	teacherID uuid.UUID,
) (*TeacherSnapshot, error) {
	var row struct {
		MasjidID    string  `gorm:"column:masjid_id"`
		Name        *string `gorm:"column:teacher_name"`
		WhatsappURL *string `gorm:"column:whatsapp_url"`
		TitlePre    *string `gorm:"column:title_prefix"`
		TitleSuf    *string `gorm:"column:title_suffix"`
	}

	// Ambil dari masjid_teachers + fallback nama dari user_teachers
	const q = `
SELECT
  mt.masjid_teacher_masjid_id::text                                       AS masjid_id,
  COALESCE(mt.masjid_teacher_user_teacher_name_snapshot, ut.user_teacher_name) AS teacher_name,
  ut.user_teacher_whatsapp_url                                             AS whatsapp_url,
  ut.user_teacher_title_prefix                                             AS title_prefix,
  ut.user_teacher_title_suffix                                             AS title_suffix
FROM masjid_teachers mt
LEFT JOIN user_teachers ut
  ON ut.user_teacher_id = mt.masjid_teacher_user_teacher_id
 AND ut.user_teacher_deleted_at IS NULL
WHERE mt.masjid_teacher_id = ?
  AND mt.masjid_teacher_deleted_at IS NULL
LIMIT 1`

	if err := tx.Raw(q, teacherID).Scan(&row).Error; err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memuat data guru")
	}

	if strings.TrimSpace(row.MasjidID) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Guru tidak ditemukan")
	}
	rmz, perr := uuid.Parse(strings.TrimSpace(row.MasjidID))
	if perr != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Format masjid_id guru tidak valid")
	}
	if rmz != expectMasjidID {
		return nil, fiber.NewError(fiber.StatusForbidden, "Guru bukan milik masjid Anda")
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

	return &TeacherSnapshot{
		Name:        trimPtr(row.Name),
		WhatsappURL: trimPtr(row.WhatsappURL),
		TitlePrefix: trimPtr(row.TitlePre),
		TitleSuffix: trimPtr(row.TitleSuf),
	}, nil
}

// ToJSON ubah TeacherSnapshot → datatypes.JSON
func ToJSON(ts *TeacherSnapshot) datatypes.JSON {
	if ts == nil {
		return datatypes.JSON([]byte("null"))
	}
	m := map[string]any{}
	if ts.Name != nil && strings.TrimSpace(*ts.Name) != "" {
		m["name"] = *ts.Name
	}
	if ts.WhatsappURL != nil && strings.TrimSpace(*ts.WhatsappURL) != "" {
		m["whatsapp_url"] = *ts.WhatsappURL
	}
	if ts.TitlePrefix != nil && strings.TrimSpace(*ts.TitlePrefix) != "" {
		m["title_prefix"] = *ts.TitlePrefix
	}
	if ts.TitleSuffix != nil && strings.TrimSpace(*ts.TitleSuffix) != "" {
		m["title_suffix"] = *ts.TitleSuffix
	}
	if b, err := json.Marshal(m); err == nil {
		return datatypes.JSON(b)
	}
	return datatypes.JSON([]byte("null"))
}
