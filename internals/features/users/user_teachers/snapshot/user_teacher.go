// file: internals/services/snapsvc/snapsvc.go
package snapsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Dipakai di banyak tempat (Section, CSST, dsb)
type TeacherSnapshot struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	WhatsappURL *string   `json:"whatsapp_url,omitempty"`
	TitlePrefix *string   `json:"title_prefix,omitempty"`
	TitleSuffix *string   `json:"title_suffix,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
}

// Error spesifik untuk beda tenant
var ErrSchoolMismatch = errors.New("school mismatch")

// Helper umum untuk konversi ke JSONB (gorm datatypes.JSON)
func ToJSONB(v any) (datatypes.JSON, error) {
	if v == nil {
		return datatypes.JSON([]byte("null")), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

// Validasi school_teacher & buat snapshot user_teacher-nya
// - kembalikan gorm.ErrRecordNotFound jika tidak ditemukan
// - kembalikan ErrSchoolMismatch jika tenant beda
func BuildTeacherSnapshot(
	ctx context.Context,
	tx *gorm.DB,
	schoolID uuid.UUID,
	schoolTeacherID uuid.UUID,
) (*TeacherSnapshot, error) {
	var row struct {
		SchoolID      uuid.UUID
		UserTeacherID uuid.UUID
		FullName      string
		WhatsappURL   *string
		TitlePrefix   *string
		TitleSuffix   *string
		AvatarURL     *string
	}

	if err := tx.WithContext(ctx).Raw(`
		SELECT
			mt.school_teacher_school_id AS school_id,
			ut.user_teacher_id          AS user_teacher_id,
			ut.user_teacher_name        AS full_name,
			ut.user_teacher_whatsapp_url AS whatsapp_url,
			ut.user_teacher_title_prefix AS title_prefix,
			ut.user_teacher_title_suffix AS title_suffix,
			ut.user_teacher_avatar_url   AS avatar_url
		FROM school_teachers mt
		JOIN user_teachers ut
		  ON ut.user_teacher_id = mt.school_teacher_user_teacher_id
		WHERE mt.school_teacher_id = ?
		  AND mt.school_teacher_deleted_at IS NULL
	`, schoolTeacherID).Scan(&row).Error; err != nil {
		return nil, err
	}

	// not found (Raw+Scan tidak kasih ErrRecordNotFound, jadi cek sendiri)
	if row.SchoolID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	// tenant check
	if row.SchoolID != schoolID {
		return nil, ErrSchoolMismatch
	}

	nz := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			return nil
		}
		return &v
	}

	name := strings.TrimSpace(row.FullName)
	return &TeacherSnapshot{
		ID:          row.UserTeacherID, // simpan user_teacher_id sebagai id snapshot
		Name:        name,
		WhatsappURL: nz(row.WhatsappURL),
		TitlePrefix: nz(row.TitlePrefix),
		TitleSuffix: nz(row.TitleSuffix),
		AvatarURL:   nz(row.AvatarURL),
	}, nil
}
