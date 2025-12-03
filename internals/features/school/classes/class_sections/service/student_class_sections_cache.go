// file: internals/features/school/classes/class_sections/service/caches.go
package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	scsModel "madinahsalam_backend/internals/features/school/classes/class_sections/model"
)

// Input minimal agar tidak import ke package profiles (hindari cycle).
type UserProfileCacheInput struct {
	UserID            uuid.UUID  // user_id (opsional jika ada UserProfileID)
	UserProfileID     *uuid.UUID // rekomendasi: kirim ini dari controller agar query hemat
	FullNameCache     *string
	AvatarURL         *string
	WhatsappURL       *string
	ParentName        *string
	ParentWhatsappURL *string
}

/* =========================================================
   Kolom optional (cache sekali)
========================================================= */

var (
	colsOnce sync.Once
	cols     struct {
		Name, Avatar, Wa, Parent, ParentWa, UpdatedAt bool
	}
)

func ensureColumns(db *gorm.DB) {
	colsOnce.Do(func() {
		// pakai model agar aman terhadap schema/alias
		cols.Name = db.Migrator().HasColumn(&scsModel.StudentClassSection{}, "student_class_section_user_profile_name_cache")
		cols.Avatar = db.Migrator().HasColumn(&scsModel.StudentClassSection{}, "student_class_section_user_profile_avatar_url_cache")
		cols.Wa = db.Migrator().HasColumn(&scsModel.StudentClassSection{}, "student_class_section_user_profile_whatsapp_url_cache")
		cols.Parent = db.Migrator().HasColumn(&scsModel.StudentClassSection{}, "student_class_section_user_profile_parent_name_cache")
		cols.ParentWa = db.Migrator().HasColumn(&scsModel.StudentClassSection{}, "student_class_section_user_profile_parent_whatsapp_url_cache")
		cols.UpdatedAt = db.Migrator().HasColumn(&scsModel.StudentClassSection{}, "student_class_section_updated_at")
	})
}

/* =========================================================
   Public API (dipanggil controller saat profil berubah)
========================================================= */

// Dipanggil dari controller manapun ketika profil user berubah.
// (Nama fungsi dipertahankan untuk backward compatibility)
func SyncUCSnapshotsFromUserProfile(ctx context.Context, db *gorm.DB, in UserProfileCacheInput, now time.Time) {
	ensureColumns(db)
	if !(cols.Name || cols.Avatar || cols.Wa || cols.Parent || cols.ParentWa) {
		log.Printf("[scs/sync] cache columns not found — skip")
		return
	}

	// 1) Ambil semua school_student_id milik user (aktif).
	msIDs := resolveSchoolStudentIDs(ctx, db, in)
	if len(msIDs) == 0 {
		return
	}

	// 2) Siapkan SET fields sesuai kolom yang tersedia.
	set := buildUpdateSet(in, now)
	if len(set) == 0 {
		return
	}

	// 3) Update dengan chunking untuk aman dari limit parameter.
	const chunk = 1000
	for i := 0; i < len(msIDs); i += chunk {
		j := i + chunk
		if j > len(msIDs) {
			j = len(msIDs)
		}
		part := msIDs[i:j]

		if err := db.WithContext(ctx).
			Model(&scsModel.StudentClassSection{}).
			Where("student_class_section_school_student_id IN ? AND student_class_section_deleted_at IS NULL", part).
			Updates(set).Error; err != nil {
			log.Printf("[scs/sync] update caches failed (batch %d-%d): %v", i, j, err)
			// lanjut batch berikutnya agar partial progress tetap jalan
		}
	}
}

/* =========================================================
   Helpers
========================================================= */

func resolveSchoolStudentIDs(ctx context.Context, db *gorm.DB, in UserProfileCacheInput) []uuid.UUID {
	var msIDs []uuid.UUID

	q := db.WithContext(ctx).
		Table("school_students AS ms").
		// index-friendly filter deleted_at
		Where("ms.school_student_deleted_at IS NULL")

	// Prefer pakai user_profile_id kalau ada (min query & paling cepat)
	if in.UserProfileID != nil && *in.UserProfileID != uuid.Nil {
		q = q.Where("ms.school_student_user_profile_id = ?", *in.UserProfileID)
	} else {
		// Fallback: join user_profiles untuk map user_id -> user_profile_id
		// note: filter deleted_at di user_profiles juga
		q = q.Joins("JOIN user_profiles up ON up.user_profile_id = ms.school_student_user_profile_id AND up.user_profile_deleted_at IS NULL").
			Where("up.user_profile_user_id = ?", in.UserID)
	}

	if err := q.Pluck("ms.school_student_id", &msIDs).Error; err != nil {
		log.Printf("[scs/sync] pluck school_student_id failed: %v", err)
		return nil
	}
	return msIDs
}

func buildUpdateSet(in UserProfileCacheInput, now time.Time) map[string]any {
	set := map[string]any{}

	// Nil di map → akan ditulis sebagai NULL (asalkan kolom NULLable)
	if cols.Name {
		set["student_class_section_user_profile_name_cache"] = in.FullNameCache
	}
	if cols.Avatar {
		set["student_class_section_user_profile_avatar_url_cache"] = in.AvatarURL
	}
	if cols.Wa {
		set["student_class_section_user_profile_whatsapp_url_cache"] = in.WhatsappURL
	}
	if cols.Parent {
		set["student_class_section_user_profile_parent_name_cache"] = in.ParentName
	}
	if cols.ParentWa {
		set["student_class_section_user_profile_parent_whatsapp_url_cache"] = in.ParentWhatsappURL
	}
	if cols.UpdatedAt {
		set["student_class_section_updated_at"] = now
	}

	return set
}
