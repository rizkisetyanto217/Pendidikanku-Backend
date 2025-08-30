package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Selaras dengan tabel:
// - user_quran_records_is_next: BOOLEAN (nullable)  -> *bool
// - user_quran_records_score:  NUMERIC(5,2)         -> *float64 (pakai type:numeric(5,2))
// - Hapus kolom: user_quran_records_status, user_quran_records_next
// - Kolom teks opsional dibuat pointer agar NULL bisa dibedakan dari "" (empty string)
// - Index dasar di-tag untuk kolom single; index gabungan ditandai via priority di MasjidID & CreatedAt, UserID & CreatedAt

type UserQuranRecordModel struct {
	UserQuranRecordID            uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:user_quran_records_id" json:"user_quran_records_id"`
	UserQuranRecordMasjidID      uuid.UUID       `gorm:"type:uuid;not null;column:user_quran_records_masjid_id;index:idx_uqr_masjid_created_at,priority:1" json:"user_quran_records_masjid_id"`
	UserQuranRecordUserID        uuid.UUID       `gorm:"type:uuid;not null;column:user_quran_records_user_id;index:idx_uqr_user_created_at,priority:1" json:"user_quran_records_user_id"`

	UserQuranRecordSessionID     *uuid.UUID      `gorm:"type:uuid;column:user_quran_records_session_id;index:idx_user_quran_records_session" json:"user_quran_records_session_id,omitempty"`
	UserQuranRecordTeacherUserID *uuid.UUID      `gorm:"type:uuid;column:user_quran_records_teacher_user_id;index:idx_uqr_teacher" json:"user_quran_records_teacher_user_id,omitempty"`

	UserQuranRecordSourceKind    *string         `gorm:"type:varchar(24);column:user_quran_records_source_kind;index:idx_uqr_source_kind" json:"user_quran_records_source_kind,omitempty"`
	UserQuranRecordScope         *string         `gorm:"type:text;column:user_quran_records_scope" json:"user_quran_records_scope,omitempty"`

	UserQuranRecordUserNote      *string         `gorm:"type:text;column:user_quran_records_user_note" json:"user_quran_records_user_note,omitempty"`
	UserQuranRecordTeacherNote   *string         `gorm:"type:text;column:user_quran_records_teacher_note" json:"user_quran_records_teacher_note,omitempty"`

	// ✅ score NUMERIC(5,2) nullable
	UserQuranRecordScore         *float64        `gorm:"type:numeric(5,2);column:user_quran_records_score" json:"user_quran_records_score,omitempty"`

	// ✅ is_next BOOLEAN nullable
	UserQuranRecordIsNext        *bool           `gorm:"column:user_quran_records_is_next;index:idx_uqr_is_next" json:"user_quran_records_is_next,omitempty"`

	UserQuranRecordCreatedAt     time.Time       `gorm:"column:user_quran_records_created_at;autoCreateTime;index:idx_uqr_masjid_created_at,priority:2,sort:desc;index:idx_uqr_user_created_at,priority:2,sort:desc" json:"user_quran_records_created_at"`
	UserQuranRecordUpdatedAt     time.Time       `gorm:"column:user_quran_records_updated_at;autoUpdateTime" json:"user_quran_records_updated_at"`
	UserQuranRecordDeletedAt     gorm.DeletedAt  `gorm:"column:user_quran_records_deleted_at;index" json:"user_quran_records_deleted_at,omitempty"`
}

func (UserQuranRecordModel) TableName() string {
	return "user_quran_records"
}
