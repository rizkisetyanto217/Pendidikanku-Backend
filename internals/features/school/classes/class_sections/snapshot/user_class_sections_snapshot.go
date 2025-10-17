// file: internals/features/school/classes/class_sections/service/snapshots.go
package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	ucsecModel "masjidku_backend/internals/features/school/classes/class_sections/model"
)

// Input minimal agar tidak import ke package profiles (hindari cycle).
type UserProfileSnapshotInput struct {
	UserID            uuid.UUID  // user_id (opsional jika ada UserProfileID)
	UserProfileID     *uuid.UUID // rekomendasi: kirim ini dari controller agar query hemat
	FullNameSnapshot  *string
	AvatarURL         *string
	WhatsappURL       *string
	ParentName        *string
	ParentWhatsappURL *string
}

// Cache hasil cek kolom biar nggak panggil Migrator.HasColumn tiap request.
var (
	colsOnce sync.Once
	cols     struct {
		Name, Avatar, Wa, Parent, ParentWa, UpdatedAt bool
	}
)

func ensureColumns(db *gorm.DB) {
	colsOnce.Do(func() {
		cols.Name = db.Migrator().HasColumn(&ucsecModel.UserClassSection{}, "user_class_section_user_profile_name_snapshot")
		cols.Avatar = db.Migrator().HasColumn(&ucsecModel.UserClassSection{}, "user_class_section_user_profile_avatar_url_snapshot")
		cols.Wa = db.Migrator().HasColumn(&ucsecModel.UserClassSection{}, "user_class_section_user_profile_whatsapp_url_snapshot")
		cols.Parent = db.Migrator().HasColumn(&ucsecModel.UserClassSection{}, "user_class_section_user_profile_parent_name_snapshot")
		cols.ParentWa = db.Migrator().HasColumn(&ucsecModel.UserClassSection{}, "user_class_section_user_profile_parent_whatsapp_url_snapshot")
		cols.UpdatedAt = db.Migrator().HasColumn(&ucsecModel.UserClassSection{}, "user_class_section_updated_at")
	})
}

// Dipanggil dari controller manapun ketika profil user berubah.
func SyncUCSnapshotsFromUserProfile(ctx context.Context, db *gorm.DB, in UserProfileSnapshotInput, now time.Time) {
	ensureColumns(db)
	if !(cols.Name || cols.Avatar || cols.Wa || cols.Parent || cols.ParentWa) {
		log.Printf("[ucs/sync] snapshot columns not found — skip")
		return
	}

	// Kumpulkan masjid_student_id milik user (aktif)
	var msIDs []uuid.UUID
	q := db.WithContext(ctx).Table("masjid_students AS ms")

	// Prefer: filter langsung dengan user_profile_id (paling efisien)
	if in.UserProfileID != nil && *in.UserProfileID != uuid.Nil {
		q = q.Where("ms.masjid_student_user_profile_id = ? AND ms.masjid_student_deleted_at IS NULL", *in.UserProfileID)
	} else {
		// Fallback: JOIN ke users_profile untuk map dari user_id -> user_profile_id
		// Catatan: pastikan nama tabel benar ("users_profile" atau "users_profiles") sesuai model kamu.
		q = q.
			Joins("JOIN users_profile up ON up.user_profile_id = ms.masjid_student_user_profile_id").
			Where("up.user_profile_user_id = ? AND ms.masjid_student_deleted_at IS NULL", in.UserID)
	}

	if err := q.Pluck("ms.masjid_student_id", &msIDs).Error; err != nil {
		log.Printf("[ucs/sync] pluck masjid_student_id failed: %v", err)
		return
	}
	if len(msIDs) == 0 {
		return
	}

	set := map[string]any{}
	if cols.Name {
		set["user_class_section_user_profile_name_snapshot"] = in.FullNameSnapshot
	}
	if cols.Avatar {
		set["user_class_section_user_profile_avatar_url_snapshot"] = in.AvatarURL
	}
	if cols.Wa {
		set["user_class_section_user_profile_whatsapp_url_snapshot"] = in.WhatsappURL
	}
	if cols.Parent {
		set["user_class_section_user_profile_parent_name_snapshot"] = in.ParentName
	}
	if cols.ParentWa {
		set["user_class_section_user_profile_parent_whatsapp_url_snapshot"] = in.ParentWhatsappURL
	}
	if cols.UpdatedAt {
		set["user_class_section_updated_at"] = now
	}
	if len(set) == 0 {
		return
	}

	if err := db.WithContext(ctx).
		Model(&ucsecModel.UserClassSection{}).
		Where("user_class_section_masjid_student_id IN ? AND user_class_section_deleted_at IS NULL", msIDs).
		Updates(set).Error; err != nil {
		log.Printf("[ucs/sync] update snapshots failed: %v", err)
	}
}
