// file: internals/features/attendance/user_attendance_urls/model/user_attendance_url.go
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

/*
  =========================================================
  GORM Model — user_attendance_urls
  - Mengikuti skema DDL yang kamu berikan
  - Soft delete pakai gorm.DeletedAt tapi kolom custom
  - Disertakan helper scopes untuk query umum
  - Disertakan fungsi EnsureUserAttendanceURLIndexes untuk bikin partial indexes
  =========================================================
*/

type UserAttendanceURL struct {
	// PK
	UserAttendanceURLID uuid.UUID `gorm:"column:user_attendance_url_id;type:uuid;primaryKey;default:gen_random_uuid()"`

	// Tenant & owner
	UserAttendanceURLMasjidID   uuid.UUID `gorm:"column:user_attendance_url_masjid_id;type:uuid;not null"`
	UserAttendanceURLAttendance uuid.UUID `gorm:"column:user_attendance_url_attendance_id;type:uuid;not null"`

	// (opsional) tipe media eksternal (lookup table)
	UserAttendanceTypeID *uuid.UUID `gorm:"column:user_attendance_type_id;type:uuid"`

	// Jenis/peran aset (e.g. image, video, attachment, link, audio)
	UserAttendanceURLKind string `gorm:"column:user_attendance_url_kind;type:varchar(24);not null"`

	// Lokasi file/link
	UserAttendanceURLHref         *string `gorm:"column:user_attendance_url_href;type:text"`
	UserAttendanceURLObjectKey    *string `gorm:"column:user_attendance_url_object_key;type:text"`
	UserAttendanceURLObjectKeyOld *string `gorm:"column:user_attendance_url_object_key_old;type:text"`

	// Metadata tampilan
	UserAttendanceURLLabel     *string `gorm:"column:user_attendance_url_label;type:varchar(160)"`
	UserAttendanceURLOrder     int32   `gorm:"column:user_attendance_url_order;type:int;not null;default:0"`
	UserAttendanceURLIsPrimary bool    `gorm:"column:user_attendance_url_is_primary;type:boolean;not null;default:false"`

	// Housekeeping (retensi/purge)
	UserAttendanceURLTrashURL           *string    `gorm:"column:user_attendance_url_trash_url;type:text"`
	UserAttendanceURLDeletePendingUntil *time.Time `gorm:"column:user_attendance_url_delete_pending_until;type:timestamptz"`

	// Uploader (opsional)
	UserAttendanceURLUploaderTeacherID *uuid.UUID `gorm:"column:user_attendance_url_uploader_teacher_id;type:uuid"`
	UserAttendanceURLUploaderStudentID *uuid.UUID `gorm:"column:user_attendance_url_uploader_student_id;type:uuid"`

	// Audit
	UserAttendanceURLCreatedAt time.Time      `gorm:"column:user_attendance_url_created_at;type:timestamptz;not null;default:now()"`
	UserAttendanceURLUpdatedAt time.Time      `gorm:"column:user_attendance_url_updated_at;type:timestamptz;not null;default:now()"`
	UserAttendanceURLDeletedAt gorm.DeletedAt `gorm:"column:user_attendance_url_deleted_at;type:timestamptz;index"`
}

// TableName mengikat struct ke tabel explicit
func (UserAttendanceURL) TableName() string {
	return "user_attendance_urls"
}

/* =========================================================
   Scopes umum (chainable)
========================================================= */

// Hanya baris hidup (belum soft-delete)
func ScopeUAULive(db *gorm.DB) *gorm.DB {
	return db.Where("user_attendance_url_deleted_at IS NULL")
}

// Filter per masjid
func ScopeUAUByMasjid(masjidID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_attendance_url_masjid_id = ?", masjidID)
	}
}

// Filter per attendance (owner)
func ScopeUAUByAttendance(attendanceID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_attendance_url_attendance_id = ?", attendanceID)
	}
}

// Filter per kind (image/video/attachment/link/audio/...)
func ScopeUAUByKind(kind string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_attendance_url_kind = ?", kind)
	}
}

// Hanya primary
func ScopeUAUPrimary(db *gorm.DB) *gorm.DB {
	return db.Where("user_attendance_url_is_primary = TRUE")
}

// Urutan tampilan default (primary dulu, lalu order, lalu created_at)
func ScopeUAUDefaultOrder(db *gorm.DB) *gorm.DB {
	return db.Order("user_attendance_url_is_primary DESC").
		Order("user_attendance_url_order ASC").
		Order("user_attendance_url_created_at ASC")
}

/* =========================================================
   Migrations: ensure partial indexes & unique indexes (WHERE ...)
   Catatan:
   - GORM belum support partial index fully lewat tag → pakai Raw SQL.
   - Idempotent: pakai IF NOT EXISTS.
========================================================= */

func EnsureUserAttendanceURLIndexes(db *gorm.DB) error {
	sqls := []string{
		// Lookup per attendance (live only) + urutan tampil
		`CREATE INDEX IF NOT EXISTS ix_uau_by_owner_live
		   ON user_attendance_urls (
		     user_attendance_url_attendance_id,
		     user_attendance_url_kind,
		     user_attendance_url_is_primary DESC,
		     user_attendance_url_order,
		     user_attendance_url_created_at
		   )
		   WHERE user_attendance_url_deleted_at IS NULL;`,

		// Filter per tenant (live only)
		`CREATE INDEX IF NOT EXISTS ix_uau_by_masjid_live
		   ON user_attendance_urls (user_attendance_url_masjid_id)
		   WHERE user_attendance_url_deleted_at IS NULL;`,

		// Satu primary per (attendance, kind) (live only)
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_uau_primary_per_kind_alive
		   ON user_attendance_urls (user_attendance_url_attendance_id, user_attendance_url_kind)
		   WHERE user_attendance_url_deleted_at IS NULL
		     AND user_attendance_url_is_primary = TRUE;`,

		// Anti-duplikat href per attendance (live only)
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_uau_attendance_href_alive
		   ON user_attendance_urls (user_attendance_url_attendance_id, LOWER(user_attendance_url_href))
		   WHERE user_attendance_url_deleted_at IS NULL
		     AND user_attendance_url_href IS NOT NULL;`,

		// Kandidat purge
		`CREATE INDEX IF NOT EXISTS ix_uau_purge_due
		   ON user_attendance_urls (user_attendance_url_delete_pending_until)
		   WHERE user_attendance_url_delete_pending_until IS NOT NULL
		     AND (
		       (user_attendance_url_deleted_at IS NULL  AND user_attendance_url_object_key_old IS NOT NULL) OR
		       (user_attendance_url_deleted_at IS NOT NULL AND user_attendance_url_object_key     IS NOT NULL)
		     );`,

		// Uploader lookups (live only)
		`CREATE INDEX IF NOT EXISTS ix_uau_uploader_teacher_live
		   ON user_attendance_urls (user_attendance_url_uploader_teacher_id)
		   WHERE user_attendance_url_deleted_at IS NULL;`,

		`CREATE INDEX IF NOT EXISTS ix_uau_uploader_student_live
		   ON user_attendance_urls (user_attendance_url_uploader_student_id)
		   WHERE user_attendance_url_deleted_at IS NULL;`,

		// BRIN untuk time-scan
		`CREATE INDEX IF NOT EXISTS brin_uau_created_at
		   ON user_attendance_urls USING BRIN (user_attendance_url_created_at);`,
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, q := range sqls {
			if err := tx.Exec(q).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

/* =========================================================
   Optional: constants untuk Kind agar konsisten
========================================================= */

type UAUKind string

const (
	UAUKindImage      UAUKind = "image"
	UAUKindVideo      UAUKind = "video"
	UAUKindAttachment UAUKind = "attachment"
	UAUKindLink       UAUKind = "link"
	UAUKindAudio      UAUKind = "audio"
)
